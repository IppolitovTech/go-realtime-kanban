package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/realtime"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
)

const cardTitleMaxLen = 255

// CardService holds card business logic: membership gating (resolved via
// the card's/column's board) plus order_num placement/reordering per ADR
// 004. Every mutation publishes a realtime event after it commits — see
// ColumnService's doc comment for why that lives here rather than in the
// hub.
type CardService struct {
	cards     repository.CardRepository
	columns   repository.ColumnRepository
	boards    repository.BoardRepository
	tx        repository.TxManager
	publisher realtime.Publisher
}

func NewCardService(cards repository.CardRepository, columns repository.ColumnRepository, boards repository.BoardRepository, tx repository.TxManager, publisher realtime.Publisher) *CardService {
	return &CardService{cards: cards, columns: columns, boards: boards, tx: tx, publisher: publisher}
}

func (s *CardService) requireColumnMember(ctx context.Context, userID, columnID uuid.UUID) (domain.Column, error) {
	column, err := s.columns.GetByID(ctx, columnID)
	if err != nil {
		return domain.Column{}, err
	}
	if err := requireMember(ctx, s.boards, column.BoardID, userID); err != nil {
		return domain.Column{}, err
	}
	return column, nil
}

func (s *CardService) Create(ctx context.Context, userID, columnID uuid.UUID, title, description string) (domain.Card, error) {
	column, err := s.requireColumnMember(ctx, userID, columnID)
	if err != nil {
		return domain.Card{}, err
	}
	if err := validateTitle("title", title, cardTitleMaxLen); err != nil {
		return domain.Card{}, err
	}

	var card domain.Card
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.cards.LockForReorder(ctx, columnID); err != nil {
			return err
		}
		maxOrder, err := s.cards.MaxOrder(ctx, columnID)
		if err != nil {
			return err
		}
		card, err = s.cards.Create(ctx, domain.Card{
			ColumnID:    columnID,
			Title:       title,
			Description: description,
			OrderNum:    appendOrder(maxOrder),
			AuthorID:    userID,
		})
		return err
	})
	if err != nil {
		return domain.Card{}, err
	}
	s.publisher.Publish(ctx, realtime.NewEvent(realtime.EventCardCreated, column.BoardID, realtime.NewCardPayload(card)))
	return card, nil
}

func (s *CardService) Get(ctx context.Context, userID, cardID uuid.UUID) (domain.Card, error) {
	card, err := s.cards.GetByID(ctx, cardID)
	if err != nil {
		return domain.Card{}, err
	}
	if _, err := s.requireColumnMember(ctx, userID, card.ColumnID); err != nil {
		return domain.Card{}, err
	}
	return card, nil
}

func (s *CardService) ListByColumn(ctx context.Context, userID, columnID uuid.UUID) ([]domain.Card, error) {
	if _, err := s.requireColumnMember(ctx, userID, columnID); err != nil {
		return nil, err
	}
	return s.cards.ListByColumn(ctx, columnID)
}

// UpdateContent applies a partial update: a nil title or description
// leaves that field unchanged, matching UpdateCardRequest in openapi.yaml
// (both properties optional).
func (s *CardService) UpdateContent(ctx context.Context, userID, cardID uuid.UUID, title, description *string) (domain.Card, error) {
	card, err := s.cards.GetByID(ctx, cardID)
	if err != nil {
		return domain.Card{}, err
	}
	column, err := s.requireColumnMember(ctx, userID, card.ColumnID)
	if err != nil {
		return domain.Card{}, err
	}

	newTitle := card.Title
	if title != nil {
		newTitle = *title
	}
	newDescription := card.Description
	if description != nil {
		newDescription = *description
	}
	if err := validateTitle("title", newTitle, cardTitleMaxLen); err != nil {
		return domain.Card{}, err
	}
	updated, err := s.cards.UpdateContent(ctx, cardID, newTitle, newDescription)
	if err != nil {
		return domain.Card{}, err
	}
	s.publisher.Publish(ctx, realtime.NewEvent(realtime.EventCardUpdated, column.BoardID, realtime.NewCardPayload(updated)))
	return updated, nil
}

