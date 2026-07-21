# Go repository conventions

The question: before any product code lands, what conventions, structure, and
checks should the repository carry — and which of the things a "professional
OSS repo" is supposed to have are ceremony a solo maintainer will not sustain?

This document gathers evidence and recommends. It is not itself the rules file;
`.claude/rules/go.md` is written from it.

---

## Provenance labels

Every recommendation below is tagged with where it came from, because the
distinction between "this is what pgx v5 does" and "this is what I would do" is
what makes the document usable.

- **[C7]** — Context7 documentation query.
- **[REPO]** — a repository file or GitHub API response read directly.
- **[FETCH]** — a URL fetched.
- **[RAN]** — a command executed on this machine, with output.
- **[JUDGEMENT]** — my recommendation. Not an established convention.

---

## What was checked

| Source | Method | Result |
|---|---|---|
| 10 Go repos, top-level layout | `gh api git/trees` | VERIFIED |
| 10 Go repos, `.golangci.*` | raw.githubusercontent fetches | 8 found, 2 absent |
| 6 Go repos, CI workflows | `gh api contents` | VERIFIED |
| 5 Go repos, contributor files | `gh api contents` | VERIFIED |
| golangci-lint v2 config schema | Context7 + release API | VERIFIED |
| sqlc pgvector support | source read + changelog | VERIFIED, with a limit |
| pgx v5 error surface | source read | VERIFIED |
| River, goose error surfaces | source read | VERIFIED |
| CGO guard behaviour | executed locally | VERIFIED, **and refined** |
| AGENTS.md vs CLAUDE.md counts | `gh api search/code` | Directionally verified |

### Three corrections to the commissioning brief

**FALSIFIED: `river-queue/river` does not exist.** `gh api
repos/river-queue/river` returns 404. The repository is **`riverqueue/river`**,
no hyphen. Three independent agents hit this. Any internal note using the
hyphenated form is wrong. [REPO]

**FALSIFIED: "many serious projects avoid `testify` deliberately."** The premise
does not survive contact with the evidence. Of seven surveyed `go.mod` files,
**five depend on testify directly**, including all three of CRED's core data
dependencies — pgx, River, and goose — all pinned to v1.11.1. [REPO] The
reasoning is examined in section 4 and it does *not* hold in the form given.

**FALSIFIED: `github.com/pkg/errors` is archived.** `gh api repos/pkg/errors`
returns `"archived": false`. What is true is weaker and should be stated
accurately: self-declared maintenance mode in its README, last functional
release v0.9.1 in January 2020. [REPO] Avoid it on dormancy grounds, not on a
false archival claim.

---

## 1. Repository layout

### What the Go team actually says

`https://go.dev/doc/modules/layout` is the current official guidance. For a
module with multiple binaries it shows `cmd/prog1/main.go`, `cmd/prog2/main.go`
alongside a root package and `internal/`. Verbatim: *"Initially, it's
recommended placing such packages into a directory named `internal`; this
prevents other modules from depending on packages we don't necessarily want to
expose"*, and *"It's recommended to keep packages in `internal` as much as
possible."* [FETCH]

**The page never mentions `pkg/` at all.** [FETCH] That is a verified absence,
not an inference.

### The contested standard, checked

`golang-standards/project-layout` disclaims its own authority, in bold, in its
own README:

> "This is **`NOT an official standard defined by the core Go dev team`**. This
> is a set of common historical and emerging project layout patterns in the Go
> ecosystem."

And, directly relevant to CRED's stage:

> "**`If you are trying to learn Go or if you are building a PoC or a simple
> project for yourself this project layout is an overkill. Start with something
> really simple instead (a single main.go file and go.mod is more than
> enough).`**"

On `pkg/` specifically: *"This is a common layout pattern, but it's not
universally accepted and some in the Go community don't recommend it."* [FETCH]

**Verdict: it is not a standard, and it says so.** Do not cite it.

### What real projects do

| Repo | `internal/`? | `pkg/`? | `cmd/` |
|---|---|---|---|
| riverqueue/river | Yes, + 3 nested | No | `cmd/river`, 3 deep |
| pressly/goose | Yes, 43 dirs | Yes (1 dir — vestigial) | `cmd/goose`, 2 deep |
| jackc/pgx | Yes | No | None (library) |
| mcp/go-sdk | Yes | No | None |
| charmbracelet/bubbletea | **No, zero** | No | None — fully flat |
| charmbracelet/glow | No | No | `main.go` at root |
| golangci/golangci-lint | Yes, 37 dirs | Yes (main tree) | `cmd/`, 2 deep |
| influxdata/telegraf | Yes | No | `cmd/telegraf`, 2 deep |
| grafana/loki | 129 nested, none at top | Yes (main tree) | 10 binaries, 2 deep |
| cockroachdb/cockroach | `pkg/internal/` | Yes (root of all) | `pkg/cmd/`, ~70 |

All rows [REPO], via `gh api git/trees`.

**`internal/` is used by 8 of 10. `pkg/` by 4 of 10, and inconsistently** —
vestigial in goose (one directory), the entire source root in Cockroach, a
public-API marker in neither.

The asymmetry that explains this: `internal/` is **enforced by the toolchain**.
Go 1.4 introduced it as a compiler rule — *"a package `.../a/b/c/internal/d/e/f`
can be imported only by code in the directory tree rooted at `.../a/b/c`"* —
enforced for all repositories from Go 1.5. [FETCH] `pkg/` is cosmetic nesting
that the compiler has never heard of.

### Recommendation

**`cmd/` + `internal/`, no `pkg/`.** [JUDGEMENT, grounded in the 8/10 adoption
and the official page's silence on `pkg/`]

`internal/` is the load-bearing choice and it is load-bearing for a reason
specific to CRED: the packaging spike commits to a pre-1.0 contract where **"the
public API is the MCP tool surface — not the Go package"**
(`packaging-and-first-run.md`). Putting essentially everything under `internal/`
makes that commitment compiler-enforced rather than documented. Nobody can
import `cred/internal/temporal` and then complain when it changes.

The cost, named: if someone later wants to embed CRED as a Go library, every
package must be lifted out of `internal/` first, and that is a large mechanical
diff. **That cost is the point** — it forces the decision to be deliberate.

### Proposed tree

```
cred/
  go.mod
  main.go                    # thin: flag parsing, calls internal/cli
  cmd/                       # only if a second binary appears; see note
  internal/
    cli/                     # subcommands: serve, curate, doctor, migrate
    config/                  # env + flag resolution, CRED_* prefix
    claim/                   # Claim, Evidence, Derivation — domain types
    temporal/                # bi-temporal algebra. PURE. no database import
    acl/                     # ACL intersection algebra. PURE. no database
    recall/                  # retrieval orchestration, RRF fusion
    nominate/                # the LLM boundary; Nominate() + fake + adapters
    curate/                  # River workers: dedup, reconcile, prune, rescore
    store/
      migrations/            # *.sql, embedded via embed.FS
      pg/                    # pgx queries. the ONLY package importing pgx
      gen/                   # sqlc output, if adopted. never hand-edited
    embed/                   # onnx-gomlx path + tokenizer
    mcpsrv/                  # MCP tool registration, transport, auth
    obs/                     # slog setup, OTel setup, attribute constants
  testdata/                  # golden files, fixtures
  docs/
  .golangci.yml
  .github/workflows/ci.yml
