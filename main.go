package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/Ha0cH/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries        *database.Queries
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}

	dbQueries := database.New(db)

	serveMux := http.NewServeMux()
	apiCfg := &apiConfig{
		queries: dbQueries,
	}

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

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
