package config

import "testing"

func TestLoadUsesDefaultsWhenEnvMissing(t *testing.T) {
	t.Setenv("CIRCLE_LINK_API_ADDR", "")
	t.Setenv("CIRCLE_LINK_RELAY_ADDR", "")

	cfg := Load()

	if cfg.API.Addr != ":8080" {
		t.Fatalf("expected default api addr :8080, got %q", cfg.API.Addr)
	}

	if cfg.Relay.Addr != ":8081" {
		t.Fatalf("expected default relay addr :8081, got %q", cfg.Relay.Addr)
	}
}
