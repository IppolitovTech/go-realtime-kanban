-- Этап 1 schema: users (with one stub row — see architecture.md,
-- "Заглушка пользователя на Этапе 1"), boards, board membership,
-- columns and cards. order_num is double precision per ADR 004.

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE boards (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title      TEXT NOT NULL,
    owner_id   UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE board_members (
    board_id  UUID NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (board_id, user_id)
);

CREATE TABLE columns (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id   UUID NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    order_num  DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_columns_board_order ON columns (board_id, order_num, id);

CREATE TABLE cards (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    column_id   UUID NOT NULL REFERENCES columns (id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    order_num   DOUBLE PRECISION NOT NULL,
    author_id   UUID NOT NULL REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cards_column_order ON cards (column_id, order_num, id);

-- Fixed seed user standing in for real auth until Stage 2; the
-- transport layer defaults X-User-ID to this UUID when unset.
INSERT INTO users (id, email, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'stub@example.local', 'Stub User');
