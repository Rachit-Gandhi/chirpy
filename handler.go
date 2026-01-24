package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Rachit-Gandhi/chirpy/internal/auth"
	"github.com/Rachit-Gandhi/chirpy/internal/database"
	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "userID"

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) middlewareJWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwtSecret := cfg.jwtSecret
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, 401, "token not found")
			return
		}
		userID, err := auth.ValidateJWT(token, jwtSecret)
		if err != nil {
			respondWithError(w, 401, fmt.Sprintf("user not found, %v", err))
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
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

func (cfg *apiConfig) validateChirp(resp http.ResponseWriter, req *http.Request) (map[string]string, error) {
	jwtSecret := cfg.jwtSecret
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		return map[string]string{}, fmt.Errorf("token not found")

	}
	userID, err := auth.ValidateJWT(token, jwtSecret)
	if err != nil {
		return map[string]string{}, fmt.Errorf("user not found")
	}

	type reqParameter struct {
		Body string `json:"body"`
	}
	request := reqParameter{}
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	err = decoder.Decode(&request)
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
	dat := map[string]string{"cleaned_body": request.Body, "user_id": userID.String()}
	return dat, nil
}

func (cfg *apiConfig) createUserHandler(resp http.ResponseWriter, req *http.Request) {
	type requestCreateUser struct {
		Email    string `json:"email"`
		Password string `json:"password"`
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
	hashedPassword, err := auth.HashPassword(request.Password)
	if err != nil {
		respondWithError(resp, 400, "error hashing password")
	}
	reqCreateUser := database.CreateUserParams{
		Email:          request.Email,
		HashedPassword: hashedPassword,
	}
	dbUser, err := cfg.dbQueries.CreateUser(req.Context(), reqCreateUser)
	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
		Password:  dbUser.HashedPassword,
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
	cleanChirp, err := cfg.validateChirp(resp, req)
	if err != nil {
		respondWithError(resp, 400, fmt.Sprintf("chirp validation failed: %v", err))
		return
	}

	defer req.Body.Close()
	user_id, err := uuid.Parse(cleanChirp["user_id"])
	if err != nil {
		respondWithError(resp, 400, "error with userId")
		return
	}
	newChirp := database.CreateChirpParams{
		Body:   cleanChirp["cleaned_body"],
		UserID: user_id,
	}
	dbChirp, err := cfg.dbQueries.CreateChirp(req.Context(), newChirp)
	if err != nil {
		respondWithError(resp, 500, "issue with creating chirp")
		return
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
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(201)
	resp.Write(finalChirp)
}

func (cfg *apiConfig) getChirpsHandler(resp http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.dbQueries.GetChirps(req.Context())
	if err != nil {
		respondWithError(resp, 500, fmt.Sprintf("error getting chirps: %v", err))
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

type LoginParams struct {
	Password           string `json:"password"`
	Email              string `json:"email"`
	Expires_In_Seconds int    `json:"expires_in_seconds"`
}

func (cfg *apiConfig) loginUser(resp http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var loginUser LoginParams
	err := decoder.Decode(&loginUser)
	if err != nil {
		respondWithError(resp, 401, "incorrect email or password")
		return
	}
	defer req.Body.Close()
	expires := time.Duration(loginUser.Expires_In_Seconds) * time.Second
	if expires == 0 || expires > time.Hour {
		expires = time.Hour
	}

	user, err := cfg.dbQueries.GetUserByEmail(req.Context(), loginUser.Email)
	if err != nil {
		respondWithError(resp, 401, "incorrect email or password")
		return
	}
	isok, err := auth.CheckPasswordHash(loginUser.Password, user.HashedPassword)
	if !isok || err != nil {
		respondWithError(resp, 401, "incorrect email or password")
		return
	}
	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, expires)
	User := UserLogin{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     token,
	}
	dat, err := json.Marshal(User)
	if err != nil {
		respondWithError(resp, 401, "incorrect email or password")
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(200)
	resp.Write(dat)
}
