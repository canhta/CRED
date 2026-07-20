# Spike — Language, Runtime, and MCP Server Architecture

- **Date:** 2026-07-20
- **Status:** Recommendation, pending build validation
- **Scope:** Language/runtime selection, MCP server implementation, dual-mode
  serving, repository layout, and schema evolution.
- **Method:** Official SDK and specification documentation fetched via Context7
  and direct fetch. Every load-bearing claim carries a URL. Unverified items are
  flagged explicitly in [Section 7](#7-what-could-not-be-verified).

---

## Recommendation

1. **Language:** **Go**. One statically-linked binary, an official
   Anthropic-maintained MCP SDK **past v1 and tracking the spec ahead of the
   spec** (v1.6.1 on pkg.go.dev, v1.7.0+ in main), best-in-class Postgres tooling
   (pgx/v5), and the lowest long-term maintenance burden for a solo maintainer.
2. **MCP server:** Build on the **official `modelcontextprotocol/go-sdk`**
   (Anthropic-maintained), exposing **stdio** and **Streamable HTTP**. The SDK
   supports **multiple spec revisions simultaneously** — v1.7.0+ already targets
   the forthcoming **2026-07-28** while retaining 2025-11-25, 2025-06-18,
   2025-03-26 and 2024-11-05. Never implement the legacy HTTP+SSE transport.
3. **Auth:** MCP server is an **OAuth 2.1 Resource Server only**. The SDK's
   `auth` package supplies token verification, `WWW-Authenticate`, and RFC 9728
   metadata. **Do not build an Authorization Server** — delegate to an external
   IdP. stdio mode uses environment credentials, per spec.
4. **Local vs shared:** **One binary, one server core, two transport adapters**
   selected by subcommand (`cred serve --stdio` / `cred serve --http`).
   Authentication is a `Principal`-resolver interface with a trusted-local
   implementation and an OAuth implementation. The domain core never sees a
   transport or a token.
5. **Layout:** `internal/core` (pure domain, zero infrastructure imports) at the
   centre; `internal/mcp`, `internal/storage`, `internal/worker` all depend
   inward only. Enforced in CI by **`depguard`** import rules, not by convention.
6. **Evolution:** **Additive-only claim schema** with a `schema_version` column
   and expand/contract migrations; **versioned tool names only on breaking
   change** (`recall` → `recall_v2`), with the old tool retained for two minor
   releases and marked deprecated in its `description`.

**Strongest counter-argument to Go — stated up front:** the AI-agent OSS
ecosystem is overwhelmingly Python and TypeScript. Choosing Go optimises for the
maintainer and pessimises the contributor funnel, and for a project whose stated
strategy is *adoption-first open source* ([D-001](../decision-log.md)), that is a
real cost. It is addressed, not dismissed, in [Section 1.4](#14-the-counter-argument).

---

## 1. Language and runtime

### 1.1 What this system actually is

Before comparing languages, it is worth being precise about the workload,
because it disqualifies some common reasoning.

CRED is **not** a machine-learning system. Per the PRD, the model *nominates*
and code *decides* (L2), and all inference happens in the curation worker, out
of band. The request path is:

- Parse an MCP tool call
- Run one hybrid SQL query (pgvector + BM25 + scope proximity)
- Assemble a deterministic context package under a token ceiling
- Return structured JSON

That is an **IO-bound JSON-and-SQL service** with a token-counting step. There is
no tensor math, no model serving, no numerical stack. The gravitational pull of
Python in AI infrastructure — that is where the ML libraries are — **does not
apply here**, because the ML libraries are not in the request path. Embeddings
are produced by an external API or an external embedding server; the database
does the vector math.

This matters because it removes the single strongest argument for Python.

### 1.2 The comparison

| Criterion | Go | Rust | TypeScript/Node | Python |
|---|---|---|---|---|
| **Official MCP SDK** | `go-sdk` **v1.6.1** stable since 2025-09-30, no-breaking-change promise; v1.7.0-pre.3 tracks 2026-07-28. Google Go team + Anthropic | `rmcp` **2.2.0** — but **community-maintained** (Apollo, Block, ZTE), **zero server-side OAuth** | `typescript-sdk` **1.29.0**; **v2.0.0-beta.4 GA 2026-07-28** — mid-rewrite | `python-sdk` **1.28.1**; **v2.0.0b2 GA 2026-07-27** — mid-rewrite. Strongest auth (RS *and* AS) |
| **Single self-contained binary** | **Native.** `CGO_ENABLED=0 go build` → one static file, cross-compiled from any host | **Native**, but slower builds and libc/musl care needed | Poor. Needs `pkg`/SEA/Bun; ships a runtime; native modules break it | **Poor.** PyInstaller/Nuitka are fragile; a Python "binary" is a bundled interpreter |
| **Postgres driver** | **pgx/v5** — best-in-class in any language | sqlx / tokio-postgres — very good | `pg` / `postgres.js` — adequate | psycopg3 / asyncpg — very good |
| **Concurrency for IO-bound work** | Goroutines; trivially correct | async/await; correct but ceremonious | Event loop; fine, single-threaded by default | asyncio; fine, but two ecosystems (sync/async) |
| **Solo-maintainer burden** | **Lowest.** Tiny stdlib-centric dep tree, no runtime, gofmt ends style debate, exceptional backward compatibility | Highest. Borrow checker tax on a CRUD-shaped domain; slow CI | Medium-high. Dependency churn, build config, transitive supply chain | Medium-high. Packaging and env management are the perennial tax |
| **Contributor pool (AI-agent OSS)** | Smaller | Smallest | **Largest** | **Largest** |
| **Deploy footprint** | ~20–30 MB scratch image | ~10–20 MB | ~150 MB+ | ~200 MB+ |

### 1.3 Why Go wins for *this* system

**1. The deployment requirement is a hard constraint, and it selects for Go or
Rust.** The PRD is unambiguous: "One container plus PostgreSQL, or a single
binary with `DATABASE_URL`", and acceptance criterion 8 requires
`docker compose up` to reach a working instance with no additional steps. The
[RAGFlow review](../evidence/ragflow.md) records exactly what the alternative
costs: 6+ containers, 16 GB RAM, x86-only images, `vm.max_map_count` kernel
tuning, and `pip install torch` at container runtime. That review's own verdict —
"Ops cost is disqualifying" — is the strongest available evidence for this
constraint, and it came from a Python codebase whose packaging problems are
structural rather than incidental.

A Go binary is `FROM scratch` plus one file, cross-compiled to
linux/amd64, linux/arm64, and darwin/arm64 from a single machine with no
toolchain per target. Neither Node nor Python can match this, and both would
reintroduce exactly the friction that D-003 says must not exist at rung 1.

**2. The Postgres story is the best in any language.** `pgx/v5` is not merely
adequate — it is the reference implementation of a modern Postgres client:
native binary protocol, real `COPY`, `LISTEN/NOTIFY`, connection pooling in
`pgxpool`, and a type system extensible enough to map `vector` directly.
Combined with **sqlc**, which compiles SQL into type-safe Go at build time, the
storage layer becomes checked against the real schema by the compiler.

Critically, `Queries.WithTx(tx)` gives an explicit, ordinary way to run domain
operations inside one transaction
([docs.sqlc.dev/howto/transactions](https://docs.sqlc.dev/en/latest/howto/transactions.html)) —
which is what **L3** demands: "every claim resting on it is invalidated in the
same transaction."

**3. Postgres-only background work is a solved problem in Go.** L7 forbids a
second datastore, which rules out Celery, BullMQ, Sidekiq, and every Redis-backed
queue. **River** is a Postgres-backed job queue for Go with transactional
enqueue:

```go
// The claim write and the curation job are committed atomically.
tx, _ := pool.Begin(ctx)
defer tx.Rollback(ctx)

qtx := queries.WithTx(tx)
claim, _ := qtx.InsertClaim(ctx, params)
_, _ = riverClient.InsertTx(ctx, tx, CurateClaimArgs{ClaimID: claim.ID}, nil)

_ = tx.Commit(ctx)
```

River also provides leader-elected periodic jobs and unique-job constraints
(`UniqueOpts{ByArgs: true, ByPeriod: 24*time.Hour}`), which map directly onto the
five curation stages
([riverqueue.com/docs/transactional-enqueueing](https://riverqueue.com/docs/transactional-enqueueing)).

This is worth weighing against the two concrete queue bugs the RAGFlow review
found: an unconditional ack outside the `try/except` that silently drops failed
work, and a retry counter incremented on every claim including successful ones.
Those are the failure modes of a hand-rolled queue. Not hand-rolling it is the
mitigation.

**4. Maintenance burden over years, by one person.** Go's compatibility promise
means a `go.mod` from 2019 still builds. The dependency tree for this system is
roughly: MCP SDK, pgx, sqlc-generated code, River, a CLI library, and a logger.
That is a supply-chain surface a single maintainer can actually audit.

**5. Rust loses on the SDK, not on the language.** The initial assumption that
`rmcp` is pre-1.0 was **wrong** — it is at **2.2.0** with 16.4M downloads, only
42 open issues, and a conformance suite in CI. The real disqualifiers are
different and more serious:

- **`rmcp` has no server-side OAuth at all** — no bearer verification, no RFC
  9728 metadata serving. Every OAuth fix in its release notes is *client*-side,
  and its own examples hand-roll axum auth middleware. CRED would be writing an
  OAuth resource server by hand, which Section 2.4 argues against on security
  grounds.
- **It is not Anthropic-maintained.** Top contributors are from Apollo, Block,
  and ZTE; Anthropic has ~4 commits. It is a partner/community effort.
- **2026-07-28 conformance is incomplete** (open issues track 18/25 passing), and
  there is no declared MSRV.

Secondarily, the domain here is CRUD-with-scoring: the borrow checker collects
tax and returns little, and compile times slow the edit-test loop a solo
maintainer lives in. Go delivers ~95% of Rust's operational benefit at a fraction
of the cognitive and CI cost — and gets server OAuth in the box.

**6. TypeScript and Python are both mid-rewrite this month.** TS is at 1.29.0
with **v2.0.0-beta.4 going GA on 2026-07-28**; Python is at 1.28.1 with
**v2.0.0b2 targeting 2026-07-27**. Starting CRED on either right now means
picking between a line about to become legacy and a beta, during the exact weeks
the protocol itself is changing. The Go SDK, by contrast, shipped **v1.0.0 on
2025-09-30 with an explicit no-breaking-changes promise** and a monthly cadence.
For a solo maintainer, that stability difference is worth more than the
ecosystem-size advantage.

In fairness to Python: it has the **strongest auth story of the four**,
implementing both Resource Server *and* Authorization Server roles (`TokenVerifier`,
PRM routes, DCR, revocation). Since Section 2.3 recommends *not* building an AS,
most of that advantage is unused by CRED.

### 1.4 The counter-argument

**The contributor pool for AI-agent tooling is Python and TypeScript, and Go
narrows the funnel.** D-001 commits CRED to adoption-first open source, where
contributors are a primary asset. A Python or TypeScript CRED would be more
forkable and more hackable by the exact people who write MCP servers today.

Three things reduce but do not eliminate this:

1. **Adoption ≠ contribution for infrastructure.** Users of CRED interact over
   MCP and never read the source. The install experience — one binary, one
   `docker compose up` — is what drives adoption, and Go optimises it. Go is
   also the lingua franca of self-hosted developer infrastructure that engineers
   actually deploy.
2. **The contribution surface can be language-neutral.** Curation policies,
   claim-kind definitions, prompts, and evaluation cases should live in
   **declarative files (SQL, YAML, prompt templates)**, not Go code. Most
   community contributions to a memory system are policy and evaluation, not
   protocol plumbing.
3. **Solo maintenance dominates early.** Before there is a contributor community
   there must be a working, operable product. Optimising for hypothetical
   contributors at the cost of the maintainer's own velocity is the wrong trade
   at n=1.

**If the counter-argument is judged decisive, the fallback is TypeScript, not
Python** — the TS SDK is the protocol's reference implementation and gets
features first, and Node's packaging story, while poor, is less poor than
Python's. The cost is accepting a runtime in the deployment artifact.

---

## 2. MCP server implementation

### 2.1 Current specification state

**The current revision is `2025-11-25`.**
[modelcontextprotocol.io/specification/versioning](https://modelcontextprotocol.io/specification/versioning)
states: *"The current protocol version is 2025-11-25."* Versions are `YYYY-MM-DD`
strings marking the last date backwards-incompatible changes were made.

Revision history: `2024-11-05` → `2025-03-26` → `2025-06-18` → **`2025-11-25`
(current)** → `draft`.

> ### ⚠️ A breaking revision is imminent — and it is a tagged Release Candidate
>
> **`2026-07-28` exists as a tagged Release Candidate** (`2026-07-28-RC`, tagged
> 2026-05-29) — **eight days after this spike**, though not yet Final. Confirmed
> from three directions: the repository releases API, `schema/draft/schema.ts`
> (`LATEST_PROTOCOL_VERSION = "2026-07-28"`), and the Go SDK's pre-release line.
> The [draft changelog](https://modelcontextprotocol.io/specification/draft/changelog)
> describes sweeping breaking changes, including:
>
> - **`initialize` handshake replaced by `server/discover`**; sessions and
>   `Mcp-Session-Id` removed
> - **The `initialize` handshake removed** — the protocol becomes stateless,
>   with version and capabilities moving into per-request `_meta`
> - **SSE resumability and `Last-Event-ID` removed**; HTTP GET streams replaced
>   by `subscriptions/listen`
> - New `server/discover` RPC; new required `resultType` on all results
> - Roots, Sampling, Logging, and Dynamic Client Registration all deprecated
> - `inputSchema`/`outputSchema` loosened to arbitrary JSON Schema 2020-12
>
> **This is the single most important architectural finding in this spike.** The
> MCP wire protocol is not yet stable and will churn under CRED during v1
> development.
>
> **The mitigation is already available, and it is a direct argument for Go.**
> The Go SDK's README confirms that **v1.7.0+ targets `2026-07-28` as its primary
> specification while maintaining backward compatibility with 2025-11-25,
> 2025-06-18, 2025-03-26, and 2024-11-05**, and commits to supporting deprecated
> features "for compatibility during the deprecation window (at least twelve
> months)"
> ([README](https://github.com/modelcontextprotocol/go-sdk/blob/main/README.md)).
> The SDK absorbs multi-revision compatibility so CRED does not have to — which
> is precisely the property to select an SDK on when the protocol is moving.
>
> **Consequences, which drive the rest of this document:**
> 1. **Never hand-roll the transport.** Use the official SDK so protocol churn
>    arrives as a dependency bump rather than a rewrite.
> 2. **Keep the MCP layer thin and free of domain logic** — it is the layer that
>    will be rewritten.
> 3. **Do not persist anything keyed on MCP session identity**, since sessions
>    are being removed. Persist against the *principal*, never the session.

### 2.2 Transports

Two standard transports exist: **stdio** and **Streamable HTTP**. Clients
*SHOULD* support stdio whenever possible
([transports](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports)).

**stdio.** Newline-delimited JSON-RPC on stdin/stdout. Messages MUST NOT contain
embedded newlines, and the server MUST NOT write non-MCP data to stdout. As of
2025-11-25, stderr carries *all* logging and clients SHOULD NOT treat stderr
output as indicating an error.

> **Practical trap:** any stray `fmt.Println` or logger defaulting to stdout
> corrupts the protocol stream. In CRED, the logger must be pinned to stderr at
> construction, and this deserves a test.

**Streamable HTTP.** A single MCP endpoint handling POST and GET.

- Every client message is a new POST; `Accept` MUST list both
  `application/json` and `text/event-stream`.
- Responses/notifications → `202 Accepted`, no body. Requests → either
  `text/event-stream` or `application/json`; **clients MUST support both**.
- GET opens a server→client SSE stream, or the server returns `405`.
- `Mcp-Session-Id` MAY be assigned on `InitializeResult`; visible ASCII
  0x21–0x7E; echoed by the client thereafter. Terminated session → MUST `404`,
  on which the client MUST start a new session.
- Resumability: SSE `id` fields MUST be globally unique within a session; the
  client resumes via GET + `Last-Event-ID`.
- `MCP-Protocol-Version` header MUST be sent on all post-initialization HTTP
  requests; absent → server SHOULD assume `2025-03-26`; unsupported → MUST `400`.

**HTTP+SSE (the 2024-11-05 transport) is deprecated**, superseded by Streamable
HTTP in 2025-03-26, with removal targeted three months after SEP-2596 reaches
Final. **CRED should not implement it.** The cost of omitting it is limited to
clients that have not been updated since early 2025.

**Security (MUST/SHOULD-level):** servers MUST validate the `Origin` header as
DNS-rebinding defence and MUST return `403` on invalid origin; local servers
SHOULD bind to `127.0.0.1`, not `0.0.0.0`.

### 2.3 Authorization

Authorization is **OPTIONAL** overall; HTTP transports SHOULD conform
([authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)).

> **Directly answering the stdio question:** the spec states that **stdio
> implementations SHOULD NOT follow the authorization spec, and should instead
> "retrieve credentials from the environment."** This is what makes the dual-mode
> design in Section 3 clean rather than a compromise — the spec itself prescribes
> two different auth models for the two transports.

For HTTP, the MCP server is an **OAuth 2.1 Resource Server**. MUST-level
requirements relevant to CRED:

- MCP servers **MUST** implement **RFC 9728** Protected Resource Metadata,
  including `authorization_servers` with at least one entry.
- Servers **MUST** implement either a `WWW-Authenticate` header carrying
  `resource_metadata` on 401, **or** the well-known URI. (2025-11-25 relaxed
  `WWW-Authenticate` to optional for servers via SEP-985; clients must support
  both.)
- Servers **MUST** validate that tokens were issued for them as the intended
  audience, and **"MUST NOT pass through the token it received from the MCP
  client."** This is the confused-deputy defence and is the single most
  important security rule to get right.
- Clients MUST implement PKCE (`S256`) and **RFC 8707** resource indicators.

**Client registration changed materially in 2025-11-25.** Priority order is now:
pre-registration → **Client ID Metadata Documents** (new, SEP-991) → **Dynamic
Client Registration (RFC 7591), demoted to MAY** and retained only for backwards
compatibility. DCR is formally deprecated in the draft. **CRED should not build
DCR.**

### 2.4 What the SDK gives you vs. what you must build

This is the concrete answer, verified against the Go SDK API
([pkg.go.dev/github.com/modelcontextprotocol/go-sdk](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp)).

**The SDK gives you:**

| Concern | API |
|---|---|
| stdio transport | `mcp.StdioTransport` |
| Streamable HTTP transport | `mcp.NewStreamableHTTPHandler(getServer, opts)` |
| Schema inference from Go types | `mcp.AddTool[In, Out]` via `google/jsonschema-go` |
| Input validation against the schema | automatic in `ToolHandlerFor` |
| `structuredContent` population | automatic from the `Out` value |
| Backward-compat JSON text block | automatic when `Content` is unset |
| Tool-error vs protocol-error split | returned `error` → `IsError` result |
| Bearer token verification | `auth.RequireBearerToken(verifier, opts)` |
| `WWW-Authenticate` on 401 | automatic in the above |
| RFC 9728 metadata endpoint | `auth.ProtectedResourceMetadataHandler(md)` |
| Progress, cancellation, pagination plumbing | protocol-level, handled |

The typed-tool signature is the load-bearing ergonomic win:

```go
type ToolHandlerFor[In, Out any] func(
    ctx context.Context, req *mcp.CallToolRequest, input In,
) (result *mcp.CallToolResult, output Out, err error)
```

Per the SDK docs, the input is *"automatically unmarshaled from
`req.Params.Arguments` and validated against its input schema"*, the `Out` value
*"is used to populate `result.StructuredOutput`"*, and *"an error result is
treated as a tool error, rather than a protocol error, and is therefore packed
into `CallToolResult.Content`, with `IsError` set."*

So a CRED tool is close to pure domain code:

```go
type RecallIn struct {
    Task    string   `json:"task" jsonschema:"the task the agent is about to perform"`
    Scope   []string `json:"scope,omitempty" jsonschema:"repository or path scopes to prefer"`
    Budget  int      `json:"token_budget,omitempty" jsonschema:"hard ceiling on returned tokens"`
}

type RecallOut struct {
    Claims  []ClaimView `json:"claims"`
    Dropped []DropNote  `json:"dropped"`   // in-band budget reporting, per PRD §6
    Hidden  int         `json:"hidden_by_permission"`
}

mcp.AddTool(server, &mcp.Tool{
    Name:  "recall",
    Title: "Recall organizational knowledge",
    Annotations: &mcp.ToolAnnotations{
        ReadOnlyHint: true,   // default is false — must be set explicitly
        OpenWorldHint: ptr(false),
    },
}, h.Recall)
```

**You must build:**

1. **An Authorization Server — or rather, you must *not*.** The SDK covers the
   resource-server side only. CRED should delegate to an external IdP and
   require only a `TokenVerifier`. Writing an OAuth AS as a solo maintainer is a
   security liability with no product value.
2. **`TokenVerifier` implementation** — JWKS fetch/cache, signature validation,
   issuer and **audience** checks (the confused-deputy defence), and mapping
   token claims to a CRED `Principal`.

   > **This is not a Go gap — it is universal, and it is the sharpest security
   > finding in this spike.** A survey of all four official SDKs found **none of
   > them demonstrably enforces RFC 8707 audience matching server-side**; all
   > delegate it to the token verifier you supply. Since the spec makes audience
   > validation a **MUST**, an SDK-default MCP server is non-compliant and
   > vulnerable to token passthrough until you write this check yourself. It
   > deserves a dedicated unit test asserting that a token minted for a
   > *different* audience is rejected.
3. **Principal → ACL-set resolution** with TTL and fail-closed semantics (L5).
   Entirely CRED's own logic.
4. **Token budgeting and context assembly** — the PRD's hard ceiling with
   in-band drop reporting. Note that the **spec says nothing about tool result
   size limits**; this is a genuine gap and entirely CRED's responsibility.
5. **Origin validation and localhost binding policy** for HTTP mode.

### 2.5 Tool definition and error conventions

`inputSchema` MUST be a valid JSON Schema **object** (never null); a no-parameter
tool uses `{"type":"object","additionalProperties":false}`. The default dialect
is **JSON Schema 2020-12** when `$schema` is absent, and implementations MUST
support at least 2020-12 (SEP-1613).

**Tool names** (new in 2025-11-25, SEP-986, all SHOULD-level): 1–128 characters,
case-sensitive, `A-Za-z0-9_-.` only, unique within a server. CRED's
`recall`/`remember`/`revise`/`confirm` comply, and so does the `recall_v2` scheme
in Section 5.

**Annotations are hints, not guarantees**, and clients MUST treat them as
untrusted from untrusted servers. Note the defaults, which are easy to get
wrong: `readOnlyHint` defaults to **false**, `destructiveHint` defaults to
**true**, `idempotentHint` defaults to **false**, `openWorldHint` defaults to
**true**. For CRED:

| Tool | readOnly | destructive | idempotent |
|---|---|---|---|
| `recall` | `true` | — | `true` |
| `remember` | `false` | `false` | `false` |
| `revise` | `false` | `false` (supersedes, never deletes — L6) | `false` |
| `confirm` | `false` | `false` | `true` |

**Error handling has two tiers**, and the distinction is a correctness issue
rather than a style preference:

- **Protocol errors** (JSON-RPC `error`): unknown tool, malformed request.
- **Tool execution errors** (`isError: true` in a normal result): business logic
  failures — **including input validation errors**.

2025-11-25 (SEP-1303) explicitly moved input validation errors into the
*execution* tier because *"Tool Execution Errors contain actionable feedback that
language models can use to self-correct."* Clients SHOULD pass execution errors
to the model but only MAY pass protocol errors.

**This matters for L1 directly.** When `remember` is called without evidence, the
correct behaviour is an execution error explaining that a claim requires
evidence — so the agent can retry with evidence attached. Returning a protocol
error would hide the reason from the model and make L1 feel like a malfunction
rather than a rule. In Go this is simply returning an `error` from the handler.

### 2.6 Pagination, progress, cancellation

**Pagination** is opaque-cursor-based and supported on exactly four operations:
`resources/list`, `resources/templates/list`, `prompts/list`, `tools/list`.
Clients MUST treat cursors as opaque and MUST NOT persist them across sessions.
Invalid cursors SHOULD return `-32602`.

> **`tools/call` is not paginated.** CRED's `recall` therefore cannot use MCP
> pagination to manage result size — which is correct anyway, since the PRD
> requires a *deterministically assembled package* under a token ceiling, not a
> pageable list. Budget management is CRED's own concern, reported in-band.

**Progress:** `progressToken` in request `_meta`, with `notifications/progress`
carrying `progress`, optional `total`, optional `message`. `progress` MUST
increase on each notification. Relevant to CRED only for cold-start seeding,
which is long-running; `recall` should never need it.

**Cancellation:** `notifications/cancelled` with `requestId`. Receivers SHOULD
stop work, free resources, and **send no response**. Both sides MUST handle the
inherent race gracefully. In Go this maps onto `context.Context` cancellation
propagating into `pgx` query cancellation — essentially free, provided **every
storage method takes a `ctx` as its first parameter**. Make that a lint rule.

---

## 3. Serving local and shared modes from one codebase

### 3.1 The shape of the problem

The two modes differ in exactly three respects:

| | Local (individual, rung 1) | Shared (team, rung 2) |
|---|---|---|
| Transport | stdio | Streamable HTTP |
| Authentication | none — OS process boundary is the trust boundary | OAuth 2.1 bearer token |
| Principal | a single implicit local user | resolved per request from token claims |

Everything else — tools, domain logic, storage, curation — is identical. The
spec itself endorses this split, since stdio "SHOULD NOT" use the auth spec and
should read credentials from the environment.

### 3.2 Recommendation: one binary, two transport adapters, injected principal

**Not** separate entry points, and **not** a boolean flag threaded through the
code. Use subcommands that construct the same core with different adapters.

```
cred serve --stdio                  # local; default
cred serve --http --addr :8080      # shared
cred worker                         # curation worker
cred seed --repo .                  # cold-start
```

The key insight is that `NewStreamableHTTPHandler` takes a **`getServer`
closure**, which is the natural injection point for the authenticated principal:

```go
func NewStreamableHTTPHandler(
    getServer func(*http.Request) *mcp.Server,
    opts *mcp.StreamableHTTPOptions,
) *StreamableHTTPHandler
```

So the whole dual-mode design reduces to this:

```go
// internal/mcp/server.go — transport-agnostic. Built once, knows nothing of
// tokens, HTTP, or stdio.
func NewServer(app *core.App, p core.Principal) *mcp.Server {
    s := mcp.NewServer(&mcp.Implementation{Name: "cred", Version: build.Version}, nil)
    h := &handlers{app: app, principal: p}
    mcp.AddTool(s, recallTool, h.Recall)
    mcp.AddTool(s, rememberTool, h.Remember)
    mcp.AddTool(s, reviseTool, h.Revise)
    mcp.AddTool(s, confirmTool, h.Confirm)
    return s
}
```

```go
// cmd/cred/serve_stdio.go
func runStdio(ctx context.Context, app *core.App) error {
    // Spec: stdio retrieves credentials from the environment.
    p := core.LocalPrincipal(os.Getenv("CRED_USER"))
    return NewServer(app, p).Run(ctx, &mcp.StdioTransport{})
}
```

```go
// cmd/cred/serve_http.go
func runHTTP(ctx context.Context, app *core.App, cfg HTTPConfig) error {
    handler := mcp.NewStreamableHTTPHandler(
        func(r *http.Request) *mcp.Server {
            // Populated by RequireBearerToken; never nil past the middleware.
            info := auth.TokenInfoFromContext(r.Context())
            return NewServer(app, core.PrincipalFromToken(info))
        },
        nil,
    )

    mux := http.NewServeMux()
    mux.Handle("/.well-known/oauth-protected-resource",
        auth.ProtectedResourceMetadataHandler(cfg.ResourceMetadata))
    mux.Handle("/mcp", auth.RequireBearerToken(cfg.Verifier,
        &auth.RequireBearerTokenOptions{
            ResourceMetadataURL: cfg.ResourceMetadataURL,
            Scopes:              []string{"cred.recall", "cred.write"},
        })(requireValidOrigin(cfg.AllowedOrigins, handler)))

    return http.ListenAndServe(cfg.Addr, mux)
}
```

### 3.3 Why this is the right shape

1. **The domain core never learns which mode it is in.** `core.App` takes a
   `Principal` and nothing else. There is no `if shared { ... }` anywhere below
   `cmd/`, which is the branch that metastasises.
2. **L5 is enforced in exactly one place.** Because both modes produce a
   `Principal` whose ACL set is resolved at recall time with a TTL, fail-closed
   permission checking is written once and tested once. The local principal is
   not a bypass — it is a principal that happens to own everything, so the code
   path is identical.
3. **The rungs of D-003's ladder are genuinely free to climb.** Moving from
   individual to team is `--stdio` → `--http` plus an IdP, against the same
   database and the same schema. No migration, no re-modelling, no "team
   edition."
4. **It survives the 2026-07-28 session removal.** Nothing is keyed on
   `Mcp-Session-Id`; identity is carried by the `Principal`, derived per request.

### 3.4 The local-mode security caveat

Local mode has no authentication, which is correct for stdio — the OS process
boundary is the trust boundary and the client spawns the server. **The danger is
`--http` with authentication disabled**, which teams will inevitably want for a
quick trial on a private network.

Recommendation: allow it, but make it loud and safe by construction —
`--http --no-auth` MUST refuse to bind to anything other than `127.0.0.1`, and
MUST log a warning on every startup. This follows the spec's own guidance that
local servers SHOULD bind to `127.0.0.1` rather than `0.0.0.0`. Silent
unauthenticated exposure of an organisation's memory is the worst failure this
system can have.

---

## 4. Project layout and maintainability

### 4.1 The failure being designed against

From the [RAGFlow review](../evidence/ragflow.md), stated precisely:

> `rag/` and `api/` are circularly dependent. 32 of 181 files in `rag/` import
> `api.*`; 28 files in `api/` import `rag.*`. There is no layered "RAG core"
> behind an interface; there is one application.

And the subtler failure in the same codebase — an abstraction that exists but is
not enforced:

> `DocStoreConnection` is a proper ABC, yet `search.py` still branches on
> `DOC_ENGINE_INFINITY`/`DOC_ENGINE_OCEANBASE` and calls `get_scores()`, which
> isn't on the interface. **If CRED abstracts its store, enforce the boundary in
> tests or don't bother.**

Both failures share one root cause: **the architecture was documented but not
mechanically enforced.** Go's package system plus a CI lint gate closes this,
because in Go an import cycle is a *compile error*, not a code smell.

### 4.2 Proposed structure

```
cred/
├── cmd/
│   └── cred/                    # the only main package; single binary
│       ├── main.go               # cobra root, config loading
│       ├── serve.go              # serve --stdio | --http
│       ├── worker.go             # curation worker
│       └── seed.go               # cold-start seeding
│
├── internal/
│   ├── core/                     # ── THE DOMAIN. Imports nothing but stdlib. ──
│   │   ├── claim.go              # Claim, ClaimKind, Scope, Confidence
│   │   ├── evidence.go           # Evidence, Locator (geometric | structural)
│   │   ├── principal.go          # Principal, ACL set, TTL, fail-closed check
│   │   ├── assemble.go           # context assembly: PURE function, token ceiling
│   │   ├── score.go              # additive explainable scorer
│   │   ├── ports.go              # interfaces the core REQUIRES (see 4.3)
│   │   └── app.go                # use cases: Recall, Remember, Revise, Confirm
│   │
│   ├── storage/                  # implements core ports over Postgres
│   │   ├── sqlc/                 # GENERATED — never edited by hand
│   │   ├── migrations/           # embedded via embed.FS
│   │   ├── claims.go             # ClaimRepo: core.ClaimStore
│   │   └── tx.go                 # transaction boundary helper
│   │
│   ├── mcp/                      # the MCP surface. THIN. Rewritten on spec churn.
│   │   ├── server.go             # tool registration
│   │   ├── tools_v1.go           # In/Out DTOs + schemas — versioned surface
│   │   └── convert.go            # DTO ↔ core translation
│   │
│   ├── worker/                   # River jobs
│   │   ├── dedupe.go             # MinHash/LSH, zero inference
│   │   ├── reconcile.go          # model nominates; core decides (L2)
│   │   ├── expire.go             # evidence hash re-check (L3)
│   │   ├── prune.go
│   │   └── rescore.go
│   │
│   ├── nominator/                # the ONLY package permitted to call an LLM
│   │   └── llm.go
│   │
│   └── config/
│
├── pkg/                          # (empty until something is genuinely reusable)
├── testdata/
├── .golangci.yml
├── compose.yaml
└── Dockerfile
```

**Notes on specific choices:**

- **`internal/` for everything.** Go's `internal/` is compiler-enforced against
  external import. This prevents accidental public API commitments, which for a
  solo maintainer is a real hazard: every exported symbol outside `internal/` is
  a support obligation. Promote to `pkg/` only on demand.
- **One `main` package.** The single-binary requirement makes multiple entry
  points a liability. Subcommands share config parsing and DB setup.
- **`internal/nominator` isolated.** L2 says the model never mutates state.
  Confining LLM calls to one package makes that mechanically checkable: no other
  package may import an LLM client (see 4.4).
- **`internal/mcp/tools_v1.go` versioned by filename.** The MCP surface will
  churn; naming it `v1` makes adding `v2` an additive act (see Section 5).

### 4.3 Dependency injection

**Recommendation: plain constructor injection with interfaces defined by the
consumer. No DI framework** — not wire, not fx, not dig.

Go's implicit interface satisfaction means the core declares what it needs
without importing any implementation:

```go
// internal/core/ports.go — interfaces defined where they are CONSUMED.
package core

type ClaimStore interface {
    Get(ctx context.Context, id ClaimID) (*Claim, error)
    Search(ctx context.Context, q Query, acl ACLSet) ([]ScoredClaim, error)
    Insert(ctx context.Context, c *Claim, ev []Evidence) (ClaimID, error)
    Supersede(ctx context.Context, old ClaimID, new *Claim) error
}

// InvalidateByEvidence must run in the SAME transaction as the write (L3).
type TxRunner interface {
    InTx(ctx context.Context, fn func(ClaimStore) error) error
}
```

Wiring is explicit in `main`:

```go
pool, _ := pgxpool.New(ctx, cfg.DatabaseURL)
store := storage.NewClaimRepo(pool)
app := core.NewApp(store, storage.NewTxRunner(pool), clock.System{})
```

Three reasons this beats a framework here: compile-time-checked wiring with
readable stack traces; trivially fake-able tests (`core` tests use an in-memory
`ClaimStore` and never touch Postgres); and no framework to maintain,
understand, or upgrade — which is the whole point at n=1.

**Interfaces are defined in `core`, not in `storage`.** This is what inverts the
dependency and makes the RAGFlow cycle structurally impossible: `storage` imports
`core`, `core` imports nothing.

### 4.4 What to enforce in CI

Convention is what decayed in RAGFlow. These are mechanical gates.

**1. Import boundaries via `depguard`** — the primary defence. `depguard`
supports per-file-glob rules with `allow`/`deny` lists
([golangci-lint.run/docs/linters/configuration](https://golangci-lint.run/docs/linters/configuration)):

```yaml
# .golangci.yml
linters:
  enable: [depguard, errcheck, govet, staticcheck, gosec, containedctx]
  settings:
    depguard:
      rules:
        # The domain core is pure: stdlib only. This is the load-bearing rule.
        core-is-pure:
          list-mode: strict
          files: ["**/internal/core/**"]
          allow: ["$gostd", "github.com/canhta/cred/internal/core"]

        # Only the nominator may talk to a model (L2).
        llm-isolation:
          files:
            - "**/internal/**"
            - "!**/internal/nominator/**"
          deny:
            - pkg: "github.com/anthropics/anthropic-sdk-go"
              desc: "L2 — only internal/nominator may call an LLM"

        # The MCP surface must not reach into storage; it goes through core.
        mcp-no-storage:
          files: ["**/internal/mcp/**"]
          deny:
            - pkg: "github.com/canhta/cred/internal/storage"
              desc: "MCP layer must call core, never storage directly"
            - pkg: "github.com/jackc/pgx"
              desc: "no direct database access from the MCP layer"

        # Storage must not import the MCP layer — prevents the RAGFlow cycle.
        storage-no-mcp:
          files: ["**/internal/storage/**"]
          deny:
            - pkg: "github.com/canhta/cred/internal/mcp"
              desc: "storage must not depend on the transport surface"
```

The `core-is-pure` rule with `list-mode: strict` is the one that matters. Had
RAGFlow had its equivalent, `deepdoc/parser/figure_parser.py` could never have
imported `api.db.services.llm_service`.

**2. Import cycles** — free. Go refuses to compile them. This alone makes the
specific RAGFlow failure unreachable.

**3. A layering test that fails loudly.** A cheap belt-and-braces check that
reads as documentation:

```go
func TestCoreHasNoInfrastructureDeps(t *testing.T) {
    pkgs, err := packages.Load(&packages.Config{
        Mode: packages.NeedImports | packages.NeedDeps | packages.NeedName,
    }, "github.com/canhta/cred/internal/core/...")
    require.NoError(t, err)

    for _, p := range pkgs {
        for imp := range p.Imports {
            if isStdlib(imp) || strings.HasPrefix(imp, "github.com/canhta/cred/internal/core") {
                continue
            }
            t.Errorf("internal/core must stay pure; %s imports %s", p.PkgPath, imp)
        }
    }
}
```

**4. Generated code is verified, not trusted.** `make generate && git diff
--exit-code` in CI catches sqlc drift between schema and queries.

**5. Migration checks.** Migrations are append-only and never edited: CI fails if
a committed migration file's hash changes.

**6. Coverage floor on `internal/core` only.** Coverage targets on glue code
produce theatre. A floor (say 85%) on the pure domain is meaningful, and is
achievable precisely because the core has no infrastructure dependencies.

**7. The acceptance test that matters most.** L3 is the product's core claim, so
it gets a dedicated integration test against a real Postgres via testcontainers:
change a source file's content hash, then assert every dependent claim is
invalidated **in the same transaction**. That is acceptance criterion 3, tested
directly.

---

## 5. API and schema evolution

Two surfaces evolve independently and need different strategies.

### 5.1 The MCP tool surface

**Strategy: additive by default; new tool name on breaking change; never mutate
a tool's contract in place.**

Tool discovery is dynamic — clients call `tools/list` and read schemas at
runtime — so there is no client-side compile step to break. The risk is a *model*
that has learned a tool's shape, and a client that caches it.

Rules:

1. **Adding an optional input field is safe.** Always add optional, never
   required. A new required field is a breaking change.
2. **Adding an output field is safe** if `outputSchema` does not set
   `additionalProperties: false` on the affected object. **Recommendation: do
   not set it on output schemas**, precisely to keep this door open. Do set it on
   *input* schemas, where strictness gives better model feedback.
3. **Breaking changes get a new tool name**: `recall` → `recall_v2`. Both are
   listed for a deprecation window of **two minor releases**. The old tool's
   `description` gains a leading `DEPRECATED: use recall_v2. ` — descriptions are
   read by the model, so this is the deprecation channel that actually works.
   Names remain SEP-986-compliant.
4. **Never renumber, never reuse a name.** `recall` never comes back with
   different semantics.
5. **Serve `notifications/tools/list_changed`** when the surface changes at
   runtime, declaring the `tools.listChanged` capability.

Keeping DTOs in `internal/mcp/tools_v1.go`, distinct from `core` types, is what
makes this affordable: a v2 surface is a new file mapping to the *same* core, and
the core is never versioned for transport reasons. The alternative — serialising
core structs directly — couples the wire format to the domain and makes every
domain refactor a breaking API change.

### 5.2 The claim schema

Bi-temporality (L6) helps here: nothing is deleted, things expire. That makes
most schema evolution additive by nature.

**Strategy: `schema_version` on every claim row, additive migrations only,
expand/contract for anything else.**

1. **`schema_version int not null` on `claims`.** Cheap now, essential later —
   it lets old and new claim shapes coexist while a background migration job
   upgrades rows lazily. Without it, a breaking claim-model change requires
   downtime or a full rewrite.
2. **Expand/contract for every migration**, always three deploys:
   - *Expand*: add the new nullable column; write both; read old.
   - *Migrate*: backfill via a River job; switch reads to new.
   - *Contract*: stop writing old; drop the column — **a full release later**.
   Never combine these. This is what makes rolling upgrades and rollbacks safe.
3. **`kind` is a closed set at v1** per the PRD, so store it as `text` with a
   `CHECK` constraint, **not a Postgres `enum`**. Adding a value to a Postgres
   enum is awkward and removing one is effectively impossible; changing a CHECK
   constraint is an ordinary migration. Claim kinds *will* change.
4. **Version the confidence scorer explicitly.** The PRD requires an explainable
   additive score, and D-005 stakes credibility on auditability. Persist
   `scorer_version` alongside the score, so a historical score remains
   reproducible after the scorer changes — otherwise "why does the system believe
   X" becomes unanswerable for anything scored before the last deploy.
5. **Embedding model changes are a schema change.** Store `embedding_model` and
   `embedding_dim` per row. When the model changes, write a new column or table
   and backfill via the worker — never reinterpret existing vectors under a new
   model. Mixed-model vectors in one index silently corrupt retrieval, and it is
   the kind of bug that is very hard to see.
6. **Never edit a committed migration** (enforced in CI, 4.4 #5).

### 5.3 Protocol version churn

Given the 2026-07-28 draft, this deserves explicit treatment:

- **Pin the SDK version and upgrade deliberately**, reading the changelog each
  time. Do not float.
- **Advertise the protocol version the SDK implements** — do not hand-maintain
  it.
- **Persist nothing keyed on MCP session identity.** Sessions are being removed.
- **Keep an integration test per supported protocol revision**, so a spec bump
  reports exactly what broke.

---

## 6. Summary of decisions

| # | Decision | Confidence | Chief risk |
|---|---|---|---|
| 1 | Go | High | Narrower contributor pool |
| 2 | Official `go-sdk`, stdio + Streamable HTTP | High | Breaking spec revision 2026-07-28 — mitigated by the SDK's multi-revision support |
| 3 | Resource Server only; external IdP; no AS, no DCR | High | Requires an IdP for team mode |
| 4 | One binary, two transport adapters, injected `Principal` | High | `--http --no-auth` misuse |
| 5 | `internal/core` pure, enforced by `depguard` | High | Rules must be written on day one |
| 6 | pgx/v5 + sqlc + River, Postgres only | High | River is a young dependency |
| 7 | Additive schema, `schema_version`, versioned tool names | Medium | Costs discipline before it pays |

---

## 7. What could not be verified

Stated plainly, per [D-002](../decision-log.md).

1. **WebSearch was unavailable** for this spike (session budget exhausted), so
   ecosystem and benchmark questions could not be researched directly. Findings
   below inherit that limitation.
2. **Contributor-pool claims are unquantified.** The assertion that AI-agent OSS
   skews Python/TypeScript is consistent with the reviewed projects (Mem0,
   Letta, Graphiti, RAGFlow, Onyx are Python; the MCP reference SDK is
   TypeScript) but no download, contributor, or language-share statistics were
   gathered. **This is the main input to the counter-argument in 1.4 and should
   be checked before the decision is locked.**
3. ~~The Python, TypeScript, and Rust SDK version numbers were not verified.~~
   **Resolved, and one assumption was disproven.** `rmcp` is at **2.2.0**, not
   pre-1.0 — Section 1.3 was corrected, and the case against Rust now rests on
   the absence of server-side OAuth and on community rather than Anthropic
   maintenance. Two further premises were disproven and should not be
   propagated: **pgx v6 does not exist** (v5.10.0 is current) and **`pg` v9 does
   not exist** (8.22.0).
4. ~~The Go SDK's advertised spec revision was not confirmed.~~ **Resolved.** The
   repository README confirms multi-revision support with v1.7.0+ targeting
   2026-07-28. Note the discrepancy: pkg.go.dev showed **v1.6.1 (2026-05-22)**
   with a "not in the latest version of its module" notice, so **v1.7.0+ exists
   but its exact current version and release date were not pinned down.** Check
   the releases page before pinning a version.
5. **The 2026-07-28 revision is a tagged Release Candidate, not Final.** Its
   existence is confirmed from three independent directions (releases API,
   `schema/draft/schema.ts`, the Go SDK pre-release line), so it is real — but
   RC contents can still change before it goes Final.
8. **No SDK was confirmed to enforce RFC 8707 audience matching server-side.**
   Flagged in 2.4 as work CRED must do. Worth re-confirming against the Go SDK
   directly, since a spec **MUST** is at stake.
9. **`pgx` LISTEN/NOTIFY requires `Conn.Hijack()`** to remove a connection
   permanently from `pgxpool`, and **reconnect logic is hand-rolled** — unlike
   Rust's `sqlx::PgListener`, which auto-reconnects *and* auto-re-subscribes.
   This is a small but real cost of the Go choice. It is mitigated by River
   owning the queue rather than CRED polling `LISTEN` directly, but should be
   confirmed in the vertical slice.
6. **River's operational maturity was not assessed** — no production-scale
   reports, failure modes, or maintainer-bus-factor review. Given that the
   RAGFlow queue bugs are a documented cautionary tale, this is worth a focused
   look before committing.
7. **No code was written or benchmarked.** Every ergonomic claim about the Go
   SDK comes from its documentation, not from use. The token-budgeting and
   context-assembly work in 2.4 is the least understood part and the most likely
   to surprise.

---

## 8. Suggested next step

A one-day vertical slice that de-risks the two least-verified areas at once:
`recall` and `remember` only, over stdio, backed by a real Postgres, with a
hand-written `assemble()` honouring a token ceiling. This confirms the SDK's
typed-tool ergonomics, the sqlc/pgx transaction shape needed for L3, and the
token-budget design — and produces the `depguard` configuration as a by-product.
