package realtime

import "context"

// Publisher is what the service layer depends on to announce that
// something changed. Hub implements it. Kept as its own interface (rather
// than having service depend on *Hub directly) so service tests can stub
// it out without spinning up a hub — the same reasoning as service
// depending on repository interfaces instead of the postgres package, see
// architecture.md.
type Publisher interface {
	Publish(ctx context.Context, event Event)
}

// NoopPublisher discards every event. It exists for tests that need
// something satisfying Publisher but don't exercise realtime delivery
// themselves — e.g. the service-layer table tests and the card/order
// integration test, which have their own hub-less fixtures.
type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, Event) {}
