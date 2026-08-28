package domain

import (
	"time"

	"github.com/google/uuid"
)

// Column order is a float with local renumbering on precision collapse —
// see ADR 004.
type Column struct {
	ID        uuid.UUID
	BoardID   uuid.UUID
	Title     string
	OrderNum  float64
	CreatedAt time.Time
}
