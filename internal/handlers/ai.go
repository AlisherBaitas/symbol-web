package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"symbol-web/internal/ai"
)

// [Suggestions] HTTP-обработчик подсказок: POST /api/suggest (маршрут в cmd/server/main.go).
// Цепочка: suggestions.js -> HandleSuggestion -> ai.GetSuggestions (internal/ai/suggestions.go).
// HandleSuggestion принимает JSON с текстом и возвращает 3-5 вариантов продолжения.
// Ответ: {"suggestions": ["...", ...]}; ошибки: {"error": "..."} со статусом 400/503/500.
func HandleSuggestion(w http.ResponseWriter, r *http.Request) {
	text, ok := readTextRequest(w, r)
	if !ok {
		return
	}
	// [Suggestions] GetSuggestions не проверяет максимальную длину, поэтому проверяем здесь.
	// Тот же лимит, что и у формы: дальше 1000 символов генератор текст не примет.
	if len(text) > maxTextBytes {
		writeVariationError(w, http.StatusBadRequest, "Enter at most 1000 characters.")
		return
	}
	// [Validation] Те же символы, что умеет рисовать генератор: иначе подсказку
	// с кириллицей или переносом строки потом нельзя нарисовать.
	if _, message := validateArtText(text); message != "" && text != "" {
		writeVariationError(w, http.StatusBadRequest, message)
		return
	}

	// [Suggestions] Без LLM_BASE_URL клиент работает в демо-режиме (mockSuggestions).
	suggestions, err := ai.NewAIClient().GetSuggestions(r.Context(), text)
	if err != nil {
		// [Suggestions] Перевод ошибок слоя ai в HTTP-статусы для suggestions.js.
		switch {
		case errors.Is(err, ai.ErrTextTooShort):
			writeVariationError(w, http.StatusBadRequest, "Enter at least 3 characters to get suggestions.")
		case errors.Is(err, ai.ErrLLMUnavailable):
			log.Printf("suggestions: %v", err)
			writeVariationError(w, http.StatusServiceUnavailable, "The suggestions service is unavailable. Please try again later.")
		default:
			log.Printf("suggestions: %v", err)
			writeVariationError(w, http.StatusInternalServerError, "Could not create suggestions. Please try again.")
		}
		return
	}

	writeVariationJSON(w, http.StatusOK, struct {
		Suggestions []string `json:"suggestions"`
	}{Suggestions: suggestions})
}

// HandleBannerRec возвращает рекомендацию по локальным Go-правилам без вызова LLM.
func HandleBannerRec(w http.ResponseWriter, r *http.Request) {}

// HandleVariation принимает JSON с текстом и возвращает проверенные творческие варианты.
func HandleVariation(w http.ResponseWriter, r *http.Request) {
	// [Suggestions] При внедрении подсказок разбор запроса вынесен в readTextRequest,
	// чтобы HandleVariation и HandleSuggestion не дублировали один и тот же код.
	text, ok := readTextRequest(w, r)
	if !ok {
		return
	}

	// Контекст запроса отменяет вызов LLM при отключении браузера.
	// Клиент сам выбирает демонстрационный режим по переменным окружения.
	variations, err := ai.NewAIClient().GetVariations(r.Context(), text)
	if err != nil {
		switch {
		case errors.Is(err, ai.ErrInvalidVariationInput):
			writeVariationError(w, http.StatusBadRequest, "Enter 1-1000 ASCII characters; line breaks are allowed.")
		case errors.Is(err, ai.ErrLLMUnavailable):
			log.Printf("variations: %v", err)
			writeVariationError(w, http.StatusServiceUnavailable, "The ideas service is unavailable. Please try again later.")
		default:
			// Подробности нужны в журнале сервера; пользователю возвращаем понятное сообщение.
			log.Printf("variations: %v", err)
			writeVariationError(w, http.StatusInternalServerError, "Could not create ideas. Please try again.")
		}
		return
	}

	writeVariationJSON(w, http.StatusOK, struct {
		Variations []ai.Variation `json:"variations"`
	}{Variations: variations})
}

// [Suggestions] Общие части, добавленные вместе с подсказками.
// maxTextBytes совпадает с лимитом ввода формы (1000 ASCII-символов).
const maxTextBytes = 1000

// [Suggestions] readTextRequest вынесен из HandleVariation, используется обоими обработчиками.
// readTextRequest проверяет метод, Content-Type и тело {"text": "..."}.
// При ошибке сама отправляет JSON-ответ и возвращает ok == false.
func readTextRequest(w http.ResponseWriter, r *http.Request) (text string, ok bool) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeVariationError(w, http.StatusMethodNotAllowed, "Use POST for this endpoint.")
		return "", false
	}

	// Разрешаем параметры заголовка, например application/json; charset=utf-8.
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeVariationError(w, http.StatusUnsupportedMediaType, "Send JSON with Content-Type: application/json.")
		return "", false
	}

	// 8 КиБ достаточно для 1000 ASCII-символов, даже записанных как \\uXXXX.
	// Ограничиваем всё тело, включая поля и пробелы вокруг JSON.
	const maxRequestBytes = 8 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	defer r.Body.Close()

	var input struct {
		Text string `json:"text"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&input)
	if err == nil {
		// После объекта допускаются только пробелы и конец тела запроса.
		var extra any
		if trailingErr := decoder.Decode(&extra); trailingErr != io.EOF {
			err = trailingErr
			if err == nil {
				err = errors.New("multiple JSON values")
			}
		}
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeVariationError(w, http.StatusRequestEntityTooLarge, "The request is too large. Shorten the text and try again.")
		} else {
			writeVariationError(w, http.StatusBadRequest, "Send one JSON object containing only a string field named text.")
		}
		return "", false
	}

	return input.Text, true
}

// writeVariationError сохраняет единый формат ошибок для JavaScript и других клиентов API.
func writeVariationError(w http.ResponseWriter, status int, message string) {
	writeVariationJSON(w, status, map[string]string{"error": message})
}

// writeVariationJSON кодирует ответ до отправки заголовков, чтобы ошибка кодирования
// позволяла вернуть статус 500. Готовые варианты и ошибки всегда передаются как JSON.
func writeVariationJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("variations: encode response: %v", err)
		status = http.StatusInternalServerError
		body = []byte(`{"error":"Could not create ideas. Please try again."}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Printf("variations: write response: %v", err)
	}
}

func validateArtText(input string) (bool, string) {
    if strings.TrimSpace(input) == "" {
        return false, "Text is empty"
    }

    if utf8.RuneCountInString(input) > 1000 {
        return false, "Text is too long (max 1000 characters)"
    }

    for _, r := range input {
        if r == '\n' {
            continue
        }

        if r < ' ' || r > '~' {
            return false, "Invalid character"
        }
    }

    return true, ""
}