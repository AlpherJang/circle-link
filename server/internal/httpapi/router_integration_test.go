package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/circle-link/circle-link/server/internal/service/auth"
	"github.com/circle-link/circle-link/server/internal/service/device"
	"github.com/circle-link/circle-link/server/internal/service/message"
)

func TestAuthDeviceAndMessageFlow(t *testing.T) {
	authService := auth.NewMemoryService()
	deviceService := device.NewMemoryService()
	messageService := message.NewMemoryService()
	router := NewRouterWithServices(authService, deviceService, messageService)

	aliceEmail := "alice@example.com"
	alicePassword := "strong-pass"
	bobEmail := "bob@example.com"
	bobPassword := "strong-pass"

	aliceToken := signUpAndVerify(t, router, aliceEmail, alicePassword, "Alice")
	bobToken := signUpAndVerify(t, router, bobEmail, bobPassword, "Bob")

	registerDevice(t, router, aliceToken, "Alice's Mac")
	bobDeviceID := registerDevice(t, router, bobToken, "Bob's Mac")

	sendResp := doJSONRequest(t, router, http.MethodPost, "/v1/messages", aliceToken, map[string]any{
		"recipientEmail":   bobEmail,
		"contentType":      "text/plain",
		"header":           map[string]any{"scheme": "debug-placeholder", "encoding": "debug-base64-utf8", "version": 1},
		"ratchetPublicKey": "debug-rpk-router",
		"ciphertext":       "aGVsbG8gYm9i",
	})
	if sendResp.Code != http.StatusCreated {
		t.Fatalf("expected send status 201, got %d", sendResp.Code)
	}

	inboxResp := doJSONRequest(t, router, http.MethodGet, "/v1/messages?deviceId="+bobDeviceID, bobToken, nil)
	if inboxResp.Code != http.StatusOK {
		t.Fatalf("expected inbox status 200, got %d", inboxResp.Code)
	}

	payload := decodeEnvelope(t, inboxResp)
	data := payload["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 inbox item, got %d", len(items))
	}

	item := items[0].(map[string]any)
	if item["ciphertext"] != "aGVsbG8gYm9i" {
		t.Fatalf("expected ciphertext aGVsbG8gYm9i, got %#v", item["ciphertext"])
	}
	if item["deliveryStatus"] != "stored_offline" {
		t.Fatalf("expected delivery status stored_offline, got %#v", item["deliveryStatus"])
	}
	if item["body"] != "hello bob" {
		t.Fatalf("expected message body hello bob, got %#v", item["body"])
	}
	if item["senderEmail"] != aliceEmail {
		t.Fatalf("expected sender email %q, got %#v", aliceEmail, item["senderEmail"])
	}
}

func signUpAndVerify(t *testing.T, router http.Handler, email, password, displayName string) string {
	t.Helper()

	signUpResp := doJSONRequest(t, router, http.MethodPost, "/v1/auth/signup", "", map[string]any{
		"email":       email,
		"password":    password,
		"displayName": displayName,
	})
	if signUpResp.Code != http.StatusCreated {
		t.Fatalf("expected signup status 201, got %d", signUpResp.Code)
	}
	signUpPayload := decodeEnvelope(t, signUpResp)
	signUpData := signUpPayload["data"].(map[string]any)
	verifyToken := signUpData["verificationToken"].(string)

	verifyResp := doJSONRequest(t, router, http.MethodPost, "/v1/auth/verify-email", "", map[string]any{
		"email":             email,
		"verificationToken": verifyToken,
	})
	if verifyResp.Code != http.StatusOK {
		t.Fatalf("expected verify status 200, got %d", verifyResp.Code)
	}

	loginResp := doJSONRequest(t, router, http.MethodPost, "/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": password,
	})
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginResp.Code)
	}
	loginPayload := decodeEnvelope(t, loginResp)
	loginData := loginPayload["data"].(map[string]any)
	return loginData["accessToken"].(string)
}

func registerDevice(t *testing.T, router http.Handler, accessToken, deviceName string) string {
	t.Helper()

	resp := doJSONRequest(t, router, http.MethodPost, "/v1/devices", accessToken, map[string]any{
		"deviceName": deviceName,
		"platform":   "macos",
		"pushToken":  "",
		"keyBundle": map[string]any{
			"identityKeyPublic":     "identity",
			"signedPrekeyPublic":    "signed",
			"signedPrekeySignature": "sig",
			"signedPrekeyVersion":   1,
			"oneTimePrekeys":        []string{"otp1", "otp2"},
		},
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected device registration status 201, got %d", resp.Code)
	}

	payload := decodeEnvelope(t, resp)
	return payload["data"].(map[string]any)["deviceId"].(string)
}

func doJSONRequest(t *testing.T, router http.Handler, method, path, accessToken string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	return payload
}
