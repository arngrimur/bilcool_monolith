-- migrate:up
INSERT INTO users (username, email, role_id)
VALUES ('admin', 'admin@bilcool.local', 1);

-- migrate:down
DELETE FROM users WHERE username = 'admin' AND email = 'admin@bilcool.local';
