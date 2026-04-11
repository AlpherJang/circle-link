package auth

import (
	"context"
	"time"
)

type SignUpInput struct {
	Email       string
	Password    string
	DisplayName string
}

type SignUpResult struct {
	UserID                    string
	EmailVerificationRequired bool
}

type VerifyEmailInput struct {
	Email             string
	VerificationToken string
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginResult struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type RefreshSessionInput struct {
	RefreshToken string
}

type RefreshSessionResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type LogoutInput struct {
	RefreshToken string
}

type ChangePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

type AccessSession struct {
	UserID    string
	ExpiresAt time.Time
}

type Service interface {
	SignUp(ctx context.Context, input SignUpInput) (SignUpResult, error)
	VerifyEmail(ctx context.Context, input VerifyEmailInput) error
	Login(ctx context.Context, input LoginInput) (LoginResult, error)
	RefreshSession(ctx context.Context, input RefreshSessionInput) (RefreshSessionResult, error)
	Logout(ctx context.Context, input LogoutInput) error
	ChangePassword(ctx context.Context, input ChangePasswordInput) error
	AuthenticateAccessToken(ctx context.Context, accessToken string) (AccessSession, error)
}
