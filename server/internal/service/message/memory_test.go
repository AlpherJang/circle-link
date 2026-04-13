package message

import (
	"context"
	"testing"
	"time"

	"github.com/circle-link/circle-link/server/internal/domain"
)

func TestMemoryServiceSendAndListInbox(t *testing.T) {
	service := NewMemoryService()
	ctx := context.Background()

	_, err := service.Send(ctx, SendInput{
		MessageID:         "msg_1",
		ConversationID:    "conv_1",
		SenderUserID:      "usr_a",
		SenderDeviceID:    "dev_a",
		SenderEmail:       "a@example.com",
		RecipientUserID:   "usr_b",
		RecipientDeviceID: "dev_b",
		RecipientEmail:    "b@example.com",
		ContentType:       "text/plain",
		ClientMessageSeq:  1,
		Header:            map[string]any{"encoding": "debug-base64-utf8"},
		RatchetPublicKey:  "debug-rpk-1",
		Ciphertext:        "aGVsbG8=",
		Body:              "hello",
		RecipientOnline:   false,
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	items, err := service.ListInbox(ctx, "usr_b", "dev_b")
	if err != nil {
		t.Fatalf("list inbox failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Body != "hello" {
		t.Fatalf("expected body hello, got %q", items[0].Body)
	}
	if items[0].Ciphertext != "aGVsbG8=" {
		t.Fatalf("expected ciphertext aGVsbG8=, got %q", items[0].Ciphertext)
	}
	if items[0].RatchetPublicKey != "debug-rpk-1" {
		t.Fatalf("expected ratchet public key debug-rpk-1, got %q", items[0].RatchetPublicKey)
	}
	if items[0].DeliveryStatus != domain.DeliveryStatusStoredOffline {
		t.Fatalf("expected delivery status stored_offline, got %q", items[0].DeliveryStatus)
	}
	if items[0].ID != "msg_1" {
		t.Fatalf("expected message id msg_1, got %q", items[0].ID)
	}
	if items[0].ConversationID != "conv_1" {
		t.Fatalf("expected conversation id conv_1, got %q", items[0].ConversationID)
	}
}

func TestMemoryServiceSubscribeInbox(t *testing.T) {
	service := NewMemoryService()
	ctx := context.Background()

	stream, cancel := service.SubscribeInbox(ctx, "usr_b", "dev_b")
	defer cancel()

	_, err := service.Send(ctx, SendInput{
		MessageID:         "msg_live",
		ConversationID:    "conv_live",
		SenderUserID:      "usr_a",
		SenderDeviceID:    "dev_a",
		SenderEmail:       "a@example.com",
		RecipientUserID:   "usr_b",
		RecipientDeviceID: "dev_b",
		RecipientEmail:    "b@example.com",
		ContentType:       "text/plain",
		ClientMessageSeq:  2,
		Header:            map[string]any{"encoding": "debug-base64-utf8"},
		RatchetPublicKey:  "debug-rpk-live",
		Ciphertext:        "bGl2ZSBoZWxsbw==",
		Body:              "live hello",
		RecipientOnline:   true,
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	select {
	case item := <-stream:
		if item.Body != "live hello" {
			t.Fatalf("expected body live hello, got %q", item.Body)
		}
		if item.DeliveryStatus != domain.DeliveryStatusAccepted {
			t.Fatalf("expected accepted delivery status, got %q", item.DeliveryStatus)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscription event")
	}
}

func TestMemoryServiceAcknowledgeClaimsGenericMailboxMessage(t *testing.T) {
	service := NewMemoryService()
	ctx := context.Background()

	_, err := service.Send(ctx, SendInput{
		MessageID:        "msg_claim",
		ConversationID:   "conv_claim",
		SenderUserID:     "usr_a",
		SenderDeviceID:   "dev_a",
		SenderEmail:      "a@example.com",
		RecipientUserID:  "usr_b",
		RecipientEmail:   "b@example.com",
		ContentType:      "text/plain",
		ClientMessageSeq: 3,
		Body:             "claim me",
		RecipientOnline:  false,
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	item, err := service.Acknowledge(ctx, AcknowledgeInput{
		MessageID:         "msg_claim",
		RecipientUserID:   "usr_b",
		RecipientDeviceID: "dev_b_phone",
		Status:            domain.DeliveryStatusDelivered,
	})
	if err != nil {
		t.Fatalf("acknowledge failed: %v", err)
	}
	if item.RecipientDeviceID != "dev_b_phone" {
		t.Fatalf("expected claimed recipient device dev_b_phone, got %q", item.RecipientDeviceID)
	}
	if item.DeliveryStatus != domain.DeliveryStatusDelivered {
		t.Fatalf("expected delivered status, got %q", item.DeliveryStatus)
	}
	if item.DeliveredAt == nil {
		t.Fatal("expected delivered timestamp to be set")
	}

	phoneItems, err := service.ListInbox(ctx, "usr_b", "dev_b_phone")
	if err != nil {
		t.Fatalf("list phone inbox failed: %v", err)
	}
	if len(phoneItems) != 1 {
		t.Fatalf("expected 1 phone inbox item, got %d", len(phoneItems))
	}

	tabletItems, err := service.ListInbox(ctx, "usr_b", "dev_b_tablet")
	if err != nil {
		t.Fatalf("list tablet inbox failed: %v", err)
	}
	if len(tabletItems) != 0 {
		t.Fatalf("expected 0 tablet inbox items after claim, got %d", len(tabletItems))
	}
}

func TestMemoryServiceAcknowledgeReadSetsReadAndDeliveredTimestamps(t *testing.T) {
	service := NewMemoryService()
	ctx := context.Background()

	_, err := service.Send(ctx, SendInput{
		MessageID:         "msg_read",
		ConversationID:    "conv_read",
		SenderUserID:      "usr_a",
		SenderDeviceID:    "dev_a",
		SenderEmail:       "a@example.com",
		RecipientUserID:   "usr_b",
		RecipientDeviceID: "dev_b",
		RecipientEmail:    "b@example.com",
		ContentType:       "text/plain",
		ClientMessageSeq:  4,
		Body:              "read me",
		RecipientOnline:   true,
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	item, err := service.Acknowledge(ctx, AcknowledgeInput{
		MessageID:         "msg_read",
		RecipientUserID:   "usr_b",
		RecipientDeviceID: "dev_b",
		Status:            domain.DeliveryStatusRead,
	})
	if err != nil {
		t.Fatalf("acknowledge read failed: %v", err)
	}
	if item.DeliveryStatus != domain.DeliveryStatusRead {
		t.Fatalf("expected read status, got %q", item.DeliveryStatus)
	}
	if item.DeliveredAt == nil {
		t.Fatal("expected delivered timestamp when marking as read")
	}
	if item.ReadAt == nil {
		t.Fatal("expected read timestamp when marking as read")
	}
}

func TestMemoryServiceListConversations(t *testing.T) {
	service := NewMemoryService()
	ctx := context.Background()

	_, err := service.Send(ctx, SendInput{
		MessageID:         "msg_conv_1",
		ConversationID:    "conv_alpha",
		SenderUserID:      "usr_a",
		SenderDeviceID:    "dev_a",
		SenderEmail:       "alice@example.com",
		RecipientUserID:   "usr_b",
		RecipientDeviceID: "dev_b",
		RecipientEmail:    "bob@example.com",
		ContentType:       "text/plain",
		ClientMessageSeq:  1,
		Body:              "hello bob",
		RecipientOnline:   true,
	})
	if err != nil {
		t.Fatalf("send alpha failed: %v", err)
	}

	_, err = service.Send(ctx, SendInput{
		MessageID:         "msg_conv_2",
		ConversationID:    "conv_beta",
		SenderUserID:      "usr_c",
		SenderDeviceID:    "dev_c",
		SenderEmail:       "charlie@example.com",
		RecipientUserID:   "usr_a",
		RecipientDeviceID: "dev_a",
		RecipientEmail:    "alice@example.com",
		ContentType:       "text/plain",
		ClientMessageSeq:  2,
		Body:              "hello alice",
		RecipientOnline:   false,
	})
	if err != nil {
		t.Fatalf("send beta failed: %v", err)
	}

	conversations, err := service.ListConversations(ctx, "usr_a")
	if err != nil {
		t.Fatalf("list conversations failed: %v", err)
	}
	if len(conversations) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(conversations))
	}
	if conversations[0].ConversationID != "conv_beta" {
		t.Fatalf("expected latest conversation conv_beta, got %q", conversations[0].ConversationID)
	}
	if conversations[0].PeerEmail != "charlie@example.com" {
		t.Fatalf("expected latest peer charlie@example.com, got %q", conversations[0].PeerEmail)
	}
	if conversations[0].UnreadCount != 1 {
		t.Fatalf("expected latest unreadCount 1, got %d", conversations[0].UnreadCount)
	}
	if conversations[1].PeerEmail != "bob@example.com" {
		t.Fatalf("expected second peer bob@example.com, got %q", conversations[1].PeerEmail)
	}
	if conversations[1].MessageCount != 1 {
		t.Fatalf("expected second conversation messageCount 1, got %d", conversations[1].MessageCount)
	}
}
