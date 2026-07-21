# Code anchoring prior art: multi-language symbol extraction, CGO-free

**Provenance.** Commissioned research to inform how CRED computes the L3
fingerprint ladder (tier-1 relocatable symbol path, tier-2 normalized
enclosing-node hash) for **code** across many languages, under the
non-negotiable `CGO_ENABLED=0` constraint (D-008, `.claude/rules/go.md` §3).

This document is the *prior-art survey*. The empirical CGO gate for one specific
pure-Go parser was already run in
[semantic-anchoring.md](spikes/semantic-anchoring.md); this survey does not
repeat it, it situates it. Where the two touch, the spike is the authority
because it ran commands; this document cites external sources.

Every external claim carries VERIFIED (URL fetched / command run) or UNVERIFIED
(plausible, load-bearing, says what would check it), per `.claude/rules/docs.md`
§1.

---

## Summary

For anchoring — which needs a *stable relocatable symbol id per line* and *the
extent of the enclosing definition*, not a full parse tree, and which degrades
safely to a raw-hash tier when it cannot name a symbol — the 20-year-old
**universal-ctags optlib regex model is the closest-fit prior art**: a
table-driven, one-regex-set-per-language registry where **adding a language is
adding data, not code**. That is exactly the extensibility CRED wants.

The existing anchoring spike proved a *second* path is also open: a pure-Go,
CGO-free tree-sitter (`gotreesitter`) builds and anchors Go correctly. The two
are not rivals so much as two fidelity tiers. The recommendation below keeps the
regex registry as the default breadth mechanism and treats the pure-Go parser as
an opt-in high-fidelity upgrade for the few languages that justify its ~10 MB and
its v0.x single-maintainer risk.

---

## (a) The ctags optlib regex model, and why it maps onto tier-1/tier-2

### What optlib is

