package domain

import "time"

type UserStatus string

const (
	UserStatusPendingVerification UserStatus = "pending_verification"
	UserStatusActive              UserStatus = "active"
	UserStatusDisabled            UserStatus = "disabled"
)

type DevicePlatform string

const (
	DevicePlatformIOS     DevicePlatform = "ios"
	DevicePlatformMacOS   DevicePlatform = "macos"
	DevicePlatformAndroid DevicePlatform = "android"
)

type ContactState string

const (
	ContactStatePending  ContactState = "pending"
	ContactStateAccepted ContactState = "accepted"
	ContactStateBlocked  ContactState = "blocked"
)

type RetentionMode string

const (
	RetentionModePersistent   RetentionMode = "persistent"
	RetentionModeDisappearing RetentionMode = "disappearing"
)

type DeliveryStatus string

const (
	DeliveryStatusAccepted      DeliveryStatus = "accepted"
	DeliveryStatusStoredOffline DeliveryStatus = "stored_offline"
	DeliveryStatusDelivered     DeliveryStatus = "delivered"
	DeliveryStatusRead          DeliveryStatus = "read"
	DeliveryStatusFailed        DeliveryStatus = "failed"
)

type User struct {
	ID              string
	Email           string
	PasswordHash    string
	DisplayName     string
	Status          UserStatus
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Device struct {
	ID         string
	UserID     string
	DeviceName string
	Platform   DevicePlatform
	PushToken  string
	LastSeenAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

type DeviceKeyBundle struct {
	DeviceID              string
	IdentityKeyPublic     string
	SignedPrekeyPublic    string
	SignedPrekeySignature string
	SignedPrekeyVersion   int
	OneTimePrekeyCount    int
	UpdatedAt             time.Time
}

type AuthSession struct {
	ID               string
	UserID           string
	DeviceID         string
	RefreshTokenHash string
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
}

type Contact struct {
	OwnerUserID     string
	PeerUserID      string
	State           ContactState
	InvitedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ConversationSummary struct {
	ConversationID       string
	LastMessageID        string
	PeerUserID           string
	PeerEmail            string
	PeerDisplayName      string
	LastMessagePreview   string
	LastMessageAt        time.Time
	UnreadCount          int
	MessageCount         int
	LatestDeliveryStatus DeliveryStatus
}

type MessageEnvelope struct {
	MessageID             string
	ConversationID        string
	SenderUserID          string
	SenderDeviceID        string
	RecipientUserID       string
	RecipientDeviceID     string
	ContentType           string
	Header                []byte
	RatchetPublicKey      []byte
	Ciphertext            []byte
	ClientMessageSeq      uint64
	OfflineEligible       bool
	RetentionMode         RetentionMode
	DisappearAfterSeconds uint64
	ExpiresAt             *time.Time
	SentAt                time.Time
}

type MailboxMessage struct {
	ID                    string
	MessageID             string
	ConversationID        string
	SenderUserID          string
	SenderDeviceID        string
	RecipientUserID       string
	RecipientDeviceID     string
	ContentType           string
	EnvelopeBytes         []byte
	OfflineEligible       bool
	RetentionMode         RetentionMode
	DisappearAfterSeconds uint64
	ExpiresAt             time.Time
	CreatedAt             time.Time
	DeliveredAt           *time.Time
}

type DeliveryAck struct {
	MessageID         string
	RecipientDeviceID string
	Status            DeliveryStatus
	AckedAt           time.Time
	FromMailbox       bool
}

type DebugMessage struct {
	ID                string
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
	DeliveryStatus    DeliveryStatus
	StoredAt          time.Time
	DeliveredAt       *time.Time
	ReadAt            *time.Time
	SentAt            time.Time
}
