-- migrate:up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE inbox (
  id  SERIAL PRIMARY KEY ,
    event_id UUID NOT NULL,
    type VARCHAR NOT NULL ,
    correlation_id UUID NOT NULL,
    producer VARCHAR NOT NULL,
    emitted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_inbox_event_id ON inbox (event_id); --  UNIQUE stops duplicate events from being inserted

CREATE TABLE users(
                      id SERIAL PRIMARY KEY ,
                      user_ref UUID NOT NULL ,
                      created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE booking_ended_events (
    id SERIAL PRIMARY KEY ,
    fk_user int REFERENCES users(id),
    booking_ref uuid NOT NULL,
    start_date  TIMESTAMPTZ NOT NULL ,
    end_date TIMESTAMPTZ NOT NULL,
    distance_meters int NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- migrate:down
DROP TABLE booking_ended_events;
DROP TABLE users;
DROP INDEX idx_inbox_event_id;
DROP TABLE inbox;