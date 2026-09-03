package domain

import (
	"time"

	"github.com/google/uuid"
)

// User is a registered account. PasswordHash is only ever populated/read
// by the repository and AuthService — no transport handler marshals
// domain.User directly, they all build their own response DTOs, so this
// never leaks into a JSON response.
type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
}
