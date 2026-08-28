package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository/postgres/generated"
)

var _ repository.CardRepository = (*CardRepository)(nil)

type CardRepository struct {
	pool *pgxpool.Pool
}

func NewCardRepository(pool *pgxpool.Pool) *CardRepository {
	return &CardRepository{pool: pool}
}

func (r *CardRepository) Create(ctx context.Context, card domain.Card) (domain.Card, error) {
	row, err := queriesFor(ctx, r.pool).CreateCard(ctx, generated.CreateCardParams{
		ColumnID:    toPgUUID(card.ColumnID),
		Title:       card.Title,
		Description: card.Description,
		OrderNum:    card.OrderNum,
		AuthorID:    toPgUUID(card.AuthorID),
	})
	if err != nil {
		return domain.Card{}, err
	}
	return cardFromRow(row), nil
}

func (r *CardRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Card, error) {
	row, err := queriesFor(ctx, r.pool).GetCardByID(ctx, toPgUUID(id))
	if err != nil {
		return domain.Card{}, mapNotFound(err, domain.ErrCardNotFound)
	}
	return cardFromRow(row), nil
}

func (r *CardRepository) ListByColumn(ctx context.Context, columnID uuid.UUID) ([]domain.Card, error) {
	rows, err := queriesFor(ctx, r.pool).ListCardsByColumn(ctx, toPgUUID(columnID))
	if err != nil {
		return nil, err
	}
	cards := make([]domain.Card, len(rows))
	for i, row := range rows {
		cards[i] = cardFromRow(row)
	}
	return cards, nil
}

func (r *CardRepository) UpdateContent(ctx context.Context, id uuid.UUID, title, description string) (domain.Card, error) {
	row, err := queriesFor(ctx, r.pool).UpdateCardContent(ctx, generated.UpdateCardContentParams{
		ID:          toPgUUID(id),
		Title:       title,
		Description: description,
	})
	if err != nil {
		return domain.Card{}, mapNotFound(err, domain.ErrCardNotFound)
	}
	return cardFromRow(row), nil
}

func (r *CardRepository) UpdateOrder(ctx context.Context, id, columnID uuid.UUID, orderNum float64) (domain.Card, error) {
	row, err := queriesFor(ctx, r.pool).UpdateCardOrder(ctx, generated.UpdateCardOrderParams{
		ID:       toPgUUID(id),
		ColumnID: toPgUUID(columnID),
		OrderNum: orderNum,
	})
	if err != nil {
		return domain.Card{}, mapNotFound(err, domain.ErrCardNotFound)
	}
	return cardFromRow(row), nil
}

func (r *CardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return queriesFor(ctx, r.pool).DeleteCard(ctx, toPgUUID(id))
}

func (r *CardRepository) MaxOrder(ctx context.Context, columnID uuid.UUID) (float64, error) {
	return queriesFor(ctx, r.pool).MaxCardOrder(ctx, toPgUUID(columnID))
}

func (r *CardRepository) LockForReorder(ctx context.Context, columnID uuid.UUID) error {
	return queriesFor(ctx, r.pool).LockColumnForReorder(ctx, columnID.String())
}

func cardFromRow(row generated.Card) domain.Card {
	return domain.Card{
		ID:          fromPgUUID(row.ID),
		ColumnID:    fromPgUUID(row.ColumnID),
		Title:       row.Title,
		Description: row.Description,
		OrderNum:    row.OrderNum,
		AuthorID:    fromPgUUID(row.AuthorID),
		CreatedAt:   fromPgTime(row.CreatedAt),
	}
}
