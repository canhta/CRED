# Usage page — design

- **Date:** 2026-07-21
- **Status:** Approved by operator, pending implementation plan.

## Problem

The console's side nav already links `/usage` (`web/src/router.tsx`), but it
renders a bare `<Placeholder title="Usage" />`. Everything the page needs
already exists in the engine — `cred usage` (`internal/cli/usage.go`) prints a
principal's limit headroom, denied-contribution count, per-scope cost, and
per-scope growth by calling store methods that already exist
(`internal/store/pg/usage.go`) and running them through the pure decision
functions in `internal/limit`. Nothing in the write or read path needs to
change; this is a new HTTP projection of data the CLI already reports, plus
the page that renders it — the same relationship `internal/api/recall.go` has
to `internal/recall`.

## Approach

**One `GET /api/usage` endpoint, one page.** Rejected alternatives:

- **Split into `/api/usage/limits` (mine) vs `/api/usage/scopes` (org-wide).**
  More REST-ish, but two round trips for one page, and `cred usage` itself
  already treats them as one report. Not worth it for a single-principal
  self-hosted console.
- **Auto-refresh / poll the page.** Fits a "live headroom" feel, but nothing
  else in the console polls — Claims loads once, Recall loads on submit. A
  one-line addition later (`refetchInterval`) if it turns out to matter.
  Skipped for v1.

## Backend

New file `internal/api/usage.go`, same shape as `recall.go`: a query struct, a
handler method on `*server`, a response-building helper.

```go
// GET /api/usage?scopes=N   (default 10, matches `cred usage --scopes`)
```

Handler body mirrors `runUsage` (`internal/cli/usage.go:22-129`) exactly,
swapping tabwriter output for a JSON response:

1. `principalFrom(c)` for identity — same seam every other handler uses.
2. `s.store.PrincipalWindowState(...)` with window starts from
   `limit.WindowStart(now, cfg.Limits.*Window)` — one call for all three
   windowed counters, exactly as the CLI does.
3. `limit.Contribution(...)`, `limit.Cost(...)`, `limit.RecallRate(...)` —
   the same pure decision functions the enforcement path uses, so what the
   page shows is what the engine actually decides over, never a separate
   estimate.
4. `s.store.DeniedInWindow(...)`, `s.store.UsageByScope(...)`,
   `s.store.ScopeSizes(ctx, topN)` — unchanged store calls.
5. Assemble `UsageResponse` and `c.JSON(http.StatusOK, ...)`.

New types in `internal/api/types.go` (tygo-generated into
`web/src/api/types.gen.ts` — never hand-written per `web.md` §3):

```go
// LimitStatus is one limit's window state: what's used, the ceiling, and the
// remaining headroom before it binds. Remaining reuses internal/limit's own
// sentinel (-1 = unlimited) rather than a new one, and Ceiling <= 0 means the
// control is disabled — the same convention internal/limit.Decision already
// carries, so the frontend formats it once instead of the API inventing a
// second "off"/"unlimited" string convention on top of it.
type LimitStatus struct {
    Window    string `json:"window"`     // e.g. "1h0m0s"
    Used      int    `json:"used"`
    Ceiling   int    `json:"ceiling"`
    Remaining int    `json:"remaining"`
    Allowed   bool   `json:"allowed"`
    Reason    string `json:"reason"`     // internal/limit.Reason; "" when allowed
}

type ScopeCost struct {
    Scope        Scope `json:"scope"`
    Calls        int   `json:"calls"`
    InputTokens  int   `json:"input_tokens"`
    OutputTokens int   `json:"output_tokens"`
}

type ScopeGrowth struct {
    Scope     Scope `json:"scope"`
    Live      int   `json:"live"`
    Ceiling   int   `json:"ceiling"`
    NextPrune int   `json:"next_prune"`
}

type UsageResponse struct {
    Principal          string        `json:"principal"`
    Contribution        LimitStatus  `json:"contribution"`
    Cost                 LimitStatus `json:"cost"`
    InputTokensUsed      int         `json:"input_tokens_used"`
    InputTokensCeiling   int         `json:"input_tokens_ceiling"`
    Recall               LimitStatus `json:"recall"`
    DeniedWindow         string      `json:"denied_window"`
    Denied               int         `json:"denied"`
    CostByScope          []ScopeCost `json:"cost_by_scope"`
    ScopeGrowth        []ScopeGrowth `json:"scope_growth"`
}
```

