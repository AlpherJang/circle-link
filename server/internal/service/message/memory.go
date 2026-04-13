package message

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/circle-link/circle-link/server/internal/domain"
	"github.com/circle-link/circle-link/server/internal/platform/ids"
)

var ErrMessageNotFound = errors.New("message was not found")

type MemoryService struct {
	mu                sync.RWMutex
	messageByID       map[string]domain.DebugMessage
	inboxIDsByUserID  map[string][]string
	subscribersByUser map[string]map[int]subscriber
	nextSubscriberID  int
}

type subscriber struct {
	deviceID string
	ch       chan domain.DebugMessage
}

func NewMemoryService() *MemoryService {
	return &MemoryService{
		messageByID:       make(map[string]domain.DebugMessage),
		inboxIDsByUserID:  make(map[string][]string),
		subscribersByUser: make(map[string]map[int]subscriber),
	}
}

func (s *MemoryService) Send(_ context.Context, input SendInput) (domain.DebugMessage, error) {
	messageID := input.MessageID
	if messageID == "" {
		messageID = ids.New("msg")
	}
	conversationID := input.ConversationID
	if conversationID == "" {
		conversationID = ids.New("conv")
	}
	contentType := input.ContentType
	if contentType == "" {
		contentType = "text/plain"
	}

	now := time.Now().UTC()
	deliveryStatus := domain.DeliveryStatusStoredOffline
	if input.RecipientOnline {
		deliveryStatus = domain.DeliveryStatusAccepted
	}

	message := domain.DebugMessage{
		ID:                messageID,
		ConversationID:    conversationID,
		SenderUserID:      input.SenderUserID,
		SenderDeviceID:    input.SenderDeviceID,
		SenderEmail:       input.SenderEmail,
		RecipientUserID:   input.RecipientUserID,
		RecipientDeviceID: input.RecipientDeviceID,
		RecipientEmail:    input.RecipientEmail,
		ContentType:       contentType,
		ClientMessageSeq:  input.ClientMessageSeq,
		Header:            cloneHeader(input.Header),
		RatchetPublicKey:  input.RatchetPublicKey,
		Ciphertext:        input.Ciphertext,
		Body:              input.Body,
		DeliveryStatus:    deliveryStatus,
		StoredAt:          now,
		SentAt:            now,
	}

	s.mu.Lock()
	s.messageByID[message.ID] = message
	s.inboxIDsByUserID[input.RecipientUserID] = append(s.inboxIDsByUserID[input.RecipientUserID], message.ID)
	subscribers := s.copySubscribersLocked(input.RecipientUserID)
	s.mu.Unlock()

	for _, subscriber := range subscribers {
		if !shouldDeliverToDevice(message, subscriber.deviceID) {
			continue
		}
		select {
		case subscriber.ch <- cloneMessage(message):
		default:
		}
	}

	return cloneMessage(message), nil
}

func (s *MemoryService) ListInbox(_ context.Context, recipientUserID, recipientDeviceID string) ([]domain.DebugMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messageIDs := s.inboxIDsByUserID[recipientUserID]
	result := make([]domain.DebugMessage, 0, len(messageIDs))
	for _, messageID := range messageIDs {
		item, ok := s.messageByID[messageID]
		if !ok {
			continue
		}
		if !shouldDeliverToDevice(item, recipientDeviceID) {
			continue
		}
		result = append(result, cloneMessage(item))
	}

	return result, nil
}

func (s *MemoryService) SubscribeInbox(_ context.Context, recipientUserID, recipientDeviceID string) (<-chan domain.DebugMessage, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSubscriberID++
	subscriberID := s.nextSubscriberID
	ch := make(chan domain.DebugMessage, 8)

	if s.subscribersByUser[recipientUserID] == nil {
		s.subscribersByUser[recipientUserID] = make(map[int]subscriber)
	}
	s.subscribersByUser[recipientUserID][subscriberID] = subscriber{
		deviceID: recipientDeviceID,
		ch:       ch,
	}

	cancel := func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		subscribers := s.subscribersByUser[recipientUserID]
		entry, ok := subscribers[subscriberID]
		if !ok {
			return
		}

		delete(subscribers, subscriberID)
		close(entry.ch)
		if len(subscribers) == 0 {
			delete(s.subscribersByUser, recipientUserID)
		}
	}

	return ch, cancel
}

func (s *MemoryService) Acknowledge(_ context.Context, input AcknowledgeInput) (domain.DebugMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.messageByID[input.MessageID]
	if !ok || item.RecipientUserID != input.RecipientUserID {
		return domain.DebugMessage{}, ErrMessageNotFound
	}

	if input.RecipientDeviceID != "" {
		if item.RecipientDeviceID != "" && item.RecipientDeviceID != input.RecipientDeviceID {
			return domain.DebugMessage{}, ErrMessageNotFound
		}
		item.RecipientDeviceID = input.RecipientDeviceID
	}

	now := time.Now().UTC()
	switch input.Status {
	case domain.DeliveryStatusRead:
		item.DeliveryStatus = domain.DeliveryStatusRead
		if item.DeliveredAt == nil {
			deliveredAt := now
			item.DeliveredAt = &deliveredAt
		}
		readAt := now
		item.ReadAt = &readAt
	case domain.DeliveryStatusDelivered:
		item.DeliveryStatus = domain.DeliveryStatusDelivered
		deliveredAt := now
		item.DeliveredAt = &deliveredAt
	case domain.DeliveryStatusAccepted:
		item.DeliveryStatus = domain.DeliveryStatusAccepted
	default:
		item.DeliveryStatus = input.Status
	}

	s.messageByID[input.MessageID] = item
	return cloneMessage(item), nil
}

func (s *MemoryService) copySubscribersLocked(recipientUserID string) []subscriber {
	subscribers := s.subscribersByUser[recipientUserID]
	result := make([]subscriber, 0, len(subscribers))
	for _, subscriber := range subscribers {
		result = append(result, subscriber)
	}

	return result
}

func shouldDeliverToDevice(item domain.DebugMessage, recipientDeviceID string) bool {
	if recipientDeviceID == "" {
		return true
	}
	if item.RecipientDeviceID == "" {
		return true
	}

	return item.RecipientDeviceID == recipientDeviceID
}

func cloneMessage(item domain.DebugMessage) domain.DebugMessage {
	item.Header = cloneHeader(item.Header)
	if item.DeliveredAt != nil {
		deliveredAt := *item.DeliveredAt
		item.DeliveredAt = &deliveredAt
	}
	if item.ReadAt != nil {
		readAt := *item.ReadAt
		item.ReadAt = &readAt
	}
	return item
}

func cloneHeader(header map[string]any) map[string]any {
	if len(header) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(header))
	for key, value := range header {
		cloned[key] = value
	}

	return cloned
}
