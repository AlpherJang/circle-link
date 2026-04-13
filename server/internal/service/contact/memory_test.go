package contact

import (
	"context"
	"testing"

	"github.com/circle-link/circle-link/server/internal/domain"
)

func TestMemoryServiceInviteCreatesPendingContacts(t *testing.T) {
	service := NewMemoryService()
	ctx := context.Background()

	contact, err := service.Invite(ctx, "usr_a", "usr_b")
	if err != nil {
		t.Fatalf("invite failed: %v", err)
	}
	if contact.State != domain.ContactStatePending {
		t.Fatalf("expected pending state, got %q", contact.State)
	}
	if contact.InvitedByUserID != "usr_a" {
		t.Fatalf("expected invitedBy usr_a, got %q", contact.InvitedByUserID)
	}

	ownerContacts, err := service.List(ctx, "usr_a")
	if err != nil {
		t.Fatalf("list owner contacts failed: %v", err)
	}
	if len(ownerContacts) != 1 || ownerContacts[0].PeerUserID != "usr_b" {
		t.Fatalf("unexpected owner contacts: %#v", ownerContacts)
	}

	peerContacts, err := service.List(ctx, "usr_b")
	if err != nil {
		t.Fatalf("list peer contacts failed: %v", err)
	}
	if len(peerContacts) != 1 || peerContacts[0].PeerUserID != "usr_a" {
		t.Fatalf("unexpected peer contacts: %#v", peerContacts)
	}
	if peerContacts[0].State != domain.ContactStatePending {
		t.Fatalf("expected peer state pending, got %q", peerContacts[0].State)
	}
	if peerContacts[0].InvitedByUserID != "usr_a" {
		t.Fatalf("expected peer invitedBy usr_a, got %q", peerContacts[0].InvitedByUserID)
	}
}

func TestMemoryServiceInviteRejectsSelfContact(t *testing.T) {
	service := NewMemoryService()

	_, err := service.Invite(context.Background(), "usr_a", "usr_a")
	if err != ErrInvalidPeer {
		t.Fatalf("expected ErrInvalidPeer, got %v", err)
	}
}

func TestMemoryServiceAcceptPromotesBothSides(t *testing.T) {
	service := NewMemoryService()
	ctx := context.Background()

	if _, err := service.Invite(ctx, "usr_a", "usr_b"); err != nil {
		t.Fatalf("invite failed: %v", err)
	}

	accepted, err := service.Accept(ctx, "usr_b", "usr_a")
	if err != nil {
		t.Fatalf("accept failed: %v", err)
	}
	if accepted.State != domain.ContactStateAccepted {
		t.Fatalf("expected accepted state, got %q", accepted.State)
	}

	aliceContacts, _ := service.List(ctx, "usr_a")
	if aliceContacts[0].State != domain.ContactStateAccepted {
		t.Fatalf("expected alice contact accepted, got %q", aliceContacts[0].State)
	}
}

func TestMemoryServiceRejectRemovesPendingContacts(t *testing.T) {
	service := NewMemoryService()
	ctx := context.Background()

	if _, err := service.Invite(ctx, "usr_a", "usr_b"); err != nil {
		t.Fatalf("invite failed: %v", err)
	}

	if err := service.Reject(ctx, "usr_b", "usr_a"); err != nil {
		t.Fatalf("reject failed: %v", err)
	}

	aliceContacts, _ := service.List(ctx, "usr_a")
	if len(aliceContacts) != 0 {
		t.Fatalf("expected alice contacts to be empty, got %#v", aliceContacts)
	}
	bobContacts, _ := service.List(ctx, "usr_b")
	if len(bobContacts) != 0 {
		t.Fatalf("expected bob contacts to be empty, got %#v", bobContacts)
	}
}
