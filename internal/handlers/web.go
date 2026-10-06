package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"strings"

	"symbol-web/internal/ai"
	"symbol-web/internal/ascii"
)

type Assistant interface {
	Mock() bool
	Suggest(context.Context, string) ([]string, error)
	Variations(context.Context, string) ([]ai.Variation, error)
}

type Web struct {
	files     fs.FS
	assistant Assistant
	logger    *log.Logger
}

type pageData struct {
	Text, Banner, Art, Mode, Message string
	Status                           int
}

// NewWeb подключает надстройку в существующем слое handlers.
func NewWeb(files fs.FS, assistant Assistant, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &Web{files, assistant, logger}
}

func (app *Web) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
	w.Header().Set("Cache-Control", "no-store")
	defer func() {
		if recovered := recover(); recovered != nil {
			app.logger.Printf("ошибка обработчика: %T", recovered)
			app.fail(w, r, http.StatusInternalServerError, "Не удалось выполнить запрос. Попробуйте ещё раз.")
		}
	}()
	wanted := http.MethodPost
	switch r.URL.Path {
	case "/", "/static/css/studio.css", "/static/js/studio.js":
		wanted = http.MethodGet
	case "/symbol-art", "/api/suggest", "/api/recommend-banner", "/api/variations":
	default:
		app.fail(w, r, http.StatusNotFound, "Такой страницы нет. Вернитесь к генератору.")
		return
	}
	if r.Method != wanted && !(wanted == http.MethodGet && r.Method == http.MethodHead) {
		allow := wanted
		if wanted == http.MethodGet {
			allow = "GET, HEAD"
		}
		w.Header().Set("Allow", allow)
		app.fail(w, r, http.StatusMethodNotAllowed, "Этот способ запроса не поддерживается.")
		return
	}
	switch r.URL.Path {
	case "/":
		app.page(w, r, pageData{Banner: "standard"})
	case "/symbol-art":
		app.art(w, r)
	case "/api/suggest", "/api/recommend-banner", "/api/variations":
		app.api(w, r)
	default:
		content, err := fs.ReadFile(app.files, strings.TrimPrefix(r.URL.Path, "/"))
		if err != nil {
			app.fileError(w, r, err)
			return
		}
		contentType := "text/css; charset=utf-8"
		if strings.HasSuffix(r.URL.Path, ".js") {
			contentType = "text/javascript; charset=utf-8"
		}
		w.Header().Set("Content-Type", contentType)
		if r.Method != http.MethodHead {
			_, _ = w.Write(content)
		}
	}
}

func (app *Web) page(w http.ResponseWriter, r *http.Request, data pageData) {
	data.Mode = "live"
	if app.assistant.Mock() {
		data.Mode = "mock"
	}
	content, err := app.render("templates/studio.html", data)
	if err != nil {
		app.fileError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method != http.MethodHead {
		_, _ = w.Write(content)
	}
}

func (app *Web) render(name string, data pageData) ([]byte, error) {
	content, err := fs.ReadFile(app.files, name)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New(name).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (app *Web) art(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		app.fail(w, r, 415, "Отправьте текст и стиль через форму.")
		return
	}
	if err := r.ParseForm(); err != nil {
		app.fail(w, r, 400, "Форма слишком большая или повреждена.")
		return
	}
	text, banner := r.PostForm.Get("inputWord"), r.PostForm.Get("banners")
	if len(r.PostForm["inputWord"]) != 1 || len(r.PostForm["banners"]) != 1 {
		app.fail(w, r, 400, "Укажите один текст и один стиль.")
		return
	}
	art, err := ascii.Render(app.files, text, banner)
	if err != nil {
		if errors.Is(err, ascii.ErrInvalidText) || errors.Is(err, ascii.ErrInvalidBanner) {
			app.fail(w, r, 400, err.Error())
		} else {
			app.fileError(w, r, err)
		}
		return
	}
	banner, _ = ascii.BannerName(banner)
	if wantsJSON(r) {
		app.json(w, 200, map[string]string{"art": art, "text": text, "banner": banner})
		return
	}
	app.page(w, r, pageData{Text: text, Banner: banner, Art: art})
}

func (app *Web) api(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		app.fail(w, r, 415, "Ожидается JSON с полем text.")
		return
	}
	var input struct {
		Text *string `json:"text"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || input.Text == nil {
		app.fail(w, r, 400, "Передайте JSON с текстовым полем text.")
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		app.fail(w, r, 400, "Ожидается ровно один JSON-объект.")
		return
	}
	text, err := ascii.Normalize(*input.Text)
	if err != nil || strings.TrimSpace(text) == "" {
		app.fail(w, r, 400, ascii.ErrInvalidText.Error())
		return
	}
	if r.URL.Path == "/api/suggest" && len(strings.TrimSpace(text)) < 3 {
		app.fail(w, r, 400, "Для подсказок введите хотя бы 3 символа.")
		return
	}
	if r.URL.Path == "/api/recommend-banner" {
		app.json(w, 200, ai.Recommend(text))
		return
	}
	w.Header().Set("X-AI-Mode", "live")
	if app.assistant.Mock() {
		w.Header().Set("X-AI-Mode", "mock")
	}
	if r.URL.Path == "/api/suggest" {
		values, err := app.assistant.Suggest(r.Context(), text)
		if err != nil {
			app.aiError(w, r, err)
			return
		}
		app.json(w, 200, map[string]any{"suggestions": values})
		return
	}
	values, err := app.assistant.Variations(r.Context(), text)
	if err != nil {
		app.aiError(w, r, err)
		return
	}
	app.json(w, 200, map[string]any{"variations": values})
}

func (app *Web) aiError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Printf("сбой AI: %v", err)
	if errors.Is(err, ai.ErrUnavailable) || errors.Is(err, context.DeadlineExceeded) {
		app.fail(w, r, 503, "Подсказки временно недоступны. Генератор и подбор стиля продолжают работать.")
		return
	}
	app.fail(w, r, 500, "Не удалось получить корректные подсказки. Попробуйте позже.")
}

func (app *Web) fileError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Printf("ошибка ресурса: %v", err)
	if errors.Is(err, fs.ErrNotExist) {
		app.fail(w, r, 404, "Не найден файл интерфейса или баннера. Проверьте комплектность проекта.")
		return
	}
	app.fail(w, r, 500, "Не удалось обработать ресурс сервера. Попробуйте позже.")
}

func wantsJSON(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/") || strings.Contains(r.Header.Get("Accept"), "application/json")
}

func (app *Web) fail(w http.ResponseWriter, r *http.Request, status int, message string) {
	if wantsJSON(r) {
		app.json(w, status, map[string]any{"error": message, "status": status})
		return
	}
	content, err := app.render("templates/studio-error.html", pageData{Status: status, Message: message})
	if err != nil {
		http.Error(w, message, status)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(content)
	}
}

func (app *Web) json(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
