-- migrate:up
CREATE TABLE roles (
    id   SERIAL PRIMARY KEY,
    name varchar NOT NULL UNIQUE
);

INSERT INTO roles (name) VALUES ('user'), ('admin');

ALTER TABLE users
    ADD COLUMN role_id integer NOT NULL DEFAULT (SELECT id FROM roles WHERE name = 'user') REFERENCES roles(id);

CREATE INDEX ON users (role_id);

-- migrate:down
ALTER TABLE users DROP COLUMN role_id;
DROP TABLE roles;