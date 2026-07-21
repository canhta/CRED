-- +goose Up
-- +goose StatementBegin

-- Two concurrent first-registrations can both observe UserCount() == 0 before
-- either has committed, and the email UNIQUE constraint only catches a
-- same-email race -- it does nothing when the two requests use different
-- emails. A partial unique index over a constant expression makes "at most
-- one admin row exists" a fact Postgres itself enforces, closing the race the
-- application-level count-then-insert check cannot.
CREATE UNIQUE INDEX user_credentials_one_admin
    ON user_credentials ((role))
    WHERE role = 'admin';

-- sessions.token_hash is already UNIQUE above (00006_identity.sql), which
-- creates sessions_token_hash_key automatically. The separate btree index
-- this migration drops was redundant with that constraint's own index.
DROP INDEX IF EXISTS sessions_token_hash_idx;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE INDEX sessions_token_hash_idx ON sessions (token_hash);
DROP INDEX IF EXISTS user_credentials_one_admin;
-- +goose StatementEnd
