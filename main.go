package main

import (
	"log"
	"net/http"
)

func main() {
	serveMux := http.NewServeMux()

	// handle static files
	serveMux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))

	// readiness endpoint
	serveMux.HandleFunc("/healthz", handlerHealthz)

	server := &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
