package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/circle-link/circle-link/server/internal/domain"
	"github.com/circle-link/circle-link/server/internal/service/auth"
	"github.com/circle-link/circle-link/server/internal/service/contact"
)

type inviteContactRequest struct {
	PeerEmail string `json:"peerEmail"`
}

func (s *Server) handleListContacts(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	contacts, err := s.contactService.List(r.Context(), session.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load contacts.")
		return
	}

	items := make([]map[string]any, 0, len(contacts))
	for _, item := range contacts {
		peer, err := s.authService.GetUser(r.Context(), item.PeerUserID)
		if err != nil {
			continue
		}
		items = append(items, serializeContact(item, peer.Email, peer.DisplayName, session.UserID))
	}

	writeData(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (s *Server) handleInviteContact(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	var req inviteContactRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid JSON request body.")
		return
	}

	peerEmail := strings.TrimSpace(strings.ToLower(req.PeerEmail))
	if peerEmail == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Peer email is required.")
		return
	}

	currentUser, err := s.authService.GetUser(r.Context(), session.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Current user is invalid.")
		return
	}
	if currentUser.Email == peerEmail {
		writeError(w, http.StatusBadRequest, "CONTACT_INVALID_PEER", "You cannot add yourself as a contact.")
		return
	}

	peer, err := s.authService.FindUserByEmail(r.Context(), peerEmail)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "CONTACT_NOT_FOUND", "Peer email was not found.")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve peer email.")
		return
	}

	created, err := s.contactService.Invite(r.Context(), session.UserID, peer.ID)
	if err != nil {
		if errors.Is(err, contact.ErrInvalidPeer) {
			writeError(w, http.StatusBadRequest, "CONTACT_INVALID_PEER", "Peer user is invalid.")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create contact.")
		return
	}

	writeContactResponse(w, http.StatusCreated, created, peer.Email, peer.DisplayName, session.UserID)
}

func (s *Server) handleAcceptContact(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	peerUserID := strings.TrimSpace(r.PathValue("peerUserId"))
	if peerUserID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Peer user id is required.")
		return
	}

	accepted, peer, ok := s.acceptOrRejectContactCommon(w, r, session.UserID, peerUserID, true)
	if !ok {
		return
	}

	writeContactResponse(w, http.StatusOK, accepted, peer.Email, peer.DisplayName, session.UserID)
}

func (s *Server) handleRejectContact(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAccessSession(w, r)
	if !ok {
		return
	}

	peerUserID := strings.TrimSpace(r.PathValue("peerUserId"))
	if peerUserID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Peer user id is required.")
		return
	}

	_, _, ok = s.acceptOrRejectContactCommon(w, r, session.UserID, peerUserID, false)
	if !ok {
		return
	}

	writeData(w, http.StatusOK, map[string]any{
		"peerUserId": peerUserID,
		"status":     "rejected",
	})
}

func (s *Server) acceptOrRejectContactCommon(w http.ResponseWriter, r *http.Request, ownerUserID, peerUserID string, accept bool) (domain.Contact, domain.User, bool) {
	peer, err := s.authService.GetUser(r.Context(), peerUserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "CONTACT_NOT_FOUND", "Peer user was not found.")
		return domain.Contact{}, domain.User{}, false
	}

	if accept {
		accepted, err := s.contactService.Accept(r.Context(), ownerUserID, peerUserID)
		if err != nil {
			handleContactMutationError(w, err)
			return domain.Contact{}, domain.User{}, false
		}
		return accepted, peer, true
	}

	if err := s.contactService.Reject(r.Context(), ownerUserID, peerUserID); err != nil {
		handleContactMutationError(w, err)
		return domain.Contact{}, domain.User{}, false
	}

	return domain.Contact{}, peer, true
}

func handleContactMutationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, contact.ErrInvalidPeer):
		writeError(w, http.StatusBadRequest, "CONTACT_INVALID_PEER", "Peer user is invalid.")
	case errors.Is(err, contact.ErrContactNotFound):
		writeError(w, http.StatusNotFound, "CONTACT_NOT_FOUND", "Contact was not found.")
	case errors.Is(err, contact.ErrContactNotPending):
		writeError(w, http.StatusConflict, "CONTACT_NOT_PENDING", "Contact invite is no longer pending.")
	case errors.Is(err, contact.ErrContactNotIncoming):
		writeError(w, http.StatusConflict, "CONTACT_NOT_INCOMING", "Only incoming invites can be changed.")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update contact.")
	}
}

func writeContactResponse(w http.ResponseWriter, status int, item domain.Contact, peerEmail, displayName, ownerUserID string) {
	writeData(w, status, serializeContact(item, peerEmail, displayName, ownerUserID))
}

func serializeContact(item domain.Contact, peerEmail, displayName, ownerUserID string) map[string]any {
	direction := "outgoing"
	canAccept := false
	canReject := false

	switch {
	case item.State == domain.ContactStateAccepted:
		direction = "accepted"
	case item.InvitedByUserID == ownerUserID:
		direction = "outgoing"
	case item.InvitedByUserID != "" && item.InvitedByUserID != ownerUserID:
		direction = "incoming"
		canAccept = item.State == domain.ContactStatePending
		canReject = item.State == domain.ContactStatePending
	}

	return map[string]any{
		"peerUserId":  item.PeerUserID,
		"peerEmail":   peerEmail,
		"displayName": displayName,
		"state":       item.State,
		"direction":   direction,
		"canAccept":   canAccept,
		"canReject":   canReject,
		"createdAt":   item.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":   item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
