-- migrate:up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE apa
(
    id         SERIAL PRIMARY KEY,
    v varchar
);

CREATE TABLE bepa (
    id SERIAL PRIMARY KEY,
    v varchar
);

-- migrate:down