```

Rationale per directory that is not self-evident:

- **`main.go` at root, not `cmd/cred/main.go`.** One binary today. The
  packaging spike is explicit: *"One image, two services (`serve` and
  `curate`)"* — that is one binary with two subcommands, not two binaries.
  `cmd/` is the answer to a question CRED does not currently have. glow does
  exactly this. [REPO] Add `cmd/` the day a second binary genuinely exists; the
  move is one `git mv`.
- **`temporal/` and `acl/` must not import `internal/store`.** This is the
  structural expression of the testing strategy's load-bearing instruction. See
  section 5 — it is enforced by a linter, not by good intentions.
- **`store/pg/` is the only package importing pgx.** Same reason.
- **`obs/` holds every OTel attribute name as a constant.** Directly from
  `tech-worker-ops-packaging.md`: *"isolate every attribute name behind
  constants in one module"*, because `gen_ai.*` has no stable release and
  already renamed `gen_ai.system` to `gen_ai.provider.name`.
- **No `pkg/`.** See above.
- **No `api/`, `build/`, `scripts/`, `test/`, `third_party/`.** These are
  `project-layout` directories that its own README calls overkill at this stage.

---

## 2. Linting and static analysis

### Version and schema state

Latest release is **v2.12.2, published 2026-05-06**. [FETCH,
`api.github.com/repos/golangci/golangci-lint/releases/latest`] The action is at
**v9.3.0, 2026-06-29**. [FETCH]

The v2 schema requires `version: "2"` at the top; *"The only supported value is
'2'."* [C7] Three changes that break a config written from v1 memory:

1. **Formatters moved out of `linters:`.** `gofmt`, `goimports`, `gofumpt`, and
   `gci` now live in a top-level `formatters:` block. [C7]
2. **`gosimple` and `stylecheck` were merged into `staticcheck`.** Listing them
   separately is now an error. [C7]
3. **`issues.exclude-rules` became `linters.exclusions.rules`**, and v1's
   `EXC####` default-exclusion IDs became named `exclusions.presets`. [C7]

**All 8 configs found in the survey are `version: "2"`. Zero v1 configs.**
[REPO] The v1 schema is extinct in this cohort.

Two repos have no golangci-lint config at all: `modelcontextprotocol/go-sdk`
(uses `go vet` + `dominikh/staticcheck-action` pinned to staticcheck v0.6.1) and
`cockroachdb/cockroach` (Bazel `nogo`). [REPO]

### What is actually enabled

Default set is errcheck, govet, ineffassign, staticcheck, unused. [C7]

Beyond that, across 8 configs [REPO]:

| Linter | Count | Note |
|---|---|---|
| misspell | 6/8 | strongest non-default |
| revive | 6/8 | |
| unconvert | 6/8 | |
| gocritic | 5/8 | but 3 of 5 run `disable-all` + allowlist |
| gosec | 5/8 | **never bare** — all five constrain it |

### gosec: enabled by a majority, trusted by none

Every one of the five constrains it, with reasons in the YAML [REPO]:

