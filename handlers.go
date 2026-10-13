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
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
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
		ID:          usr.ID,
		CreatedAt:   usr.CreatedAt,
		UpdatedAt:   usr.UpdatedAt,
		Email:       usr.Email,
		IsChirpyRed: usr.IsChirpyRed,
	}

	helperRespondWithJSON(w, http.StatusCreated, usrResponse)
}

type userLoginReq struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

type userLoginResp struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

func (cfg *apiConfig) handlerUserLogin(w http.ResponseWriter, r *http.Request) {
	var u userLoginReq
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

	expiresIn := time.Hour
	token, err := auth.MakeJWT(usr.ID, cfg.jwtSecret, expiresIn)
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Error generating JWT")
		return
	}

	refreshTokenParams := database.CreateRefreshTokenParams{
		Token:  auth.MakeRefreshToken(),
		UserID: usr.ID,
	}

	refreshToken, err := cfg.queries.CreateRefreshToken(r.Context(), refreshTokenParams)
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Error creating refresh token")
		return
	}

	response := userLoginResp{
		ID:           usr.ID,
		CreatedAt:    usr.CreatedAt,
		UpdatedAt:    usr.UpdatedAt,
		Email:        usr.Email,
		Token:        token,
		RefreshToken: refreshToken.Token,
		IsChirpyRed:  usr.IsChirpyRed,
	}
	helperRespondWithJSON(w, http.StatusOK, response)
}

type refreshResp struct {
	Token string `json:"token"`
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		helperRespondWithError(w, http.StatusUnauthorized, "Missing or invalid refresh token")
		return
	}

	usrID, err := cfg.queries.GetUserFromRefreshToken(r.Context(), token)
	if errors.Is(err, sql.ErrNoRows) {
		helperRespondWithError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	newAccessToken, err := auth.MakeJWT(usrID, cfg.jwtSecret, time.Hour)
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Error generating JWT")
		return
	}

	response := refreshResp{
		Token: newAccessToken,
	}
	helperRespondWithJSON(w, http.StatusOK, response)
}

func (cfg *apiConfig) handlerRevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		helperRespondWithError(w, http.StatusUnauthorized, "Missing or invalid refresh token")
		return
	}
	err = cfg.queries.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Error revoking refresh token")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateLoginInfoReq struct {
	NewEmail    string `json:"email"`
	NewPassword string `json:"password"`
}

func (cfg *apiConfig) handlerUpdateLoginInfo(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		helperRespondWithError(w, http.StatusUnauthorized, "Missing or invalid access token")
		return
	}

	usrID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		helperRespondWithError(w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	decoder := json.NewDecoder(r.Body)
	var newLoginInfo updateLoginInfoReq
	err = decoder.Decode(&newLoginInfo)
	if err != nil {
		helperRespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if newLoginInfo.NewEmail == "" || newLoginInfo.NewPassword == "" {
		helperRespondWithError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	newHashedPwd, err := auth.HashPassword(newLoginInfo.NewPassword)
	if err != nil {
		log.Println(err)
		helperRespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	params := database.UpdateUserLoginInfoParams{
		Email:          newLoginInfo.NewEmail,
		HashedPassword: newHashedPwd,
		ID:             usrID,
	}

	updatedUsr, err := cfg.queries.UpdateUserLoginInfo(r.Context(), params)
	if err != nil {
		log.Println(err)
		helperRespondWithError(w, http.StatusInternalServerError, "Error updating database")
		return
	}

	response := userInfoResp{
		Email:       updatedUsr.Email,
		ID:          updatedUsr.ID,
		CreatedAt:   updatedUsr.CreatedAt,
		UpdatedAt:   updatedUsr.UpdatedAt,
		IsChirpyRed: updatedUsr.IsChirpyRed,
	}

	helperRespondWithJSON(w, http.StatusOK, response)
}

type polkaWebhook struct {
	Event string `json:"event"`
	Data  struct {
		UserID uuid.UUID `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) handlerUpgradeUserByID(w http.ResponseWriter, r *http.Request) {
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if apiKey != cfg.polkaKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var webhook polkaWebhook
	err = decoder.Decode(&webhook)
	if err != nil {
		helperRespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error decoding the request body: %s", err))
		return
	}

	if webhook.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = cfg.queries.UpgradeUserByID(r.Context(), webhook.Data.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		helperRespondWithError(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type chirpReq struct {
	Body string `json:"body"`
}

type chirpResp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		helperRespondWithError(w, http.StatusUnauthorized, "Missing or invalid bearer token")
		return
	}

	usrID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		helperRespondWithError(w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	decoder := json.NewDecoder(r.Body)
	var ch chirpReq
	err = decoder.Decode(&ch)
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
		UserID: usrID,
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

func (cfg *apiConfig) handlerDeleteChirpByID(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		helperRespondWithError(w, http.StatusUnauthorized, "Invalid or missing access token")
		return
	}

	usrID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		helperRespondWithError(w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		helperRespondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	chirp, err := cfg.queries.GetChirpByID(r.Context(), chirpID)
	if errors.Is(err, sql.ErrNoRows) {
		helperRespondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if usrID != chirp.UserID {
		helperRespondWithError(w, http.StatusForbidden, "Forbidden request")
		return
	}

	err = cfg.queries.DeleteChirpByID(r.Context(), chirpID)
	if err != nil {
		helperRespondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
