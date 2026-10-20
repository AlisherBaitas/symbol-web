package handlers

import (
	"bytes"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"symbol-web/internal/ascii"
)

type pageData struct {
	Text   string
	Banner string
	Art    string
}

// [Fix] Шаблоны главной страницы: блок результата вынесен в templates/result.html
// (файл требуется структурой проекта), index.html подключает его через {{template "result" .}}.
var pageTemplates = []string{"templates/index.html", "templates/result.html"}

// [Fix] AI-маршруты: GET на них раньше отдавал главную страницу с кодом 200.
var apiRoutes = map[string]bool{
	"/api/suggest":          true,
	"/api/recommend-banner": true,
	"/api/variations":       true,
}

// GET
func HandleRoot(w http.ResponseWriter, r *http.Request) {

	// [Fix] Шаблон "GET /" совпадает с любым путём, поэтому /nope и т.п.
	// раньше возвращали главную страницу с кодом 200. Теперь — 404 (и 405 для API).
	if r.URL.Path != "/" {
		if apiRoutes[r.URL.Path] {
			w.Header().Set("Allow", http.MethodPost)
			writeVariationError(w, http.StatusMethodNotAllowed, "Use POST for this endpoint.")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeVariationError(w, http.StatusNotFound, "Endpoint not found.")
			return
		}
		renderError(w, http.StatusNotFound, "Page not found: "+r.URL.Path)
		return
	}

	renderPage(w, http.StatusOK, pageData{Banner: "standard.txt"})
}

// [Fix] GET /symbol-art (обновление страницы, прямая ссылка) -> на главную,
// вместо того чтобы молча отдавать главную страницу по чужому адресу.
func HandleSymbolArtPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// POST
func HandleSymbolArt(w http.ResponseWriter, r *http.Request) {

	// Value from FORM
	inputWord := r.FormValue("inputWord")
	bannerType := r.FormValue("banners")

	// [Fix] Баннер только из белого списка: неизвестное имя -> 400 по ТЗ.
	// Раньше значение из формы открывалось как файл (banners=go.mod -> паника).
	bannerFile, err := ascii.BannerFile(bannerType)
	if err != nil {
		renderError(w, http.StatusBadRequest, err.Error())
		return
	}

	art, errMessage := ascii.Generate(inputWord, bannerFile)

	if errMessage != nil {

		statusCode := http.StatusBadRequest // 400 by default

		switch {
		case errors.Is(errMessage, ascii.ErrBannerNotFound):
			statusCode = http.StatusNotFound // 404
		case errors.Is(errMessage, ascii.ErrBannerCorrupted):
			// [Fix] Испорченный файл баннера — ошибка сервера, а не пользователя.
			log.Printf("symbol-art: %v (%s)", errMessage, bannerFile)
			statusCode = http.StatusInternalServerError // 500
		}

		renderError(w, statusCode, errMessage.Error())
		return
	}

	data := pageData{
		// [Fix] В поле возвращаем текст с "\n", чтобы повторная отправка не копила "\r".
		Text:   ascii.NormalizeLineBreaks(inputWord),
		Banner: bannerFile,
		Art:    art,
	}

	renderPage(w, http.StatusOK, data)
}

// [Fix] renderPage — общий вывод главной страницы. По ТЗ: шаблон не найден -> 404,
// ошибка выполнения -> 500. Раньше в обоих случаях был 500 и сырой текст ошибки.
func renderPage(w http.ResponseWriter, status int, data pageData) {
	tmpl, err := template.ParseFiles(pageTemplates...)
	if err != nil {
		log.Printf("templates: %v", err)
		templateParseError(w, err)
		return
	}

	// Buffer before sending to the client
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "index.html", data); err != nil {
		log.Printf("templates: %v", err)
		http.Error(w, "Internal Server Error: could not render the page", http.StatusInternalServerError) // 500
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

// [Fix] renderError — общая страница ошибки (раньше код дублировался в HandleSymbolArt,
// а 404 для неизвестных путей не было вовсе).
func renderError(w http.ResponseWriter, status int, message string) {
	tmpl, err := template.ParseFiles("templates/error.html")
	if err != nil {
		log.Printf("templates: %v", err)
		// Без шаблона всё равно отдаём исходный код ошибки и понятный текст.
		http.Error(w, http.StatusText(status)+": "+message, status)
		return
	}

	data := struct {
		Code    int
		Title   string
		Message string
	}{
		Code:    status,
		Title:   http.StatusText(status),
		Message: message,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("templates: %v", err)
		http.Error(w, http.StatusText(status)+": "+message, status)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

// [Fix] Файл шаблона отсутствует -> 404, иначе (синтаксическая ошибка) -> 500.
func templateParseError(w http.ResponseWriter, err error) {
	if errors.Is(err, fs.ErrNotExist) {
		http.Error(w, "Not Found: template file is missing", http.StatusNotFound) // 404
		return
	}
	http.Error(w, "Internal Server Error: template is invalid", http.StatusInternalServerError) // 500
}