- **telegraf** — explicit 32-rule allowlist; comment flags **G115** with a link
  to [securego/gosec#1212](https://github.com/securego/gosec/issues/1212).
- **mimir** — excludes G301/G306: *"Relies on system umask to restrict
  directory permissions."*
- **golangci-lint** — excludes G115 and the G702–G704 taint rules: *"those
  reports are not relevant."*
- **river** — excludes G404: *"use of non-crypto random; overly broad."*
- **bubbletea** — enables it bare, but sets `run.tests: false`.

The recurring false positives are **G115** (integer overflow on conversion — the
notorious one), **G101** (hardcoded credentials, fires on test fixtures),
**G404** (weak RNG), and **G301/G302/G306** (file permissions vs umask).

### sqlvet

**FALSIFIED as abandoned.** `github.com/houqp/sqlvet` is not archived; last push
2026-07-03, steady cadence, 499 stars. [REPO] But it has **no golangci-lint
integration** and runs standalone from a `sqlvet.toml`.

**Recommendation: skip it.** [JUDGEMENT] It validates SQL string literals
against a schema. If CRED adopts sqlc (section 5), sqlc's own compile step does
that job strictly better, and running both is duplicated work for a solo
maintainer. Note that River explicitly *disables* `unqueryvet` — the
golangci-integrated SQL linter — with the comment *"bans all use of `SELECT *`;
just … sigh."* [REPO]

### The most instructive artifact

River's config is the only one that documents *why* it disables things, and it
is a direct statement of what experienced Go teams reject. Verbatim [REPO]:

- `cyclop` — *"screams into the void at 'cyclomatic complexity'"*
- `gocyclo` — *"ANOTHER 'cyclomatic complexity' checker"*
- `gocognit` — *"yells that 'cognitive complexity' is too high; why"*
- `funlen` — *"screams when functions are more than 60 lines long; what are we
  even doing here guys"*
- `mnd` — *"detects 'magic numbers', which it defines as any number; annoying"*
- `err113` — *"wants all errors to be defined as variables at the package
  level; quite obnoxious"*
- `ireturn` — *"bans returning interfaces; questionable as is, but also buggy
  as hell"*

**Counter-evidence, and it matters:** golangci-lint's own config enables
`funlen`, `gocyclo`, `mnd`, `goconst`, `lll`, and `godox` — exactly what River
rejects. But its file opens with a disclaimer: *"This configuration file is not
a recommendation. We intentionally use a limited set of linters… We have
specific constraints."* [REPO]

**I side with River.** [JUDGEMENT] Complexity-metric linters produce a constant
low-grade tax with no correctness signal, and the failure mode for a solo
maintainer is not "code gets complex" — it is "maintainer adds `//nolint`
reflexively, then stops reading lint output entirely." A linter that trains you
to ignore linters is negative value. The trade-off, named: genuinely sprawling
functions will not be caught mechanically, so they must be caught in review —
and CRED's review is the maintainer plus an agent.

### Proposed `.golangci.yml`

Targets **golangci-lint v2.12.x**. [JUDGEMENT, assembled from the majority
findings above; every non-default linter here appears in at least 4 of 8
surveyed configs, except the three marked CRED-specific.]

```yaml
version: "2"

run:
  timeout: 3m

linters:
  default: standard      # errcheck, govet, ineffassign, staticcheck, unused
  enable:
    # --- majority of surveyed configs ---
    - misspell           # 6/8
    - revive             # 6/8
    - unconvert          # 6/8
    - gocritic           # 5/8, with allowlist discipline below
    - gosec              # 5/8, constrained below
    # --- correctness, high signal, low noise ---
    - errorlint          # catches %v where %w is meant, and == on errors
    - bodyclose
    - nilerr             # returns nil after a non-nil error check
    - rowserrcheck
    - sqlclosecheck
    - nolintlint         # every //nolint must name a linter and a reason
    - testifylint        # only if testify is adopted; see section 4
    # --- CRED-specific, load-bearing ---
    - depguard           # enforces the layering laws; see section 5
    - usetesting         # pushes t.Context()/t.Chdir() over hand-rolled

  settings:
    errcheck:
      check-type-assertions: true

    govet:
      enable-all: true
      disable:
        - fieldalignment   # struct field reordering for padding; churn

    gocritic:
      disable-all: true
      enabled-checks:
        - appendAssign
        - argOrder
        - badCond
        - caseOrder
        - dupArg
        - dupCase
        - nilValReturn
        - offBy1
        - weakCond

    gosec:
      excludes:
        - G115  # integer overflow on conversion; securego/gosec#1212
        - G404  # weak RNG; CRED uses math/rand for jitter, not secrets
        - G301  # dir permissions; relies on umask
        - G306  # file permissions; relies on umask

    nolintlint:
      require-explanation: true
      require-specific: true

    depguard:
      rules:
        pure-algebra:
          # The testing-strategy law, made mechanical.
          list-mode: lax
          files:
            - "**/internal/temporal/**"
            - "**/internal/acl/**"
          deny:
            - pkg: "github.com/jackc/pgx/v5"
              desc: >-
                Temporal and ACL algebra must be pure. Postgres stores and
                filters; it does not decide. See testing-strategy.md.
            - pkg: "database/sql"
              desc: "Same law."
        driver-isolation:
          list-mode: lax
          files:
            - "**/internal/**"
            - "!**/internal/store/**"
          deny:
            - pkg: "github.com/jackc/pgx/v5"
              desc: "Only internal/store may import the database driver."

  exclusions:
    presets:
      - comments
      - common-false-positives
      - legacy
      - std-error-handling
    rules:
      - path: '_test\.go'
        linters:
          - gosec        # G101 fires on fixture credentials
          - bodyclose

formatters:
  enable:
    - gofmt
    - goimports
```

Two notes on this config:

- **`govet: enable-all` with `fieldalignment` disabled** is a deliberate
  choice. `enable-all` turns on `shadow` and `nilness`, which have real
  correctness value; `fieldalignment` demands struct reordering for memory
  padding and is pure churn at CRED's data volumes. [JUDGEMENT]
- **`depguard` is the most important entry in the file.** It is what converts
  the testing strategy's instruction from aspiration to a build failure. See
  section 5.

---

## 3. Error handling

### What the Go project itself says

From `https://go.dev/blog/go1.13-errors`, verbatim [FETCH]:

> "Wrap an error to expose it to callers. Do not wrap an error when doing so
> would expose implementation details."

> "In other words, wrapping an error makes that error part of your API. If you
> don't want to commit to supporting that error as part of your API in the
> future, you shouldn't wrap the error."

> "The choice to wrap is about whether to give *programs* additional information
> so they can make more informed decisions, or to withhold that information to
> preserve an abstraction layer."

This is canonical guidance from the Go blog, not community folklore. **`%w` is
an API commitment; `%v` is human context.**

### What the chosen dependencies do

**pgx v5** [REPO, `conn.go` and `pgconn/errors.go`]:

```go
var ErrNoRows = newProxyErr(sql.ErrNoRows, "no rows in result set")
```

`pgx.ErrNoRows` is a **proxy that unwraps to `sql.ErrNoRows`**, so
`errors.Is(err, sql.ErrNoRows)` is true for pgx errors. This is the single most
useful detail found: CRED's domain layer can match `sql.ErrNoRows` and never
import pgx.

`*pgconn.PgError` is matched with `errors.As` and carries 19 fields including
`Code`, `Message`, and — the useful one — **`ConstraintName`**, which lets a
specific unique index map to a specific domain error rather than treating all
23505s alike. [REPO, C7]

`pgerrcode` is a **separate module**, `github.com/jackc/pgerrcode`, not part of
pgx. [REPO] It ships class predicates; `IsTransactionRollback` covers
serialization failure and deadlock in one call — the correct predicate for a
retry branch.

**River** [REPO, `rivertype/river_type.go`, `error.go`]: sentinels are
`rivertype.ErrNotFound`, `ErrJobRunning`. Snooze and cancel are **control flow
via returned error**: `river.JobCancel(err)` and `river.JobSnooze(d)`. [C7
confirms both signatures] `JobSnooze` does not increment the attempt count, and
its docstring recommends `JobSnooze(0)` for graceful-shutdown interruption
rather than returning a context-cancelled error. River's error types implement a
custom `Is` that **ignores field values**, so they are matched with
`errors.Is(err, &river.UnknownJobKindError{})` — a zero-valued pointer literal,
not `errors.As`. That is unusual and easy to get wrong.

**goose** [REPO, `provider_errors.go`]: `ErrVersionNotFound`, `ErrNoMigrations`,
`ErrAlreadyApplied`, `ErrNotApplied`, and critically:

```go
type PartialError struct {
    Applied []*MigrationResult // succeeded before the failure
    Failed  *MigrationResult   // never nil
    Err     error
}
```

**Migration failure is not atomic.** `*goose.PartialError` must be handled with
`errors.As`; it carries the recovery state. Treating a migration failure as
all-or-nothing is a real bug, not a style question. Also
`lock.ErrLockNotImplemented` from the locking path. [C7]

### Recommendation

[JUDGEMENT, following the Go blog rule above]

1. **Sentinels for conditions callers branch on; typed errors when the caller
   needs data.** CRED's typed cases are few and known: an ACL denial that must
   be byte-identical to nonexistent (testing-strategy adversarial case 8), and a
   nomination rejection carrying which IDs were dropped.
2. **Wrap with `%w` only across CRED's own package boundaries where a caller is
   meant to match.** Inside a package, `%v` plus context.
3. **Never wrap a `*pgconn.PgError` upward out of `internal/store`.** Translate
   it at that boundary into a CRED error. Otherwise `internal/store` becomes an
   API commitment to pgx, and the layering in section 5 is defeated through the
   error channel — which is the least visible way to defeat it.
4. **Match `sql.ErrNoRows`, not `pgx.ErrNoRows`,** at the boundary. The proxy
   makes both work; the former keeps the driver out of the domain layer.
5. **Enable `errorlint`** (in the config above). It catches `%v` where `%w` was
   meant and `==` comparisons on errors.
6. **Do not enable `err113`.** River's assessment is correct: demanding every
   error be a package-level variable is obnoxious and produces sentinel sprawl
   for errors nobody matches.
7. **Do not use `github.com/pkg/errors`.** Dormant since January 2020; stdlib
   `errors` has covered its surface since Go 1.20's `errors.Join`.

**Library versus binary:** the distinction mostly collapses here, because
everything is under `internal/` and the public API is the MCP tool surface. The
one place it survives is the MCP boundary itself — errors crossing into a tool
response must be scrubbed, because testing-strategy adversarial case 10 requires
that *"errors never echo restricted text."* That is a scrubbing function at
`internal/mcpsrv`, not an error-wrapping convention.

---

## 4. Testing

### The testify question, honestly

The brief's premise was that serious projects deliberately avoid testify. The
data does not support it [REPO, `go.mod` of each]:

| Repo | testify | go-cmp |
|---|---|---|
| riverqueue/river | **yes**, v1.11.1 | no |
| pressly/goose | **yes**, v1.11.1 | no |
| jackc/pgx | **yes**, v1.11.1 | no |
| modelcontextprotocol/go-sdk | no | yes v0.7.0 |
| charmbracelet/bubbletea | no | no |
| golangci/golangci-lint | **yes**, v1.11.1 | indirect |
| grafana/loki | **yes**, v1.11.1 | yes |

Five of seven, including all three CRED data dependencies, on an identical
version.

The commonly cited authority is the **Google** Go Style Guide. Verbatim [FETCH,
`google.github.io/styleguide/go/decisions#assertion-libraries`]:

> "Do not create 'assertion libraries' as helpers for testing."

Two caveats that are routinely dropped when this is quoted: it says *do not
**create*** — its target is writing your own — and **it never mentions testify
by name**. It is a Google style guide, not the Go project's guidance. Anyone
citing it as "Go says don't use testify" is overreading it.

The correlation in the table is real but narrower than claimed: the two
abstainers are the Google-affiliated projects. The mature position is visible in
golangci-lint, which uses testify **and** depends on
`github.com/Antonboom/testifylint` to enforce correct usage. [REPO]

**Recommendation: use testify's `require` only, plus `go-cmp` for structural
diffs, with `testifylint` enabled.** [JUDGEMENT]

Reasoning, and the trade-off: the real hazard with testify is not the dependency
— it is `assert` versus `require`. `assert` continues after a failed assertion,
so a nil-pointer dereference on the next line replaces a clear failure message
with a panic. `require` stops. Restricting to `require` removes the actual
failure mode, and `testifylint` mechanically enforces the choice. The cost is a
dependency in `go.mod` that CRED's three core dependencies already carry
transitively — so the marginal supply-chain cost is approximately zero.

For comparing structs — claims, evidence sets, temporal intervals — use
`cmp.Diff`, not `require.Equal`. A diff of a 12-field claim is readable; an
equality failure dump is not.

### What Go 1.25 changed

[FETCH, `go.dev/doc/go1.24` and `go.dev/doc/go1.25`]

- Go 1.24: `T.Context`, `T.Chdir`, `B.Loop`; `testing/synctest` experimental.
- Go 1.25: **`testing/synctest` graduated to general availability.**
  `synctest.Test` runs a test in a bubble with a **virtualised clock** that
  advances instantly when all bubble goroutines block. The Go 1.24
  `GOEXPERIMENT` form **is removed in Go 1.26.**

**This matters more for CRED than the testify question does.** [JUDGEMENT] CRED
has retry/backoff logic (River), TTL expiry (adversarial case 5), and
bi-temporal invariants tested under *"an injected clock"* (testing-strategy
invariant 8). `synctest` replaces the clock-injection interface those tests
would otherwise require, and eliminates `time.Sleep` from the suite — which is
the single largest source of both wall-clock waste and flakiness in a Go test
suite. Directly relevant to the 2–5 minute budget.

Current Go is **1.26.5**; **1.24 went EOL 2026-02-11**. [FETCH, endoflife.date]
So `synctest` is unconditionally available.

### Table-driven tests and golden files

Table-driven is universal in Go and needs no defence. The one convention worth
stating: **use `t.Run(tc.name, ...)` subtests**, because `synctest` and golden
files both key off `t.Name()`.

For golden files, bubbletea's pattern is the cleanest template [REPO,
`charmbracelet/x/exp/golden`]:

