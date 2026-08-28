package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository/postgres/generated"
)

var _ repository.ColumnRepository = (*ColumnRepository)(nil)

type ColumnRepository struct {
	pool *pgxpool.Pool
}

func NewColumnRepository(pool *pgxpool.Pool) *ColumnRepository {
	return &ColumnRepository{pool: pool}
}

func (r *ColumnRepository) Create(ctx context.Context, column domain.Column) (domain.Column, error) {
	row, err := queriesFor(ctx, r.pool).CreateColumn(ctx, generated.CreateColumnParams{
		BoardID:  toPgUUID(column.BoardID),
		Title:    column.Title,
		OrderNum: column.OrderNum,
	})
	if err != nil {
		return domain.Column{}, err
	}
	return columnFromRow(row), nil
}

func (r *ColumnRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Column, error) {
	row, err := queriesFor(ctx, r.pool).GetColumnByID(ctx, toPgUUID(id))
	if err != nil {
		return domain.Column{}, mapNotFound(err, domain.ErrColumnNotFound)
	}
	return columnFromRow(row), nil
}

func (r *ColumnRepository) ListByBoard(ctx context.Context, boardID uuid.UUID) ([]domain.Column, error) {
	rows, err := queriesFor(ctx, r.pool).ListColumnsByBoard(ctx, toPgUUID(boardID))
	if err != nil {
		return nil, err
	}
	columns := make([]domain.Column, len(rows))
	for i, row := range rows {
		columns[i] = columnFromRow(row)
	}
	return columns, nil
}

func (r *ColumnRepository) UpdateTitle(ctx context.Context, id uuid.UUID, title string) (domain.Column, error) {
	row, err := queriesFor(ctx, r.pool).UpdateColumnTitle(ctx, generated.UpdateColumnTitleParams{
		ID:    toPgUUID(id),
		Title: title,
	})
	if err != nil {
		return domain.Column{}, mapNotFound(err, domain.ErrColumnNotFound)
	}
	return columnFromRow(row), nil
}

func (r *ColumnRepository) UpdateOrder(ctx context.Context, id uuid.UUID, orderNum float64) (domain.Column, error) {
	row, err := queriesFor(ctx, r.pool).UpdateColumnOrder(ctx, generated.UpdateColumnOrderParams{
		ID:       toPgUUID(id),
		OrderNum: orderNum,
	})
	if err != nil {
		return domain.Column{}, mapNotFound(err, domain.ErrColumnNotFound)
	}
	return columnFromRow(row), nil
}

func (r *ColumnRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return queriesFor(ctx, r.pool).DeleteColumn(ctx, toPgUUID(id))
}

func (r *ColumnRepository) MaxOrder(ctx context.Context, boardID uuid.UUID) (float64, error) {
	return queriesFor(ctx, r.pool).MaxColumnOrder(ctx, toPgUUID(boardID))
}

func (r *ColumnRepository) LockForReorder(ctx context.Context, boardID uuid.UUID) error {
	return queriesFor(ctx, r.pool).LockBoardForReorder(ctx, boardID.String())
}

func columnFromRow(row generated.Column) domain.Column {
	return domain.Column{
		ID:        fromPgUUID(row.ID),
		BoardID:   fromPgUUID(row.BoardID),
		Title:     row.Title,
		OrderNum:  row.OrderNum,
		CreatedAt: fromPgTime(row.CreatedAt),
	}
}
