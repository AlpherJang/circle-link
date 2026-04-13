package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/circle-link/circle-link/server/internal/domain"
	"github.com/circle-link/circle-link/server/internal/platform/ids"
	"github.com/circle-link/circle-link/server/internal/platform/security"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

var (
	ErrEmailAlreadyExists      = errors.New("email already exists")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrInvalidVerification     = errors.New("invalid verification token")
	ErrInvalidRefreshToken     = errors.New("invalid refresh token")
	ErrSessionExpired          = errors.New("session expired")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrWeakPassword            = errors.New("password does not meet minimum requirements")
	ErrUserNotFound            = errors.New("user not found")
	ErrCurrentPasswordMismatch = errors.New("current password is incorrect")
)

type MemoryService struct {
	mu sync.RWMutex

	usersByID      map[string]domain.User
	userIDByEmail  map[string]string
	verifyByEmail  map[string]string
	accessSessions map[string]AccessSession
	refreshTokens  map[string]refreshRecord
}

type refreshRecord struct {
	Session domain.AuthSession
}

func NewMemoryService() *MemoryService {
	return &MemoryService{
		usersByID:      make(map[string]domain.User),
		userIDByEmail:  make(map[string]string),
		verifyByEmail:  make(map[string]string),
		accessSessions: make(map[string]AccessSession),
		refreshTokens:  make(map[string]refreshRecord),
	}
}

func (s *MemoryService) SignUp(_ context.Context, input SignUpInput) (SignUpResult, error) {
	email := normalizeEmail(input.Email)
	if email == "" || !strings.Contains(email, "@") {
		return SignUpResult{}, ErrInvalidCredentials
	}
	if len(input.Password) < 8 {
		return SignUpResult{}, ErrWeakPassword
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.userIDByEmail[email]; exists {
		return SignUpResult{}, ErrEmailAlreadyExists
	}

	now := time.Now().UTC()
	userID := ids.New("usr")
	passwordHash, err := security.HashPassword(input.Password)
	if err != nil {
		return SignUpResult{}, err
	}

	s.usersByID[userID] = domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: passwordHash,
		DisplayName:  strings.TrimSpace(input.DisplayName),
		Status:       domain.UserStatusPendingVerification,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.userIDByEmail[email] = userID
	s.verifyByEmail[email] = ids.Token("verify")

	return SignUpResult{
		UserID:                    userID,
		EmailVerificationRequired: true,
	}, nil
}

func (s *MemoryService) VerifyEmail(_ context.Context, input VerifyEmailInput) error {
	email := normalizeEmail(input.Email)

	s.mu.Lock()
	defer s.mu.Unlock()

	expected, ok := s.verifyByEmail[email]
	if !ok || expected != strings.TrimSpace(input.VerificationToken) {
		return ErrInvalidVerification
	}

	userID, ok := s.userIDByEmail[email]
	if !ok {
		return ErrUserNotFound
	}

	user := s.usersByID[userID]
	now := time.Now().UTC()
	user.Status = domain.UserStatusActive
	user.EmailVerifiedAt = &now
	user.UpdatedAt = now
	s.usersByID[userID] = user
	delete(s.verifyByEmail, email)

	return nil
}

func (s *MemoryService) Login(_ context.Context, input LoginInput) (LoginResult, error) {
	email := normalizeEmail(input.Email)

	s.mu.Lock()
	defer s.mu.Unlock()

	userID, ok := s.userIDByEmail[email]
	if !ok {
		return LoginResult{}, ErrInvalidCredentials
	}

	user := s.usersByID[userID]
	match, err := security.VerifyPassword(input.Password, user.PasswordHash)
	if err != nil {
		return LoginResult{}, err
	}
	if !match {
		return LoginResult{}, ErrInvalidCredentials
	}

	return s.issueSessionLocked(userID), nil
}

func (s *MemoryService) RefreshSession(_ context.Context, input RefreshSessionInput) (RefreshSessionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.refreshTokens[security.HashToken(strings.TrimSpace(input.RefreshToken))]
	if !ok {
		return RefreshSessionResult{}, ErrInvalidRefreshToken
	}

	if time.Now().UTC().After(record.Session.ExpiresAt) || record.Session.RevokedAt != nil {
		delete(s.refreshTokens, security.HashToken(strings.TrimSpace(input.RefreshToken)))
		return RefreshSessionResult{}, ErrSessionExpired
	}

	delete(s.refreshTokens, security.HashToken(strings.TrimSpace(input.RefreshToken)))
	result := s.issueSessionLocked(record.Session.UserID)
	return RefreshSessionResult{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt,
	}, nil
}

func (s *MemoryService) Logout(_ context.Context, input LogoutInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.refreshTokens, security.HashToken(strings.TrimSpace(input.RefreshToken)))
	return nil
}

func (s *MemoryService) ChangePassword(_ context.Context, input ChangePasswordInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.usersByID[input.UserID]
	if !ok {
		return ErrUserNotFound
	}

	match, err := security.VerifyPassword(input.CurrentPassword, user.PasswordHash)
	if err != nil {
		return err
	}
	if !match {
		return ErrCurrentPasswordMismatch
	}
	if len(input.NewPassword) < 8 {
		return ErrWeakPassword
	}

	nextHash, err := security.HashPassword(input.NewPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = nextHash
	user.UpdatedAt = time.Now().UTC()
	s.usersByID[input.UserID] = user

	return nil
}

func (s *MemoryService) AuthenticateAccessToken(_ context.Context, accessToken string) (AccessSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.accessSessions[strings.TrimSpace(accessToken)]
	if !ok || time.Now().UTC().After(session.ExpiresAt) {
		return AccessSession{}, ErrUnauthorized
	}

	return session, nil
}

func (s *MemoryService) DebugVerificationToken(_ context.Context, email string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.verifyByEmail[normalizeEmail(email)]
	return token, ok
}

func (s *MemoryService) GetUser(_ context.Context, userID string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.usersByID[userID]
	if !ok {
		return domain.User{}, ErrUserNotFound
	}

	return user, nil
}

func (s *MemoryService) FindUserByEmail(_ context.Context, email string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, ok := s.userIDByEmail[normalizeEmail(email)]
	if !ok {
		return domain.User{}, ErrUserNotFound
	}

	return s.usersByID[userID], nil
}

func (s *MemoryService) issueSessionLocked(userID string) LoginResult {
	now := time.Now().UTC()
	accessToken := ids.Token("acc")
	refreshToken := ids.Token("ref")

	accessSession := AccessSession{
		UserID:    userID,
		ExpiresAt: now.Add(accessTokenTTL),
	}
	s.accessSessions[accessToken] = accessSession

	s.refreshTokens[security.HashToken(refreshToken)] = refreshRecord{
		Session: domain.AuthSession{
			ID:               ids.New("ses"),
			UserID:           userID,
			RefreshTokenHash: security.HashToken(refreshToken),
			ExpiresAt:        now.Add(refreshTokenTTL),
			CreatedAt:        now,
		},
	}

	return LoginResult{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessSession.ExpiresAt,
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
