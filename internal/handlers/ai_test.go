package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// jsonRequest builds a request with a JSON body and Content-Type
func jsonRequest(method, target, body string) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// textBody builds {"text": "..."} with proper escaping
func textBody(text string) string {
	b, _ := json.Marshal(map[string]string{"text": text})
	return string(b)
}

// checkJSONError checks that the answer is JSON with a non-empty "error" field
func checkJSONError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] == "" {
		t.Errorf("body = %q, want JSON with an error message", rec.Body.String())
	}
}

func TestHandleSuggestion(t *testing.T) {
	tests := []struct {
		name       string
		llmURL     string // "" -> mock mode
		request    *http.Request
		wantStatus int
		want       []string
	}{
		{
			name:       "mock suggestions",
			request:    jsonRequest(http.MethodPost, "/api/suggest", textBody("Hello")),
			wantStatus: http.StatusOK,
			want:       []string{"Hello!", "Hello World", "~ Hello ~", "HELLO"},
		},
		{name: "too short", request: jsonRequest(http.MethodPost, "/api/suggest", textBody("ab")), wantStatus: http.StatusBadRequest},
		{name: "empty text", request: jsonRequest(http.MethodPost, "/api/suggest", textBody("")), wantStatus: http.StatusBadRequest},
		{name: "cyrillic", request: jsonRequest(http.MethodPost, "/api/suggest", textBody("Привет")), wantStatus: http.StatusBadRequest},
		{name: "1001 characters", request: jsonRequest(http.MethodPost, "/api/suggest", textBody(strings.Repeat("a", 1001))), wantStatus: http.StatusBadRequest},
		{name: "broken JSON", request: jsonRequest(http.MethodPost, "/api/suggest", "{"), wantStatus: http.StatusBadRequest},
		{name: "GET is not allowed", request: jsonRequest(http.MethodGet, "/api/suggest", ""), wantStatus: http.StatusMethodNotAllowed},
		{
			name:       "LLM is down",
			llmURL:     "http://127.0.0.1:1",
			request:    jsonRequest(http.MethodPost, "/api/suggest", textBody("Hello")),
			wantStatus: http.StatusServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LLM_BASE_URL", tt.llmURL)
			t.Setenv("LLM_MODEL", "test-model")
			rec := httptest.NewRecorder()
			HandleSuggestion(rec, tt.request)

			if rec.Code != tt.wantStatus {
				t.Fatalf("HandleSuggestion() status = %d, want %d, body %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
				t.Errorf("HandleSuggestion() Content-Type = %q, want JSON", rec.Header().Get("Content-Type"))
			}
			if tt.wantStatus != http.StatusOK {
				checkJSONError(t, rec)
				return
			}
			var got struct {
				Suggestions []string `json:"suggestions"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("HandleSuggestion() body is not JSON: %v", err)
			}
			if !reflect.DeepEqual(got.Suggestions, tt.want) {
				t.Errorf("HandleSuggestion() = %v, want %v", got.Suggestions, tt.want)
			}
		})
	}
}

func TestHandleBannerRec(t *testing.T) {
	// [Fix] Убран t.Skip: обработчик теперь принимает JSON из ТЗ, тест выполняется.
	// Кейсы по заданию: запрос {"text": "..."}, ответ JSON
	// {"recommended": "...", "reasoning": "...", "alternatives": [...]}.

	tests := []struct {
		name            string
		request         *http.Request
		wantStatus      int
		wantRecommended string
	}{
		{name: "uppercase word", request: jsonRequest(http.MethodPost, "/api/recommend-banner", textBody("WELCOME")), wantStatus: http.StatusOK, wantRecommended: "shadow"},
		{name: "long text", request: jsonRequest(http.MethodPost, "/api/recommend-banner", textBody("Hello there friend")), wantStatus: http.StatusOK, wantRecommended: "standard"},
		{name: "medium mixed case", request: jsonRequest(http.MethodPost, "/api/recommend-banner", textBody("Hello World")), wantStatus: http.StatusOK, wantRecommended: "thinkertoy"},
		{name: "empty text", request: jsonRequest(http.MethodPost, "/api/recommend-banner", textBody("")), wantStatus: http.StatusBadRequest},
		// [Fix] Дополнительные позитивные и негативные кейсы.
		{name: "symbol heavy", request: jsonRequest(http.MethodPost, "/api/recommend-banner", textBody("@#$%!abc")), wantStatus: http.StatusOK, wantRecommended: "thinkertoy"},
		{name: "browser CRLF", request: jsonRequest(http.MethodPost, "/api/recommend-banner", textBody("Hello\r\nWorld")), wantStatus: http.StatusOK, wantRecommended: "standard"},
		{name: "cyrillic text", request: jsonRequest(http.MethodPost, "/api/recommend-banner", textBody("Привет")), wantStatus: http.StatusBadRequest},
		{name: "invalid JSON", request: jsonRequest(http.MethodPost, "/api/recommend-banner", "{bad"), wantStatus: http.StatusBadRequest},
		{name: "form instead of JSON", request: formRequest("WELCOME", ""), wantStatus: http.StatusUnsupportedMediaType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			HandleBannerRec(rec, tt.request)

			if rec.Code != tt.wantStatus {
				t.Fatalf("HandleBannerRec() status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusOK {
				// [Fix] Ошибки тоже в JSON (раньше — простой текст).
				checkJSONError(t, rec)
				return
			}
			var got struct {
				Recommended  string            `json:"recommended"`
				Reasoning    string            `json:"reasoning"`
				Alternatives []json.RawMessage `json:"alternatives"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("HandleBannerRec() body is not JSON: %v", err)
			}
			if got.Recommended != tt.wantRecommended || got.Reasoning == "" || len(got.Alternatives) != 2 {
				t.Errorf("HandleBannerRec() = %+v, want recommended %q with reasoning and 2 alternatives", got, tt.wantRecommended)
			}
		})
	}
}

