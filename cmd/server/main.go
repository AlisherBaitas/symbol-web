package main

import (
	"log"
	"net/http"

	"symbol-web/internal/handlers"
)

func main() {

	// [Fix] Маршруты перенесены в handlers.NewRouter() (internal/handlers/routes.go),
	// чтобы handlers_test.go проверял настоящую маршрутизацию: 404, 405, статику.
	mux := handlers.NewRouter()

	log.Println("Server running on http://localhost:8080")

	// Run server on port 8080 and os.Exit(1) if any error
	log.Fatal(http.ListenAndServe(":8080", mux))
}
