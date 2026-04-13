package contact

import (
	"context"
	"sync"
	"time"

	"github.com/circle-link/circle-link/server/internal/domain"
)

type MemoryService struct {
	mu             sync.RWMutex
	contactsByUser map[string]map[string]domain.Contact
}

func NewMemoryService() *MemoryService {
	return &MemoryService{
		contactsByUser: make(map[string]map[string]domain.Contact),
	}
}

func (s *MemoryService) Invite(_ context.Context, ownerUserID, peerUserID string) (domain.Contact, error) {
	if ownerUserID == "" || peerUserID == "" || ownerUserID == peerUserID {
		return domain.Contact{}, ErrInvalidPeer
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if contact, ok := s.contactsByUser[ownerUserID][peerUserID]; ok {
		return contact, nil
	}

	now := time.Now().UTC()
	ownerContact := domain.Contact{
		OwnerUserID:     ownerUserID,
		PeerUserID:      peerUserID,
		State:           domain.ContactStatePending,
		InvitedByUserID: ownerUserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	peerContact := domain.Contact{
		OwnerUserID:     peerUserID,
		PeerUserID:      ownerUserID,
		State:           domain.ContactStatePending,
		InvitedByUserID: ownerUserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if s.contactsByUser[ownerUserID] == nil {
		s.contactsByUser[ownerUserID] = make(map[string]domain.Contact)
	}
	if s.contactsByUser[peerUserID] == nil {
		s.contactsByUser[peerUserID] = make(map[string]domain.Contact)
	}

	s.contactsByUser[ownerUserID][peerUserID] = ownerContact
	s.contactsByUser[peerUserID][ownerUserID] = peerContact

	return ownerContact, nil
}

func (s *MemoryService) Accept(_ context.Context, ownerUserID, peerUserID string) (domain.Contact, error) {
	if ownerUserID == "" || peerUserID == "" || ownerUserID == peerUserID {
		return domain.Contact{}, ErrInvalidPeer
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ownerContacts := s.contactsByUser[ownerUserID]
	peerContacts := s.contactsByUser[peerUserID]
	ownerContact, ok := ownerContacts[peerUserID]
	if !ok {
		return domain.Contact{}, ErrContactNotFound
	}
	peerContact, ok := peerContacts[ownerUserID]
	if !ok {
		return domain.Contact{}, ErrContactNotFound
	}
	if ownerContact.State != domain.ContactStatePending || peerContact.State != domain.ContactStatePending {
		return domain.Contact{}, ErrContactNotPending
	}
	if ownerContact.InvitedByUserID != peerUserID {
		return domain.Contact{}, ErrContactNotIncoming
	}

	now := time.Now().UTC()
	ownerContact.State = domain.ContactStateAccepted
	ownerContact.UpdatedAt = now
	peerContact.State = domain.ContactStateAccepted
	peerContact.UpdatedAt = now

	ownerContacts[peerUserID] = ownerContact
	peerContacts[ownerUserID] = peerContact
	return ownerContact, nil
}

func (s *MemoryService) Reject(_ context.Context, ownerUserID, peerUserID string) error {
	if ownerUserID == "" || peerUserID == "" || ownerUserID == peerUserID {
		return ErrInvalidPeer
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ownerContacts := s.contactsByUser[ownerUserID]
	peerContacts := s.contactsByUser[peerUserID]
	ownerContact, ok := ownerContacts[peerUserID]
	if !ok {
		return ErrContactNotFound
	}
	if ownerContact.State != domain.ContactStatePending {
		return ErrContactNotPending
	}
	if ownerContact.InvitedByUserID != peerUserID {
		return ErrContactNotIncoming
	}

	delete(ownerContacts, peerUserID)
	if len(ownerContacts) == 0 {
		delete(s.contactsByUser, ownerUserID)
	}
	if peerContacts != nil {
		delete(peerContacts, ownerUserID)
		if len(peerContacts) == 0 {
			delete(s.contactsByUser, peerUserID)
		}
	}
	return nil
}

func (s *MemoryService) List(_ context.Context, ownerUserID string) ([]domain.Contact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contacts := s.contactsByUser[ownerUserID]
	result := make([]domain.Contact, 0, len(contacts))
	for _, contact := range contacts {
		result = append(result, contact)
	}

	return result, nil
}
