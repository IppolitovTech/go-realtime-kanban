package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
)

const boardTitleMaxLen = 100

// BoardService holds board-level business logic: membership gating and the
// board/board-member lifecycle. Every method takes userID explicitly so the
// dev-header stub of Stage 1 and the JWT middleware of Stage 2 are
// interchangeable callers — see architecture.md, "User context".
type BoardService struct {
	boards repository.BoardRepository
	users  repository.UserRepository
	tx     repository.TxManager
}

func NewBoardService(boards repository.BoardRepository, users repository.UserRepository, tx repository.TxManager) *BoardService {
	return &BoardService{boards: boards, users: users, tx: tx}
}

// Create creates a board owned by userID and immediately adds userID as a
// board member — ListByMember (and therefore List) would otherwise never
// see the board its own creator just made.
func (s *BoardService) Create(ctx context.Context, userID uuid.UUID, title string) (domain.Board, error) {
	if err := validateTitle("title", title, boardTitleMaxLen); err != nil {
		return domain.Board{}, err
	}

	var board domain.Board
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		board, err = s.boards.Create(ctx, domain.Board{Title: title, OwnerID: userID})
		if err != nil {
			return err
		}
		_, err = s.boards.AddMember(ctx, board.ID, userID)
		return err
	})
	if err != nil {
		return domain.Board{}, err
	}
	return board, nil
}

func (s *BoardService) Get(ctx context.Context, userID, boardID uuid.UUID) (domain.Board, error) {
	board, err := s.boards.GetByID(ctx, boardID)
	if err != nil {
		return domain.Board{}, err
	}
	if err := requireMember(ctx, s.boards, boardID, userID); err != nil {
		return domain.Board{}, err
	}
	return board, nil
}

func (s *BoardService) List(ctx context.Context, userID uuid.UUID) ([]domain.Board, error) {
	return s.boards.ListByMember(ctx, userID)
}

func (s *BoardService) ListMembers(ctx context.Context, userID, boardID uuid.UUID) ([]domain.BoardMember, error) {
	if err := requireMember(ctx, s.boards, boardID, userID); err != nil {
		return nil, err
	}
	return s.boards.ListMembers(ctx, boardID)
}

func (s *BoardService) UpdateTitle(ctx context.Context, userID, boardID uuid.UUID, title string) (domain.Board, error) {
	if _, err := s.boards.GetByID(ctx, boardID); err != nil {
		return domain.Board{}, err
	}
	if err := requireMember(ctx, s.boards, boardID, userID); err != nil {
		return domain.Board{}, err
	}
	if err := validateTitle("title", title, boardTitleMaxLen); err != nil {
		return domain.Board{}, err
	}
	return s.boards.UpdateTitle(ctx, boardID, title)
}

func (s *BoardService) Delete(ctx context.Context, userID, boardID uuid.UUID) error {
	if _, err := s.boards.GetByID(ctx, boardID); err != nil {
		return err
	}
	if err := requireMember(ctx, s.boards, boardID, userID); err != nil {
		return err
	}
	return s.boards.Delete(ctx, boardID)
}

// InviteMember adds the user with the given email as a board member. Any
// existing member can invite — MVP has no owner-only restriction, see
// vision.md, "Users and roles".
func (s *BoardService) InviteMember(ctx context.Context, userID, boardID uuid.UUID, email string) (domain.BoardMember, error) {
	if _, err := s.boards.GetByID(ctx, boardID); err != nil {
		return domain.BoardMember{}, err
	}
	if err := requireMember(ctx, s.boards, boardID, userID); err != nil {
		return domain.BoardMember{}, err
	}

	invitee, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.BoardMember{}, domain.NewValidationError("email", "no user with this email")
		}
		return domain.BoardMember{}, err
	}

	already, err := s.boards.IsMember(ctx, boardID, invitee.ID)
	if err != nil {
		return domain.BoardMember{}, err
	}
	if already {
		return domain.BoardMember{}, domain.NewValidationError("email", "user is already a board member")
	}

	return s.boards.AddMember(ctx, boardID, invitee.ID)
}
