-- migrate:up
CREATE TABLE positions (
    id                        SERIAL PRIMARY KEY,
    fk_booking_ended_event_id INT NOT NULL REFERENCES booking_ended_events(id),
    lat                       DOUBLE PRECISION NOT NULL,
    lon                       DOUBLE PRECISION NOT NULL
);

-- migrate:down
DROP TABLE positions;
