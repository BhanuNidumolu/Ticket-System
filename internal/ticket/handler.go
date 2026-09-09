package ticket

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"ticket-system/internal/auth"
	"ticket-system/internal/httpx"
	"ticket-system/internal/idgen"
)

// Handler wires the ticket store into HTTP handlers. All handlers here
// assume they run behind auth.RequireAuth middleware, i.e. a userID is
// always present in the request context.
type Handler struct {
	Store Store
}

func NewHandler(store Store) *Handler {
	return &Handler{Store: store}
}

func (h *Handler) requireUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "authentication required")
		return "", false
	}
	return userID, true
}

type createTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Create handles POST /tickets.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	var req createTicketRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		httpx.WriteError(w, http.StatusBadRequest, "title is required")
		return
	}

	now := time.Now().UTC()
	t := &Ticket{
		ID:          idgen.New(),
		UserID:      userID,
		Title:       req.Title,
		Description: strings.TrimSpace(req.Description),
		Status:      StatusOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.Store.Create(t); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to create ticket")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, t)
}

// List handles GET /tickets — returns only tickets owned by the
// authenticated user.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	tickets, err := h.Store.ListByUser(userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list tickets")
		return
	}
	if tickets == nil {
		tickets = []*Ticket{} // always return [] instead of null
	}

	httpx.WriteJSON(w, http.StatusOK, tickets)
}

// Get handles GET /tickets/{id}. Returns 404 both when the ticket
// doesn't exist and when it belongs to another user, so we never leak
// the existence of other users' tickets.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	id := r.PathValue("id")
	t, err := h.Store.FindByID(id)
	if err != nil || t.UserID != userID {
		httpx.WriteError(w, http.StatusNotFound, "ticket not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, t)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus handles PATCH /tickets/{id}/status.
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	id := r.PathValue("id")
	t, err := h.Store.FindByID(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "ticket not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "failed to load ticket")
		return
	}
	if t.UserID != userID {
		httpx.WriteError(w, http.StatusNotFound, "ticket not found")
		return
	}

	var req updateStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	newStatus := Status(req.Status)
	if !newStatus.Valid() {
		httpx.WriteError(w, http.StatusBadRequest, "status must be one of: open, in_progress, closed")
		return
	}

	if !CanTransition(t.Status, newStatus) {
		httpx.WriteError(w, http.StatusBadRequest,
			"invalid status transition from "+string(t.Status)+" to "+string(newStatus))
		return
	}

	t.Status = newStatus
	t.UpdatedAt = time.Now().UTC()

	if err := h.Store.Update(t); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to update ticket")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, t)
}
