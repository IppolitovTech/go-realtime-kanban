package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/realtime"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
)

const columnTitleMaxLen = 50

// ColumnService holds column business logic: membership gating plus
// order_num placement/reordering per ADR 004. Every mutation publishes a
// realtime event after it commits — see roadmap.md, Stage 3, on
// broadcasting an event to every client on the board after each REST
// change — so WS clients stay in sync without the hub knowing any business
// logic itself (architecture.md's REST-vs-WebSocket roles section).
type ColumnService struct {
	columns   repository.ColumnRepository
	boards    repository.BoardRepository
	tx        repository.TxManager
	publisher realtime.Publisher
}

func NewColumnService(columns repository.ColumnRepository, boards repository.BoardRepository, tx repository.TxManager, publisher realtime.Publisher) *ColumnService {
	return &ColumnService{columns: columns, boards: boards, tx: tx, publisher: publisher}
}

func (s *ColumnService) Create(ctx context.Context, userID, boardID uuid.UUID, title string) (domain.Column, error) {
	if err := requireMember(ctx, s.boards, boardID, userID); err != nil {
		return domain.Column{}, err
	}
	if err := validateTitle("title", title, columnTitleMaxLen); err != nil {
		return domain.Column{}, err
	}

	var column domain.Column
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.columns.LockForReorder(ctx, boardID); err != nil {
			return err
		}
		maxOrder, err := s.columns.MaxOrder(ctx, boardID)
		if err != nil {
			return err
		}
		column, err = s.columns.Create(ctx, domain.Column{
			BoardID:  boardID,
			Title:    title,
			OrderNum: appendOrder(maxOrder),
		})
		return err
	})
	if err != nil {
		return domain.Column{}, err
	}
	s.publisher.Publish(ctx, realtime.NewEvent(realtime.EventColumnCreated, boardID, realtime.NewColumnPayload(column)))
	return column, nil
}

func (s *ColumnService) ListByBoard(ctx context.Context, userID, boardID uuid.UUID) ([]domain.Column, error) {
	if err := requireMember(ctx, s.boards, boardID, userID); err != nil {
		return nil, err
	}
	return s.columns.ListByBoard(ctx, boardID)
}

func (s *ColumnService) UpdateTitle(ctx context.Context, userID, columnID uuid.UUID, title string) (domain.Column, error) {
	column, err := s.columns.GetByID(ctx, columnID)
	if err != nil {
		return domain.Column{}, err
	}
	if err := requireMember(ctx, s.boards, column.BoardID, userID); err != nil {
		return domain.Column{}, err
	}
	if err := validateTitle("title", title, columnTitleMaxLen); err != nil {
		return domain.Column{}, err
	}
	updated, err := s.columns.UpdateTitle(ctx, columnID, title)
	if err != nil {
		return domain.Column{}, err
	}
	s.publisher.Publish(ctx, realtime.NewEvent(realtime.EventColumnUpdated, updated.BoardID, realtime.NewColumnPayload(updated)))
	return updated, nil
}

func (s *ColumnService) Delete(ctx context.Context, userID, columnID uuid.UUID) error {
	column, err := s.columns.GetByID(ctx, columnID)
	if err != nil {
		return err
	}
	if err := requireMember(ctx, s.boards, column.BoardID, userID); err != nil {
		return err
	}
	if err := s.columns.Delete(ctx, columnID); err != nil {
		return err
	}
	s.publisher.Publish(ctx, realtime.NewEvent(realtime.EventColumnDeleted, column.BoardID, realtime.NewColumnPayload(column)))
	return nil
}

// Move relocates a column within its board to sit right after prevColumnID
// and right before nextColumnID (either may be nil for "start"/"end" of the
// list; prevColumnID's order_num must precede nextColumnID's). The new
// order_num is the midpoint of its new neighbors; if that midpoint has
// collapsed into one of them (ADR 004), the whole board's columns are
// renumbered with a fresh orderStep gap first.
func (s *ColumnService) Move(ctx context.Context, userID, columnID uuid.UUID, prevColumnID, nextColumnID *uuid.UUID) (domain.Column, error) {
	column, err := s.columns.GetByID(ctx, columnID)
	if err != nil {
		return domain.Column{}, err
	}
	if err := requireMember(ctx, s.boards, column.BoardID, userID); err != nil {
		return domain.Column{}, err
	}

	var moved domain.Column
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.columns.LockForReorder(ctx, column.BoardID); err != nil {
			return err
		}

		var prevOrder, nextOrder *float64
		if prevColumnID != nil {
			prev, err := s.columns.GetByID(ctx, *prevColumnID)
			if err != nil {
				return err
			}
			if prev.BoardID != column.BoardID {
				return domain.NewValidationError("prev_column_id", "must belong to the same board")
			}
			prevOrder = &prev.OrderNum
		}
		if nextColumnID != nil {
			next, err := s.columns.GetByID(ctx, *nextColumnID)
			if err != nil {
				return err
			}
			if next.BoardID != column.BoardID {
				return domain.NewValidationError("next_column_id", "must belong to the same board")
			}
			nextOrder = &next.OrderNum
		}

		siblings, err := s.columns.ListByBoard(ctx, column.BoardID)
		if err != nil {
			return err
		}
		orderedSiblings := make([]orderedSibling, len(siblings))
		for i, c := range siblings {
			orderedSiblings[i] = orderedSibling{ID: c.ID, OrderNum: c.OrderNum}
		}

		newOrder, err := resolveMoveOrder(moveOrderRequest{
			MovingID:  columnID,
			PrevID:    prevColumnID,
			PrevOrder: prevOrder,
			NextOrder: nextOrder,
			PrevField: "prev_column_id",
			Siblings:  orderedSiblings,
			UpdateOrder: func(id uuid.UUID, order float64) error {
				_, err := s.columns.UpdateOrder(ctx, id, order)
				return err
			},
		})
		if err != nil {
			return err
		}

		moved, err = s.columns.UpdateOrder(ctx, columnID, newOrder)
		return err
	})
	if err != nil {
		return domain.Column{}, err
	}
	s.publisher.Publish(ctx, realtime.NewEvent(realtime.EventColumnMoved, moved.BoardID, realtime.NewColumnPayload(moved)))
	return moved, nil
}
