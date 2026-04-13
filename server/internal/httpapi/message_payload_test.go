package httpapi

import "testing"

func TestParseMessagePayloadSupportsEnvelopeShape(t *testing.T) {
	parsed, err := parseMessagePayload(map[string]any{
		"envelope": map[string]any{
			"messageId":         "msg_env_1",
			"conversationId":    "conv_env_1",
			"recipientUserId":   "usr_b",
			"recipientDeviceId": "dev_b",
			"contentType":       "text/plain",
			"clientMessageSeq":  float64(7),
			"header": map[string]any{
				"scheme":   "debug-placeholder",
				"encoding": "debug-base64-utf8",
				"version":  float64(1),
			},
			"ratchetPublicKey": "debug-rpk-env",
			"ciphertext":       "aGVsbG8gZW52ZWxvcGU=",
		},
	})
	if err != nil {
		t.Fatalf("parse envelope payload failed: %v", err)
	}
	if parsed.MessageID != "msg_env_1" {
		t.Fatalf("expected message id msg_env_1, got %q", parsed.MessageID)
	}
	if parsed.ConversationID != "conv_env_1" {
		t.Fatalf("expected conversation id conv_env_1, got %q", parsed.ConversationID)
	}
	if parsed.RecipientUserID != "usr_b" {
		t.Fatalf("expected recipient user usr_b, got %q", parsed.RecipientUserID)
	}
	if parsed.RecipientDeviceID != "dev_b" {
		t.Fatalf("expected recipient device dev_b, got %q", parsed.RecipientDeviceID)
	}
	if parsed.ClientMessageSeq != 7 {
		t.Fatalf("expected seq 7, got %d", parsed.ClientMessageSeq)
	}
	if parsed.Ciphertext != "aGVsbG8gZW52ZWxvcGU=" {
		t.Fatalf("expected ciphertext to round-trip, got %q", parsed.Ciphertext)
	}
	if parsed.RatchetPublicKey != "debug-rpk-env" {
		t.Fatalf("expected ratchet public key debug-rpk-env, got %q", parsed.RatchetPublicKey)
	}
	if parsed.Body != "hello envelope" {
		t.Fatalf("expected decoded body hello envelope, got %q", parsed.Body)
	}
}

func TestParseMessagePayloadSupportsLegacyShape(t *testing.T) {
	parsed, err := parseMessagePayload(map[string]any{
		"recipientEmail": "bob@example.com",
		"body":           "hello legacy",
	})
	if err != nil {
		t.Fatalf("parse legacy payload failed: %v", err)
	}
	if parsed.RecipientEmail != "bob@example.com" {
		t.Fatalf("expected recipient email bob@example.com, got %q", parsed.RecipientEmail)
	}
	if parsed.Body != "hello legacy" {
		t.Fatalf("expected body hello legacy, got %q", parsed.Body)
	}
	if parsed.Ciphertext == "" {
		t.Fatal("expected generated ciphertext")
	}
	if parsed.Header == nil {
		t.Fatal("expected generated debug header")
	}
	if parsed.RatchetPublicKey == "" {
		t.Fatal("expected generated ratchet public key")
	}
	if parsed.MessageID == "" {
		t.Fatal("expected generated message id")
	}
	if parsed.ConversationID == "" {
		t.Fatal("expected generated conversation id")
	}
}
