# Semantic anchoring and the tree-sitter CGO gate

L3 anchors evidence with a fingerprint ladder, not a byte hash of a line range
(PRD section 4). Tiers 1 and 2 for **code** are a tree-sitter symbol path and a
normalized hash of the enclosing AST node. Tree-sitter is a C library, and the
whole product is built on a non-negotiable constraint:

> **`CGO_ENABLED=0` everywhere** (D-008, `.claude/rules/go.md` §3). It buys
> cross-compilation without QEMU, distroless-static bases, and musl
> compatibility — losing it loses all three at once.

So before writing any code anchorer, one question gates the slice, and it is the
same shape as the two that gated the tokenizer and the reranker:

> **Does a usable tree-sitter binding exist for Go with `CGO_ENABLED=0`?**

This spike answers it empirically. Everything below was executed on the machine
in [Environment](#environment); no result is expected rather than run.

---

## Decisions

| Area | Decision |
|---|---|
| The CGO gate | **Cleared.** A pure-Go, CGO-free tree-sitter exists, parses Go, and anchors correctly |
| Ladder for the current corpus | **Ship it.** Markdown/text anchorer, fully implemented in `internal/anchor`, wired into seed + write |
| Code anchorer | **Built as a seam, not shipped.** Gated — but on dependency + grammar-fidelity, **not** on CGO |
| The three C bindings | **Rejected.** smacker, official, and go-sitter-forest are all CGO and all fail the `CGO_ENABLED=0` build |
| The WASM binding | **Rejected for now.** `malivvan/tree-sitter` is CGO-free but ships no Go grammar |

---

## The question and why it gates the slice

L3 says a byte hash of a line range "fails in both directions": a formatting
commit erases a module's memory, and an insertion above silently re-anchors a
claim onto different code, which then validates. Evidence today carries
`source_path`, `line_start`, `line_end`, `content_sha256` — **tier 4 only**. The
point of L3 is to not ship that.

For **Markdown**, which is the entire corpus today (D-016 seeds documentation),
tiers 1–2 need no tree-sitter: tier 1 is the heading path, tier 2 is the
normalized hash of the enclosing section. For **code**, tiers 1–2 are the
tree-sitter symbol path and the normalized AST-node hash — and that needs a
parser. If the only Go parser needs CGO, dragging it into the build forfeits the
packaging strategy the whole language choice rests on. Hence the gate.

---

## Method

A scratch module outside the repo, one candidate per build tag, each checked two
ways — the [D-008 two-assertion CGO guard](../decision-log.md#d-008):

```sh
# assertion 1: enumerate cgo packages in the build graph
CGO_ENABLED=1 go list -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' -deps .
# assertion 2: the static build must actually succeed
CGO_ENABLED=0 go build -tags <candidate> -o /dev/null .
```

`go list` under `CGO_ENABLED=0` alone is vacuous (it writes to stderr and emits
nothing on stdout when a cgo-only dependency is present — D-008), so assertion 2
is the one that cannot be faked. For the pure-Go candidate, a real program parses
a Go file and computes a symbol path and a normalized-node hash, then the four
mutations L3 cares about are applied to the same source.

Candidates, drawn from the current library landscape:

| Module | Fetched version | Mechanism |
|---|---|---|
| `github.com/smacker/go-tree-sitter` | v0.0.0-20240827 | C via cgo |
| `github.com/tree-sitter/go-tree-sitter` (official) | v0.25.0 | C via cgo (`mattn/go-pointer`) |
| `github.com/alexaandru/go-sitter-forest` + `go-tree-sitter-bare` | v1.9.x / v1.10.0 | C via cgo |
| `github.com/malivvan/tree-sitter` | v0.x (wraps ts v0.24.7) | **WASM via wazero**, CGO-free |
| `codeberg.org/hum3/gotreesitter` | **v0.6.7** | **Pure Go**, GLR reimplementation |

---

## Results: the three C bindings are CGO (FALSIFIED as a pure-Go path)

**VERIFIED.** All three fail the `CGO_ENABLED=0` build.

```
smacker  — CGO_ENABLED=1 cgo packages:
  runtime/cgo, github.com/smacker/go-tree-sitter, .../golang
  CGO_ENABLED=0 build: FAIL — build constraints exclude all Go files in .../golang;
                       undefined: Node (the cgo-gated files that define the API)

official — CGO_ENABLED=1 cgo packages:
  runtime/cgo, github.com/mattn/go-pointer,
  github.com/tree-sitter/go-tree-sitter, .../tree-sitter-go/bindings/go
  CGO_ENABLED=0 build: FAIL — build constraints exclude all Go files in the grammar binding

go-sitter-forest / go-tree-sitter-bare —
  import "C" in language.go, parser.go, node.go, query.go, tree.go, sitter.go, ...
  vendors error_costs.h, ptypes.h, subtree.h, ...
  CGO_ENABLED=0 build: FAIL — build constraints exclude all Go files in every grammar package
```

These are the bindings the PRD's L3 table assumed. On the CGO constraint, none of
them can ship in CRED's default build.

## Results: the WASM binding is CGO-free but has no Go grammar (rejected for now)

**VERIFIED.** `malivvan/tree-sitter` wraps a WASM build of tree-sitter and runs
it with `wazero` (pure Go), so it is genuinely CGO-free. But its grammar surface
is `LanguageC` and `LanguageCpp` only — **no Go grammar** — and it is pre-release
(v0.0.1). It cannot anchor Go code today. Worth revisiting if it grows a Go
grammar, because the WASM path reuses upstream tree-sitter's *own* parse tables
rather than reimplementing the parser.

## Results: pure-Go tree-sitter builds, parses, and anchors CGO-free (VERIFIED)

**VERIFIED — the gate is cleared.** `codeberg.org/hum3/gotreesitter` v0.6.7 (a
from-scratch GLR reimplementation that loads tree-sitter's parse-table format,
with 206 grammars embedded) plus its `grammars` subpackage:

```
CGO_ENABLED=1 cgo packages in the gts build graph:
  (empty — none)
CGO_ENABLED=0 go build -tags gts: BUILD OK
file: Mach-O 64-bit executable arm64          # and cross-compiles; no C toolchain
```

It parses Go into a correct AST:

```
(source_file (package_clause (package_identifier))
  (function_declaration (identifier) (parameter_list) (type_identifier)
    (block (statement_list (return_statement (expression_list (int_literal)))))) ...)
```

And the ladder behaves **exactly** as L3 requires. A claim's evidence is the body
of `Hello`; the anchor is the tier-1 symbol path plus the tier-2 normalized hash
of the enclosing `function_declaration`:

```
[base]            tier1 "Hello"  tier2 497c3f5a11895367
[reformatted]     tier1 "Hello"  tier2 497c3f5a11895367   <- formatting churn: 1+2 hold
[inserted-above]  tier1 "Hello"  tier2 497c3f5a11895367   <- a function added above: 1+2 hold
[semantic-edit]   tier1 "Hello"  tier2 0076d44b1a533686   <- body changed: tier2 flips
```

Formatting churn and an insertion above leave tiers 1 and 2 untouched; only the
semantic edit moves tier 2. This is the whole point of L3, and the pure-Go parser
delivers it.

### The cost, named

The gate is cleared, but adopting the parser is not free:

- **~10 MB of binary.** The stripped `gts` program is **19 MB** against CRED's
  current ~9 MB. Importing the Go grammar pulls the `grammars` package, which
  embeds **all 206** grammars as compressed blobs even though CRED uses one. That
  is a size cost, not a CGO cost, and it lands against the ~2 MiB-base packaging
  ethos.
- **Zero third-party transitive modules** — the module is self-contained (only
  `codeberg.org/hum3/gotreesitter` in the build graph), which is a genuine point
  in its favour against the accepted dependency set.
- **Maturity.** v0.6.x, single maintainer, first published mid-2026 (Show HN),
  carrying headline benchmark claims ("~158x faster than C on incremental edits",
  "~41,800x on no-edit reparses") of exactly the seductive shape D-010 caught
  itself believing once. It reimplements the GLR parser rather than reusing
  upstream's engine, so its **grammar fidelity to upstream tree-sitter across
  real Go code is unverified** beyond this smoke test.

---

## What this changes, and what it does not

**The CGO blocker is gone.** The PRD's L3 table and the tech-decisions
code-anchoring row both assumed tree-sitter meant C, and the open worry was that
code anchoring would force CGO into the build. It does not have to: a pure-Go,
CGO-free path exists and anchors correctly.

**Code anchoring still does not ship in this slice — for a smaller, different
reason.** Two facts, neither about CGO:

1. **Nothing produces code evidence yet.** The corpus is Markdown (D-016). Every
   `Evidence` row is `document` or `attestation`; there is no `code` producer to
   anchor. Shipping a 10 MB parser dependency for a code path with zero callers
   is weight for nothing.
2. **A GLR reimplementation must be diffed before it is trusted — the D-008
   tokenizer lesson.** A hand-written WordPiece tokenizer scored 99.67% and was
   silently wrong on non-English text; the rule that came out of it is *write it,
   or diff it against the reference — never adopt a reimplementation on faith.* A
   from-scratch Go tree-sitter loading extracted parse tables is the same class of
   risk: "nearly right is worthless" for an anchor, because a confidently wrong
   claim is worse than no claim (L3). A four-snippet smoke test is not the
   fidelity verification D-008 demands.

So `internal/anchor` ships the **pluggable seam** — `anchor.For(kind)` — and the
markdown/text anchorer behind it, fully implemented and wired into seed and
write. The code anchorer drops in at `SourceCode` with no caller change, exactly
as the tokenizer's ONNX escape hatch was designed to. It is **gated on a code
producer existing and a grammar-fidelity diff against upstream tree-sitter**, not
on CGO.

---

## What is unverified, and why

- **UNVERIFIED: `gotreesitter`'s grammar fidelity to upstream tree-sitter.** The
  gate answers "can it build and parse CGO-free" — yes. It does **not** establish
  that its Go parse trees are node-for-node identical to upstream across real
  code. The tokenizer suites in `go-embeddings-tokenizer.md` are the template for
  the diff that would close this: a corpus of real Go files, compared against the
  C tree-sitter's S-expressions. Assume non-conformant until run.
- **UNVERIFIED: the library's headline performance claims.** Not measured here
  and not needed for the gate — anchoring is not on the interactive path. Treated
  as marketing, per D-010's rule about seductive ratios.
- **UNVERIFIED: `malivvan/tree-sitter` with a Go grammar.** It has none today;
  if it grows one, the WASM path may be preferable to a reimplementation because
  it reuses upstream's own parser.
- **UNVERIFIED: binary-size mitigation.** Whether a build tag or a trimmed-grammar
  fork can avoid embedding all 206 grammars was not explored.

---

## Verdict

> **The CGO gate is cleared. A pure-Go, `CGO_ENABLED=0` tree-sitter
> (`codeberg.org/hum3/gotreesitter` v0.6.7) builds, cross-compiles statically,
> parses Go, and computes a symbol path and a normalized AST-node hash that hold
> under formatting churn and an insertion above, and change only on a semantic
> edit. Code anchoring no longer needs CGO.**
>
> **L3 ships now for text.** The fingerprint ladder is implemented in
> `internal/anchor` and wired into seeding and the write path; a pure-formatting
> change to a document expires zero claims and a semantic change expires exactly
> the right ones, verified end to end. Code anchoring is a drop-in behind
> `anchor.For(kind)`, **deferred not on CGO but on a code-evidence producer
> existing and a grammar-fidelity diff against upstream tree-sitter** — the same
> "diff it before you trust it" discipline D-008 applied to the tokenizer.

The three C bindings are rejected on the CGO constraint. The WASM binding is
rejected only for lacking a Go grammar. Adopting the pure-Go parser costs ~10 MB
of binary and takes on a v0.x single-maintainer dependency, which is why its
adoption is a deliberate later decision rather than a side effect of this slice.

---

## What would change the verdict

- **A code-evidence producer lands** (e.g. anchoring extracted claims to Go
  symbols). That is the trigger to run the fidelity diff and, if it passes, wire
  `gotreesitter` in behind `anchor.For(SourceCode)`.
- **The fidelity diff fails.** If `gotreesitter`'s trees diverge from upstream on
  real Go, the fallbacks are the WASM path (once it has a Go grammar, reusing
  upstream tables) or a narrower own-parser for the few node kinds anchoring
  needs. Do not ship a divergent parser as the anchoring authority.
- **The 10 MB binary cost becomes unacceptable.** Then the grammar embedding must
  be trimmed (one language, not 206) before adoption, or the code anchorer moves
  behind a build tag — which, being pure Go, does **not** reintroduce the CGO
  footgun the tokenizer spike warned about.
- **`gotreesitter` stalls.** It is a single-maintainer library, and the Go ML/
  parsing graveyard (`onnx-go` abandoned, `spago` paused) shows this is the normal
  outcome. Vendorability and the WASM alternative are the mitigations.

---

## Environment

| | |
|---|---|
| Machine | Apple M1 Pro, darwin/arm64 |
| Go | 1.26.0 |
| `codeberg.org/hum3/gotreesitter` | v0.6.7 |
| `smacker/go-tree-sitter` | v0.0.0-20240827094217 |
| `tree-sitter/go-tree-sitter` (official) | v0.25.0 |
| `alexaandru/go-sitter-forest` / `-bare` | v1.9.x / v1.10.0 |
| `malivvan/tree-sitter` | pre-release, wraps tree-sitter v0.24.7 |
