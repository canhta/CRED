# CRED

**C**laims · **R**ights · **E**vidence · **D**erivation

> A claim lives only while its evidence does.

[![CI](https://github.com/canhta/CRED/actions/workflows/ci.yml/badge.svg)](https://github.com/canhta/CRED/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8.svg)](go.mod)

Evidence-governed memory for AI agents — open-source, self-hostable, and shipped
as **one static binary plus Postgres**. Agents connect over MCP to retrieve what
your team already knows and to contribute what they learn, and a web console lets
you see all of it.

---

## Why CRED

Most memory systems decide what to trust from **usage signal**: a fact becomes
trusted because it is retrieved and upvoted. That lags reality — a fact that was
true and became false stays trusted until enough negative signal accumulates.

CRED decides from **evidence**. Every claim points to the source that produced
it. When that source changes, the claim is invalidated — no inference call, no
waiting for signal, and **the reason is a diff, not a score**.

This is not a novel idea in the abstract (GitHub Copilot Memory validates code
citations against the current branch). CRED's position is specific and, as far as
we know, unmatched as an open package: **reformat-immune, symbol-granular,
self-hostable, deterministic, and inspectable.**

## Features

- **Evidence-governed expiry** — no claim without a pointer to what produced it;
  when the evidence changes semantically, exactly the affected claims expire.
- **Multi-language code anchoring** — a claim about code is anchored to the
  *symbol*, not a line range. Reformatting expires nothing; a real edit expires
  precisely what changed. 15+ languages, pure Go, no CGO. ([details](#code-anchoring))
- **Inspectable hybrid recall** — dense + lexical retrieval fused by RRF, with
  every result showing *which arm contributed what*. No opaque scores.
- **A web console** — browse claims and their evidence, and a **recall inspector**
  that shows why each result ranked. ([details](#the-web-console))
- **Team access control** — permissions evaluated at recall, failing closed. A
  claim derived from several sources inherits the **intersection** of their
  rights, never the union.
- **Usage limits as a security control** — four per-principal ceilings ship on by
  default; poisoning a shared memory is a bounded, loud, recorded act.
- **Bring your own model** — reads and explicit `remember` need no API key at all;
  the automatic write path speaks Anthropic, OpenAI, DeepSeek, or a self-hosted
  OpenAI-compatible endpoint.

## Quickstart

```sh
docker compose up -d db          # PostgreSQL 17 + pgvector on 127.0.0.1:5433
go build -o cred .
./cred migrate                   # schema + job-queue tables
./cred seed .                    # index this repo's own docs
./cred recall "how is access control evaluated"
./cred web                       # console + API on http://localhost:8080
```

Full walkthrough — the web console, connecting an agent, and automatic capture —
in **[QUICKSTART.md](QUICKSTART.md)**. `./cred doctor` checks the install and
names the fix for anything broken.

## The web console

`cred web` serves a JSON API and an embedded React SPA from the single binary.
Two views ship today:

- **Claims** — every stored claim with its evidence (`path:lines` + symbol),
  scope, and live/expired status; filter by status and scope.
- **Recall inspector** — CRED's signature view. Type a query and see the ranked
  results *with the per-arm contributions that placed them* — dense vs lexical,
  each arm's rank and its RRF points — plus the retrieval accounting (candidates
  retrieved, how many survived access control, the dominant arm, timings). A short
  result is never a silent one.

The console is a compiled SPA, so build it once with `task build` (needs Node),
which embeds it into the binary and serves it on `:8080`; `task dev` runs it with
hot reload on the Vite dev server at `:5173` (proxying `/api` → `:8080`). Gate it
with `CRED_WEB_TOKEN`. Access control at the API is the engine's own — the store
returns rows and `internal/acl` decides — so the console never shows a principal
a claim it may not read.

## Connecting an agent (MCP)

```sh
claude mcp add cred -- /absolute/path/to/cred serve
```

Two tools: `recall` (read-only) and `remember` (explicit contribution by
attestation). Recall output is fenced as untrusted data, never interpolated into
a prompt. `remember` stores the statement the agent asserts, attributed to the
calling principal, and is reversible with `cred forget`.

## Automatic capture

Knowledge is written two ways.

**Explicit — `remember`.** A person (or agent) asserts a statement. Human
attestation is evidence, so the assertion is its own source. No model, no key.

**Automatic — nomination, off the turn.** A hook captures material and enqueues a
job; a background worker (`cred curate`) extracts *candidate* claims and
deterministic code decides what is stored. It never blocks the agent's turn, and
it is a nomination — not raw storage — so every automatic write is validated,
visible (`cred log`), and reversible (`cred forget`).

```
hook → cred capture → [job queue] → cred curate → nominate → validate → write → dedup
        (returns now)               (worker, off the turn, needs a model key)
```

A Claude Code hook example (opt out with `CRED_AUTO_CAPTURE=false`):

```jsonc
// .claude/settings.json
{
  "hooks": {
    "PostToolUse": [{
      "matcher": "Bash",
      "hooks": [{
        "type": "command",
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

```sh
export CRED_LLM_API_KEY=sk-...    # or CRED_LLM_BASE_URL for OpenAI/DeepSeek/self-hosted
cred curate                       # drains the queue, off the turn
```

## Code anchoring

A claim drawn from code is anchored to the *symbol it is about*. `internal/anchor`
finds the enclosing definition and stores a **relocatable symbol path** (tier 1,
e.g. `Service > handle` or `Executor.writeOne`) and a **whitespace-normalized hash
of that definition** (tier 2). `cred reanchor` recomputes both against the current
file:

- Tiers 1 and 2 agree → the claim **survives**, whatever happened to the bytes
  between (reindent, blank lines, reformatting expire nothing).
- Tier 2 changed under an unchanged tier 1 → a real edit to the anchored code →
  **that claim expires**.
- Tier 1 gone (renamed/deleted) or ambiguous → **expires** rather than guesses.

Language detection is a table-driven regex registry modeled on universal-ctags'
optlib parsers — adding a language is a row of patterns, not code — on Go's
standard `regexp` (RE2): pure Go, no CGO, no grammar blobs. Shipped: **Go,
TypeScript/JavaScript, Python, Rust, C, C++, Java, C#, Ruby, PHP, Swift, Kotlin,
Scala, CSS**. An unrecognized language, or a span that matches no pattern,
degrades to a raw hash (tier 4) that never expires a claim by accident — a wrong
anchor is worse than none.

```sh
cred capture --path src/service.ts --repo "$PWD" < span.txt   # kind inferred
cred curate                                                   # nominate off the turn
# ...edit src/service.ts...
cred reanchor "$PWD"   # reformatting survives; a real edit expires that one claim
```

## Commands

| Command | What it does |
|---|---|
| `cred migrate` | Apply database migrations (schema + queue tables) |
| `cred seed <path>` | Index a repo's docs (`AGENTS.md`, `README.md`, `docs/**`, …). Idempotent |
| `cred recall <query>` | Retrieve claims, showing what each retrieval arm contributed |
| `cred remember <text>` | Contribute a claim by attestation. Deterministic, no API key |
| `cred capture` | Enqueue captured material for automatic extraction (hook entry point) |
| `cred curate` | Run the background worker: nominate off the turn (needs a key), then dedup |
| `cred reanchor [path]` | Re-check anchors; expire claims whose source changed |
| `cred log` | Show recent writes — live, superseded, or forgotten |
| `cred forget <id>` | Reverse a write by expiring its claim |
| `cred usage` | Per-principal quota headroom and per-scope cost |
| `cred serve` | Run the MCP server over stdio (`recall` + `remember`) |
| `cred web` | Serve the web console: the JSON API and the embedded SPA |
| `cred doctor` | Check the installation; every failure names its fix |

## Configuration

Every variable has a working default, so the commands above need no `.env`. There
is deliberately no `.env.example` — a file you must copy first is a step, and
steps cost users.

| Variable | Default | Meaning |
|---|---|---|
| `DATABASE_URL` | `postgres://cred:cred@127.0.0.1:5433/cred?sslmode=disable` | The one datastore |
| `CRED_MODEL_DIR` | user cache dir | Directory holding `model.onnx` |
| `CRED_ALLOW_MODEL_DOWNLOAD` | `true` | Fetch the embedding model on first run |
| `CRED_PRINCIPAL` | `local` | Identity recall is evaluated against |
| `CRED_WEB_ADDR` | `:8080` | Console listen address |
| `CRED_WEB_TOKEN` | — | Bearer token gating the console (unset = open) |
| `CRED_AUTO_CAPTURE` | `true` | Automatic nomination on `capture`; `false` to opt out |
| `CRED_LLM_API_KEY` | — | Model key for `cred curate` (automatic write path only) |
| `CRED_LLM_MODEL` | `claude-opus-4-8` | Model id the nominator uses |
| `CRED_LLM_BASE_URL` | — | OpenAI-compatible endpoint (OpenAI/DeepSeek/self-hosted); unset = Anthropic |

Usage limits ship on with working ceilings — `CRED_CONTRIBUTION_QUOTA` (120),
`CRED_COST_MAX_CALLS` (500), `CRED_COST_MAX_TOKENS` (2,000,000),
`CRED_RECALL_RATE` (120), `CRED_SCOPE_CLAIM_CEILING` (5,000), each per window — a
non-positive override disables that one control. Windows
(`CRED_CONTRIBUTION_WINDOW`, `CRED_COST_WINDOW`, `CRED_RECALL_WINDOW`) take a Go
duration such as `1h`. See `cred usage`.

## Architecture

CRED is a Go-first repository: the module lives at the root and the SPA in `web/`.
`cred` is one binary — the CLI, the MCP server (`serve`), the web console (`web`),
and the background worker (`curate`) are subcommands, not separate services.

```
main.go                thin: signals, calls internal/cli
internal/
  cli/                 the subcommands
  claim/               Claim, Evidence, Principal, ACL, Interval
  temporal/            bi-temporal algebra          — PURE
  acl/                 ACL intersection algebra      — PURE
  anchor/              the code/text anchor ladder   — PURE
  limit/               usage-and-limits policy       — PURE
  recall/              retrieval orchestration, RRF fusion
  nominate/            the LLM boundary (holds no store)
  curate/              the deterministic write executor + workers
  api/                 the console's Gin JSON API
  store/pg/            the ONLY package that imports pgx
  embed/  wordpiece/   the pure-Go embedding stack + tokenizer
  mcpsrv/  obs/  seed/  config/
web/                   Vite + React 19 + Astryx console (tygo-typed API client)
```

Three rules are load-bearing and enforced by `depguard`, not by good intentions:

- **The pure packages take no database connection.** The temporal algebra, the
  ACL intersection, the anchor ladder, and the limit policy decide over values;
  the store only supplies them. A stray `pgx` import fails the build.
- **Access control is never a SQL predicate.** No function in `store/pg` takes a
  principal. The store returns rows; `internal/acl` computes
  `claim.acl ⊆ ⋂(evidence.acl)` in Go, failing closed.
- **The model nominates, code decides.** `internal/nominate` may not import the
  store, so an extractor with a database handle does not compile. Evidence is
  materialized from the trusted source, never from the model's output — a
  candidate whose quote is not a verbatim span is dropped, not stored.

See the [decision log](docs/research/decision-log.md) for why each choice was
made and what it rules out.

## Development

```sh
task api:build   # build every Go package (CGO off)
task api:test    # Go tests
task api:lint    # golangci-lint
task web:test    # SPA tests (Vitest)
task gen:types   # regenerate the TS API types from the Go structs (tygo)
task build       # SPA + single embedded binary
task dev         # API + Vite dev server together
```

Testing:

```sh
go test ./...                    # unit and conformance; no database, no API key
go test -tags=integration ./...  # adds seed, recall, and the write path vs Postgres
```

The write-path integration tests run the real queue and worker against a real
Postgres but drive it with a **fake nominator**, so they need no API key. The
temporal and ACL algebra is unit-tested as pure functions; if either ever needs a
database, the boundary has already been violated.

## Status and roadmap

The engine is built and verified: read path, automatic write path, text **and**
multi-language code anchoring, usage limits, multi-provider LLM support, and a
web console with a claims browser and a recall inspector. See
[benchmarks](docs/benchmarks.md) for reproduced measurements.

Not yet built, named rather than left silent:

- **`revise` and `confirm` MCP tools**, and the LLM-driven contradiction
  reconciler (dedup and supersession ship; nominating contradictions does not).
- **Console features** beyond Claims and Recall: usage/limits management,
  analytics, team management, SSO, projects, file upload.
- **MaxSim second-stage ranking** — retrieval is dense + lexical + RRF, no
  reranking; MaxSim's storage strategy is an open question.
- **Streamable HTTP / OAuth for MCP** (stdio only today) and **row-level security**
  as a redundant backstop to the Go-side decision.
- **An OpenTelemetry exporter** — usage is emitted as OTel-named `slog` records,
  but nothing ships them to a collector yet.
- **A build-tagged ONNX Runtime embedder** for faster bulk seeding.

## Documentation

- [Quickstart](QUICKSTART.md) — get running
- [Benchmarks](docs/benchmarks.md) — reproduced measurements and their conditions
- [Product requirements](docs/product/prd.md) — what to build and the laws it must not violate
- [Decision log](docs/research/decision-log.md) — decisions, their reasoning, and what each rules out
- [Documentation index](docs/README.md)

## Contributing

There is one maintainer and no formal contribution process yet — a process
written before any contribution is a guess. Open an issue describing what you want
to change before writing code, and the process will be written around the first
real case. Security reports go to the address in [SECURITY.md](SECURITY.md).

## License

Copyright © 2026 canhta. Licensed under the [Apache License 2.0](LICENSE).
