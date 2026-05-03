-- migrate:up
CREATE TABLE booking_pauses (
    id            SERIAL PRIMARY KEY,
    fk_booking_id INT NOT NULL REFERENCES bookings(id),
    lat           DOUBLE PRECISION NOT NULL,
    lon           DOUBLE PRECISION NOT NULL,
    paused_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resumed_at    TIMESTAMPTZ
);

-- migrate:down
DROP TABLE booking_pauses;
