package http

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/realtime"
	"github.com/IppolitovTech/go-realtime-kanban/internal/service"
)

// cardResponse is realtime.CardPayload under a transport-local name — the
// REST and WS responses for a card are the same shape on purpose (see
// docs/ru/websocket-events.md), so this reuses that type instead of
// hand-maintaining a second copy of its fields.
type cardResponse = realtime.CardPayload

func newCardResponse(c domain.Card) cardResponse {
	return realtime.NewCardPayload(c)
}

type createCardRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type updateCardRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type moveCardRequest struct {
	TargetColumnID uuid.UUID  `json:"target_column_id"`
	PrevCardID     *uuid.UUID `json:"prev_card_id"`
	NextCardID     *uuid.UUID `json:"next_card_id"`
}

type CardHandler struct {
	cards *service.CardService
}

func NewCardHandler(cards *service.CardService) *CardHandler {
	return &CardHandler{cards: cards}
}

func (h *CardHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	columnID, ok := parseUUIDParam(w, r, "columnId", "column id")
	if !ok {
		return
	}
	var req createCardRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	card, err := h.cards.Create(r.Context(), userID, columnID, req.Title, req.Description)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusCreated, newCardResponse(card))
}

func (h *CardHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	cardID, ok := parseUUIDParam(w, r, "cardId", "card id")
	if !ok {
		return
	}
	var req updateCardRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	card, err := h.cards.UpdateContent(r.Context(), userID, cardID, req.Title, req.Description)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, newCardResponse(card))
}

func (h *CardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	cardID, ok := parseUUIDParam(w, r, "cardId", "card id")
	if !ok {
		return
	}

	if err := h.cards.Delete(r.Context(), userID, cardID); err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusNoContent, nil)
}

func (h *CardHandler) Move(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	cardID, ok := parseUUIDParam(w, r, "cardId", "card id")
	if !ok {
		return
	}
	var req moveCardRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.TargetColumnID == uuid.Nil {
		RespondError(w, http.StatusBadRequest, "target_column_id is required", "ERR_VALIDATION")
		return
	}

	card, err := h.cards.Move(r.Context(), userID, cardID, req.TargetColumnID, req.PrevCardID, req.NextCardID)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, newCardResponse(card))
}
