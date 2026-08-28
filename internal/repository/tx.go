package repository

import "context"

// TxManager runs fn inside a single database transaction so that
// multi-step operations spanning several repository calls stay atomic —
// e.g. a card/column move: lock, read neighbors, write order_num (see
// ADR 004). Repository implementations built on the same underlying
// store pick up the transaction from ctx automatically.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