```go
var update = flag.Bool("update", false, "update .golden files")
// golden path: filepath.Join("testdata", tb.Name()+".golden")
```

Subtest names become filenames automatically. golangci-lint's alternative —
hand-edited golden files with no `-update` flag in the printers package — is
worse and should not be copied. [REPO]

**Recommendation: adopt the bubbletea pattern**, own the ~30 lines rather than
depending on `charmbracelet/x`. [JUDGEMENT] CRED's golden candidates are the
first-run startup log (`packaging-and-first-run.md` treats it as product
surface), `cred doctor` output, and MCP tool JSON schemas.

### Postgres in CI: reconciling with the testing strategy

`testing-strategy.md` already decided: **testcontainers with
`pgvector/pgvector:pg17` pinned, one container, N template-cloned databases,
`withReuse()` locally, Ryuk disabled and reuse off in CI.** That decision stands
and is not reopened here.

Worth recording that **none of the six surveyed repos uses testcontainers-go**
[REPO]:

- **river** — GitHub Actions service container, `image: postgres:${{
  matrix.postgres-version }}`.
- **pgx** — a bash script that apt-installs Postgres onto the runner, because
  it must control `pg_hba.conf`, TLS certs, and unix sockets.
- **goose** — `ory/dockertest` from inside Go test code; the CI YAML contains
  no Postgres at all. Notably, dockertest lives in a **nested
  `internal/testing/go.mod`**, keeping the root module dependency-light.

**The survey does not overturn the decision**, because the reason for
testcontainers is specific to CRED and absent from all three: CRED needs
`CREATE EXTENSION vector` and template-database cloning, and a service container
gives no clean hook for template setup. The pgvector image requirement is what
forces it.

**One convention worth stealing from goose: put test-only heavy dependencies in
a nested module.** [JUDGEMENT] `internal/testing/go.mod` carrying testcontainers
keeps `docker` and `moby` out of the root `go.mod` that users see and that the
supply-chain attestation covers. The cost is a second module to keep in sync —
River runs a `submodule_check` job asserting all `go`/`toolchain` directives
match [REPO], which is the mitigation.

### Skipping explicitly rather than passing vacuously

This is the brief's sharpest testing question, and testcontainers answers it
directly. [REPO, `testcontainers-go/testing.go`]:

```go
func SkipIfProviderIsNotHealthy(t *testing.T) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Recovered from panic: %v. Docker is not running...", r)
		}
	}()
	ctx := context.Background()
	provider, err := ProviderDocker.GetProvider()
	if err != nil {
		t.Skipf("Docker is not running...: %s", err)
	}
	err = provider.Health(ctx)
	if err != nil {
		t.Skipf("Docker is not running...: %s", err)
	}
}
```

**Recommendation** [JUDGEMENT]: use `SkipIfProviderIsNotHealthy` in
`TestMain`, **but do not rely on skip alone.** A skipped test and a passing test
are the same green checkmark in CI, which is exactly the vacuous-pass failure
the brief warns about. Pair it with an environment assertion:

```go
func TestMain(m *testing.M) {
	if os.Getenv("CRED_REQUIRE_DB") == "1" {
		// In CI this is set. Docker absence is a hard failure, never a skip.
		mustHaveDocker()
	}
	os.Exit(m.Run())
}
```

The rule: **skipping is a local-developer affordance; in CI it is a failure.**
Set `CRED_REQUIRE_DB=1` in the workflow. Without this, a broken Docker setup in
CI produces a green build with zero integration coverage, silently — and silent
degradation is the failure mode this project has repeatedly identified as its
worst enemy (pgvector recall, LLM structured output, RLS, the `nm` CGO guard).

Additionally, **assert the count**: have the integration suite emit the number
of tests it ran and fail if it is zero. A skip that is supposed to be impossible
should be loud when it happens.

---

## 5. Database access

### The decisive finding on sqlc and pgvector

sqlc has **native pgvector support**, and it is real, not a workaround. From the
changelog [REPO, `docs/reference/changelog.md:635`]:

> "If you're using pgvector, say goodbye to custom overrides! sqlc now generates
> code using pgvector-go as long as you're using `pgx`."

Confirmed in source [REPO,
`internal/codegen/golang/postgresql_type.go:550`]:

```go
	case "vector":
		if driver == opts.SQLDriverPGXV5 {
			if emitPointersForNull {
				return "*pgvector.Vector"
			} else {
				return "pgvector.Vector"
			}
		}
```

**But — and this is decisive for CRED — the mapping covers `vector` only.**
`grep -n -i "halfvec\|sparsevec"` against that file returns **no match**, while
`pgvector-go` itself does export `NewHalfVector` and `NewSparseVector`. [RAN,
REPO]

CRED's storage decision in `tech-decisions.md` is **`halfvec`**, not `vector`:
*"stored as `halfvec` truncated to 768 or 1024 for the indexed path."* So sqlc's
headline pgvector support does not cover CRED's actual column type, and would
need a manual `overrides` entry mapping `db_type: "halfvec"` to
`pgvector.HalfVector`.

