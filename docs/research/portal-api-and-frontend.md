# Portal API and frontend — best-practice research

- **Date:** 2026-07-21
- **Status:** Research complete, decision forks open (see end)
- **Scope:** The web portal for CRED — a read-mostly admin/observability
  dashboard over a single-binary, CGO-free Go backend. Covers the Go HTTP/API
  layer, the Go↔TypeScript typed contract, the React stack, and the Go-native
  no-SPA alternative (templ + htmx).
- **Method:** Web search plus direct fetch of primary docs. Every claim carries
  a source. **VERIFIED** = I fetched the page or ran the command.
  **VERIFIED (snippet)** = taken from a search-result excerpt, not the full
  page; the URL is given and would confirm it. This project has a fabrication
  incident on record (`docs/README.md`); labels are not decoration.

Provenance note: this is a commissioned research task. No product code was
written.

---

## (a) API-layer verdict

**Lean: stdlib `net/http` with the Go 1.22+ `ServeMux`, adding Huma only if the
OpenAPI-generated TS client (see part b) is chosen.**

Go 1.22 moved method-based routing and path parameters into the standard
library. `ServeMux` now supports:

- Method matching — `mux.HandleFunc("GET /claims/{id}", h)`; `GET` also matches
  `HEAD`, other methods match exactly, and a wrong method returns `405` with an
  `Allow` header automatically.
- Single-segment wildcards `{id}` and multi-segment `{path...}`, read with
  `req.PathValue("id")`.
- Specificity-ranked precedence (`/posts/latest` beats `/posts/{id}`);
  genuinely ambiguous patterns **panic at registration**, not at request time.
  VERIFIED — fetched <https://go.dev/blog/routing-enhancements>.

The Go team states the design intent plainly: "Adding these features to the
standard library means one fewer dependency for many projects. But third-party
web frameworks remain a fine choice for current users or programs with advanced
routing needs." VERIFIED — same page.

What `ServeMux` still does **not** give you, and third-party routers do:
first-class middleware chaining, route groups/sub-routers, and regex/type
constraints on path params. The Go blog acknowledges this by omission (it lists
no middleware or grouping story). VERIFIED — same page. Middleware is trivially
hand-rolled as `func(http.Handler) http.Handler`, and CRED's surface is small
(claims list, evidence, retrieval inspector, usage views, plus a few write
endpoints for principals/scopes/ACLs), so groups and regex constraints buy
little.

Where the frameworks sit for context:

- **chi** — a thin router that stays `net/http`-compatible; its selling point
  (idiomatic middleware, sub-routers) is now largely what stdlib + a few
  helpers give you. Reasonable, but a dependency for a shrinking gap. VERIFIED
  (snippet) — <https://reintech.io/blog/go-chi-vs-gin-vs-echo-web-framework-comparison-2026>.
- **gin / echo** — batteries-included (binding, validation, rendering). Gin is
  the most-used Go web framework (~48% in the 2025 Go Developer Survey /
  JetBrains report). For a small typed JSON API this is more surface than the
  job needs. VERIFIED (snippet) —
  <https://blog.jetbrains.com/go/2025/11/10/go-language-trends-ecosystem-2025/>.
- **Huma** — a router-agnostic framework (sits on chi, gin, or stdlib) that is
  code-first: you declare input/output structs and it does validation, request
  parsing, and **auto-generates and serves an OpenAPI 3.1 spec** at
  `/openapi.json` (and 3.0 variants). Requires Go 1.25+. VERIFIED — fetched
  <https://huma.rocks/features/openapi-generation/>; version from
  <https://github.com/danielgtaylor/huma>. Huma is the bridge that makes the
  OpenAPI type-contract path (part b) nearly free on the Go side.
- **connect-go** — see part b; it is an RPC/schema choice, not just a router.

**Cost of the lean:** hand-rolled middleware and no built-in request validation
if you stay pure-stdlib. If you adopt the OpenAPI contract, you take Huma
anyway, which erases both costs — so the practical fork is "pure stdlib +
hand-written types" vs "Huma + generated types," decided in part b, not here.

---

## (b) The Go↔TypeScript typed contract (the crux)

Four mainstream families. The question is how frontend types stay in sync with
Go with the least ceremony for a small, self-hosted app.

