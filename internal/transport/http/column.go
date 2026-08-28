package http

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/service"
)

type columnResponse struct {
	ID        uuid.UUID `json:"id"`
	BoardID   uuid.UUID `json:"board_id"`
	Title     string    `json:"title"`
	OrderNum  float64   `json:"order_num"`
	CreatedAt string    `json:"created_at"`
}

func newColumnResponse(c domain.Column) columnResponse {
	return columnResponse{
		ID:        c.ID,
		BoardID:   c.BoardID,
		Title:     c.Title,
		OrderNum:  c.OrderNum,
		CreatedAt: c.CreatedAt.Format(timeFormat),
	}
}

type columnDetailResponse struct {
	ID        uuid.UUID      `json:"id"`
	BoardID   uuid.UUID      `json:"board_id"`
	Title     string         `json:"title"`
	OrderNum  float64        `json:"order_num"`
	Cards     []cardResponse `json:"cards"`
	CreatedAt string         `json:"created_at"`
}

func newColumnDetailResponse(c domain.Column, cards []domain.Card) columnDetailResponse {
	cardResponses := make([]cardResponse, len(cards))
	for i, card := range cards {
		cardResponses[i] = newCardResponse(card)
	}
	return columnDetailResponse{
		ID:        c.ID,
		BoardID:   c.BoardID,
		Title:     c.Title,
		OrderNum:  c.OrderNum,
		Cards:     cardResponses,
		CreatedAt: c.CreatedAt.Format(timeFormat),
	}
}

type createColumnRequest struct {
	Title string `json:"title"`
}

type updateColumnRequest struct {
	Title string `json:"title"`
}

type moveColumnRequest struct {
	PrevColumnID *uuid.UUID `json:"prev_column_id"`
	NextColumnID *uuid.UUID `json:"next_column_id"`
}

type ColumnHandler struct {
	columns *service.ColumnService
}

func NewColumnHandler(columns *service.ColumnService) *ColumnHandler {
	return &ColumnHandler{columns: columns}
}

func (h *ColumnHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	boardID, ok := parseUUIDParam(w, r, "boardId", "board id")
	if !ok {
		return
	}
	var req createColumnRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	column, err := h.columns.Create(r.Context(), userID, boardID, req.Title)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusCreated, newColumnResponse(column))
}

func (h *ColumnHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	columnID, ok := parseUUIDParam(w, r, "columnId", "column id")
	if !ok {
		return
	}
	var req updateColumnRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	column, err := h.columns.UpdateTitle(r.Context(), userID, columnID, req.Title)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, newColumnResponse(column))
}

func (h *ColumnHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	columnID, ok := parseUUIDParam(w, r, "columnId", "column id")
	if !ok {
		return
	}

	if err := h.columns.Delete(r.Context(), userID, columnID); err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusNoContent, nil)
}

func (h *ColumnHandler) Move(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	columnID, ok := parseUUIDParam(w, r, "columnId", "column id")
	if !ok {
		return
	}
	var req moveColumnRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	column, err := h.columns.Move(r.Context(), userID, columnID, req.PrevColumnID, req.NextColumnID)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, newColumnResponse(column))
}