That is surmountable. The harder problem is the schema shape.
`tech-decisions.md` commits to a parent table with a column of **unspecified
`vector` type**, partitioned by `model_id`, with per-partition
dimension-specific **expression indexes** on `(emb::vector(384))`. sqlc analyses
a static schema; a deliberately-untyped partitioned column with casted
expression indexes is squarely the case its type inference is weakest on.
**UNVERIFIED** — I did not run sqlc against that schema, and doing so is the
check that would settle it.

### Recommendation: split the boundary

[JUDGEMENT, grounded in the verified sqlc limitation above]

**Do not adopt an ORM.** No surveyed project uses one, and CRED's queries are
the product — RRF fusion, ACL intersection, bi-temporal slicing. An ORM
abstracts away exactly the layer that must be explicit.

**Do not adopt sqlc as the single access path either.** The verified `halfvec`
gap plus the untyped-partitioned-column risk means the vector path would fight
the tool, and fighting a codegen tool as a solo maintainer is a bad trade.

**Use pgx directly, with sqlc deferred as an option, not a commitment.**
Concretely: `internal/store/pg/` hand-writes queries against `pgxpool`. Register
pgvector types in `AfterConnect`, which is pgx's documented hook for exactly
this [C7, pgx wiki]:

```go
cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
	return pgxvec.RegisterTypes(ctx, conn)
}
```

The `internal/store/gen/` directory in the proposed tree is a placeholder. If
the schema stabilises and sqlc handles it, generated code lands there and
coexists. Keeping the directory named but empty costs nothing and records the
option.

### Making the temporal/ACL law structurally enforceable

This is the part of the brief that matters most, so it gets a mechanism rather
than a principle.

The instruction from `testing-strategy.md`: *"push the temporal and ACL algebra
out of SQL and into testable code; let Postgres store and filter, not decide."*

Three layers of enforcement, weakest to strongest [JUDGEMENT]:

1. **Package boundary.** `internal/temporal` and `internal/acl` contain pure
   functions over domain types. They have no `internal/store` import and no
   database handle in any signature.
2. **`depguard`, as configured in section 2.** Importing `pgx` or
   `database/sql` from `internal/temporal` or `internal/acl` **fails the
   build**. This is the step that converts the instruction from a comment into a
   check. It is also why `depguard` is in the linter list despite appearing in
   only 4 of 8 surveyed configs — it is CRED-specific, and that is labelled in
   the config.
3. **Signature shape.** `internal/store` returns *rows*; it does not return
   *decisions*. Concretely: no method named `GetVisibleClaims(principal)`. The
   store offers `LoadClaims(ids)` and `LoadEvidence(claimIDs)`; `internal/acl`
   computes `claim.acl ⊆ ⋂(evidence_i.acl)` in Go. **L5 is never expressed as a
   SQL predicate.**

The trade-off, named honestly: this is slower. Filtering in Go means
transferring rows Postgres could have discarded. For CRED's stated scale — one
instance per organization, tenant count 1 — that is affordable, and
`tech-decisions.md` already shows the alternative is worse: pgvector filtering
under ACL selectivity silently returns 4 results instead of 40. **Deciding in
SQL is not merely harder to test; it is the known silent-failure path.** The
performance cost buys correctness and testability at once.

There is one genuine tension to record. RLS is discussed at length in
`tech-decisions.md` as a defence-in-depth layer, and RLS *is* Postgres deciding.
These are not contradictory if RLS is treated as a **backstop that must never be
the only check** — the Go-side intersection is the decision, RLS catches
programmer error. But it means the RLS coverage test
(`tech-decisions.md`: *"a migration-time invariant that must be tested"*) is
asserting a redundancy, not the primary guarantee, and should say so.

---

## 6. Concurrency and context

[C7 for all dependency behaviour; JUDGEMENT for the conventions]

**What the dependencies expect:**

- **River** — `Work(ctx, job)` receives a cancellable context.
  `Client.Stop(ctx)` waits for active jobs; `StopAndCancel(ctx)` cancels them
  immediately. River's docs state jobs must respect cancellation, and it
  *"detects and logs jobs that become stuck and do not respond to cancellation
  signals."* [C7]
- **OpenTelemetry** — from its CONTRIBUTING: *"OpenTelemetry API
  implementations must ignore context cancellation when recording values like
  spans or measurements… Pipeline shutdown is handled via `Shutdown` methods,
  not user-provided context."* [C7] So a cancelled context does not stop spans
  from recording, and shutdown must be an explicit call with its own timeout.
- **pgx** — `pgxpool` takes `ctx` on every operation; `AfterConnect` and
  `PrepareConn` hooks run per connection. [C7]

**Recommendations:**

1. **`ctx` is the first parameter of every function that does I/O, and is never
   stored in a struct.** Standard Go, universally observed.
2. **The goroutine that starts a goroutine owns its lifetime.** No package
   spawns a background goroutine at import time or in a constructor without
   returning a stop function. Go's failure mode here is silence — a leaked
   goroutine holding a database connection is invisible until the pool exhausts.
3. **Shutdown is ordered, and the order is not arbitrary:** stop accepting MCP
   requests → `river.Client.Stop(ctx)` with a bounded timeout → close the pgx
   pool → **OTel `Shutdown` last**, because the shutdown path itself emits
   spans. Getting this backwards loses exactly the telemetry describing the
   shutdown.
4. **Use a separate, non-cancelled context for shutdown work.** The signal
   context is already cancelled by the time shutdown runs; passing it to `Stop`
   makes graceful shutdown instantly ungraceful. This is the most common Go
   shutdown bug.
5. **Prefer `JobSnooze(0)` over a context-cancelled error on shutdown**, per
   River's own docstring — it does not consume an attempt. [C7]
6. **Use `testing/synctest` to test all of the above**, per section 4.

---

## 7. Logging and observability

`tech-worker-ops-packaging.md` already decided this, and the decision is sound
and is not reopened: **plain JSON to stdout via `log/slog`; do not use the OTel
logs bridge**, because *"an in-process logs SDK with a batching exporter is a
mechanism that can silently drop records"* and CRED's core promise is never
silently dropping data.

**Is anything beyond `slog` still justified? No.** [JUDGEMENT] `zap` and
`zerolog` predate `slog` and were justified by allocation counts under load that
CRED does not approach — this is a memory service, not an ad exchange. `slog` is
stdlib, zero-dependency, JSON-native, and has an official OTel bridge if the
decision is ever revisited. The one thing to add is a `slog.Handler` wrapper
that injects `trace_id` and `span_id`.

Conventions that follow, all consistent with the existing spike:

1. **Correlation fields are `trace_id` and `span_id`, hex-encoded W3C**
   (32-char trace, 16-char span), snake_case, injected from the active span
   context by a handler wrapper — not by every call site. That format is what
   Tempo, Loki, Jaeger, and Honeycomb auto-link on.
2. **Every OTel attribute name is a constant in `internal/obs`.**
   Non-negotiable: `gen_ai.*` has no stable release, 127 open issues, and
   already renamed `gen_ai.system` to `gen_ai.provider.name`. A spec bump must
   be a one-file diff.
3. **Content capture defaults OFF.** Memory records are user IP. The spec marks
   `gen_ai.input.messages` Opt-In; CRED must too.
4. **Never log claim or evidence body text at INFO.** This is a rule an agent
   will violate by default, because "log the thing you are debugging" is the
   normal instinct. It belongs in the rules file, not only here.
