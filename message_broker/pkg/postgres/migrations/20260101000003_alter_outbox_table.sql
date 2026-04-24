-- migrate:up
ALTER TABLE outbox ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE outbox ALTER COLUMN emitted_at SET DATA TYPE timestamp with time zone USING emitted_at::timestamp with time zone;

-- migrate:down
ALTER TABLE outbox DROP COLUMN IF EXISTS created_at;
ALTER TABLE outbox ALTER COLUMN emitted_at SET DATA TYPE timestamp USING emitted_at::timestamp;
