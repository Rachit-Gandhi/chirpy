package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

func (cfg *apiConfig) resetMetricHandler(resp http.ResponseWriter, req *http.Request) {
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
