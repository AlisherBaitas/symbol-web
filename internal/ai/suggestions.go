package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"
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

// GetSuggestions returns 3-5 text completions for the given input
func (c *AIClient) GetSuggestions(ctx context.Context, text string) ([]string, error) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) < minSuggestInput {
		return nil, ErrTextTooShort
	}

	// Mock mode: no backend configured, return deterministic canned suggestions
	if c.baseURL == "" {
		return mockSuggestions(text), nil
	}

	prompt := fmt.Sprintf(
		"Complete this text creatively for ASCII art display:\n"+
			"Input: %q\n"+
			"Provide 3-5 short, creative completions suitable for ASCII art.\n"+
			"Each completion should be under 50 characters.\n"+
			"Return only the completions, one per line.",
		text,
	)

	answer, err := c.chatSuggest(ctx, prompt)
	if err != nil {
		return nil, err
	}

	suggestions := parseSuggestions(answer)
	if len(suggestions) < minSuggestions {
		return nil, fmt.Errorf("%w: got %d usable suggestions", ErrInvalidLLMResponse, len(suggestions))
	}
	return suggestions, nil
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

// chatSuggest sends the prompt to the LLM and returns the raw answer text
func (c *AIClient) chatSuggest(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: []chatMessage{{Role: "user", Content: prompt}},
		Stream:   false,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL(c.baseURL), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// The client has a 10 second timeout, network errors and timeouts mean "unavailable"
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("%w: status %d", ErrLLMUnavailable, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", ErrInvalidLLMResponse, resp.StatusCode)
	}

	// [Validation] Не читаем ответ LLM больше 64 КиБ.
	var parsed chatResponse
	// Было json.NewDecoder(resp.Body) без ограничения.
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxSuggestReplyBytes)).Decode(&parsed); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidLLMResponse, err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("%w: no choices", ErrInvalidLLMResponse)
	}
	return parsed.Choices[0].Message.Content, nil
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
