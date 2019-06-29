-- @migrate.up
INSERT INTO users(id, name) VALUES (1, 'taro');
INSERT INTO users(id, name) VALUES (2, 'hanako');

-- @migrate.down
DELETE FROM users WHERE id = 1;
DELETE FROM users WHERE id = 2;
