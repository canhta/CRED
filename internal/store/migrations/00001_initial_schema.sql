-- +goose Up
-- +goose StatementBegin

-- pgvector is not a "trusted" extension, so CREATE EXTENSION needs superuser.
-- On managed Postgres this line fails and `cred doctor` prints the exact
-- command for a DBA to run. It is never created automatically there.
CREATE EXTENSION IF NOT EXISTS vector;

-- Principals.
--
-- D-014: org, member and role primitives live in the binary a solo developer
-- runs offline, not in a client wrapper over a hosted service. This slice
-- ships exactly one principal and the table is here anyway, because every
-- competitor that deferred the principal type could not add it back.
CREATE TABLE principals (
    id             text PRIMARY KEY,
    kind           text        NOT NULL CHECK (kind IN ('user', 'team', 'org', 'agent')),
    display_name   text        NOT NULL,
    recorded_at    timestamptz NOT NULL DEFAULT now(),
    superseded_at  timestamptz,

    CONSTRAINT principals_recorded_half_open
        CHECK (superseded_at IS NULL OR recorded_at < superseded_at)
);

-- Evidence. L1: a claim with no evidence cannot be written.
CREATE TABLE evidence (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_kind    text NOT NULL CHECK (source_kind IN ('document', 'code', 'attestation')),

    source_repo    text NOT NULL,
    source_path    text NOT NULL,
    chunk_ordinal  int  NOT NULL,
    line_start     int  NOT NULL CHECK (line_start >= 1),
    line_end       int  NOT NULL CHECK (line_end >= line_start),

    -- Retained, not pointed at. A vectors-only store can never migrate to a
    -- new embedding model, and that is decided at ingest.
    extracted_text text NOT NULL,

    -- Computed in Go, never as a GENERATED column: sha256 is immutable but
    -- convert_to is only stable, so the generated form is rejected, and an
    -- IMMUTABLE wrapper would assert a guarantee Postgres does not make. If
    -- server encoding ever changed, every hash would shift and invalidation
    -- would fire across the whole corpus.
    content_sha256 bytea NOT NULL CHECK (octet_length(content_sha256) = 32),

    attested_by    text REFERENCES principals (id),
    attested_at    timestamptz,

    -- L6. Half-open throughout: [from, until). NULL means open-ended.
    valid_from     timestamptz NOT NULL,
    valid_until    timestamptz,
    recorded_at    timestamptz NOT NULL,
    superseded_at  timestamptz,

    CONSTRAINT evidence_valid_half_open
        CHECK (valid_until IS NULL OR valid_from < valid_until),
    CONSTRAINT evidence_recorded_half_open
        CHECK (superseded_at IS NULL OR recorded_at < superseded_at),
    CONSTRAINT evidence_attestation_is_complete
        CHECK ((attested_by IS NULL) = (attested_at IS NULL))
);

-- One live chunk per (repo, path, ordinal). Superseded rows are kept — nothing
-- is deleted, things expire — so the uniqueness is partial.
CREATE UNIQUE INDEX evidence_live_chunk
    ON evidence (source_repo, source_path, chunk_ordinal)
    WHERE superseded_at IS NULL;

CREATE INDEX evidence_by_path ON evidence (source_repo, source_path);

-- Lexical arm of retrieval. The regconfig is named explicitly because
-- to_tsvector(text) alone is only STABLE and cannot back a stored column.
ALTER TABLE evidence
    ADD COLUMN search_tsv tsvector
    GENERATED ALWAYS AS (to_tsvector('english', extracted_text)) STORED;

CREATE INDEX evidence_search_tsv ON evidence USING gin (search_tsv);

-- Claims.
CREATE TABLE claims (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind               text NOT NULL CHECK (kind IN (
                           'Convention', 'Decision', 'Constraint',
                           'RejectedApproach', 'Failure', 'Reference')),
    statement          text NOT NULL CHECK (statement <> ''),
    scope_kind         text NOT NULL CHECK (scope_kind IN (
                           'organization', 'repository', 'path', 'service')),
    scope_value        text NOT NULL,

    valid_from         timestamptz NOT NULL,
    valid_until        timestamptz,
    recorded_at        timestamptz NOT NULL,
    superseded_at      timestamptz,
    superseded_by      uuid REFERENCES claims (id),

    -- An explainable additive score, never an opaque posterior.
    confidence         double precision NOT NULL CHECK (confidence BETWEEN 0 AND 1),

    -- Provenance, immutable, required by L8.
    source_repo        text NOT NULL,
    source_commit      text NOT NULL DEFAULT '',
    extracted_by_model text NOT NULL,
    prompt_version     text NOT NULL DEFAULT '',

    CONSTRAINT claims_valid_half_open
        CHECK (valid_until IS NULL OR valid_from < valid_until),
    CONSTRAINT claims_recorded_half_open
        CHECK (superseded_at IS NULL OR recorded_at < superseded_at),
    -- A supersession edge and a closed transaction interval are the same
    -- event; allowing one without the other produces a claim that is current
    -- and replaced at once.
    CONSTRAINT claims_supersession_is_atomic
        CHECK ((superseded_by IS NULL) OR (superseded_at IS NOT NULL)),
    CONSTRAINT claims_no_self_supersession
        CHECK (superseded_by IS NULL OR superseded_by <> id)
);

