package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestService(t *testing.T, handle http.HandlerFunc) *Service {
	t.Helper()
	server := httptest.NewServer(handle)
	t.Cleanup(server.Close)
	return NewService(&AIClient{baseURL: server.URL, model: "test-model", client: &http.Client{Timeout: 10 * time.Second}})
}

func upstream(w http.ResponseWriter, content string) {
	_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}})
}

func TestRules(t *testing.T) {
	for input, expected := range map[string]string{"WELCOME": "shadow", "Hi": "shadow", "abcdef": "shadow", "HelloHi": "thinkertoy", "Hello world!": "thinkertoy", "Hello, everyone": "standard", "LONG UPPERCASE TEXT": "shadow", "1234567": "thinkertoy", "  WELCOME  ": "shadow"} {
		got := Recommend(input)
		if got.Recommended != expected || got.Reasoning == "" || len(got.Alternatives) != 2 {
			t.Errorf("%q: %+v", input, got)
		}
		if !reflect.DeepEqual(got, Recommend(input)) {
			t.Fatal("недетерминированное правило")
		}
		for _, alt := range got.Alternatives {
			if alt.Banner == expected || alt.Score < 0 || alt.Score > 1 || alt.Reason == "" {
				t.Fatal(alt)
			}
		}
	}
}

func TestMockAndEnvironment(t *testing.T) {
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_MODEL", "")
	service := NewService(NewAIClient())
	if !service.Mock() || service.client.client.Timeout != 10*time.Second {
		t.Fatal("неверная конфигурация")
	}
	for _, text := range []string{"Hello", "Happy Birth", strings.Repeat("A", 1000), "A\nB"} {
		got, err := service.Suggest(context.Background(), text)
		if err != nil || len(got) != 3 {
			t.Fatal(got, err)
		}
		again, _ := service.Suggest(context.Background(), text)
		if !reflect.DeepEqual(got, again) {
			t.Fatal("mock должен быть детерминированным")
		}
		v, err := service.Variations(context.Background(), text)
		if err != nil || len(v) != 4 {
			t.Fatal(v, err)
		}
	}
	t.Setenv("LLM_BASE_URL", "http://configured.example/v1")
	t.Setenv("LLM_MODEL", "configured-model")
	client := NewAIClient()
	if client.baseURL != "http://configured.example/v1" || client.model != "configured-model" {
		t.Fatal("переменные окружения не прочитаны")
	}
}

func TestLiveProtocolAndCache(t *testing.T) {
	t.Setenv("LLM_API_KEY", "local-test-token")
	var calls atomic.Int32
	service := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer local-test-token" {
			t.Error("не передан настроенный ключ")
		}
		if r.Method != "POST" || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("неверный протокол")
		}
		var body struct {
			Model    string
			Stream   bool
			Messages []struct{ Role, Content string }
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || body.Model != "test-model" || body.Stream || len(body.Messages) != 1 || !strings.Contains(body.Messages[0].Content, "Hello") {
			t.Error("неверный запрос")
		}
		upstream(w, `["Hello!","Hello World","Hello Team"]`)
	})
	first, err := service.Suggest(context.Background(), "Hello")
	if err != nil {
		t.Fatal(err)
	}
	first[0] = "changed"
	second, _ := service.Suggest(context.Background(), "Hello")
	if calls.Load() != 1 || second[0] != "Hello!" {
		t.Fatal("кэш повреждён или не использован")
	}
	service.cache["Hello"] = cacheItem{values: second, expires: time.Now().Add(-time.Second)}
	if _, err := service.Suggest(context.Background(), "Hello"); err != nil || calls.Load() != 2 {
		t.Fatal("кэш не истёк", err)
	}
	service.client.baseURL += "/v1"
	if _, err := service.Suggest(context.Background(), "Hello again"); err != nil {
		t.Fatal(err)
	}
}

func TestLiveVariations(t *testing.T) {
	service := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		upstream(w, `[{"text":"Hello.","description":"Formal","suggested_banner":"standard"},{"text":"HELLO!","description":"Bold","suggested_banner":"shadow"},{"text":"~ Hello ~","description":"Playful","suggested_banner":"thinkertoy"}]`)
	})
	values, err := service.Variations(context.Background(), "Hello")
	if err != nil || len(values) != 3 || values[1].SuggestedBanner != "shadow" {
		t.Fatal(values, err)
	}
}

