-- migrate:up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id         SERIAL PRIMARY KEY,
    user_ref   uuid        NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    username   varchar     NOT NULL UNIQUE,
    email      varchar     NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE INDEX ON users (email);

CREATE TABLE security_tokens (
    id         SERIAL PRIMARY KEY,
    user_ref   uuid        NOT NULL REFERENCES users(user_ref) ON DELETE CASCADE,
    token      varchar(6)  NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX ON security_tokens (user_ref, expires_at);
CREATE INDEX ON security_tokens (token, expires_at);

CREATE TABLE passkeys (
    id            SERIAL PRIMARY KEY,
    user_ref      uuid        NOT NULL REFERENCES users(user_ref) ON DELETE CASCADE,
    credential_id bytea       NOT NULL UNIQUE,
    credential    jsonb       NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX ON passkeys (user_ref);

CREATE TABLE webauthn_sessions (
    id           SERIAL PRIMARY KEY,
    session_id   uuid        NOT NULL UNIQUE,
    user_ref     uuid        NOT NULL REFERENCES users(user_ref) ON DELETE CASCADE,
    session_type varchar     NOT NULL,
    data         jsonb       NOT NULL,
    expires_at   timestamptz NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX ON webauthn_sessions (session_id, expires_at);

-- migrate:down
DROP TABLE webauthn_sessions;
DROP TABLE passkeys;
DROP TABLE security_tokens;
DROP TABLE users;