| Approach | One schema source | Go side | TS side | Extra toolchain | Gives a client? |
|---|---|---|---|---|---|
| **A. OpenAPI, code-first** | Go structs | Huma (auto-spec) or swaggo | openapi-typescript / hey-api / orval | Node codegen step | Types; client optional |
| **A′. OpenAPI, spec-first** | `openapi.yaml` | oapi-codegen | same TS generators | YAML + 2 codegens | Yes, both sides |
| **B. Protobuf + Connect** | `.proto` | connect-go | connect-es + protobuf-es | buf CLI + protoc plugins | Yes, typed client both sides |
| **C. Hand-written shared types** | none (duplicated) | write structs | write interfaces | none | No |
| **D. tygo** | Go structs | (structs) | tygo → `.d.ts` | one Go tool | No — types only |

Details and evidence:

- **A / A′ — OpenAPI.** `oapi-codegen` generates Go server/client boilerplate
  and models from an OpenAPI 3.0/3.1 spec (spec-first). VERIFIED (snippet) —
  <https://github.com/oapi-codegen/oapi-codegen/>. Code-first inverts this:
  Huma emits the spec from your Go types, so the Go structs are the single
  source and no YAML is hand-maintained. VERIFIED —
  <https://huma.rocks/features/openapi-generation/>. On the TS side the field
  has consolidated: **openapi-typescript** emits types only (thin, fast);
  **hey-api / `openapi-ts`** is the current frontrunner and the successor to
  `openapi-typescript-codegen`, with a plugin architecture and a TanStack Query
  plugin; **orval** generates framework hooks and mocks. Guidance for new
  projects: hey-api. VERIFIED (snippet) —
  <https://dev.to/nyaomaru/which-openapi-codegen-should-you-choose-openapi-typescript-vs-hey-api-vs-orval-vs-kubb-100p>.

- **B — Protobuf + Connect.** One `.proto` generates typed Go handlers
  (connect-go) and a typed TS client (connect-es + protobuf-es) whose ergonomics
  Buf claims match "a well-written REST client," staying close to `fetch`; a
  `connect-query-es` package wires it into TanStack Query. Connect is
  wire-compatible with gRPC/gRPC-Web and speaks its own HTTP/JSON-capable
  protocol. VERIFIED (snippet) —
  <https://github.com/connectrpc/connect-es>,
  <https://connectrpc.com/docs/faq/>. **The cost is the buf/protoc toolchain**:
  `.proto` files, `buf generate`, protoc plugins, and a second code-generation
  mental model. That is real weight to carry for a small read-mostly dashboard,
  and it is CGO-free but not toolchain-free. VERIFIED (snippet) —
  <https://github.com/bufbuild/protobuf-es>.

- **C — hand-written.** Zero tooling, immediate to start, and it drifts. Two
  copies of every type, kept in sync by discipline. Fine for a handful of
  endpoints; a maintenance tax that grows with the surface. (No external source
  needed — this is the null option.)

- **D — tygo.** One Go tool converts Go structs to TS interfaces (generics,
  const groups, `time.Time` mapping, doc-comment preservation). It generates
  **types only — no client, no RPC, no endpoint knowledge**; unmapped custom
  types fall back to `any` and unexported fields are dropped. VERIFIED —
  fetched <https://github.com/gzuidhof/tygo>. tygo syncs *shapes*; you still
  hand-write `fetch` calls and can still mistype a URL or method.

**Lean, with the trade-off named:**

1. **Default: OpenAPI code-first — Huma on Go, hey-api on TS.** One source of
   truth (Go structs), the spec falls out for free and is self-documenting
   (a `/docs` page ships in the binary), and the TS side gets typed models plus
   an optional typed TanStack Query client. This is the least-ceremony path that
   still catches URL/method/shape mismatches at compile time. Cost: a Node
   codegen step in the build, Huma pins Go 1.25+, and OpenAPI's type system
   can't express every Go type losslessly.
2. **If write endpoints stay trivial (3–5 forms) and you want zero Node
   codegen: tygo for types + hand-written `fetch`.** Lowest total toolchain —
   one Go tool. Cost: no endpoint-level type safety (URLs/methods are strings);
   suited to a genuinely tiny, stable surface.
