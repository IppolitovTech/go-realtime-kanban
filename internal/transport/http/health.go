package http

import (
	"context"
	"net/http"
	"time"
)

// Pinger is satisfied by *pgxpool.Pool; kept as a narrow interface so the
// transport layer does not depend on pgx directly (see ADR 003).
type Pinger interface {
	Ping(ctx context.Context) error
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func HealthHandler(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbStatus := "connected"
		if err := db.Ping(ctx); err != nil {
			dbStatus = "disconnected"
		}

		RespondJSON(w, http.StatusOK, healthResponse{
			Status:   "ok",
			Database: dbStatus,
		})
	}
}
