// Package realtime holds the WebSocket broadcast hub (see ADR 002) and the
// wire-format types for the events it delivers — the counterpart to
// internal/transport/http on the push side. See docs/ru/websocket-events.md
// for the event catalogue and docs/ru/architecture.md's REST-vs-WebSocket
// roles section for why Hub carries no business logic of its own.
package realtime

import (
	"time"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

type EventType string

const (
	EventColumnCreated EventType = "column.created"
	EventColumnUpdated EventType = "column.updated"
	EventColumnDeleted EventType = "column.deleted"
	EventColumnMoved   EventType = "column.moved"
	EventCardCreated   EventType = "card.created"
	EventCardUpdated   EventType = "card.updated"
	EventCardDeleted   EventType = "card.deleted"
	EventCardMoved     EventType = "card.moved"
)

// Event is the envelope broadcast to every client subscribed to BoardID.
type Event struct {
	Type       EventType `json:"type"`
	BoardID    uuid.UUID `json:"board_id"`
	Data       any       `json:"data"`
	OccurredAt time.Time `json:"occurred_at"`
}

func NewEvent(t EventType, boardID uuid.UUID, data any) Event {
	return Event{Type: t, BoardID: boardID, Data: data, OccurredAt: time.Now().UTC()}
}

// CardPayload/ColumnPayload are the wire shape shared by the REST and WS
// responses for a card/column, so a frontend can parse both with the same
// code — see websocket-events.md. They live here, not in transport/http,
// because service depends on realtime (as the Publisher interface's
// argument type) and transport/http depends on service; putting them in
// transport/http would make realtime depend on it too, inverting that
// chain. transport/http's cardResponse/columnResponse are aliases of these
// two types rather than separate structs, so the two transports can't
// drift apart.
type CardPayload struct {
	ID          uuid.UUID `json:"id"`
	ColumnID    uuid.UUID `json:"column_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	OrderNum    float64   `json:"order_num"`
	AuthorID    uuid.UUID `json:"author_id"`
	CreatedAt   string    `json:"created_at"`
}

func NewCardPayload(c domain.Card) CardPayload {
	return CardPayload{
		ID:          c.ID,
		ColumnID:    c.ColumnID,
		Title:       c.Title,
		Description: c.Description,
		OrderNum:    c.OrderNum,
		AuthorID:    c.AuthorID,
		CreatedAt:   c.CreatedAt.Format(time.RFC3339),
	}
}

type ColumnPayload struct {
	ID        uuid.UUID `json:"id"`
	BoardID   uuid.UUID `json:"board_id"`
	Title     string    `json:"title"`
	OrderNum  float64   `json:"order_num"`
	CreatedAt string    `json:"created_at"`
}

func NewColumnPayload(c domain.Column) ColumnPayload {
	return ColumnPayload{
		ID:        c.ID,
		BoardID:   c.BoardID,
		Title:     c.Title,
		OrderNum:  c.OrderNum,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
}
