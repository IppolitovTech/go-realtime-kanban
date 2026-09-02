package realtime

import (
	"context"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

const (
	// clientSendBuffer bounds how many not-yet-delivered events a client
	// can queue up. A client that falls further behind than this is
	// dropped in Run's broadcast case rather than letting one slow reader
	// stall delivery to the rest of the board.
	clientSendBuffer = 16
	broadcastBuffer  = 256

	defaultPingInterval = 30 * time.Second
	defaultPongTimeout  = 10 * time.Second
	defaultWriteTimeout = 5 * time.Second
)

type client struct {
	conn    *websocket.Conn
	boardID uuid.UUID
	send    chan Event
}

// boardSizeQuery is a test-only request to read how many clients are
// registered for a board, round-tripped through Run's goroutine since
// boards is intentionally unguarded (see the Hub doc comment).
type boardSizeQuery struct {
	boardID uuid.UUID
	reply   chan int
}

// Hub is a WebSocket broadcast hub: it tracks which clients are subscribed
// to which board and fans out Publish calls to them over register/
// unregister/broadcast channels (see ADR 002 and roadmap.md, Stage 3). It
// carries no business logic — callers decide what happened and what to
// broadcast (see the Publisher interface); Hub only delivers it. The
// boards map is owned exclusively by the Run goroutine, so it needs no
// mutex.
type Hub struct {
	register         chan *client
	unregister       chan *client
	broadcast        chan Event
	boardSizeQueries chan boardSizeQuery

	boards map[uuid.UUID]map[*client]struct{}

	// Heartbeat timing (see HandleConnection). Fixed at construction;
	// hub_test.go shortens them via newTestHub to exercise the dead-client
	// path without waiting out the production intervals.
	pingInterval time.Duration
	pongTimeout  time.Duration
	writeTimeout time.Duration
}

func NewHub() *Hub {
	return &Hub{
		register:         make(chan *client),
		unregister:       make(chan *client),
		broadcast:        make(chan Event, broadcastBuffer),
		boardSizeQueries: make(chan boardSizeQuery),
		boards:           make(map[uuid.UUID]map[*client]struct{}),
		pingInterval:     defaultPingInterval,
		pongTimeout:      defaultPongTimeout,
		writeTimeout:     defaultWriteTimeout,
	}
}

// Run owns the hub's state until ctx is cancelled. It must be started
// exactly once (e.g. `go hub.Run(ctx)` alongside the HTTP server's own
// shutdown context — see roadmap.md's hub graceful-shutdown item). On
// cancellation it closes every connection it still knows about so their
// HandleConnection goroutines can return.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for _, clients := range h.boards {
				for c := range clients {
					c.conn.Close(websocket.StatusGoingAway, "server shutting down")
				}
			}
			return

		case c := <-h.register:
			if h.boards[c.boardID] == nil {
				h.boards[c.boardID] = make(map[*client]struct{})
			}
			h.boards[c.boardID][c] = struct{}{}

		case c := <-h.unregister:
			h.dropClient(c)

		case q := <-h.boardSizeQueries:
			q.reply <- len(h.boards[q.boardID])

		case event := <-h.broadcast:
			for c := range h.boards[event.BoardID] {
				select {
				case c.send <- event:
				default:
					slog.Warn("dropping slow websocket client", "board_id", event.BoardID)
					h.dropClient(c)
					c.conn.CloseNow()
				}
			}
		}
	}
}

// boardSize reports how many clients are currently registered for boardID.
// It only exists for tests — see boardSizeQuery's doc comment.
func (h *Hub) boardSize(ctx context.Context, boardID uuid.UUID) int {
	reply := make(chan int, 1)
	select {
	case h.boardSizeQueries <- boardSizeQuery{boardID: boardID, reply: reply}:
	case <-ctx.Done():
		return 0
	}
	select {
	case n := <-reply:
		return n
	case <-ctx.Done():
		return 0
	}
}

