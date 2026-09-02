//go:build integration

package service_test

// Real-database integration tests for CardService's order_num placement
// under concurrency — see roadmap.md, Stage 1: "Integration test for
// concurrent card moves within a single column (multiple goroutines, a
// real DB) — verifying that the advisory lock protects the invariant
// prev < order_num < next" (ADR 004).
//
// Requires a reachable Postgres with the schema migrated — run via:
//   make test-integration

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/realtime"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository/postgres"
	"github.com/IppolitovTech/go-realtime-kanban/internal/service"
)

// stubUserID matches the seed row inserted by migration 000002 — see
// architecture.md, "User stub in Stage 1".
var stubUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func testDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://kanban:kanban@localhost:5432/kanban?sslmode=disable"
}

// newIntegrationFixture connects to a real Postgres, wires up real
// repositories + services, and creates a board/column owned by the seed
// user for the test to run against. Returns a cleanup func that deletes
// the board (cascading to its columns/cards) and closes the pool.
func newIntegrationFixture(t *testing.T) (cards *service.CardService, cardRepo *postgres.CardRepository, columnID uuid.UUID, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("database not reachable (is `make up` running?): %v", err)
	}

	boardRepo := postgres.NewBoardRepository(pool)
	columnRepo := postgres.NewColumnRepository(pool)
	cardRepo = postgres.NewCardRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	tx := postgres.NewTxManager(pool)

	boardSvc := service.NewBoardService(boardRepo, userRepo, tx)
	columnSvc := service.NewColumnService(columnRepo, boardRepo, tx, realtime.NoopPublisher{})
	cardSvc := service.NewCardService(cardRepo, columnRepo, boardRepo, tx, realtime.NoopPublisher{})

	board, err := boardSvc.Create(ctx, stubUserID, "Integration test board")
	if err != nil {
		t.Fatalf("failed to create fixture board: %v", err)
	}
	column, err := columnSvc.Create(ctx, stubUserID, board.ID, "Integration test column")
	if err != nil {
		t.Fatalf("failed to create fixture column: %v", err)
	}

	cleanup = func() {
		_ = boardSvc.Delete(context.Background(), stubUserID, board.ID)
		pool.Close()
	}
	return cardSvc, cardRepo, column.ID, cleanup
}

// TestCardService_ConcurrentCreate_NoDuplicateOrder appends cards to the
// same column from many goroutines at once. Create reads MaxOrder and
// writes the new card's order_num under the same per-column advisory lock
// used by Move (ADR 004); without it, two concurrent appends could read
// the same max and produce two cards with an identical order_num.
func TestCardService_ConcurrentCreate_NoDuplicateOrder(t *testing.T) {
	cards, _, columnID, cleanup := newIntegrationFixture(t)
	defer cleanup()

	const goroutines = 20
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make([]domain.Card, goroutines)
	errs := make([]error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = cards.Create(ctx, stubUserID, columnID, "concurrent card", "")
		}(i)
	}
	wg.Wait()

	seen := make(map[float64]int)
	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: Create() unexpected error: %v", i, err)
		}
		seen[results[i].OrderNum]++
	}
	for orderNum, count := range seen {
		if count > 1 {
			t.Errorf("order_num %v was assigned to %d cards, want unique per card", orderNum, count)
		}
	}
}

// TestCardService_ConcurrentMove_PreservesOrderInvariant sets up two
// neighbor cards whose order_num has already collapsed to float64's
// precision limit (ADR 004), then has many goroutines concurrently Move
// distinct cards to be inserted between them. Each such Move must trigger
// (or safely observe another goroutine's already-completed) local
// renumbering; the advisory lock must prevent two renumbering passes from
// interleaving and corrupting the column. After all moves complete, no
// card may violate prev.OrderNum < card.OrderNum < next.OrderNum.
func TestCardService_ConcurrentMove_PreservesOrderInvariant(t *testing.T) {
	cards, cardRepo, columnID, cleanup := newIntegrationFixture(t)
	defer cleanup()

	ctx := context.Background()

	// Seed the two boundary cards with an already-collapsed order_num gap
	// directly through the repository (bypassing CardService.Create, which
	// always appends to the end) — mirrors the collapse fixture used in
	// column_test.go/card_test.go for the mock-based unit tests.
	low, err := cardRepo.Create(ctx, domain.Card{ColumnID: columnID, Title: "low", OrderNum: 1000, AuthorID: stubUserID})
	if err != nil {
		t.Fatalf("failed to create low boundary card: %v", err)
	}
	high, err := cardRepo.Create(ctx, domain.Card{ColumnID: columnID, Title: "high", OrderNum: 1000 + 1e-9, AuthorID: stubUserID})
	if err != nil {
		t.Fatalf("failed to create high boundary card: %v", err)
	}

	const goroutines = 15
	moverIDs := make([]uuid.UUID, goroutines)
	for i := range moverIDs {
		card, err := cards.Create(ctx, stubUserID, columnID, "mover", "")
		if err != nil {
			t.Fatalf("failed to create mover card %d: %v", i, err)
		}
		moverIDs[i] = card.ID
	}

	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	for i, id := range moverIDs {
		wg.Add(1)
		go func(i int, cardID uuid.UUID) {
			defer wg.Done()
			_, errs[i] = cards.Move(ctx, stubUserID, cardID, columnID, &low.ID, &high.ID)
		}(i, id)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: Move() unexpected error: %v", i, err)
		}
	}

	finalLow, err := cards.Get(ctx, stubUserID, low.ID)
	if err != nil {
		t.Fatalf("failed to re-fetch low boundary card: %v", err)
	}
	finalHigh, err := cards.Get(ctx, stubUserID, high.ID)
	if err != nil {
		t.Fatalf("failed to re-fetch high boundary card: %v", err)
	}
	if finalLow.OrderNum >= finalHigh.OrderNum {
		t.Fatalf("boundary cards themselves are out of order after concurrent moves: low=%v high=%v", finalLow.OrderNum, finalHigh.OrderNum)
	}

	seenOrder := make(map[float64]uuid.UUID)
	for _, id := range moverIDs {
		card, err := cards.Get(ctx, stubUserID, id)
		if err != nil {
			t.Fatalf("failed to re-fetch mover card %s: %v", id, err)
		}
		if card.ColumnID != columnID {
			t.Errorf("mover card %s ended up in column %s, want %s", id, card.ColumnID, columnID)
		}
		if !(finalLow.OrderNum < card.OrderNum && card.OrderNum < finalHigh.OrderNum) {
			t.Errorf("invariant violated for card %s: low(%v) < order(%v) < high(%v)", id, finalLow.OrderNum, card.OrderNum, finalHigh.OrderNum)
		}
		if other, ok := seenOrder[card.OrderNum]; ok {
			t.Errorf("order_num %v assigned to both card %s and card %s, want unique per card", card.OrderNum, other, id)
		}
		seenOrder[card.OrderNum] = id
	}
}
