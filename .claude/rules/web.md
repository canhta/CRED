---
paths:
  - "web/**"
  - "package.json"
  - "Taskfile.yml"
  - "internal/api/**"
  - "internal/cli/web.go"
  - "assets_embed.go"
  - "assets_noembed.go"
---

# Web portal rules

Derived from [portal-monorepo-stack.md](../../docs/research/portal-monorepo-stack.md)
and [portal-api-and-frontend.md](../../docs/research/portal-api-and-frontend.md),
which cite the Go+UI projects each convention came from. Read them when a rule
here seems arbitrary.

## 1. The repo is Go-first, not a JS monorepo

The Go module stays at the repo root. The SPA is one directory, `web/`. There is
no `apps/web` + `apps/api` split and no workspace tool (Nx, Turborepo, pnpm
workspaces) — one frontend app is far below where any of them pays off. A plain
`package.json` in `web/` and a thin `Taskfile.yml` at the root orchestrate the
two toolchains. This matches every Go+UI project surveyed; do not reintroduce a
JS-monorepo layout.

## 2. One binary still ships everything

`cred web` serves the API, the embedded SPA, and the SPA history fallback from a
single CGO-free binary. The build stays `CGO_ENABLED=0`. The Vite build embeds
via `go:embed` behind the `embed` build tag; without the tag a disk/stub
fallback keeps `go build` and `go test` working before `web/dist` exists. The
embedding file lives at the repo root — `go:embed` cannot reach through `..`, so
it must sit at or above `web/dist`.

Content-hashed assets are served `Cache-Control: public, max-age=31536000,
immutable`; `index.html` is served `no-cache`. Serving a stale bundle after a
deploy is the failure this prevents.

## 3. Types are generated from the Go structs; routes are centralized

The Go API is plain **Gin** handlers with typed request/response structs. There is
no OpenAPI spec — the console is the only consumer, so a spec + generated client
is machinery it does not earn. **tygo** generates the TypeScript types from those
Go structs into `web/src/api/types.ts`. **Never hand-write a TS type that mirrors
a Go struct** — regenerate with `task gen:types` after changing a request or
response struct; the generated file is committed so a fresh clone type-checks.

Routes and fetch calls are hand-written but **centralized in `web/src/api/`** —
one module of typed endpoint functions over a single route-constant table, so a
URL lives in exactly one place. The accepted cost: a mistyped route is a runtime
404, not a `tsc` error; centralizing the endpoints is what keeps that cost small.
If CRED later exposes a documented external HTTP API, revisit OpenAPI then — not
for the console.

## 4. The stack, and what not to add

- UI: **Astryx** (`@astryxdesign/core` + `@astryxdesign/theme-neutral`),
  React 19. Path A — no StyleX build: import `core/reset.css`, `core/astryx.css`,
  `theme-neutral/theme.css` once at the entry and wrap the app in the theme
  provider. No `<div>` for layout — compose `AppShell`, `SideNav`, `Layout`,
  `Stack`; dense data is `Table`/`List`. Style through component props first,
  then tokens (`var(--color-*)`, `var(--spacing-*)`, `var(--radius-*)`) — never
  raw hex or px, never a utility class or hand-rolled `.css`. Discover components
  with `npx astryx component <Name>` / `search` before writing UI; do not guess a
  prop exists. Read `web/.claude/CLAUDE.md` (Astryx's own agent guide) too.
- Routing: TanStack Router (typed routes and search params).
- Server state: TanStack Query. Component state stays local; reach for a global
  store only when two distant components share mutable state, and say why.
- Tables: Astryx `Table` (it ships sort/filter/pagination/selection hooks) — do
  not add TanStack Table.
- Charts: Astryx's own `@astryxdesign/charts`, used directly to keep a single
  design system and token set. It is canary — do not pin, and if it proves
  unusable the fallback is Recharts themed from `useTheme` tokens.
- Tests: Vitest + @testing-library/react (>=16, for React 19) on jsdom.

Dependencies track latest (no pinned versions); `npx astryx upgrade` is the
ritual after an `@astryxdesign/*` bump. Adding a *new* library is still a
decision with a cost — prefer Astryx and the stack above; a new one needs a
reason a reviewer would accept.

## 5. Auth is a seam from the first commit

Every API request carries a principal. Today it comes from a header
(`X-CRED-Principal`), optionally gated by a shared bearer token — honest for a
self-hosted box, and explicitly not full authentication. It resolves in one
middleware so that OIDC/SSO replaces that middleware later without touching a
handler. No handler reads the header directly; it reads the principal the
middleware put on the request context.

## 6. Recall the engine's laws hold at the API boundary too

The API is a projection over the store, not a second model. Recall through the
API is evaluated against the caller's principal with the same ACL intersection
the CLI uses — the store returns rows, `internal/acl` decides. An endpoint that
returns claims a principal may not see is the L5 failure, now over HTTP.

## 7. Comments and tests

Comments follow the Go rule: sparse, self-contained, explaining why, never
citing the PRD, a decision number, or a law number. A component test asserts
behavior a user can observe (a row renders, a filter narrows, an error shows),
not implementation detail. Test the screen through the generated client with the
network mocked, not the fetch plumbing.
