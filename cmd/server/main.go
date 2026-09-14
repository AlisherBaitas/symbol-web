package main

import (
	"log"
	"net/http"
	"html/template"
)

func main() {

	// Router 
	mux := http.NewServeMux()

	// Endpoints 
	mux.HandleFunc("GET /", handleRoot)
	mux.HandleFunc("POST /symbol-art", handleSymbolArt)

	// Return handler and collection of files in the folder 
	fs := http.FileServer(http.Dir("static"))

	// Handle static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	log.Println("Server running on http://localhost:8080")

	// Run server on port 8080 and os.Exit(1) if any error 
	log.Fatal(http.ListenAndServe(":8080", mux))

}

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

func handleSymbolArt(w http.ResponseWriter, r *http.Request){}
