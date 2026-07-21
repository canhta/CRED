# Testing Strategy

The design laws and acceptance criteria in the PRD are assertions, not prose.
Each becomes a named executable suite.

| Law | Test form |
|---|---|
| L1 — no claim without evidence | Property: `∀ writes. claim.evidence ≠ ∅` |
| L2 — model nominates, code decides | Stubbed nominator + adversarial matrix (unit) |
| L3 — deterministic invalidation | Integration, same transaction |
| L5 — access control at recall | Property + adversarial |
| L6 — bi-temporal | Property |

**Consequence of L2:** the model boundary is a list of IDs. That collapses most
"LLM testing" into ordinary unit testing.

---

## Postgres in CI

**Decisive finding: no embedded Postgres can provide pgvector**, and the
official `postgres` image lacks it. Both roads end at `pgvector/pgvector`, so the
"embedded is lighter" argument evaporates. Embedded builds pull stripped binaries
with no PGXS — supporting pgvector would mean compiling against every build × OS
× architecture forever.

**Isolation uses template databases, not transaction rollback.** Rollback breaks
on pooled multi-connection reads, `LISTEN`/`NOTIFY` (commit-only), explicit
commits, and session advisory locks. **An MCP server plus a separate curation
worker is exactly that profile** — rollback would produce false passes.

Template databases also win because `CREATE EXTENSION vector` is per-database:
install once, every clone inherits it.

**Setup:** testcontainers with `pgvector/pgvector:pg17` pinned, one container
with N template-cloned databases — never one container per worker. `withReuse()`
locally; Ryuk disabled and reuse off in CI.

---

## The bi-temporal invariants

Use half-open intervals `[lo, hi)` throughout. It makes "no gaps and no overlaps"
a single equality and eliminates the boundary off-by-one class entirely.

1. Single current claim: `∀ key, valid_time, tx_time: |current| ≤ 1`
2. No valid-time overlap within a transaction slice, per kind
3. Gap policy explicit — catches stale-fact resurrection
4. Supersession acyclic and antisymmetric; one direct successor
5. Supersession monotone in transaction time
6. An evidence-hash change kills **exactly** the dependent set — assert both
   directions; a one-sided test passes trivially
7. Invalidation is transactional, verified with a concurrent reader
8. Replay determinism including IDs and timestamps under an injected clock
9. **Order sensitivity is explicit** — disjoint keys commute, same key does not.
   Assert the difference rather than glossing it as commutativity. This is the
   most common bi-temporal design error.
10. Reopening is disjoint; the gap stays empty
11. Pruning never removes a reachable current claim; pruning is idempotent
12. L1 as a property: no empty-evidence claim is reachable
13. Confidence is monotone, order-independent, idempotent, within `[0,1]`
14. Dedup equivalence — **similarity is not naturally transitive**; a clustering
    step that assumes it will merge unrelated claims

**Generator technique worth more than any single invariant:** draw timestamps
from a pool of about twenty values so boundary collisions are common, and
generate intervals as `(start, duration ≥ 1)` so `lo < hi` cannot be broken by
any shrinker.

---

## Adversarial cases

### Access control

1. Derived ACL is the **intersection**; an empty ACL is reachable by nobody —
   treating it as public is the fail-open bug
2. Three-deep derivation chain re-derives correctly
3. ACL never widens under curation
4. **Merge and dedupe intersect, never union**
5. TTL expiry **denies rather than errors** — an error is an existence oracle
6. Revocation takes effect immediately

### Side channels

7. Counts are computed after ACL filtering
8. **Unauthorized is byte-identical and timing-identical to nonexistent**
9. In-band permission metadata leaks only the intended existence bit
10. Errors never echo restricted text
11. Identifiers are opaque
12. No ordering side channel
13. **Supersession must be evaluated per principal, not globally** — this is an
    architecture decision, not merely a test
14. No channel through token budget or dropped counts

### Poisoning

15. Confidence is server-computed only
16. **Sybil and repetition** — per-principal contribution caps
17. **Near-duplicate flooding just below the similarity threshold** — thresholds
    can always be approached from below
