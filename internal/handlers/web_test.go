package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"symbol-web/internal/ai"
)

func testFiles(t *testing.T) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	for _, name := range []string{"standard.txt", "shadow.txt", "thinkertoy.txt", "templates/studio.html", "templates/studio-error.html", "static/css/studio.css", "static/js/studio.js"} {
		data, err := os.ReadFile("../../" + name)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = &fstest.MapFile{Data: data, Mode: 0644}
	}
	return files
}

func mockApp(t *testing.T, files fs.FS) http.Handler {
	t.Helper()
	t.Setenv("LLM_BASE_URL", "")
	return NewWeb(files, ai.NewService(ai.NewAIClient()), nil)
}

func perform(app http.Handler, method, path, body, contentType string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	return w
}

func TestRoutesMethodsAndHeaders(t *testing.T) {
	app := mockApp(t, testFiles(t))
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/", 200}, {"HEAD", "/", 200}, {"POST", "/", 405}, {"GET", "/missing", 404}, {"GET", "/templates/index.html", 404}, {"GET", "/static/", 404}, {"GET", "/static/css/studio.css", 200}, {"GET", "/static/js/studio.js", 200}, {"POST", "/static/css/studio.css", 405}, {"GET", "/symbol-art", 405}, {"GET", "/api/suggest", 405}, {"POST", "/api/missing", 404}, {"GET", "/.env.example", 404}, {"GET", "/api/suggest/", 404}, {"GET", "/static/../go.mod", 404}} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := perform(app, tc.method, tc.path, "", "")
			if w.Code != tc.status {
				t.Fatalf("got %d want %d: %s", w.Code, tc.status, w.Body.String())
			}
			if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Content-Security-Policy") == "" {
				t.Fatal("нет защитных заголовков")
			}
			if tc.status == 405 && w.Header().Get("Allow") == "" {
				t.Fatal("нет Allow")
			}
			if tc.method == "HEAD" && w.Body.Len() != 0 {
				t.Fatal("тело HEAD")
			}
			if strings.HasPrefix(tc.path, "/api/") && !json.Valid(w.Body.Bytes()) {
				t.Fatal("API вернул не JSON")
			}
		})
	}
}

