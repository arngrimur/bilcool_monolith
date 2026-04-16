-- migrate:up
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;

-- migrate:down
ALTER TABLE users DROP COLUMN deleted_at;
