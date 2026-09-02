package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/realtime"
)

func newCardServiceForTest() (*CardService, *memCardRepo, *memColumnRepo, *memBoardRepo) {
	cards := newMemCardRepo()
	columns := newMemColumnRepo()
	boards := newMemBoardRepo()
	return NewCardService(cards, columns, boards, memTxManager{}, realtime.NoopPublisher{}), cards, columns, boards
}

func TestCardService_Create(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr error
	}{
		{"valid title", "Fix bug", nil},
		{"empty title", "", domain.ErrValidation},
		{"title too long", strings.Repeat("a", cardTitleMaxLen+1), domain.ErrValidation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, columns, boards := newCardServiceForTest()
			userID := uuid.New()
			board := testBoard(t, boards, userID)
			column, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "To Do", OrderNum: 1000})

			card, err := svc.Create(context.Background(), userID, column.ID, tt.title, "desc")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Create() unexpected error: %v", err)
			}
			if card.AuthorID != userID {
				t.Errorf("card.AuthorID = %v, want %v", card.AuthorID, userID)
			}
		})
	}
}

func TestCardService_Create_NonMemberForbidden(t *testing.T) {
	svc, _, columns, boards := newCardServiceForTest()
	owner := uuid.New()
	stranger := uuid.New()
	board := testBoard(t, boards, owner)
	column, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "To Do", OrderNum: 1000})

	_, err := svc.Create(context.Background(), stranger, column.ID, "title", "")
	if !errors.Is(err, domain.ErrNotBoardMember) {
		t.Errorf("Create() error = %v, want ErrNotBoardMember", err)
	}
}

func TestCardService_UpdateContent_PartialUpdate(t *testing.T) {
	svc, _, columns, boards := newCardServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)
	column, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "To Do", OrderNum: 1000})
	card, _ := svc.Create(context.Background(), userID, column.ID, "Original title", "Original description")

	newTitle := "Updated title"
	updated, err := svc.UpdateContent(context.Background(), userID, card.ID, &newTitle, nil)
	if err != nil {
		t.Fatalf("UpdateContent() unexpected error: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("Title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Description != "Original description" {
		t.Errorf("Description = %q, want unchanged %q", updated.Description, "Original description")
	}
}

func TestCardService_Move_WithinColumn(t *testing.T) {
	svc, cards, columns, boards := newCardServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)
	column, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "To Do", OrderNum: 1000})

	a, _ := svc.Create(context.Background(), userID, column.ID, "A", "")
	b, _ := svc.Create(context.Background(), userID, column.ID, "B", "")
	c, _ := svc.Create(context.Background(), userID, column.ID, "C", "")

	moved, err := svc.Move(context.Background(), userID, c.ID, column.ID, &a.ID, &b.ID)
	if err != nil {
		t.Fatalf("Move() unexpected error: %v", err)
	}
	freshA, _ := cards.GetByID(context.Background(), a.ID)
	freshB, _ := cards.GetByID(context.Background(), b.ID)
	if !(freshA.OrderNum < moved.OrderNum && moved.OrderNum < freshB.OrderNum) {
		t.Errorf("expected a.OrderNum(%v) < moved.OrderNum(%v) < b.OrderNum(%v)", freshA.OrderNum, moved.OrderNum, freshB.OrderNum)
	}
}

func TestCardService_Move_AcrossColumns(t *testing.T) {
	svc, cards, columns, boards := newCardServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)
	source, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "To Do", OrderNum: 1000})
	target, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "Done", OrderNum: 2000})

	card, _ := svc.Create(context.Background(), userID, source.ID, "Task", "")

	moved, err := svc.Move(context.Background(), userID, card.ID, target.ID, nil, nil)
	if err != nil {
		t.Fatalf("Move() unexpected error: %v", err)
	}
	if moved.ColumnID != target.ID {
		t.Errorf("moved.ColumnID = %v, want %v", moved.ColumnID, target.ID)
	}

	remaining, _ := cards.ListByColumn(context.Background(), source.ID)
	if len(remaining) != 0 {
		t.Errorf("expected source column to be empty, got %d cards", len(remaining))
	}
}

func TestCardService_Move_RejectsCrossBoardTarget(t *testing.T) {
	svc, _, columns, boards := newCardServiceForTest()
	userID := uuid.New()
	boardA := testBoard(t, boards, userID)
	boardB := testBoard(t, boards, userID)
	source, _ := columns.Create(context.Background(), domain.Column{BoardID: boardA.ID, Title: "To Do", OrderNum: 1000})
	target, _ := columns.Create(context.Background(), domain.Column{BoardID: boardB.ID, Title: "Done", OrderNum: 1000})

	card, _ := svc.Create(context.Background(), userID, source.ID, "Task", "")

	_, err := svc.Move(context.Background(), userID, card.ID, target.ID, nil, nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("Move() error = %v, want ErrValidation", err)
	}
}