universal-ctags generates a tag index of "language objects" (definitions) found
in source. It inherits from Exuberant Ctags three things that matter here:
multi-language support, **user-definable languages searched by regular
expressions** (called *optlib*), and a per-language notion of tag *kinds*
(VERIFIED — README, <https://github.com/universal-ctags/ctags>).

A parser can be defined entirely from the command line or a `.ctags` option
file, with no C code. The core options (VERIFIED — Debian manpage
`ctags-optlib.7`,
<https://manpages.debian.org/testing/universal-ctags/ctags-optlib.7.en.html>;
and <https://docs.ctags.io/en/latest/optlib.html>):

- `--regex-<LANG>=/<line_pattern>/<name_pattern>/<kind-spec>/[<flags>]` — a
  single-line pattern; the name capture becomes the tag name, the kind-spec
  assigns its kind.
- `--mline-regex-<LANG>=/<line_pattern>/<name_pattern>/<kind-spec>/{mgroup=<N>}[<flags>]`
  — a multi-line pattern; `{mgroup=<N>}` (mandatory) names the capture group that
  fixes the tag's line.
- `--kinddef-<LANG>==<letter>,<name>,<description>` — declares a kind up front
  (a universal-ctags addition; Exuberant defined kinds as a side effect of
  `--regex-`).
- `--kinds-<LANG>=` — enables/disables kinds for a language.

Flags that matter for extent and nesting (VERIFIED — same manpage):

- `exclusive` (`x`): if a line matches this pattern, skip the other patterns for
  that line — the mechanism that stops one construct matching as several kinds.
- `scope=push|pop|ref`: an internal stack. `push` when entering a container
  (class/module), `pop` when leaving, `ref` to stamp the current top onto a
  tag's scope field. This is how ctags produces qualified names like
  `Class::method` without a real parse tree.

Concrete shipped example (VERIFIED — same source), the Perl POD parser:

```
--kinddef-pod=c,chapter,chapters
--regex-pod=/^=head1[ \t]+(.+)/\1/c/
```

An optlib parser can be **translated to C** (`misc/optlib2c`) to become a
built-in parser for speed, so the regex definition is not a second-class citizen
— it is the same data the maintainers themselves start from (VERIFIED — DeepWiki
summary of the repo, <https://deepwiki.com/universal-ctags/ctags>; optlib2c is
referenced in <https://docs.ctags.io/en/latest/optlib.html>).

### Scale of the registry

- **173 parsers total**, organized in three categories: built-in C
  (`parsers/*.c`), optlib regex (`optlib/*.ctags`), and conditional parsers that
  need external libraries (UNVERIFIED-as-exact — the count and the three-category
  split come from DeepWiki's reading of the repo,
  <https://deepwiki.com/universal-ctags/ctags>; check with
  `ctags --list-languages | wc -l` against a current build).
- **31 optlib `.ctags` files ship in-tree** (VERIFIED — `gh api
  repos/universal-ctags/ctags/contents/optlib --jq '[.[]|select(.name|endswith(".ctags"))]|length'`
  → 31; sample: `cmake`, `elixir`, `gomod`, `kconfig`, `meson`, `pod`,
  `pkgConfig`). So the *majority* of languages are hand-written C parsers, but a
  meaningful, self-contained, copy-pasteable **data corpus of regex parsers
  exists** and is reusable — it is POSIX ERE text, not code.

### Why this maps onto our need

CRED does not need a parse tree. It needs, per line: (1) a stable name for the
enclosing definition that survives the definition moving down the file — **tier
1**; and (2) the extent of that definition so its normalized text can be hashed —
**tier 2**. The optlib model produces both by construction:

- The `name_pattern` capture is a **relocatable symbol id** — it is content-based
  (the identifier), not line-based, so line moves do not change it. That is tier
  1's requirement verbatim.
- `scope=push/pop` bounds the **enclosing container's extent**; the region
  between a push and its pop is the node whose normalized bytes become tier 2.
- `kind` gives us a typed anchor (function vs. field vs. import) so the ladder can
  prefer anchoring to a function over a variable when a line falls inside both.

The 20-year-old design already solved "a relocatable symbol id plus an enclosing
extent, table-driven per language, extensible by data." That is the reference
model to copy, not merely cite.

---

## (b) The CGO-free option landscape, and the trade-off named

The constraint is absolute: a single static binary, `CGO_ENABLED=0`, distroless,
cross-compiled without QEMU (D-008). Anything needing a C library *per language*
or a language server *per language* is disqualified as a bundled dependency.

| Approach | CGO-free single binary? | Extensible by data? | Fidelity | Verdict for CRED |
|---|---|---|---|---|
| ctags C bindings (via subprocess) | Binary is external, not bundled | Yes (optlib) | High | Subprocess only; not embeddable statically |
| Standard tree-sitter Go bindings | **No — all CGO** | n/a | High | Disqualified as bundled dep |
| tree-sitter via WASM on wazero (`malivvan`) | Yes | Per-grammar WASM blob | High | Real but pre-release; grammar-poor |
| Pure-Go tree-sitter (`gotreesitter`) | **Yes (verified)** | Embedded grammar blobs | High, fidelity unverified | Opt-in upgrade, +10 MB, v0.x |
| Regex registry (optlib-style, in Go) | **Yes (`regexp` is stdlib)** | **Yes — pure data** | Lower, degrades safely | **Default breadth mechanism** |
| LSP `documentSymbol` / SCIP / LSIF | No (server/indexer per lang) | No | Highest | Off the table for a single binary |

Point-by-point, with sources:

**Standard tree-sitter Go bindings are all CGO.** The official
`tree-sitter/go-tree-sitter` wraps the C library and ships `allocator.c` /
`allocator.h`, warning about `runtime.SetFinalizer`/CGO memory (VERIFIED —
<https://github.com/tree-sitter/go-tree-sitter>). The anchoring spike
independently FALSIFIED `smacker`, the official binding, and
`alexaandru/go-sitter-forest` as pure-Go paths — all fail `CGO_ENABLED=0`
(VERIFIED — [semantic-anchoring.md](spikes/semantic-anchoring.md), lines 81–100).

**tree-sitter-via-WASM-on-wazero is real but thin.** `malivvan/tree-sitter`
genuinely runs a WASM build of tree-sitter under wazero, so it is CGO-free — but
it is pre-release ("expect bugs and API breaking changes"), ~3 commits, single
digit stars, and ships only C/C++ grammars, no Go (VERIFIED —
<https://github.com/malivvan/tree-sitter>; corroborated by
[semantic-anchoring.md](spikes/semantic-anchoring.md) lines 105–112). Not usable
for breadth today; worth watching because it reuses upstream's *own* parse tables
rather than reimplementing them.

**A pure-Go tree-sitter now exists and clears the gate.** `gotreesitter`
(`codeberg.org/hum3/gotreesitter`, GitHub mirror `drummonds/gotreesitter`) is a
from-scratch GLR reimplementation that loads tree-sitter's parse-table format,
embeds ~206 grammars, and needs no C toolchain (VERIFIED as CGO-free and
Go-parsing —
[semantic-anchoring.md](spikes/semantic-anchoring.md) lines 114–148;
project self-description, <https://github.com/drummonds/gotreesitter>). Its costs
are named and real: **~10 MB of binary** (all 206 grammars embedded), **v0.x with
a single maintainer**, headline perf claims of the "seductive ratio" shape D-010
warns about, and — decisively — **grammar fidelity to upstream tree-sitter across
real code is UNVERIFIED** beyond a four-snippet smoke test (VERIFIED that these
are open —
[semantic-anchoring.md](spikes/semantic-anchoring.md) lines 150–218).

**LSP / SCIP / LSIF are the wrong tool for a single binary.** LSP
`textDocument/documentSymbol` returns a symbol outline but **requires a running
language server per language** — a request/response protocol, not a library
(VERIFIED — LSP 3.17 spec,
<https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/>).
LSIF and its successor SCIP are *static index formats* that dump a server's
knowledge so LSP queries can be answered without the server — but *producing*
them still requires a per-language indexer (scip-go, scip-typescript, scip-clang,
scip-java, rust-analyzer, scip-python, scip-ruby, scip-dotnet …), each a separate
toolchain (VERIFIED — SCIP announcement,
<https://sourcegraph.com/blog/announcing-scip>; Sourcegraph indexers,
<https://sourcegraph.com/docs/code-navigation/writing-an-indexer>). SCIP is a
better *format* than LSIF (Protobuf, human-readable string symbol ids, ~4–5x
smaller) but that is irrelevant to our constraint: both need N toolchains, which
a single CGO-free binary cannot bundle.

### How the reference implementation (Sourcegraph) actually does breadth

Sourcegraph is the load-bearing precedent for "multi-language symbols without a
server per language": it runs **universal-ctags as a subprocess**, sandboxed. Its
symbols service takes `CTAGS_COMMAND` (default `universal-ctags`) and requires the
binary be **compiled with JSON and seccomp support**, running
`CTAGS_PROCESSES` parallel parser processes (VERIFIED — Sourcegraph search-based
navigation docs surfaced via search;
<https://docs.sourcegraph.com/code_navigation/explanations/search_based_code_navigation>,
and the `sourcegraph/go-ctags` wrapper,
<https://github.com/sourcegraph/go-ctags>). Search-based (ctags) navigation
covers **~40 languages** and is the *fallback breadth tier*; precise SCIP
indexers are the per-language *high-fidelity tier* (VERIFIED — same docs).
Zoekt, their indexer, extracts symbols with **tree-sitter queries for common
languages plus universal-ctags for languages without good tree-sitter grammars**
(VERIFIED — same search surface).

The architectural lesson transfers directly: **regex/ctags is the breadth tier;
a real parser is the fidelity tier; you keep both and route by language.** The
only adaptation CRED makes is that it cannot shell out to a C binary and stay a
single static artifact — so its breadth tier is an *in-process reimplementation*
of the optlib idea using Go's stdlib `regexp` (RE2, pure Go, no CGO), fed by the
same kind of per-language regex data.

**Trade-off, named.** The regex registry buys breadth, tiny footprint, and
data-only extensibility at the cost of fidelity: RE2 lacks backreferences and
lookaround (a deliberate linear-time guarantee), and line-oriented regex misses
multi-line signatures and mis-fires inside strings/comments (see (d)). The
pure-Go tree-sitter buys correctness at the cost of ~10 MB, a v0.x dependency,
and an unpaid fidelity-verification debt. Neither is free; the design keeps the
cheap one as default and the expensive one as an explicit opt-in.

---

## (c) Recommended default language set and definition kinds

Rationale for the set: cover the languages CRED's own corpus and its likely
early users produce, ordered by prevalence in agent-written repos, and stop where
ctags itself has a parser to copy kinds from. "Kinds" below are the *conceptual*
definition types worth anchoring; ctags assigns each a single-letter code per
language, enumerable with `ctags --list-kinds-full=<LANG>` (UNVERIFIED per-letter
— the exact letters vary by language and version; verify against a current build
before encoding them, do not assume `f`=function everywhere).

Anchor priority within a line: prefer the **innermost named definition** whose
extent contains the line; fall back outward; fall back to file-level raw hash
(tier 4) when nothing matches. See (d).

| Language | Definition kinds worth detecting (tier-1 anchors) |
|---|---|
| Go | package, func, method, type, struct, interface, const, var |
| TypeScript / JavaScript | function, method, class, interface, namespace/module, const/var, property, enum |
| Python | module, class, function/method, variable (module-level) |
| Rust | mod, fn, struct, enum, trait, impl, const/static, macro |
| C | function, struct, union, enum, typedef, macro (`#define`), variable |
| C++ | + namespace, class, method, template, using |
| Java | class, interface, enum, method, field, package |
| C# | namespace, class, interface, struct, enum, method, property, field |
| Ruby | module, class, method (def), constant |
| PHP | namespace, class, interface, trait, function, method, const |
| Swift | class, struct, enum, protocol, func/method, extension, property |
| Kotlin | package, class, interface, object, fun, property |
| Scala | package/object, class, trait, def, val/var |
| CSS | selector, id-selector, class-selector (each rule block is an extent) |
| HTML | element with `id`, `<h1..h6>` headings, `<script>`/`<style>` regions |

Notes:

- These kind lists are drawn from ctags' own per-language kind sets (the concept
  is VERIFIED — ctags defines kinds per language, README /
  `--list-kinds-full`; the *specific membership per language above is UNVERIFIED*
  and must be reconciled against `ctags --list-kinds-full=<LANG>` output).
- The optlib `.ctags` files that ship in-tree (`elixir`, `cmake`, `meson`,
  `gomod`, `kconfig`, `pod`, `pkgConfig`, …) are ready-made data for those
  languages and are the template for the file format CRED should adopt (VERIFIED
  — GitHub API listing, above).
- **Data, not code, is the deliverable per language.** A language is a small
  table: for each kind, a line-regex, the name capture group, and optional
  scope-push/pop regexes for containers. Adding Elixir should be adding an Elixir
  table, changing zero Go.

---

## (d) Failure modes and safe degradation

Regex symbol extraction has well-known, sourced failure modes. For a *general*
code-intelligence tool these are correctness bugs; for **anchoring** most are
survivable because the ladder is allowed to not identify a symbol and fall back
to a raw hash — a missed anchor costs precision, not correctness.

**1. False positives in strings and comments.** A definition-looking pattern
inside a string literal or comment is matched as a real definition. This is the
canonical regex-parsing hazard — "the problem with regex patterns is false
positives with things which look like [code] but aren't, e.g. inside strings"
(VERIFIED — discussion at
<https://blogs.perl.org/users/ben_bullock/2017/08/c-comments-and-regular-expressions.html>,
surfaced from the ctags limitations search). ctags mitigates with multi-table
regex (a comment/string sub-table that consumes those regions so the definition
table never sees them) (VERIFIED — running-multi-parsers docs referenced from
the same search;
<https://docs.ctags.io/en/stable/running-multi-parsers.html>).

**2. Multi-line signatures.** ctags' regex engine "processes source files
line-by-line," so a C/C++ function signature or parameter list spanning several
lines is not seen as one unit by `--regex-`; the fix is `--mline-regex-` with an
explicit `{mgroup=N}` (VERIFIED — optlib manpage and the line-by-line note from
the limitations search,
<https://manpages.debian.org/testing/universal-ctags/ctags-optlib.7.en.html>).

**3. Macros and preprocessor definitions.** `#define` and macro-generated
declarations can hide or fabricate definitions; ctags treats macros as their own
kind rather than trying to expand them (VERIFIED-conceptually — macro is a
standard C kind; expansion is out of scope for a regex parser).

**4. POSIX-ERE / RE2 engine limits.** universal-ctags' POSIX regex API "misses
pretty much every addition of the last 20 years — lazy captures, non-capturing
groups, lookahead/behind" and the glibc engine is slow (VERIFIED — maintainer
discussion, <https://github.com/universal-ctags/ctags/issues/1861>). Go's stdlib
`regexp` (RE2) has the *same* expressive limits (no backreferences, no
lookaround) by design, but is linear-time and pure Go — a good trade for
anchoring, where a pathological-backtracking DoS on hostile input would be worse
than a missed anchor.

### Safe-degradation guidance (the "good enough" bar)

Anchoring's bar is lower than a code-navigation tool's, and the ladder is built
to degrade:

- **Degrade to tier 4 (raw hash) when no symbol is identified.** If the line
  falls in no matched definition — an unknown language, a construct with no kind,
  a false-negative — anchor to the file-plus-line raw hash (the tier-4 the PRD
  already ships). A missing tier-1/2 must never *invent* an anchor.
- **A false positive is worse than a false negative.** A confidently wrong anchor
  re-validates a claim against the wrong code (the exact failure L3 exists to
  prevent — [semantic-anchoring.md](spikes/semantic-anchoring.md) line 39). So
  tune the regex tables toward *precision*: only emit tier-1 when the name and
  kind are unambiguous; otherwise fall back. `exclusive` flags and a
  comment/string pre-pass (mitigations 1) buy most of this.
- **Tier-2 normalization absorbs reformatting.** Tier 2 hashes the *normalized*
  bytes of the enclosing extent (whitespace/formatting collapsed), which is what
  lets a gofmt/prettier churn leave the hash unchanged — the property the spike
  verified end-to-end for the tree-sitter path and that the regex path must match.
- **Language coverage degrades gracefully.** An unregistered language is not an
  error; it is tier-4-only until someone adds a table. This is the whole point of
  data-not-code extensibility: coverage grows without releases.

The failure modes are real and sourced, but none of them break anchoring as long
as the tool prefers falling back over guessing. That asymmetry — cheap breadth
that degrades to a raw hash, with an opt-in real parser for the languages that
earn it — is the recommended shape.

---

## What is unverified, and what would check it

- **UNVERIFIED — the "173 parsers / three categories" figure.** From DeepWiki's
  reading of the repo, not from a run. Check: `ctags --list-languages | wc -l`
  and inspect `parsers/` vs `optlib/` in a checkout.
- **UNVERIFIED — the per-language kind letters in (c).** The *concept* of
  per-language kinds is verified; the specific membership is not. Check:
  `ctags --list-kinds-full=<LANG>` per target language.
- **UNVERIFIED — `gotreesitter` grammar fidelity.** Inherited open item from the
  anchoring spike; a divergent parser must not be the anchoring authority. Check:
  diff its S-expressions against C tree-sitter over a real corpus (the tokenizer
  spike's method).
- **UNVERIFIED — whether an in-process Go regex registry reaches ctags-level
  breadth.** No one has been found porting the optlib corpus to RE2 in Go; the
  `.ctags` files are portable data but RE2's lack of backreferences may force
  rewrites of some patterns. Check: attempt to load a handful of shipped `.ctags`
  files (`elixir`, `pod`) against `regexp` and measure what breaks.
