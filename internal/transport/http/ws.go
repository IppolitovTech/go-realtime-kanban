package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/coder/websocket"

	"github.com/IppolitovTech/go-realtime-kanban/internal/realtime"
	"github.com/IppolitovTech/go-realtime-kanban/internal/service"
)

// NewWSHandler upgrades GET /boards/{boardId}/ws to a WebSocket and hands
// the connection to hub for the rest of its lifetime — see
// docs/ru/websocket-events.md for the wire format.
//
// appCtx is the application's shutdown context — the same one passed to
// `go hub.Run(appCtx)` and to server.Shutdown in main.go — not r.Context().
// net/http's graceful Shutdown explicitly does not wait for or cancel
// hijacked connections such as WebSockets (see its doc comment), so
// propagating the shutdown signal to every open connection is this
// handler's own responsibility — see roadmap.md, Stage 3, on tying the
// hub's graceful shutdown to the same context.Context as the HTTP
// server's.
func NewWSHandler(appCtx context.Context, boards *service.BoardService, hub *realtime.Hub, allowedOrigins []string) http.HandlerFunc {
	originPatterns := originHosts(allowedOrigins)

	return func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		boardID, ok := parseUUIDParam(w, r, "boardId", "board id")
		if !ok {
			return
		}

		// Reuse the same membership check as GET /boards/{boardId} — a
		// non-member gets a rejected handshake, not an open-but-useless
		// connection (see websocket-events.md's connection section).
		if _, err := boards.Get(r.Context(), userID, boardID); err != nil {
			RespondDomainError(w, err)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: originPatterns})
		if err != nil {
			return
		}
		defer conn.CloseNow()

		hub.HandleConnection(appCtx, conn, boardID)
	}
}

// originHosts strips the scheme off each configured CORS origin (e.g.
// "http://localhost:5173" -> "localhost:5173"): AcceptOptions.OriginPatterns
// matches against "host[:port]" by default, not a full URL.
func originHosts(origins []string) []string {
	hosts := make([]string, len(origins))
	for i, o := range origins {
		hosts[i] = strings.TrimPrefix(strings.TrimPrefix(o, "https://"), "http://")
	}
	return hosts
}
