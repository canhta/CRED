# CRED

**C**laims — **R**ights — **E**vidence — **D**erivation

> A claim lives only while its evidence does.

Evidence-governed memory for AI agents.

---

## What this is

CRED is an open-source, self-hostable memory layer that agents connect to over
MCP. They retrieve what an organization already knows before starting work, and
contribute what they learn as they finish it.

Four ideas carry the whole design:

- **Claims** — the atomic unit of knowledge. Small, typed, independently
  expirable.
- **Rights** — access control evaluated at recall, failing closed. A claim
  derived from several sources inherits the **intersection** of their
  permissions, never the union.
- **Evidence** — no claim exists without a pointer to what produced it. Human
  attestation counts; orphan claims do not.
- **Derivation** — where a claim came from, and how its permissions were
  inherited, is always reconstructible.

### The difference

Other systems decide what to trust from **usage signal** — a claim becomes
trusted because it is retrieved and upvoted. That lags reality: a claim that was
true and became false stays trusted until enough negative signal accumulates.

CRED decides from **evidence**. When a source changes, every claim resting on it
is invalidated — no inference call, no waiting for signal, and the reason is a
diff rather than a score.

## Status

**Read path plus the automatic write path.** Discovery is complete and
documented, including the evidence that contradicts parts of the original
thesis.

This build seeds claims from a repository's own documentation, retrieves them
through MCP and a CLI, and now **contributes** them too. Contribution follows
[D-017](docs/research/decision-log.md): automatic **nomination**, not automatic
storage. A model proposes candidate claims into a constrained schema;
deterministic code decides what is written. Extraction runs **off the turn** on
a River worker, so an agent is never blocked waiting for it. Every write is
visible (`cred log`) and reversible (`cred forget`).

