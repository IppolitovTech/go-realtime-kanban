package realtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"go.uber.org/goleak"
)

// newTestHub shortens the heartbeat timing so TestHub_RemovesDeadConnection
// doesn't have to wait out the production 30s/10s intervals.
func newTestHub(pingInterval, pongTimeout time.Duration) *Hub {
	h := NewHub()
	h.pingInterval = pingInterval
	h.pongTimeout = pongTimeout
	return h
}

// waitForBoardSize polls Hub's client count for boardID (via the
// channel-safe boardSize query) until it matches want or timeout elapses.
func waitForBoardSize(t *testing.T, ctx context.Context, hub *Hub, boardID uuid.UUID, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if n := hub.boardSize(ctx, boardID); n == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for board %s to reach size %d (last=%d)", boardID, want, hub.boardSize(ctx, boardID))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// hubServerHandler upgrades every request to a WebSocket and hands it to
// hub.HandleConnection under the "board" query parameter.
func hubServerHandler(hub *Hub, ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		boardID, err := uuid.Parse(r.URL.Query().Get("board"))
		if err != nil {
			http.Error(w, "bad board", http.StatusBadRequest)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		hub.HandleConnection(ctx, conn, boardID)
	}
}

// newHubServer starts an httptest server via hubServerHandler and tears it
// down through t.Cleanup — fine as long as the test isn't also asserting
// goroutine cleanup with goleak, since t.Cleanup runs after any deferred
// goleak.VerifyNone in the test body (see the goleak tests below, which
// close their server explicitly instead, before that check runs).
func newHubServer(t *testing.T, hub *Hub, ctx context.Context) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(hubServerHandler(hub, ctx))
	t.Cleanup(srv.Close)
	return srv
}

func dialBoard(t *testing.T, srv *httptest.Server, boardID uuid.UUID) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "?board=" + boardID.String()
	conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("dial board %s: %v", boardID, err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

func TestHub_BroadcastsOnlyToBoardSubscribers(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	boardA, boardB := uuid.New(), uuid.New()
	srv := newHubServer(t, hub, ctx)

	connA1 := dialBoard(t, srv, boardA)
	connA2 := dialBoard(t, srv, boardA)
	connB := dialBoard(t, srv, boardB)

	waitForBoardSize(t, ctx, hub, boardA, 2, time.Second)
	waitForBoardSize(t, ctx, hub, boardB, 1, time.Second)

	event := NewEvent(EventCardCreated, boardA, map[string]string{"title": "demo"})
	hub.Publish(ctx, event)

	for _, conn := range []*websocket.Conn{connA1, connA2} {
		readCtx, cancelRead := context.WithTimeout(context.Background(), time.Second)
		var got Event
		err := wsjson.Read(readCtx, conn, &got)
		cancelRead()
		if err != nil {
			t.Fatalf("boardA subscriber did not receive the event: %v", err)
		}
		if got.Type != EventCardCreated || got.BoardID != boardA {
			t.Fatalf("unexpected event delivered: %+v", got)
		}
	}

	readCtx, cancelRead := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelRead()
	var got Event
	if err := wsjson.Read(readCtx, connB, &got); err == nil {
		t.Fatalf("boardB subscriber unexpectedly received a boardA event: %+v", got)
	}
}

// TestHub_RemovesDeadConnection is the heartbeat test roadmap.md and ADR
// 002 call for explicitly: a client that stops answering pings must be
// evicted from the hub, and its per-connection goroutines must actually
// exit (checked with goleak) rather than leak on a blocked Read/Ping.
func TestHub_RemovesDeadConnection(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	hub := newTestHub(50*time.Millisecond, 50*time.Millisecond)
	hubCtx, cancelHub := context.WithCancel(context.Background())
	go hub.Run(hubCtx)

	boardID := uuid.New()
	srv := httptest.NewServer(hubServerHandler(hub, hubCtx))
	// Registered after goleak's defer above, so — by LIFO defer order —
	// this (and cancelHub, and the conn close below) runs first and tears
	// everything down before goleak.VerifyNone checks for leftover
	// goroutines, even if a t.Fatalf below exits the test early. See
	// TestHub_ShutdownClosesConnections for the same pattern.
	defer srv.Close()
	defer cancelHub()

	// Dial but deliberately never read from this connection — coder/
	// websocket only answers ping/pong control frames while something is
	// reading (see Conn's doc comment), so a peer that never reads is
	// indistinguishable, from the server's side, from one that's gone
	// dark. That's exactly the case the heartbeat timeout exists for.
	deadConn := dialBoard(t, srv, boardID)

	waitForBoardSize(t, hubCtx, hub, boardID, 1, time.Second)
	waitForBoardSize(t, hubCtx, hub, boardID, 0, 2*time.Second)

	deadConn.CloseNow()
}

func TestHub_ShutdownClosesConnections(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)

	boardID := uuid.New()
	srv := httptest.NewServer(hubServerHandler(hub, ctx))
	// See the matching comment in TestHub_RemovesDeadConnection for why
	// these are defers registered after goleak's, not bare statements.
	defer srv.Close()
	defer cancel()

	conn := dialBoard(t, srv, boardID)

	waitForBoardSize(t, ctx, hub, boardID, 1, time.Second)

	cancel()

	readCtx, cancelRead := context.WithTimeout(context.Background(), time.Second)
	_, _, err := conn.Read(readCtx)
	cancelRead()
	if err == nil {
		t.Fatal("expected the connection to be closed by hub shutdown")
	}
	if status := websocket.CloseStatus(err); status != websocket.StatusGoingAway {
		t.Fatalf("expected StatusGoingAway, got %v (err=%v)", status, err)
	}
}