// dropClient removes c from its board and closes its send channel. Safe to
// call more than once for the same client (e.g. once from the slow-client
// path in the broadcast case, and again from HandleConnection's deferred
// unregister) — the second call is a no-op because c is no longer present
// in the map.
func (h *Hub) dropClient(c *client) {
	clients, ok := h.boards[c.boardID]
	if !ok {
		return
	}
	if _, ok := clients[c]; !ok {
		return
	}
	delete(clients, c)
	close(c.send)
	if len(clients) == 0 {
		delete(h.boards, c.boardID)
	}
}

// Publish delivers event to Run's broadcast channel. It never blocks the
// caller on a slow or stalled hub: REST handlers call this synchronously
// after committing a change (see service layer), and a hung broadcast must
// not hang the HTTP response with it — the default case alone already
// guarantees that.
//
// ctx is accepted only to satisfy the Publisher interface and is
// deliberately not part of the select below: it's the initiating request's
// own context, and selecting on it here raced the broadcast send whenever
// that request's client disconnected between its commit and this call —
// Go picks pseudo-randomly among ready select cases, so the event could be
// silently dropped for every other tab on the board even though the
// channel had room.
func (h *Hub) Publish(ctx context.Context, event Event) {
	select {
	case h.broadcast <- event:
	default:
		slog.Warn("dropping websocket event: hub broadcast channel full", "type", event.Type, "board_id", event.BoardID)
	}
}

// HandleConnection registers conn under boardID and blocks, delivering
// broadcast Events to conn as JSON text frames until the connection dies
// (heartbeat timeout, client disconnect, or ctx cancellation — including
// Run's own shutdown) or the caller's ctx is cancelled. The HTTP upgrade
// itself is the caller's responsibility (see transport/http/ws.go); Hub
// only owns the connection's lifecycle from here on.
func (h *Hub) HandleConnection(ctx context.Context, conn *websocket.Conn, boardID uuid.UUID) {
	c := &client{conn: conn, boardID: boardID, send: make(chan Event, clientSendBuffer)}

	select {
	case h.register <- c:
	case <-ctx.Done():
		return
	}
	defer func() {
		// ctx here is deliberately the outer, hub-shutdown-scoped context
		// passed into HandleConnection — NOT connCtx below. Once Run's own
		// ctx fires it closes every connection and returns for good (see
		// Run's ctx.Done case), so it stops reading h.unregister forever;
		// guarding on the same ctx here avoids blocking on that dead
		// channel forever. connCtx, by contrast, is also done whenever
		// this one connection merely drops — in that ordinary case Run is
		// very much still running and waiting on h.unregister, so this
		// send must still go through.
		select {
		case h.unregister <- c:
		case <-ctx.Done():
		}
	}()

	// The client is push-only (see architecture.md's REST-vs-WebSocket
	// roles section) — CloseRead spins up a background reader that discards
	// any incoming data message (closing the connection if one arrives)
	// and answers ping/pong/close control frames automatically. Deriving
	// connCtx from its return value means a dead connection unblocks both
	// the ticker loop below and this goroutine's exit from a single
	// signal, per ADR 002.
	//
	// Deliberately NOT derived from ctx: CloseRead uses whatever context
	// it's given for its own background Read, and per Conn's doc comment
	// that context expiring closes the connection too ("this applies to
	// context expirations as well unfortunately") — with no specific
	// status code. If ctx were the outer hub-shutdown context, that would
	// race Run's own explicit conn.Close(StatusGoingAway, ...) below on
	// shutdown, and clients could see a bare dropped connection instead of
	// that status. Using context.Background() here leaves closing this
	// connection entirely up to Run's explicit Close call (or a failed
	// Ping/Write below) — see TestHub_ShutdownClosesConnections.
	connCtx := conn.CloseRead(context.Background())

	ticker := time.NewTicker(h.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-connCtx.Done():
			return

		case event, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(connCtx, h.writeTimeout)
			err := wsjson.Write(writeCtx, conn, event)
			cancel()
			if err != nil {
				return
			}

		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(connCtx, h.pongTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				// No pong within pongTimeout (or the connection is
				// otherwise dead) — Ping already closed conn, which in
				// turn cancels connCtx; return so the deferred unregister
				// runs.
				return
			}
		}
	}
}
