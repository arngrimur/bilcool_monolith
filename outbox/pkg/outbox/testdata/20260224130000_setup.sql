-- migrate:up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE apa
(
    id         SERIAL PRIMARY KEY,
);

CREATE TABLE bepa (
    id SERIAL PRIMARY KEY
);

-- migrate:down