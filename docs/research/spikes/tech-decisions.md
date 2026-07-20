# Technical Decisions

Consolidated findings from the implementation spikes. Claims marked **verified**
were confirmed against primary sources or executed against a real database
during the spike.

Companion document: [tech-language-and-mcp.md](tech-language-and-mcp.md).

---

## Decisions

| Area | Decision |
|---|---|
| Language | **Go** — official MCP `go-sdk`, v1.0.0 since Sept 2025 with a no-breaking-changes promise |
| MCP transport | stdio + Streamable HTTP. **Never** legacy HTTP+SSE |
| MCP auth | OAuth 2.1 **Resource Server only**. External IdP, no authorization server, no DCR |
| Datastore | PostgreSQL 17+ with pgvector. Nothing else |
| BM25 | `pg_textsearch` (TigerData) — **conditional**, must benchmark first |
| Job queue | **River** — pure SQL tables, no extension required |
| Dedup at v1 | **Exact hash only.** No MinHash/LSH |
| Migrations | Mechanics in-language; **`squawk`** as the CI linter |
| Code anchoring | **tree-sitter** symbol path + normalized AST-node hash |
| Reranking | `BAAI/bge-reranker-v2-m3` (Apache-2.0), top-50 only, opt-in per call |
| Observability | OpenTelemetry emitted; never a bundled trace UI |

---

## Language: Go

**The deciding factor was timing, not language merit.** The TypeScript SDK
(v2 GA 2026-07-28) and Python SDK (v2 GA 2026-07-27) are **both mid
major-version rewrite this month**. Go shipped v1.0.0 in September 2025 and
already tracks the upcoming spec RC.

A solo maintainer cannot absorb two SDK rewrites.

Rust was reconsidered and rejected on better grounds than first assumed: `rmcp`
is at 2.2.0, not pre-1.0. The real disqualifiers are **no server-side OAuth**,
community rather than Anthropic maintenance, and incomplete RC conformance.

### The accepted counter-argument

The AI-agent contributor pool skews heavily to **Python and TypeScript**. Go
trades SDK stability for a smaller pool of likely contributors — which matters
more than usual given that adoption is the strategy. This was not quantified;
the spike lacked search budget.

---

## MCP: a breaking spec revision is imminent

**Spec revision `2026-07-28` is a tagged release candidate**, confirmed three
ways. It **removes the `initialize` handshake, sessions, and SSE resumability**.

> **Rule: persist nothing keyed on MCP session identity.**

### An SDK-default server is non-compliant

**No official SDK enforces RFC 8707 audience matching server-side** (verified).
The specification makes this a **MUST**, and explicitly forbids token
passthrough. Every CRED server must implement the audience check itself, with a
dedicated test.

---

## PostgreSQL

### Vector search and ACL filtering conflict

From the pgvector README (verified):

> "If a condition matches 10% of rows, with HNSW and the default `hnsw.ef_search`
> of 40, only 4 rows will match on average."

**Filters apply after the index scan.** Since access control is enforced at
recall, a user who can read 10% of claims receives roughly 4 results instead of
40 — with no error. Recall silently degrades and reads as "CRED has a poor
memory."

AWS benchmark, 10M vectors, filter selectivity 10–25%:

| Query | pgvector 0.7.4 | 0.7.4 at ef=200 | 0.8.0 iterative |
|---|---|---|---|
| Category filter | 10% complete | **0%** | 100% |
| Complex filter | 1% complete | **0%** | 100% |

Note the middle column: **raising `ef_search` made completeness worse.**
Brute force is not the fix. Iterative scans are, and they are **off by default**:

```sql
SET hnsw.iterative_scan = relaxed_order;  -- ~1.3–1.8x faster than strict_order
SET hnsw.max_scan_tuples = 20000;
```

**Caveat that must not be lost:** no published benchmark exists below ~0.1%
selectivity. These numbers do not prove iterative scan solves highly selective
ACL filtering. **Benchmark against real ACL shapes before relying on it.**

### Self-hosting removes the hard part of multi-tenancy

