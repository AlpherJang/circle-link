package mailbox

import (
	"context"
	"time"

	"github.com/circle-link/circle-link/server/internal/domain"
)

type Service interface {
	Store(ctx context.Context, item domain.MailboxMessage) error
	ListPending(ctx context.Context, recipientDeviceID string, limit int) ([]domain.MailboxMessage, error)
	AckDelivered(ctx context.Context, messageID, recipientDeviceID string, ackedAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time, limit int) (int, error)
}
