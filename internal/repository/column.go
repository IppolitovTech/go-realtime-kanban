package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

type ColumnRepository interface {
	Create(ctx context.Context, column domain.Column) (domain.Column, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Column, error)
	ListByBoard(ctx context.Context, boardID uuid.UUID) ([]domain.Column, error)
	UpdateTitle(ctx context.Context, id uuid.UUID, title string) (domain.Column, error)
	UpdateOrder(ctx context.Context, id uuid.UUID, orderNum float64) (domain.Column, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// MaxOrder returns the highest order_num among the board's columns,
	// 0 if it has none — used to place a newly created column last.
	MaxOrder(ctx context.Context, boardID uuid.UUID) (float64, error)

	// LockForReorder takes a Postgres advisory lock scoped to boardID,
	// serializing concurrent column moves within the same board (ADR
	// 004). Must be called inside repository.TxManager.WithinTx, before
	// reading neighbor order_num values.
	LockForReorder(ctx context.Context, boardID uuid.UUID) error
}
