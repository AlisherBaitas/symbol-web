package handlers

// [Fix] Файл handlers_test.go требуется структурой проекта, но содержал только
// "package handlers". Здесь проверяется маршрутизация целиком (NewRouter):
// коды 404/405/303, статика, ошибки шаблонов и параллельные запросы.

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// serve отправляет запрос через настоящий роутер приложения
func serve(r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	NewRouter().ServeHTTP(rec, r)
	return rec
}

func TestNewRouter_Routes(t *testing.T) {
	chdirProjectRoot(t)

	tests := []struct {
		name        string
		request     *http.Request
		wantStatus  int
		wantContain string
	}{
		{name: "main page", request: httptest.NewRequest(http.MethodGet, "/", nil), wantStatus: http.StatusOK, wantContain: `id="text-input"`},
		{name: "unknown page", request: httptest.NewRequest(http.MethodGet, "/nope", nil), wantStatus: http.StatusNotFound, wantContain: "404"},
		{name: "unknown nested page", request: httptest.NewRequest(http.MethodGet, "/a/b/c", nil), wantStatus: http.StatusNotFound},
		{name: "GET /symbol-art redirects home", request: httptest.NewRequest(http.MethodGet, "/symbol-art", nil), wantStatus: http.StatusSeeOther},
		{name: "POST /symbol-art", request: formRequest("Hi", "standard.txt"), wantStatus: http.StatusOK, wantContain: `id="art-output"`},
		{name: "GET on API endpoint", request: httptest.NewRequest(http.MethodGet, "/api/suggest", nil), wantStatus: http.StatusMethodNotAllowed, wantContain: `"error"`},
		{name: "unknown API endpoint", request: httptest.NewRequest(http.MethodGet, "/api/nope", nil), wantStatus: http.StatusNotFound, wantContain: `"error"`},
		{name: "POST to unknown page", request: httptest.NewRequest(http.MethodPost, "/nope", nil), wantStatus: http.StatusMethodNotAllowed},
		{name: "stylesheet", request: httptest.NewRequest(http.MethodGet, "/static/css/style.css", nil), wantStatus: http.StatusOK},
		{name: "script", request: httptest.NewRequest(http.MethodGet, "/static/js/recommendations.js", nil), wantStatus: http.StatusOK},
		{name: "missing static file", request: httptest.NewRequest(http.MethodGet, "/static/js/nope.js", nil), wantStatus: http.StatusNotFound},
		{name: "suggest", request: jsonRequest(http.MethodPost, "/api/suggest", textBody("Hello")), wantStatus: http.StatusOK, wantContain: `"suggestions"`},
		{name: "recommend", request: jsonRequest(http.MethodPost, "/api/recommend-banner", textBody("WELCOME")), wantStatus: http.StatusOK, wantContain: `"recommended":"shadow"`},
		{name: "variations", request: jsonRequest(http.MethodPost, "/api/variations", textBody("hello")), wantStatus: http.StatusOK, wantContain: `"variations"`},
	}
	t.Setenv("LLM_BASE_URL", "") // mock-режим
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(tt.request)
			if rec.Code != tt.wantStatus {
				t.Fatalf("%s %s status = %d, want %d", tt.request.Method, tt.request.URL.Path, rec.Code, tt.wantStatus)
			}
			if tt.wantContain != "" && !strings.Contains(rec.Body.String(), tt.wantContain) {
				t.Errorf("body does not contain %q", tt.wantContain)
			}
		})
	}
}

func TestNewRouter_AuditExample(t *testing.T) {
	chdirProjectRoot(t)

	// Пример из аудита, введённый в textarea двумя строками (браузер шлёт CRLF).
	rec := serve(formRequest("{123}\r\n<Hello> (World)!", "standard.txt"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	for _, line := range []string{
		`  / /  _   ____    _____  \ \`,
		`&lt; &lt;   |  __  |  / _ \ | | | |  / _ \    &gt; &gt;`,
	} {
		if !strings.Contains(rec.Body.String(), line) {
			t.Errorf("page does not contain the line %q", line)
		}
	}
}

func TestTemplates_MissingFiles(t *testing.T) {
	// Пустая папка без templates/: по ТЗ шаблон не найден -> 404.
	t.Chdir(t.TempDir())

	if rec := serve(httptest.NewRequest(http.MethodGet, "/", nil)); rec.Code != http.StatusNotFound {
		t.Errorf("GET / without templates: status = %d, want 404", rec.Code)
	}
	// Страница ошибки тоже не найдена — код ошибки всё равно сохраняется.
	if rec := serve(httptest.NewRequest(http.MethodGet, "/nope", nil)); rec.Code != http.StatusNotFound {
		t.Errorf("GET /nope without templates: status = %d, want 404", rec.Code)
	}
	if rec := serve(formRequest("Hi", "nope")); rec.Code != http.StatusBadRequest {
		t.Errorf("bad banner without templates: status = %d, want 400", rec.Code)
	}
}

func TestTemplates_ExecutionError(t *testing.T) {
	// Шаблон есть, но выполнение падает -> 500.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"index.html":  `{{template "result" .}}{{.Missing}}`,
		"result.html": `{{define "result"}}ok{{end}}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, "templates", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)

	if rec := serve(httptest.NewRequest(http.MethodGet, "/", nil)); rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestBannerFile_Missing(t *testing.T) {
	// Шаблоны есть, а файла баннера нет -> 404 по ТЗ.
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "templates"), filepath.Join(dir, "templates")); err != nil {
		t.Skip("symlinks are not available:", err)
	}
	t.Chdir(dir)

	rec := serve(formRequest("Hi", "standard.txt"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestNewRouter_ConcurrentRequests(t *testing.T) {
	// Раньше генератор писал в общую глобальную переменную: при параллельных
	// запросах ответы смешивались. Запускайте с -race.
	chdirProjectRoot(t)
	srv := httptest.NewServer(NewRouter())
	t.Cleanup(srv.Close)

	var wg sync.WaitGroup
	errs := make(chan string, 100)
	for i := 0; i < 50; i++ {
		for _, banner := range []string{"standard.txt", "shadow.txt"} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				text := fmt.Sprintf("Ab%d", i)
				resp, err := http.PostForm(srv.URL+"/symbol-art", url.Values{"inputWord": {text}, "banners": {banner}})
				if err != nil {
					errs <- err.Error()
					return
				}
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				page := string(body)
				start := strings.Index(page, `<pre id="art-output"`)
				end := strings.Index(page[start:], "</pre>")
				if resp.StatusCode != http.StatusOK || start < 0 || end < 0 {
					errs <- fmt.Sprintf("%s: status %d", text, resp.StatusCode)
					return
				}
				// Ровно 8 строк арта для одной строки текста.
				art := page[start : start+end]
				art = art[strings.Index(art, ">")+1:]
				if n := strings.Count(art, "\n"); n != 8 {
					errs <- fmt.Sprintf("%s (%s): %d lines, want 8", text, banner, n)
				}
			}()
		}
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}