One instance per organization means tenant count is 1. Partition-per-tenant
planner limits (a few thousand), per-session relation metadata growth, and
lock-manager contention **do not apply**. Only intra-organization ACL
selectivity remains — orders of magnitude easier.

### Embedding model migration

Verified working end to end. Parent column typed as **unspecified `vector`**,
partitioned by `model_id`, with per-partition dimension-specific expression
indexes:

```sql
CREATE TABLE emb (model_id int, item_id bigint, emb vector,
                  PRIMARY KEY (model_id, item_id)) PARTITION BY LIST (model_id);
CREATE TABLE emb_m1 PARTITION OF emb FOR VALUES IN (1);
CREATE INDEX ON emb_m1 USING hnsw ((emb::vector(384)) vector_l2_ops);
```

Partition pruning and HNSW index scan both engage. `CREATE INDEX CONCURRENTLY`
works per partition. Retiring a model is `DETACH PARTITION`.

Every alternative is blocked (verified): `ALTER COLUMN TYPE` across dimensions
**fails on populated data**; `CREATE OR REPLACE VIEW` **cannot change a column
type**.

**Caveat:** the planner only chooses the expression index at scale — ignored at
500 rows, used at ~50k. Verify with `EXPLAIN` on realistic data.

### Re-embedding requires retained source text

> Every project that can migrate stores content alongside vectors. **A
> vectors-only store can never migrate.**

This is decided at ingest, long before the first model swap. `Evidence` must
retain normalized extracted text, not only a pointer.

### Content hashing has an immutability trap

```sql
GENERATED ALWAYS AS (sha256(convert_to(body,'UTF8'))) STORED
-- ERROR: generation expression is not immutable
```

`sha256` is immutable; **`convert_to` is stable** — it depends on server
encoding. An `IMMUTABLE` wrapper works but **asserts a guarantee Postgres does
not make**. If encoding ever changes, hashes shift and invalidation fires
incorrectly across the corpus.

### Model-version safety

Learned from Khoj, Dify, Open WebUI, and Mem0, each of which ships a silent
failure here:

- `embedding_model_id` **NOT NULL** on every vector row; no defaults, no
  backfill-with-guess.
- **Reads must filter on it.** A per-row model column that reads ignore is
  worthless — Khoj's exact bug.
- Enforce one PRESENT and one FUTURE as **database constraints**, per Onyx:

```sql
CREATE UNIQUE INDEX ... ON search_settings (status) WHERE status = 'PRESENT';
CREATE UNIQUE INDEX ... ON search_settings (status) WHERE status = 'FUTURE';
```

- **Reads never touch the index being built.** A structural invariant beats a
  progress threshold. Onyx is the only surveyed project where this failure is
  impossible rather than merely unlikely.
- Norm invariant as a database CHECK, catching provider drift at write time.

### Row-level security

Well-written RLS on an indexed column is nearly free; naive RLS is catastrophic
**and silent** — correct results, a thousand times slower.

Footguns that matter:

- **Table owners bypass RLS** unless `FORCE ROW LEVEL SECURITY` is set.
- **Forgetting `ENABLE` fails open**; enabling without a policy fails closed. The
  dangerous mistake is forgetting to enable, making RLS coverage a
  **migration-time invariant that must be tested**.
- **PgBouncer with a shared role makes `current_user` useless.** Identity must
  live in application-controlled session variables.
- Wrap policy function calls so they become an initPlan evaluated once per
  statement rather than once per row.

**Rejected advice:** marking functions `LEAKPROOF` for speed. It tells the
planner it may run them on rows the caller cannot see, destroying the guarantee
RLS exists to provide.

---

## Job queue

**River** for Go: pure SQL tables (no extension — critical, since RDS, Cloud SQL,
and Neon restrict extensions), transactional enqueue, unique jobs, periodic jobs,
dead-lettering, and a web UI that removes any need to build an admin surface.

**Transactional enqueue is the load-bearing feature.** When `remember` writes a
claim, the curation job must commit atomically with it, or claims exist that are
never curated.

### The queue table is the MVCC worst case

Roughly four dead tuples per job minimum. At 1,000 jobs/sec that is ~14M dead
tuples per hour against a live set of a few thousand.

