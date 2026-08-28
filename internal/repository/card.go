package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

type CardRepository interface {
	Create(ctx context.Context, card domain.Card) (domain.Card, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Card, error)
	ListByColumn(ctx context.Context, columnID uuid.UUID) ([]domain.Card, error)
	UpdateContent(ctx context.Context, id uuid.UUID, title, description string) (domain.Card, error)
	// UpdateOrder also writes columnID, since a card move can cross
	// into a different column (see MoveCardRequest in openapi.yaml).
	UpdateOrder(ctx context.Context, id, columnID uuid.UUID, orderNum float64) (domain.Card, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// MaxOrder returns the highest order_num among the column's cards,
	// 0 if it has none — used to place a newly created card last.
	MaxOrder(ctx context.Context, columnID uuid.UUID) (float64, error)

	// LockForReorder takes a Postgres advisory lock scoped to columnID,
	// serializing concurrent card moves within the same column (ADR
	// 004). Must be called inside repository.TxManager.WithinTx, before
	// reading neighbor order_num values.
	LockForReorder(ctx context.Context, columnID uuid.UUID) error
}
