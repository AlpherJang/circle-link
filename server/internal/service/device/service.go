package device

import (
	"context"

	"github.com/circle-link/circle-link/server/internal/domain"
)

type RegisterInput struct {
	DeviceName     string
	Platform       domain.DevicePlatform
	PushToken      string
	KeyBundle      domain.DeviceKeyBundle
	OneTimePrekeys []string
}

type Service interface {
	Register(ctx context.Context, userID string, input RegisterInput) (domain.Device, error)
	List(ctx context.Context, userID string) ([]domain.Device, error)
	Revoke(ctx context.Context, userID, deviceID string) error
	UpdatePushToken(ctx context.Context, userID, deviceID, pushToken string) error
}
