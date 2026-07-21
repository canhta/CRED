-- +goose Up
-- +goose StatementBegin

DROP INDEX IF EXISTS user_credentials_one_admin;

-- invited_by distinguishes how an account was created: NULL is the open
-- bootstrap path (register with no invite), non-null is an invite
-- redemption. This is what the replacement race-closer below keys on --
-- admin count itself is no longer capped.
ALTER TABLE user_credentials
    ADD COLUMN invited_by text REFERENCES principals(id);

-- Two concurrent bootstrap registrations can still both observe
-- UserCount() == 0 before either commits, the same race
-- user_credentials_one_admin used to close -- but that index capped total
-- admins, which this sub-project's role model no longer allows. This closes
-- the same race by capping bootstrap-created rows instead: at most one
-- account may ever be created without an invite.
CREATE UNIQUE INDEX user_credentials_one_bootstrap
    ON user_credentials ((invited_by IS NULL))
    WHERE invited_by IS NULL;

CREATE TABLE invites (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email        text        NOT NULL,
    role         text        NOT NULL CHECK (role IN ('admin', 'member')),
    token_hash   text        NOT NULL UNIQUE,
    invited_by   text        NOT NULL REFERENCES principals(id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    used_at      timestamptz,
    revoked_at   timestamptz
);

CREATE INDEX invites_token_hash_idx ON invites (token_hash);

-- Nothing in this sub-project ever updates or deletes a user_credentials
-- row -- there is no demote or remove capability yet. This trigger is
-- forward-looking scaffolding for when one exists: it is cheap to add now
-- and expensive to retrofit once a demote/remove path is live and already
-- shipped without this guard.
CREATE FUNCTION user_credentials_admin_floor() RETURNS trigger AS $$
BEGIN
    IF (SELECT count(*) FROM user_credentials WHERE role = 'admin') = 0 THEN
        RAISE EXCEPTION 'at least one admin must exist';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER user_credentials_admin_floor_trigger
    AFTER UPDATE OR DELETE ON user_credentials
    FOR EACH ROW
    WHEN (OLD.role = 'admin')
    EXECUTE FUNCTION user_credentials_admin_floor();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS user_credentials_admin_floor_trigger ON user_credentials;
DROP FUNCTION IF EXISTS user_credentials_admin_floor();
DROP TABLE IF EXISTS invites;
DROP INDEX IF EXISTS user_credentials_one_bootstrap;
ALTER TABLE user_credentials DROP COLUMN IF EXISTS invited_by;
CREATE UNIQUE INDEX user_credentials_one_admin
    ON user_credentials ((role))
    WHERE role = 'admin';
-- +goose StatementEnd
