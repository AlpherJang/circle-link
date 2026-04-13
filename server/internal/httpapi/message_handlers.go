package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/circle-link/circle-link/server/internal/domain"
)

type sendMessageRequest struct {
	MessageID         string         `json:"messageId"`
	ConversationID    string         `json:"conversationId"`
	RecipientEmail    string         `json:"recipientEmail"`
	RecipientUserID   string         `json:"recipientUserId"`
	RecipientDeviceID string         `json:"recipientDeviceId"`
	ContentType       string         `json:"contentType"`
	ClientMessageSeq  uint64         `json:"clientMessageSeq"`
	Header            map[string]any `json:"header"`
	RatchetPublicKey  string         `json:"ratchetPublicKey"`
	Ciphertext        string         `json:"ciphertext"`
	Body              string         `json:"body"`
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	user, err := s.authService.GetUser(r.Context(), session.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "AUTH_USER_NOT_FOUND", "Current user was not found.")
		return
	}

	writeData(w, http.StatusOK, map[string]any{
		"userId":      user.ID,
		"email":       user.Email,
		"displayName": user.DisplayName,
		"status":      user.Status,
	})
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	var req sendMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON request body.")
		return
	}
	item, _, err := s.sendDebugMessage(r.Context(), session.UserID, "", map[string]any{
		"messageId":         req.MessageID,
		"conversationId":    req.ConversationID,
		"recipientEmail":    req.RecipientEmail,
		"recipientUserId":   req.RecipientUserID,
		"recipientDeviceId": req.RecipientDeviceID,
		"contentType":       req.ContentType,
		"clientMessageSeq":  req.ClientMessageSeq,
		"header":            req.Header,
		"ratchetPublicKey":  req.RatchetPublicKey,
		"ciphertext":        req.Ciphertext,
		"body":              req.Body,
	})
	if err != nil {
		message := err.Error()
		status := http.StatusInternalServerError
		code := "INTERNAL_ERROR"
		if message == "recipient email was not found" {
			status = http.StatusNotFound
			code = "MESSAGE_RECIPIENT_NOT_FOUND"
		} else if message == "recipientEmail or recipientUserId is required" || message == "message ciphertext or body is required" {
			status = http.StatusBadRequest
			code = "VALIDATION_FAILED"
		} else if message == "recipient user was not found" {
			status = http.StatusNotFound
			code = "MESSAGE_RECIPIENT_NOT_FOUND"
		} else if message == "current user is invalid" {
			status = http.StatusUnauthorized
			code = "AUTH_UNAUTHORIZED"
		}
		writeError(w, status, code, upperFirst(message))
		return
	}

	writeData(w, http.StatusCreated, serializeMessage(item))
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	deviceID := r.URL.Query().Get("deviceId")
	items, err := s.messageService.ListInbox(r.Context(), session.UserID, deviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load inbox.")
		return
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, serializeMessage(item))
	}

	writeData(w, http.StatusOK, map[string]any{
		"items": result,
	})
}

func (s *Server) handleMessageStream(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSessionFromRequestOrQuery(w, r)
	if !ok {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "STREAM_UNSUPPORTED", "Streaming is not supported by this server.")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	deviceID := r.URL.Query().Get("deviceId")

	stream, cancel := s.messageService.SubscribeInbox(r.Context(), session.UserID, deviceID)
	defer cancel()

	existingItems, err := s.messageService.ListInbox(r.Context(), session.UserID, deviceID)
	if err == nil {
		for _, item := range existingItems {
			if err := writeSSEEvent(w, "snapshot", serializeMessage(item)); err != nil {
				return
			}
			flusher.Flush()
		}
	}

	if err := writeSSEEvent(w, "ready", map[string]any{"ok": true}); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case item, ok := <-stream:
			if !ok {
				return
			}
			if err := writeSSEEvent(w, "message", serializeMessage(item)); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func serializeMessage(item domain.DebugMessage) map[string]any {
	return map[string]any{
		"messageId":         item.ID,
		"conversationId":    item.ConversationID,
		"senderUserId":      item.SenderUserID,
		"senderDeviceId":    item.SenderDeviceID,
		"senderEmail":       item.SenderEmail,
		"recipientUserId":   item.RecipientUserID,
		"recipientDeviceId": item.RecipientDeviceID,
		"recipientEmail":    item.RecipientEmail,
		"contentType":       item.ContentType,
		"clientMessageSeq":  item.ClientMessageSeq,
		"header":            item.Header,
		"ratchetPublicKey":  item.RatchetPublicKey,
		"ciphertext":        item.Ciphertext,
		"body":              item.Body,
		"deliveryStatus":    item.DeliveryStatus,
		"storedAt":          item.StoredAt.UTC().Format(time.RFC3339),
		"deliveredAt":       formatOptionalTime(item.DeliveredAt),
		"readAt":            formatOptionalTime(item.ReadAt),
		"sentAt":            item.SentAt.UTC().Format(time.RFC3339),
	}
}

func formatOptionalTime(value *time.Time) any {
	if value == nil {
		return nil
	}

	return value.UTC().Format(time.RFC3339)
}

func writeSSEEvent(w http.ResponseWriter, event string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", encoded); err != nil {
		return err
	}

	return nil
}

func upperFirst(value string) string {
	if value == "" {
		return value
	}

	return string(value[0]-32) + value[1:]
}
