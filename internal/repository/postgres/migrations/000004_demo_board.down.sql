-- board_members/columns/cards all reference boards with ON DELETE CASCADE
-- (see 000002_boards_columns_cards.up.sql), so removing the board is enough.
DELETE FROM boards WHERE id = '00000000-0000-0000-0000-000000000010';
