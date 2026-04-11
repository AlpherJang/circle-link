package relay

import (
	"context"

	"github.com/circle-link/circle-link/server/internal/domain"
)

type DeliveryDisposition string

const (
	DeliveryDispositionDirect        DeliveryDisposition = "direct"
	DeliveryDispositionStoredOffline DeliveryDisposition = "stored_offline"
)

type SendResult struct {
	Disposition DeliveryDisposition
}

type Service interface {
	BindSession(ctx context.Context, connectionID, userID, deviceID string) error
	SendMessage(ctx context.Context, envelope domain.MessageEnvelope) (SendResult, error)
	AckMessage(ctx context.Context, ack domain.DeliveryAck) error
}
