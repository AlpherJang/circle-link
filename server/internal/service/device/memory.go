package device

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/circle-link/circle-link/server/internal/domain"
	"github.com/circle-link/circle-link/server/internal/platform/ids"
)

var (
	ErrDeviceNotFound = errors.New("device not found")
	ErrDeviceRevoked  = errors.New("device revoked")
)

type MemoryService struct {
	mu             sync.RWMutex
	devicesByID    map[string]domain.Device
	deviceIDsByUID map[string][]string
	keyBundles     map[string]domain.DeviceKeyBundle
}

func NewMemoryService() *MemoryService {
	return &MemoryService{
		devicesByID:    make(map[string]domain.Device),
		deviceIDsByUID: make(map[string][]string),
		keyBundles:     make(map[string]domain.DeviceKeyBundle),
	}
}

func (s *MemoryService) Register(_ context.Context, userID string, input RegisterInput) (domain.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	deviceID := ids.New("dev")
	device := domain.Device{
		ID:         deviceID,
		UserID:     userID,
		DeviceName: input.DeviceName,
		Platform:   input.Platform,
		PushToken:  input.PushToken,
		LastSeenAt: &now,
		CreatedAt:  now,
	}
	s.devicesByID[deviceID] = device
	s.deviceIDsByUID[userID] = append(s.deviceIDsByUID[userID], deviceID)

	keyBundle := input.KeyBundle
	keyBundle.DeviceID = deviceID
	keyBundle.OneTimePrekeyCount = len(input.OneTimePrekeys)
	keyBundle.UpdatedAt = now
	s.keyBundles[deviceID] = keyBundle

	return device, nil
}

func (s *MemoryService) List(_ context.Context, userID string) ([]domain.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceIDs := s.deviceIDsByUID[userID]
	devices := make([]domain.Device, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		devices = append(devices, s.devicesByID[deviceID])
	}

	return devices, nil
}

func (s *MemoryService) Revoke(_ context.Context, userID, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devicesByID[deviceID]
	if !ok || device.UserID != userID {
		return ErrDeviceNotFound
	}
	if device.RevokedAt != nil {
		return ErrDeviceRevoked
	}

	now := time.Now().UTC()
	device.RevokedAt = &now
	s.devicesByID[deviceID] = device
	return nil
}

func (s *MemoryService) UpdatePushToken(_ context.Context, userID, deviceID, pushToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devicesByID[deviceID]
	if !ok || device.UserID != userID {
		return ErrDeviceNotFound
	}
	if device.RevokedAt != nil {
		return ErrDeviceRevoked
	}

	device.PushToken = pushToken
	s.devicesByID[deviceID] = device
	return nil
}
