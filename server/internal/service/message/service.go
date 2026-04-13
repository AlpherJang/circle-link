package message

import (
	"context"

	"github.com/circle-link/circle-link/server/internal/domain"
)

type SendInput struct {
	MessageID         string
	ConversationID    string
	SenderUserID      string
	SenderDeviceID    string
	SenderEmail       string
	RecipientUserID   string
	RecipientDeviceID string
	RecipientEmail    string
	ContentType       string
	ClientMessageSeq  uint64
	Header            map[string]any
	RatchetPublicKey  string
	Ciphertext        string
	Body              string
	RecipientOnline   bool
}

type AcknowledgeInput struct {
	MessageID         string
	RecipientUserID   string
	RecipientDeviceID string
	Status            domain.DeliveryStatus
}

type Service interface {
	Send(ctx context.Context, input SendInput) (domain.DebugMessage, error)
	ListInbox(ctx context.Context, recipientUserID, recipientDeviceID string) ([]domain.DebugMessage, error)
	ListConversations(ctx context.Context, userID string) ([]domain.ConversationSummary, error)
	SubscribeInbox(ctx context.Context, recipientUserID, recipientDeviceID string) (<-chan domain.DebugMessage, func())
	Acknowledge(ctx context.Context, input AcknowledgeInput) (domain.DebugMessage, error)
}
