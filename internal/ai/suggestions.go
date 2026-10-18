package ai

import (
	"regexp"
	"strings"
	"unicode/utf8"
	"errors"
)

const (
	minSuggestInput    = 3
	minSuggestions     = 3
	maxSuggestions     = 5
	maxSuggestionChars = 50
	// [Validation] Ограничение размера ответа LLM, как у вариаций.
	maxSuggestReplyBytes = 64 * 1024
)

var (
	// ErrTextTooShort is returned when the input has fewer than 3 characters
	ErrTextTooShort = errors.New("text must be at least 3 characters")
	// ErrLLMUnavailable is returned when the LLM backend can't be reached or times out
	ErrLLMUnavailable = errors.New("LLM backend is unavailable")
	// ErrInvalidLLMResponse is returned when the LLM answer can't be used
	ErrInvalidLLMResponse = errors.New("invalid response from LLM backend")
)

// listPrefix matches list markers like "1.", "2)", "-", "*" at the start of a line
var listPrefix = regexp.MustCompile(`^(?:[-*•]|\d+[.)])\s*`)

// Request/response shapes of an OpenAI-compatible chat completions API (Ollama supports it too)
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}


// mockSuggestions builds canned suggestions that depend only on the input
func mockSuggestions(text string) []string {
	// [Validation] Переносы и повторные пробелы -> один пробел; длинный текст
	// сокращаем, чтобы каждая подсказка была короче 50 символов (самый длинный
	// вариант добавляет " World", 6 символов).
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > maxSuggestionChars-7 {
		text = text[:maxSuggestionChars-10] + "..."
	}
	return []string{
		text + "!",
		text + " World",
		"~ " + text + " ~",
		strings.ToUpper(text),
	}
}


// chatURL turns LLM_BASE_URL into the chat completions URL
func chatURL(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}


// parseSuggestions cleans the LLM answer: one suggestion per line,
// no numbering/bullets/quotes, no duplicates, under 50 chars, at most 5
func parseSuggestions(answer string) []string {
	var result []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(answer, "\n") {
		line = strings.TrimSpace(line)
		line = listPrefix.ReplaceAllString(line, "")
		line = strings.Trim(line, "\"'`")
		line = strings.TrimSpace(line)

		if line == "" || utf8.RuneCountInString(line) >= maxSuggestionChars {
			continue
		}
		// [Validation] Подсказку должно быть можно нарисовать: только печатный ASCII.
		if !printableASCII(line) {
			continue
		}
		key := strings.ToLower(line)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, line)

		if len(result) == maxSuggestions {
			break
		}
	}
	return result
}


// [Validation] printableASCII сообщает, состоит ли строка только из символов 32-126,
// которые есть в файлах баннеров.
func printableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > '~' {
			return false
		}
	}
	return true
}
