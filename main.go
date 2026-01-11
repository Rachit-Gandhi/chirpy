package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	var apiCfg apiConfig
	customHandler := http.NewServeMux()
	customHandler.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir("./static")))))
	customHandler.HandleFunc("GET /api/healthz", readyHandler)
	customHandler.HandleFunc("GET /admin/metrics", apiCfg.metricHandler)
	customHandler.HandleFunc("POST /admin/reset", apiCfg.resetMetricHandler)
	customHandler.HandleFunc("POST /api/validate_chirp", validateChirpHandler)
	s := &http.Server{
		Addr:    ":8080",
		Handler: customHandler,
	}
	fmt.Println("Server Starting on 8080 port")
	log.Fatal(s.ListenAndServe())
}
