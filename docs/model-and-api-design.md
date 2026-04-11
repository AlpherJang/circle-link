# circle-link Model And API Design

## 1. Purpose

This document complements `technical-design.md` with implementation-facing model and interface details:

- server-side domain models
- database entity shapes
- client-side domain models
- REST API contracts
- WebSocket event contracts
- shared error model

The goal is to reduce ambiguity before server and client scaffolding begins.

## 2. Design Conventions

### 2.1 ID Strategy

Recommended:

- use ULID or UUIDv7 for all primary IDs
- keep IDs as opaque strings in API contracts
- use server-generated IDs for users, devices, sessions, contacts, and mailbox rows
- use client-generated IDs for `message_id` and `conversation_id`

### 2.2 Time Strategy

- persist timestamps in UTC
- expose timestamps as ISO 8601 strings in REST JSON
- protobuf contracts may continue to use unix seconds or milliseconds until codegen conventions are finalized

### 2.3 API Versioning

- REST prefix: `/v1`
- WebSocket event schema version included in event envelope when needed later
- protobuf package versioning: `*.v1`

## 3. Domain Model Design

### 3.1 User

Represents the logical account owner.

Suggested fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | `string` | ULID or UUIDv7 |
| `email` | `string` | unique, normalized lowercase |
| `password_hash` | `string` | Argon2id output only |
| `display_name` | `string` | user-visible name |
| `status` | `string` | `active`, `pending_verification`, `disabled` |
| `email_verified_at` | `timestamp?` | null before verification |
| `created_at` | `timestamp` | server time |
| `updated_at` | `timestamp` | server time |

Rules:

- `email` must be unique
- login allowed only if account is not disabled
- message sending allowed only after email verification in v1

### 3.2 Device

Represents one physical or logical client installation.

Suggested fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | `string` | server-generated |
| `user_id` | `string` | owner account |
| `device_name` | `string` | example: `Zhanghao's iPhone` |
| `platform` | `string` | `ios`, `macos`, `android` |
| `push_token` | `string?` | APNs or FCM |
| `last_seen_at` | `timestamp?` | updated on connect |
| `revoked_at` | `timestamp?` | null when active |
| `created_at` | `timestamp` | server time |

Rules:

- revoked devices cannot open WebSocket sessions
- refresh tokens are bound to device sessions

### 3.3 DeviceKeyBundle

Represents public key material used for E2EE session setup.

Suggested fields:

| Field | Type | Notes |
| --- | --- | --- |
| `device_id` | `string` | one-to-one with device |
| `identity_key_public` | `base64` | long-term public key |
| `signed_prekey_public` | `base64` | rotated public prekey |
| `signed_prekey_signature` | `base64` | signature over signed prekey |
| `signed_prekey_version` | `int` | incrementing version |
| `one_time_prekey_count` | `int` | remaining stock |
| `updated_at` | `timestamp` | rotation tracking |

### 3.4 AuthSession

Represents a login session bound to one device.

Suggested fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | `string` | server-generated |
| `user_id` | `string` | owner |
| `device_id` | `string` | bound device |
| `refresh_token_hash` | `string` | hashed refresh token |
| `expires_at` | `timestamp` | refresh-token expiry |
| `revoked_at` | `timestamp?` | null when active |
| `created_at` | `timestamp` | server time |

### 3.5 Contact

Represents an allowlist relationship for private chat.

Suggested fields:

| Field | Type | Notes |
| --- | --- | --- |
| `owner_user_id` | `string` | contact owner |
| `peer_user_id` | `string` | target user |
| `state` | `string` | `pending`, `accepted`, `blocked` |
| `created_at` | `timestamp` | server time |

### 3.6 Conversation

`conversation_id` is mostly a client-owned logical identifier for a one-to-one thread.

Suggested derived model:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | `string` | client-generated stable ID |
| `owner_user_id` | `string` | local owner |
| `peer_user_id` | `string` | other user |
| `last_message_at` | `timestamp?` | local summary |
| `last_message_preview` | `string?` | local only, encrypted at rest |
| `unread_count` | `int` | local counter |
| `retention_default` | `string` | `persistent` by default |

The server does not need a permanent `conversations` table in v1 unless contact and mailbox queries later require one.

### 3.7 Message

Represents the client-side logical message record.

Suggested fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | `string` | client-generated |
| `conversation_id` | `string` | client-owned thread ID |
| `sender_user_id` | `string` | logical sender |
| `sender_device_id` | `string` | device sender |
| `recipient_user_id` | `string` | logical target |
| `recipient_device_id` | `string` | target device |
| `content_type` | `string` | `text/plain`, later attachment types |
| `ciphertext` | `bytes` | stored encrypted |
| `status` | `string` | `pending`, `sent`, `stored_offline`, `delivered`, `read`, `expired`, `failed` |
| `retention_mode` | `string` | `persistent`, `disappearing` |
| `disappear_after_seconds` | `int?` | null for persistent |
| `expires_at` | `timestamp?` | local computed deadline |
| `sent_at` | `timestamp` | sender local time normalized |
| `delivered_at` | `timestamp?` | receipt time |
| `read_at` | `timestamp?` | optional future field |

