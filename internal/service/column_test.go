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

// testBoard wires up a board+member pair directly against the mock repos,
// bypassing BoardService, so column/card tests can set up fixtures without
// depending on board_test.go's assertions.
func testBoard(t *testing.T, boards *memBoardRepo, userID uuid.UUID) domain.Board {
	t.Helper()
	board, err := boards.Create(context.Background(), domain.Board{Title: "Board", OwnerID: userID})
	if err != nil {
		t.Fatalf("failed to create fixture board: %v", err)
	}
	if _, err := boards.AddMember(context.Background(), board.ID, userID); err != nil {
		t.Fatalf("failed to add fixture board member: %v", err)
	}
	return board
}

func newColumnServiceForTest() (*ColumnService, *memColumnRepo, *memBoardRepo) {
	columns := newMemColumnRepo()
	boards := newMemBoardRepo()
	return NewColumnService(columns, boards, memTxManager{}, realtime.NoopPublisher{}), columns, boards
}

func TestColumnService_Create(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr error
	}{
		{"valid title", "To Do", nil},
		{"empty title", "", domain.ErrValidation},
		{"title too long", strings.Repeat("a", columnTitleMaxLen+1), domain.ErrValidation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, boards := newColumnServiceForTest()
			userID := uuid.New()
			board := testBoard(t, boards, userID)

			_, err := svc.Create(context.Background(), userID, board.ID, tt.title)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Create() unexpected error: %v", err)
			}
		})
	}
}

func TestColumnService_Create_AppendsToEnd(t *testing.T) {
	svc, _, boards := newColumnServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)

	first, err := svc.Create(context.Background(), userID, board.ID, "To Do")
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	second, err := svc.Create(context.Background(), userID, board.ID, "Done")
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if second.OrderNum <= first.OrderNum {
		t.Errorf("second.OrderNum = %v, want > first.OrderNum %v", second.OrderNum, first.OrderNum)
	}
}

func TestColumnService_Create_NonMemberForbidden(t *testing.T) {
	svc, _, boards := newColumnServiceForTest()
	owner := uuid.New()
	stranger := uuid.New()
	board := testBoard(t, boards, owner)

	_, err := svc.Create(context.Background(), stranger, board.ID, "To Do")
	if !errors.Is(err, domain.ErrNotBoardMember) {
		t.Errorf("Create() error = %v, want ErrNotBoardMember", err)
	}
}

func TestColumnService_Move(t *testing.T) {
	svc, columns, boards := newColumnServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)

	a, _ := svc.Create(context.Background(), userID, board.ID, "A") // order 1000
	b, _ := svc.Create(context.Background(), userID, board.ID, "B") // order 2000
	c, _ := svc.Create(context.Background(), userID, board.ID, "C") // order 3000

	t.Run("move to start", func(t *testing.T) {
		moved, err := svc.Move(context.Background(), userID, c.ID, nil, &a.ID)
		if err != nil {
			t.Fatalf("Move() unexpected error: %v", err)
		}
		if moved.OrderNum >= a.OrderNum {
			t.Errorf("moved.OrderNum = %v, want < a.OrderNum %v", moved.OrderNum, a.OrderNum)
		}
	})

	t.Run("move between neighbors", func(t *testing.T) {
		moved, err := svc.Move(context.Background(), userID, c.ID, &a.ID, &b.ID)
		if err != nil {
			t.Fatalf("Move() unexpected error: %v", err)
		}
		fresh, _ := columns.GetByID(context.Background(), a.ID)
		freshB, _ := columns.GetByID(context.Background(), b.ID)
		if !(fresh.OrderNum < moved.OrderNum && moved.OrderNum < freshB.OrderNum) {
			t.Errorf("expected a.OrderNum(%v) < moved.OrderNum(%v) < b.OrderNum(%v)", fresh.OrderNum, moved.OrderNum, freshB.OrderNum)
		}
	})

	t.Run("neighbor from a different board is rejected", func(t *testing.T) {
		otherBoard := testBoard(t, boards, userID)
		foreign, _ := svc.Create(context.Background(), userID, otherBoard.ID, "Foreign")

		_, err := svc.Move(context.Background(), userID, c.ID, &foreign.ID, nil)
		if !errors.Is(err, domain.ErrValidation) {
			t.Errorf("Move() error = %v, want ErrValidation", err)
		}
	})
}

