package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/circle-link/circle-link/server/internal/service/auth"
	"github.com/circle-link/circle-link/server/internal/service/contact"
	"github.com/circle-link/circle-link/server/internal/service/device"
	"github.com/circle-link/circle-link/server/internal/service/message"
	"golang.org/x/net/websocket"
)

func TestWebSocketRelayFlow(t *testing.T) {
	authService := auth.NewMemoryService()
	contactService := contact.NewMemoryService()
	deviceService := device.NewMemoryService()
	messageService := message.NewMemoryService()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("websocket integration test requires local listen support: %v", err)
	}
	server := httptest.NewUnstartedServer(NewRouterWithServices(authService, contactService, deviceService, messageService))
	server.Listener = listener
	server.Start()
	defer server.Close()

	aliceSession := signUpVerifyAndLoginOverHTTP(t, server.URL, "alice@example.com", "strong-pass", "Alice")
	bobSession := signUpVerifyAndLoginOverHTTP(t, server.URL, "bob@example.com", "strong-pass", "Bob")
	aliceDeviceID := registerDeviceOverHTTP(t, server.URL, aliceSession.accessToken, "Alice's Mac")
	bobDeviceID := registerDeviceOverHTTP(t, server.URL, bobSession.accessToken, "Bob's Mac")

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/ws"
	aliceConn := dialWebSocket(t, wsURL)
	defer aliceConn.Close()
	bobConn := dialWebSocket(t, wsURL)
	defer bobConn.Close()

	sendWSBind(t, aliceConn, aliceSession, aliceDeviceID)
	sendWSBind(t, bobConn, bobSession, bobDeviceID)
	awaitWSEvent(t, aliceConn, "session.bound")
	awaitWSEvent(t, bobConn, "session.bound")

	if err := websocket.JSON.Send(aliceConn, map[string]any{
		"type": "message.send",
		"payload": map[string]any{
			"messageId":        "msg_ws_1",
			"conversationId":   "conv_ws_1",
			"recipientEmail":   "bob@example.com",
			"contentType":      "text/plain",
			"clientMessageSeq": 1,
			"header": map[string]any{
				"scheme":   "debug-placeholder",
				"encoding": "debug-base64-utf8",
				"version":  1,
			},
			"ratchetPublicKey": "debug-rpk-ws-1",
			"ciphertext":       "aGVsbG8gb3ZlciB3ZWJzb2NrZXQ=",
		},
	}); err != nil {
		t.Fatalf("send websocket message: %v", err)
	}

	event := awaitWSEvent(t, bobConn, "message.deliver")
	payload := event["payload"].(map[string]any)
	if payload["body"] != "hello over websocket" {
		t.Fatalf("expected websocket body %q, got %#v", "hello over websocket", payload["body"])
	}
	if payload["ciphertext"] != "aGVsbG8gb3ZlciB3ZWJzb2NrZXQ=" {
		t.Fatalf("expected websocket ciphertext to round-trip, got %#v", payload["ciphertext"])
	}
	if payload["senderEmail"] != "alice@example.com" {
		t.Fatalf("expected sender email alice@example.com, got %#v", payload["senderEmail"])
	}
	if payload["messageId"] != "msg_ws_1" {
		t.Fatalf("expected message id msg_ws_1, got %#v", payload["messageId"])
	}
	if payload["conversationId"] != "conv_ws_1" {
		t.Fatalf("expected conversation id conv_ws_1, got %#v", payload["conversationId"])
	}

	acceptedAck := awaitWSEvent(t, aliceConn, "delivery.ack")
	acceptedPayload := acceptedAck["payload"].(map[string]any)
	if acceptedPayload["status"] != "accepted" {
		t.Fatalf("expected accepted ack, got %#v", acceptedPayload["status"])
	}
	if acceptedPayload["recipientDeviceId"] != bobDeviceID {
		t.Fatalf("expected accepted ack recipient device %q, got %#v", bobDeviceID, acceptedPayload["recipientDeviceId"])
	}
	if acceptedPayload["conversationId"] != "conv_ws_1" {
		t.Fatalf("expected accepted ack conversation id conv_ws_1, got %#v", acceptedPayload["conversationId"])
	}
	if acceptedPayload["senderDeviceId"] != aliceDeviceID {
		t.Fatalf("expected accepted ack sender device %q, got %#v", aliceDeviceID, acceptedPayload["senderDeviceId"])
	}

	if err := websocket.JSON.Send(bobConn, map[string]any{
		"type": "message.ack",
		"payload": map[string]any{
			"messageId": payload["messageId"],
			"status":    "delivered",
		},
	}); err != nil {
		t.Fatalf("send websocket ack: %v", err)
	}

	ackEvent := awaitWSEvent(t, aliceConn, "delivery.ack")
	ackPayload := ackEvent["payload"].(map[string]any)
	if ackPayload["status"] != "delivered" {
		t.Fatalf("expected delivered ack, got %#v", ackPayload["status"])
	}
	if ackPayload["recipientDeviceId"] != bobDeviceID {
		t.Fatalf("expected delivered ack recipient device %q, got %#v", bobDeviceID, ackPayload["recipientDeviceId"])
	}
	if ackPayload["conversationId"] != "conv_ws_1" {
		t.Fatalf("expected delivered ack conversation id conv_ws_1, got %#v", ackPayload["conversationId"])
	}
	if ackPayload["senderDeviceId"] != aliceDeviceID {
		t.Fatalf("expected delivered ack sender device %q, got %#v", aliceDeviceID, ackPayload["senderDeviceId"])
	}

	if err := websocket.JSON.Send(bobConn, map[string]any{
		"type": "message.ack",
		"payload": map[string]any{
			"messageId": payload["messageId"],
			"status":    "read",
		},
	}); err != nil {
		t.Fatalf("send websocket read ack: %v", err)
	}

	readEvent := awaitWSEvent(t, aliceConn, "delivery.ack")
	readPayload := readEvent["payload"].(map[string]any)
	if readPayload["status"] != "read" {
		t.Fatalf("expected read ack, got %#v", readPayload["status"])
	}
	if readPayload["recipientDeviceId"] != bobDeviceID {
		t.Fatalf("expected read ack recipient device %q, got %#v", bobDeviceID, readPayload["recipientDeviceId"])
	}
}

