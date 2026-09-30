package main

import (
	"log"
	"net/http"
	"html/template"
	
	"symbol-web/internal/handlers"
)

func main() {

	// Router 
	mux := http.NewServeMux()

	// Endpoints 
	mux.HandleFunc("GET /", handleRoot)
	mux.HandleFunc("POST /symbol-art", handleSymbolArt)

	// AI endpoints
	mux.HandleFunc("POST /api/suggest", handleSuggestion)
	mux.HandleFunc("POST /api/recommend-banner", handleBannerRec)
	mux.HandleFunc("POST /api/variations", handleVariation)

	// Return handler and collection of files in the folder 
	fs := http.FileServer(http.Dir("static"))

	// Handle static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	log.Println("Server running on http://localhost:8080")

	// Run server on port 8080 and os.Exit(1) if any error 
	log.Fatal(http.ListenAndServe(":8080", mux))

}

// GET
func handleRoot(w http.ResponseWriter, _ *http.Request) {

	// Parse index.html file
	tmpl, err := template.ParseFiles("templates/index.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	
	// Run template
	err = tmpl.Execute(w, nil) // nil, because of no data

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// POST
func handleSymbolArt(w http.ResponseWriter, r *http.Request){
	// Check only using POST method
	/* 
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	*/

	// Value from FORM
	inputWord := r.FormValue("inputWord")
	bannerType := r.FormValue("banners")

	art := handlers.Generate(inputWord, bannerType)

	// Parse result.html file
	tmpl, err := template.ParseFiles("templates/result.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    data := struct {
		ArtMessage string
	}{
		ArtMessage: art,
	}

	// Run template
	err = tmpl.Execute(w, data)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}


func handleSuggestion(w http.ResponseWriter, r *http.Request) {}

func handleBannerRec(w http.ResponseWriter, r *http.Request) {}

func handleVariation(w http.ResponseWriter, r *http.Request) {}

