package handlers

import "net/http"

// [Fix] Маршруты вынесены из main.go, чтобы handlers_test.go проверял
// настоящую маршрутизацию (404 для неизвестных путей, 405, статика).
func NewRouter() http.Handler {

	// Router
	mux := http.NewServeMux()

	// Endpoints
	mux.HandleFunc("GET /", HandleRoot)
	mux.HandleFunc("GET /symbol-art", HandleSymbolArtPage)
	mux.HandleFunc("POST /symbol-art", HandleSymbolArt)

	// AI endpoints
	mux.HandleFunc("POST /api/suggest", HandleSuggestion)
	mux.HandleFunc("POST /api/recommend-banner", HandleBannerRec)
	mux.HandleFunc("POST /api/variations", HandleVariation)

	// Return handler and collection of files in the folder
	fs := http.FileServer(http.Dir("static"))

	// Handle static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	return mux
}
