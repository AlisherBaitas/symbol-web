package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// chatBody builds an OpenAI-compatible chat completion response with the given content
func chatBody(content string) string {
	body, _ := json.Marshal(map[string]any{
		"choices": []any{
			map[string]any{"message": map[string]string{"role": "assistant", "content": content}},
		},
	})
	return string(body)
}

// fakeLLM answers on /v1/chat/completions with the given status and body after an optional delay
func fakeLLM(status int, body string, delay time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return
			}
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}
}

// newTestClient points the client at the fake server; nil server means mock mode
func newTestClient(t *testing.T, handler http.HandlerFunc, timeout time.Duration) *AIClient {
	t.Helper()
	c := &AIClient{model: "test-model", client: &http.Client{Timeout: timeout}}
	if handler != nil {
		srv := httptest.NewServer(handler)
		t.Cleanup(srv.Close)
		c.baseURL = srv.URL
	}
	return c
}

func TestNewAIClient(t *testing.T) {
	tests := []struct {
		name        string
		envBaseURL  string
		envModel    string
		wantBaseURL string
		wantModel   string
	}{
		{name: "mock mode when nothing is set", envBaseURL: "", envModel: "", wantBaseURL: "", wantModel: ""},
		{
			name:        "values come from environment",
			envBaseURL:  "http://127.0.0.1:11434",
			envModel:    "llama3:latest",
			wantBaseURL: "http://127.0.0.1:11434",
			wantModel:   "llama3:latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LLM_BASE_URL", tt.envBaseURL)
			t.Setenv("LLM_MODEL", tt.envModel)
			got := NewAIClient()
			if got.baseURL != tt.wantBaseURL || got.model != tt.wantModel {
				t.Errorf("NewAIClient() = {%q, %q}, want {%q, %q}", got.baseURL, got.model, tt.wantBaseURL, tt.wantModel)
			}
			if got.client == nil || got.client.Timeout != 10*time.Second {
				t.Errorf("NewAIClient() HTTP client timeout is not 10s")
			}
		})
	}
}

