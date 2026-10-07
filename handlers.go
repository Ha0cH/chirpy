package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

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

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type userEmail struct {
		Email string `json:"email"`
	}

	type userInfo struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	var email userEmail
	err := decoder.Decode(&email)
	if err != nil {
		msg := fmt.Sprintf("Error decoding the request body: %s", err)
		helperRespondWithError(w, http.StatusBadRequest, msg)
		return
	}

	usr, err := cfg.queries.CreateUser(r.Context(), email.Email)
	if err != nil {
		msg := fmt.Sprintf("Error creating user: %s", err)
		helperRespondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	usrResponse := userInfo{
		Id:        usr.ID,
		CreatedAt: usr.CreatedAt,
		UpdatedAt: usr.UpdatedAt,
		Email:     usr.Email,
	}

	helperRespondWithJSON(w, http.StatusCreated, usrResponse)
}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}

	type chirpValid struct {
		Cleaned_body string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	var ch chirp
	err := decoder.Decode(&ch)
	if err != nil {
		msg := fmt.Sprintf("Error decoding the request body: %s", err)
		helperRespondWithError(w, http.StatusBadRequest, msg)
		return
	}
	chirpLen := len(ch.Body)
	if chirpLen <= 140 {
		cleaned := helperReplaceProfane(ch.Body)
		v := chirpValid{
			Cleaned_body: cleaned,
		}
		helperRespondWithJSON(w, http.StatusOK, v)
	} else {
		msg := "Chirp is too long"
		helperRespondWithError(w, http.StatusBadRequest, msg)
	}
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