5. **`slog` levels follow the existing spike:** ERROR includes every
   `cred.ingest.dropped` increment; INFO includes every mutation of a record.
6. **The audit trail is written transactionally to Postgres; the log is a
   derived stream.** Already recorded, restated because it is the reason the
   logging decision is not merely stylistic.

The accepted cost, already on the record: Go has no auto-instrumentation, so
every span is hand-wired and an unwrapped library emits nothing with no error.
Full OTel takes the binary from 11.29 MB / 16 modules to 27.73 MB / 72 modules.

---

## 8. CI

Budget from `testing-strategy.md`: **2–5 minutes per commit**, and *"above
roughly ten minutes a solo maintainer skips the suite."*

### What the surveyed projects run

[REPO, all]

| Repo | PR jobs | Go matrix | Postgres |
|---|---|---|---|
| riverqueue/river | ~12 | 1.26 × PG 14–18, + 1.25 | service container |
| pressly/goose | 4 | oldstable, stable | dockertest, in-test |
| jackc/pgx | 14 | 1.25, 1.26 × 6 DBs | apt-install script |
| mcp/go-sdk | 4+ | 1.25, 1.26 | n/a |
| bubbletea | ~12 | stable + go.mod | n/a |
| golangci-lint | 8+ | 1.25, 1.26 | n/a |

Consistent patterns worth adopting:

- **Go matrix is current + previous only**, never more. River's comment: *"the
  Go version previous to current is the only other officially supported Go
  version."* For CRED that is **1.25 and 1.26**.
- **Nobody adds an explicit module-cache step**; all rely on `actions/setup-go`
  built-in caching.
- **Race detector as a separate single job**, not across the matrix — go-sdk's
  pattern, and the cheapest way to get race coverage. [REPO]
- **`GOTOOLCHAIN: local`** (River) so the matrix Go version is really what
  runs. Without it Go silently downloads a newer toolchain and the matrix is a
  lie.
- **go-sdk pins every third-party action to a full commit SHA** with a `# vX`
  comment, and sets `permissions: contents: read`. It is the only repo doing
  this consistently, and it aligns with the supply-chain minimum already
  recorded in `packaging-and-first-run.md`.

### The CGO guard, refined

D-008 records that `go tool nm | grep cgo` is broken and that the working guard
is `go list -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' -deps ./...`. I ran both
sides of this to confirm the guard, and **found a way the recommended guard can
also pass while broken.** [RAN, Go 1.26.0 darwin/arm64]

Setup: a module whose dependency package contains only a cgo file.

```
--- CGO_ENABLED=1 go list guard ---
runtime/cgo
cgotest/dep
--- CGO_ENABLED=0 go list guard ---
package cgotest
	imports cgotest/dep: build constraints exclude all Go files in .../dep
```

Under `CGO_ENABLED=0` the guard emits **nothing on stdout** — the error goes to
stderr. Exit codes:

```
CGO_ENABLED=0 go list  → exit 1
CGO_ENABLED=0 go build → exit 1
CGO_ENABLED=1 go build → exit 0
```

So a guard written the natural way:

```bash
out=$(CGO_ENABLED=0 go list -f '...' -deps ./... 2>/dev/null)
[ -z "$out" ] && echo "clean"       # ← reports CLEAN on a broken build
```

**reports clean while the build is broken** — the same failure shape as the `nm`
check D-008 already rejected, arrived at from a different direction. Verified by
execution:

```
stdout was: []
>>> naive guard would report CLEAN despite broken build
```

**The correct guard is two assertions, not one** [JUDGEMENT, from the run]:

```bash
set -euo pipefail
# 1. Enumerate anything that WOULD use cgo. Must run with CGO_ENABLED=1;
#    under =0 the build constraints hide cgo files and the check is vacuous.
CGO_ENABLED=1 go list -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' -deps ./... \
  | grep -v '^runtime/cgo$' \
  | tee /dev/stderr | wc -l | grep -qx 0

# 2. Assert the real shipping build succeeds with cgo off.
#    Never swallow stderr; never ignore the exit code.
CGO_ENABLED=0 go build ./...
```

Assertion 1 catches a dependency that pulls in cgo. Assertion 2 catches the
case where a cgo-only package breaks the static build. **Neither alone is
sufficient**, and assertion 1 run under `CGO_ENABLED=0` — the intuitive
reading of "run against the real shipping build command" — is vacuous.

This refines rather than contradicts D-008. Worth folding back into that entry.

### Proposed CI workflow

[JUDGEMENT, assembled from the patterns above; budget-driven]

```yaml
name: ci
on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

env:
  GOTOOLCHAIN: local

jobs:
  # ~30s. Fails fast on the cheapest signal.
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with: { persist-credentials: false }
      - uses: actions/setup-go@v6
        with: { go-version-file: go.mod }
      - run: go mod tidy && git diff --exit-code
      - uses: golangci/golangci-lint-action@v9
        with: { version: v2.12.2 }

  # ~20s. The D-008 guard, in its refined two-assertion form.
  cgo-guard:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with: { persist-credentials: false }
      - uses: actions/setup-go@v6
        with: { go-version-file: go.mod }
      - name: No cgo in the dependency tree
        run: |
          set -euo pipefail
          found=$(CGO_ENABLED=1 go list \
            -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' -deps ./... \
            | grep -v '^runtime/cgo$' || true)
          if [ -n "$found" ]; then
            echo "cgo packages in dependency tree:"; echo "$found"; exit 1
          fi
      - name: Static build succeeds with cgo disabled
        run: CGO_ENABLED=0 go build ./...

  # ~60-90s. Unit + property. No database.
  test-unit:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go: ['1.25', '1.26']
    steps:
      - uses: actions/checkout@v5
        with: { persist-credentials: false }
      - uses: actions/setup-go@v6
        with: { go-version: ${{ matrix.go }} }
      - run: go test -shuffle=on ./...

  # ~1-3 min. The budget's largest slice.
  test-integration:
    runs-on: ubuntu-latest
    env:
      CRED_REQUIRE_DB: '1'          # Docker absence is a failure, never a skip
      TESTCONTAINERS_RYUK_DISABLED: 'true'
    steps:
      - uses: actions/checkout@v5
        with: { persist-credentials: false }
      - uses: actions/setup-go@v6
        with: { go-version-file: go.mod }
      - run: go test -tags=integration -shuffle=on ./...

  # ~90s, single job — not across the matrix.
  test-race:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with: { persist-credentials: false }
      - uses: actions/setup-go@v6
        with: { go-version-file: go.mod }
      - run: go test -race ./...
```

Six jobs, all parallel, wall-clock set by the slowest — the integration job at
1–3 minutes. **Within the 2–5 minute budget**, with the caveat that this is an
estimate from job composition, not a measurement. There is no code to time yet.
UNVERIFIED until the first real suite runs.

Deliberately **not** on every PR: `govulncheck` and migration linting
(`squawk`, per `tech-decisions.md`) belong on a nightly or a
migrations-path-filtered trigger; multi-arch build verification belongs on the
release workflow, where `packaging-and-first-run.md` already puts it. Adding
them per-PR is how a 3-minute pipeline becomes 11 minutes and gets skipped.

---

## 9. Contributor-facing files

Presence across five small-team repos [REPO, `gh api contents`]:

| File | river | goose | go-sdk | bubbletea | pgx |
|---|---|---|---|---|---|
| CONTRIBUTING.md | — | — | ✅ | org | ✅ |
| SECURITY.md | — | — | ✅ | org | — |
| CODE_OF_CONDUCT.md | — | — | — | — | — |
| ISSUE_TEMPLATE/ | — | — | ✅ | ✅ | ✅ |
| PR template | — | — | ✅ | org | — |
| CODEOWNERS | — | — | — | — | — |
| .editorconfig | — | — | — | — | — |
| dependabot | ✅ | ✅ | ✅ | ✅ | — |

"org" = absent from the repo, present in `charmbracelet/.github` org fallback. I
checked for the same fallback at `riverqueue/.github`, `pressly/.github`, and
`jackc/.github` — **all 404**, so those three genuinely have nothing. [REPO]

**Zero of five have a CODE_OF_CONDUCT.md. Zero have CODEOWNERS. Zero have
.editorconfig.** River and goose — both well-regarded, both company-adjacent —
have *nothing* contributor-facing beyond dependabot and workflows.

### Recommendation

D-013 found the parity gap is roughly three engineers. Ceremony competes with
that gap directly.

**Adopt now:**

| File | Why |
|---|---|
| `.github/dependabot.yml` | 4/5 have it. Zero ongoing effort, real supply-chain value, and CRED already commits to attestations for the CRA path. |
| `SECURITY.md` | **Not for parity — because of L8.** CRED's threat model includes stored prompt injection reaching another agent, which testing-strategy calls *"the highest-severity case."* A product making a security claim needs a disclosure address. Ten lines. |

**Defer until there is a second contributor:**

| File | Why it is premature |
|---|---|
| `CONTRIBUTING.md` | River and goose ship without one. A contributing guide at zero contributors documents a process nobody has run. Write it when the first PR arrives and you learn what the reviewer actually needs. |
| Issue templates | Real value once issue volume creates triage load. At zero users, a template is friction on the rare early report you most want. |
| PR template | The PR author is the maintainer. A template you fill in for yourself is a form you will start leaving blank, which trains you to ignore templates. |

**Skip, and say so plainly:**

| File | Why |
|---|---|
| `CODE_OF_CONDUCT.md` | Zero of five have one. It governs a community that does not exist. Add it the week a community does. |
| `CODEOWNERS` | Zero of five. It routes review to owners. There is one owner. It is pure ceremony at n=1. |
| `.editorconfig` | Zero of five, and the reason is Go-specific: **`gofmt` already decides formatting, and CI enforces it.** `.editorconfig` would encode a weaker version of a rule already mechanically enforced. |
| Renovate | Dependabot is native, zero-config, and already what 4/5 use. Renovate is more powerful and more configuration to maintain. |

**Judgement call, recorded as such:** `CONTRIBUTING.md` is the closest to the
line. The argument for it is that CRED's adoption strategy makes contributors
strategic, so the door should look open. The argument against — which I take —
is that a `CONTRIBUTING.md` written before any contribution describes a guessed
process, and a wrong process is worse than an absent one because it must be
unlearned. **A short "Contributing" section in the README is the right size at
this stage**, and it upgrades to a file when it outgrows the README.

---

## 10. AI-agent-facing conventions

### AGENTS.md versus CLAUDE.md

The prior claim from D-007's operating parameters: AGENTS.md 154,496 files vs
CLAUDE.md 51,100.

**Directionally VERIFIED. The exact numbers are NOT reproduced.** [REPO, `gh api
search/code`, run twice with identical results]

```
filename:AGENTS.md  → 151,168
filename:CLAUDE.md  →  53,132
```

AGENTS.md leads by ~2.85×. Against the prior claim, AGENTS.md is ~3,300 lower
and CLAUDE.md ~2,000 higher. **Do not cite 154,496 / 51,100 as current.**

Caveat that matters more than the digits: GitHub REST code-search `total_count`
is approximate and index-dependent, not a census. The `path:` qualifier on the
REST endpoint returns wildly different output (`path:AGENTS.md` → 130), showing
it does not behave as the web UI does. **The ratio is the durable finding; the
absolutes are not.** The web UI figure could not be obtained — GitHub code
search requires login and WebFetch cannot authenticate. FALSIFIED as a method.

`agents.md` describes itself as *"a simple, open format for guiding coding
agents,"* emerged from Codex/Amp/Jules/Cursor/Factory, and *"now stewarded by
the Agentic AI Foundation under the Linux Foundation."* It claims *"used by over
60k open-source projects."* [FETCH] Note the tension with the 151k file count —
different denominators, and neither validates the other.

**The decisive fact is not the count.** From Claude Code's own documentation,
verbatim [FETCH, `code.claude.com/docs/en/memory`]:

> "Claude Code reads `CLAUDE.md`, not `AGENTS.md`. If your repository already
> uses `AGENTS.md` for other coding agents, create a `CLAUDE.md` that imports it
> so both tools read the same instructions without duplicating them."

**Recommendation: `AGENTS.md` is canonical; `CLAUDE.md` is a one-line import.**
[JUDGEMENT]

```
CLAUDE.md:   @AGENTS.md
```

Reasoning: the content is not Claude-specific — it is repository truth. CRED's
own thesis is that durable context should not be siloed per consumer, and
splitting the same instructions across two vendor-named files is the exact
anti-pattern the product argues against. AGENTS.md is the vendor-neutral name,
under Linux Foundation stewardship, with ~24 tools reading it and a ~2.85×
adoption lead.

The cost, named: `CLAUDE.md` currently *is* the file with the content, so this
is a rename plus a one-line stub, and every existing reference to `CLAUDE.md` in
`docs/` and in `.claude/rules/docs.md`'s `paths:` frontmatter must be updated in
the same change. `.claude/rules/docs.md` lists `CLAUDE.md` in its paths; it must
gain `AGENTS.md` or the documentation rules will stop loading for the file that
matters.

Note River already ships an `AGENTS.md`, and pgx ships a `CLAUDE.md`. [REPO] The
ecosystem has not converged; the recommendation is a judgement, not a consensus.

### What actually helps an agent write correct code here

[JUDGEMENT throughout — this section is design, not survey. I found no
empirical study of agent-facing repository conventions and will not invent one.]

The general principle worth stating: **an agent cannot verify a law it can only
read.** Every convention below is chosen for being checkable.

1. **Design laws live in exactly one place and are referenced, never
   restated.** `CLAUDE.md` already does this correctly for L1–L8: *"Do not
   restate them here from memory — they are maintained in the PRD, and this list
   is a pointer."* That instinct is right and should extend to Go rules.

2. **Prefer a lint rule over a written rule.** The `depguard` block in section
   2 is the model: "temporal and ACL algebra must be pure" is a sentence an
   agent can agree with and then violate; an import that fails the build is not.
   **Any rule that can be expressed as a linter, a test, or a CI assertion
   should be.** The rules file should then say *which check enforces it*, so an
   agent knows the rule is real and knows how to verify compliance without
   asking.

3. **Invariants live next to the code they constrain, as tests.** The 14
   bi-temporal invariants and 20 adversarial cases in `testing-strategy.md`
   should exist as named test functions whose names match the document
   (`TestInvariant09_OrderSensitivityIsExplicit`). An agent asked to change
   temporal logic then discovers the constraint by running the suite rather than
   by having read a document it may not have loaded.

