package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/auth"
	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

type userIDCtxKey struct{}

// JWTAuth verifies the request's JWT and stores the userID it was issued
// for in the request context, responding 401 and short-circuiting the
// chain otherwise. It reads the token from the standard
// "Authorization: Bearer <token>" header, falling back to a "token" query
// parameter — the browser WebSocket API can't set custom headers on the
// handshake request, so /boards/{boardId}/ws (the only route that needs
// it) passes the token that way instead.
func JWTAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				RespondDomainError(w, domain.ErrUnauthorized)
				return
			}

			userID, err := auth.Verify(secret, token)
			if err != nil {
				RespondDomainError(w, domain.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDCtxKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) string {
	if header := r.Header.Get("Authorization"); header != "" {
		if rest, ok := strings.CutPrefix(header, "Bearer "); ok {
			return rest
		}
		return ""
	}
	return r.URL.Query().Get("token")
}

// UserIDFromContext retrieves the userID JWTAuth placed in the request
// context.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDCtxKey{}).(uuid.UUID)
	return id
}
