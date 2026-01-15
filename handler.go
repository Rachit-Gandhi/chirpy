package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Rachit-Gandhi/chirpy/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricHandler(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "text/html")
	resp.WriteHeader(200)
	responseText := fmt.Sprintf(`
	<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	resp.Write([]byte(responseText))
}

func (cfg *apiConfig) resetAdminHandler(resp http.ResponseWriter, req *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(resp, 403, fmt.Sprintf("only accessible in local dev environment currently it is: %v", cfg.platform))
		return
	}
	err := cfg.dbQueries.DeleteUser(req.Context())
	if err != nil {
		respondWithError(resp, 500, fmt.Sprintf("error deleting all users: %v", err))
		return
	}
	resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	resp.WriteHeader(200)
	cfg.fileserverHits.Store(0)
	resp.Write([]byte("Action Completed"))
}

func readyHandler(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.WriteHeader(200)
	resp.Write([]byte("OK"))
}

func respondWithError(resp http.ResponseWriter, errorCode int, errorMessage string) {
	resp.WriteHeader(errorCode)
	dat, _ := json.Marshal((map[string]string{"error": errorMessage}))
	resp.Write(dat)
}

func validateChirpHandler(req *http.Request) (map[string]string, error) {
	type reqParameter struct {
		Body   string `json:"body"`
		UserId string `json:"user_id"`
	}
	request := reqParameter{}
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	err := decoder.Decode(&request)
	if err != nil {
		return map[string]string{}, err
	}
	if len(request.Body) > 140 {
		return map[string]string{}, fmt.Errorf("chirp too long, has to be less than 140 characters")
	}
	for _, badWord := range []string{"Kerfuffle", "Sharbert", "Fornax"} {
		request.Body = strings.ReplaceAll(request.Body, badWord, "****")
		request.Body = strings.ReplaceAll(request.Body, strings.ToLower(badWord), "****")
		request.Body = strings.ReplaceAll(request.Body, strings.ToUpper(badWord), "****")
	}
	dat := map[string]string{"cleaned_body": request.Body, "user_id": request.UserId}
	return dat, nil
}

func (cfg *apiConfig) createUserHandler(resp http.ResponseWriter, req *http.Request) {
	type requestCreateUser struct {
		Email string `json:"email"`
	}
	var request requestCreateUser
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	err := decoder.Decode(&request)
	if err != nil {
		respondWithError(resp, 400, "error decoding the request: invalid request")
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	reqCreateUser := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email:     request.Email,
	}
	dbUser, err := cfg.dbQueries.CreateUser(req.Context(), reqCreateUser)
	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
	dat, err := json.Marshal(user)
	if err != nil {
		respondWithError(resp, 500, "could not marshal user")
		return
	}
	resp.WriteHeader(201)

	resp.Write(dat)
}

func (cfg *apiConfig) createChirpHandler(resp http.ResponseWriter, req *http.Request) {
	cleanChirp, err := validateChirpHandler(req)
	if err != nil {
		respondWithError(resp, 400, fmt.Sprintf("chirp validation failed: %v", err))
	}

	defer req.Body.Close()
	user_id, err := uuid.Parse(cleanChirp["user_id"])
	if err != nil {
		respondWithError(resp, 400, "error with userId")
	}
	newChirp := database.CreateChirpParams{
		Body:   cleanChirp["cleaned_body"],
		UserID: user_id,
	}
	dbChirp, err := cfg.dbQueries.CreateChirp(req.Context(), newChirp)
	if err != nil {
		respondWithError(resp, 500, "issue with creating chirp")
	}
	chirp := Chirp{
		ID:        dbChirp.ID,
		UserID:    dbChirp.UserID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
	}
	finalChirp, err := json.Marshal(chirp)
	if err != nil {
		respondWithError(resp, 500, "issue with marshalling")
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(201)
	resp.Write(finalChirp)
}

func (cfg *apiConfig) getChirpsHandler(resp http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.dbQueries.GetChirps(req.Context())
	if err != nil {
		respondWithError(resp, 500, fmt.Sprintf("error getting chirps: %w", err))
	}
	chirpsResp := []Chirp{}
	for _, chirp := range chirps {
		chirpResp := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		chirpsResp = append(chirpsResp, chirpResp)
	}
	dat, err := json.Marshal(chirpsResp)
	if err != nil {
		respondWithError(resp, 500, fmt.Sprintf("error marshalling the chirps: %v", err))
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(200)
	resp.Write(dat)
}

func (cfg *apiConfig) getChirpHandler(resp http.ResponseWriter, req *http.Request) {
	uuidChirp, err := uuid.Parse(req.PathValue("chirpId"))
	if err != nil {
		respondWithError(resp, 404, "invalid uuid")
	}
	chirp, err := cfg.dbQueries.GetChirp(req.Context(), uuidChirp)
	if err != nil {
		respondWithError(resp, 404, "error not found chirp with uuid")
	}
	finChirp := Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}
	dat, err := json.Marshal(finChirp)
	if err != nil {
		respondWithError(resp, 404, "error marshalling json")
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(200)
	resp.Write(dat)

}
