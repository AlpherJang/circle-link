package auth

import (
	"context"
	"testing"
)

func TestMemoryServiceSignUpLoginAndRefresh(t *testing.T) {
	ctx := context.Background()
	service := NewMemoryService()

	signUpResult, err := service.SignUp(ctx, SignUpInput{
		Email:       "alice@example.com",
		Password:    "strong-pass",
		DisplayName: "Alice",
	})
	if err != nil {
		t.Fatalf("sign up failed: %v", err)
	}
	if signUpResult.UserID == "" {
		t.Fatal("expected user id")
	}

	token, ok := service.DebugVerificationToken(ctx, "alice@example.com")
	if !ok || token == "" {
		t.Fatal("expected verification token")
	}

	if err := service.VerifyEmail(ctx, VerifyEmailInput{
		Email:             "alice@example.com",
		VerificationToken: token,
	}); err != nil {
		t.Fatalf("verify email failed: %v", err)
	}

	loginResult, err := service.Login(ctx, LoginInput{
		Email:    "alice@example.com",
		Password: "strong-pass",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loginResult.AccessToken == "" || loginResult.RefreshToken == "" {
		t.Fatal("expected tokens")
	}

	session, err := service.AuthenticateAccessToken(ctx, loginResult.AccessToken)
	if err != nil {
		t.Fatalf("authenticate access token failed: %v", err)
	}
	if session.UserID != signUpResult.UserID {
		t.Fatalf("expected user id %q, got %q", signUpResult.UserID, session.UserID)
	}

	refreshResult, err := service.RefreshSession(ctx, RefreshSessionInput{
		RefreshToken: loginResult.RefreshToken,
	})
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshResult.AccessToken == "" || refreshResult.RefreshToken == "" {
		t.Fatal("expected refreshed tokens")
	}
}

func TestMemoryServiceChangePassword(t *testing.T) {
	ctx := context.Background()
	service := NewMemoryService()

	result, err := service.SignUp(ctx, SignUpInput{
		Email:       "alice@example.com",
		Password:    "strong-pass",
		DisplayName: "Alice",
	})
	if err != nil {
		t.Fatalf("sign up failed: %v", err)
	}

	if err := service.ChangePassword(ctx, ChangePasswordInput{
		UserID:          result.UserID,
		CurrentPassword: "strong-pass",
		NewPassword:     "even-stronger-pass",
	}); err != nil {
		t.Fatalf("change password failed: %v", err)
	}

	if _, err := service.Login(ctx, LoginInput{
		Email:    "alice@example.com",
		Password: "even-stronger-pass",
	}); err != nil {
		t.Fatalf("login with changed password failed: %v", err)
	}
}
