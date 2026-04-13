package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/circle-link/circle-link/server/internal/domain"
	"github.com/circle-link/circle-link/server/internal/service/message"
	"golang.org/x/net/websocket"
)

type wsClientEvent struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type wsServerEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

func (s *Server) websocketHandler() http.Handler {
	return websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()

		bound, stream, cancel, err := s.bindWebSocketSession(conn)
		if err != nil {
			_ = websocket.JSON.Send(conn, wsServerEvent{
				Type: "system.error",
				Payload: map[string]any{
					"message": err.Error(),
				},
			})
			return
		}
		defer cancel()
		defer s.wsHub.unbind(bound.userID, bound.deviceID)

		if existing, err := s.messageService.ListInbox(conn.Request().Context(), bound.userID, bound.deviceID); err == nil {
			for _, item := range existing {
				if !shouldDeliverToBoundDevice(item, bound.deviceID) {
					continue
				}
				bound.send <- wsServerEvent{
					Type:    "message.mailbox",
					Payload: serializeMessage(item),
				}
			}
		}
		bound.send <- wsServerEvent{
			Type: "session.bound",
			Payload: map[string]any{
				"userId":   bound.userID,
				"deviceId": bound.deviceID,
			},
		}

		go func() {
			for item := range stream {
				if !shouldDeliverToBoundDevice(item, bound.deviceID) {
					continue
				}
				bound.send <- wsServerEvent{
					Type:    "message.deliver",
					Payload: serializeMessage(item),
				}
			}
		}()

		go func() {
			for event := range bound.send {
				if err := websocket.JSON.Send(conn, event); err != nil {
					return
				}
			}
		}()

		for {
			var event wsClientEvent
			if err := websocket.JSON.Receive(conn, &event); err != nil {
				return
			}

			switch event.Type {
			case "message.send":
				item, recipientOnline, err := s.sendDebugMessage(conn.Request().Context(), bound.userID, bound.deviceID, event.Payload)
				if err != nil {
					bound.send <- wsServerEvent{
						Type: "system.error",
						Payload: map[string]any{
							"message": err.Error(),
						},
					}
					continue
				}

				s.wsHub.trackPendingAck(wsAckContextFromMessage(item), bound)
				bound.send <- wsServerEvent{
					Type: "delivery.ack",
					Payload: serializeDeliveryAck(
						wsAckContextFromMessage(item),
						"",
						string(domain.DeliveryStatusAccepted),
						!recipientOnline,
						time.Now().UTC(),
					),
				}
			case "message.ack":
				messageID, _ := event.Payload["messageId"].(string)
				if messageID == "" {
					bound.send <- wsServerEvent{
						Type: "system.error",
						Payload: map[string]any{
							"message": "messageId is required for message.ack",
						},
					}
					continue
				}

				updatedItem, err := s.messageService.Acknowledge(conn.Request().Context(), message.AcknowledgeInput{
					MessageID:         messageID,
					RecipientUserID:   bound.userID,
					RecipientDeviceID: bound.deviceID,
					Status:            domain.DeliveryStatus(ackStatusOrDefault(event.Payload["status"], string(domain.DeliveryStatusDelivered))),
				})
				if err != nil {
					continue
				}

				entry, ok := s.wsHub.pendingAck(messageID)
				if !ok || entry.sender == nil {
					continue
				}

				entry.sender.send <- wsServerEvent{
					Type: "delivery.ack",
					Payload: serializeDeliveryAck(
						wsAckContextFromMessage(updatedItem),
						updatedItem.RecipientDeviceID,
						string(updatedItem.DeliveryStatus),
						false,
						time.Now().UTC(),
					),
				}
				if shouldClearPendingAck(updatedItem.DeliveryStatus) {
					s.wsHub.clearPendingAck(messageID)
				}
			case "ping":
				bound.send <- wsServerEvent{
					Type: "pong",
					Payload: map[string]any{
						"ok": true,
					},
				}
			default:
				bound.send <- wsServerEvent{
					Type: "system.error",
					Payload: map[string]any{
						"message": "unknown websocket event type",
					},
				}
			}
		}
	})
}

