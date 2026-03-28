package main

import (
	"home24se-take-home/internal/api"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	handler := api.NewHandler()

	// API
	mux.HandleFunc("/api/analyze", handler.Analyze)

	// Static files (frontend)
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