// TestCardService_Move_RenumberOnCollapse mirrors
// TestColumnService_Move_RenumberOnCollapse for cards — see ADR 004.
func TestCardService_Move_RenumberOnCollapse(t *testing.T) {
	svc, cards, columns, boards := newCardServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)
	column, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "To Do", OrderNum: 1000})

	low, _ := cards.Create(context.Background(), domain.Card{ColumnID: column.ID, Title: "low", OrderNum: 1000, AuthorID: userID})
	high, _ := cards.Create(context.Background(), domain.Card{ColumnID: column.ID, Title: "high", OrderNum: 1000 + 1e-9, AuthorID: userID})
	mover, _ := cards.Create(context.Background(), domain.Card{ColumnID: column.ID, Title: "mover", OrderNum: 5000, AuthorID: userID})

	moved, err := svc.Move(context.Background(), userID, mover.ID, column.ID, &low.ID, &high.ID)
	if err != nil {
		t.Fatalf("Move() unexpected error: %v", err)
	}

	freshLow, _ := cards.GetByID(context.Background(), low.ID)
	freshHigh, _ := cards.GetByID(context.Background(), high.ID)
	if !(freshLow.OrderNum < moved.OrderNum && moved.OrderNum < freshHigh.OrderNum) {
		t.Fatalf("invariant violated after renumbering: low(%v) < moved(%v) < high(%v)", freshLow.OrderNum, moved.OrderNum, freshHigh.OrderNum)
	}
	if freshHigh.OrderNum-freshLow.OrderNum < orderStep {
		t.Errorf("expected renumbering to restore a healthy gap, got low=%v high=%v", freshLow.OrderNum, freshHigh.OrderNum)
	}
}

// TestCardService_Move_RejectsSwappedPrevNext guards against a regression
// where a caller (e.g. stale drag-n-drop state) passes prevCardID/nextCardID
// in the wrong order: orderSlotOccupied's continue-conditions assume
// prev.OrderNum < next.OrderNum, so a swapped pair used to skip every
// sibling as "not occupying" and silently hand out a duplicate order_num.
func TestCardService_Move_RejectsSwappedPrevNext(t *testing.T) {
	svc, cards, columns, boards := newCardServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)
	column, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "To Do", OrderNum: 1000})

	low, _ := cards.Create(context.Background(), domain.Card{ColumnID: column.ID, Title: "low", OrderNum: 1000, AuthorID: userID})
	mid, _ := cards.Create(context.Background(), domain.Card{ColumnID: column.ID, Title: "mid", OrderNum: 2000, AuthorID: userID})
	high, _ := cards.Create(context.Background(), domain.Card{ColumnID: column.ID, Title: "high", OrderNum: 3000, AuthorID: userID})

	_, err := svc.Move(context.Background(), userID, mid.ID, column.ID, &high.ID, &low.ID)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Move() error = %v, want ErrValidation", err)
	}

	freshLow, _ := cards.GetByID(context.Background(), low.ID)
	freshHigh, _ := cards.GetByID(context.Background(), high.ID)
	if freshLow.OrderNum != 1000 || freshHigh.OrderNum != 3000 {
		t.Errorf("siblings changed after rejected move: low=%v high=%v, want unchanged 1000/3000", freshLow.OrderNum, freshHigh.OrderNum)
	}
}

// TestCardService_Move_SequentialSameSlotDoesNotCollide reproduces the race
// where two Move calls target the same stale (prevCardID, nextCardID) pair
// (e.g. two clients dragging different cards into the same visual gap
// before either has refreshed the board): the second Move must not land on
// the same order_num the first one just claimed, even though prevCardID's
// and nextCardID's own order_num values never change.
func TestCardService_Move_SequentialSameSlotDoesNotCollide(t *testing.T) {
	svc, cards, columns, boards := newCardServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)
	column, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "To Do", OrderNum: 1000})

	p, _ := cards.Create(context.Background(), domain.Card{ColumnID: column.ID, Title: "P", OrderNum: 1000, AuthorID: userID})
	n, _ := cards.Create(context.Background(), domain.Card{ColumnID: column.ID, Title: "N", OrderNum: 2000, AuthorID: userID})
	a, _ := cards.Create(context.Background(), domain.Card{ColumnID: column.ID, Title: "A", OrderNum: 3000, AuthorID: userID})
	b, _ := cards.Create(context.Background(), domain.Card{ColumnID: column.ID, Title: "B", OrderNum: 4000, AuthorID: userID})

	if _, err := svc.Move(context.Background(), userID, a.ID, column.ID, &p.ID, &n.ID); err != nil {
		t.Fatalf("first Move() unexpected error: %v", err)
	}
	movedB, err := svc.Move(context.Background(), userID, b.ID, column.ID, &p.ID, &n.ID)
	if err != nil {
		t.Fatalf("second Move() unexpected error: %v", err)
	}

	freshA, err := cards.GetByID(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("failed to re-fetch card A: %v", err)
	}
	if freshA.OrderNum == movedB.OrderNum {
		t.Fatalf("second Move() collided with first: both card A and card B hold order_num %v", movedB.OrderNum)
	}
}