Two things are still deliberately absent — `revise`, `confirm`, and the
LLM-driven contradiction reconciler — plus everything under
[what is not built](#what-is-not-built-yet).

The **read path and explicit `remember` require no API key, no provider choice,
and no vector-store decision** — embeddings run in-process in pure Go with
`CGO_ENABLED=0`. Only the automatic-nomination worker (`cred curate`) calls a
model, and only it needs a key.

## Quick start

```sh
docker compose up -d db     # PostgreSQL 17 + pgvector, on 127.0.0.1:5433
go build -o cred .
./cred migrate              # apply the schema and the River queue tables
./cred seed .               # index this repository's own documentation
./cred recall "how is access control evaluated"
./cred remember "we fused the two retrieval arms with RRF at k=60"  # no key
./cred log                  # see what has been written
```

`./cred doctor` checks every part of the installation and names the fix for
anything broken.

The first run downloads the embedding model once (~127 MiB, pinned by SHA-256)
into your user cache directory. Set `CRED_MODEL_DIR` to point at a directory
that already has `model.onnx`, and `CRED_ALLOW_MODEL_DOWNLOAD=false` to require
it.

### Connecting an agent

```sh
claude mcp add cred -- /absolute/path/to/cred serve
```

Two tools: `recall` (read-only) and `remember` (explicit contribution by
attestation). Recall output is fenced as data with an explicit warning, never
interpolated into a prompt — ingested content is untrusted (L8). `remember`
stores the statement the agent asserts, attributed to the calling principal;
the write is confirmed in-band and is reversible with `cred forget`.

For the **automatic** path — extracting claims from the work as it happens — see
[contributing knowledge](#contributing-knowledge-the-write-path) below. It is a
hook plus a background worker, not part of `serve`.

## Commands

| Command | What it does |
|---|---|
| `cred migrate` | Apply database migrations (CRED schema + River tables). Reports a partial application honestly |
| `cred seed <path>` | Index `AGENTS.md`, `CLAUDE.md`, `README.md`, `.cursorrules`, `.windsurfrules` and `docs/**/*.md`. Idempotent |
| `cred recall <query>` | Retrieve claims, showing what each retrieval arm contributed |
| `cred remember <text>` | Contribute a claim by attestation. Deterministic, no API key |
| `cred capture` | Enqueue captured material for automatic extraction. The hook entry point; returns immediately |
| `cred curate` | Run the background worker: nominate off the turn (needs a key), then deduplicate |
| `cred log` | Show recent writes — live, superseded, or forgotten (D-016) |
| `cred forget <id>` | Reverse a write by expiring its claim (D-016) |
| `cred serve` | Run the MCP server over stdio (`recall` + `remember`) |
| `cred doctor` | Check the installation; every failure names its fix |

`cred recall` prints the score decomposition on purpose. The most reliable
failure mode across every surveyed memory system is silent write acceptance
followed by empty reads — thousands of documents visible in a UI and zero
results from search, with no way to tell why. `cred log` is the same instrument
for the write side: every automatic write is inspectable, never silent.

## Configuration

Every variable has a working default, so the commands above need no `.env`.

| Variable | Default | Meaning |
|---|---|---|
| `DATABASE_URL` | `postgres://cred:cred@127.0.0.1:5433/cred?sslmode=disable` | The one datastore (L7) |
| `CRED_MODEL_DIR` | user cache directory | Directory holding `model.onnx` |
| `CRED_ALLOW_MODEL_DOWNLOAD` | `true` | Fetch the model on first run |
| `CRED_PRINCIPAL` | `local` | Identity recall is evaluated against |
| `CRED_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `CRED_LOG_FORMAT` | `text` | `text` or `json` |
| `CRED_AUTO_CAPTURE` | `true` | Automatic nomination on `capture`; opt out with `false` |
| `CRED_LLM_API_KEY` | — | Model key for `cred curate` (falls back to `ANTHROPIC_API_KEY`) |
| `CRED_LLM_MODEL` | `claude-opus-4-8` | Model id the nominator uses |

There is deliberately no `.env` file and no `.env.example`. Every variable has a
working default, the binary loads no dotenv file, and a `.env.example` you must
copy first is a step — and steps cost users (PRD acceptance criterion 11). Set
`CRED_LLM_API_KEY` (or `ANTHROPIC_API_KEY`) in your environment only when you run
the automatic-nomination worker; nothing else needs configuring.

## Architecture, and the laws it encodes

The design laws in [the PRD](docs/product/prd.md) are enforced by package
boundaries and linters wherever that is possible. A law that only exists in a
comment is a law that will be broken by someone in a hurry.

```
main.go                    thin: signals, calls internal/cli
internal/
  cli/                     migrate, seed, recall, remember, capture, curate, log, forget, serve, doctor
  config/                  CRED_* resolution
  claim/                   Claim, Evidence, Principal, ACL, Interval
  temporal/                bi-temporal algebra. PURE
  acl/                     ACL intersection algebra. PURE
  recall/                  retrieval orchestration, RRF fusion
  seed/                    documentation chunking and ingestion
  nominate/                the LLM boundary: Nominator + fake + Anthropic adapter
  curate/                  the write executor and the River workers
  store/
    migrations/            *.sql, embedded via embed.FS
    pg/                    the ONLY package importing pgx
  embed/                   onnx-gomlx forward pass
    wordpiece/             the tokenizer and its generated tables
  mcpsrv/                  the MCP tool surface
  obs/                     slog setup, telemetry attribute constants
```

Three of these are load-bearing rather than tidy:

**`internal/temporal` and `internal/acl` are pure.** They import no database
driver and take no connection. `depguard` fails the build if that changes, and
the rule is verified against a deliberate bad import rather than assumed.

**L5 is never a SQL predicate.** No exported function in `internal/store/pg`
takes a principal. The store returns rows; `internal/acl` computes
`claim.acl ⊆ ⋂(evidence_i.acl)` in Go. This costs a round trip of rows Postgres
could have discarded. It is affordable at one instance per organization, and the
alternative is the known silent-failure path — pgvector filtering under ACL
selectivity returns 4 results where 40 were expected, with no error.

**The principal type lives in the engine.** D-014's standing check applies to
CRED itself: grep the engine for the principal type, and if it only appears in a
client package, the retreat has already happened. This slice ships one
principal, and `principals`, `claim_acl` and `evidence_acl` are real tables from
the first commit.

**The model nominates, code decides — structurally (L2).** `internal/nominate`
emits candidate claims and holds no store: `depguard` forbids it from importing
`internal/store`, so an extractor with a database handle does not compile. The
deterministic write executor lives on the other side of that boundary, in
`internal/curate`. Evidence is materialised from the trusted source, never from
the model's output — a candidate whose quote is not a verbatim span of the
source is dropped, not stored (L1). Automatic does not mean unvalidated: every
model response is validated locally and gated on `stop_reason != "max_tokens"`,
because constrained decoding makes a truncated response a valid JSON prefix that
parses cleanly and is silently wrong.

## Contributing knowledge (the write path)

There are two ways knowledge is written, and they differ in what they cost.

**Explicit — `remember`, by attestation.** `cred remember "..."` and the MCP
`remember` tool store the statement a person asserts. Human attestation is
evidence (L1), so the assertion is its own evidence and the principal is the
attester. This calls no model, needs no API key, and returns the claim id
immediately.

**Automatic — nomination, off the turn.** Matching the shipped Mem0 pattern
(D-017), an agent hook captures material and enqueues a job; a background worker
extracts candidate claims from it and code decides what to store. It defaults to
on and requires a model key, but it never blocks the agent's turn — the trigger
returns as soon as the job is durably enqueued.

```
hook → cred capture → [River queue] → cred curate → nominate → validate → write → dedup
        (returns now)                  (worker, off the turn, needs a key)
```

A documented hook example for Claude Code — capture Bash results and the session
summary, exactly Mem0's trigger points, defaulting to on with a `CRED_AUTO_*`
opt-out:

```jsonc
// .claude/settings.json — CRED automatic capture (opt out with CRED_AUTO_CAPTURE=false)
{
  "hooks": {
    "PostToolUse": [{
      "matcher": "Bash",
      "hooks": [{
        "type": "command",
        // The tool result on stdin is enqueued and control returns at once;
        // extraction happens later in `cred curate`, off the turn.
        "command": "jq -r '.tool_response.stdout // empty' | cred capture --repo \"$CLAUDE_PROJECT_DIR\" --trigger tool_result"
      }]
    }],
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": "jq -r '.transcript_path' | xargs tail -c 8000 | cred capture --repo \"$CLAUDE_PROJECT_DIR\" --trigger session_end"
      }]
    }]
  }
}
```

Then run the worker alongside the server:

```sh
export CRED_LLM_API_KEY=sk-...       # or ANTHROPIC_API_KEY
cred curate                          # drains the queue; nominate off the turn, then dedup
```

This is a documented example, not a shipped harness: the trigger model and the
`CRED_AUTO_*` opt-out are the parts CRED provides; wiring them into a specific
agent is one settings file.

## Verified results

Reproduced in this repository, not carried over from the spikes.

**Tokenizer: 212,291 of 212,291 inputs match the reference exactly.** 43 curated
edge cases, 17,688 fuzz strings, and 194,560 single-codepoint probes, all diffed
against HuggingFace `tokenizers` 0.22.2. The character-class tables are
generated by probing that pinned release — never from Go's `unicode` package,
because the goal is byte-identity with the tokenizer that trained the model, not
Unicode correctness. Regenerate with `go generate ./internal/embed/wordpiece/...`.

**Embeddings match the recorded reference vector.** The CLS-pooled, normalized
embedding agrees with the value in
[the spike](docs/research/spikes/go-embeddings-tokenizer.md) — itself
cross-checked against ONNX Runtime at cosine 1.00000000 — to within 1e-6.

**`depguard` genuinely fails.** A `pgx` import in `internal/temporal`,
`internal/acl`, or `internal/recall`, and a `database/sql` import in
`internal/temporal`, were each added and confirmed to fail the build, then
reverted. This also closes an item the conventions spike left UNVERIFIED:
depguard's `files:` glob negation does behave as written in v2.12.2, since
`internal/store/pg` imports pgx while the full lint run is clean.

**The schema applies to a real PostgreSQL 17 + pgvector 0.8.5.** Including the
`halfvec` column partitioned by model, the per-partition
`hnsw ((embedding::halfvec(384)) halfvec_ip_ops)` expression index, and the
norm `CHECK`.

### Measured recall latency

Stated with its conditions, because a prior measurement on this project was
taken under CPU contention and had to be retracted.

**Conditions.** Apple M1 Pro, 10 cores, 16 GB, darwin/arm64, Go 1.26.0,
`CGO_ENABLED=0`. PostgreSQL 17 + pgvector 0.8.5 in Docker Desktop on the same
machine. 1,247 claims seeded from this repository's own documentation. Eight
distinct queries × 10 rounds = 80 measurements, in-process, after a warm-up
pass so the figures exclude one-off graph compilation. Load average 2.5–3.6 at
the start and end of the run; **not a fully idle machine**, and the Docker VM
is part of what is measured.

| Stage | median | p95 | p99 | max |
|---|---|---|---|---|
| **total** | **123.5 ms** | **126.7 ms** | 127.3 ms | 127.4 ms |
| embed | 116.1 ms | 119.1 ms | 120.2 ms | 120.3 ms |
| dense (pgvector) | 1.7 ms | 5.4 ms | 6.4 ms | 6.5 ms |
| lexical (full-text) | 0.6 ms | 1.2 ms | 1.3 ms | 1.4 ms |

That sits inside Mem0's stated 150–200 ms comfort band and well under Zep's
published 576 ms p95 — though Zep's figure is a cloud call over a network at
concurrency 20 with a cross-encoder, so it is a reference point rather than a
like-for-like comparison.

**Embedding is 94% of it, and the retrieval CRED actually does is nearly free.**
Both database arms together are under 3 ms at the median against 1,247 claims.

### A correction to the embeddings spike

`go-embeddings-tokenizer.md` concludes *"Interactive recall is unaffected
(51 ms to embed a query)"*. That figure comes from its batch-8 latency table,
and it does not hold for a single query, which is what recall actually issues.

Measured on this machine, same model, same backend:

| Configuration | per text |
|---|---|
| seq 16, batch 8 | 26.3 ms |
| seq 16, **batch 1** | **116.1 ms** |
| seq 256, batch 8 | 503.5 ms |
| seq 256, batch 1 | 522.3 ms |

The batch-8 numbers reproduce the spike closely (it reports 25.4 ms at seq 16
and 537.0 ms at seq 256), so the model path is confirmed rather than
contradicted. What is new is roughly **100 ms of fixed per-execution overhead**
in the `simplego` backend, independent of batch size. At batch 8 it is amortized
away and invisible; at batch 1 it is the entire cost of a short query.

This is not a defect in this implementation as far as could be determined —
gomlx caches variable buffers between calls, so it is not weight re-upload — and
it is not fixed here. It is recorded because it is the single largest available
win on the recall path: eliminating it would take recall from ~123 ms to ~25 ms.
**Check:** profile a `simplego` execution of a 12-layer BERT at batch 1 and
attribute the fixed cost, before assuming it is irreducible.

### Measured seeding throughput

Seeding this repository — 40 files, 1,247 chunks — took **25 m 38 s**, or
roughly 1.23 s per chunk, on the machine above **while the test suite, the
linter and Docker were also running**. Treat it as an upper bound rather than a
clean measurement.

This is D-008's named condition arriving on schedule: the pure-Go forward pass
is 9–16x slower than ONNX Runtime, interactive recall is unaffected, and bulk
ingestion is where it hurts. The accepted answers are a build-tagged ONNX
Runtime variant behind the existing `Embedder` interface, and honest progress
reporting on first ingest. Neither is built yet.

## Testing

```sh
go test ./...                      # unit and conformance; no database, no API key
go test -tags=integration ./...    # adds seed, recall, and the write path against Postgres
```

The write-path integration tests run the real River queue and worker against a
real Postgres, but drive it with the **fake nominator** — so they need no API
key even though the production write path does. That is the property the whole
`nominate`/`curate` split is arranged to preserve.

Integration tests skip when Postgres is unreachable. In CI, `CRED_REQUIRE_DB=1`
turns that skip into a failure, and the suite fails if zero integration tests
ran — a skipped test and a passing test are the same green check, and a broken
database setup would otherwise produce a green build with no coverage, silently.

The temporal and ACL algebra is unit-tested as pure functions. If a test for
either ever needs a database, the boundary has already been violated.

## What is not built yet

Named explicitly rather than left silent.

- **`revise` and `confirm`.** Two of the four MCP tools are still absent. `recall`
  and `remember` ship; supersession-in-place and in-task affirmation do not.
- **The LLM-driven contradiction reconciler.** The curation worker deduplicates
  (exact-hash, D-010) and supersedes duplicates through the bi-temporal
  machinery, but it does not yet nominate *contradictions* for the reconciler to
  expire. That step needs the same LLM boundary the extractor uses and is the
  next piece of curation. Expire, prune, and rescore are also not built.
- **Semantic anchoring (L3).** Evidence carries a file path, a line range, and a
  content hash — tier 4 of the fingerprint ladder only. There is no tree-sitter
  symbol path and no normalized AST-node hash, so a pure-formatting commit
  currently *would* expire claims. That is the acceptance criterion the ladder
  exists to meet.
- **MaxSim second-stage ranking (D-010).** Retrieval is dense + lexical fused by
  RRF, with no reranking of any kind. MaxSim costs 242x storage per document and
  its storage strategy is an open design question, not a settled one.
- **Row-level security.** The Go-side intersection is the decision and it is
  implemented; RLS as a redundant backstop is not. Its coverage test would be
  asserting a redundancy, and the redundancy does not exist yet.
- **Streamable HTTP transport, and OAuth.** stdio only. The packaging spike
  makes Streamable HTTP primary for a team deployment; one local principal does
  not exercise it.
- **OpenTelemetry.** `internal/obs` holds the attribute constants and the `slog`
  setup. No spans are emitted and no exporter is wired.
- **Usage limits.** Contribution quota, cost attribution, recall budget per
  principal, and scope growth bounds. The write path exists now, so these are
  the next security control to add: shared memory with unbounded per-principal
  write access is a poisoning vector. The `nominate` boundary already exposes a
  `UsageSink` hook for the cost ledger; nothing is wired to it yet.
- **A durable cost ceiling.** `internal/nominate` records token usage per call
  through `UsageSink`, but there is no ledger and no hard USD ceiling enforced at
  job dispatch. Until there is, `cred curate` will call the model as often as the
  queue asks it to.
- **`squawk` migration linting, `govulncheck`, release automation, and the
  published container image.** CI runs lint, the two-assertion CGO guard, unit,
  integration, and race.

### Deviations from the written plan, and why

- **`internal/seed` is not in the layout in `.claude/rules/go.md` §2.** Cold-start
  ingestion needed somewhere to live and none of the listed packages fit it.
- **Integration tests use a plain `DATABASE_URL` rather than testcontainers.**
  `testing-strategy.md` specifies testcontainers with template-cloned databases.
  That would add `docker` and `moby` to `go.mod`, which is outside the accepted
  dependency set, and template cloning buys isolation this slice does not yet
  need. CI supplies a service container instead. Revisit when the suite grows
  enough that cross-test interference is real.
- **`pgvector-go` is not used.** The only thing needed was one text encoding for
  `halfvec`, which is thirty lines. Preferring thirty lines over a module is the
  standing rule.
- **The Anthropic adapter is hand-rolled, not the official SDK.** `internal/
  nominate` talks to the Messages API over `net/http` with `encoding/json` — zero
  new dependencies. The surface CRED needs is one POST with a JSON-schema output
  format and a `stop_reason` back; the worker-ops spike also warns the Stainless
  SDKs ship breaking changes in *minor* releases weekly, a standing tax not worth
  taking for this little API. A second provider would be a second `Model`, not a
  framework.
- **A pre-existing seed bug was fixed while adding supersession.** The
  read-only slice's `seed` path inserted a changed chunk's new evidence *before*
  superseding the old one, in two separate transactions — which violates the
  `evidence_live_chunk` unique index (two live rows for one repo/path/ordinal).
  It was reproduced failing at the previous commit on a clean database. The fix
  is `pg.ReplaceSeed`: supersede-then-insert in one transaction, so the new live
  row is unique and a crash leaves neither a duplicate nor a gap. Write-path
  evidence carries a NULL `chunk_ordinal` and is exempt from that index entirely,
  since many claims may point at spans of the same file.

## Dependencies

| Module | Why |
|---|---|
| `github.com/jackc/pgx/v5` | Accepted set. The driver, in `internal/store/pg` only |
| `github.com/pressly/goose/v3` | Accepted set. Migrations, embedded via `embed.FS` |
| `github.com/modelcontextprotocol/go-sdk` | Accepted set. The MCP tool surface |
| `github.com/gomlx/onnx-gomlx`, `github.com/gomlx/gomlx` | Accepted set. The pure-Go embedding stack |
| `github.com/stretchr/testify` | Accepted set, `require` only, enforced by `testifylint` |
| `golang.org/x/text` | **Outside the accepted set.** NFD normalization for the tokenizer's accent stripping, which the standard library does not provide. It is the tokenizer's sole dependency in the spike as well |
| `github.com/riverqueue/river` | Accepted set (D-013). Postgres-backed job queue for the write path — extraction runs off the turn on a River worker (D-017). Taken for correct leader handover on long-running jobs, transactional enqueue, and first-class OTel middleware, not for throughput |

The LLM client is **not** a dependency: the Anthropic adapter is hand-rolled over
`net/http` (see the deviations above). Reads and explicit `remember` add no
dependency at all.

## Documentation

- [Product requirements](docs/product/prd.md) — what to build, and the laws it
  must not violate
- [Research synthesis](docs/research/synthesis.md) — what discovery found, what
  it killed, and what survived
- [Decision log](docs/research/decision-log.md) — decisions with their reasoning
  and what each rules out
- [Documentation index](docs/README.md)

## Contributing

There is one maintainer and no contribution process yet, because a process
written before any contribution describes a guess. Open an issue describing what
you want to change before writing code, and the process will be written around
the first real case.

Security reports go to the address in [SECURITY.md](SECURITY.md).

## License

Copyright © 2026 canhta.

Licensed under the [Apache License 2.0](LICENSE).
