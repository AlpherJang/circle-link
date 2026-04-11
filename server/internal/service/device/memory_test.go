package device

import (
	"context"
	"testing"

	"github.com/circle-link/circle-link/server/internal/domain"
)

func TestMemoryServiceRegisterListAndRevoke(t *testing.T) {
	ctx := context.Background()
	service := NewMemoryService()

	deviceRecord, err := service.Register(ctx, "usr_123", RegisterInput{
		DeviceName: "Alice's iPhone",
		Platform:   domain.DevicePlatformIOS,
		PushToken:  "push-token",
		KeyBundle: domain.DeviceKeyBundle{
			IdentityKeyPublic:     "identity",
			SignedPrekeyPublic:    "signed",
			SignedPrekeySignature: "sig",
			SignedPrekeyVersion:   1,
		},
		OneTimePrekeys: []string{"k1", "k2"},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if deviceRecord.ID == "" {
		t.Fatal("expected device id")
	}

	items, err := service.List(ctx, "usr_123")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 device, got %d", len(items))
	}

	if err := service.Revoke(ctx, "usr_123", deviceRecord.ID); err != nil {
		t.Fatalf("revoke failed: %v", err)
	}
}
