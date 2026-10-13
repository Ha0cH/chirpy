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
	platform       string
	jwtSecret      string
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}

	dbQueries := database.New(db)

	serveMux := http.NewServeMux()
	apiCfg := &apiConfig{
		queries:   dbQueries,
		platform:  platform,
		jwtSecret: jwtSecret,
	}

	// handle static files
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	// readiness endpoint
	serveMux.HandleFunc("GET /api/healthz", handlerHealthz)

	// create user endpoint
	serveMux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)

	// create chirps endpoint
	serveMux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)

	// get all chirps endpoint
	serveMux.HandleFunc("GET /api/chirps", apiCfg.handlerGetAllChirps)

	// get one chirp by ID endpoint
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpByID)

	// user login endpoint
	serveMux.HandleFunc("POST /api/login", apiCfg.handlerUserLogin)

	// user refresh endpoint
	serveMux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)

	// refresh token revoke endpoint
	serveMux.HandleFunc("POST /api/revoke", apiCfg.handlerRevokeRefreshToken)

	// user login info update endpoint
	serveMux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateLoginInfo)

	// delete chirp endpoint
	serveMux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDeleteChirpByID)

	// upgrade user webhook
	serveMux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerUpgradeUserByID)

	// metrics endpoint
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)

	// metrcis endpoint reset
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	server := &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
