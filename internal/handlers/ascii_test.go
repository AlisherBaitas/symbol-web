package handlers

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Handlers read templates/ and banner files by relative path, so tests run from the project root
func chdirProjectRoot(t *testing.T) {
	t.Helper()
	t.Chdir("../..")
}

// formRequest builds a POST /symbol-art request like the HTML form does
func formRequest(text, banner string) *http.Request {
	form := url.Values{"inputWord": {text}, "banners": {banner}}
	r := httptest.NewRequest(http.MethodPost, "/symbol-art", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestHandleRoot(t *testing.T) {
	chdirProjectRoot(t)

	tests := []struct {
		name         string
		wantStatus   int
		wantContains []string
	}{
		{
			name:       "main page",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`id="text-input"`,
				`name="inputWord"`,
				`value="standard.txt" checked`,
				`value="shadow.txt"`,
				`value="thinkertoy.txt"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			HandleRoot(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("HandleRoot() status = %d, want %d", rec.Code, tt.wantStatus)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(rec.Body.String(), s) {
					t.Errorf("HandleRoot() page does not contain %q", s)
				}
			}
		})
	}
}

func TestHandleSymbolArt(t *testing.T) {
	chdirProjectRoot(t)

	tests := []struct {
		name         string
		request      *http.Request
		wantStatus   int
		wantContains []string
	}{
		{
			name:       "standard banner",
			request:    formRequest("Hi", "standard.txt"),
			wantStatus: http.StatusOK,
			// Первая и вторая строки "Hi" в standard
			wantContains: []string{html.EscapeString(" _    _   _  "), html.EscapeString("| |  | | (_) ")},
		},
		{
			name:         "shadow banner",
			request:      formRequest("A", "shadow.txt"),
			wantStatus:   http.StatusOK,
			wantContains: []string{"_|_|_|_| "},
		},
		{
			name:         "thinkertoy banner",
			request:      formRequest("A", "thinkertoy.txt"),
			wantStatus:   http.StatusOK,
			wantContains: []string{"o---o "},
		},
		{name: "empty text", request: formRequest("", "standard.txt"), wantStatus: http.StatusBadRequest},
		{name: "cyrillic text", request: formRequest("Привет", "standard.txt"), wantStatus: http.StatusBadRequest},
		{name: "text too long", request: formRequest(strings.Repeat("a", 1001), "standard.txt"), wantStatus: http.StatusBadRequest},
		{name: "unknown banner", request: formRequest("Hi", "comic.txt"), wantStatus: http.StatusNotFound},
		{name: "no banner", request: formRequest("Hi", ""), wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			HandleSymbolArt(rec, tt.request)

			if rec.Code != tt.wantStatus {
				t.Fatalf("HandleSymbolArt() status = %d, want %d", rec.Code, tt.wantStatus)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(rec.Body.String(), s) {
					t.Errorf("HandleSymbolArt() page does not contain %q", s)
				}
			}
		})
	}
}
