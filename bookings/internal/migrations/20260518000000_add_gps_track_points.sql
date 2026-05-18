-- migrate:up
CREATE TABLE gps_track_points (
    id            SERIAL PRIMARY KEY,
    fk_booking_id INT NOT NULL REFERENCES bookings(id),
    lat           DOUBLE PRECISION NOT NULL,
    lon           DOUBLE PRECISION NOT NULL,
    recorded_at   TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- migrate:down
DROP TABLE gps_track_points;