func (s *CardService) Delete(ctx context.Context, userID, cardID uuid.UUID) error {
	card, err := s.cards.GetByID(ctx, cardID)
	if err != nil {
		return err
	}
	column, err := s.requireColumnMember(ctx, userID, card.ColumnID)
	if err != nil {
		return err
	}
	if err := s.cards.Delete(ctx, cardID); err != nil {
		return err
	}
	s.publisher.Publish(ctx, realtime.NewEvent(realtime.EventCardDeleted, column.BoardID, realtime.NewCardPayload(card)))
	return nil
}

// Move relocates a card into targetColumnID (which may equal its current
// column) to sit right after prevCardID and right before nextCardID
// (either may be nil for "top"/"bottom" of the list; prevCardID's order_num
// must precede nextCardID's). The new order_num is the midpoint of its new
// neighbors; if that midpoint has collapsed into one of them (ADR 004), the
// whole target column's cards are renumbered with a fresh orderStep gap
// first.
func (s *CardService) Move(ctx context.Context, userID, cardID, targetColumnID uuid.UUID, prevCardID, nextCardID *uuid.UUID) (domain.Card, error) {
	card, err := s.cards.GetByID(ctx, cardID)
	if err != nil {
		return domain.Card{}, err
	}
	targetColumn, err := s.requireColumnMember(ctx, userID, targetColumnID)
	if err != nil {
		return domain.Card{}, err
	}
	if card.ColumnID != targetColumnID {
		sourceColumn, err := s.columns.GetByID(ctx, card.ColumnID)
		if err != nil {
			return domain.Card{}, err
		}
		if sourceColumn.BoardID != targetColumn.BoardID {
			return domain.Card{}, domain.NewValidationError("target_column_id", "must belong to the same board as the card")
		}
	}

	var moved domain.Card
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.cards.LockForReorder(ctx, targetColumnID); err != nil {
			return err
		}

		var prevOrder, nextOrder *float64
		if prevCardID != nil {
			prev, err := s.cards.GetByID(ctx, *prevCardID)
			if err != nil {
				return err
			}
			if prev.ColumnID != targetColumnID {
				return domain.NewValidationError("prev_card_id", "must belong to the target column")
			}
			prevOrder = &prev.OrderNum
		}
		if nextCardID != nil {
			next, err := s.cards.GetByID(ctx, *nextCardID)
			if err != nil {
				return err
			}
			if next.ColumnID != targetColumnID {
				return domain.NewValidationError("next_card_id", "must belong to the target column")
			}
			nextOrder = &next.OrderNum
		}

		siblings, err := s.cards.ListByColumn(ctx, targetColumnID)
		if err != nil {
			return err
		}
		orderedSiblings := make([]orderedSibling, len(siblings))
		for i, c := range siblings {
			orderedSiblings[i] = orderedSibling{ID: c.ID, OrderNum: c.OrderNum}
		}

		newOrder, err := resolveMoveOrder(moveOrderRequest{
			MovingID:  cardID,
			PrevID:    prevCardID,
			PrevOrder: prevOrder,
			NextOrder: nextOrder,
			PrevField: "prev_card_id",
			Siblings:  orderedSiblings,
			UpdateOrder: func(id uuid.UUID, order float64) error {
				_, err := s.cards.UpdateOrder(ctx, id, targetColumnID, order)
				return err
			},
		})
		if err != nil {
			return err
		}

		moved, err = s.cards.UpdateOrder(ctx, cardID, targetColumnID, newOrder)
		return err
	})
	if err != nil {
		return domain.Card{}, err
	}
	s.publisher.Publish(ctx, realtime.NewEvent(realtime.EventCardMoved, targetColumn.BoardID, realtime.NewCardPayload(moved)))
	return moved, nil
}
