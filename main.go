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
	serveMux.HandleFunc("/healthz", handlerHealthz)

	// metrics endpoint
	serveMux.HandleFunc("/metrics", apiCfg.handlerMetrics)

	// metrcis endpoint reset
	serveMux.HandleFunc("/reset", apiCfg.handlerMetricsReset)

	server := &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
