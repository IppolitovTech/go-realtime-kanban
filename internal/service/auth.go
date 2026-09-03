package service

import (
	"context"
	"errors"
	"net/mail"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/IppolitovTech/go-realtime-kanban/internal/auth"
	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
)

const (
	nameMaxLen      = 100
	passwordMinLen  = 8
	defaultTokenTTL = 24 * time.Hour
)

// AuthService handles registration and login, issuing JWTs signed with
// jwtSecret — see docs/ru/adr/005-jwt-vs-sessions.md.
type AuthService struct {
	users     repository.UserRepository
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewAuthService(users repository.UserRepository, jwtSecret []byte, tokenTTL time.Duration) *AuthService {
	if tokenTTL <= 0 {
		tokenTTL = defaultTokenTTL
	}
	return &AuthService{users: users, jwtSecret: jwtSecret, tokenTTL: tokenTTL}
}

// Register creates a new user and returns it along with a signed token, so
// the caller doesn't need a separate Login call right after registering.
func (s *AuthService) Register(ctx context.Context, email, name, password string) (domain.User, string, error) {
	if err := validateEmail(email); err != nil {
		return domain.User{}, "", err
	}
	if err := validateTitle("name", name, nameMaxLen); err != nil {
		return domain.User{}, "", err
	}
	if len(password) < passwordMinLen {
		return domain.User{}, "", domain.NewValidationError("password", "must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, "", err
	}

	user, err := s.users.Create(ctx, domain.User{Email: email, Name: name, PasswordHash: string(hash)})
	if err != nil {
		return domain.User{}, "", err
	}

	token, err := auth.Sign(s.jwtSecret, user.ID, s.tokenTTL)
	if err != nil {
		return domain.User{}, "", err
	}
	return user, token, nil
}

// Login verifies email/password and returns the user with a signed token.
// A missing user and a wrong password both collapse to
// domain.ErrInvalidCredentials so the response never reveals whether the
// email is registered.
func (s *AuthService) Login(ctx context.Context, email, password string) (domain.User, string, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.User{}, "", domain.ErrInvalidCredentials
		}
		return domain.User{}, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return domain.User{}, "", domain.ErrInvalidCredentials
	}

	token, err := auth.Sign(s.jwtSecret, user.ID, s.tokenTTL)
	if err != nil {
		return domain.User{}, "", err
	}
	return user, token, nil
}

func validateEmail(email string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return domain.NewValidationError("email", "must be a valid email address")
	}
	return nil
}