Input tokens get their own two fields rather than a second `LimitStatus`
row, because `internal/limit.Cost` doesn't compute a token-remaining figure —
it gates on tokens as a boolean and reports remaining only for calls
(`internal/limit/limit.go:128-137`). Inventing a token "remaining" on the API
side would be a number the engine doesn't actually decide over.

Registered in `api.go`: `api.GET("/usage", s.usage)`, alongside the other
three routes.

## Frontend

New file `web/src/pages/UsagePage.tsx`, replacing the `usageRoute`'s
`Placeholder` in `router.tsx`. Follows `RecallPage.tsx` and `ClaimsPage.tsx`
conventions throughout — no new UI idioms.

- `web/src/api/routes.ts`: add `usage: () => \`${API_BASE}/usage\``.
- `web/src/api/client.ts`: add `getUsage(params?: { scopes?: number })`.
- `web/src/api/hooks.ts`: add `useUsage(params)`, `queryKeys.usage(params)`.
- Page layout: `Layout` + `LayoutHeader` ("Usage" heading, a "Refresh" button
  that invalidates the query — no polling) + `LayoutContent`.
- Top: a `Stat` row reusing the exact pattern from `RecallPage`'s
  `AccountingBar` — three limits (contribution, inference cost + tokens,
  recall), each a `ProgressBar` (used/ceiling) with a warning `Badge` when
  `allowed: false`. `Remaining: -1` renders as "unlimited" text instead of a
  bar, `Ceiling <= 0` renders as "off" — client-side formatting of the same
  sentinels `cred usage`'s `ceilingLabel`/`remainingLabel` already encode
  (`internal/cli/usage.go:131-151`), not duplicated server-side.
- If `denied > 0`: a warning banner above the tables. Exact Astryx component
  TBD at implementation time via `astryx search` — deferred to the plan, not
  guessed here.
- Two `Table`s below, styled like `ClaimsPage`'s: **cost by scope** (scope,
  calls, input/output tokens) and **scope growth** (scope, live, ceiling,
  next prune).
- Loads on mount via `useUsage()` (same as `ClaimsPage`), not gated behind a
  submit action (unlike `RecallPage`, which is query-driven).

## Testing

Precedent check: `internal/api` has **no existing Go-level HTTP tests** —
`claims.go` and `recall.go` are covered only by frontend tests against a
mocked generated client, plus manual end-to-end verification noted in their
commit messages ("Verified end to end: real Postgres-backed data..."). This
page follows the same precedent rather than introducing a new Go testing
pattern unilaterally:

- Frontend: `web/src/pages/Usage.test.tsx`, mocking the generated client
  (same pattern as `Recall.test.tsx`/`Claims.test.tsx`) — asserts the three
  limit stats render, a scope-cost row renders, a scope-growth row renders,
  and the exhausted/denied state shows its warning.
- Backend: no new Go test file planned, consistent with `claims.go`/
  `recall.go`. Manual end-to-end verification (real Postgres-backed data,
  live browser) before calling this done, per this repo's established
  practice for console features.
- `internal/limit` and `internal/store/pg/usage.go` already have their own
  test coverage (`limit_test.go`, exercised via `cred usage` and
  `internal/curate/integration_test.go`) — this feature adds no new logic to
  either, only a projection, so no new tests are needed there.

## Out of scope

- Switching principals from the Usage page (that's `Settings`' job, not
  built yet).
- Any write path (denial appeals, quota overrides) — this page is read-only,
  matching `cred usage`.
- Polling / live refresh (see Approach).
