-- migrate:up
CREATE TABLE positions (
    id           SERIAL PRIMARY KEY,
    fk_booking_id INT NOT NULL REFERENCES bookings(id),
    lat          DOUBLE PRECISION NOT NULL,
    lon          DOUBLE PRECISION NOT NULL
);

-- migrate:down
DROP TABLE positions;
