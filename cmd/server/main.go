package main

import (
	"log"
	"net/http"
	
	"symbol-web/internal/handlers"
)

func main() {

	// Router 
	mux := http.NewServeMux()

	// Endpoints 
	mux.HandleFunc("GET /", handlers.HandleRoot)
	mux.HandleFunc("POST /symbol-art", handlers.HandleSymbolArt)

	// AI endpoints
	mux.HandleFunc("POST /api/suggest", handlers.HandleSuggestion)
	mux.HandleFunc("POST /api/recommend-banner", handlers.HandleBannerRec)
	mux.HandleFunc("POST /api/variations", handlers.HandleVariation)

	// Return handler and collection of files in the folder 
	fs := http.FileServer(http.Dir("static"))

	// Handle static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	log.Println("Server running on http://localhost:8080")

	// Run server on port 8080 and os.Exit(1) if any error 
	log.Fatal(http.ListenAndServe(":8080", mux))
}

