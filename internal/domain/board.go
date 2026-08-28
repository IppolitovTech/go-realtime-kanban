package domain

import (
	"time"

	"github.com/google/uuid"
)

type Board struct {
	ID        uuid.UUID
	Title     string
	OwnerID   uuid.UUID
	CreatedAt time.Time
}

// BoardMember is the User <-> Board membership link.
type BoardMember struct {
	BoardID  uuid.UUID
	UserID   uuid.UUID
	Email    string
	Name     string
	JoinedAt time.Time
}
