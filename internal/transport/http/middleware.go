package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// stubUserID is the seed row inserted by migration 000002 (see
// architecture.md, "User stub in Stage 1"). Used whenever the
// request has no X-User-ID header, so handlers can be exercised by hand
// without setting one.
var stubUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type userIDCtxKey struct{}

// DevUserID reads X-User-ID (see the DevUserIdHeader security scheme in
// openapi.yaml) and stores it in the request context, defaulting to
// stubUserID when the header is absent or not a valid UUID. Stage 2
// replaces this middleware with real JWT auth without touching any
// downstream signature — see architecture.md, "User context".
func DevUserID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := stubUserID
		if header := r.Header.Get("X-User-ID"); header != "" {
			if parsed, err := uuid.Parse(header); err == nil {
				userID = parsed
			}
		}
		ctx := context.WithValue(r.Context(), userIDCtxKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext retrieves the userID DevUserID (or, from Stage 2
// onward, the JWT middleware) placed in the request context.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDCtxKey{}).(uuid.UUID)
	return id
}