func TestFormAndEscaping(t *testing.T) {
	app := mockApp(t, testFiles(t))
	for _, banner := range []string{"standard", "shadow", "thinkertoy", "standard.txt"} {
		body := url.Values{"inputWord": {"<script>alert('x')</script>\nHello"}, "banners": {banner}}.Encode()
		w := perform(app, "POST", "/symbol-art", body, "application/x-www-form-urlencoded")
		if w.Code != 200 || strings.Contains(w.Body.String(), "<script>alert") || !strings.Contains(w.Body.String(), "&lt;script&gt;") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("POST", "/symbol-art", strings.NewReader("inputWord=Hi&banners=shadow"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	var result map[string]string
	if json.Unmarshal(w.Body.Bytes(), &result) != nil || result["art"] == "" || result["banner"] != "shadow" {
		t.Fatal(w.Body.String())
	}
	for _, body := range []string{"inputWord=&banners=standard", "inputWord=Hi&banners=../../etc/passwd", "inputWord=Hi", "inputWord=Hi&inputWord=Bye&banners=shadow", "inputWord=%00&banners=standard", url.Values{"inputWord": {"Привет"}, "banners": {"standard"}}.Encode(), "inputWord=" + strings.Repeat("x", 1001) + "&banners=standard", strings.Repeat("x", 20000)} {
		if w := perform(app, "POST", "/symbol-art", body, "application/x-www-form-urlencoded"); w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	if w := perform(app, "POST", "/symbol-art", "{}", "application/json"); w.Code != 415 {
		t.Fatal(w.Code)
	}
	if w := perform(app, "POST", "/symbol-art?inputWord=Hi&banners=standard", "", "application/x-www-form-urlencoded"); w.Code != 400 {
		t.Fatal("параметры URL заменили форму", w.Code)
	}
}

func TestAPIValidationAndContract(t *testing.T) {
	app := mockApp(t, testFiles(t))
	for _, endpoint := range []string{"/api/suggest", "/api/recommend-banner", "/api/variations"} {
		for _, body := range []string{`{}`, `null`, `[]`, `{"text":null}`, `{"text":2}`, `{"text":""}`, `{"text":"   "}`, `{"text":"Привет"}`, `{"text":"Hello","extra":true}`, `{"text":"Hello"} {}`, `{"text":"Hello"} garbage`, `{"text":"\u0000"}`, `{"text":"` + strings.Repeat("a", 1001) + `"}`, strings.Repeat("x", 20000)} {
			w := perform(app, "POST", endpoint, body, "application/json")
			if w.Code != 400 || !json.Valid(w.Body.Bytes()) {
				t.Errorf("%s %q: %d", endpoint, body[:min(40, len(body))], w.Code)
			}
		}
		w := perform(app, "POST", endpoint, `{"text":"WELCOME"}`, "application/json; charset=utf-8")
		if w.Code != 200 || !json.Valid(w.Body.Bytes()) {
			t.Fatal(endpoint, w.Code, w.Body.String())
		}
		if w := perform(app, "POST", endpoint, `{"text":"Hi"}`, "text/plain"); w.Code != 415 {
			t.Fatal(w.Code)
		}
	}
	if w := perform(app, "POST", "/api/suggest", `{"text":"Hi"}`, "application/json"); w.Code != 400 {
		t.Fatal(w.Code)
	}
	first := perform(app, "POST", "/api/suggest", `{"text":"Hello"}`, "application/json")
	second := perform(app, "POST", "/api/suggest", `{"text":"Hello"}`, "application/json")
	if first.Body.String() != second.Body.String() || first.Header().Get("X-AI-Mode") != "mock" {
		t.Fatal("недетерминированный mock")
	}
}

type failingAI struct {
	err      error
	panicNow bool
}

func (f failingAI) Mock() bool { return false }
func (f failingAI) Suggest(context.Context, string) ([]string, error) {
	if f.panicNow {
		panic("test")
	}
	return nil, f.err
}
func (f failingAI) Variations(context.Context, string) ([]ai.Variation, error) { return nil, f.err }

func TestAIFailureIsolation(t *testing.T) {
	for _, tc := range []struct {
		client failingAI
		status int
	}{{failingAI{err: ai.ErrUnavailable}, 503}, {failingAI{err: ai.ErrResponse}, 500}, {failingAI{err: ai.ErrConfiguration}, 500}, {failingAI{err: context.DeadlineExceeded}, 503}, {failingAI{panicNow: true}, 500}, {failingAI{err: errors.New("private upstream details")}, 500}} {
		app := NewWeb(testFiles(t), tc.client, nil)
		w := perform(app, "POST", "/api/suggest", `{"text":"Hello"}`, "application/json")
		if w.Code != tc.status || !json.Valid(w.Body.Bytes()) || strings.Contains(w.Body.String(), "private") {
			t.Fatal(w.Code, w.Body.String())
		}
		for _, path := range []string{"/api/recommend-banner", "/symbol-art", "/"} {
			method, body, ct := "POST", `{"text":"WELCOME"}`, "application/json"
			if path == "/symbol-art" {
				body = "inputWord=Hi&banners=standard"
				ct = "application/x-www-form-urlencoded"
			}
			if path == "/" {
				method = "GET"
				body = ""
				ct = ""
			}
			if w := perform(app, method, path, body, ct); w.Code != 200 {
				t.Fatal("сбой AI повредил другой маршрут", path, w.Code)
			}
		}
	}
}

func TestMissingAndBrokenResources(t *testing.T) {
	for _, tc := range []struct {
		file, body, path string
		status           int
	}{{"templates/studio.html", "", "/", 404}, {"templates/studio.html", "{{", "/", 500}, {"templates/studio.html", "partial {{.Missing}}", "/", 500}, {"standard.txt", "", "/symbol-art", 404}, {"standard.txt", "broken", "/symbol-art", 500}, {"static/css/studio.css", "", "/static/css/studio.css", 404}} {
		files := testFiles(t)
		if tc.body == "" {
			delete(files, tc.file)
		} else {
			files[tc.file] = &fstest.MapFile{Data: []byte(tc.body)}
		}
		app := mockApp(t, files)
		method, body, ct := "GET", "", ""
		if tc.path == "/symbol-art" {
			method = "POST"
			body = "inputWord=Hi&banners=standard"
			ct = "application/x-www-form-urlencoded"
		}
		w := perform(app, method, tc.path, body, ct)
		if w.Code != tc.status || strings.HasPrefix(w.Body.String(), "partial") {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
	files := testFiles(t)
	delete(files, "templates/studio-error.html")
	if w := perform(mockApp(t, files), "GET", "/missing", "", ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
}

func TestConcurrentHTTP(t *testing.T) {
	app := mockApp(t, testFiles(t))
	var wg sync.WaitGroup
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var w *httptest.ResponseRecorder
			if i%3 == 0 {
				w = perform(app, "POST", "/symbol-art", fmt.Sprintf("inputWord=Hello%d&banners=shadow", i), "application/x-www-form-urlencoded")
			} else {
				w = perform(app, "POST", "/api/suggest", fmt.Sprintf(`{"text":"Hello %d"}`, i), "application/json")
			}
			if w.Code != 200 {
				t.Errorf("HTTP %d", w.Code)
			}
		}(i)
	}
	wg.Wait()
}

func FuzzAPI(f *testing.F) {
	f.Setenv("LLM_BASE_URL", "")
	app := NewWeb(os.DirFS("../.."), ai.NewService(ai.NewAIClient()), nil)
	for _, body := range []string{`{"text":"Hello"}`, `{}`, `null`, `{"text":"\u0000"}`, `{"text":"Hi"} {}`} {
		f.Add(body)
	}
	f.Fuzz(func(t *testing.T, body string) {
		w := perform(app, "POST", "/api/suggest", body, "application/json")
		if !json.Valid(w.Body.Bytes()) {
			t.Fatal("ответ не JSON")
		}
		if w.Code != 200 && w.Code != 400 {
			t.Fatalf("неожиданный код %d", w.Code)
		}
	})
}
