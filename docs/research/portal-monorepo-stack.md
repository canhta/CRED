# Portal repo layout, embedding, and polyglot tooling

The question: CRED is one Go module at the repo root (`module
github.com/canhta/cred`, code in `internal/`, `main.go` at root), and it ships a
single static, CGO-free, self-hostable binary. We want to add a web portal (a
measurement/admin dashboard). How do mature Go projects that also ship a web UI
structure the repo, embed the frontend, and orchestrate two toolchains — and
which of those choices preserve the single-binary property that is a core value?

This document surveys real projects, extracts the common pattern, and lays out
the decision forks. It does not decide them; the maintainer does. The sibling
spike `go-repo-conventions.md` already fixed the Go-side layout (`main.go` at
root, deep `internal/`, no `pkg/`); this document is about the *frontend*
addition and does not reopen that.

## Provenance labels

Same scheme as `go-repo-conventions.md`, so the two read together.

- **[REPO]** — a repository file or GitHub API response read directly. VERIFIED.
- **[FETCH]** — a URL fetched. VERIFIED.
- **[C7]** — Context7 documentation query. VERIFIED against the tool's index.
- **[JUDGEMENT]** — my recommendation. Not an established convention.
- **UNVERIFIED** — plausible, load-bearing, not checked. The check is named.

Every row in the survey table below was read first-hand via `gh api
.../contents/...` or `raw.githubusercontent.com`, or by a research agent that
fetched the same and whose claims I spot-checked against the primary files. The
directive text quoted for Prometheus, Gitea, and Grafana I pulled myself.

---

## (a) The common repo-layout pattern

### What each surveyed project actually does

Ten projects, all VERIFIED [REPO] on their default branch (`main`, except
CockroachDB `master`):

| Project | Frontend dir | Framework | JS pkg mgr | Frontend a Go module? | Go modules in repo |
|---|---|---|---|---|---|
| Gitea | `web_src/` → built to `public/` | Vue 3 + Vite | (root `package.json`) | No | 1, at root |
| Prometheus | `web/ui/` (`mantine-ui`, `react-app`, `module`) | React + Vite | pnpm/npm workspace | No | 1, at root |
| Grafana | `public/app` + `packages/` | React + Webpack | yarn + **Nx** + Lerna | No | Multiple (root + `go.work`) |
| CockroachDB | `pkg/ui/workspaces/*` | React + Redux + Webpack | pnpm workspace | No | 1, at root |
| Vault | `ui/` | **Ember** | pnpm | No | 3 (root + `api/` + `sdk/`) |
| Consul | `ui/` (`packages/consul-ui`) | **Ember** | pnpm workspace | No | 3 (root + `api/` + `sdk/`) |
| SigNoz | `frontend/` | React 18 + Vite | pnpm workspace | No | 1, at root |
| Coder | `site/` | React 19 + Vite 8 | pnpm | No | 1, at root |
| Woodpecker CI | `web/` | Vue 3 + Vite 8 | pnpm | No | 1, at root |
| Syncthing | `gui/` (themed static) | AngularJS (vendored) | none | No | 1, at root |

Sources, spot-checked directly: Gitea root has `web_src/`, `public/`,
`vite.config.ts`, `package.json`, `go.mod`, `main.go` all at the top level
[REPO, `gh api repos/go-gitea/gitea/contents`]. Prometheus `web/ui/` holds
`mantine-ui`, `react-app`, `module`, `pnpm-workspace.yaml` [REPO, `gh api
repos/prometheus/prometheus/contents/web/ui`]. Grafana root has `go.mod`,
`go.work` (with ~30 submodules under `apps/*` and `pkg/*`), `package.json`,
`nx.json`, `packages/`, `pkg/`, `public/` [REPO, `gh api
repos/grafana/grafana/contents` and `.../contents/go.work`]; its frontend is
served from disk (`public/build`), not embedded. Cockroach, Vault, Consul,
SigNoz, Coder, Woodpecker, and Syncthing rows from agent fetches of `contents`
and `package.json`, cross-checked against the embed files quoted in section (b).

### The pattern, extracted

