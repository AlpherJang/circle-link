package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/circle-link/circle-link/server/internal/service/auth"
)

type signUpRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type verifyEmailRequest struct {
	Email             string `json:"email"`
	VerificationToken string `json:"verificationToken"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type logoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type debugVerificationTokenReader interface {
	DebugVerificationToken(ctx context.Context, email string) (string, bool)
}

func (s *Server) handleSignUp(w http.ResponseWriter, r *http.Request) {
	var req signUpRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON request body.")
		return
	}

	result, err := s.authService.SignUp(r.Context(), auth.SignUpInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailAlreadyExists):
			writeError(w, http.StatusConflict, "AUTH_EMAIL_EXISTS", "An account with this email already exists.")
		case errors.Is(err, auth.ErrWeakPassword):
			writeError(w, http.StatusBadRequest, "AUTH_WEAK_PASSWORD", "Password does not meet minimum requirements.")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to sign up.")
		}
		return
	}

	response := map[string]any{
		"userId":                    result.UserID,
		"emailVerificationRequired": result.EmailVerificationRequired,
	}
	if debugReader, ok := s.authService.(interface {
		DebugVerificationToken(ctx context.Context, email string) (string, bool)
	}); ok {
		if token, found := debugReader.DebugVerificationToken(r.Context(), req.Email); found {
			response["verificationToken"] = token
		}
	}

	writeData(w, http.StatusCreated, response)
}

func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON request body.")
		return
	}

	if err := s.authService.VerifyEmail(r.Context(), auth.VerifyEmailInput{
		Email:             req.Email,
		VerificationToken: req.VerificationToken,
	}); err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidVerification):
			writeError(w, http.StatusBadRequest, "AUTH_INVALID_VERIFICATION", "Verification token is invalid.")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify email.")
		}
		return
	}

	writeData(w, http.StatusOK, map[string]any{
		"verified": true,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON request body.")
		return
	}

	result, err := s.authService.Login(r.Context(), auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", "Email or password is incorrect.")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to log in.")
		}
		return
	}

	writeData(w, http.StatusOK, map[string]any{
		"userId":       result.UserID,
		"accessToken":  result.AccessToken,
		"refreshToken": result.RefreshToken,
		"expiresAt":    result.ExpiresAt.Format(time.RFC3339),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON request body.")
		return
	}

	result, err := s.authService.RefreshSession(r.Context(), auth.RefreshSessionInput{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidRefreshToken), errors.Is(err, auth.ErrSessionExpired):
			writeError(w, http.StatusUnauthorized, "AUTH_SESSION_EXPIRED", "Refresh token is invalid or expired.")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to refresh session.")
		}
		return
	}

	writeData(w, http.StatusOK, map[string]any{
		"accessToken":  result.AccessToken,
		"refreshToken": result.RefreshToken,
		"expiresAt":    result.ExpiresAt.Format(time.RFC3339),
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON request body.")
		return
	}

	if err := s.authService.Logout(r.Context(), auth.LogoutInput{
		RefreshToken: req.RefreshToken,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to log out.")
		return
	}

	writeData(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	var req changePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON request body.")
		return
	}

	if err := s.authService.ChangePassword(r.Context(), auth.ChangePasswordInput{
		UserID:          session.UserID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		switch {
		case errors.Is(err, auth.ErrCurrentPasswordMismatch):
			writeError(w, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", "Current password is incorrect.")
		case errors.Is(err, auth.ErrWeakPassword):
			writeError(w, http.StatusBadRequest, "AUTH_WEAK_PASSWORD", "Password does not meet minimum requirements.")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to change password.")
		}
		return
	}

	writeData(w, http.StatusOK, map[string]any{
		"success": true,
	})
}
