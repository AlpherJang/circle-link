package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/circle-link/circle-link/server/internal/config"
)

func main() {
	cfg := config.Load()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"service": "relay",
			"status":  "ok",
		})
	})
	mux.HandleFunc("/v1/ws", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "websocket relay bootstrap not implemented yet",
		})
	})

	server := &http.Server{
		Addr:    cfg.Relay.Addr,
		Handler: mux,
	}

	log.Printf("circle-link relay listening on %s", cfg.Relay.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("relay server failed: %v", err)
	}
}