func TestHandleVariation(t *testing.T) {
	tests := []struct {
		name       string
		llmURL     string // "" -> mock mode
		request    *http.Request
		wantStatus int
		wantCount  int
	}{
		{name: "mock variations", request: jsonRequest(http.MethodPost, "/api/variations", textBody("hello")), wantStatus: http.StatusOK, wantCount: 4},
		{name: "empty text", request: jsonRequest(http.MethodPost, "/api/variations", textBody("")), wantStatus: http.StatusBadRequest},
		{name: "cyrillic", request: jsonRequest(http.MethodPost, "/api/variations", textBody("Привет")), wantStatus: http.StatusBadRequest},
		{name: "GET is not allowed", request: jsonRequest(http.MethodGet, "/api/variations", ""), wantStatus: http.StatusMethodNotAllowed},
		{
			name:       "LLM is down",
			llmURL:     "http://127.0.0.1:1",
			request:    jsonRequest(http.MethodPost, "/api/variations", textBody("hello")),
			wantStatus: http.StatusServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LLM_BASE_URL", tt.llmURL)
			t.Setenv("LLM_MODEL", "test-model")
			rec := httptest.NewRecorder()
			HandleVariation(rec, tt.request)

			if rec.Code != tt.wantStatus {
				t.Fatalf("HandleVariation() status = %d, want %d, body %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantStatus != http.StatusOK {
				checkJSONError(t, rec)
				return
			}
			var got struct {
				Variations []struct {
					Text            string `json:"text"`
					Description     string `json:"description"`
					SuggestedBanner string `json:"suggested_banner"`
				} `json:"variations"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("HandleVariation() body is not JSON: %v", err)
			}
			if len(got.Variations) != tt.wantCount {
				t.Errorf("HandleVariation() returned %d variations, want %d", len(got.Variations), tt.wantCount)
			}
			for _, v := range got.Variations {
				if v.Text == "" || v.Description == "" || v.SuggestedBanner == "" {
					t.Errorf("HandleVariation() variation %+v has empty fields", v)
				}
			}
		})
	}
}

func Test_readTextRequest(t *testing.T) {
	noType := httptest.NewRequest(http.MethodPost, "/api/suggest", strings.NewReader(textBody("hi")))
	withCharset := jsonRequest(http.MethodPost, "/api/suggest", textBody("hi"))
	withCharset.Header.Set("Content-Type", "application/json; charset=utf-8")

	tests := []struct {
		name       string
		request    *http.Request
		wantText   string
		wantOk     bool
		wantStatus int // checked only when wantOk is false
	}{
		{name: "valid body", request: jsonRequest(http.MethodPost, "/", textBody("hello")), wantText: "hello", wantOk: true},
		{name: "charset in Content-Type", request: withCharset, wantText: "hi", wantOk: true},
		{name: "spaces after JSON", request: jsonRequest(http.MethodPost, "/", textBody("hello")+"  \n"), wantText: "hello", wantOk: true},
		{name: "GET", request: jsonRequest(http.MethodGet, "/", ""), wantStatus: http.StatusMethodNotAllowed},
		{name: "no Content-Type", request: noType, wantStatus: http.StatusUnsupportedMediaType},
		{name: "broken JSON", request: jsonRequest(http.MethodPost, "/", "{"), wantStatus: http.StatusBadRequest},
		{name: "unknown field", request: jsonRequest(http.MethodPost, "/", `{"txt":"hello"}`), wantStatus: http.StatusBadRequest},
		{name: "text is a number", request: jsonRequest(http.MethodPost, "/", `{"text":5}`), wantStatus: http.StatusBadRequest},
		{name: "two JSON objects", request: jsonRequest(http.MethodPost, "/", `{"text":"a"}{"text":"b"}`), wantStatus: http.StatusBadRequest},
		{name: "body larger than 8 KiB", request: jsonRequest(http.MethodPost, "/", textBody(strings.Repeat("a", 9000))), wantStatus: http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			gotText, gotOk := readTextRequest(rec, tt.request)
			if gotText != tt.wantText || gotOk != tt.wantOk {
				t.Fatalf("readTextRequest() = (%q, %v), want (%q, %v)", gotText, gotOk, tt.wantText, tt.wantOk)
			}
			if tt.wantOk {
				return
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("readTextRequest() status = %d, want %d", rec.Code, tt.wantStatus)
			}
			checkJSONError(t, rec)
			if tt.wantStatus == http.StatusMethodNotAllowed && rec.Header().Get("Allow") != http.MethodPost {
				t.Errorf("readTextRequest() Allow header = %q, want POST", rec.Header().Get("Allow"))
			}
		})
	}
}

func Test_writeVariationError(t *testing.T) {
	type args struct {
		status  int
		message string
	}
	tests := []struct {
		name     string
		args     args
		wantBody string
	}{
		{name: "bad request", args: args{status: http.StatusBadRequest, message: "Text is empty"}, wantBody: `{"error":"Text is empty"}`},
		{name: "service unavailable", args: args{status: http.StatusServiceUnavailable, message: "Try later"}, wantBody: `{"error":"Try later"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeVariationError(rec, tt.args.status, tt.args.message)
			if rec.Code != tt.args.status {
				t.Errorf("writeVariationError() status = %d, want %d", rec.Code, tt.args.status)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("writeVariationError() body = %s, want %s", rec.Body, tt.wantBody)
			}
		})
	}
}

func Test_writeVariationJSON(t *testing.T) {
	type args struct {
		status  int
		payload any
	}
	tests := []struct {
		name       string
		args       args
		wantStatus int
		wantBody   string
	}{
		{
			name:       "object",
			args:       args{status: http.StatusOK, payload: map[string][]string{"suggestions": {"a", "b"}}},
			wantStatus: http.StatusOK,
			wantBody:   `{"suggestions":["a","b"]}`,
		},
		{
			name:       "payload that cannot be encoded",
			args:       args{status: http.StatusOK, payload: make(chan int)},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"Could not create ideas. Please try again."}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeVariationJSON(rec, tt.args.status, tt.args.payload)
			if rec.Code != tt.wantStatus {
				t.Errorf("writeVariationJSON() status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("writeVariationJSON() body = %s, want %s", rec.Body, tt.wantBody)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("writeVariationJSON() Content-Type = %q", got)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("writeVariationJSON() Cache-Control = %q, want no-store", got)
			}
		})
	}
}

func Test_validateArtText(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		{name: "simple text", args: args{input: "Hello"}, want: true, want1: ""},
		{name: "line break is allowed", args: args{input: "a\nb"}, want: true, want1: ""},
		{name: "1000 characters", args: args{input: strings.Repeat("a", 1000)}, want: true, want1: ""},
		{name: "empty", args: args{input: ""}, want: false, want1: "Text is empty"},
		{name: "only spaces", args: args{input: "   "}, want: false, want1: "Text is empty"},
		{name: "1001 characters", args: args{input: strings.Repeat("a", 1001)}, want: false, want1: "Text is too long (max 1000 characters)"},
		{name: "cyrillic", args: args{input: "Привет"}, want: false, want1: "Invalid character"},
		{name: "tab", args: args{input: "a\tb"}, want: false, want1: "Invalid character"},
		{name: "windows line break", args: args{input: "a\r\nb"}, want: false, want1: "Invalid character"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := validateArtText(tt.args.input)
			if got != tt.want {
				t.Errorf("validateArtText() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("validateArtText() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
