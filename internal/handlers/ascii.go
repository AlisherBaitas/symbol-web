package handlers

import (
	"net/http"
	"html/template"

	"symbol-web/internal/ascii"
)

// GET
func HandleRoot(w http.ResponseWriter, _ *http.Request) {

	// Parse index.html file
	tmpl, err := template.ParseFiles("../templates/index.html")

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
func HandleSymbolArt(w http.ResponseWriter, r *http.Request){
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

	art := ascii.Generate(inputWord, bannerType)

	// Parse result.html file
	tmpl, err := template.ParseFiles("../templates/result.html")

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