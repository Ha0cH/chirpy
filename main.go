package main

import (
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	serveMux := http.NewServeMux()
	apiCfg := &apiConfig{}

	// handle static files
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	// readiness endpoint
	serveMux.HandleFunc("GET /api/healthz", handlerHealthz)

	// chirps length validation endpoint
	serveMux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)

	// metrics endpoint
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)

	// metrcis endpoint reset
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerMetricsReset)

	server := &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
