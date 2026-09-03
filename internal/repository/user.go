package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	// Create inserts a new user. It returns domain.ErrEmailTaken if email
	// is already registered.
	Create(ctx context.Context, user domain.User) (domain.User, error)
}
