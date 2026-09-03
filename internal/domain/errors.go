package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors mapped to HTTP statuses in the transport layer via
// errors.Is (see roadmap.md, Stage 1 — "Unified domain-error-to-HTTP-status
// mapping strategy").
var (
	ErrBoardNotFound  = errors.New("board not found")
	ErrColumnNotFound = errors.New("column not found")
	ErrCardNotFound   = errors.New("card not found")
	ErrUserNotFound   = errors.New("user not found")
	ErrNotBoardMember = errors.New("user is not a board member")

	// Auth sentinels — see internal/service/auth.go and
	// docs/ru/adr/005-jwt-vs-sessions.md.
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrUnauthorized       = errors.New("missing or invalid authentication token")

	// ErrValidation is the sentinel every *ValidationError unwraps to, so
	// the transport layer can map the whole family with one errors.Is
	// check instead of enumerating field-specific errors.
	ErrValidation = errors.New("validation error")
)

// ValidationError carries which field failed validation and why, while
// still satisfying errors.Is(err, ErrValidation) for the transport-layer
// mapping to a 400 response.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

func NewValidationError(field, reason string) error {
	return &ValidationError{Field: field, Reason: reason}
}