func (s *Server) bindWebSocketSession(conn *websocket.Conn) (*boundWSSession, <-chan domain.DebugMessage, func(), error) {
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, nil, nil, err
	}

	var event wsClientEvent
	if err := websocket.JSON.Receive(conn, &event); err != nil {
		return nil, nil, nil, err
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return nil, nil, nil, err
	}

	if event.Type != "session.bind" {
		return nil, nil, nil, errors.New("first websocket event must be session.bind")
	}

	accessToken, _ := event.Payload["accessToken"].(string)
	userID, _ := event.Payload["userId"].(string)
	deviceID, _ := event.Payload["deviceId"].(string)
	if accessToken == "" || userID == "" || deviceID == "" {
		return nil, nil, nil, errors.New("session.bind requires accessToken, userId, and deviceId")
	}

	authSession, err := s.authService.AuthenticateAccessToken(context.Background(), accessToken)
	if err != nil {
		return nil, nil, nil, errors.New("access token is invalid or expired")
	}
	if authSession.UserID != userID {
		return nil, nil, nil, errors.New("session.bind userId does not match access token")
	}
	if !s.userOwnsActiveDevice(context.Background(), userID, deviceID) {
		return nil, nil, nil, errors.New("session.bind deviceId is invalid or revoked")
	}

	bound := &boundWSSession{
		userID:   userID,
		deviceID: deviceID,
		send:     make(chan wsServerEvent, 16),
	}
	s.wsHub.bind(userID, deviceID, bound)

	stream, cancel := s.messageService.SubscribeInbox(context.Background(), userID, deviceID)
	return bound, stream, cancel, nil
}

func (s *Server) userOwnsActiveDevice(ctx context.Context, userID, deviceID string) bool {
	devices, err := s.deviceService.List(ctx, userID)
	if err != nil {
		return false
	}

	for _, device := range devices {
		if device.ID == deviceID && device.RevokedAt == nil {
			return true
		}
	}

	return false
}

func ackStatusOrDefault(value any, fallback string) string {
	if status, ok := value.(string); ok && status != "" {
		return status
	}

	return fallback
}

func shouldClearPendingAck(status domain.DeliveryStatus) bool {
	return status == domain.DeliveryStatusRead || status == domain.DeliveryStatusFailed
}

func shouldDeliverToBoundDevice(item domain.DebugMessage, boundDeviceID string) bool {
	if item.RecipientDeviceID == "" {
		return true
	}

	return item.RecipientDeviceID == boundDeviceID
}

func wsAckContextFromMessage(item domain.DebugMessage) wsAckContext {
	return wsAckContext{
		MessageID:         item.ID,
		ConversationID:    item.ConversationID,
		SenderUserID:      item.SenderUserID,
		SenderDeviceID:    item.SenderDeviceID,
		RecipientUserID:   item.RecipientUserID,
		RecipientDeviceID: item.RecipientDeviceID,
		ClientMessageSeq:  item.ClientMessageSeq,
	}
}

func serializeDeliveryAck(context wsAckContext, recipientDeviceID, status string, fromMailbox bool, ackedAt time.Time) map[string]any {
	if recipientDeviceID == "" {
		recipientDeviceID = context.RecipientDeviceID
	}

	return map[string]any{
		"messageId":         context.MessageID,
		"conversationId":    context.ConversationID,
		"senderUserId":      context.SenderUserID,
		"senderDeviceId":    context.SenderDeviceID,
		"recipientUserId":   context.RecipientUserID,
		"recipientDeviceId": recipientDeviceID,
		"clientMessageSeq":  context.ClientMessageSeq,
		"status":            status,
		"ackedAt":           ackedAt.UTC().Format(time.RFC3339),
		"fromMailbox":       fromMailbox,
	}
}
