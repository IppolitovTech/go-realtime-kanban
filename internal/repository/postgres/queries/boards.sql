-- name: CreateBoard :one
INSERT INTO boards (title, owner_id) VALUES ($1, $2) RETURNING *;

-- name: GetBoardByID :one
SELECT * FROM boards WHERE id = $1;

-- name: ListBoardsByMember :many
SELECT b.* FROM boards b
JOIN board_members bm ON bm.board_id = b.id
WHERE bm.user_id = $1
ORDER BY b.created_at, b.id;

-- name: UpdateBoardTitle :one
UPDATE boards SET title = $2 WHERE id = $1 RETURNING *;

-- name: DeleteBoard :exec
DELETE FROM boards WHERE id = $1;

-- name: AddBoardMember :one
WITH inserted AS (
    INSERT INTO board_members (board_id, user_id) VALUES ($1, $2)
    RETURNING board_id, user_id, joined_at
)
SELECT i.board_id, i.user_id, i.joined_at, u.email, u.name
FROM inserted i
JOIN users u ON u.id = i.user_id;

-- name: ListBoardMembers :many
SELECT bm.board_id, bm.user_id, bm.joined_at, u.email, u.name
FROM board_members bm
JOIN users u ON u.id = bm.user_id
WHERE bm.board_id = $1
ORDER BY bm.joined_at, bm.user_id;

-- name: IsBoardMember :one
SELECT EXISTS (
    SELECT 1 FROM board_members WHERE board_id = $1 AND user_id = $2
);