CREATE INDEX claims_live ON claims (recorded_at) WHERE superseded_at IS NULL;

-- L1, as a table. A claim reaches a reader only through this join, so an
-- orphan claim is unreachable by construction rather than by a check.
CREATE TABLE claim_evidence (
    claim_id    uuid NOT NULL REFERENCES claims (id) ON DELETE RESTRICT,
    evidence_id uuid NOT NULL REFERENCES evidence (id) ON DELETE RESTRICT,
    PRIMARY KEY (claim_id, evidence_id)
);

CREATE INDEX claim_evidence_by_evidence ON claim_evidence (evidence_id);

-- Access control.
--
-- These are rows, not predicates. L5's intersection is computed in Go by
-- internal/acl. Nothing in this schema filters by principal, and no query in
-- internal/store/pg takes one. Deciding in SQL is the known silent-failure
-- path: pgvector filtering under ACL selectivity returns 4 results where 40
-- were expected, with no error.
--
-- expires_at carries L5's per-entry TTL. NULL means no expiry.
CREATE TABLE claim_acl (
    claim_id     uuid NOT NULL REFERENCES claims (id) ON DELETE RESTRICT,
    principal_id text NOT NULL REFERENCES principals (id),
    expires_at   timestamptz,
    PRIMARY KEY (claim_id, principal_id)
);

CREATE TABLE evidence_acl (
    evidence_id  uuid NOT NULL REFERENCES evidence (id) ON DELETE RESTRICT,
    principal_id text NOT NULL REFERENCES principals (id),
    expires_at   timestamptz,
    PRIMARY KEY (evidence_id, principal_id)
);

-- Embedding models.
--
-- One PRESENT and one FUTURE, enforced as database constraints rather than as
-- application discipline. Onyx is the only surveyed project where the
-- read-from-a-half-built-index failure is impossible rather than merely
-- unlikely, and this is the mechanism.
CREATE TABLE embedding_models (
    id         int PRIMARY KEY,
    name       text NOT NULL UNIQUE,
    dimensions int  NOT NULL CHECK (dimensions > 0),
    status     text NOT NULL CHECK (status IN ('PRESENT', 'FUTURE', 'RETIRED'))
);

CREATE UNIQUE INDEX embedding_models_one_present
    ON embedding_models (status) WHERE status = 'PRESENT';
CREATE UNIQUE INDEX embedding_models_one_future
    ON embedding_models (status) WHERE status = 'FUTURE';

-- Claim embeddings, partitioned by model.
--
-- The parent column is a deliberately unspecified halfvec. Dimension lives on
-- the per-partition expression index, because ALTER COLUMN TYPE across
-- dimensions fails on populated data and a view cannot change a column type.
-- Retiring a model is DETACH PARTITION.
--
-- embedding_model_id is NOT NULL with no default and no backfill-by-guess, and
-- reads filter on it. A per-row model column that reads ignore is worthless.
CREATE TABLE claim_embeddings (
    embedding_model_id int  NOT NULL REFERENCES embedding_models (id),
    claim_id           uuid NOT NULL REFERENCES claims (id) ON DELETE RESTRICT,
    embedding          halfvec NOT NULL,
    PRIMARY KEY (embedding_model_id, claim_id)
) PARTITION BY LIST (embedding_model_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS claim_embeddings;
DROP TABLE IF EXISTS embedding_models;
DROP TABLE IF EXISTS evidence_acl;
DROP TABLE IF EXISTS claim_acl;
DROP TABLE IF EXISTS claim_evidence;
DROP TABLE IF EXISTS claims;
DROP TABLE IF EXISTS evidence;
DROP TABLE IF EXISTS principals;
-- +goose StatementEnd
