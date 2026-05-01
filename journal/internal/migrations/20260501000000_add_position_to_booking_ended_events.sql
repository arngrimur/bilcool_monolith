-- migrate:up
ALTER TABLE booking_ended_events ADD COLUMN lat DOUBLE PRECISION;
ALTER TABLE booking_ended_events ADD COLUMN lon DOUBLE PRECISION;

-- migrate:down
ALTER TABLE booking_ended_events DROP COLUMN lat;
ALTER TABLE booking_ended_events DROP COLUMN lon;
