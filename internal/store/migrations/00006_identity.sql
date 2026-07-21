-- +goose Up
-- +goose StatementBegin

-- Activates the principals table D-014 shipped and D-025 spends: a human
-- account is a principals row with kind='user', plus the auth-specific
-- fields that don't belong on team/org/agent principals. Separate table
-- rather than columns on principals, so principals stays the general
-- identity model every kind shares.
CREATE TABLE user_credentials (
    principal_id   text        PRIMARY KEY REFERENCES principals(id),
    email          text        NOT NULL UNIQUE,
    password_hash  text        NOT NULL,
    role           text        NOT NULL DEFAULT 'member'
                       CHECK (role IN ('admin', 'member')),
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- Sessions are revocable by construction: a row deleted (or expired) here is
-- a session that stops working on the next request, unlike a stateless JWT.
-- token_hash, never the raw token -- the same reason claim content is never
-- logged: a leaked table dump must not itself be a bearer credential.
CREATE TABLE sessions (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    principal_id   text        NOT NULL REFERENCES principals(id),
    token_hash     text        NOT NULL UNIQUE,
    created_at     timestamptz NOT NULL DEFAULT now(),
    expires_at     timestamptz NOT NULL
);

CREATE INDEX sessions_token_hash_idx ON sessions (token_hash);
CREATE INDEX sessions_principal_idx ON sessions (principal_id);

-- login_attempts backs the login rate limit. Keyed by email, not
-- principal_id: a failed login against an email with no account must still
-- count, or an attacker learns which emails are registered by which ones
-- never trip the limiter -- the same existence-oracle failure the login
-- handler's identical 401 already avoids at the response level.
CREATE TABLE login_attempts (
    id           bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email        text        NOT NULL,
    succeeded    boolean     NOT NULL,
    recorded_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX login_attempts_email_time ON login_attempts (email, recorded_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS user_credentials;
-- +goose StatementEnd
