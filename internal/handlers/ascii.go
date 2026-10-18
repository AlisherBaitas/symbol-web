package handlers

import (
	"net/http"
	"html/template"
	"errors"
	"bytes"

	"symbol-web/internal/ascii"
)

type pageData struct {
	Text   string
	Banner string
	Art    string
}

// GET
func HandleRoot(w http.ResponseWriter, _ *http.Request) {

	// Parse index.html file
	tmpl, err := template.ParseFiles("templates/index.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	var buf bytes.Buffer

	// Buffer before sending to the client
	err = tmpl.Execute(&buf, pageData{Banner: "standard.txt"})

	if err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
	}

	// Send after checking send to the client
	w.Write(buf.Bytes())
}

// POST
func HandleSymbolArt(w http.ResponseWriter, r *http.Request){

	// Value from FORM
	inputWord := r.FormValue("inputWord")
	bannerType := r.FormValue("banners")

	art, errMessage := ascii.Generate(inputWord, bannerType)

	if errMessage  != nil {

    	statusCode := http.StatusBadRequest // 400 by default

   		if errors.Is(errMessage, ascii.ErrBannerNotFound) {
        	statusCode = http.StatusNotFound // 404
    	}

		// Parse error.html file
    	tmpl, err := template.ParseFiles("templates/error.html")

   		if err != nil {
        	http.Error(w, err.Error(), http.StatusInternalServerError) // 500
        	return
    	}

		// Set status code
    	w.WriteHeader(statusCode)

		data := struct {
			ErrorMessage error
			Code         int    
			Title        string
			Message      string
		}{
			ErrorMessage: errMessage,
			Code:         statusCode,                 
			Title:        http.StatusText(statusCode),
			Message:      errMessage.Error(),         
		}

    	// Run template
    	err = tmpl.Execute(w, data)
    	if err != nil {
        	return
    	}
		
		return
	}

	// Parse result.html file
	tmpl, err := template.ParseFiles("templates/index.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // 500
		return
	}

	data := pageData{
		Text:   inputWord,
		Banner: bannerType,
		Art:    art,
	}

	// Run template
	err = tmpl.Execute(w, data) 

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // 500
		return
	}
}

