# Packaging, Distribution, and First Run

The first run is a strategic requirement, not a nicety. It is the moment a new
user decides whether the project works.

**Target: under three minutes, dominated by the image pull.**

---

## Decisions

| Area | Decision |
|---|---|
| Postgres image | `pgvector/pgvector:0.8.5-pg17` — **pin pg17, not pg18** |
| Services | One image, two services (`serve` and `curate`) |
| Base image | `gcr.io/distroless/static-debian13:nonroot` (~2 MiB) |
| Registry | **GHCR primary.** Docker Hub only via Sponsored OSS |
| Multi-arch | Cross-compile on amd64. **No QEMU** |
| Embedding model | Baked into its own image layer; fetched for the bare binary |
| Release | release-please → tag → GoReleaser |
| MCP transport | Streamable HTTP primary, stdio bridge, **SSE never** |

---

## The finding that gates the language decision

> **The pure-Go WordPiece tokenizer for `bge-small-en-v1.5` is unverified, and
> the entire Go recommendation rests on it.**

`hugot` v0.7.5 (June 2026) defaults to a pure-Go backend over `gomlx` v0.27.3,
which claims no C or C++ dependencies. If that path does not handle WordPiece at
acceptable throughput, local embeddings require CGO — and **CGO costs
cross-compilation, `FROM scratch`, and musl compatibility in a single move.** It
is not a partial commitment.

**Spike this before writing production code — roughly half a day.** Keep the
embedder behind an interface so an ONNX Runtime fallback is a build tag rather
than a rewrite.

Mitigating factor: the worker embeds in batches, where throughput matters far
more than latency, so the trade is probably acceptable even if slower.

---

## Two silent-data-loss traps

**Pin Postgres 17.** PG18 changed `PGDATA` to a version-specific path — mount
`/var/lib/postgresql` on 18+, but `/var/lib/postgresql/data` on 17 and earlier.
The official image documentation warns in capitals that the wrong path on ≤17
**will not persist data**. One wrong volume line, total loss, no error message.

**Health check must name the database.** Use `pg_isready -U cred -d cred`, not
bare `pg_isready`. During initialization the entrypoint runs a temporary
local-socket server, and a bare check passes against it — letting the
application race real startup.

---

## Why GHCR, not Docker Hub

Docker Hub allows **100 anonymous pulls per 6 hours, per IPv4 address or IPv6
/64 subnet**.

The per-subnet clause is disqualifying: one corporate NAT shares 100 pulls, and
a CI fleet exhausts them before lunch. For a project whose entire pitch is
`docker compose up`, that is an uncontrollable adoption tax.

GHCR has no such limit for public images, is free for public repositories, and
is where build attestations naturally land.

---

## Multi-arch without QEMU

With `CGO_ENABLED=0`, both architectures cross-compile on one amd64 runner and
the Dockerfile becomes COPY-only. No QEMU (10–40x slower and flaky), no build
matrix, no manifest merge.

This is the largest available CI simplification, and another point for Go.

---

## Compose shape

One image, two services. Server and worker share the data layer, migrations,
config, and embedding code — `recall` embeds the query text, so both need the
model regardless. Splitting buys nothing and costs two builds, two SBOMs, and a
version-skew failure mode.

Principles that matter more than the file itself:

- **`${VAR:-default}` everywhere** so the file works with no `.env` at all. A
  `.env.example` you must copy first is a step, and steps cost users.
- **No published port on the database** — 5432 collides with the Postgres half
  the audience already runs.
- **Loopback-bound application port.** Binding `0.0.0.0` should require an
  explicit flag plus a configured token.
- **No resource limits in the default file.** A limit tuned for one laptop
  OOM-kills someone else's worker and presents as a mystery restart loop. Ship
  limits as an opt-in overlay; document the real floor of roughly 2 GB.
- Prefix environment variables `CRED_`, except `DATABASE_URL`, which is a
  de-facto standard users expect.

### Model delivery

**Bake into the image, in its own layer.** `docker compose up` must work on an
air-gapped laptop, on hotel wifi, and in CI. A first-run model download makes a
third party a dependency at exactly the wrong moment.

Layer ordering solves the bloat objection: the model layer changes rarely, so a
patch release re-pushes only the binary layer.

**Never embed the model in the binary** — it lands in the data segment, is read
resident, and produces a 150 MB binary with matching memory use, re-downloaded
every patch. For the bare binary, fetch into the user cache directory verified
against a **pinned SHA-256**. An unpinned model download is a supply-chain hole.

Expected size: ~15 MB binary + ~2 MB base + ~130 MB model ≈ 150 MB uncompressed.
Publish a `:slim` variant without the model.

---

## MCP transport

Current spec revision is **2025-11-25**. Two standard transports: **stdio** and
**Streamable HTTP**. The old **HTTP+SSE transport is deprecated** — only a
backwards-compatibility section remains.

**Streamable HTTP is primary**, and the reason is architectural rather than
stylistic: stdio spawns a fresh server subprocess per client per project —
several processes contending over one database with no shared pool, and no path
to running on a team server. Streamable HTTP matches the deployment shape: one
server, many agents. Ship a thin stdio-to-HTTP bridge for stdio-only clients.

Security requirements: validate `Origin` and return 403 on mismatch
(DNS-rebinding defense), and bind localhost when running locally.

