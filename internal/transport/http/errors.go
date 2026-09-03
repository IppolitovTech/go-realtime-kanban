package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

// RespondDomainError maps a domain-layer error to an HTTP response via
// errors.Is/As, so handlers don't each hand-roll their own status mapping —
// see roadmap.md, Stage 1, "Unified domain-error-to-HTTP-status mapping
// strategy".
func RespondDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrBoardNotFound):
		RespondError(w, http.StatusNotFound, err.Error(), "ERR_BOARD_NOT_FOUND")
	case errors.Is(err, domain.ErrColumnNotFound):
		RespondError(w, http.StatusNotFound, err.Error(), "ERR_COLUMN_NOT_FOUND")
	case errors.Is(err, domain.ErrCardNotFound):
		RespondError(w, http.StatusNotFound, err.Error(), "ERR_CARD_NOT_FOUND")
	case errors.Is(err, domain.ErrUserNotFound):
		RespondError(w, http.StatusNotFound, err.Error(), "ERR_USER_NOT_FOUND")
	case errors.Is(err, domain.ErrNotBoardMember):
		RespondError(w, http.StatusForbidden, err.Error(), "ERR_NOT_BOARD_MEMBER")
	case errors.Is(err, domain.ErrInvalidCredentials):
		RespondError(w, http.StatusUnauthorized, err.Error(), "ERR_INVALID_CREDENTIALS")
	case errors.Is(err, domain.ErrEmailTaken):
		RespondError(w, http.StatusConflict, err.Error(), "ERR_EMAIL_TAKEN")
	case errors.Is(err, domain.ErrUnauthorized):
		RespondError(w, http.StatusUnauthorized, err.Error(), "ERR_UNAUTHORIZED")
	case errors.Is(err, domain.ErrValidation):
		RespondError(w, http.StatusBadRequest, err.Error(), "ERR_VALIDATION")
	default:
		slog.Error("unhandled domain error", "error", err)
		RespondError(w, http.StatusInternalServerError, "internal server error", "ERR_INTERNAL")
	}
}
