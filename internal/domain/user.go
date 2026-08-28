package domain

import (
	"time"

	"github.com/google/uuid"
)

// User is a stub until Stage 2 wires up real registration/auth — see
// architecture.md, "User stub in Stage 1".
type User struct {
	ID        uuid.UUID
	Email     string
	Name      string
	CreatedAt time.Time
}
