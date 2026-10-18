package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxVariationInputChars  = 1000
	minVariations           = 3
	maxVariations           = 5
	maxVariationTextChars   = 49
	maxVariationDescChars   = 29
	maxVariationReplyBytes  = 64 * 1024
	variationRequestTimeout = 10 * time.Second
)

var (
	// ErrInvalidVariationInput позволяет HTTP-обработчику вернуть ответ со статусом 400.
	ErrInvalidVariationInput = errors.New("text must contain 1-1000 ASCII characters; line breaks are allowed")
	// ErrVariationsConfiguration указывает на ошибку настройки клиента.
	ErrVariationsConfiguration = errors.New("invalid variations client configuration")
)

// Variation описывает одну идею для выбора. Имена баннеров указаны без суффикса .txt.
type Variation struct {
	Text            string `json:"text"`
	Description     string `json:"description"`
	SuggestedBanner string `json:"suggested_banner"`
}

func validateVariationInput(text string) error {
	if len(text) == 0 || len(text) > maxVariationInputChars {
		return ErrInvalidVariationInput
	}
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if ch == '\n' || (ch >= ' ' && ch <= '~') {
			continue
		}
		if ch == '\r' && i+1 < len(text) && text[i+1] == '\n' {
			continue
		}
		return ErrInvalidVariationInput
	}
	if strings.TrimSpace(text) == "" {
		return ErrInvalidVariationInput
	}
	return nil
}

func mockVariations(text string) []Variation {
	// Заменяем переносы строк и последовательности пробелов одним пробелом.
	// Сокращаем текст заранее, оставляя четыре символа для "Hi, " или "~ " и " ~",
	// чтобы готовый вариант укладывался в ограничение длины: 49 символов.
	base := strings.Join(strings.Fields(text), " ")
	if len(base) > maxVariationTextChars-4 {
		base = base[:maxVariationTextChars-7] + "..."
	}
	formal := strings.ToUpper(base[:1]) + base[1:]
	return []Variation{
		{Text: formal, Description: "Professional style", SuggestedBanner: "standard"},
		{Text: strings.ToUpper(base) + "!", Description: "Bold and emphatic", SuggestedBanner: "shadow"},
		{Text: "Hi, " + strings.ToLower(base), Description: "Friendly and casual", SuggestedBanner: "standard"},
		{Text: "~ " + base + " ~", Description: "Decorative style", SuggestedBanner: "thinkertoy"},
	}
}


func parseVariations(answer string) ([]Variation, error) {
	answer = strings.TrimSpace(answer)
	// Некоторые модели оборачивают JSON в блок кода Markdown. Принимаем только
	// один полностью закрытый блок с пометкой json. Пояснения, неполные массивы
	// и дополнительные значения JSON отклоняются при проверке.
	if strings.HasPrefix(answer, "```json\n") && strings.HasSuffix(answer, "\n```") {
		answer = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(answer, "```json\n"), "\n```"))
	}
	var variations []Variation
	if err := json.Unmarshal([]byte(answer), &variations); err != nil {
		return nil, fmt.Errorf("%w: expected a JSON array of variations", ErrInvalidLLMResponse)
	}
	if len(variations) < minVariations || len(variations) > maxVariations {
		return nil, fmt.Errorf("%w: expected 3-5 variations", ErrInvalidLLMResponse)
	}
	seen := make(map[string]bool, len(variations))
	for i := range variations {
		variation := &variations[i]
		// Убираем обычные пробелы по краям; управляющие символы отклонит проверка ниже.
		variation.Text = strings.Trim(variation.Text, " ")
		variation.Description = strings.Trim(variation.Description, " ")
		if !validVariationText(variation.Text) || !validVariationDescription(variation.Description) ||
			!validVariationBanner(variation.SuggestedBanner) || seen[variation.Text] {
			return nil, fmt.Errorf("%w: variation %d has invalid or duplicate fields", ErrInvalidLLMResponse, i+1)
		}
		// Разный регистр допустим: "Hello" и "HELLO" считаются отдельными вариантами.
		seen[variation.Text] = true
	}
	return variations, nil
}


func validVariationText(text string) bool {
	if len(text) == 0 || len(text) > maxVariationTextChars {
		return false
	}
	for i := 0; i < len(text); i++ {
		if text[i] < ' ' || text[i] > '~' {
			return false
		}
	}
	return true
}


func validVariationDescription(description string) bool {
	length := utf8.RuneCountInString(description)
	if !utf8.ValidString(description) || strings.TrimSpace(description) == "" || length > maxVariationDescChars {
		return false
	}
	for _, ch := range description {
		if unicode.IsControl(ch) {
			return false
		}
	}
	return true
}


func validVariationBanner(banner string) bool {
	return banner == "standard" || banner == "shadow" || banner == "thinkertoy"
}
