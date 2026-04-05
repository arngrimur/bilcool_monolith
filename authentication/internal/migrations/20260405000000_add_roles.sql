-- migrate:up
CREATE TABLE roles (
    id   SERIAL PRIMARY KEY,
    name varchar NOT NULL UNIQUE
);

INSERT INTO roles (id, name) VALUES (2, 'user'), (1,'admin');

ALTER TABLE users
    ADD COLUMN role_id integer NOT NULL DEFAULT 2 REFERENCES roles(id);

CREATE INDEX ON users (role_id);

-- migrate:down
ALTER TABLE users DROP COLUMN role_id;
DROP TABLE roles;