// TestColumnService_Move_RenumberOnCollapse is the ADR 004-mandated
// edge-case test: moving a column to the midpoint between two neighbors
// whose order_num has already collapsed to float64's precision limit must
// trigger a local renumbering pass — restoring a healthy orderStep gap —
// rather than silently violating prev.OrderNum < moved.OrderNum <
// next.OrderNum.
func TestColumnService_Move_RenumberOnCollapse(t *testing.T) {
	svc, columns, boards := newColumnServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)

	low, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "low", OrderNum: 1000})
	high, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "high", OrderNum: 1000 + 1e-9})
	mover, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "mover", OrderNum: 5000})

	if !orderCollapsed(low.OrderNum, high.OrderNum) {
		t.Fatalf("fixture is not actually collapsed: low=%v high=%v", low.OrderNum, high.OrderNum)
	}

	moved, err := svc.Move(context.Background(), userID, mover.ID, &low.ID, &high.ID)
	if err != nil {
		t.Fatalf("Move() unexpected error: %v", err)
	}

	freshLow, _ := columns.GetByID(context.Background(), low.ID)
	freshHigh, _ := columns.GetByID(context.Background(), high.ID)
	if !(freshLow.OrderNum < moved.OrderNum && moved.OrderNum < freshHigh.OrderNum) {
		t.Fatalf("invariant violated after renumbering: low(%v) < moved(%v) < high(%v)", freshLow.OrderNum, moved.OrderNum, freshHigh.OrderNum)
	}
	if freshHigh.OrderNum-freshLow.OrderNum < orderStep {
		t.Errorf("expected renumbering to restore a healthy gap, got low=%v high=%v", freshLow.OrderNum, freshHigh.OrderNum)
	}
}

// TestColumnService_Move_RejectsSwappedPrevNext mirrors
// TestCardService_Move_RejectsSwappedPrevNext for columns.
func TestColumnService_Move_RejectsSwappedPrevNext(t *testing.T) {
	svc, columns, boards := newColumnServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)

	low, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "low", OrderNum: 1000})
	mid, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "mid", OrderNum: 2000})
	high, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "high", OrderNum: 3000})

	_, err := svc.Move(context.Background(), userID, mid.ID, &high.ID, &low.ID)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Move() error = %v, want ErrValidation", err)
	}

	freshLow, _ := columns.GetByID(context.Background(), low.ID)
	freshHigh, _ := columns.GetByID(context.Background(), high.ID)
	if freshLow.OrderNum != 1000 || freshHigh.OrderNum != 3000 {
		t.Errorf("siblings changed after rejected move: low=%v high=%v, want unchanged 1000/3000", freshLow.OrderNum, freshHigh.OrderNum)
	}
}

// TestColumnService_Move_SequentialSameSlotDoesNotCollide mirrors
// TestCardService_Move_SequentialSameSlotDoesNotCollide for columns: two
// Move calls targeting the same stale (prevColumnID, nextColumnID) pair
// must not leave two columns with the same order_num.
func TestColumnService_Move_SequentialSameSlotDoesNotCollide(t *testing.T) {
	svc, columns, boards := newColumnServiceForTest()
	userID := uuid.New()
	board := testBoard(t, boards, userID)

	p, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "P", OrderNum: 1000})
	n, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "N", OrderNum: 2000})
	a, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "A", OrderNum: 3000})
	b, _ := columns.Create(context.Background(), domain.Column{BoardID: board.ID, Title: "B", OrderNum: 4000})

	if _, err := svc.Move(context.Background(), userID, a.ID, &p.ID, &n.ID); err != nil {
		t.Fatalf("first Move() unexpected error: %v", err)
	}
	movedB, err := svc.Move(context.Background(), userID, b.ID, &p.ID, &n.ID)
	if err != nil {
		t.Fatalf("second Move() unexpected error: %v", err)
	}

	freshA, err := columns.GetByID(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("failed to re-fetch column A: %v", err)
	}
	if freshA.OrderNum == movedB.OrderNum {
		t.Fatalf("second Move() collided with first: both column A and column B hold order_num %v", movedB.OrderNum)
	}
}
