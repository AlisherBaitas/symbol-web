package handlers

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"symbol-web/internal/ascii"
)

func artPageFixture(t *testing.T) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	if err := os.CopyFS(filepath.Join(fixture, "templates"), os.DirFS(filepath.Join(root, "templates"))); err != nil {
		t.Fatal(err)
	}
	for _, banner := range []string{"standard.txt", "shadow.txt", "thinkertoy.txt"} {
		data, err := os.ReadFile(filepath.Join(root, banner))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(fixture, banner), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(fixture)
}

func artPageElement(t *testing.T, page, tag, id string) (attributes, content string) {
	t.Helper()
	pattern := `(?s)<` + regexp.QuoteMeta(tag) + `\b([^>]*\bid="` + regexp.QuoteMeta(id) + `"[^>]*)>(.*?)</` + regexp.QuoteMeta(tag) + `>`
	match := regexp.MustCompile(pattern).FindStringSubmatch(page)
	if match == nil {
		t.Fatalf("missing %s#%s", tag, id)
	}
	return match[1], match[2]
}

func artPageText(content string) string {
	// Browsers discard the first literal newline after a textarea/pre start tag.
	return html.UnescapeString(strings.TrimPrefix(content, "\n"))
}

func assertArtEditor(t *testing.T, page, text, banner string) {
	t.Helper()
	form, _ := artPageElement(t, page, "form", "art-form")
	for _, attribute := range []string{`action="/symbol-art"`, `method="post"`} {
		if !strings.Contains(form, attribute) {
			t.Errorf("editor form is missing %s", attribute)
		}
	}
	for _, script := range []string{"suggestions.js", "variations.js"} {
		if !strings.Contains(page, `src="/static/js/`+script+`"`) {
			t.Errorf("editor is missing %s", script)
		}
	}
	_, textarea := artPageElement(t, page, "textarea", "text-input")
	if got := artPageText(textarea); got != text {
		t.Errorf("textarea = %q, want %q", got, text)
	}
	checked := regexp.MustCompile(`\bchecked(?:\s|>|=)`)
	var selected []string
	for _, input := range regexp.MustCompile(`<input\b[^>]*\bname="banners"[^>]*>`).FindAllString(page, -1) {
		if checked.MatchString(input) {
			value := regexp.MustCompile(`\bvalue="([^"]*)"`).FindStringSubmatch(input)
			if value == nil {
				t.Fatal("selected banner is missing its value")
			}
			selected = append(selected, value[1])
		}
	}
	if len(selected) != 1 || selected[0] != banner {
		t.Errorf("selected banners = %v, want [%s]", selected, banner)
	}
}

func TestArtPageInitiallyShowsEditor(t *testing.T) {
	artPageFixture(t)
	response := httptest.NewRecorder()
	HandleRoot(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", response.Code)
	}
	page := response.Body.String()
	assertArtEditor(t, page, "", "standard.txt")
	output, content := artPageElement(t, page, "pre", "art-output")
	if !regexp.MustCompile(`\bhidden\b`).MatchString(output) || artPageText(content) != "" {
		t.Errorf("initial output should be empty and hidden; attributes = %q, content = %q", output, content)
	}
	empty, _ := artPageElement(t, page, "div", "empty-state")
	if regexp.MustCompile(`\bhidden\b`).MatchString(empty) {
		t.Error("initial empty state is hidden")
	}
}

func TestArtPagePostKeepsEditorAndShowsSelectedBanner(t *testing.T) {
	artPageFixture(t)
	for _, banner := range []string{"standard.txt", "shadow.txt", "thinkertoy.txt"} {
		t.Run(banner, func(t *testing.T) {
			assertArtPost(t, "Hi", "Hi", banner)
		})
	}
}

func TestArtPagePostPreservesMultilineTextSafely(t *testing.T) {
	artPageFixture(t)
	for _, input := range []string{
		"</textarea><script>alert(\"x\")</script> & 'quoted'\r\nHi",
		"\r\nHi\r\n",
	} {
		t.Run(input, func(t *testing.T) {
			page := assertArtPost(t, input, strings.ReplaceAll(input, "\r\n", "\n"), "thinkertoy.txt")
			if strings.Contains(page, `<script>alert("x")</script>`) {
				t.Error("entered markup was rendered as executable HTML")
			}
			_, textarea := artPageElement(t, page, "textarea", "text-input")
			if strings.ContainsAny(textarea, "<>") {
				t.Error("textarea contains unescaped markup")
			}
		})
	}
}

func assertArtPost(t *testing.T, input, normalized, banner string) string {
	t.Helper()
	form := url.Values{"inputWord": {input}, "banners": {banner}}
	request := httptest.NewRequest(http.MethodPost, "/symbol-art", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	HandleSymbolArt(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("POST /symbol-art status = %d, want 200; body: %s", response.Code, response.Body.String())
	}
	page := response.Body.String()
	assertArtEditor(t, page, normalized, banner)
	output, content := artPageElement(t, page, "pre", "art-output")
	if regexp.MustCompile(`\bhidden\b`).MatchString(output) {
		t.Error("generated output is hidden")
	}
	want, err := ascii.Generate(normalized, banner)
	if err != nil {
		t.Fatal(err)
	}
	if got := artPageText(content); got == "" || got != want {
		t.Errorf("displayed output does not match generated %s art", banner)
	}
	empty, _ := artPageElement(t, page, "div", "empty-state")
	if !regexp.MustCompile(`\bhidden\b`).MatchString(empty) {
		t.Error("empty state is still visible after generation")
	}
	return page
}

func TestArtPagePostRetainsValidationStatuses(t *testing.T) {
	artPageFixture(t)
	cases := []struct {
		name   string
		body   string
		status int
	}{
		{"empty text", "inputWord=&banners=standard.txt", http.StatusBadRequest},
		{"missing banner", "inputWord=Hi", http.StatusBadRequest},
		{"unknown banner", "inputWord=Hi&banners=other.txt", http.StatusBadRequest},
		{"unsupported character", "inputWord=%09&banners=standard.txt", http.StatusBadRequest},
		{"malformed form", "inputWord=%zz&banners=standard.txt", http.StatusBadRequest},
		{"text limit", "inputWord=" + strings.Repeat("A", 1001) + "&banners=standard.txt", http.StatusBadRequest},
		{"body limit", "inputWord=" + strings.Repeat("A", 16*1024) + "&banners=standard.txt", http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/symbol-art", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			HandleSymbolArt(response, request)
			if response.Code != tc.status {
				t.Errorf("status = %d, want %d", response.Code, tc.status)
			}
		})
	}
}
