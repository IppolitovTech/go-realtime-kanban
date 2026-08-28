package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository/postgres/generated"
)

var _ repository.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := queriesFor(ctx, r.pool).GetUserByID(ctx, toPgUUID(id))
	if err != nil {
		return domain.User{}, mapNotFound(err, domain.ErrUserNotFound)
	}
	return userFromRow(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	row, err := queriesFor(ctx, r.pool).GetUserByEmail(ctx, email)
	if err != nil {
		return domain.User{}, mapNotFound(err, domain.ErrUserNotFound)
	}
	return userFromRow(row), nil
}

func userFromRow(row generated.User) domain.User {
	return domain.User{
		ID:        fromPgUUID(row.ID),
		Email:     row.Email,
		Name:      row.Name,
		CreatedAt: fromPgTime(row.CreatedAt),
	}
}
