package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/Rachit-Gandhi/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	jwtSecret      string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
}

type UserLogin struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("error loading .env file: %v", err)
	}
	dbURL := os.Getenv("DB_URL")
	jwtSecret := os.Getenv("JWT_Secret")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}
	dbQueries := database.New(db)
	var apiCfg apiConfig
	apiCfg.dbQueries = dbQueries
	apiCfg.jwtSecret = jwtSecret
	platform := os.Getenv("PLATFORM")
	apiCfg.platform = platform
	customHandler := http.NewServeMux()
	customHandler.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir("./static")))))
	customHandler.HandleFunc("GET /api/healthz", readyHandler)
	customHandler.HandleFunc("GET /admin/metrics", apiCfg.metricHandler)
	customHandler.HandleFunc("POST /admin/reset", apiCfg.resetAdminHandler)
	customHandler.HandleFunc("GET /api/chirps", apiCfg.getChirpsHandler)
	customHandler.HandleFunc("POST /api/login", apiCfg.loginUser)
	customHandler.HandleFunc("POST /api/users", apiCfg.createUserHandler)
	customHandler.HandleFunc("POST /api/chirps", apiCfg.createChirpHandler)
	customHandler.HandleFunc("GET /api/chirps/{chirpId}", apiCfg.getChirpHandler)
	s := &http.Server{
		Addr:    ":8080",
		Handler: customHandler,
	}
	fmt.Println("Server Starting on 8080 port")
	log.Fatal(s.ListenAndServe())
}
