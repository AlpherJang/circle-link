package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/circle-link/circle-link/server/internal/service/auth"
	"github.com/circle-link/circle-link/server/internal/service/contact"
	"github.com/circle-link/circle-link/server/internal/service/device"
	"github.com/circle-link/circle-link/server/internal/service/message"
)

func TestAuthDeviceAndMessageFlow(t *testing.T) {
	authService := auth.NewMemoryService()
	contactService := contact.NewMemoryService()
	deviceService := device.NewMemoryService()
	messageService := message.NewMemoryService()
	router := NewRouterWithServices(authService, contactService, deviceService, messageService)

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

	aliceConversationsResp := doJSONRequest(t, router, http.MethodGet, "/v1/conversations", aliceToken, nil)
	if aliceConversationsResp.Code != http.StatusOK {
		t.Fatalf("expected alice conversations status 200, got %d", aliceConversationsResp.Code)
	}
	aliceConversationItems := decodeEnvelope(t, aliceConversationsResp)["data"].(map[string]any)["items"].([]any)
	if len(aliceConversationItems) != 1 {
		t.Fatalf("expected 1 alice conversation, got %d", len(aliceConversationItems))
	}
	aliceConversation := aliceConversationItems[0].(map[string]any)
	if aliceConversation["peerEmail"] != bobEmail {
		t.Fatalf("expected alice conversation peer bob@example.com, got %#v", aliceConversation["peerEmail"])
	}
	if aliceConversation["messageCount"] != float64(1) {
		t.Fatalf("expected alice conversation messageCount 1, got %#v", aliceConversation["messageCount"])
	}

	bobConversationsResp := doJSONRequest(t, router, http.MethodGet, "/v1/conversations", bobToken, nil)
	if bobConversationsResp.Code != http.StatusOK {
		t.Fatalf("expected bob conversations status 200, got %d", bobConversationsResp.Code)
	}
	bobConversationItems := decodeEnvelope(t, bobConversationsResp)["data"].(map[string]any)["items"].([]any)
	if len(bobConversationItems) != 1 {
		t.Fatalf("expected 1 bob conversation, got %d", len(bobConversationItems))
	}
	bobConversation := bobConversationItems[0].(map[string]any)
	if bobConversation["peerEmail"] != aliceEmail {
		t.Fatalf("expected bob conversation peer alice@example.com, got %#v", bobConversation["peerEmail"])
	}
	if bobConversation["unreadCount"] != float64(1) {
		t.Fatalf("expected bob conversation unreadCount 1, got %#v", bobConversation["unreadCount"])
	}
}

func TestContactInviteAndListFlow(t *testing.T) {
	authService := auth.NewMemoryService()
	contactService := contact.NewMemoryService()
	deviceService := device.NewMemoryService()
	messageService := message.NewMemoryService()
	router := NewRouterWithServices(authService, contactService, deviceService, messageService)

	aliceToken := signUpAndVerify(t, router, "alice@example.com", "strong-pass", "Alice")
	bobToken := signUpAndVerify(t, router, "bob@example.com", "strong-pass", "Bob")

	inviteResp := doJSONRequest(t, router, http.MethodPost, "/v1/contacts/invite", aliceToken, map[string]any{
		"peerEmail": "bob@example.com",
	})
	if inviteResp.Code != http.StatusCreated {
		t.Fatalf("expected invite status 201, got %d", inviteResp.Code)
	}

	listResp := doJSONRequest(t, router, http.MethodGet, "/v1/contacts", aliceToken, nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected contact list status 200, got %d", listResp.Code)
	}

	payload := decodeEnvelope(t, listResp)
	data := payload["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 contact item, got %d", len(items))
	}

	item := items[0].(map[string]any)
	if item["peerEmail"] != "bob@example.com" {
		t.Fatalf("expected contact peerEmail bob@example.com, got %#v", item["peerEmail"])
	}
	if item["state"] != "pending" {
		t.Fatalf("expected contact state pending, got %#v", item["state"])
	}
	if item["direction"] != "outgoing" {
		t.Fatalf("expected contact direction outgoing, got %#v", item["direction"])
	}

	bobListResp := doJSONRequest(t, router, http.MethodGet, "/v1/contacts", bobToken, nil)
	if bobListResp.Code != http.StatusOK {
		t.Fatalf("expected bob contact list status 200, got %d", bobListResp.Code)
	}

	bobPayload := decodeEnvelope(t, bobListResp)
	bobItems := bobPayload["data"].(map[string]any)["items"].([]any)
	if len(bobItems) != 1 {
		t.Fatalf("expected 1 bob contact item, got %d", len(bobItems))
	}
	bobItem := bobItems[0].(map[string]any)
	if bobItem["direction"] != "incoming" {
		t.Fatalf("expected bob contact direction incoming, got %#v", bobItem["direction"])
	}
	if bobItem["canAccept"] != true {
		t.Fatalf("expected bob contact canAccept true, got %#v", bobItem["canAccept"])
	}
}

