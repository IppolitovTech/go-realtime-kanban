package service

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

// The mocks below are minimal in-memory stand-ins for the repository
// interfaces, used to table-test the service layer without a real
// database — see roadmap.md, Stage 1, "Table-driven tests for the service
// layer (with an in-memory repository mock)".

type memTxManager struct{}

func (memTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type memBoardRepo struct {
	mu      sync.Mutex
	boards  map[uuid.UUID]domain.Board
	members map[uuid.UUID]map[uuid.UUID]domain.BoardMember // boardID -> userID -> member
}

func newMemBoardRepo() *memBoardRepo {
	return &memBoardRepo{
		boards:  map[uuid.UUID]domain.Board{},
		members: map[uuid.UUID]map[uuid.UUID]domain.BoardMember{},
	}
}

func (r *memBoardRepo) Create(ctx context.Context, board domain.Board) (domain.Board, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	board.ID = uuid.New()
	r.boards[board.ID] = board
	return board, nil
}

func (r *memBoardRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Board, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.boards[id]
	if !ok {
		return domain.Board{}, domain.ErrBoardNotFound
	}
	return b, nil
}

func (r *memBoardRepo) ListByMember(ctx context.Context, userID uuid.UUID) ([]domain.Board, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Board
	for boardID, members := range r.members {
		if _, ok := members[userID]; ok {
			out = append(out, r.boards[boardID])
		}
	}
	return out, nil
}

func (r *memBoardRepo) UpdateTitle(ctx context.Context, id uuid.UUID, title string) (domain.Board, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.boards[id]
	if !ok {
		return domain.Board{}, domain.ErrBoardNotFound
	}
	b.Title = title
	r.boards[id] = b
	return b, nil
}

func (r *memBoardRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.boards, id)
	delete(r.members, id)
	return nil
}

func (r *memBoardRepo) AddMember(ctx context.Context, boardID, userID uuid.UUID) (domain.BoardMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.members[boardID] == nil {
		r.members[boardID] = map[uuid.UUID]domain.BoardMember{}
	}
	m := domain.BoardMember{BoardID: boardID, UserID: userID}
	r.members[boardID][userID] = m
	return m, nil
}

func (r *memBoardRepo) ListMembers(ctx context.Context, boardID uuid.UUID) ([]domain.BoardMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.BoardMember
	for _, m := range r.members[boardID] {
		out = append(out, m)
	}
	return out, nil
}

func (r *memBoardRepo) IsMember(ctx context.Context, boardID, userID uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.members[boardID][userID]
	return ok, nil
}

type memUserRepo struct {
	mu    sync.Mutex
	users map[uuid.UUID]domain.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{users: map[uuid.UUID]domain.User{}}
}

func (r *memUserRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *memUserRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

// Create mirrors the real repository's unique-email constraint (see
// internal/repository/postgres/user.go), so AuthService.Register's
// duplicate-email path is exercised the same way against both.
func (r *memUserRepo) Create(ctx context.Context, user domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == user.Email {
			return domain.User{}, domain.ErrEmailTaken
		}
	}
	user.ID = uuid.New()
	r.users[user.ID] = user
	return user, nil
}

type memColumnRepo struct {
	mu      sync.Mutex
	columns map[uuid.UUID]domain.Column
}

func newMemColumnRepo() *memColumnRepo {
	return &memColumnRepo{columns: map[uuid.UUID]domain.Column{}}
}

func (r *memColumnRepo) Create(ctx context.Context, column domain.Column) (domain.Column, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	column.ID = uuid.New()
	r.columns[column.ID] = column
	return column, nil
}

func (r *memColumnRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Column, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.columns[id]
	if !ok {
		return domain.Column{}, domain.ErrColumnNotFound
	}
	return c, nil
}

func (r *memColumnRepo) ListByBoard(ctx context.Context, boardID uuid.UUID) ([]domain.Column, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Column
	for _, c := range r.columns {
		if c.BoardID == boardID {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OrderNum != out[j].OrderNum {
			return out[i].OrderNum < out[j].OrderNum
		}
		return out[i].ID.String() < out[j].ID.String()
	})
	return out, nil
}

func (r *memColumnRepo) UpdateTitle(ctx context.Context, id uuid.UUID, title string) (domain.Column, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.columns[id]
	if !ok {
		return domain.Column{}, domain.ErrColumnNotFound
	}
	c.Title = title
	r.columns[id] = c
	return c, nil
}

func (r *memColumnRepo) UpdateOrder(ctx context.Context, id uuid.UUID, orderNum float64) (domain.Column, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.columns[id]
	if !ok {
		return domain.Column{}, domain.ErrColumnNotFound
	}
	c.OrderNum = orderNum
	r.columns[id] = c
	return c, nil
}

func (r *memColumnRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.columns, id)
	return nil
}

func (r *memColumnRepo) MaxOrder(ctx context.Context, boardID uuid.UUID) (float64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var max float64
	for _, c := range r.columns {
		if c.BoardID == boardID && c.OrderNum > max {
			max = c.OrderNum
		}
	}
	return max, nil
}

func (r *memColumnRepo) LockForReorder(ctx context.Context, boardID uuid.UUID) error {
	return nil
}

type memCardRepo struct {
	mu    sync.Mutex
	cards map[uuid.UUID]domain.Card
}

func newMemCardRepo() *memCardRepo {
	return &memCardRepo{cards: map[uuid.UUID]domain.Card{}}
}

func (r *memCardRepo) Create(ctx context.Context, card domain.Card) (domain.Card, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	card.ID = uuid.New()
	r.cards[card.ID] = card
	return card, nil
}

func (r *memCardRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Card, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.cards[id]
	if !ok {
		return domain.Card{}, domain.ErrCardNotFound
	}
	return c, nil
}

func (r *memCardRepo) ListByColumn(ctx context.Context, columnID uuid.UUID) ([]domain.Card, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Card
	for _, c := range r.cards {
		if c.ColumnID == columnID {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OrderNum != out[j].OrderNum {
			return out[i].OrderNum < out[j].OrderNum
		}
		return out[i].ID.String() < out[j].ID.String()
	})
	return out, nil
}

func (r *memCardRepo) UpdateContent(ctx context.Context, id uuid.UUID, title, description string) (domain.Card, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.cards[id]
	if !ok {
		return domain.Card{}, domain.ErrCardNotFound
	}
	c.Title = title
	c.Description = description
	r.cards[id] = c
	return c, nil
}

func (r *memCardRepo) UpdateOrder(ctx context.Context, id, columnID uuid.UUID, orderNum float64) (domain.Card, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.cards[id]
	if !ok {
		return domain.Card{}, domain.ErrCardNotFound
	}
	c.ColumnID = columnID
	c.OrderNum = orderNum
	r.cards[id] = c
	return c, nil
}

func (r *memCardRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cards, id)
	return nil
}

func (r *memCardRepo) MaxOrder(ctx context.Context, columnID uuid.UUID) (float64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var max float64
	for _, c := range r.cards {
		if c.ColumnID == columnID && c.OrderNum > max {
			max = c.OrderNum
		}
	}
	return max, nil
}

func (r *memCardRepo) LockForReorder(ctx context.Context, columnID uuid.UUID) error {
	return nil
}
