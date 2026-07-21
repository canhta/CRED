-- +goose Up
-- +goose StatementBegin

-- L3's fingerprint ladder, persisted on evidence. Tier 4 already exists as
-- content_sha256 (the raw span hash) and is kept for diagnostics. This adds the
-- two tiers the law actually consults plus the tier-3 fallback:
--
--   tier 1  anchor_symbol_path   heading path / symbol path — survives line moves
--   tier 2  anchor_node_hash     normalized enclosing-node hash — survives reflow
--   tier 3  anchor_window_hash   normalized context-window hash — survives local edits
--
-- An anchor is valid iff tiers 1 and 2 agree; the resolver lives in
-- internal/anchor and the invalidation in internal/curate. These columns are
-- only the storage.
--
-- Pre-existing evidence (seeded or written before this migration) gets an empty
-- symbol path and NULL hashes: it is tier-4-only, and re-anchoring leaves it
-- untouched rather than expiring it, because expiring on tier 4 alone is the
-- exact failure the ladder exists to prevent. A re-seed recomputes the anchor
-- for such rows going forward.
ALTER TABLE evidence
    ADD COLUMN anchor_symbol_path text NOT NULL DEFAULT '',
    ADD COLUMN anchor_node_hash   bytea
        CHECK (anchor_node_hash   IS NULL OR octet_length(anchor_node_hash)   = 32),
    ADD COLUMN anchor_window_hash bytea
        CHECK (anchor_window_hash IS NULL OR octet_length(anchor_window_hash) = 32);

-- Re-anchoring reads live, anchored evidence for one repository and resolves
-- each against the current file. A partial index on the anchored live set keeps
-- that scan small: attestations and tier-4-only rows carry an empty path and are
-- not candidates.
CREATE INDEX evidence_live_anchored
    ON evidence (source_repo, source_path)
    WHERE superseded_at IS NULL AND anchor_symbol_path <> '';

-- 'stale_anchor' joins the supersession reasons: a claim whose evidence failed
-- re-anchoring (tier 1 or 2 disagreed, or resolution was ambiguous) is expired
-- with this reason, so `cred log` shows an L3 expiry as distinct from a dedup
-- merge, a contradiction, or a human `cred forget`.
--
-- The column-level CHECK from migration 00003 is auto-named
-- claims_supersede_reason_check; drop it and replace with a named one carrying
-- the extended set.
ALTER TABLE claims DROP CONSTRAINT IF EXISTS claims_supersede_reason_check;
ALTER TABLE claims
    ADD CONSTRAINT claims_supersede_reason_valid
    CHECK (supersede_reason IS NULL
           OR supersede_reason IN ('duplicate', 'contradiction', 'forgotten', 'stale_anchor'));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE claims DROP CONSTRAINT IF EXISTS claims_supersede_reason_valid;
ALTER TABLE claims
    ADD CONSTRAINT claims_supersede_reason_check
    CHECK (supersede_reason IS NULL
           OR supersede_reason IN ('duplicate', 'contradiction', 'forgotten'));

DROP INDEX IF EXISTS evidence_live_anchored;
ALTER TABLE evidence
    DROP COLUMN IF EXISTS anchor_window_hash,
    DROP COLUMN IF EXISTS anchor_node_hash,
    DROP COLUMN IF EXISTS anchor_symbol_path;
-- +goose StatementEnd