1. **The frontend is a single top-level directory, and it is never a Go
   module.** All ten. The directory name varies — `ui/` (Vault, Consul,
   Prometheus, Cockroach), `web/` (Woodpecker; Gitea's *build output*),
   `frontend/` (SigNoz), `site/` (Coder), `gui/` (Syncthing), `public/`
   (Grafana) — but the shape is identical: one directory, a plain JS project or
   static-asset tree, with no `go.mod` of its own. **Zero** of the ten use the
   `apps/web/` + `apps/api/` layout. [REPO] That layout is a JavaScript-monorepo
   convention (Nx/Turborepo scaffolding); it does not appear in this cohort even
   in the one project that uses Nx.

2. **The Go module stays at the repo root.** All ten keep `go.mod` (and the main
   entry) at the top level, with the frontend as a subdirectory beside it. Vault
   and Consul carry *extra* modules (`api/`, `sdk/`) for their client libraries,
   but the main binary's module is still at root and the frontend is still not a
   module. [REPO] Nobody moved the Go module under a subdirectory for symmetry
   with the frontend.

3. **`ui/` and `web/` are the two most common names.** [REPO] For CRED, `web/`
   reads most naturally for "the web portal" and does not collide with Go's
   convention that `internal/` is where packages live. (`ui/` is equally
   defensible; the choice is cosmetic.)

**The trade-off in this pattern, named:** keeping the frontend as a subdirectory
of a root Go module means the JS toolchain and the Go toolchain share a working
tree, so `git status`, editor indexers, and CI checkouts see both. That is the
cost SigNoz pays for by making `frontend/` a *self-contained* pnpm workspace
(its `pnpm-workspace.yaml`, `.npmrc`, lockfile all live inside `frontend/`)
[REPO] — the JS tooling is namespaced to one directory so it does not leak into
the repo root. That containment is the one refinement worth copying regardless
of which name is chosen.

---

## (b) The `go:embed` serving pattern and the dev proxy

### Embedding is how the single-binary projects ship the UI — and only them

The split in the survey is sharp and it maps onto the single-binary property:

| Ships UI **inside** the binary (go:embed) | Ships UI as a **directory** beside the binary |
|---|---|
| Gitea, Prometheus, Vault, Consul, CockroachDB, Coder, Woodpecker | Grafana, SigNoz |
| Syncthing (older code-generator, same effect) | |

Grafana's root `embed.go` embeds only CUE schemas, **not** the frontend
(`//go:embed cue.mod/module.cue`) [REPO, `gh api
repos/grafana/grafana/contents/embed.go`]. Grafana ships `public/build` as files
in its image. SigNoz serves the built frontend off disk at runtime —
`http.FileServer(http.Dir(config.Directory))`, default `/etc/signoz/web`, with
`pnpm build` output copied next to the binary in the Dockerfile [REPO,
`pkg/web/routerweb/provider.go`]. Both are multi-service products that never
committed to single-binary distribution.

**The projects in CRED's category — self-hostable, single-binary — all embed.**
[JUDGEMENT, grounded in the table] That is the decisive alignment: `go:embed` is
what makes "one file you can `scp` and run" true for a Go+UI tool. Grafana's and
SigNoz's disk-directory approach is the thing CRED's core value rules out.

### The dominant embedding shape: embed behind a build tag, disk fallback

The mature embedders do not embed unconditionally. They put the `go:embed` in a
tagged file and provide a disk-serving (or stub) file for the other tag, so the
Go binary **builds without the frontend present**:

- **Vault** — `http/assets.go`: `//go:build ui` then `//go:embed web_ui/*`,
  with `assetFS()` doing `fs.Sub(content, "web_ui")`; `http/assets_stub.go`
  compiles when the `ui` tag is absent. [REPO,
  `raw.githubusercontent.com/hashicorp/vault/main/http/assets.go`]
- **Prometheus** — `web/ui/assets_embed.go`: `//go:build builtinassets`; the
  actual `//go:embed` line is *generated* from `web/ui/embed.go.tmpl` by
  `build_ui.sh` during `make ui-build`; `web/ui/ui.go` carries `//go:build
  !builtinassets` and serves from disk (`http.Dir("./web/ui/static")`) instead.
  [REPO, `gh api repos/prometheus/prometheus/contents/web/ui/*`]
- **Coder** — `site/site_embed.go`: `//go:build embed` then `//go:embed
  all:out`; the untagged `site/site.go` takes an injected `SiteFS fs.FS`. Real
  UI is compiled in only with `-tags embed`. [REPO,
  `raw.githubusercontent.com/coder/coder/main/site/site_embed.go`]
- **Gitea** — `modules/public/public_bindata.go`: `//go:build bindata` +
  `//go:embed bindata.dat`; `public_dynamic.go`: `//go:build !bindata` serves
  from disk. [REPO, `gh api
  repos/go-gitea/gitea/contents/modules/public/*`]
- **CockroachDB** — `pkg/ui/distccl/distccl_no_bazel.go`: `//go:build !bazel` +
  `//go:embed assets/*`; `pkg/ui/ui.go` holds a stub `var Assets fs.FS`. [REPO]

**Woodpecker is the simplest counter-example: it embeds unconditionally.**
`web/web.go`: `//go:embed all:dist/*` with an `HTTPFS()` helper that `fs.Sub`s
into `"dist"` [REPO,
`raw.githubusercontent.com/woodpecker-ci/woodpecker/main/web/web.go`]. This is
the least code, at the cost that `go build ./...` fails if `web/dist/` has not
been built (the directory must contain at least one file, see the constraint
below).

**The build-tag pattern exists to solve exactly one gotcha** [JUDGEMENT]: a
`//go:embed dist` line requires `dist` to exist and be non-empty at compile
time, so a fresh clone that has not run `vite build` cannot build the Go binary.
Gating the embed behind a tag (`-tags embed`/`ui`/`builtinassets`) lets
`go build` and `go test` run against a disk fallback (or a stub) during
development, and only the *release* build produces the frontend and passes the
tag. The recommendation for CRED depends on how much that dev-friction matters
(see fork 2).

### The `go:embed` constraints that dictate file placement

From the official package docs, verbatim [C7 / FETCH, `pkg.go.dev/embed`]:

- *"Patterns may not contain '.' or '..' or empty path elements, nor may they
  begin or end with a slash."* — **an embed pattern cannot reach a parent
  directory.**
- *"files with names beginning with '.' or '_' are excluded"* unless the pattern
  uses the **`all:`** prefix: `//go:embed all:dist` includes dotfiles that
  `vite build` may emit (e.g. `.vite/manifest.json`). This is why Woodpecker and
  Coder write `all:` — VERIFIED in their source [REPO].
- *"each pattern in a //go:embed line must match at least one file or non-empty
  directory"* and *"Matches for empty directories are ignored."* — the dist must
  exist and be non-empty at build time.
- *"Patterns must not match files outside the package's module … or any
  directories containing go.mod (these are separate modules)."*

**The load-bearing consequence** [JUDGEMENT]: because a pattern cannot contain
`..`, the `.go` file holding the `//go:embed` **must live in a directory at or
above the built assets**. This is why every embedder puts the embed file *inside
or beside* the frontend output — Woodpecker `web/web.go` embeds `web/dist`,
Consul `agent/uiserver/uiserver.go` embeds `agent/uiserver/dist` (assets are
*copied* there during build), Vault `http/assets.go` embeds `http/web_ui`
(copied there). CRED cannot put the embed directive in `internal/server/` and
reach `../../web/dist`. Options: (i) embed file at the frontend directory, e.g.
`web/embed.go` embedding `web/dist`, exposing an `fs.FS` the server imports; or
(ii) copy the built dist into a package under `internal/` and embed there. Real
projects do (i) or the copy variant of (ii). [REPO]

### Serving: SPA history-fallback

An SPA needs unmatched non-API routes to return `index.html` so client-side
routing works, while genuine asset misses still 404. The community-standard
shape [FETCH, `hackandsla.sh/posts/2021-11-06-serve-spa-from-go`]: wrap the
`http.FileServer` over the embedded FS, and when it would write a 404, serve
`index.html` instead — but **only for requests whose `Accept` header includes
`text/html`**, so that a missing `/api/...` or a missing `.js` returns a real
404 rather than an HTML page. The article's own example uses `os.DirFS`; for the
embedded case the only change is `fs.Sub(embeddedFS, "dist")` in place of
`os.DirFS`. This is verified as the shape; the exact handler is ~30 lines and
would be written, not imported. UNVERIFIED that any single library is worth a
dependency here — the logic is small enough to own.

**Asset caching gotcha** [JUDGEMENT, standard practice]: Vite emits
content-hashed filenames (`app.4f3a1b.js`) for everything except `index.html`.
The correct headers are `Cache-Control: public, max-age=31536000, immutable` for
the hashed assets and `no-cache` (or a short max-age) for `index.html`, so a
deploy is picked up immediately while assets cache forever. `http.FileServer`
does not set these; the wrapper must. This is a real, easy-to-miss correctness
issue, not a nicety — a long-cached `index.html` pins clients to a stale bundle.

### Dev-time story: two valid shapes

**Shape A — Vite proxies `/api` to Go.** Run `vite dev` (port 5173) and the Go
server (its own port) separately; `vite.config` proxies API calls to Go:

```js
// vite.config — VERIFIED shape [C7, vitejs/vite server-options]
server: {
  proxy: { '/api': { target: 'http://localhost:8080', changeOrigin: true } },
}
```

The browser hits the Vite dev server, which gives HMR for the frontend and
forwards `/api` to Go. Vite's proxy **only exists in dev** — after `vite build`
it is gone [FETCH, `vite.dev/guide/backend-integration`], which is fine because
production serves from the embedded FS, not the dev server.

**Shape B — Go proxies to the Vite dev server.** The Go binary is the single
entry point in dev too, and reverse-proxies non-API routes to `vite dev`. Gitea
does exactly this: `modules/public/vitedev.go` builds an
`httputil.ReverseProxy` to `http://localhost:<port>`, reading the port from
`public/assets/.vite/dev-port` [REPO, `gh api
repos/go-gitea/gitea/contents/modules/public/vitedev.go`]. Shape B keeps one
origin (no CORS, one URL) at the cost of more Go glue.

[JUDGEMENT] Shape A is less code and the common default; Shape B is worth it
only if you want a single dev URL. For a solo maintainer, Shape A.

---

## (c) Polyglot tooling: when a monorepo tool is warranted

### What the survey shows about tooling

The single JS-monorepo tool in the cohort is **Grafana's Nx** (`nx.json` at
root) [REPO] — and Grafana is a repo with a `packages/` set of many shared
frontend libraries plus `public/app`, i.e. *multiple* JS packages that share
code. CockroachDB, Vault, Consul, SigNoz, Prometheus use a **pnpm (or npm)
workspace** at most, and only because they too have more than one JS package
(Prometheus: `mantine-ui` + `react-app` + `module`; Cockroach: `db-console` +
`cluster-ui`; Consul: `packages/consul-ui` + siblings). Coder and Woodpecker,
which have **one** frontend app, use a plain `package.json`, no workspace.
[REPO] Cross-language orchestration ("build the UI, then build Go") is a
**Makefile or a shell script** — Prometheus `make ui-build` → `build_ui.sh`
[REPO], Gitea a Makefile. None of the surveyed projects uses Nx, Turborepo,
Bazel, or Moon to drive the *Go+JS boundary* (Cockroach uses Bazel, but for the
whole build, and it is widely regarded as its own tax).

