package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

func newBoardServiceForTest() (*BoardService, *memBoardRepo, *memUserRepo) {
	boards := newMemBoardRepo()
	users := newMemUserRepo()
	return NewBoardService(boards, users, memTxManager{}), boards, users
}

func TestBoardService_Create(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr error
	}{
		{"valid title", "My Board", nil},
		{"empty title", "", domain.ErrValidation},
		{"title too long", strings.Repeat("a", boardTitleMaxLen+1), domain.ErrValidation},
		{"title at max length", strings.Repeat("a", boardTitleMaxLen), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, boards, _ := newBoardServiceForTest()
			userID := uuid.New()

			board, err := svc.Create(context.Background(), userID, tt.title)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Create() unexpected error: %v", err)
			}
			if board.OwnerID != userID {
				t.Errorf("board.OwnerID = %v, want %v", board.OwnerID, userID)
			}
			isMember, _ := boards.IsMember(context.Background(), board.ID, userID)
			if !isMember {
				t.Error("creator was not added as a board member")
			}
		})
	}
}

func TestBoardService_Get(t *testing.T) {
	svc, boards, _ := newBoardServiceForTest()
	owner := uuid.New()
	stranger := uuid.New()
	board, _ := svc.Create(context.Background(), owner, "Board")

	t.Run("member can get", func(t *testing.T) {
		got, err := svc.Get(context.Background(), owner, board.ID)
		if err != nil {
			t.Fatalf("Get() unexpected error: %v", err)
		}
		if got.ID != board.ID {
			t.Errorf("got board ID = %v, want %v", got.ID, board.ID)
		}
	})

	t.Run("non-member forbidden", func(t *testing.T) {
		_, err := svc.Get(context.Background(), stranger, board.ID)
		if !errors.Is(err, domain.ErrNotBoardMember) {
			t.Errorf("Get() error = %v, want ErrNotBoardMember", err)
		}
	})

	t.Run("unknown board not found", func(t *testing.T) {
		_, err := svc.Get(context.Background(), owner, uuid.New())
		if !errors.Is(err, domain.ErrBoardNotFound) {
			t.Errorf("Get() error = %v, want ErrBoardNotFound", err)
		}
	})

	_ = boards
}

func TestBoardService_UpdateTitle(t *testing.T) {
	svc, _, _ := newBoardServiceForTest()
	owner := uuid.New()
	stranger := uuid.New()
	board, _ := svc.Create(context.Background(), owner, "Board")

	t.Run("member can update", func(t *testing.T) {
		updated, err := svc.UpdateTitle(context.Background(), owner, board.ID, "New title")
		if err != nil {
			t.Fatalf("UpdateTitle() unexpected error: %v", err)
		}
		if updated.Title != "New title" {
			t.Errorf("Title = %q, want %q", updated.Title, "New title")
		}
	})

	t.Run("non-member forbidden", func(t *testing.T) {
		_, err := svc.UpdateTitle(context.Background(), stranger, board.ID, "Another title")
		if !errors.Is(err, domain.ErrNotBoardMember) {
			t.Errorf("UpdateTitle() error = %v, want ErrNotBoardMember", err)
		}
	})

	t.Run("unknown board not found", func(t *testing.T) {
		_, err := svc.UpdateTitle(context.Background(), owner, uuid.New(), "Title")
		if !errors.Is(err, domain.ErrBoardNotFound) {
			t.Errorf("UpdateTitle() error = %v, want ErrBoardNotFound", err)
		}
	})
}

func TestBoardService_InviteMember(t *testing.T) {
	svc, boards, users := newBoardServiceForTest()
	owner := uuid.New()
	board, _ := svc.Create(context.Background(), owner, "Board")

	invitee := domain.User{ID: uuid.New(), Email: "invitee@example.com", Name: "Invitee"}
	users.users[invitee.ID] = invitee

	t.Run("unknown email is a validation error", func(t *testing.T) {
		_, err := svc.InviteMember(context.Background(), owner, board.ID, "ghost@example.com")
		if !errors.Is(err, domain.ErrValidation) {
			t.Errorf("InviteMember() error = %v, want ErrValidation", err)
		}
	})

	t.Run("inviter must be a board member", func(t *testing.T) {
		_, err := svc.InviteMember(context.Background(), uuid.New(), board.ID, invitee.Email)
		if !errors.Is(err, domain.ErrNotBoardMember) {
			t.Errorf("InviteMember() error = %v, want ErrNotBoardMember", err)
		}
	})

	t.Run("unknown board not found", func(t *testing.T) {
		_, err := svc.InviteMember(context.Background(), owner, uuid.New(), invitee.Email)
		if !errors.Is(err, domain.ErrBoardNotFound) {
			t.Errorf("InviteMember() error = %v, want ErrBoardNotFound", err)
		}
	})

	t.Run("successful invite adds membership", func(t *testing.T) {
		member, err := svc.InviteMember(context.Background(), owner, board.ID, invitee.Email)
		if err != nil {
			t.Fatalf("InviteMember() unexpected error: %v", err)
		}
		if member.UserID != invitee.ID {
			t.Errorf("member.UserID = %v, want %v", member.UserID, invitee.ID)
		}
		isMember, _ := boards.IsMember(context.Background(), board.ID, invitee.ID)
		if !isMember {
			t.Error("invitee was not added as a board member")
		}
	})

	t.Run("re-inviting an existing member is a validation error", func(t *testing.T) {
		_, err := svc.InviteMember(context.Background(), owner, board.ID, invitee.Email)
		if !errors.Is(err, domain.ErrValidation) {
			t.Errorf("InviteMember() error = %v, want ErrValidation", err)
		}
	})
}

func TestBoardService_Delete(t *testing.T) {
	svc, _, _ := newBoardServiceForTest()
	owner := uuid.New()
	stranger := uuid.New()
	board, _ := svc.Create(context.Background(), owner, "Board")

	if err := svc.Delete(context.Background(), stranger, board.ID); !errors.Is(err, domain.ErrNotBoardMember) {
		t.Errorf("Delete() by non-member error = %v, want ErrNotBoardMember", err)
	}

	if err := svc.Delete(context.Background(), owner, uuid.New()); !errors.Is(err, domain.ErrBoardNotFound) {
		t.Errorf("Delete() of unknown board error = %v, want ErrBoardNotFound", err)
	}

	if err := svc.Delete(context.Background(), owner, board.ID); err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	if _, err := svc.Get(context.Background(), owner, board.ID); !errors.Is(err, domain.ErrBoardNotFound) {
		t.Errorf("Get() after delete error = %v, want ErrBoardNotFound", err)
	}
}
