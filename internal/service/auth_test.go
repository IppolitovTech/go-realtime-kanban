package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/IppolitovTech/go-realtime-kanban/internal/auth"
	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

func newAuthServiceForTest() (*AuthService, *memUserRepo) {
	users := newMemUserRepo()
	return NewAuthService(users, []byte("test-secret"), time.Hour), users
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		userName string
		password string
		wantErr  error
	}{
		{"valid", "alice@example.com", "Alice", "password123", nil},
		{"invalid email", "not-an-email", "Alice", "password123", domain.ErrValidation},
		{"empty name", "alice@example.com", "", "password123", domain.ErrValidation},
		{"short password", "alice@example.com", "Alice", "short", domain.ErrValidation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newAuthServiceForTest()

			user, token, err := svc.Register(context.Background(), tt.email, tt.userName, tt.password)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Register() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Register() unexpected error: %v", err)
			}
			if user.Email != tt.email || user.Name != tt.userName {
				t.Errorf("Register() user = %+v, want email=%s name=%s", user, tt.email, tt.userName)
			}
			if user.PasswordHash == "" || user.PasswordHash == tt.password {
				t.Error("Register() did not store a hashed password")
			}
			if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.password)) != nil {
				t.Error("Register() stored hash does not match the given password")
			}
			if token == "" {
				t.Error("Register() returned an empty token")
			}
			if _, err := auth.Verify([]byte("test-secret"), token); err != nil {
				t.Errorf("Register() returned an unverifiable token: %v", err)
			}
		})
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	svc, _ := newAuthServiceForTest()
	ctx := context.Background()

	if _, _, err := svc.Register(ctx, "alice@example.com", "Alice", "password123"); err != nil {
		t.Fatalf("first Register() unexpected error: %v", err)
	}
	if _, _, err := svc.Register(ctx, "alice@example.com", "Alice Two", "password456"); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("second Register() error = %v, want ErrEmailTaken", err)
	}
}

func TestAuthService_Login(t *testing.T) {
	svc, _ := newAuthServiceForTest()
	ctx := context.Background()
	registered, _, err := svc.Register(ctx, "alice@example.com", "Alice", "password123")
	if err != nil {
		t.Fatalf("Register() unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{"correct credentials", "alice@example.com", "password123", nil},
		{"wrong password", "alice@example.com", "wrong-password", domain.ErrInvalidCredentials},
		{"unknown email", "bob@example.com", "password123", domain.ErrInvalidCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, token, err := svc.Login(ctx, tt.email, tt.password)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Login() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Login() unexpected error: %v", err)
			}
			if user.ID != registered.ID {
				t.Errorf("Login() user.ID = %v, want %v", user.ID, registered.ID)
			}
			userID, err := auth.Verify([]byte("test-secret"), token)
			if err != nil {
				t.Fatalf("Login() returned an unverifiable token: %v", err)
			}
			if userID != registered.ID {
				t.Errorf("Login() token subject = %v, want %v", userID, registered.ID)
			}
		})
	}
}

func TestAuthService_Register_NameTooLong(t *testing.T) {
	svc, _ := newAuthServiceForTest()
	_, _, err := svc.Register(context.Background(), "alice@example.com", strings.Repeat("a", nameMaxLen+1), "password123")
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Register() error = %v, want ErrValidation", err)
	}
}