### 3.8 MailboxMessage

Represents a server-side offline-delivery item.

Suggested fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | `string` | mailbox row ID |
| `message_id` | `string` | original client message ID |
| `conversation_id` | `string` | logical thread ID |
| `sender_user_id` | `string` | sender |
| `sender_device_id` | `string` | sender device |
| `recipient_user_id` | `string` | recipient |
| `recipient_device_id` | `string` | recipient device |
| `content_type` | `string` | routing metadata only |
| `envelope_bytes` | `bytes` | opaque ciphertext envelope |
| `offline_eligible` | `bool` | must be true to persist |
| `retention_mode` | `string` | persistent or disappearing |
| `disappear_after_seconds` | `int?` | if present |
| `expires_at` | `timestamp` | mailbox expiry deadline |
| `delivered_at` | `timestamp?` | delivery completion |
| `created_at` | `timestamp` | insertion time |

Rules:

- only ciphertext envelope is stored
- row must be deleted on acknowledgement or expiry
- server must never expose mailbox rows to non-recipient devices without explicit policy

## 4. Client Model Design

### 4.1 Auth Models

Suggested Dart-side models:

- `AuthUser`
- `AuthTokens`
- `LoginFormState`
- `EmailVerificationState`
- `SessionSnapshot`

### 4.2 Chat Models

Suggested Dart-side models:

- `ConversationSummary`
- `MessageRecord`
- `OutgoingMessageDraft`
- `DeliveryReceipt`
- `DisappearingPolicy`
- `MailboxSyncState`

### 4.3 Security Models

Suggested Rust-bridge models:

- `IdentityKeyPair`
- `SignedPrekey`
- `OneTimePrekey`
- `PrekeyBundle`
- `EncryptedEnvelope`
- `DecryptedMessagePayload`

## 5. REST API Design

REST should use JSON request and response bodies. Authentication uses `Authorization: Bearer <access_token>` after login.

### 5.1 Common Response Envelope

Recommended response shape:

```json
{
  "data": {},
  "error": null,
  "requestId": "01JABC..."
}
```

Error shape:

```json
{
  "data": null,
  "error": {
    "code": "AUTH_INVALID_CREDENTIALS",
    "message": "Email or password is incorrect."
  },
  "requestId": "01JABC..."
}
```

### 5.2 Auth APIs

#### `POST /v1/auth/signup`

Request:

```json
{
  "email": "alice@example.com",
  "password": "strong-password",
  "displayName": "Alice"
}
```

Response:

