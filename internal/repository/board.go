package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

type BoardRepository interface {
	Create(ctx context.Context, board domain.Board) (domain.Board, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Board, error)
	ListByMember(ctx context.Context, userID uuid.UUID) ([]domain.Board, error)
	UpdateTitle(ctx context.Context, id uuid.UUID, title string) (domain.Board, error)
	Delete(ctx context.Context, id uuid.UUID) error

	AddMember(ctx context.Context, boardID, userID uuid.UUID) (domain.BoardMember, error)
	ListMembers(ctx context.Context, boardID uuid.UUID) ([]domain.BoardMember, error)
	IsMember(ctx context.Context, boardID, userID uuid.UUID) (bool, error)
}
