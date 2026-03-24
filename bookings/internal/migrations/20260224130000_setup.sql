-- migrate:up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- CREATE ROLE repl_user WITH LOGIN REPLICATION PASSWORD 'password';
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO repl_user;

CREATE TABLE bookings
(
    id                SERIAL PRIMARY KEY,
    booking_reference uuid        NOT NULL UNIQUE,
    start_date        timestamptz NOT NULL,
    end_date          timestamptz,
    user_ref          uuid        NOT NULL
);

CREATE INDEX ON bookings (user_ref, start_date, end_date); -- don't use concurrently as we can accept locks while doing changes


CREATE TABLE distances
(
id            SERIAL PRIMARY KEY,
start_distance        int NOT NULL,
end_distance           int NOT NULL,
created_at    timestamptz NOT NULL DEFAULT now(),
fk_booking_id int NOT NULL REFERENCES bookings (id)
);

CREATE INDEX ON distances (fk_booking_id, start_distance);


-- migrate:down
DROP TABLE distances;
DROP TABLE bookings;