package httpapi

import "sync"

type wsHub struct {
	mu              sync.RWMutex
	sessionsByUser  map[string]map[string]*boundWSSession
	pendingAckByMsg map[string]pendingAckEntry
}

type boundWSSession struct {
	userID   string
	deviceID string
	send     chan wsServerEvent
}

type pendingAckEntry struct {
	sender *boundWSSession
	item   wsAckContext
}

type wsAckContext struct {
	MessageID         string
	ConversationID    string
	SenderUserID      string
	SenderDeviceID    string
	RecipientUserID   string
	RecipientDeviceID string
	ClientMessageSeq  uint64
}

func newWSHub() *wsHub {
	return &wsHub{
		sessionsByUser:  make(map[string]map[string]*boundWSSession),
		pendingAckByMsg: make(map[string]pendingAckEntry),
	}
}

func (h *wsHub) bind(userID, deviceID string, session *boundWSSession) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.sessionsByUser[userID] == nil {
		h.sessionsByUser[userID] = make(map[string]*boundWSSession)
	}
	h.sessionsByUser[userID][deviceID] = session
}

func (h *wsHub) unbind(userID, deviceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sessions := h.sessionsByUser[userID]
	if sessions == nil {
		return
	}
	delete(sessions, deviceID)
	if len(sessions) == 0 {
		delete(h.sessionsByUser, userID)
	}
}

func (h *wsHub) firstSessionForUser(userID string) *boundWSSession {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, session := range h.sessionsByUser[userID] {
		return session
	}

	return nil
}

func (h *wsHub) sessionForUserAndDevice(userID, deviceID string) *boundWSSession {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.sessionsByUser[userID][deviceID]
}

func (h *wsHub) trackPendingAck(item wsAckContext, sender *boundWSSession) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.pendingAckByMsg[item.MessageID] = pendingAckEntry{
		sender: sender,
		item:   item,
	}
}

func (h *wsHub) pendingAck(messageID string) (pendingAckEntry, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entry, ok := h.pendingAckByMsg[messageID]
	return entry, ok
}

func (h *wsHub) clearPendingAck(messageID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.pendingAckByMsg, messageID)
}
