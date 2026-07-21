-- +goose Up
-- +goose StatementBegin

-- Usage and limits (PRD section 8). Every limit here is a security control
-- first — shared memory with unbounded per-principal write access is a
-- poisoning vector — and a capacity concern second. The policy is pure Go in
-- internal/limit; this migration is only the counters it decides over: "columns
-- on the claim table and a usage table".

-- contributed_by: the principal that contributed a claim. The contribution
-- quota counts accepted claims per principal per window, and this is what it
-- counts. NOT NULL DEFAULT '' so seeded and pre-existing rows carry the empty
-- principal: seeding is deterministic, not a per-principal flood vector, so it
-- is exempt from the quota by construction rather than by a special case.
--
-- ADD COLUMN with a constant default is a metadata-only change on PG11+ — no
-- table rewrite, no long lock — so it is safe on a populated claims table.
ALTER TABLE claims ADD COLUMN contributed_by text NOT NULL DEFAULT '';

-- The quota lookup is "claims this principal has had accepted since a cutoff".
-- A partial index over the contributed set keeps it small; the empty-principal
-- seed rows never participate.
CREATE INDEX claims_contributor_time
    ON claims (contributed_by, recorded_at)
    WHERE contributed_by <> '';

-- The scope-growth bound counts live claims per scope. A partial index on the
-- live set backs that count without scanning superseded history.
CREATE INDEX claims_live_by_scope
    ON claims (scope_kind, scope_value)
    WHERE superseded_at IS NULL;

-- 'pruned' joins the supersession reasons: the scope-growth bound expires the
-- lowest-value live claims in a scope rather than letting it grow unbounded, and
-- `cred log` must show a prune as distinct from a dedup, a contradiction, a
-- human forget, or an L3 stale-anchor expiry. The named constraint from
-- migration 00004 is dropped and replaced with the extended set.
ALTER TABLE claims DROP CONSTRAINT IF EXISTS claims_supersede_reason_valid;
ALTER TABLE claims
    ADD CONSTRAINT claims_supersede_reason_valid
    CHECK (supersede_reason IS NULL
           OR supersede_reason IN ('duplicate', 'contradiction', 'forgotten',
                                    'stale_anchor', 'pruned'));

-- usage_events: the cost-attribution ledger. One row per accountable event —
-- an inference call, a recall, or a denied contribution. Cost attribution
-- answers two questions the PRD names: the hard cost ceiling enforced in code,
-- and "which teams actually use this", which is the per-scope aggregation no
-- competitor exposes.
--
-- No foreign key to principals. This is accounting, not an access surface: a
-- denial must be recordable for any principal id, including one that is not (or
-- not yet) a grantable identity, because a silent drop under the off-the-turn
-- write path (D-017) is exactly how a poisoning attempt would hide (L8). The
-- ledger records what happened; it never gates what may be read.
CREATE TABLE usage_events (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    principal_id    text        NOT NULL,
    scope_kind      text        NOT NULL DEFAULT '',
    scope_value     text        NOT NULL DEFAULT '',
    kind            text        NOT NULL CHECK (kind IN ('inference', 'recall', 'denied')),

    -- Cost dimensions. Zero for a 'denied' or 'recall' row that spent none.
    inference_calls int         NOT NULL DEFAULT 0,
    input_tokens    int         NOT NULL DEFAULT 0,
    output_tokens   int         NOT NULL DEFAULT 0,
    wall_ms         bigint      NOT NULL DEFAULT 0,

    -- Recall dimension: how many claims the assembled package carried.
    package_claims  int         NOT NULL DEFAULT 0,

    -- Set only on a 'denied' row: the machine reason from internal/limit
    -- (contribution_quota, cost_ceiling, recall_rate). This is what makes an
    -- exhaustion loud and queryable rather than a silent no-op.
    denied_reason   text,

    recorded_at     timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT usage_denied_has_reason
        CHECK (kind <> 'denied' OR denied_reason IS NOT NULL)
);

-- Windowed per-principal reads: contribution/cost/recall counts since a cutoff.
CREATE INDEX usage_events_principal_time
    ON usage_events (principal_id, kind, recorded_at);

-- Per-scope cost aggregation for `cred usage` ("which teams use this").
CREATE INDEX usage_events_scope_time
    ON usage_events (scope_kind, scope_value, recorded_at)
    WHERE kind = 'inference';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS usage_events;

ALTER TABLE claims DROP CONSTRAINT IF EXISTS claims_supersede_reason_valid;
ALTER TABLE claims
    ADD CONSTRAINT claims_supersede_reason_valid
    CHECK (supersede_reason IS NULL
           OR supersede_reason IN ('duplicate', 'contradiction', 'forgotten', 'stale_anchor'));

DROP INDEX IF EXISTS claims_live_by_scope;
DROP INDEX IF EXISTS claims_contributor_time;
ALTER TABLE claims DROP COLUMN IF EXISTS contributed_by;
-- +goose StatementEnd
