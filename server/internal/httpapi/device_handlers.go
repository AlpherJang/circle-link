package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/circle-link/circle-link/server/internal/domain"
	"github.com/circle-link/circle-link/server/internal/service/device"
)

type registerDeviceRequest struct {
	DeviceName string `json:"deviceName"`
	Platform   string `json:"platform"`
	PushToken  string `json:"pushToken"`
	KeyBundle  struct {
		IdentityKeyPublic     string   `json:"identityKeyPublic"`
		SignedPrekeyPublic    string   `json:"signedPrekeyPublic"`
		SignedPrekeySignature string   `json:"signedPrekeySignature"`
		SignedPrekeyVersion   int      `json:"signedPrekeyVersion"`
		OneTimePrekeys        []string `json:"oneTimePrekeys"`
	} `json:"keyBundle"`
}

func (s *Server) handleRegisterDevice(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	var req registerDeviceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON request body.")
		return
	}

	platform, valid := parseDevicePlatform(req.Platform)
	if !valid {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Platform must be ios, macos, or android.")
		return
	}

	deviceRecord, err := s.deviceService.Register(r.Context(), session.UserID, device.RegisterInput{
		DeviceName: req.DeviceName,
		Platform:   platform,
		PushToken:  req.PushToken,
		KeyBundle: domain.DeviceKeyBundle{
			IdentityKeyPublic:     req.KeyBundle.IdentityKeyPublic,
			SignedPrekeyPublic:    req.KeyBundle.SignedPrekeyPublic,
			SignedPrekeySignature: req.KeyBundle.SignedPrekeySignature,
			SignedPrekeyVersion:   req.KeyBundle.SignedPrekeyVersion,
		},
		OneTimePrekeys: req.KeyBundle.OneTimePrekeys,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to register device.")
		return
	}

	writeData(w, http.StatusCreated, map[string]any{
		"deviceId":     deviceRecord.ID,
		"registeredAt": deviceRecord.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	items, err := s.deviceService.List(r.Context(), session.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list devices.")
		return
	}

	responseItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, map[string]any{
			"deviceId":   item.ID,
			"deviceName": item.DeviceName,
			"platform":   item.Platform,
			"lastSeenAt": formatTimePtr(item.LastSeenAt),
			"revokedAt":  formatTimePtr(item.RevokedAt),
		})
	}

	writeData(w, http.StatusOK, map[string]any{
		"items": responseItems,
	})
}

func (s *Server) handleRevokeDevice(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	deviceID := r.PathValue("deviceId")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Device id is required.")
		return
	}

	if err := s.deviceService.Revoke(r.Context(), session.UserID, deviceID); err != nil {
		switch {
		case errors.Is(err, device.ErrDeviceNotFound):
			writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Device not found.")
		case errors.Is(err, device.ErrDeviceRevoked):
			writeError(w, http.StatusConflict, "DEVICE_REVOKED", "Device is already revoked.")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke device.")
		}
		return
	}

	writeData(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func parseDevicePlatform(raw string) (domain.DevicePlatform, bool) {
	switch raw {
	case string(domain.DevicePlatformIOS):
		return domain.DevicePlatformIOS, true
	case string(domain.DevicePlatformMacOS):
		return domain.DevicePlatformMacOS, true
	case string(domain.DevicePlatformAndroid):
		return domain.DevicePlatformAndroid, true
	default:
		return "", false
	}
}

func formatTimePtr(value *time.Time) any {
	if value == nil {
		return nil
	}

	return value.UTC().Format(time.RFC3339)
}
