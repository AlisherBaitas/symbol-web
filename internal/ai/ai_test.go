package ai

// [Fix] Файл ai_test.go требуется структурой проекта, но содержал только
// "package ai". Здесь проверяется поведение пакета целиком: mock-режим через
// NewAIClient (как его запускает сервер) и то, что правила
// выбора баннера работают без сети.

import (
	"context"
	"reflect"
	"testing"
)

func TestNewAIClient_MockModeWithoutBackend(t *testing.T) {
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_MODEL", "")

	c := NewAIClient()
	suggestions, err := c.GetSuggestions(context.Background(), "Happy Birth")
	if err != nil {
		t.Fatalf("GetSuggestions() in mock mode error = %v", err)
	}
	if len(suggestions) < minSuggestions || len(suggestions) > maxSuggestions {
		t.Errorf("GetSuggestions() returned %d suggestions, want 3-5", len(suggestions))
	}
	for _, s := range suggestions {
		if len(s) >= maxSuggestionChars || !printableASCII(s) {
			t.Errorf("suggestion %q is too long or not printable ASCII", s)
		}
	}

	// Детерминированность: повторный вызов даёт тот же ответ.
	again, _ := c.GetSuggestions(context.Background(), "Happy Birth")
	if !reflect.DeepEqual(suggestions, again) {
		t.Errorf("mock suggestions are not deterministic: %q vs %q", suggestions, again)
	}

	variations, err := c.GetVariations(context.Background(), "hello")
	if err != nil {
		t.Fatalf("GetVariations() in mock mode error = %v", err)
	}
	if len(variations) < minVariations || len(variations) > maxVariations {
		t.Errorf("GetVariations() returned %d variations, want 3-5", len(variations))
	}
	for _, v := range variations {
		if !validVariationBanner(v.SuggestedBanner) {
			t.Errorf("variation %+v has an unknown banner", v)
		}
	}
}

func TestNewAIClient_MockModeIgnoresCanceledContext(t *testing.T) {
	// В mock-режиме сеть не используется, поэтому таймауты на него не влияют.
	t.Setenv("LLM_BASE_URL", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewAIClient().GetSuggestions(ctx, "Hello"); err != nil {
		t.Errorf("GetSuggestions() in mock mode error = %v, want nil", err)
	}
}

func TestNewAIClient_MockModeShortText(t *testing.T) {
	t.Setenv("LLM_BASE_URL", "")
	for _, text := range []string{"", "Hi", "  a  "} {
		if _, err := NewAIClient().GetSuggestions(context.Background(), text); err != ErrTextTooShort {
			t.Errorf("GetSuggestions(%q) error = %v, want ErrTextTooShort", text, err)
		}
	}
}

func TestGetRecommend_WorksWithUnreachableBackend(t *testing.T) {
	// Правила не используют LLM: даже с недоступным адресом ответ есть.
	t.Setenv("LLM_BASE_URL", "http://127.0.0.1:9")
	got, err := GetRecommend("WELCOME")
	if err != nil || got.Recommended != "shadow" {
		t.Errorf("GetRecommend() = %+v, %v; want shadow without error", got, err)
	}
}