func TestWebSocketRelayTargetsSpecificRecipientDevice(t *testing.T) {
	authService := auth.NewMemoryService()
	contactService := contact.NewMemoryService()
	deviceService := device.NewMemoryService()
	messageService := message.NewMemoryService()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("websocket integration test requires local listen support: %v", err)
	}
	server := httptest.NewUnstartedServer(NewRouterWithServices(authService, contactService, deviceService, messageService))
	server.Listener = listener
	server.Start()
	defer server.Close()

	aliceSession := signUpVerifyAndLoginOverHTTP(t, server.URL, "alice@example.com", "strong-pass", "Alice")
	bobSession := signUpVerifyAndLoginOverHTTP(t, server.URL, "bob@example.com", "strong-pass", "Bob")
	aliceDeviceID := registerDeviceOverHTTP(t, server.URL, aliceSession.accessToken, "Alice's Mac")
	bobPhoneID := registerDeviceOverHTTP(t, server.URL, bobSession.accessToken, "Bob's Phone")
	bobTabletID := registerDeviceOverHTTP(t, server.URL, bobSession.accessToken, "Bob's Tablet")

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/ws"
	aliceConn := dialWebSocket(t, wsURL)
	defer aliceConn.Close()
	bobPhoneConn := dialWebSocket(t, wsURL)
	defer bobPhoneConn.Close()
	bobTabletConn := dialWebSocket(t, wsURL)
	defer bobTabletConn.Close()

	sendWSBind(t, aliceConn, aliceSession, aliceDeviceID)
	sendWSBind(t, bobPhoneConn, bobSession, bobPhoneID)
	sendWSBind(t, bobTabletConn, bobSession, bobTabletID)
	awaitWSEvent(t, aliceConn, "session.bound")
	awaitWSEvent(t, bobPhoneConn, "session.bound")
	awaitWSEvent(t, bobTabletConn, "session.bound")

	if err := websocket.JSON.Send(aliceConn, map[string]any{
		"type": "message.send",
		"payload": map[string]any{
			"messageId":         "msg_ws_targeted",
			"conversationId":    "conv_ws_targeted",
			"recipientEmail":    "bob@example.com",
			"recipientDeviceId": bobPhoneID,
			"contentType":       "text/plain",
			"clientMessageSeq":  2,
			"header": map[string]any{
				"scheme":   "debug-placeholder",
				"encoding": "debug-base64-utf8",
				"version":  1,
			},
			"ratchetPublicKey": "debug-rpk-targeted",
			"ciphertext":       "aGVsbG8gc3BlY2lmaWMgZGV2aWNl",
		},
	}); err != nil {
		t.Fatalf("send targeted websocket message: %v", err)
	}

	targetedEvent := awaitWSEvent(t, bobPhoneConn, "message.deliver")
	targetedPayload := targetedEvent["payload"].(map[string]any)
	if targetedPayload["recipientDeviceId"] != bobPhoneID {
		t.Fatalf("expected targeted recipient device %q, got %#v", bobPhoneID, targetedPayload["recipientDeviceId"])
	}
	assertNoWSEvent(t, bobTabletConn, 250*time.Millisecond)

	acceptedAck := awaitWSEvent(t, aliceConn, "delivery.ack")
	acceptedPayload := acceptedAck["payload"].(map[string]any)
	if acceptedPayload["recipientDeviceId"] != bobPhoneID {
		t.Fatalf("expected accepted ack recipient device %q, got %#v", bobPhoneID, acceptedPayload["recipientDeviceId"])
	}
}