18. Evidence hashes are recomputed server-side
19. **Stored prompt injection reaching another agent** — the highest-severity
    case; shared memory is the delivery vehicle
20. Unicode, homoglyph, zero-width, and bidi normalized at write

### Make it a property, not a list

```
assert retrieved ⊆ ground_truth_visible(graph, principal)
```

Write the oracle **as differently as possible** from production — no SQL, no
shared helpers, `O(n³)` permitted. Correlated bugs are how property-based
security tests silently stop working.

Pair it with a liveness property in a **separate test function**, so nobody
fixes a flaky recall test by loosening the security one.

---

## The nomination boundary

Contract: `Nominate(ctx, scope, []Claim) ([]Nomination, error)`

**A fake carries roughly 95% of tests.** Adversarial inputs are constructible —
you cannot *record* a hallucinated ID, but you can trivially stub one, and that
is the key security surface. A fake couples only to the interface, so it never
rots.

> **Rule: record only what you cannot construct.**

**Tiers.** T1 stubbed, per-commit, ~95% of cases, milliseconds. T2 recorded,
single-digit cassettes, covering the prompt-to-parse seam only — schema
conformance and graceful degradation on truncated, fenced, or prose responses;
**never assert nomination content**. T3 nightly — recall, precision, cost,
latency, cross-model behaviour; gates releases, not commits.

**Recorded-response caveat:** cassettes replay concatenated bodies, not chunk
boundaries. They test the JSON parser and never the streaming accumulator.
Hand-write chunk-boundary tests: mid-UTF-8, mid-token, split `data:`, early
`[DONE]`, two events per chunk.

**LLM-as-judge stays out of CI** — nondeterministic even at temperature 0, costs
money per push, and **model drift under a stable alias moves the baseline with no
code change**.

### The adversarial nomination matrix

Nonexistent ID → drop and warn. **Unauthorized ID → output identical to
nonexistent.** Self-pair. Duplicate and reversed-duplicate → canonicalize.
100k list → bounded. Out-of-scope ID → drop. Malformed input (empty, null, path
traversal, 10KB, SQL) → rejected at parse. Confidence of `-1`, `999`, or **NaN**
— NaN must not silently pass a `> threshold` check. Cycles → deterministic or
rejected, never looping. All-invalid → empty result with success. Partial
validity → the good pair still applies. Injection in a rationale field → treated
as data. Homoglyph IDs must not match.

Most of these collapse into one property: **every output ID lies in the
intersection of the input set and the caller-visible set.**

---

## Shape and budget

| Layer | Share | Time |
|---|---|---|
| Unit (pure, no database) | 50% | <5s |
| Property | 20% | 30–90s |
| Integration | 20% | 1–3 min |
| Adversarial | 7% | 30s |
| Evaluation | 3% | nightly |

**Why 50% pure:** CRED's distinctive logic is pure functions over small data. If
it is not, that is a design smell — **push the temporal and ACL algebra out of
SQL and into testable code; let Postgres store and filter, not decide.**

That single choice is what makes this shape achievable instead of inverting to
70% integration, and it is the largest lever on long-term maintenance burden.

**Budget two to five minutes per commit.** Above roughly ten minutes a solo
maintainer skips the suite, and everything else here becomes irrelevant.

---

## Postgres-specific integration tests

- **Table owners bypass RLS** unless `FORCE ROW LEVEL SECURITY` is set — run the
  suite as the schema-owning role, not as superuser
- Pooled-connection plan reuse
- L3 same-transaction invalidation with a concurrent reader
- pgvector filtering under realistic ACL selectivity
- Migration linting in CI

## Anchor tests

Apply a pure-formatting commit and assert **zero** claims expire. Apply a
pure-insertion-above commit and assert zero expire. Apply a semantic change and
assert exactly the right claims expire.

This test is worth more than any amount of design discussion about anchoring.

## Determinism harness

Run the reconciler five times on identical fixtures in CI and assert
byte-identical output. Catches both model nondeterminism and ordering-dependent
bugs in local code.