### The comparison for a ONE-Go-module + ONE-frontend-app repo

| Tool | What it is | Cost for CRED's shape | Warranted here? |
|---|---|---|---|
| **Makefile** | Incumbent in Go repos; POSIX make | Tab/portability warts; opaque to newcomers | Yes — minimal, universal |
| **Taskfile (go-task)** | Single Go binary, zero deps, no Node; YAML tasks [FETCH, taskfile.dev] | One more tool to install (but it is *a Go binary*) | Yes — best Go-first fit |
| **pnpm/npm/yarn/bun workspace** | JS package-manager workspaces | Solves a problem you don't have with one app | No — one app is not a workspace |
| **Turborepo** | Rust task-runner + cache for JS/TS monorepos [FETCH, daily.dev] | Node-centric; value is task-graph caching across many JS packages | No — needs many packages |
| **Nx** | Integrated JS monorepo platform; `@nx-go` plugin exists [FETCH, npmjs @nx-go/nx-go] | Heavy config, Node runtime, plugin upkeep; `@nx-go` is community-maintained, last npm release ~1yr old | No — overkill; and it would put Go under Nx's model |
| **Bazel** | Hermetic polyglot build | Large adoption tax; Cockroach-scale problem | No |
| **Moonrepo / proto** | Rust task-runner + toolchain manager | Declarative, but another ecosystem to learn | No |
| **mise** | Polyglot tool-version manager + tasks [FETCH, mise.jdx.dev] | Real value for *pinning toolchain versions* (Go, Node); tasks are a bonus | Optional — for version pinning, not orchestration |

