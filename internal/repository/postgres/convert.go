package postgres

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
	return uuid.UUID(id.Bytes)
}

func fromPgTime(t pgtype.Timestamptz) time.Time {
	return t.Time
}

// mapNotFound translates pgx.ErrNoRows into notFound, the domain sentinel
// for "no row matched", leaving any other error (including nil) unchanged.
// Centralizes the pgx-to-domain not-found translation that every
// repository's single-row query needs, instead of repeating the same
// errors.Is check at each call site.
func mapNotFound(err, notFound error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return err
}
