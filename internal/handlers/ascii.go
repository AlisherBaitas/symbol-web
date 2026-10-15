package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"symbol-web/internal/ascii"
)

const (
	// [Validation] Лимит текста из README: 1000 символов.
	maxArtTextChars = 1000
	// [Validation] Лимит тела формы. 1000 символов даже в %XX-кодировке
	// занимают меньше 16 КиБ; всё, что больше, отклоняется сразу.
	maxArtFormBytes = 16 * 1024
)

// [Validation] Разрешённые баннеры. Значение из формы больше не передаётся
// в os.Open напрямую: раньше через него можно было прочитать любой файл
// (например, banners=/etc/passwd) или уронить сервер несуществующим именем.
var allowedBanners = map[string]bool{
	"standard.txt":   true,
	"shadow.txt":     true,
	"thinkertoy.txt": true,
}

type artPageData struct {
	Text       string
	Banner     string
	ArtMessage string
}

// GET
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	// [Validation] Маршрут "GET /" ловит все пути; неизвестные отдаём как 404,
	// а не главную страницу с кодом 200.
	if r.URL.Path != "/" {
		renderError(w, http.StatusNotFound, "Page not found.")
		return
	}

	// [Validation] Шаблон выполняется в буфер: ошибка шаблона -> 500 без утечки err.Error().
	renderPage(w, "templates/index.html", artPageData{Banner: "standard.txt"})
}

// POST
func HandleSymbolArt(w http.ResponseWriter, r *http.Request) {
	// Check only using POST method
	/*
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	*/

	// [Validation] Ограничиваем размер формы до разбора.
	r.Body = http.MaxBytesReader(w, r.Body, maxArtFormBytes)
	if err := r.ParseForm(); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			renderError(w, http.StatusRequestEntityTooLarge, "The text is too long. Use at most 1000 characters.")
		} else {
			renderError(w, http.StatusBadRequest, "The form could not be read. Please try again.")
		}
		return
	}

	// Value from FORM
	// [Validation] PostForm вместо FormValue: значения берём только из тела POST,
	// а не из строки запроса (?banners=...).
	inputWord := r.PostForm.Get("inputWord")
	bannerType := r.PostForm.Get("banners")

	// [Validation] Проверяем ввод до генерации: ошибки пользователя -> 400.
	inputWord, message := validateArtText(inputWord)
	if message != "" {
		renderError(w, http.StatusBadRequest, message)
		return
	}
	if bannerType == "" {
		renderError(w, http.StatusBadRequest, "Choose a banner: Standard, Shadow or Thinkertoy.")
		return
	}
	if !allowedBanners[bannerType] {
		renderError(w, http.StatusBadRequest, "Unknown banner. Choose Standard, Shadow or Thinkertoy.")
		return
	}

	art, err := ascii.Generate(inputWord, bannerType)
	if err != nil {
		// [Validation] Подробности - в журнал сервера, пользователю - понятный текст без путей.
		log.Printf("symbol-art: %v", err)
		if errors.Is(err, ascii.ErrBannerNotFound) {
			renderError(w, http.StatusNotFound, "The banner file was not found on the server.")
		} else {
			// Текст уже проверен, поэтому любая ошибка генератора - проблема сервера.
			renderError(w, http.StatusInternalServerError, "Could not draw the text. Please try again later.")
		}
		return
	}

	data := artPageData{
		Text:       inputWord,
		Banner:     bannerType,
		ArtMessage: art,
	}

	// [Validation] Раньше ошибка шаблона отдавала пользователю err.Error() с путями к файлам.
	renderPage(w, "templates/index.html", data)
}

// [Validation] validateArtText возвращает нормализованный текст или сообщение об ошибке.
// Шрифты содержат только печатные ASCII-символы (32-126); кириллица, emoji,
// Tab и другие управляющие символы раньше роняли запрос паникой.
func validateArtText(text string) (string, string) {
	// CRLF из многострочного ввода приводим к \n.
	text = strings.ReplaceAll(text, "\r\n", "\n")

	if text == "" {
		return "", "Enter some text to draw."
	}
	if len([]rune(text)) > maxArtTextChars {
		return "", "The text is too long. Use at most 1000 characters."
	}
	for _, ch := range text {
		if ch == '\n' || (ch >= ' ' && ch <= '~') {
			continue
		}
		if ch < ' ' || ch == 0x7F {
			return "", "Unsupported character: Tab and other control characters are not allowed. " +
				"Use English letters, digits, spaces and symbols."
		}
		return "", fmt.Sprintf("Unsupported character %q. Use English letters, digits, spaces and symbols.", ch)
	}
	return text, ""
}

// [Validation] renderPage сначала выполняет шаблон в буфер: при ошибке шаблона
// пользователь получит 500, а не обрезанную страницу с кодом 200.
func renderPage(w http.ResponseWriter, file string, data any) {
	tmpl, err := template.ParseFiles(file)
	if err != nil {
		log.Printf("template %s: %v", file, err)
		renderError(w, http.StatusInternalServerError, "The page could not be loaded. Please try again later.")
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("template %s: %v", file, err)
		renderError(w, http.StatusInternalServerError, "The page could not be loaded. Please try again later.")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("template %s: write response: %v", file, err)
	}
}

// [Validation] renderError показывает страницу error.html с кодом и понятным
// сообщением. Если и этот шаблон сломан, отвечает простым текстом.
func renderError(w http.ResponseWriter, status int, message string) {
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
	tmpl, err := template.ParseFiles("templates/error.html")
	if err == nil {
		err = tmpl.Execute(&buf, data)
	}
	if err != nil {
		log.Printf("template error.html: %v", err)
		http.Error(w, fmt.Sprintf("%d %s: %s", status, data.Title, message), status)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("template error.html: write response: %v", err)
	}
}
