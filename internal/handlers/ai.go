package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"

	"symbol-web/internal/ai"
)

func HandleSuggestion(w http.ResponseWriter, r *http.Request) {}

func HandleBannerRec(w http.ResponseWriter, r *http.Request) {}

// HandleVariation принимает JSON с текстом и возвращает проверенные творческие варианты.
func HandleVariation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeVariationError(w, http.StatusMethodNotAllowed, "Use POST to request ideas.")
		return
	}

	// Разрешаем параметры заголовка, например application/json; charset=utf-8.
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeVariationError(w, http.StatusUnsupportedMediaType, "Send JSON with Content-Type: application/json.")
		return
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
		return
	}

	// Контекст запроса отменяет вызов LLM при отключении браузера.
	// Клиент сам выбирает демонстрационный режим по переменным окружения.
	variations, err := ai.NewAIClient().GetVariations(r.Context(), input.Text)
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
