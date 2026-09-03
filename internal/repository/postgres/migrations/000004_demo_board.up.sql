-- Sample board for the demo user (000003_demo_user.up.sql), so a reviewer
-- who logs in with it lands on a populated board instead of an empty list —
-- see README.md, "Demo account". order_num values follow the orderStep
-- convention from internal/service/order.go (ADR 004): 1000, 2000, 3000...

INSERT INTO boards (id, title, owner_id)
VALUES ('00000000-0000-0000-0000-000000000010', 'Demo Board', '00000000-0000-0000-0000-000000000002');

INSERT INTO board_members (board_id, user_id)
VALUES ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000002');

INSERT INTO columns (id, board_id, title, order_num) VALUES
    ('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000010', 'To Do', 1000),
    ('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000010', 'In Progress', 2000),
    ('00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000010', 'Done', 3000);

INSERT INTO cards (column_id, title, description, order_num, author_id) VALUES
    ('00000000-0000-0000-0000-000000000011', 'Set up project skeleton', 'Go module, folder layout, Docker Compose.', 1000, '00000000-0000-0000-0000-000000000002'),
    ('00000000-0000-0000-0000-000000000011', 'Design the data model', '', 2000, '00000000-0000-0000-0000-000000000002'),
    ('00000000-0000-0000-0000-000000000012', 'Wire up JWT authentication', 'Hand-rolled HS256, see ADR 005.', 1000, '00000000-0000-0000-0000-000000000002'),
    ('00000000-0000-0000-0000-000000000013', 'Real-time board updates', 'WebSocket hub broadcasting to open tabs.', 1000, '00000000-0000-0000-0000-000000000002');
