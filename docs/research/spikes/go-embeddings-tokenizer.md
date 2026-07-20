# Go Embeddings and the WordPiece Tokenizer

The open question from [packaging-and-first-run.md](packaging-and-first-run.md):

> **The pure-Go WordPiece tokenizer for `bge-small-en-v1.5` is unverified, and
> the entire Go recommendation rests on it.**

This spike closes it empirically. Everything below was executed on the machine
described under [Environment](#environment); no number is estimated.

---

## Decisions

| Area | Decision |
|---|---|
| Tokenizer | **Pure Go, written in-repo.** Not a dependency |
| Character classes | **Generated tables**, extracted from the Rust tokenizer. Never `unicode.Is` |
| Inference at v1 | **Pure Go** via `onnx-gomlx` + gomlx `simplego` |
| Inference escape hatch | ONNX Runtime behind a **build tag**, not a rewrite |
| CI guard | `go list -deps` must report **zero cgo packages** |
| Language | **Go survives.** Conditionally — see [Verdict](#verdict) |

---

## The question and why it gates Go

CRED needs to turn text into a 384-dimension vector locally, with no API key, on
an air-gapped laptop. Two things have to happen: WordPiece tokenization, and a
BERT forward pass.

If either requires CGO, three packaging properties die at once:

- cross-compilation from a single amd64 runner (QEMU is 10–40x slower and flaky)
- `distroless/static` and `FROM scratch` base images
- musl/Alpine compatibility

Those are not independent losses that can be traded off one at a time. CGO is a
single switch that flips all three, and they are the substance of the Go
recommendation. Without them Go's advantage over Node and Python narrows to
approximately nothing, and the language decision should be revisited.

The tokenizer was the sharper risk. A forward pass that is numerically close is
useful; a tokenizer that is *nearly* right is worthless. One wrong token ID
produces a different vector, and the failure is silent — recall quality degrades
with no error, which is the worst failure shape for this project.

---

## Method

Scratch directory outside the repo. Model files from HuggingFace:

```
curl -sSL -o vocab.txt https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/vocab.txt
# also tokenizer.json, tokenizer_config.json, special_tokens_map.json, config.json
curl -sSL -o model.onnx https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/onnx/model.onnx
```

`vocab.txt` 231,508 bytes (30,522 tokens), `tokenizer.json` 711,396 bytes,
`model.onnx` 133,093,490 bytes.

The pipeline to reproduce, read out of `tokenizer.json` rather than assumed:

```
NORMALIZER: {"type":"BertNormalizer","clean_text":true,"handle_chinese_chars":true,
             "strip_accents":null,"lowercase":true}
PRETOK:     {"type":"BertPreTokenizer"}
POST:       TemplateProcessing  [CLS] $A [SEP]
MODEL_CFG:  {'type':'WordPiece','unk_token':'[UNK]',
             'continuing_subword_prefix':'##','max_input_chars_per_word':100}
```

Note `strip_accents: null`, which means *inherit `lowercase`* — so accents are
stripped, and stripped **before** lowercasing. Getting that order backwards is a
silent corruption for every accented input.

Three test suites, all diffed against `transformers` 5.14.1 with
`use_fast=True` (`is_fast: True`), calling
`tok(text, truncation=True, max_length=512)["input_ids"]`:

1. **43 curated cases** — English, code identifiers, punctuation-dense code,
   URLs, accents, Greek/Cyrillic/CJK/Hangul, emoji, a ZWJ family sequence, empty
   string, whitespace-only, NUL and BEL control characters, U+FFFD, NBSP, a
   120-character unknown token, literal `[CLS]`/`[SEP]`/`[MASK]` in text,
   Turkish dotted İ, German ß, ligatures, and a 1,900-word paragraph to force
   512-token truncation.
2. **47,676 fuzz cases** — random strings drawn from 16 Unicode blocks, mixed-block
   strings, generated code-like identifiers, every markdown file in this repo cut
   into 300-character chunks, and randomly embedded special-token literals.
3. **194,528 single-codepoint probes** — `"ab" + chr(cp) + "cd"` for every
   codepoint from U+0020 to U+2FFFF excluding surrogates. This isolates
   per-character classification from everything else.

Forward pass compared against `onnxruntime` 1.27.0 running the identical
`model.onnx` on the identical token IDs, both CLS-pooled and L2-normalized.

---

## Results: tokenizer

**VERIFIED — exact match, 242,247 of 242,247 inputs, zero mismatches.**

```
cases: 43  match: 43  mismatch: 0
exact match rate: 100.0%

fuzz cases: 47676  match: 47676  mismatch: 0
exact match rate: 100.0000%

codepoint probes: 194528  mismatch: 0  match rate: 100.0000%
```

This did not happen on the first attempt, and the two failure modes are the
substance of this spike. A textbook-correct WordPiece implementation — one that
follows the published BERT algorithm and uses Go's standard library Unicode
tables — scored **99.67%** on the fuzz corpus and **99.58%** on the codepoint
probe. Both remaining defects are invisible on English text.

### Defect 1: added tokens are matched before normalization

First run, 41/43 curated cases:

```
--- case 31: '[CLS] literal special token in text [SEP]'
    py  n=9  [101, 101, 18204, 2569, 19204, 1999, 3793, 102, 102]
    go  n=14 [101, 1031, 18856, 2015, 1033, 18204, ...]
    py toks: ['[CLS]', '[CLS]', 'literal', 'special', 'token', 'in', 'text', '[SEP]', '[SEP]']
    go toks: ['[CLS]', '[', 'cl', '##s', ']', 'literal', ..., '[', 'sep', ']', '[SEP]']
```

HuggingFace's `AddedVocabulary` scans the **raw** input for the five special
tokens and splits around them before the normalizer ever runs. All five carry
`normalized: false`, so lowercasing never reaches them. An implementation that
normalizes first can never recover the match, because `[CLS]` has become `[cls]`.

This matters for CRED specifically: the corpus is source code and commit
messages, and `[MASK]`-shaped bracketed tokens occur naturally in logs, changelogs,
and template strings.

### Defect 2: the Rust tokenizer's Unicode tables are frozen

The remaining 155 fuzz failures were all Arabic-script, which pointed at
character classification. The codepoint probe made it exact: **824 of 194,528
codepoints (0.42%) are classified differently by Go's standard library than by
the Rust `tokenizers` crate.**

```
by unicode general category (Python 3.13 tables):
  {'Mn': 419, 'Lo': 256, 'Po': 117, 'Cf': 20, 'Ps': 4, 'Pe': 4, 'Pd': 2, 'So': 1, 'Mc': 1}
```

Two independent causes:

**Unicode version skew (568 codepoints).** The Rust crate classifies characters
using tables frozen at an older Unicode release. Go 1.26 ships current tables. So
Go strips `U+1ABF COMBINING LATIN SMALL LETTER W BELOW` as an accent and splits
on `U+061D ARABIC END OF TEXT MARK` as punctuation; HuggingFace does neither,
because in its tables those codepoints are unassigned. Verified directly against
the Rust normalizer:

```
U+061D: HF-normalized='ab؝cd'   toks=['[CLS]', '[UNK]', '[SEP]']
U+0898: HF-normalized='ab࢘cd'   toks=['[CLS]', '[UNK]', '[SEP]']
```

**A CJK range bug in the Rust implementation (256 codepoints).** Extracting the
CJK ranges empirically from `BertNormalizer`:

```
HF tokenizers is_chinese_char ranges:
  0x3400..0x4DBF   0x4E00..0x9FFF   0xF900..0xFAFF   0x20000..0x2A6DF
  0x2A700..0x2B81F   0x2B920..0x2CEAF   0x2F800..0x2FA1F
```

Original BERT specifies `0x2B820..0x2CEAF` as one contiguous range. The Rust
implementation has `0x2B920..0x2CEAF` — a **256-codepoint hole at
U+2B820..U+2B91F**, CJK Extension E. Confirmed at the boundary:

```
CJK U+2B81F: HF-normalized='ab \U0002b81f cd'   <- padded, treated as CJK
CJK U+2B820: HF-normalized='ab𫠠cd'              <- not padded
```

This is a defect in HuggingFace, not in Go. But "correct" is not the goal —
**byte-identical to the tokenizer that trained the model** is the goal, and that
means reproducing the bug.

### The fix, and why it is the right shape

Rather than hand-transcribe historical Unicode tables, the character-class
predicates were **extracted from the Rust tokenizer by exhaustive probe** and
emitted as sorted Go range tables with binary-search lookup:

```
control:    137679 codepoints,  24 ranges
whitespace:     22 codepoints,  10 ranges
Mn:           1549 codepoints, 279 ranges
punct:         726 codepoints, 156 ranges
cjk:         81264 codepoints,   7 ranges
```

The generator is ~40 lines of Python and runs against whatever `tokenizers`
version is pinned. This turns "match a foreign library's frozen Unicode
semantics" from an unbounded correctness risk into a build step with a
regenerable artifact. **This is the load-bearing recommendation of the spike.**

> **Rule: never call `unicode.Is` in the tokenizer.** Go's tables are current and
> correct, and both properties are wrong here. Use the generated tables, pin the
> `tokenizers` version that produced them, and regenerate deliberately.

### Tokenizer throughput

```
Go   : 8400 texts, 264600 bytes, 93200 tokens in 46.798084ms
       => 179495 texts/s, 5.65 MB/s, 1991535 tokens/s
Python (HF fast, one at a time):
       8400 texts, 93200 tokens in 0.243s => 34530 texts/s, 383119 tokens/s
```

Both produce 93,200 tokens. Go is **5.2x faster** than the Rust-backed fast
tokenizer invoked per-text from Python — the comparison flatters Go by including
Python's per-call overhead, but per-call is how CRED would use it. Tokenization
is not a bottleneck at any scale this project will reach.

Binary, tokenizer only, `-ldflags="-s -w"`: **2,194,450 bytes**. Sole dependency
is `golang.org/x/text` for NFD normalization.

---

## Results: forward pass

**VERIFIED — pure Go, CGO off, numerically equivalent to ONNX Runtime.**

`onnx-gomlx` v0.4.2 loading `model.onnx` onto the gomlx `simplego` backend:

```
backend: SimpleGo (go)
onnx inputs: [input_ids attention_mask token_type_ids]
onnx outputs: [last_hidden_state]
embedded 8 texts in 1.18287825s => 147.9 ms/text
vec0[:5]=[-0.10406199 -0.013690416 -0.009501907 0.107154176 0.010600818] norm=1.000000
```

Against `onnxruntime` 1.27.0 on the same IDs, CLS-pooled and L2-normalized:

```
cosine similarity Go(simplego) vs onnxruntime, per text:
  1.00000000  maxabs=1.099e-07  'The quick brown fox jumps over the lazy dog.'
  1.00000000  maxabs=8.956e-08  'Hybrid retrieval fuses BM25 and dense vectors with r'
  1.00000000  maxabs=1.379e-07  'func (s *Server) Recall(ctx context.Context, q *Quer'
  1.00000000  maxabs=9.857e-08  'SELECT * FROM claims WHERE embedding <=> $1 LIMIT 10'
  ...
min cosine: 1.00000000   mean: 1.00000000
all > 0.999: True   all > 0.99999: True
```

Maximum per-element deviation 1.4e-7 — float32 rounding. The spike's bar was
cosine > 0.999; the result is indistinguishable from 1.

### Latency: the real cost

Same model, same shapes, batch 8, M1 Pro:

| Sequence length | Go `simplego` | ONNX Runtime | Ratio |
|---|---|---|---|
| 16 | 25.4 ms | 2.7 ms | 9.4x |
| 32 | 51.1 ms | 4.8 ms | 10.6x |
| 64 | 103.2 ms | 9.5 ms | 10.9x |
| 128 | 222.9 ms | 18.4 ms | 12.1x |
| 256 | 537.0 ms | 42.1 ms | 12.8x |
| 512 | 1503.7 ms | 94.2 ms | 16.0x |

Batching barely helps — at seq 128, batch 1 is 276.8 ms/text and batch 32 is
245.1 ms/text, an 11% gain. gomlx's own documentation scopes the Go backend to
"smaller models" and "batches of roughly 32 inputs".

Parallelism is partial: 26.08s user against 6.19s wall is ~4.2x on 10 cores.
The gap is unoptimized kernels, not a fundamental limit — which means it may
narrow, but betting on that is not a plan.

Peak RSS, batch of 8 at seq 128: **631,635,968 bytes (~632 MB)**. Against the
~2 GB floor already documented for CRED, this is the single largest consumer and
deserves a limit test before v1.

**The two paths have opposite sensitivities, and this decides the verdict:**

- **Recall (interactive).** A query is short. At seq 32, 51 ms to embed. Well
  inside any sane MCP tool-call budget. **Not a problem.**
- **Ingestion (batch).** 10,000 chunks at seq 256 is ~1.5 hours in Go versus
  ~7 minutes with ONNX Runtime. **This is the problem.**

First-run seeding is fine — the ~37 claims in the documented first-run output is
about 8 seconds. It is bulk backfill of a large repository that hurts.

---

## Results: CGO and portability

**VERIFIED.**

```
=== packages in build graph using cgo ===
(empty = none)
total packages in build: 218
```

```
=== cross-compile the FULL stack (onnx-gomlx + simplego), CGO off ===
  OK   linux/amd64   10012 KiB
  OK   linux/arm64    9088 KiB
  OK   darwin/arm64   9374 KiB
  OK   windows/amd64 10260 KiB
=== linux/amd64 static? ===
/tmp/fwd_linux-amd64: ELF 64-bit LSB executable, x86-64, version 1 (SYSV),
statically linked, ... stripped
```

The tokenizer alone cross-compiles to five targets at ~3.1–3.3 MiB including
freebsd/amd64.

`hugot` v0.7.5 was verified independently to the same standard:

```
=== CGO_ENABLED=0 build ===  BUILD OK
=== packages in build graph using cgo ===  (empty = none)
/tmp/hchk_linux: ELF 64-bit LSB executable, x86-64, ... statically linked
=== run ===
hugot NewGoSession err: <nil> | session non-nil: true
```

### A methodology trap worth recording

**`go tool nm | grep cgo` is not a CGO test and will produce a false positive.**
The Go runtime always contains cgo stub symbols:

```
=== _cgo_ prefixed symbols ===
CGO_ENABLED=0: 10
CGO_ENABLED=1: 10
```

Identical counts, and `runtime.cgocall`, `_cgo_init`, `runtime.iscgo` are present
in a binary with CGO disabled. On macOS `otool -L` is also misleading, because Go
links `libSystem` regardless.

The two checks that actually work:

```sh
go list -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' -deps ./...   # must be empty
file ./binary                                                     # "statically linked"
```

Use the first in CI. It answers the question directly — which packages in the
build graph compile C — rather than inferring it from artifacts.

### The build-tag footgun

`hugot` keeps the CGO dependencies in `go.mod` and gates them behind build tags.
Adding one tag silently reintroduces every dependency the packaging strategy
exists to avoid:

```
=== does -tags ORT reintroduce cgo? ===
runtime/cgo
github.com/daulet/tokenizers
github.com/knights-analytics/ortgenai
github.com/yalue/onnxruntime_go
```

This is the mechanism by which CRED would lose static builds without anyone
deciding to. **CI must assert the empty `go list -deps` result on the exact build
command that ships**, not on a default build.

---

## Library survey

Verified by building, or by reading `go.mod` and source. Star counts are
approximate and were not independently recounted.

| Module | Version | License | CGO | Status |
|---|---|---|---|---|
| `knights-analytics/hugot` | v0.7.5 (2026-06) | Apache-2.0 | **No**, unless `-tags ORT/XLA/ALL` | Active |
| `gomlx/gomlx` (`simplego`) | v0.27.3 | Apache-2.0 | **No** | Active |
| `gomlx/onnx-gomlx` | v0.4.2 | Apache-2.0 | **No** | Active |
| `gomlx/go-huggingface` | v0.3.5 | Apache-2.0 | **No** | Active; parses `tokenizer.json`, WordPiece |
| `sugarme/tokenizer` | v0.3.0 | Apache-2.0 | **No** | Slow-moving; has WordPiece |
| `daulet/tokenizers` | v1.27.0 | MIT | **Yes** — Rust FFI | Active |
| `yalue/onnxruntime_go` | v1.31.0 | MIT | **Yes** — needs ORT shared lib | Active |
| `owulveryck/onnx-go` | v0.5.0 (2019) | MIT | No | **Abandoned** |
| `AdvancedClimateSystems/gonnx` | v1.1.0 (2024) | MPL-2.0 | No | Stale; no BERT support |
| `nlpodyssey/spago` + `cybertron` | 2023 | BSD-2 | No | **Paused by maintainer (2024-01)** |
| `shota3506/onnxruntime-purego` | — | MIT | No cgo, **but dlopens `libonnxruntime`** | Unstable |

Two notes. `gomlx/onnx-go` does not exist; the correct path is
`gomlx/onnx-gomlx`. And `onnxruntime-purego` avoids the *compiler* dependency
while keeping the *shared library* dependency — it does not solve this problem.

`onnx-gomlx` + `simplego` is the only credible pure-Go execution path. The
standalone pure-Go ONNX runtimes are all stale or incomplete.

---

## What is unverified, and why

- **UNVERIFIED: retrieval quality on CRED's actual corpus.** Everything here
  measures fidelity to a reference implementation, not whether
  `bge-small-en-v1.5` retrieves code well. `tech-decisions.md` already flags that
  MTEB is not evidence for code retrieval. Unchanged by this spike.
- **UNVERIFIED: performance on non-Apple-Silicon hardware.** All timings are one
  M1 Pro. The Go/ORT *ratio* is likely more portable than the absolute numbers,
  but neither was tested on amd64, and gomlx's kernel optimization may differ
  substantially by architecture.
- **UNVERIFIED: memory under sustained batch load.** Peak RSS was measured for a
  single batch of 8. No soak test, no fragmentation measurement, no concurrent
  server+worker figure.
- **UNVERIFIED: `hugot`'s own tokenizer.** This spike verified that `hugot`
  builds and instantiates a session CGO-free; it did **not** verify that
  `go-huggingface`'s WordPiece implementation matches HuggingFace. Given that a
  careful hand implementation was 99.67% correct on first attempt, **assume it
  does not until diffed.** The suites here are reusable for that.
- **UNVERIFIED: quantized (int8) ONNX.** Only fp32 was tested. Quantization is
  the most obvious lever on both latency and the 632 MB resident set, and it was
  not pulled.
- **UNVERIFIED: contributor-pool effect.** Unchanged and still unquantified.

---

## Fallback analysis

### (a) Out-of-process embedding sidecar

Costs the core packaging claim. "One image, two services" becomes two images
built from two toolchains, and the Python image is 300 MB–1 GB against a 2 MiB
distroless base. It also adds a startup ordering dependency and a health check
that `/readyz` must now cover, plus an IPC failure mode on the recall path that
currently cannot exist.

The deeper objection: it reintroduces Python packaging, which
`packaging-and-first-run.md` rejected on evidence. Choosing Go and then shipping
a Python container gets the costs of both.

**Rejected as the default. Acceptable as a documented opt-in for large-scale
backfill.**

### (b) Remote embedding API only

Cheapest to build and fastest to run, and it forecloses the positioning.
Air-gapped and offline self-hosting is the sovereignty argument, and
`docker compose up` requiring an API key contradicts the first-run target
directly. `tech-decisions.md` already settled this: both hosted and local
supported, neither mandatory.

**Rejected as the only path. Already the plan as an option.**

### (c) CGO with ONNX Runtime

Concretely, what breaks:

- **Cross-compilation.** Needs a cross toolchain per target or QEMU (10–40x
  slower, flaky). The COPY-only Dockerfile and single-runner build both die.
- **Distroless/static.** `distroless/static` is out; `distroless/cc` at minimum,
  realistically a glibc base. Directly contradicts the ~2 MiB base decision.
- **musl/Alpine.** ONNX Runtime ships glibc builds. Alpine needs a source build
  or `gcompat`.
- **Build time and cache.** Every build compiles C; the Go build cache does not
  help across containers.
- **Distribution.** GoReleaser's cross-compile matrix collapses; the "single
  binary" claim in the language comparison table becomes false.

**Verified fact that changes the shape of this option:** measured numerically
exact agreement between `simplego` and ONNX Runtime (cosine 1.00000000). The two
backends are interchangeable at the vector level. So this is a **build tag behind
a stable interface**, not a rewrite — a `:fast` image variant for users who want
throughput and can accept a glibc base, with the default staying static.

**Adopt as a build-tagged escape hatch. Never as the default.**

### (d) A different language for the whole product

The premise of this option was that pure-Go local embeddings might be
impossible. **That premise is now falsified.** Rust would buy native ONNX
Runtime bindings and the genuine `tokenizers` crate, at the cost of the MCP SDK
disqualifiers already recorded in `tech-decisions.md` — no server-side OAuth,
community maintenance, incomplete RC conformance. Those objections are untouched
by this spike and remain decisive.

**Rejected.**

---

## Verdict

> **Go survives.**
>
> A pure-Go, `CGO_ENABLED=0`, statically linked, cross-compiling binary tokenizes
> **byte-identically** to HuggingFace across 242,247 test inputs and produces
> embeddings **numerically indistinguishable** from ONNX Runtime (cosine
> 1.00000000). The packaging strategy — distroless static, no QEMU, musl-clean,
> one image two services — is intact and now measured rather than assumed.
>
> **The condition is throughput, and it is a real constraint, not a caveat.**
> The pure-Go forward pass is **9–16x slower** than ONNX Runtime, worsening with
> sequence length. Interactive recall is unaffected (51 ms to embed a query).
> Bulk ingestion is materially affected: ~1.5 hours versus ~7 minutes for 10,000
> chunks at sequence length 256.

Go is therefore the right default **provided the embedder sits behind an
interface from the first commit**, so that ONNX Runtime is a build tag and a
`:fast` image variant rather than a refactor. That was already the recommendation
in `packaging-and-first-run.md`; it is now a requirement with a number attached.

Two findings that must not be lost, because both are silent:

1. **Write the tokenizer; do not adopt one without diffing it.** A careful
   implementation using Go's standard Unicode tables scores 99.67%, and every
   failure is invisible on English. Generate the character-class tables from the
   pinned `tokenizers` version. Keep the three suites as regression tests.
2. **Guard CGO in CI with `go list -deps`,** on the shipping build command. One
   build tag reintroduces `runtime/cgo`, `daulet/tokenizers`, and
   `yalue/onnxruntime_go` with no visible signal, and `nm | grep cgo` will not
   catch it.

---

## What would change this verdict

- **Ingestion throughput becomes a top user complaint.** The most likely trigger.
  First response is int8 quantization (untested, and the obvious unpulled lever),
  then the ORT build tag. Only if both fail does the language question reopen.
- **`gomlx` stalls or `simplego` regresses.** It is the single point of failure
  for the pure-Go path, and the surrounding graveyard — `onnx-go` abandoned,
  `spago` paused, `gonnx` stale — shows this is the normal outcome for Go ML
  libraries. Vendor-ability and the ORT escape hatch are the mitigation.
- **The model changes to something `simplego` cannot run.** The reranker
  (`bge-reranker-v2-m3`) is a larger model and was **not tested here**; gomlx
  explicitly scopes the Go backend to small models. Spike it separately before
  committing — do not assume this result transfers.
- **Measured memory exceeds the deployment floor.** 632 MB peak for one batch of
  8 is the largest single consumer against a ~2 GB target.
- **The MCP SDK calculus shifts.** Independent of embeddings, and still the
  reason Go was chosen first.

---

## Environment

| | |
|---|---|
| Machine | Apple M1 Pro, 10 cores, 16 GB, darwin/arm64 |
| Go | 1.26.0 |
| Python | 3.13.7 |
| `transformers` / `tokenizers` | 5.14.1 / 0.22.2 |
| `onnxruntime` / `numpy` | 1.27.0 / 2.5.1 |
| `gomlx` / `onnx-gomlx` / `hugot` | v0.27.3 / v0.4.2 / v0.7.5 |
| `golang.org/x/text` | v0.40.0 |
| Model | `BAAI/bge-small-en-v1.5`, fp32 ONNX, 384-dim |
