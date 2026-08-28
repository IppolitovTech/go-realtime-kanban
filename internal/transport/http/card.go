package http

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/service"
)

type cardResponse struct {
	ID          uuid.UUID `json:"id"`
	ColumnID    uuid.UUID `json:"column_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	OrderNum    float64   `json:"order_num"`
	AuthorID    uuid.UUID `json:"author_id"`
	CreatedAt   string    `json:"created_at"`
}

func newCardResponse(c domain.Card) cardResponse {
	return cardResponse{
		ID:          c.ID,
		ColumnID:    c.ColumnID,
		Title:       c.Title,
		Description: c.Description,
		OrderNum:    c.OrderNum,
		AuthorID:    c.AuthorID,
		CreatedAt:   c.CreatedAt.Format(timeFormat),
	}
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
