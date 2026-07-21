# Quickstart

Get CRED running locally in a few minutes. For what CRED *is* and why, see the
[README](README.md).

## Prerequisites

- **Docker** (to run PostgreSQL 17 + pgvector), or your own Postgres 17 with the
  `pgvector` extension.
- **Go 1.25+** to build the binary.
- **Node 20+** — only if you want to build the web console from source. The CLI,
  the MCP server, and the JSON API do not need it.

The first `recall`/`seed` downloads the embedding model once (~127 MiB, pinned by
SHA-256) into your user cache directory. No API key is needed for reading or for
explicit `remember` — embeddings run in-process, pure Go, `CGO_ENABLED=0`.

## 1. Start the database

```sh
docker compose up -d db          # PostgreSQL 17 + pgvector on 127.0.0.1:5433
```

The compose file needs no `.env`; the development credentials are the defaults.

## 2. Build and migrate

```sh
go build -o cred .               # the CLI, MCP server, and JSON API
./cred migrate                   # apply the schema and the job-queue tables
```

`./cred doctor` checks every part of the install and names the fix for anything
broken.

## 3. Seed, recall, remember

```sh
./cred seed .                    # index this repo's own docs (AGENTS.md, docs/**, …)
./cred recall "how is access control evaluated"
./cred remember "we fused the two retrieval arms with RRF at k=60"   # no API key
./cred log                       # every write is visible
```

`cred recall` prints the score decomposition — which retrieval arm contributed
what — on purpose. Silent write-accept-then-empty-read is the failure this
prevents.

## 4. The web console

The console (a claims browser and a recall inspector) ships inside the binary,
but it must be built first because it is a compiled SPA.

```sh
# One-time, needs Node: build the SPA and embed it into the binary.
task build                       # → ./cred with the console baked in
./cred web                       # console + API on http://localhost:8080

# Or, for development with hot reload (two processes):
task dev                         # open the Vite dev server: http://localhost:5173
```

In dev mode the console is served by Vite on `:5173` and proxies `/api` → the Go
server on `:8080` — so open **`:5173`**, not `:8080`. The embedded build
(`./cred web`) serves the console itself on `:8080`.

A plain `go build` (step 2) still runs `cred web` — it serves the JSON API under
`/api` and a stub page — but the full console needs `task build` or `task dev`.

Gate the console with a bearer token by setting `CRED_WEB_TOKEN`; change the
address with `CRED_WEB_ADDR` (default `:8080`).

## 5. Connect an agent over MCP

```sh
claude mcp add cred -- /absolute/path/to/cred serve
```

Two tools: `recall` (read-only) and `remember` (explicit contribution by
attestation). Recall output is fenced as untrusted data, never interpolated into
a prompt.

## 6. Automatic capture (optional, needs an LLM key)

Reading needs no key. The *automatic* write path — extracting claims from work as
it happens — does, and only it does. It runs off the turn on a background worker,
so it never blocks the agent.

```sh
export CRED_LLM_API_KEY=sk-...   # Anthropic by default; set CRED_LLM_BASE_URL for
                                 # OpenAI / DeepSeek / a self-hosted model
cred curate                      # drains the capture queue: nominate, then dedup
```

Wire the capture hook into your agent (a Claude Code example is in the
[README](README.md#automatic-capture)). Every automatic write is a *nomination*
that deterministic code validates — never raw storage — and is visible
(`cred log`) and reversible (`cred forget <id>`).

## Configuration

Every setting has a working default, so nothing above needs an `.env`. The full
table is in the [README](README.md#configuration); the ones you are most likely
to touch:

| Variable | Default | Meaning |
|---|---|---|
| `DATABASE_URL` | `postgres://cred:cred@127.0.0.1:5433/cred?sslmode=disable` | The datastore |
| `CRED_PRINCIPAL` | `local` | Identity recall is evaluated against |
| `CRED_WEB_ADDR` | `:8080` | Console listen address |
| `CRED_WEB_TOKEN` | — | Bearer token gating the console (unset = open) |
| `CRED_LLM_API_KEY` | — | Model key for `cred curate` (automatic write path only) |
| `CRED_LLM_BASE_URL` | — | OpenAI-compatible endpoint; unset = Anthropic |

## Common tasks

```sh
task api:build      # build every Go package (CGO off)
task api:test       # Go tests
task api:lint       # golangci-lint
task web:test       # SPA tests (Vitest)
task build          # SPA + single embedded binary
task dev            # API + Vite dev server together
```
