-- name: CreateColumn :one
INSERT INTO columns (board_id, title, order_num) VALUES ($1, $2, $3) RETURNING *;

-- name: GetColumnByID :one
SELECT * FROM columns WHERE id = $1;

-- name: ListColumnsByBoard :many
SELECT * FROM columns WHERE board_id = $1 ORDER BY order_num, id;

-- name: UpdateColumnTitle :one
UPDATE columns SET title = $2 WHERE id = $1 RETURNING *;

-- name: UpdateColumnOrder :one
UPDATE columns SET order_num = $2 WHERE id = $1 RETURNING *;

-- name: DeleteColumn :exec
DELETE FROM columns WHERE id = $1;

-- name: MaxColumnOrder :one
SELECT COALESCE(MAX(order_num), 0)::double precision FROM columns WHERE board_id = $1;

-- LockBoardForReorder serializes concurrent column moves within one
-- board — see ADR 004. Must run inside the same transaction that reads
-- neighbors and writes the new order_num.
-- name: LockBoardForReorder :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(board_id)::text, 0));