func TestInvalidUpstream(t *testing.T) {
	for _, content := range []string{"not json", `[]`, `["one","two"]`, `["same","same","same"]`, `["Привет","Hello","World"]`, `["a\nb","Hello","World"]`, `["one","two","three","four","five","six"]`, `[null,"two","three"]`} {
		t.Run(content, func(t *testing.T) {
			service := newTestService(t, func(w http.ResponseWriter, r *http.Request) { upstream(w, content) })
			if _, err := service.Suggest(context.Background(), "Hello"); !errors.Is(err, ErrResponse) {
				t.Fatal(err)
			}
			if len(service.cache) != 0 {
				t.Fatal("ошибка не должна попадать в кэш")
			}
		})
	}
	for _, body := range []string{`{}`, `{"choices":[]}`, `{"choices":[{"message":{"content":""}}]}`, strings.Repeat("x", 65537)} {
		service := newTestService(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) })
		if _, err := service.Suggest(context.Background(), "Hello"); !errors.Is(err, ErrResponse) {
			t.Fatal(err)
		}
	}
	service := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		upstream(w, `[{"text":"Hello","description":"x","suggested_banner":"../secret"},{"text":"Hi","description":"x","suggested_banner":"standard"},{"text":"Hey","description":"x","suggested_banner":"shadow"}]`)
	})
	if _, err := service.Variations(context.Background(), "Hello"); !errors.Is(err, ErrResponse) {
		t.Fatal(err)
	}
}

func TestUpstreamStatusAndNoRedirect(t *testing.T) {
	for _, status := range []int{301, 400, 401, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			service := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "http://invalid.example/")
				w.WriteHeader(status)
			})
			_, err := service.Suggest(context.Background(), "Hello")
			want := ErrResponse
			if status == 429 || status >= 500 {
				want = ErrUnavailable
			}
			if !errors.Is(err, want) {
				t.Fatal(err)
			}
		})
	}
}

func TestTimeoutAndCancellation(t *testing.T) {
	release := make(chan struct{})
	service := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	t.Cleanup(func() { close(release) })
	start := time.Now()
	if _, err := service.Suggest(context.Background(), "Timeout"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 9*time.Second || elapsed > 12*time.Second {
		t.Fatalf("таймаут %s", elapsed)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start = time.Now()
	if _, err := service.Suggest(ctx, "Cancel"); !errors.Is(err, ErrUnavailable) || time.Since(start) > time.Second {
		t.Fatal("отмена не обработана", err)
	}
}

func TestConfigurationAndBusy(t *testing.T) {
	for _, base := range []string{"ftp://example.com", "http://", ":bad", "https://u:p@example.com", "https://example.com?q=1", "https://example.com#frag"} {
		service := NewService(&AIClient{baseURL: base, model: "model", client: http.DefaultClient})
		if _, err := service.Suggest(context.Background(), "Hello"); !errors.Is(err, ErrConfiguration) {
			t.Fatal(base, err)
		}
	}
	service := NewService(&AIClient{baseURL: "http://localhost:1", model: "model", client: http.DefaultClient})
	for i := 0; i < cap(service.slots); i++ {
		service.slots <- struct{}{}
	}
	if _, err := service.Suggest(context.Background(), "Hello"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	service.client.model = ""
	if _, err := service.Suggest(context.Background(), "Hello"); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
}

func TestConcurrentBoundedCache(t *testing.T) {
	t.Setenv("LLM_BASE_URL", "")
	service := NewService(NewAIClient())
	var wg sync.WaitGroup
	for i := 0; i < 600; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			values, err := service.Suggest(context.Background(), fmt.Sprintf("Hello %d", i))
			if err != nil || len(values) != 3 {
				t.Errorf("%v", err)
			}
		}(i)
	}
	wg.Wait()
	if len(service.cache) > 256 {
		t.Fatal("неограниченный кэш")
	}
}

func TestConnectionFailureAndTruncatedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()
	service := NewService(&AIClient{baseURL: server.URL, model: "model", client: &http.Client{Timeout: 10 * time.Second}})
	if _, err := service.Suggest(context.Background(), "Hello"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	service = newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte("short"))
	})
	if _, err := service.Suggest(context.Background(), "Hello"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}
