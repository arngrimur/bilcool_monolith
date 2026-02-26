-- migrate:up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE bookings
(
    id         SERIAL PRIMARY KEY,
    booking_reference uuid NOT NULL,
    start_date timestamptz NOT NULL,
    end_date   timestamptz,
    user_ref   uuid NOT NULL
);
-- migrate:down
DROP TABLE bookings;