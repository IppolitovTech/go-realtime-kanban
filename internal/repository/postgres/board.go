package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository/postgres/generated"
)

// pgUniqueViolation is the SQLSTATE code Postgres reports when an INSERT
// violates a unique/primary-key constraint.
const pgUniqueViolation = "23505"

var _ repository.BoardRepository = (*BoardRepository)(nil)

type BoardRepository struct {
	pool *pgxpool.Pool
}

func NewBoardRepository(pool *pgxpool.Pool) *BoardRepository {
	return &BoardRepository{pool: pool}
}

func (r *BoardRepository) Create(ctx context.Context, board domain.Board) (domain.Board, error) {
	row, err := queriesFor(ctx, r.pool).CreateBoard(ctx, generated.CreateBoardParams{
		Title:   board.Title,
		OwnerID: toPgUUID(board.OwnerID),
	})
	if err != nil {
		return domain.Board{}, err
	}
	return boardFromRow(row), nil
}

func (r *BoardRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Board, error) {
	row, err := queriesFor(ctx, r.pool).GetBoardByID(ctx, toPgUUID(id))
	if err != nil {
		return domain.Board{}, mapNotFound(err, domain.ErrBoardNotFound)
	}
	return boardFromRow(row), nil
}

func (r *BoardRepository) ListByMember(ctx context.Context, userID uuid.UUID) ([]domain.Board, error) {
	rows, err := queriesFor(ctx, r.pool).ListBoardsByMember(ctx, toPgUUID(userID))
	if err != nil {
		return nil, err
	}
	boards := make([]domain.Board, len(rows))
	for i, row := range rows {
		boards[i] = boardFromRow(row)
	}
	return boards, nil
}

func (r *BoardRepository) UpdateTitle(ctx context.Context, id uuid.UUID, title string) (domain.Board, error) {
	row, err := queriesFor(ctx, r.pool).UpdateBoardTitle(ctx, generated.UpdateBoardTitleParams{
		ID:    toPgUUID(id),
		Title: title,
	})
	if err != nil {
		return domain.Board{}, mapNotFound(err, domain.ErrBoardNotFound)
	}
	return boardFromRow(row), nil
}

func (r *BoardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return queriesFor(ctx, r.pool).DeleteBoard(ctx, toPgUUID(id))
}

func (r *BoardRepository) AddMember(ctx context.Context, boardID, userID uuid.UUID) (domain.BoardMember, error) {
	row, err := queriesFor(ctx, r.pool).AddBoardMember(ctx, generated.AddBoardMemberParams{
		BoardID: toPgUUID(boardID),
		UserID:  toPgUUID(userID),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return domain.BoardMember{}, domain.NewValidationError("email", "user is already a board member")
		}
		return domain.BoardMember{}, err
	}
	return domain.BoardMember{
		BoardID:  fromPgUUID(row.BoardID),
		UserID:   fromPgUUID(row.UserID),
		Email:    row.Email,
		Name:     row.Name,
		JoinedAt: fromPgTime(row.JoinedAt),
	}, nil
}

func (r *BoardRepository) ListMembers(ctx context.Context, boardID uuid.UUID) ([]domain.BoardMember, error) {
	rows, err := queriesFor(ctx, r.pool).ListBoardMembers(ctx, toPgUUID(boardID))
	if err != nil {
		return nil, err
	}
	members := make([]domain.BoardMember, len(rows))
	for i, row := range rows {
		members[i] = domain.BoardMember{
			BoardID:  fromPgUUID(row.BoardID),
			UserID:   fromPgUUID(row.UserID),
			Email:    row.Email,
			Name:     row.Name,
			JoinedAt: fromPgTime(row.JoinedAt),
		}
	}
	return members, nil
}

func (r *BoardRepository) IsMember(ctx context.Context, boardID, userID uuid.UUID) (bool, error) {
	return queriesFor(ctx, r.pool).IsBoardMember(ctx, generated.IsBoardMemberParams{
		BoardID: toPgUUID(boardID),
		UserID:  toPgUUID(userID),
	})
}

func boardFromRow(row generated.Board) domain.Board {
	return domain.Board{
		ID:        fromPgUUID(row.ID),
		Title:     row.Title,
		OwnerID:   fromPgUUID(row.OwnerID),
		CreatedAt: fromPgTime(row.CreatedAt),
	}
}
