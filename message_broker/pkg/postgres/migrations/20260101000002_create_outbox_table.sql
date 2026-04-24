-- migrate:up
CREATE TABLE IF NOT EXISTS outbox (
    id serial PRIMARY KEY,
    event_id uuid NOT NULL UNIQUE,
    type varchar NOT NULL,
    correlation_id uuid NOT NULL,
    producer varchar NOT NULL,
    emitted_at timestamp,
    payload jsonb NOT NULL
);

-- migrate:down
DROP TABLE IF EXISTS outbox;
