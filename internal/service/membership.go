package service

import (
	"context"
	"strconv"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
)

// requireMember returns domain.ErrNotBoardMember unless userID is a member
// of boardID. Every service method that reaches a board/column/card first
// goes through this — see vision.md, "Users and roles": MVP has a
// single role, so membership alone (not ownership) gates every action.
func requireMember(ctx context.Context, boards repository.BoardRepository, boardID, userID uuid.UUID) error {
	ok, err := boards.IsMember(ctx, boardID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotBoardMember
	}
	return nil
}

func validateTitle(field, title string, maxLen int) error {
	if len(title) < 1 {
		return domain.NewValidationError(field, "must not be empty")
	}
	if len(title) > maxLen {
		return domain.NewValidationError(field, "must be at most "+strconv.Itoa(maxLen)+" characters")
	}
	return nil
}
