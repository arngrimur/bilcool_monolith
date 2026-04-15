-- migrate:up

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    userref uuid NOT NULL unique ,
    received_at timestamptz NOT NULL default NOW(),
    deleted boolean NOT NULL DEFAULT false,
    deleted_at timestamptz
);

-- CREATE ROLE repl_user WITH LOGIN REPLICATION PASSWORD 'password';
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO repl_user;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


CREATE TABLE bookings
(
    id                SERIAL PRIMARY KEY,
    booking_reference uuid        NOT NULL UNIQUE,
    start_date        timestamptz NOT NULL,
    end_date          timestamptz,
    user_ref         int NOT NULL REFERENCES users (id)
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

CREATE VIEW bookings_and_users AS SELECT bookings.id, booking_reference,start_date,end_date, users.userref FROM bookings INNER JOIN users ON bookings.user_ref = users.id;

CREATE TABLE inbox (id SERIAL PRIMARY KEY, message_id varchar NOT NULL UNIQUE);
-- migrate:down
DROP TABLE inbox;
DROP VIEW bookings_and_users;
DROP TABLE distances;
DROP TABLE bookings;
DROP TABLE users;