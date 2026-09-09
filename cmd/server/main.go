package main

import (
	"fmt"
	"html/template"
	"net/http"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the home page!")
}

func main() {
	server := ":8080"
	fmt.Printf("The server started working on: %s\n", server)
	http.HandleFunc("/", homeHandler)
	http.ListenAndServe(":8080", nil)
	tmpl, err := template.ParseFiles("template/index.html")
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return
	}

}
