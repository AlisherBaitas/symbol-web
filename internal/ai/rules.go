package ai

import "strings"

type Alternative struct {
	Banner string  `json:"banner"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

type Recommendation struct {
	Recommended  string        `json:"recommended"`
	Reasoning    string        `json:"reasoning"`
	Alternatives []Alternative `json:"alternatives"`
}

// Recommend применяет правила по порядку и никогда не обращается к сети.
func Recommend(text string) Recommendation {
	text = strings.TrimSpace(text)
	hasLetter, uppercase := false, true
	for _, ch := range text {
		if ch >= 'a' && ch <= 'z' {
			hasLetter = true
			uppercase = false
		}
		if ch >= 'A' && ch <= 'Z' {
			hasLetter = true
		}
	}
	result := Recommendation{Recommended: "thinkertoy", Reasoning: "Короткий текст со смешанным регистром хорошо выглядит в игровом стиле."}
	if (hasLetter && uppercase) || len(text) <= 6 {
		result.Recommended = "shadow"
		result.Reasoning = "Короткий текст или заглавные буквы выразительнее с объёмной тенью."
	} else if len(text) > 12 {
		result.Recommended = "standard"
		result.Reasoning = "Классический шрифт помогает читать длинный текст."
	}
	for _, name := range []string{"standard", "shadow", "thinkertoy"} {
		if name != result.Recommended {
			reasons := map[string]string{"standard": "Чёткий классический рисунок.", "shadow": "Объёмный акцент.", "thinkertoy": "Лёгкий декоративный рисунок."}
			score := []float64{0.7, 0.5}[len(result.Alternatives)]
			result.Alternatives = append(result.Alternatives, Alternative{name, score, reasons[name]})
		}
	}
	return result
}
