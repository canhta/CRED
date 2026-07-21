-- +goose Up
-- +goose StatementBegin

-- The write path adds two columns to claims. River's own tables are created by
-- its migrator (rivermigrate), not here, because River owns their schema and
-- versions them independently — mixing them into goose would fork that
-- ownership. See internal/store/pg/river.go.

-- Exact-hash deduplication (D-010: exact hash at v1, fuzzy deferred). The hash
-- is over the *normalized* statement, computed in Go — never a GENERATED
-- column, for the same reason evidence.content_sha256 is not one: sha256 is
-- immutable but the normalization is application logic, and a generated
-- expression cannot call it. NULL for seeded claims, which are deduplicated by
-- their evidence content hash already and must not be swept into the write
-- path's dedup.
ALTER TABLE claims
    ADD COLUMN statement_sha256 bytea
    CHECK (statement_sha256 IS NULL OR octet_length(statement_sha256) = 32);

-- The dedup lookup is "live claims sharing a normalized statement". A partial
-- index on the live set keeps it small — superseded rows are kept forever but
-- never dedup candidates.
CREATE INDEX claims_live_statement_hash
    ON claims (statement_sha256)
    WHERE superseded_at IS NULL AND statement_sha256 IS NOT NULL;

-- Why a claim was superseded, so `cred log` can tell a duplicate merge from a
-- contradiction from a human `cred forget`. The supersession machinery
-- (internal/temporal) closes the interval; this column records the reason.
--
-- 'forgotten' is the one value that may carry a NULL superseded_by: forgetting
-- expires a claim without a replacement, which the claims_supersession_is_atomic
-- constraint already permits (superseded_by NULL is always allowed).
ALTER TABLE claims
    ADD COLUMN supersede_reason text
    CHECK (supersede_reason IS NULL
           OR supersede_reason IN ('duplicate', 'contradiction', 'forgotten'));

-- A reason without an expiry, or an expiry from the write path without a
-- reason, is a bug worth catching at write time rather than as a puzzling row.
-- Seeded supersession (a changed doc chunk) predates this column and leaves
-- both NULL/non-NULL split, so the constraint only binds when a reason is set.
ALTER TABLE claims
    ADD CONSTRAINT claims_supersede_reason_needs_expiry
    CHECK (supersede_reason IS NULL OR superseded_at IS NOT NULL);

-- The evidence_live_chunk unique index (migration 00001) enforces one live
-- evidence row per (repo, path, chunk_ordinal). That is exactly right for
-- SEEDED chunks — it is what makes re-seeding idempotent — but wrong for the
-- write path, where many claims legitimately point at different spans of the
-- same file, and an attestation has no chunk at all. Two written evidence rows
-- for the same file would otherwise collide on ordinal 0.
--
-- So chunk_ordinal becomes nullable and the uniqueness only binds when it is
-- set: seeded evidence carries an ordinal (0, 1, 2, …) and stays unique;
-- write-path evidence carries NULL and is exempt.
ALTER TABLE evidence ALTER COLUMN chunk_ordinal DROP NOT NULL;

DROP INDEX evidence_live_chunk;
CREATE UNIQUE INDEX evidence_live_chunk
    ON evidence (source_repo, source_path, chunk_ordinal)
    WHERE superseded_at IS NULL AND chunk_ordinal IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS evidence_live_chunk;
CREATE UNIQUE INDEX evidence_live_chunk
    ON evidence (source_repo, source_path, chunk_ordinal)
    WHERE superseded_at IS NULL;
-- Restoring NOT NULL fails if any write-path evidence (NULL ordinal) exists;
-- that is acceptable for a down migration in development.
ALTER TABLE evidence ALTER COLUMN chunk_ordinal SET NOT NULL;

ALTER TABLE claims DROP CONSTRAINT IF EXISTS claims_supersede_reason_needs_expiry;
ALTER TABLE claims DROP COLUMN IF EXISTS supersede_reason;
DROP INDEX IF EXISTS claims_live_statement_hash;
ALTER TABLE claims DROP COLUMN IF EXISTS statement_sha256;
-- +goose StatementEnd
