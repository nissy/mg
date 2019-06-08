-- @migrate.up
INSERT INTO users(id, name) VALUES (1, E'nissy');

-- @migrate.down
DELETE FROM users WHERE id = 1;
