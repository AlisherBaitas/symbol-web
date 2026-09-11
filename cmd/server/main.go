package main

import (
	"log"
	"log/slog"
	"net/http"
)

func main() {

	// Router 
	mux := http.NewServeMux()

	// Endpoints 
	mux.HandleFunc("GET /", handleRoot)
	mux.HandleFunc("POST /symbol-art", handleSymbolArt)


	// Run server on port 8080
	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		// Returns os.Exit(1)
		log.Fatal(err)
	}
}

func handleRoot(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte("Welcome to the page !"))

	if err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

func handleSymbolArt(w http.ResponseWriter, r *http.Request){}