Required table settings:

```sql
ALTER TABLE jobs SET (
  fillfactor = 80,                          -- leave room for HOT updates
  autovacuum_vacuum_scale_factor = 0.0,
  autovacuum_vacuum_threshold = 1000,       -- absolute, not proportional
  autovacuum_vacuum_cost_delay = 0
);
```

Partial index on the dequeue predicate; keep mutable non-state columns
unindexed so heartbeat updates stay HOT.

### The real outage cause is the xmin horizon

**Vacuum tuning is irrelevant if the xmin horizon is held back.** One
idle-in-transaction session anywhere in the cluster pins it, and dead tuples
accumulate without bound. Four holders must be monitored: long-running backends,
abandoned replication slots, orphaned prepared transactions, and standbys with
`hot_standby_feedback`.

```sql
ALTER SYSTEM SET idle_in_transaction_session_timeout = '60s';
ALTER SYSTEM SET transaction_timeout = '300s';
```

**Architectural consequence:** the queue must not share a database with
analytics workloads. Someone else's long transaction bloats it invisibly.

### LISTEN/NOTIFY is an optimization, never the correctness path

8000-byte payload limit; **no durability** — delivered only to sessions
listening at that moment, with no replay; **`LISTEN` is never supported under
PgBouncer transaction pooling**; and at high commit rates every NOTIFY-bearing
commit serializes behind a cluster-wide lock. Poll for correctness, notify for
latency, and make it disableable.

### Retention

Delete on completion; keep a **time-partitioned archive** and drop partitions.
`DELETE` of 10M rows creates 10M dead tuples — cleanup that generates the
problem it cleans. Partition the archive by **time, not state**: state
partitioning makes every transition a cross-partition DELETE+INSERT.

---

## LLM integration

The model only nominates; code decides. That principle is not merely a design
preference — **provider behaviour makes it mandatory.**

**No provider validates structured output server-side** (verified). Together,
Fireworks, and DeepInfra all document returning **incomplete, invalid JSON** when
`max_tokens` is exhausted. Constrained decoding guarantees a valid *prefix*, not
valid JSON.

Specific traps:

- **OpenRouter silently ignores `response_format` by default**, returning 200
  with free prose. Always set `provider: {require_parameters: true}`.
- **Groq honours `strict` on only two models**, and it **defaults to `false`**.
- **Anthropic's OpenAI-compatibility shim ignores `response_format` entirely.**
  Use the native API.
- **llama.cpp's README documents a request shape its code does not read** —
  producing a grammar that matches any JSON, with no error.
- `oneOf` compiles as `anyOf` in llama.cpp and Ollama: **XOR is never enforced**.
  Numeric bounds are dropped for `"number"` but honoured for `"integer"`.

**Required:** validate every schema locally, gate parsing on
`finish_reason != "length"`, probe unknown endpoints once and cache the result,
and emit telemetry on which capability tier served each call — silent
degradation to prompt-only has no error signal.

---

## Embeddings

pgvector index limits (verified): `vector` and `halfvec` hold up to 16,000
dimensions, but **HNSW indexes only 2,000** — 4,000 via `halfvec`, 64,000 with
binary quantization. A 3072-dimension embedding **cannot** be HNSW-indexed as
`vector`.

**Approach:** a Matryoshka-capable model, stored as `halfvec` truncated to 768 or
1024 for the indexed path, optionally retaining full width unindexed for exact
rerank. This resolves the index ceiling, storage cost, and two-stage retrieval
together.

Normalize at write time, use inner product, and enforce the norm with a CHECK
constraint.

**Both hosted and local must be supported, with neither mandatory.** Requiring an
API key breaks `docker compose up`; requiring local inference ships a GPU
dependency.

**The local path is settled and runs in-process.** `onnx-gomlx` on gomlx
`simplego`, `CGO_ENABLED=0`, no GPU and no sidecar — verified bit-comparable to
ONNX Runtime and exactly conformant on tokenization. It is 9–16x slower, which
is irrelevant to interactive recall and decisive for bulk ingestion. See
[Go embeddings and tokenizer](go-embeddings-tokenizer.md) and D-008.