**Configuration footgun worth documenting:** in Claude Code, a JSON entry with a
`url` but **no `type`** is a configuration error — typeless entries are read as
stdio and silently skipped. `type` accepts `"streamable-http"` as an alias for
`"http"`, so snippets copied from spec-flavoured documentation work unmodified.

---

## First-run output

The startup log is product surface. It should report what was created, and end
by **printing the next command**:

```
cred-server-1  | database   connected (PostgreSQL 17.5)
cred-server-1  | extension  vector 0.8.5 ready
cred-server-1  | migrate    applying 12 migrations... done (412ms)
cred-server-1  | embedding  bge-small-en-v1.5 (384d, local, no API key)
cred-server-1  | seeding    scanning /workspace ... 84 commits, 12 docs -> 37 claims
cred-server-1  | mcp        Streamable HTTP on http://127.0.0.1:8080/mcp
cred-server-1  |
cred-server-1  |   Ready. Connect your agent:
cred-server-1  |     claude mcp add --transport http cred http://127.0.0.1:8080/mcp
```

**The gap between "it started" and "my agent can use it" is where projects lose
people.** Users do not return to the README.

The seeding line is not decoration — an empty database at the end of
`docker compose up` fails the acceptance criterion that a single developer gets
useful recall on first run.

---

## Health endpoints

Three, and conflating them is the common mistake:

| Endpoint | Meaning | Used by |
|---|---|---|
| `/healthz` | process alive | Docker healthcheck, liveness |
| `/readyz` | database reachable, schema current, model loaded | `depends_on`, readiness |
| `/metrics` | Prometheus | optional |

**Liveness must not check the database.** If it does, a brief database blip
restart-loops the application and turns a five-second outage into a real one.

## `cred doctor`

Every check names its fix and exits non-zero for CI. The failure that will occur
most often, and which must be handled well:

```
  ✘ extension       pgvector not installed, and user "cred" lacks
                    permission to install it.

                    pgvector is not a "trusted" extension, so
                    CREATE EXTENSION requires superuser. Ask a DBA to run:

                        CREATE EXTENSION IF NOT EXISTS vector;
```

**Do not auto-create the extension on managed Postgres.** Report the exact
command instead.

---

## Release automation

**release-please + GoReleaser.** release-please owns version calculation,
changelog, and the release pull request from conventional commits; GoReleaser
owns artifacts, triggered by the tag. The loop keeps a human gate on *when* to
release, which is what a solo maintainer wants.

Rejected: semantic-release (Node-shaped, plugin sprawl), changesets (solves
JavaScript monorepo publishing — the wrong problem), GoReleaser alone (builds
artifacts but decides no versions and writes no changelog).

### Pre-1.0 contract, stated explicitly

> While CRED is `0.x`, minor bumps may contain breaking changes to the MCP tool
> schema, config, or database schema. Patch bumps never do. Every breaking change
> ships with a migration note.

Reach 1.0 when the MCP tool surface stops moving. **That surface is the public
API — not the Go package.**

### Supply chain

Minimum credible set for a solo maintainer, roughly twenty lines of CI:
build provenance via keyless OIDC attestation, SBOM generation, `provenance:
mode=max`, SHA-pinned actions, least-privilege job permissions.

Note that `actions/attest-build-provenance` is now a wrapper — capabilities were
folded into `actions/attest`, and most tutorials are stale.

**Skip:** self-managed signing keys (keyless is strictly better), SLSA L3
reusable-workflow generators (real ceremony for marginal gain), and full
scorecard campaigns until there are contributors.

Give users the one-line verification command in the README. It is a strong trust
signal for two lines of documentation.

### EU Cyber Resilience Act

In force since 2024-12-10. **Reporting obligations begin 2026-09-11**; main
obligations 2027-12-11.

CRED is **exempt today** — the Act states unmonetised free and open-source
software is not commercial activity, and stewards face a light-touch regime with
no administrative fines. **Monetising later likely moves the project in scope**,
so adopt attestations now rather than retrofitting under a deadline.

---

## Language comparison, on packaging grounds alone

| | Go | Node/TS | Python |
|---|---|---|---|
| True single binary | **Yes** | Bun approximates; SEA no | **No** |
| Cross-compile from one host | **Yes** (CGO off) | Bun yes; SEA crippled | No |
| Local embeddings, no API key | **Yes, pure-Go now** | Native addon pain | Yes, but huge |
| Distroless or scratch base | **Yes** | No | No |

**Node SEA remains Stability 1.1 after three years.** Cross-platform builds
require disabling both code cache and snapshot, there is no Alpine support, and
**linux/arm64 Docker builds produce binaries with a broken hash table that crash
on `process.dlopen()`** — precisely the combination a containerized ARM build
needs, and the ONNX Runtime binding is unambiguously a native addon. If Node were
forced, use `bun build --compile`, never SEA.

**Python cannot deliver this.** PyApp is a bootstrapper that installs a Python
distribution on first run — a downloader, not a binary. PyInstaller `--onefile`
extracts the whole bundle to a temporary directory on *every* launch and cannot
cross-compile. Even dropping heavy ML dependencies leaves the ONNX Runtime native
libraries, numpy extensions, a Rust tokenizer extension, and the database driver
— every one a binary extension the freezer must relocate.
