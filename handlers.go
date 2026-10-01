package main

import (
	"fmt"
	"log"
	"net/http"
)

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Println("failed to write response body:", err)
	}
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := fmt.Fprintf(w, "Hits: %d", hits)
	if err != nil {
		log.Println("failed to write to response body: ", err)
	}
}
