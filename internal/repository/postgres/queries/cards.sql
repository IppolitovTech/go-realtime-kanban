-- name: CreateCard :one
INSERT INTO cards (column_id, title, description, order_num, author_id)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetCardByID :one
SELECT * FROM cards WHERE id = $1;

-- name: ListCardsByColumn :many
SELECT * FROM cards WHERE column_id = $1 ORDER BY order_num, id;

-- name: UpdateCardContent :one
UPDATE cards SET title = $2, description = $3 WHERE id = $1 RETURNING *;

-- name: UpdateCardOrder :one
UPDATE cards SET column_id = $2, order_num = $3 WHERE id = $1 RETURNING *;

-- name: DeleteCard :exec
DELETE FROM cards WHERE id = $1;

-- name: MaxCardOrder :one
SELECT COALESCE(MAX(order_num), 0)::double precision FROM cards WHERE column_id = $1;

-- LockColumnForReorder serializes concurrent card moves within one
-- column — see ADR 004. Must run inside the same transaction that reads
-- neighbors and writes the new order_num.
-- name: LockColumnForReorder :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(column_id)::text, 0));