3. **Connect/buf is worth its weight only if** you expect multiple clients, want
   streaming, or already run protobuf elsewhere. For a single self-hosted SPA
   over a small JSON API, the `.proto` + buf toolchain is more governance than
   the problem demands. It is the strongest *typing*, the heaviest *ceremony*.

Ordering by ceremony (low→high): hand-written ≈ tygo < OpenAPI code-first
(Huma+hey-api) < OpenAPI spec-first (oapi-codegen) < Connect/buf.

---

## (c) React stack recommendation

A coherent, low-ceremony set for a small dashboard, if the SPA path is chosen.
The operator's lean (React + Vitest) is sound and matches current mainstream
practice.

- **Build/test: Vite + Vitest + @testing-library/react.** Vitest is the default
  test runner across the TanStack ecosystem and shares Vite's config. VERIFIED
  (snippet) —
  <https://tanstack.com/query/v4/docs/framework/react/guides/testing>.
- **Server state: TanStack Query.** The standard async-state/caching layer;
  pairs with either generated client (hey-api's TanStack Query plugin, or
  connect-query-es). VERIFIED (snippet) —
  <https://void.ma/en/publications/tanstack-react-query-table-router-guide-complet-2025/>.
- **Data tables: TanStack Table.** Headless, handles sorting/filter/pagination/
  column resize — exactly what the claims list, retrieval inspector, and usage
  views need. VERIFIED (snippet) —
  <https://stacknotice.com/blog/tanstack-table-react-guide-2026>.
- **Router — a real fork.** TanStack Router gives end-to-end type-safe routes,
  params, and **typed search params**, which is a concrete win for
  filter-driven dashboards; React Router v7 has broader adoption and easier
  onboarding but weaker param typing by default. For a type-first SPA the
  evidence leans TanStack Router. VERIFIED (snippet) —
  <https://devtoolbox.blog/tanstack-router-vs-react-router-v7-2026/>.
- **UI — a real fork.** shadcn/ui (Radix + Tailwind, copy-in components you own,
  no dependency to version) is where greenfield is heading but ships **no
  advanced data grid** out of the box (you compose TanStack Table yourself);
  Mantine (120+ components, batteries-included) and Ant Design (richest
  enterprise tables/forms) save integration days on data-heavy admin UIs at the
  cost of owning less of your styling. For a small read-mostly dashboard,
  shadcn/ui + TanStack Table is the low-lock-in default; Mantine is the pick if
  you want components pre-built and dark mode/theming for free. VERIFIED
  (snippet) — <https://adminlte.io/blog/shadcn-ui-vs-mui-vs-ant-design/>,
  <https://dev.to/devforgedev/why-i-chose-mantine-over-shadcnui-for-every-dashboard-project-5fd0>.

Coherent default set: **Vite + Vitest + TanStack Query + TanStack Table +
TanStack Router + shadcn/ui**, fed by whichever generated client part b selects.

**Cost of the SPA path (all variants):** a second toolchain (Node, bundler,
lockfile, its own CI and CVE stream), a build artifact to embed in the Go
binary, and the type-sync problem that part b exists to solve. None of that
disappears with a good stack; it is the entry price of a SPA.

---

## (d) The Go-native alternative: templ + htmx (honest tradeoffs)

Because the backend is Go, the no-SPA path is a first-class option, not a
fallback. **templ** compiles typed Go HTML components (type checking across the
view layer, no separate template runtime); **htmx** (~14 KB, ~5 KB gzipped)
swaps server-rendered HTML fragments into the page on interaction. Datastar is a
newer alternative in the same hypermedia family. VERIFIED (snippet) —
<https://medium.com/@iamsiddharths/building-reactive-uis-with-go-templ-and-htmx-a-simpler-path-beyond-spas-17e7dad2c7a2>,
<https://dev.to/pockit_tools/htmx-in-2026-when-you-dont-need-react-and-when-you-absolutely-do-2mf4>.

**What it eliminates outright** — and this is the crux for CRED:

- **The entire part-b type-sync problem.** No JSON contract crosses a language
  boundary, so there is nothing to keep in sync. templ's Go types *are* the
  view; the compiler checks them.
- **The second toolchain.** No Node, no bundler, no npm lockfile, no JS CVE
  stream, no separate CI lane. The dashboard ships inside the single CGO-free
  binary with `go build` — which is exactly CRED's distribution model.
