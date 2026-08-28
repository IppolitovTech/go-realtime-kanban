package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository/postgres/generated"
)

var _ repository.TxManager = (*TxManager)(nil)

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}

	if err := fn(context.WithValue(ctx, txCtxKey{}, tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

type txCtxKey struct{}

// queriesFor returns sqlc-generated queries bound to the transaction
// TxManager.WithinTx put in ctx, falling back to the pool for reads that
// don't need one.
func queriesFor(ctx context.Context, pool *pgxpool.Pool) *generated.Queries {
	if tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx); ok {
		return generated.New(tx)
	}
	return generated.New(pool)
}
