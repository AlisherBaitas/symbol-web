package main

import (
	"fmt"
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
}