- **Client JS weight.** ~14 KB htmx vs a React SPA's typical 200–400 KB of JS
  before app code. Reported migrations of CRUD-heavy React SPAs to htmx cite
  40–60% less frontend code. VERIFIED (snippet) — same DEV/pockit sources.
  (Treat the 7x load-time and code-reduction figures as vendor/blog claims, not
  independently verified benchmarks — UNVERIFIED as measurements; what would
  check them is a like-for-like build of CRED's own views.)

**What it costs:**

- **Rich client-side interactivity is harder.** Anything that wants local
  optimistic state, complex client-side filtering/sorting without a round trip,
  or genuinely app-like widgets fights the model. htmx's own framing is that
  admin panels/CRUD are its sweet spot and highly interactive apps are where you
  "absolutely do" still want React. VERIFIED (snippet) —
  <https://dev.to/pockit_tools/htmx-in-2026-when-you-dont-need-react-and-when-you-absolutely-do-2mf4>.
- **More server round trips / server render load**, trading client CPU for
  network + server CPU. For a self-hosted internal tool with few concurrent
  operators this is a non-issue; at public scale it matters more. VERIFIED
  (snippet) — <https://plus8soft.com/blog/htmx-vs-react-comparison/>.
- **Smaller hiring/ecosystem pool** than React, and no TanStack Table-class
  headless data-grid — you build tables in templ or lean on server-side paging.
- **The retrieval inspector is the one view to pressure-test.** If ranking
  visualization wants heavy client-side interaction, that specific screen may
  justify a JS island even in an otherwise htmx app.

**Fit for CRED specifically:** the surface is mostly read views (claims,
evidence, why-expired diffs, retrieval ranking, usage/quota) plus a few setup
forms. That is squarely the profile every source names as htmx's strongest case,
and it aligns with the product's single-binary, small-team, self-hosted
constraints better than any SPA variant. The honest counter is team preference
and skills (the operator leans React) and the retrieval inspector's
interactivity ceiling — neither is a technical disqualifier, both are real.

---

## Decision forks (evidence-based leans, not decisions)

The four choices below are independent. Each states a **lean** with its
strongest evidence; a human picks.

**Fork 0 — Architecture: SPA vs hypermedia (decide this first; it collapses the
others).**
_Lean: templ + htmx._ For a read-mostly, self-hosted, single-binary tool with a
small team, it deletes the type-sync problem (fork 2), the second toolchain, and
the JS build entirely, and CRUD/admin is the exact use case htmx sources
endorse. Choosing React is defensible on team-skill and future-interactivity
grounds (esp. the retrieval inspector), but it is buying capacity the current
surface doesn't use. If fork 0 = templ, **forks 1–3 mostly dissolve** (you still
pick a Go router — see fork 1 — and can add JS islands per-view).

**Fork 1 — Go router: stdlib ServeMux vs a framework.**
_Lean: stdlib `net/http` ServeMux (Go 1.22+)._ It covers method + path-param
routing natively; middleware is a one-liner type. Adopt **Huma** instead only if
fork 2 = OpenAPI, since Huma is what makes that path cheap. Avoid gin/echo — more
framework than a small typed JSON API needs.

**Fork 2 — Type contract (only if fork 0 = React): OpenAPI vs Connect vs tygo vs
hand-written.**
_Lean: OpenAPI code-first — Huma (Go) + hey-api (TS)._ Single source of truth in
Go structs, spec + docs for free, typed TanStack Query client optional. Drop to
**tygo + hand-written fetch** if the surface is tiny and you'll accept
string-typed URLs. Reach for **Connect/buf** only with multiple clients,
streaming, or existing protobuf — otherwise its toolchain outweighs the payoff
here.

**Fork 3 — React stack details (only if fork 0 = React).**
_Lean: Vite + Vitest + @testing-library/react + TanStack Query + TanStack Table,
with **TanStack Router** (typed search params suit filter-driven dashboards) and
**shadcn/ui** (low lock-in; compose TanStack Table for grids)._ Swap shadcn →
**Mantine** if you'd rather have batteries-included components and theming than
own every file.

One-line synthesis: **the backend being Go makes templ+htmx the lean for THIS
surface; if the team chooses React anyway, the lowest-ceremony coherent path is
stdlib-or-Huma + OpenAPI(Huma/hey-api) + the TanStack/shadcn set.**