Two constraints that follow, both easy to get silently wrong:

- The WordPiece character-class tables are **generated by probing the pinned
  HuggingFace tokenizer**, never from Go's `unicode` package. The target is
  byte-identity with the tokenizer that trained the model, not Unicode
  correctness — and the two disagree on 824 codepoints.
- The CI guard for CGO is `go list -f '{{if .CgoFiles}}...' -deps ./...` run
  against the real shipping build command. `go tool nm | grep cgo` reports
  `_cgo_` symbols with CGO both on and off, so a guard built on it passes while
  broken.

**MTEB rankings are not evidence for code retrieval** — the benchmark is heavily
overfit and its retrieval subset is largely non-code. Build a golden set from
real repositories before choosing.

---

## Hybrid ranking

**Use Reciprocal Rank Fusion**, not a weighted sum of normalized scores. BM25
scores are unbounded and corpus-dependent; cosine is bounded. Per-query
normalization makes weights query-dependent and untunable.

```
score += 1.0 / (k + rank)     -- k defaults to 60, rank is 1-based
```

Failure modes, all silent:

- **Unequal arm depth** systematically penalizes anything found by one arm only.
  Assert both arms return the same `k`.
- **Direction error** — pgvector `<=>` is a *distance* (lower is better), BM25 is
  a *score* (higher is better). Fusing without normalizing direction ranks the
  worst results first, and still returns topically related results, so it reads
  as noise rather than inversion.
- **0-based rank** shifts scores ~1.6% — enough to reorder near-ties, never
  enough to notice.

Alarm when more than 95% of the final top-10 comes from a single arm: that is
single-arm search with extra latency.

---

## BM25 extension

Core PostgreSQL 18 and 19 ship **zero** improvements to relevance ranking, so an
extension is mandatory.

| Extension | License | Note |
|---|---|---|
| ParadeDB `pg_search` | **AGPL-3.0** | Most mature; AGPL blocks enterprise self-host adoption |
| **`pg_textsearch`** (TigerData) | **PostgreSQL** | v1.0 March 2026; permissive |
| VectorChord-bm25 | AGPLv3 / ELv2 | Best CJK tokenizers; restrictive licence |
| `rum` | PostgreSQL-like | **Not BM25** — accelerates `ts_rank` only |

AGPL does not infect CRED's code — the extension runs inside Postgres and is
not linked. But shipping an AGPL extension in the default compose file puts it
in front of enterprise AGPL policies, which is friction precisely in the target
segment.

**`pg_textsearch`, conditionally.** Real limitations: no phrase queries (no term
positions), OR-only semantics at v1.0, no highlighting, and — directly relevant —
**scores computed independently per partition are not comparable**, which
collides with time partitioning.

**Every published benchmark is vendor-authored** and weakens on inspection:
pg_textsearch's advantage falls from 11.7x to 2.4x as query terms rise from 1 to
4, and it loses on index build time. Programming queries have many terms.
**Benchmark on a real corpus before committing.**

---

## Testing and evaluation

- **`promptfoo`** (MIT, runs fully locally) for evaluation and **red-teaming** —
  the only tool covering prompt-injection fuzzing. Note it is **owned by
  OpenAI**; MIT permits forking, but the roadmap risk is real for a
  vendor-neutral project.
- Deterministic reconciler test: run five times on identical fixtures in CI and
  assert identical output. Catches both model nondeterminism and
  ordering-dependent bugs.
- Anchor test: apply a pure-formatting commit and assert **zero** claims expire;
  apply a semantic change and assert the right ones do.
- The unbuyable part: roughly 200 hand-built `(query → expected claim)` pairs
  from real repositories, before v1 feature freeze.

---

## Do not build

Explicitly out, for a solo maintainer:

- An admin UI — River and OTel exporters cover the operator surface
- A second database backend — one Postgres is the architectural pitch
- A plugin system — every extension point is an API that can never change
- A bundled identity provider — owning CVE response for an IdP is untenable
- An LLM framework dependency — one thin interface, three implementations