4. **File size: guidance, not a lint rule.** River's disable list is persuasive
   here — `funlen` and `gocyclo` are noise. But there is a real agent-specific
   consideration: a 2,000-line file is a file an agent will edit without having
   read all of, which is how contradictory code gets written. **Recommend ~500
   lines as a soft ceiling, enforced by review and by package boundaries rather
   than by a linter.** A file crossing it is usually two packages.

5. **Every package gets a doc comment stating what it may not do.** For
   `internal/temporal`: *"Pure functions over claim intervals. This package must
   not import a database driver; `depguard` enforces it."* The negative space is
   the part an agent cannot infer from the code that is present — absence is
   invisible.

6. **`//nolint` requires a linter name and a reason** — already in the proposed
   config via `nolintlint`. Without it, `//nolint` is the escape hatch an agent
   reaches for when a check is inconvenient, and it is silent.

7. **The repo should be a credible instance of its own thesis.** CRED argues
   that agents should work from durable, evidence-backed context. The mechanism
   that makes that true here is the one already in `.claude/rules/docs.md`:
   claims carry VERIFIED / UNVERIFIED / FALSIFIED, and **falsified claims stay
   in the document.** Extending that to code means an agent-facing rule this
   document has already exercised three times: when a convention is checked and
   found false — `river-queue/river`, the testify premise, `pkg/errors` being
   archived — the correction is recorded rather than quietly dropped.

---

## What to skip, and why

A solo maintainer's harness fails by being too heavy. Explicit exclusions:

| Skip | Why |
|---|---|
| **`pkg/`** | No official endorsement; the Go layout page never mentions it. 4/10 use it and inconsistently. `internal/` is compiler-enforced; `pkg/` is decoration. |
| **`golang-standards/project-layout`** | Its own README disclaims standard status in bold and calls it *"overkill"* for a project at CRED's stage. |
| **Complexity linters** (`cyclop`, `gocyclo`, `gocognit`, `funlen`, `maintidx`, `mnd`) | River rejects all six with reasons. No correctness signal, constant tax, and they train the maintainer to ignore lint output. |
| **`err113`** | Sentinel sprawl for errors nobody matches. |
| **`ireturn`** | River: *"questionable as is, but also buggy as hell."* |
| **`sqlvet`** | Not abandoned, but no golangci-lint integration and it duplicates what sqlc's compile step does better. |
| **`fieldalignment`** | Struct reordering for padding. Pure churn at CRED's data volumes. |
| **An ORM** | No surveyed project uses one, and CRED's queries *are* the product. |
| **sqlc as the committed access path** | Verified: sqlc maps `vector` but **not `halfvec`**, which is CRED's actual storage type. Keep it as a deferred option. |
| **`zap` / `zerolog`** | `slog` is stdlib and CRED is not allocation-bound. |
| **The OTel logs bridge** | Already decided: a batching exporter can silently drop records, which contradicts the product's core promise. |
| **Dual-emitting OpenInference** | Already decided: doubles attribute payload and picks a spec fight inside the codebase. |
| **CODE_OF_CONDUCT.md** | 0/5 surveyed repos. Governs a community that does not exist. |
| **CODEOWNERS** | 0/5. Routes review to owners. There is one owner. |
| **.editorconfig** | 0/5, and `gofmt` + CI already enforce formatting mechanically. |
| **Renovate** | Dependabot is native and zero-config; 4/5 use it. |
| **CONTRIBUTING.md, issue and PR templates** | Deferred, not permanent. Premature at zero contributors; a guessed process is worse than none. README section for now. |
| **`govulncheck` and `squawk` on every PR** | Nightly or path-filtered. Per-PR is how a 3-minute pipeline becomes 11 and gets skipped. |
| **Multi-arch verification on PR** | Belongs in the release workflow, where the packaging spike already puts it. |
| **`go tool nm \| grep cgo`** | Already falsified in D-008: reports `_cgo_` symbols with CGO both on and off. Passes while broken. |
| **A `Makefile` at this stage** | River and goose use one, but they have many modules. `go test ./...` and a six-job workflow do not need an abstraction layer yet. |

---

## What remains unverified

1. **The 2–5 minute CI budget.** The proposed workflow is estimated from job
   composition, not measured. There is no code to time. **Check:** run it once
   real tests exist and record the wall-clock.

2. **Whether sqlc can handle CRED's actual schema.** VERIFIED that it maps
   `vector` and not `halfvec`. **UNVERIFIED** whether its analyser copes with
   the deliberately-untyped partitioned `vector` column and per-partition
   `(emb::vector(384))` expression indexes from `tech-decisions.md`. **Check:**
   run `sqlc generate` against that schema. This is a half-hour experiment and
   it settles section 5 definitively.

3. **Whether `testing/synctest` composes with River's internal timers.** The
   virtual clock only advances when *all* bubble goroutines block; a River
   client holding a database connection may never fully block. **Check:** write
   one snooze/retry test under `synctest` before committing the suite to it.

4. **Whether `depguard`'s `files:` glob negation (`!**/internal/store/**`)
   behaves as written in v2.12.x.** The pattern is from the documented schema
   but I did not execute golangci-lint against a tree. **Check:** create the two
   packages and confirm the build fails on a deliberate bad import. Do this
   before relying on it, because a depguard rule that silently matches nothing
   is exactly the vacuous-pass failure this document warns about elsewhere.

5. **Whether `pgvector-go` registration works through `pgxpool.AfterConnect`
   for `halfvec` specifically.** The `AfterConnect` hook is VERIFIED as the
   right mechanism [C7], and `pgvector-go` exports `NewHalfVector` [REPO], but I
   did not run the two together.

6. **Race-detector cost.** pgx runs `-parallel=1` because *"parallel testing
   causes Github Actions to kill the runner"* [REPO]. If CRED's race job exceeds
   budget, that constraint is the likely cause and the likely fix.

7. **Whether Cursor genuinely reads AGENTS.md.** `agents.md` lists it; I did
   not confirm against Cursor's own documentation. Treat as the standard's
   claim. It does not change the recommendation, which rests on Claude Code's
   own documented behaviour.

8. **golangci-lint v2.12.2 exact schema details beyond what Context7
   returned.** Context7's indexed snapshot is v2.3.0 while the current release
   is v2.12.2. The `version: "2"` schema is stable across that range, but
   individual linter settings may have shifted. **Check:** `golangci-lint config
   verify` against the proposed file — the tool ships a JSON Schema validator
   for exactly this.

---

## Verdict

Adopt: `cmd`-less `main.go` plus a deep `internal/` tree; a golangci-lint v2
config built from the majority of eight real configs, with `depguard` as the
mechanism that makes the temporal/ACL purity law a build failure; stdlib errors
with `%w` as an explicit API commitment and pgx errors translated at the store
boundary; testify `require` under `testifylint` alongside `go-cmp`, with
`synctest` for anything time-dependent; pgx directly rather than sqlc, on the
verified `halfvec` gap; `slog` alone; a six-job CI workflow with the CGO guard
in its two-assertion form; and `AGENTS.md` as canonical with `CLAUDE.md`
importing it.

Skip: `pkg/`, complexity linters, an ORM, a code of conduct, CODEOWNERS,
`.editorconfig`, and every contributor-facing template until a second
contributor exists.

**What would change this verdict:** sqlc handling the partitioned `halfvec`
schema cleanly would move section 5 toward generated code, which is the single
largest open question here. A measured CI run above five minutes would force
integration tests off the per-commit path and into a merge gate — which would in
turn weaken the "skip is a failure in CI" rule in section 4, since a suite that
does not run per-commit cannot fail per-commit.
