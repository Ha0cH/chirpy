package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Ha0cH/chirpy/internal/auth"
	"github.com/Ha0cH/chirpy/internal/database"
	"github.com/google/uuid"
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	htmlTemplate := fmt.Sprintf(`
	<html>
  		<body>
    	<h1>Welcome, Chirpy Admin</h1>
     	<p>Chirpy has been visited %d times!</p>
     	</body>
	</html>
	`, hits)

	_, err := fmt.Fprint(w, htmlTemplate)
	if err != nil {
		log.Println("failed to write to response body: ", err)
	}
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		helperRespondWithError(w, http.StatusForbidden, "Forbidden")
		return
	}

	err := cfg.queries.DeleteAllUsers(r.Context())
	if err != nil {
		helperRespondWithError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf("Error deleting users: %s", err),
		)
		return
	}

	cfg.fileserverHits.Store(0)

	w.WriteHeader(http.StatusOK)
}

type userInfoReq struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

type userInfoResp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var u userInfoReq
	err := decoder.Decode(&u)
	if err != nil {
		msg := fmt.Sprintf("Error decoding the request body: %s", err)
		helperRespondWithError(w, http.StatusBadRequest, msg)
		return
	}

	hashedPsw, err := auth.HashPassword(u.Password)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		helperRespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	usrParams := database.CreateUserParams{
		Email:          u.Email,
		HashedPassword: hashedPsw,
	}

	usr, err := cfg.queries.CreateUser(r.Context(), usrParams)
	if err != nil {
		msg := fmt.Sprintf("Error creating user: %s", err)
		helperRespondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	usrResponse := userInfoResp{
		ID:        usr.ID,
		CreatedAt: usr.CreatedAt,
		UpdatedAt: usr.UpdatedAt,
		Email:     usr.Email,
	}

	helperRespondWithJSON(w, http.StatusCreated, usrResponse)
}

func (cfg *apiConfig) handlerUserLogin(w http.ResponseWriter, r *http.Request) {
	var u userInfoReq
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&u)
	if err != nil {
		msg := fmt.Sprintf("Error decoding the request body: %s", err)
		helperRespondWithError(w, http.StatusBadRequest, msg)
		return
	}

	usr, err := cfg.queries.GetUserByEmail(r.Context(), u.Email)
	if errors.Is(err, sql.ErrNoRows) {
		helperRespondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	pwdMatched, err := auth.CheckPasswordHash(u.Password, usr.HashedPassword)
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Error verifying password")
		return
	}

	if !pwdMatched {
		helperRespondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	response := userInfoResp{
		ID:        usr.ID,
		CreatedAt: usr.CreatedAt,
		UpdatedAt: usr.UpdatedAt,
		Email:     usr.Email,
	}
	helperRespondWithJSON(w, http.StatusOK, response)
}

type chirpReq struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

type chirpResp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var ch chirpReq
	err := decoder.Decode(&ch)
	if err != nil {
		msg := fmt.Sprintf("Error decoding the request body: %s", err)
		helperRespondWithError(w, http.StatusBadRequest, msg)
		return
	}

	chirpLen := utf8.RuneCountInString(ch.Body)
	if chirpLen > 140 {
		msg := "Chirp is too long"
		helperRespondWithError(w, http.StatusBadRequest, msg)
		return
	}

	cleaned := helperReplaceProfane(ch.Body)
	chirpParams := database.CreateChirpParams{
		Body:   cleaned,
		UserID: ch.UserID,
	}
	newChirp, err := cfg.queries.CreateChirp(r.Context(), chirpParams)
	if err != nil {
		msg := fmt.Sprintf("Error creating new chirp: %s", err)
		helperRespondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	response := chirpResp{
		ID:        newChirp.ID,
		CreatedAt: newChirp.CreatedAt,
		UpdatedAt: newChirp.UpdatedAt,
		Body:      newChirp.Body,
		UserID:    newChirp.UserID,
	}
	helperRespondWithJSON(w, http.StatusCreated, response)
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.queries.GetAllChirps(r.Context())
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error retrieving all chirps: %s", err))
		return
	}
	responses := make([]chirpResp, len(chirps))
	for idx, c := range chirps {
		responses[idx] = chirpResp{
			ID:        c.ID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Body:      c.Body,
			UserID:    c.UserID,
		}
	}
	helperRespondWithJSON(w, http.StatusOK, responses)
}

func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {
	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		helperRespondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}
	ch, err := cfg.queries.GetChirpByID(r.Context(), chirpID)
	if errors.Is(err, sql.ErrNoRows) {
		helperRespondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}
	response := chirpResp{
		ID:        ch.ID,
		CreatedAt: ch.CreatedAt,
		UpdatedAt: ch.UpdatedAt,
		Body:      ch.Body,
		UserID:    ch.UserID,
	}
	helperRespondWithJSON(w, http.StatusOK, response)
}

func helperRespondWithError(w http.ResponseWriter, code int, msg string) {
	type errorMsg struct {
		Error string `json:"error"`
	}

	errMsg := errorMsg{
		Error: msg,
	}

	helperRespondWithJSON(w, code, errMsg)
}

func helperRespondWithJSON(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		log.Printf("Error writing response: %s", err)
	}
}

func helperReplaceProfane(msg string) string {
	profane := map[string]struct{}{
		"kerfuffle": struct{}{},
		"sharbert":  struct{}{},
		"fornax":    struct{}{},
	}
	words := strings.Split(msg, " ")
	for i, w := range words {
		word := strings.ToLower(w)
		if _, ok := profane[word]; ok {
			words[i] = "****"
		}
	}
	cleaned := strings.Join(words, " ")
	return cleaned
}