```json
{
  "data": {
    "userId": "01JUSER...",
    "emailVerificationRequired": true
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

Current scaffold note:

- the in-memory bootstrap implementation also returns `verificationToken` in the signup response so local development can complete email verification before a real mail sender exists

Validation:

- email normalized and validated
- password length and strength policy enforced
- duplicate email rejected

#### `POST /v1/auth/verify-email`

Request:

```json
{
  "email": "alice@example.com",
  "verificationToken": "token-from-email"
}
```

Response:

```json
{
  "data": {
    "verified": true
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

#### `POST /v1/auth/login`

Request:

```json
{
  "email": "alice@example.com",
  "password": "strong-password"
}
```

Response:

```json
{
  "data": {
    "userId": "01JUSER...",
    "accessToken": "jwt-or-opaque-token",
    "refreshToken": "rotating-refresh-token",
    "expiresAt": "2026-04-11T12:00:00Z"
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

#### `POST /v1/auth/refresh`

Request:

```json
{
  "refreshToken": "rotating-refresh-token"
}
```

Response:

```json
{
  "data": {
    "accessToken": "new-access-token",
    "refreshToken": "new-refresh-token",
    "expiresAt": "2026-04-11T12:15:00Z"
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

#### `POST /v1/auth/logout`

Request:

```json
{
  "refreshToken": "rotating-refresh-token"
}
```

Response:

```json
{
  "data": {
    "success": true
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

#### `POST /v1/auth/change-password`

Request:

```json
{
  "currentPassword": "old-password",
  "newPassword": "new-password"
}
```

Response:

```json
{
  "data": {
    "success": true
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

### 5.3 Device APIs

#### `POST /v1/devices`

Request:

```json
{
  "deviceName": "Alice's iPhone",
  "platform": "ios",
  "pushToken": "apns-token",
  "keyBundle": {
    "identityKeyPublic": "base64",
    "signedPrekeyPublic": "base64",
    "signedPrekeySignature": "base64",
    "signedPrekeyVersion": 1,
    "oneTimePrekeys": ["base64", "base64"]
  }
}
```

Response:

```json
{
  "data": {
    "deviceId": "01JDEV...",
    "registeredAt": "2026-04-11T08:00:00Z"
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

#### `GET /v1/devices`

Response:

```json
{
  "data": {
    "items": [
      {
        "deviceId": "01JDEV...",
        "deviceName": "Alice's iPhone",
        "platform": "ios",
        "lastSeenAt": "2026-04-11T08:03:00Z",
        "revokedAt": null
      }
    ]
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

#### `DELETE /v1/devices/{deviceId}`

Response:

```json
{
  "data": {
    "success": true
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

### 5.4 Key APIs

#### `GET /v1/users/{userId}/prekey-bundle?deviceId={deviceId}`

Response:

```json
{
  "data": {
    "userId": "01JUSER...",
    "deviceId": "01JDEV...",
    "identityKeyPublic": "base64",
    "signedPrekeyPublic": "base64",
    "signedPrekeySignature": "base64",
    "signedPrekeyVersion": 3,
    "oneTimePrekey": "base64"
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

#### `POST /v1/devices/{deviceId}/prekeys`

Request:

```json
{
  "signedPrekeyPublic": "base64",
  "signedPrekeySignature": "base64",
  "signedPrekeyVersion": 4,
  "oneTimePrekeys": ["base64", "base64", "base64"]
}
```

### 5.5 Contact APIs

#### `GET /v1/contacts`

Response:

```json
{
  "data": {
    "items": [
      {
        "peerUserId": "01JPEER...",
        "displayName": "Bob",
        "state": "accepted"
      }
    ]
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

#### `POST /v1/contacts/invite`

Request:

```json
{
  "peerEmail": "bob@example.com"
}
```

Response:

```json
{
  "data": {
    "state": "pending"
  },
  "error": null,
  "requestId": "01JREQ..."
}
```

## 6. WebSocket Interface Design

WebSocket is used for authenticated real-time messaging and mailbox recovery.

### 6.1 Connection Rules

- connect over `wss://.../v1/ws`
- client sends `session.bind` as first event
- server closes connection if bind does not arrive within timeout
- one device may have one active primary relay session in v1

### 6.2 Event Envelope

Recommended JSON debugging shape:

```json
{
  "type": "message.send",
  "traceId": "01JTRACE...",
  "payload": {}
}
```

Production transport can remain protobuf-based over WebSocket binary frames.

### 6.3 Client-To-Server Events

#### `session.bind`

```json
{
  "type": "session.bind",
  "payload": {
    "accessToken": "jwt-or-opaque-token",
    "userId": "01JUSER...",
    "deviceId": "01JDEV..."
  }
}
```

#### `message.send`

```json
{
  "type": "message.send",
  "payload": {
    "messageId": "01JMSG...",
    "conversationId": "01JCONV...",
    "senderUserId": "01JUSER...",
    "senderDeviceId": "01JDEV1...",
    "recipientUserId": "01JPEER...",
    "recipientDeviceId": "01JDEV2...",
    "contentType": "text/plain",
    "offlineEligible": true,
    "retentionMode": "disappearing",
    "disappearAfterSeconds": 300,
    "expiresAt": "2026-04-11T12:30:00Z",
    "header": "base64",
    "ratchetPublicKey": "base64",
    "ciphertext": "base64"
  }
}
```

#### `message.ack`

```json
{
  "type": "message.ack",
  "payload": {
    "messageId": "01JMSG...",
    "recipientDeviceId": "01JDEV2...",
    "status": "delivered",
    "ackedAt": "2026-04-11T08:05:00Z",
    "fromMailbox": true
  }
}
```

### 6.4 Server-To-Client Events

#### `presence.snapshot`

```json
{
  "type": "presence.snapshot",
  "payload": {
    "onlineUserIds": ["01JUSER..."],
    "onlineDeviceIds": ["01JDEV..."]
  }
}
```

#### `message.deliver`

Used for directly relayed online messages.

#### `message.mailbox`

Used for messages loaded from offline mailbox on reconnect.

Payload shape is the same as `message.send`.

#### `delivery.ack`

```json
{
  "type": "delivery.ack",
  "payload": {
    "messageId": "01JMSG...",
    "recipientDeviceId": "01JDEV2...",
    "status": "accepted",
    "ackedAt": "2026-04-11T08:04:59Z",
    "fromMailbox": false
  }
}
```

#### `message.nack`

```json
{
  "type": "message.nack",
  "payload": {
    "messageId": "01JMSG...",
    "reasonCode": "MAILBOX_QUOTA_EXCEEDED",
    "reasonMessage": "Recipient mailbox is full."
  }
}
```

#### `mailbox.drained`

```json
{
  "type": "mailbox.drained",
  "payload": {
    "deviceId": "01JDEV2...",
    "pendingCount": 0
  }
}
```

## 7. Error Code Design

Recommended error-code families:

- `AUTH_*`
- `DEVICE_*`
- `KEY_*`
- `CONTACT_*`
- `RELAY_*`
- `MAILBOX_*`
- `RATE_LIMIT_*`
- `VALIDATION_*`

Suggested initial codes:

| Code | Meaning |
| --- | --- |
| `AUTH_INVALID_CREDENTIALS` | wrong email or password |
| `AUTH_EMAIL_NOT_VERIFIED` | email verification required |
| `AUTH_SESSION_EXPIRED` | refresh or login required |
| `DEVICE_REVOKED` | device can no longer connect |
| `KEY_BUNDLE_EXHAUSTED` | recipient has no usable one-time prekey |
| `MAILBOX_QUOTA_EXCEEDED` | recipient mailbox full |
| `MAILBOX_ITEM_EXPIRED` | offline item expired before delivery |
| `RELAY_TARGET_UNAVAILABLE` | no route and mailbox disabled |
| `VALIDATION_FAILED` | malformed input |
| `RATE_LIMIT_EXCEEDED` | too many requests |

## 8. Suggested Server Interfaces

These are implementation-level interface suggestions for Go.

### 8.1 AuthService

```go
type AuthService interface {
    SignUp(ctx context.Context, input SignUpInput) (SignUpResult, error)
    VerifyEmail(ctx context.Context, input VerifyEmailInput) (VerifyEmailResult, error)
    Login(ctx context.Context, input LoginInput) (LoginResult, error)
    RefreshSession(ctx context.Context, input RefreshSessionInput) (RefreshSessionResult, error)
    Logout(ctx context.Context, input LogoutInput) error
    ChangePassword(ctx context.Context, input ChangePasswordInput) error
}
```

### 8.2 DeviceService

```go
type DeviceService interface {
    RegisterDevice(ctx context.Context, userID string, input RegisterDeviceInput) (Device, error)
    ListDevices(ctx context.Context, userID string) ([]Device, error)
    RevokeDevice(ctx context.Context, userID, deviceID string) error
    UpdatePushToken(ctx context.Context, userID, deviceID string, pushToken string) error
}
```

### 8.3 MailboxService

```go
type MailboxService interface {
    Store(ctx context.Context, item MailboxMessage) error
    ListPending(ctx context.Context, recipientDeviceID string, limit int) ([]MailboxMessage, error)
    AckDelivered(ctx context.Context, messageID string, recipientDeviceID string, ackedAt time.Time) error
    DeleteExpired(ctx context.Context, now time.Time, limit int) (int, error)
}
```

### 8.4 RelayService

```go
type RelayService interface {
    BindSession(ctx context.Context, connID string, userID string, deviceID string) error
    SendMessage(ctx context.Context, env MessageEnvelope) (DeliveryDisposition, error)
    AckMessage(ctx context.Context, ack DeliveryAck) error
}
```

## 9. Suggested Client Repository Interfaces

These are implementation-level interface suggestions for Flutter.

### 9.1 AuthRepository

```dart
abstract interface class AuthRepository {
  Future<AuthUser> signUp(SignUpCommand command);
  Future<void> verifyEmail(VerifyEmailCommand command);
  Future<AuthSession> login(LoginCommand command);
  Future<AuthSession> refresh(String refreshToken);
  Future<void> logout(String refreshToken);
  Future<void> changePassword(ChangePasswordCommand command);
}
```

### 9.2 ChatRepository

```dart
abstract interface class ChatRepository {
  Future<void> sendMessage(OutgoingMessageDraft draft);
  Stream<MessageRecord> watchConversation(String conversationId);
  Future<void> acknowledgeDelivery(DeliveryReceipt receipt);
  Future<void> applyMailboxBatch(List<MessageRecord> items);
  Future<void> expireMessage(String messageId);
}
```

### 9.3 CryptoBridge

```dart
abstract interface class CryptoBridge {
  Future<PrekeyBundle> exportPrekeyBundle();
  Future<EncryptedEnvelope> encryptMessage(EncryptCommand command);
  Future<DecryptedMessagePayload> decryptMessage(EncryptedEnvelope envelope);
}
```

## 10. Recommended Next Implementation Step

With this document in place, the cleanest next step is:

1. define canonical JSON DTOs and protobuf bindings from these shapes
2. scaffold Go service interfaces and DTO structs
3. scaffold Flutter repositories and state models
4. keep field names identical across docs, DTOs, and proto where possible
