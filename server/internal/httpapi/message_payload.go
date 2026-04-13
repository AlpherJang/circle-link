package httpapi

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/circle-link/circle-link/server/internal/domain"
	"github.com/circle-link/circle-link/server/internal/platform/ids"
	"github.com/circle-link/circle-link/server/internal/service/auth"
	"github.com/circle-link/circle-link/server/internal/service/message"
)

type parsedMessagePayload struct {
	MessageID         string
	ConversationID    string
	RecipientUserID   string
	RecipientDeviceID string
	RecipientEmail    string
	ContentType       string
	ClientMessageSeq  uint64
	Header            map[string]any
	RatchetPublicKey  string
	Ciphertext        string
	Body              string
}

func (s *Server) sendDebugMessage(ctx context.Context, senderUserID, senderDeviceID string, payload map[string]any) (domain.DebugMessage, bool, error) {
	parsed, err := parseMessagePayload(payload)
	if err != nil {
		return domain.DebugMessage{}, false, err
	}

	sender, err := s.authService.GetUser(ctx, senderUserID)
	if err != nil {
		return domain.DebugMessage{}, false, errors.New("current user is invalid")
	}

	recipient, err := s.resolveRecipient(ctx, parsed)
	if err != nil {
		return domain.DebugMessage{}, false, err
	}

	recipientDeviceID := parsed.RecipientDeviceID
	recipientSession := s.wsHub.sessionForUserAndDevice(recipient.ID, recipientDeviceID)
	if recipientDeviceID == "" {
		recipientSession = s.wsHub.firstSessionForUser(recipient.ID)
		if recipientSession != nil {
			recipientDeviceID = recipientSession.deviceID
		}
	}

	item, err := s.messageService.Send(ctx, message.SendInput{
		MessageID:         parsed.MessageID,
		ConversationID:    parsed.ConversationID,
		SenderUserID:      sender.ID,
		SenderDeviceID:    senderDeviceID,
		SenderEmail:       sender.Email,
		RecipientUserID:   recipient.ID,
		RecipientDeviceID: recipientDeviceID,
		RecipientEmail:    recipient.Email,
		ContentType:       parsed.ContentType,
		ClientMessageSeq:  parsed.ClientMessageSeq,
		Header:            parsed.Header,
		RatchetPublicKey:  parsed.RatchetPublicKey,
		Ciphertext:        parsed.Ciphertext,
		Body:              parsed.Body,
		RecipientOnline:   recipientSession != nil,
	})
	if err != nil {
		return domain.DebugMessage{}, false, errors.New("failed to send message")
	}

	return item, recipientSession != nil, nil
}

func (s *Server) resolveRecipient(ctx context.Context, parsed parsedMessagePayload) (domain.User, error) {
	if parsed.RecipientUserID != "" {
		recipient, err := s.authService.GetUser(ctx, parsed.RecipientUserID)
		if err != nil {
			if errors.Is(err, auth.ErrUserNotFound) {
				return domain.User{}, errors.New("recipient user was not found")
			}
			return domain.User{}, errors.New("failed to resolve recipient")
		}
		return recipient, nil
	}

	recipient, err := s.authService.FindUserByEmail(ctx, parsed.RecipientEmail)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return domain.User{}, errors.New("recipient email was not found")
		}
		return domain.User{}, errors.New("failed to resolve recipient")
	}

	return recipient, nil
}

func parseMessagePayload(payload map[string]any) (parsedMessagePayload, error) {
	readFrom := payload
	if rawEnvelope, ok := payload["envelope"].(map[string]any); ok {
		readFrom = rawEnvelope
	}

	recipientEmail := stringField(readFrom, "recipientEmail")
	if recipientEmail == "" {
		recipientEmail = stringField(payload, "recipientEmail")
	}
	recipientUserID := stringField(readFrom, "recipientUserId")
	if recipientUserID == "" {
		recipientUserID = stringField(payload, "recipientUserId")
	}
	header := mapField(readFrom, "header")
	if len(header) == 0 {
		header = mapField(payload, "header")
	}
	ratchetPublicKey := stringField(readFrom, "ratchetPublicKey")
	if ratchetPublicKey == "" {
		ratchetPublicKey = stringField(payload, "ratchetPublicKey")
	}
	ciphertext := stringField(readFrom, "ciphertext")
	if ciphertext == "" {
		ciphertext = stringField(payload, "ciphertext")
	}
	body := stringField(readFrom, "body")
	if body == "" {
		body = stringField(readFrom, "plaintext")
	}
	if body == "" {
		body = stringField(payload, "body")
	}
	if body == "" {
		body = stringField(payload, "plaintext")
	}
	contentType := stringField(readFrom, "contentType")
	if contentType == "" {
		contentType = "text/plain"
	}

	if recipientEmail == "" && recipientUserID == "" {
		return parsedMessagePayload{}, errors.New("recipientEmail or recipientUserId is required")
	}
	if ciphertext == "" && body == "" {
		return parsedMessagePayload{}, errors.New("message ciphertext or body is required")
	}

	messageID := stringField(readFrom, "messageId")
	if messageID == "" {
		messageID = ids.New("msg")
	}
	conversationID := stringField(readFrom, "conversationId")
	if conversationID == "" {
		conversationID = fmt.Sprintf("conv_%s", messageID)
	}
	if ciphertext == "" && body != "" {
		ciphertext = base64.StdEncoding.EncodeToString([]byte(body))
	}
	if len(header) == 0 {
		header = defaultDebugHeader()
	}
	if ratchetPublicKey == "" {
		ratchetPublicKey = fmt.Sprintf("debug-rpk-%s", messageID)
	}
	if body == "" {
		body = decodeDebugCiphertext(ciphertext, header)
	}

	return parsedMessagePayload{
		MessageID:         messageID,
		ConversationID:    conversationID,
		RecipientUserID:   recipientUserID,
		RecipientDeviceID: stringField(readFrom, "recipientDeviceId"),
		RecipientEmail:    recipientEmail,
		ContentType:       contentType,
		ClientMessageSeq:  uint64Field(readFrom, "clientMessageSeq"),
		Header:            header,
		RatchetPublicKey:  ratchetPublicKey,
		Ciphertext:        ciphertext,
		Body:              body,
	}, nil
}

func defaultDebugHeader() map[string]any {
	return map[string]any{
		"scheme":   "debug-placeholder",
		"encoding": "debug-base64-utf8",
		"version":  1,
	}
}

func decodeDebugCiphertext(ciphertext string, header map[string]any) string {
	if ciphertext == "" || len(header) == 0 {
		return ""
	}
	if stringField(header, "encoding") != "debug-base64-utf8" {
		return ""
	}

	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return ""
	}

	return string(decoded)
}

func mapField(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return nil
	}

	switch value := payload[key].(type) {
	case map[string]any:
		return cloneMap(value)
	case map[string]string:
		cloned := make(map[string]any, len(value))
		for nestedKey, nestedValue := range value {
			cloned[nestedKey] = nestedValue
		}
		return cloned
	default:
		return nil
	}
}

func cloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(value))
	for key, nestedValue := range value {
		cloned[key] = nestedValue
	}

	return cloned
}

func stringField(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, _ := payload[key].(string)
	return value
}

func uint64Field(payload map[string]any, key string) uint64 {
	if payload == nil {
		return 0
	}
	switch value := payload[key].(type) {
	case float64:
		return uint64(value)
	case int:
		return uint64(value)
	case int64:
		return uint64(value)
	case uint64:
		return value
	default:
		return 0
	}
}
