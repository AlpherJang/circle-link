package main

import (
	"log"
	"net/http"

	"github.com/circle-link/circle-link/server/internal/config"
	"github.com/circle-link/circle-link/server/internal/httpapi"
)

func main() {
	cfg := config.Load()
	server := &http.Server{
		Addr:    cfg.API.Addr,
		Handler: httpapi.NewRouter(),
	}

	log.Printf("circle-link api listening on %s", cfg.API.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("api server failed: %v", err)
	}
}
