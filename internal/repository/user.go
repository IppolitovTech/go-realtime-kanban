package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
}