func TestAIClient_GetSuggestions(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name    string
		server  http.HandlerFunc // nil -> mock mode
		args    args
		want    []string
		wantErr error
	}{
		{
			name: "mock mode",
			args: args{text: "Happy Birth"},
			want: []string{"Happy Birth!", "Happy Birth World", "~ Happy Birth ~", "HAPPY BIRTH"},
		},
		{
			name: "mock mode trims outer spaces",
			args: args{text: "  Hey  "},
			want: []string{"Hey!", "Hey World", "~ Hey ~", "HEY"},
		},
		{name: "too short", args: args{text: "ab"}, wantErr: ErrTextTooShort},
		{name: "too short after trim", args: args{text: "  ab  "}, wantErr: ErrTextTooShort},
		{name: "empty", args: args{text: ""}, wantErr: ErrTextTooShort},
		{
			name:   "live mode",
			server: fakeLLM(http.StatusOK, chatBody("1. Happy Birthday!\n2. Happy Birthday Team!\n3. Happy Birthday [Name]"), 0),
			args:   args{text: "Happy Birth"},
			want:   []string{"Happy Birthday!", "Happy Birthday Team!", "Happy Birthday [Name]"},
		},
		{
			name:    "live mode, too few usable lines",
			server:  fakeLLM(http.StatusOK, chatBody("only one"), 0),
			args:    args{text: "hello"},
			wantErr: ErrInvalidLLMResponse,
		},
		{
			name:    "live mode, backend error 503",
			server:  fakeLLM(http.StatusServiceUnavailable, "", 0),
			args:    args{text: "hello"},
			wantErr: ErrLLMUnavailable,
		},
		{
			name:    "live mode, timeout",
			server:  fakeLLM(http.StatusOK, chatBody("a1\nb2\nc3"), 500*time.Millisecond),
			args:    args{text: "hello"},
			wantErr: ErrLLMUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, tt.server, 100*time.Millisecond)
			got, err := c.GetSuggestions(context.Background(), tt.args.text)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("AIClient.GetSuggestions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AIClient.GetSuggestions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAIClient_chatSuggest(t *testing.T) {
	// Сервер проверяет, что клиент отправил модель и промпт в правильном формате
	checkRequest := func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil ||
			req.Model != "test-model" || req.Stream || len(req.Messages) != 1 ||
			req.Messages[0].Role != "user" || req.Messages[0].Content != "the prompt" ||
			r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Write([]byte(chatBody("line one\nline two")))
	}

	type args struct {
		prompt string
	}
	tests := []struct {
		name    string
		server  http.HandlerFunc
		baseURL string // used when server is nil
		args    args
		want    string
		wantErr error
	}{
		{name: "request format and answer", server: checkRequest, args: args{prompt: "the prompt"}, want: "line one\nline two"},
		{name: "status 500", server: fakeLLM(http.StatusInternalServerError, "boom", 0), args: args{prompt: "p"}, wantErr: ErrLLMUnavailable},
		{name: "status 429", server: fakeLLM(http.StatusTooManyRequests, "", 0), args: args{prompt: "p"}, wantErr: ErrLLMUnavailable},
		{name: "status 400", server: fakeLLM(http.StatusBadRequest, "bad", 0), args: args{prompt: "p"}, wantErr: ErrInvalidLLMResponse},
		{name: "not JSON", server: fakeLLM(http.StatusOK, "not json", 0), args: args{prompt: "p"}, wantErr: ErrInvalidLLMResponse},
		{name: "no choices", server: fakeLLM(http.StatusOK, `{"choices":[]}`, 0), args: args{prompt: "p"}, wantErr: ErrInvalidLLMResponse},
		{name: "backend is down", baseURL: "http://127.0.0.1:1", args: args{prompt: "p"}, wantErr: ErrLLMUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, tt.server, time.Second)
			if tt.server == nil {
				c.baseURL = tt.baseURL
			}
			got, err := c.chatSuggest(context.Background(), tt.args.prompt)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("AIClient.chatSuggest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AIClient.chatSuggest() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAIClient_GetVariations(t *testing.T) {
	llmArray := `[{"text":"Hello!","description":"Bold","suggested_banner":"shadow"},` +
		`{"text":"hello","description":"Calm","suggested_banner":"standard"},` +
		`{"text":"~ hi ~","description":"Fancy","suggested_banner":"thinkertoy"}]`

	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	type args struct {
		ctx  context.Context
		text string
	}
	tests := []struct {
		name      string
		server    http.HandlerFunc
		nilClient bool
		args      args
		want      []Variation
		wantErr   error
	}{
		{name: "mock mode", args: args{ctx: context.Background(), text: "hello"}, want: mockVariations("hello")},
		{name: "empty text", args: args{ctx: context.Background(), text: ""}, wantErr: ErrInvalidVariationInput},
		{name: "cyrillic text", args: args{ctx: context.Background(), text: "Привет"}, wantErr: ErrInvalidVariationInput},
		{name: "canceled request", args: args{ctx: canceled, text: "hello"}, wantErr: ErrLLMUnavailable},
		{name: "client is nil", nilClient: true, args: args{ctx: context.Background(), text: "hello"}, wantErr: ErrVariationsConfiguration},
		{
			name:   "live mode",
			server: fakeLLM(http.StatusOK, chatBody(llmArray), 0),
			args:   args{ctx: context.Background(), text: "hello"},
			want: []Variation{
				{Text: "Hello!", Description: "Bold", SuggestedBanner: "shadow"},
				{Text: "hello", Description: "Calm", SuggestedBanner: "standard"},
				{Text: "~ hi ~", Description: "Fancy", SuggestedBanner: "thinkertoy"},
			},
		},
		{
			name:    "live mode, bad answer",
			server:  fakeLLM(http.StatusOK, chatBody("I cannot do that"), 0),
			args:    args{ctx: context.Background(), text: "hello"},
			wantErr: ErrInvalidLLMResponse,
		},
		{
			name:    "live mode, backend error 503",
			server:  fakeLLM(http.StatusServiceUnavailable, "", 0),
			args:    args{ctx: context.Background(), text: "hello"},
			wantErr: ErrLLMUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, tt.server, time.Second)
			if tt.nilClient {
				c = nil
			}
			got, err := c.GetVariations(tt.args.ctx, tt.args.text)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("AIClient.GetVariations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AIClient.GetVariations() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAIClient_requestVariations(t *testing.T) {
	redirect := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com/steal", http.StatusFound)
	}

	type args struct {
		prompt string
	}
	tests := []struct {
		name     string
		server   http.HandlerFunc
		baseURL  string // used when server is nil
		model    string
		noClient bool
		args     args
		want     string
		wantErr  error
	}{
		{name: "answer", server: fakeLLM(http.StatusOK, chatBody("[]"), 0), model: "m", args: args{prompt: "p"}, want: "[]"},
		{name: "not http scheme", baseURL: "ftp://example.com", model: "m", args: args{prompt: "p"}, wantErr: ErrVariationsConfiguration},
		{name: "no host", baseURL: "not a url", model: "m", args: args{prompt: "p"}, wantErr: ErrVariationsConfiguration},
		{name: "password in URL", baseURL: "http://user:pass@example.com", model: "m", args: args{prompt: "p"}, wantErr: ErrVariationsConfiguration},
		{name: "query in URL", baseURL: "http://example.com?x=1", model: "m", args: args{prompt: "p"}, wantErr: ErrVariationsConfiguration},
		{name: "no model", server: fakeLLM(http.StatusOK, chatBody("[]"), 0), model: "  ", args: args{prompt: "p"}, wantErr: ErrVariationsConfiguration},
		{name: "no HTTP client", server: fakeLLM(http.StatusOK, chatBody("[]"), 0), model: "m", noClient: true, args: args{prompt: "p"}, wantErr: ErrVariationsConfiguration},
		{name: "status 408", server: fakeLLM(http.StatusRequestTimeout, "", 0), model: "m", args: args{prompt: "p"}, wantErr: ErrLLMUnavailable},
		{name: "status 500", server: fakeLLM(http.StatusInternalServerError, "", 0), model: "m", args: args{prompt: "p"}, wantErr: ErrLLMUnavailable},
		{name: "status 404", server: fakeLLM(http.StatusNotFound, "", 0), model: "m", args: args{prompt: "p"}, wantErr: ErrInvalidLLMResponse},
		{name: "redirect is not followed", server: redirect, model: "m", args: args{prompt: "p"}, wantErr: ErrInvalidLLMResponse},
		{name: "answer is too large", server: fakeLLM(http.StatusOK, chatBody(strings.Repeat("a", 70*1024)), 0), model: "m", args: args{prompt: "p"}, wantErr: ErrInvalidLLMResponse},
		{name: "invalid UTF-8", server: fakeLLM(http.StatusOK, "\xff\xfe", 0), model: "m", args: args{prompt: "p"}, wantErr: ErrInvalidLLMResponse},
		{name: "no choices", server: fakeLLM(http.StatusOK, `{"choices":[]}`, 0), model: "m", args: args{prompt: "p"}, wantErr: ErrInvalidLLMResponse},
		{name: "backend is down", baseURL: "http://127.0.0.1:1", model: "m", args: args{prompt: "p"}, wantErr: ErrLLMUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, tt.server, time.Second)
			if tt.server == nil {
				c.baseURL = tt.baseURL
			}
			c.model = tt.model
			if tt.noClient {
				c.client = nil
			}
			got, err := c.requestVariations(context.Background(), tt.args.prompt)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("AIClient.requestVariations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AIClient.requestVariations() = %q, want %q", got, tt.want)
			}
		})
	}
}