type loginSession struct {
	userID      string
	accessToken string
}

func signUpVerifyAndLoginOverHTTP(t *testing.T, baseURL, email, password, displayName string) loginSession {
	t.Helper()

	signUpPayload := doHTTPJSON(t, baseURL, http.MethodPost, "/v1/auth/signup", "", map[string]any{
		"email":       email,
		"password":    password,
		"displayName": displayName,
	})
	verifyToken := signUpPayload["data"].(map[string]any)["verificationToken"].(string)

	doHTTPJSON(t, baseURL, http.MethodPost, "/v1/auth/verify-email", "", map[string]any{
		"email":             email,
		"verificationToken": verifyToken,
	})

	loginPayload := doHTTPJSON(t, baseURL, http.MethodPost, "/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": password,
	})

	data := loginPayload["data"].(map[string]any)
	return loginSession{
		userID:      data["userId"].(string),
		accessToken: data["accessToken"].(string),
	}
}

func registerDeviceOverHTTP(t *testing.T, baseURL, accessToken, deviceName string) string {
	t.Helper()

	payload := doHTTPJSON(t, baseURL, http.MethodPost, "/v1/devices", accessToken, map[string]any{
		"deviceName": deviceName,
		"platform":   "macos",
		"pushToken":  "",
		"keyBundle": map[string]any{
			"identityKeyPublic":     "identity-" + deviceName,
			"signedPrekeyPublic":    "signed-" + deviceName,
			"signedPrekeySignature": "sig-" + deviceName,
			"signedPrekeyVersion":   1,
			"oneTimePrekeys":        []string{"otp-1", "otp-2"},
		},
	})

	return payload["data"].(map[string]any)["deviceId"].(string)
}

func doHTTPJSON(t *testing.T, baseURL, method, path, accessToken string, body any) map[string]any {
	t.Helper()

	var requestBody []byte
	var err error
	if body != nil {
		requestBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}

	req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.StatusCode >= 400 {
		t.Fatalf("unexpected status %d: %#v", resp.StatusCode, payload)
	}

	return payload
}

func dialWebSocket(t *testing.T, url string) *websocket.Conn {
	t.Helper()

	conn, err := websocket.Dial(url, "", "http://127.0.0.1/")
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}

	return conn
}

func sendWSBind(t *testing.T, conn *websocket.Conn, session loginSession, deviceID string) {
	t.Helper()

	if err := websocket.JSON.Send(conn, map[string]any{
		"type": "session.bind",
		"payload": map[string]any{
			"accessToken": session.accessToken,
			"userId":      session.userID,
			"deviceId":    deviceID,
		},
	}); err != nil {
		t.Fatalf("send websocket bind: %v", err)
	}
}

func awaitWSEvent(t *testing.T, conn *websocket.Conn, eventType string) map[string]any {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for websocket event %q", eventType)
		}

		event := awaitAnyWSEvent(t, conn)
		if event["type"] == eventType {
			return event
		}
	}
}

func awaitAnyWSEvent(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()

	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set websocket deadline: %v", err)
	}

	var event map[string]any
	if err := websocket.JSON.Receive(conn, &event); err != nil {
		t.Fatalf("receive websocket event: %v", err)
	}

	return event
}

func assertNoWSEvent(t *testing.T, conn *websocket.Conn, timeout time.Duration) {
	t.Helper()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("set websocket deadline: %v", err)
	}
	defer func() {
		if err := conn.SetDeadline(time.Time{}); err != nil {
			t.Fatalf("reset websocket deadline: %v", err)
		}
	}()

	var event map[string]any
	err := websocket.JSON.Receive(conn, &event)
	if err == nil {
		t.Fatalf("expected no websocket event, got %#v", event)
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout waiting for no event, got %v", err)
	}
}
