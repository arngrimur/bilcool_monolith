-- migrate:up
ALTER TABLE distances ADD COLUMN lat DOUBLE PRECISION;
ALTER TABLE distances ADD COLUMN lon DOUBLE PRECISION;

-- migrate:down
ALTER TABLE distances DROP COLUMN lat;
ALTER TABLE distances DROP COLUMN lon;
