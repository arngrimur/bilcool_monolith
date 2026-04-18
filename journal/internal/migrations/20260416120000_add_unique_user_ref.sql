-- migrate:up
ALTER TABLE users ADD CONSTRAINT users_user_ref_unique UNIQUE (user_ref);

-- migrate:down
ALTER TABLE users DROP CONSTRAINT users_user_ref_unique;