### Verdict on the monorepo-tool question

[JUDGEMENT, grounded in the survey]

**A JS "monorepo tool" (Nx/Turborepo) is not warranted for one Go module plus
one frontend app.** Their core value — task-graph caching, affected-project
detection, cross-package dependency graphs — only pays off with *many*
packages, which is precisely why the only surveyed adopter (Grafana) has a
`packages/` library set. A package-manager *workspace* (pnpm/yarn) is likewise
unwarranted with a single app; a workspace exists to link multiple JS packages,
and CRED would have one.

**The pragmatic best practice the survey supports is: Go module at root, a
`web/` app beside it, and a thin task runner (Taskfile or Makefile) driving
`go build` and `vite build`.** This is Coder's and Woodpecker's shape (one app,
plain `package.json`, Make/script glue) and it is the smallest thing that works.

**Taskfile vs Makefile** is a genuine Go-first tie-break: Task is itself a
single static Go binary with zero runtime dependencies and no Node [FETCH,
taskfile.dev], which matches CRED's own values better than Make's portability
quirks; but Make is the incumbent the surveyed Go projects actually use, and it
is already installed everywhere. Note the sibling spike's standing advice
was *"skip a Makefile at this stage"* (`go-repo-conventions.md`, §"What to
skip") — but that was decided for a **single-toolchain** repo where `go test
./...` needs no wrapper. Adding a second toolchain (`vite build` + asset copy +
tagged Go build) is the event that changes that call: an orchestrator now
sequences steps that a bare `go build` cannot. The trade-off is one file and one
tool to learn.

**When would a monorepo tool earn its complexity?** When CRED grows a *second*
JS package that shares code with the portal (a design-system package, an
embeddable widget), or a second deployable that needs incremental-build caching
in CI. At that point a pnpm workspace comes first, and Nx/Turborepo only if the
build graph gets slow enough that affected-detection saves real CI minutes. One
app is far below that line.

---

## (d) Go-module placement: root vs `apps/api`

### The fact that removes half the fear

Go's module path is the string in `go.mod` (`module github.com/canhta/cred`) and
is **independent of the directory the file sits in**. Moving `go.mod` from the
repo root to `apps/api/go.mod` does **not** change import paths —
`github.com/canhta/cred/internal/...` stays valid, because the import path is
`<module-path>/<dir-relative-to-go.mod>`. [C7, go.dev module semantics]
UNVERIFIED only in the trivial sense that I did not run `git mv` and rebuild;
the semantics are documented and stable.

### What actually changes if you move it

Everything that is *pathed to the module root* moves, and this is where the cost
lives [JUDGEMENT]:

- **`internal/` visibility** is relative to the directory containing `go.mod`.
  Move `go.mod` to `apps/api/`, and `apps/api/internal/` is importable only
  within `apps/api/` — which is fine, but the tree `internal/` must move under
  `apps/api/` with it.
- **`go:embed`** patterns are relative to the source file, and *cannot contain
  `..` and cannot cross a directory containing another `go.mod`* (§b). If the Go
  module is at `apps/api/` and the frontend at `apps/web/`, the API module
  **cannot embed `../web/dist`** — both a `..` violation and (if `web` ever
  gets its own tooling boundary) a cross-tree reach. The dist must be *copied*
  into the API module's tree before embedding. This is a real, recurring build
  step, not a one-time cost.
- **Docker `COPY`, CI `working-directory`, and every `go build ./...`** call
  must be re-pathed to `apps/api/`. Mechanical, but touches every pipeline file.

### What real projects choose

All ten surveyed projects keep the Go module at the **repo root** with the
frontend in a subdirectory. [REPO] None adopts the `apps/api` + `apps/web`
symmetry. The `apps/*` layout is what Nx/Turborepo scaffold, and it does not
appear in this Go-first cohort — not even in Grafana, the Nx user, which keeps
`go.mod` at root and the Go code in `pkg/`.

### Verdict

[JUDGEMENT] **Keep the Go module at the repo root; add the frontend as `web/`.**
This is what every surveyed project does, it keeps CRED's *existing* layout
(`main.go` at root, `internal/`) untouched, and it keeps the `go:embed` path
short (`web/embed.go` embedding `web/dist`, no `..`). Moving to `apps/api` buys
directory symmetry with `apps/web` and nothing else, while forcing an asset-copy
step before every embed and re-pathing every build file. The symmetry is
aesthetic; the costs are mechanical and recurring. The only real reason to move
would be a *third* top-level component that made `apps/*` genuinely
descriptive — which CRED does not have.

**The cost of staying at root, named:** the repo root mixes Go and JS
conventions (a `web/package.json` sits a directory away from `go.mod`), and a
casual reader might expect `apps/`-style symmetry. That is a documentation
problem (a `README` line), not a build problem.

---

## Decision forks (synthesis)

The maintainer chooses; each fork carries my lean, labelled as a lean.

1. **Frontend directory placement.** Root-level `web/` (frontend as a
   subdirectory of the root Go module) **vs** `apps/web` + Go to `apps/api`.
   *Lean: `web/` at root.* All 10 surveyed projects do this; moving Go buys only
   symmetry, and forces an asset-copy before `go:embed` (cannot cross `..`).

2. **Ship the UI: embed vs directory.** `go:embed` into the binary **vs** ship
   `web/dist` as files beside the binary (Grafana/SigNoz style). *Lean: embed.*
   Every single-binary self-hostable project in the survey embeds; the
   directory approach is exactly what CRED's core single-binary value rules out.

3. **Embed unconditionally vs behind a build tag.** Woodpecker's `//go:embed
   all:dist/*` always **vs** Vault/Prometheus/Coder's tagged embed + disk
   fallback. *Lean: tagged (`-tags embed`) with a disk/stub fallback.* It costs
   ~20 extra lines but lets `go build`/`go test` run on a fresh clone before
   `vite build` exists — which a solo maintainer hits constantly. Take
   Woodpecker's unconditional form only if you accept that the Go build depends
   on the JS build always having run.

4. **Dev proxy direction.** Vite proxies `/api` → Go (Shape A) **vs** Go
   reverse-proxies to `vite dev` (Shape B, Gitea's `vitedev.go`). *Lean: Shape
   A.* Less code; single-origin (Shape B) matters only if you want one dev URL.

5. **Cross-toolchain orchestration.** Taskfile (go-task) **vs** Makefile **vs**
   nothing. *Lean: a thin Taskfile* — it is a single Go binary with no Node,
   matching CRED's values; Makefile is the equally-valid incumbent. This
   supersedes the sibling spike's "skip a Makefile" call, which was made for a
   single-toolchain repo; a second toolchain is the trigger that earns an
   orchestrator.

6. **JS monorepo tool / workspace.** Nx/Turborepo/pnpm-workspace **vs** a plain
   `package.json`. *Lean: plain `package.json`, no monorepo tool, no workspace.*
   One app is below the line where any of them pays off; the only surveyed Nx
   user (Grafana) has a whole `packages/` library set. Revisit only when a
   *second* shared JS package appears — pnpm workspace first, Nx/Turborepo only
   if CI build-graph time demands affected-detection.

7. **Frontend framework/build.** *Lean (weak): React or Vue on Vite.* The survey
   is overwhelmingly Vite-based for anything built in the last few years (Gitea,
   Prometheus, SigNoz, Coder, Woodpecker all Vite); Vite's dev-proxy and
   manifest story is the one this document verified. Framework (React vs Vue)
   is a maintainer-taste call the evidence does not decide.

Two gotchas that are not forks but will bite regardless: the `go:embed` pattern
**cannot contain `..`**, so the embed `.go` file must sit at/above `web/dist`
(put it in `web/`); and Vite's content-hashed assets need
`Cache-Control: immutable` while `index.html` needs `no-cache`, set by the SPA
handler, or deploys serve stale bundles.
