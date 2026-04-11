package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/circle-link/circle-link/server/internal/platform/ids"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Data      any       `json:"data"`
	Error     *apiError `json:"error"`
	RequestID string    `json:"requestId"`
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeEnvelope(w, status, envelope{
		Data:      data,
		Error:     nil,
		RequestID: ids.New("req"),
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeEnvelope(w, status, envelope{
		Data: nil,
		Error: &apiError{
			Code:    code,
			Message: message,
		},
		RequestID: ids.New("req"),
	})
}

func writeEnvelope(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func decodeJSON(r *http.Request, out any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(out)
}

func bearerToken(r *http.Request) string {
	raw := r.Header.Get("Authorization")
	if !strings.HasPrefix(raw, "Bearer ") {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
}

func (s *Server) requireAccessSession(w http.ResponseWriter, r *http.Request) (authContext, bool) {
	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Missing bearer access token.")
		return authContext{}, false
	}

	session, err := s.authService.AuthenticateAccessToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Access token is invalid or expired.")
		return authContext{}, false
	}

	return authContext{
		UserID: session.UserID,
	}, true
}

type authContext struct {
	UserID string
}
