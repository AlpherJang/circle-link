package contact

import (
	"context"
	"errors"

	"github.com/circle-link/circle-link/server/internal/domain"
)

var (
	ErrContactAlreadyExists = errors.New("contact already exists")
	ErrInvalidPeer          = errors.New("peer user is invalid")
	ErrContactNotFound      = errors.New("contact was not found")
	ErrContactNotPending    = errors.New("contact invite is not pending")
	ErrContactNotIncoming   = errors.New("contact invite is not incoming")
)

type Service interface {
	Invite(ctx context.Context, ownerUserID, peerUserID string) (domain.Contact, error)
	Accept(ctx context.Context, ownerUserID, peerUserID string) (domain.Contact, error)
	Reject(ctx context.Context, ownerUserID, peerUserID string) error
	List(ctx context.Context, ownerUserID string) ([]domain.Contact, error)
}
