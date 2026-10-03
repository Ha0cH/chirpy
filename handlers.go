package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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

func (cfg *apiConfig) handlerMetricsReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
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