func TestContactAcceptAndRejectFlow(t *testing.T) {
	authService := auth.NewMemoryService()
	contactService := contact.NewMemoryService()
	deviceService := device.NewMemoryService()
	messageService := message.NewMemoryService()
	router := NewRouterWithServices(authService, contactService, deviceService, messageService)

	aliceToken := signUpAndVerify(t, router, "alice@example.com", "strong-pass", "Alice")
	bobToken := signUpAndVerify(t, router, "bob@example.com", "strong-pass", "Bob")
	charlieToken := signUpAndVerify(t, router, "charlie@example.com", "strong-pass", "Charlie")

	inviteResp := doJSONRequest(t, router, http.MethodPost, "/v1/contacts/invite", aliceToken, map[string]any{
		"peerEmail": "bob@example.com",
	})
	if inviteResp.Code != http.StatusCreated {
		t.Fatalf("expected invite status 201, got %d", inviteResp.Code)
	}
	bobPendingListResp := doJSONRequest(t, router, http.MethodGet, "/v1/contacts", bobToken, nil)
	bobPendingItems := decodeEnvelope(t, bobPendingListResp)["data"].(map[string]any)["items"].([]any)
	alicePeerUserID := bobPendingItems[0].(map[string]any)["peerUserId"].(string)

	acceptResp := doJSONRequest(t, router, http.MethodPost, "/v1/contacts/"+alicePeerUserID+"/accept", bobToken, nil)
	if acceptResp.Code != http.StatusOK {
		t.Fatalf("expected accept status 200, got %d", acceptResp.Code)
	}
	acceptData := decodeEnvelope(t, acceptResp)["data"].(map[string]any)
	if acceptData["state"] != "accepted" {
		t.Fatalf("expected accepted state, got %#v", acceptData["state"])
	}
	if acceptData["direction"] != "accepted" {
		t.Fatalf("expected accepted direction, got %#v", acceptData["direction"])
	}

	secondInviteResp := doJSONRequest(t, router, http.MethodPost, "/v1/contacts/invite", aliceToken, map[string]any{
		"peerEmail": "charlie@example.com",
	})
	if secondInviteResp.Code != http.StatusCreated {
		t.Fatalf("expected second invite status 201, got %d", secondInviteResp.Code)
	}
	charliePendingListResp := doJSONRequest(t, router, http.MethodGet, "/v1/contacts", charlieToken, nil)
	charliePendingItems := decodeEnvelope(t, charliePendingListResp)["data"].(map[string]any)["items"].([]any)
	aliceToCharliePeerUserID := charliePendingItems[0].(map[string]any)["peerUserId"].(string)

	rejectResp := doJSONRequest(t, router, http.MethodPost, "/v1/contacts/"+aliceToCharliePeerUserID+"/reject", charlieToken, nil)
	if rejectResp.Code != http.StatusOK {
		t.Fatalf("expected reject status 200, got %d", rejectResp.Code)
	}

	charlieListResp := doJSONRequest(t, router, http.MethodGet, "/v1/contacts", charlieToken, nil)
	charlieItems := decodeEnvelope(t, charlieListResp)["data"].(map[string]any)["items"].([]any)
	if len(charlieItems) != 0 {
		t.Fatalf("expected charlie contacts to be empty after reject, got %#v", charlieItems)
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
