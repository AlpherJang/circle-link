package httpapi

import (
	"encoding/json"
	"net/http"
)

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"service": "api",
		"status":  "ok",
	})
}

func notImplemented(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", name+" endpoint scaffolded but not implemented")
	}
}
