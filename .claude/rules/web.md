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

## 3. The type contract is generated, never hand-written

The Go API is defined with Huma (on the `humago` stdlib adapter). Huma emits the
OpenAPI spec at `/openapi.json`; hey-api generates the TypeScript client into
`web/src/client/` from that spec. **Never hand-write a request type, a URL
string, or a fetch call that duplicates what the generated client provides.** A
shape drift is meant to fail `tsc`, not surface as a runtime 400. Regenerate the
client with `task gen` after changing a Go handler's request or response struct;
the generated directory is committed so a fresh clone type-checks.

## 4. The stack, and what not to add

- Routing: TanStack Router (typed routes and search params).
- Server state: TanStack Query. Component state stays local; reach for a global
  store only when two distant components share mutable state, and say why.
- Tables: TanStack Table.
- Components: shadcn/ui + Tailwind — the components are copied into
  `web/src/components/ui/` and owned here, not imported from a kit.
- Tests: Vitest + @testing-library/react on jsdom.

Adding a dependency is a decision with a cost (a maintained surface, a CVE
stream). Prefer the platform and the stack above; a new library needs a reason
a reviewer would accept.

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
