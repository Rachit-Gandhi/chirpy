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

func validateChirpHandler(resp http.ResponseWriter, req *http.Request) {
	type reqParameter struct {
		Body string `json:"body"`
	}
	request := reqParameter{}
	resp.Header().Set("Content-Type", "application/json")
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	err := decoder.Decode(&request)
	if err != nil {
		respondWithError(resp, 500, "Something went wrong")
		return
	}
	if len(request.Body) > 140 {
		respondWithError(resp, 400, "Chirp is too long")
		return
	}
	resp.WriteHeader(200)
	for _, badWord := range []string{"Kerfuffle", "Sharbert", "Fornax"} {
		request.Body = strings.ReplaceAll(request.Body, badWord, "****")
		request.Body = strings.ReplaceAll(request.Body, strings.ToLower(badWord), "****")
		request.Body = strings.ReplaceAll(request.Body, strings.ToUpper(badWord), "****")
	}
	dat, _ := json.Marshal((map[string]string{"cleaned_body": request.Body}))
	resp.Write(dat)
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
