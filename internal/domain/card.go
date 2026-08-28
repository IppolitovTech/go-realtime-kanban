package domain

import (
	"time"

	"github.com/google/uuid"
)

// Card order is a float with local renumbering on precision collapse —
// see ADR 004.
type Card struct {
	ID          uuid.UUID
	ColumnID    uuid.UUID
	Title       string
	Description string
	OrderNum    float64
	AuthorID    uuid.UUID
	CreatedAt   time.Time
}
