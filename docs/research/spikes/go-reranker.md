# Go reranking: cross-encoders, SentencePiece, and int8

The open tension recorded against D-008 in the
[decision log](../decision-log.md):

> The **reranker was not tested**. `bge-reranker-v2-m3` is a substantially
> larger model and gomlx explicitly scopes `simplego` to small ones; this
> result does not transfer, and reranking is in v1 scope. **int8 quantization
> is untested**.

This spike closes both. Everything below was executed on the machine described
under [Environment](#environment). No number is estimated.

---

## Decisions

| Area | Decision |
|---|---|
| `bge-reranker-v2-m3` in pure Go | **Rejected.** Runs, but 9.3 s per pair at seq 512 |
| `bge-reranker-v2-m3` at all, any backend | **Rejected on CPU.** 882 ms per pair on ONNX Runtime; 44 s for 50 |
| Cross-encoder reranking at v1 | **Cut**, at a measured cost of 0.0167 NDCG@10. No candidate fits an interactive budget in pure Go |
| Replacement | **ColBERT-style MaxSim** over `bge-small-en-v1.5` token vectors |
| int8 in `simplego` | **Do not ship for latency.** 1.69–2.28x *slower* across four models. Ships 1.4–2.2x less RSS |
| Third-party Go tokenizers | **Still unusable.** `sugarme` 96.22%, 948 panics |

---

## The question and what D-008 left open

D-008 verified pure-Go, `CGO_ENABLED=0` inference for `bge-small-en-v1.5` — a
33.2M-parameter BERT with a WordPiece tokenizer. `tech-decisions.md` lists
`BAAI/bge-reranker-v2-m3` as the v1 reranker. That model differs on both axes
that were verified:

- **Size.** 567.8M parameters, XLM-RoBERTa-large: 24 layers, hidden 1024,
  250,002-token vocabulary. 17x the embedder.
- **Tokenizer.** SentencePiece / XLM-RoBERTa **Unigram**, not WordPiece. A
  different algorithm with a different failure surface.

Neither result transfers. Reranking is also the *only* v1 component that runs
N model passes per interactive query rather than one, so the latency ratio the
prior spike measured (9–16x) compounds by N.

Three questions, in order of what would kill the plan fastest:

1. Does `simplego` have the ops, and the memory, to run it at all?
2. Is there a pure-Go Unigram/SentencePiece tokenizer that matches the
   reference exactly?
3. At N = 20 to 50 pairs per query, what is the latency?

---

## Method

Scratch directory outside the repo. Parameter counts and op sets read from the
ONNX graphs directly:

```
python -c "import onnx; m=onnx.load(p,load_external_data=False); ..."

m3         params=   567.8M  ops=26
base       params=   278.0M  ops=27
jina       params=    37.8M  ops=27
mxbai      params=    70.8M  ops=39
bge-small  params=    33.2M  ops=19
```

`BAAI/bge-reranker-v2-m3` ships no ONNX. It was exported locally rather than
taken from a community re-upload, so the graph is attributable:

```
./venv/bin/optimum-cli export onnx --model BAAI/bge-reranker-v2-m3 \
    --task text-classification onnx-m3
```

```
	-[x] values not close enough, max diff: 6.866455078125e-05 (atol: 1e-05)
The ONNX export succeeded with the warning: ... max diff = 6.866455078125e-05.
 The exported model was saved at: onnx-m3
-rw-r--r--@  1 canh  wheel      489245 model.onnx
-rw-------@  1 canh  wheel  2271088656 model.onnx_data
```

2.27 GB of external tensor data. `onnx-gomlx` v0.4.2 resolves external data
(`onnx/onnx.go:53` `WithBaseDir`), so this loads without repacking.

The export missed `optimum`'s default 1e-5 tolerance by 6.9e-5 against the
PyTorch reference. That is float32 drift in a 24-layer graph, not a broken
export, and it is irrelevant to the latency and memory numbers this spike
turns on — but it means the exported graph is not bit-identical to the
published weights, and any future quality claim about this specific ONNX file
should account for it.

All file downloads were size-checked against the HuggingFace `x-linked-size`
header after one truncated download silently corrupted a tokenizer (see
[Falsified along the way](#falsified-along-the-way)).

**Inference harness.** A ~150-line Go program on `onnx-gomlx` v0.4.2 +
gomlx `simplego` v0.27.3, built `CGO_ENABLED=0`, taking `-model`, `-batch`,
`-seq`, `-reps`. Compile and weight upload happen in a warmup pass that is
excluded from the timings. Peak RSS from `/usr/bin/time -l`.

### Measurement conditions

**Read this before trusting any latency figure here.** The first pass of this
spike ran benchmarks while other jobs were on the
machine, and several published numbers were wrong because of it.** They have
been re-measured; the corrections are recorded inline rather than silently
patched. The rule that emerged, stated plainly because it invalidated a
headline finding:

> **A latency ratio is only trustworthy when both sides were measured
> back-to-back in the same command, on an otherwise idle machine.**
> Comparing two numbers taken from different commands under different load
> produces a plausible ratio that is pure artifact.

Concretely, the contended-versus-clean gap on identical configurations:

| Measurement | Contended | Clean | Inflation |
|---|---|---|---|
| `bge-small` b8 s128 fp32, Go | 331.38 ms | 206.13 ms | 1.61x |
| m3 b1 s512 fp32, ONNX Runtime | 1836.34 ms | 882.30 ms | 2.08x |
| m3 b1 s256 fp32, ONNX Runtime | 1032.98 ms | 415.00 ms | 2.49x |

The clean `bge-small` figure (206.13 ms) reconciles with D-008's independently
measured 222.9 ms for the same shape to within 7.5%. The contended one did not,
and that discrepancy is what exposed the problem — **cross-checking a new
measurement against a prior spike's number for the same configuration is worth
doing every time.**

All latency figures below are clean unless explicitly labelled otherwise.
Ratios measured back-to-back within one command survived the re-measurement
unchanged in direction and within ~15% in magnitude; cross-command ratios did
not.

**ONNX Runtime baseline.** `onnxruntime` 1.27.0, `CPUExecutionProvider`, same
graph, same shapes, 3 timed repetitions after a warmup.

**Tokenizer conformance.** The prior spike's three-suite structure, regenerated
for a cross-encoder — every case is run both as a single sequence and, for a
subset, as a `(query, passage)` **pair**, because pair encoding is the only
mode a cross-encoder ever uses:

1. **61 curated cases** — English, Go and SQL source, punctuation-dense code,
   URLs, accents, Greek/Cyrillic/CJK/Hangul/Thai, emoji, a ZWJ family sequence,
   empty and whitespace-only strings, NUL and BEL, U+FFFD, NBSP, a
   120-character unknown token, literal `<s>`/`</s>`/`<mask>`/`<pad>`/`<unk>`,
   Turkish dotted İ, German ß, ligatures, a literal `▁` U+2581, fullwidth and
   halfwidth forms, a git diff header, a fenced code block, and a 1,900-word
   paragraph to force 512-token truncation on both a single and a pair.
2. **40,900 fuzz cases** — random strings from 16 Unicode blocks, mixed-block
   strings, generated code-like identifiers, every markdown file in this repo
   cut into 300-character chunks, and randomly embedded special-token literals.
3. **194,528 single-codepoint probes** — `"ab" + chr(cp) + "cd"` for every
   codepoint U+0020..U+2FFFF excluding surrogates.

235,489 inputs total, diffed against `transformers` 5.14.1
(`XLMRobertaTokenizerFast`, `is_fast: True`) calling
`tok(a, b, truncation=True, max_length=512)["input_ids"]`.

**Quality evaluation.** BEIR SciFact (5,183 documents, 300 test queries),
downloaded from the canonical BEIR host:

```
curl -sSL -o scifact.zip \
  https://public.ukp.informatik.tu-darmstadt.de/thakur/BEIR/datasets/scifact.zip
```

First stage is `bge-small-en-v1.5` — the model D-008 already settled — CLS
pooled, L2 normalized, with BGE's documented query prefix. Top-50 by inner
product, then each reranker reorders those 50. Metric is NDCG@10 with binary
qrels, on a fixed random sample of 100 test queries (seed 20260720),
`max_length=256`.

Reranker scoring for the quality evaluation ran on **ONNX Runtime**, not
`simplego`. This is a deliberate separation: the prior spike verified the two
backends agree to cosine 1.00000000 on the embedder, so quality is a property
of the model and latency is a property of the backend. Measuring quality on the
slow backend would have cost hours and told us nothing extra. The `simplego`
latencies are measured separately and reported as such.

---

## Results: does `bge-reranker-v2-m3` run in pure Go?

**VERIFIED — it runs. No unsupported op, no OOM. It is far too slow to ship.**

The negative result some of this spike expected did not happen. `simplego`
executes all 26 op types in the graph and returns logits. First run, showing
the model loads and produces a score (the timing in this block is from a
contended run; the clean sweep follows):

```
$ ./goinf -model onnx-m3/model.onnx -batch 1 -seq 128 -reps 3
parsed in 12.604292ms
inputs: [input_ids attention_mask] [(Int64)[-1 -1] (Int64)[-1 -1]]
outputs: [logits] [(Float32)[-1 1]]
backend: SimpleGo (go) (Simple Go Portable Backend)
weights loaded in 1.140419875s
warmup+compile: 4.744996834s  out0=[-9.335603]
rep 0: 2.341346084s
rep 1: 2.107739625s
rep 2: 2.229608083s
RESULT batch=1 seq=128 avg=2.226231264s per_pair=2226.23ms
```

The clean sweep, batch 1, both backends measured on an idle machine:

| seq | Go `simplego` | ONNX Runtime | Ratio |
|---|---|---|---|
| 128 | 2120.13 ms | 203.2 ms | 10.4x |
| 256 | 4370.33 ms | 415.0 ms | 10.5x |
| 512 | 9332.56 ms | 882.3 ms | 10.6x |

```
=== CLEAN m3 fp32: Go simplego, batch 1
RESULT batch=1 seq=128 avg=2.120126889s per_pair=2120.13ms
RESULT batch=1 seq=256 avg=4.370326986s per_pair=4370.33ms
RESULT batch=1 seq=512 avg=9.332561041s per_pair=9332.56ms
=== CLEAN m3 fp32: ONNX Runtime, batch 1 (same process)
ORT b1 seq=128 avg=203.2ms
ORT b1 seq=256 avg=415.0ms
ORT b1 seq=512 avg=882.3ms
```

Peak RSS for m3 fp32 at batch 1, seq 512, clean run: **3.60 GB**. An earlier
contended run reported 4.58 GB.

**FALSIFIED: "the Go/ONNX Runtime gap narrows on larger models."** This spike
originally reported 4.7-8.9x for m3 against D-008's 9-16x for the embedder and
presented the narrowing as its one pleasant surprise. **It was an artifact of
measuring the two backends in separate commands under different load.** The ORT
baseline was inflated 2.1-2.5x, which halved every ratio.

Measured cleanly, the ratio is **10.4x, 10.5x, 10.6x** across three sequence
lengths - flat, and squarely inside D-008's 9-16x band. There is no narrowing.
The pure-Go penalty on a 567.8M-parameter cross-encoder is the same ~10x it is
on a 33.2M-parameter embedder.

The claim is kept rather than deleted because it reached the decision log, and
because how it failed is more useful than the number: **a wrong ratio from
contended measurement looks exactly like a real finding.** It arrived with a
plausible mechanism attached - "larger matmuls suit `simplego`'s kernels
better" - which made it easier to believe, not harder.

Two things follow, and the first is the one that matters.

**`bge-reranker-v2-m3` is not viable on CPU at all, in any language.** Fifty
pairs at sequence 512 costs **44 seconds on ONNX Runtime with CGO** and
**7.8 minutes in pure Go**. int8 is the best case available and still does not
save it: 351 ms/pair on ONNX Runtime is **17.6 seconds** for fifty, and the
same quantized graph is *slower* in pure Go (see
[int8](#results-int8-quantization)). The build-tagged ORT escape hatch D-008
established for bulk ingestion does not rescue this — it converts an impossible
latency into a differently impossible one. The model choice in
`tech-decisions.md` is wrong for a CPU-only deployment independent of the Go
question.

**Memory exceeds the deployment floor before anything else runs.** Clean peak
RSS is **3.60 GB** for a single pair at sequence 512, against the ~2 GB floor
documented for CRED. int8 brings it to 2.19 GB — still at the floor, with
nothing left for Postgres, the server, or the embedder.

---

## Results: substitutes on the same harness

All four candidates load and execute in `simplego` — including
`mxbai-rerank-xsmall-v1`, whose DeBERTa-v2 disentangled attention needs 39 op
types (`GatherElements`, `Tile`, `Sign`, `Ceil`, `LessOrEqual`). **Op coverage
was never the constraint.** Latency and memory are.

Pure Go, `simplego`, batch 1, sequence 512, all measured clean:

| Model | Params | Arch | per pair | peak RSS |
|---|---|---|---|---|
| `bge-reranker-v2-m3` | 567.8M | XLM-R-large, 24L/1024 | 9509.65 ms | 3.60 GB |
| `bge-reranker-v2-m3` int8 | — | — | 21657.83 ms | 2.19 GB |
| `bge-reranker-base` | 278.0M | XLM-R-base, 12L/768 | 3493.30 ms | 4.19 GB |
| `mxbai-rerank-xsmall-v1` | 70.8M | DeBERTa-v2, 12L/384 | 1724.11 ms | 1.37 GB |
| `mxbai-rerank-xsmall-v1` int8 | — | — | 2910.39 ms | 853 MB |
| `jina-reranker-v1-turbo-en` | 37.8M | JinaBERT, 6L/384 | **690.68 ms** | **779 MB** |
| `jina-reranker-v1-turbo-en` int8 | — | — | 1346.62 ms | 557 MB |

```
=== CLEAN b1 s512, remaining candidates
cand/base/model            | RESULT per_pair=3493.30ms  4192960512 maximum resident set size
cand/mxbai/model           | RESULT per_pair=1724.11ms  1368309760 maximum resident set size
cand/mxbai/model_quantized | RESULT per_pair=2910.39ms   852787200 maximum resident set size
```

**Peak RSS is far noisier than latency and should be read as an order of
magnitude, not a figure.** Across contended and clean runs of identical
configurations: `mxbai` and `jina` reproduced to within 2%, but
`bge-reranker-base` moved 2.65 → 4.19 GB (1.58x) and m3 moved 4.58 → 3.60 GB
(1.27x) — and in
opposite directions, so this is allocator and GC timing, not load. Latency
ratios in this spike are trustworthy when paired; RSS figures are not
trustworthy to better than ~1.6x either way.

The fastest candidate, swept over the operating range. **These rows were
measured under background load and are inflated roughly 1.5-2.4x** — the clean
figures for the shapes that matter are in the next block; these are kept only
to show the shape of the batch/sequence curve:

```
jina fp32 b=1  s=128: RESULT batch=1  seq=128 avg=176.764187ms  per_pair=176.76ms
jina fp32 b=8  s=128: RESULT batch=8  seq=128 avg=1.53412825s   per_pair=191.77ms
jina fp32 b=25 s=128: RESULT batch=25 seq=128 avg=4.72006177s   per_pair=188.80ms
jina fp32 b=1  s=256: RESULT batch=1  seq=256 avg=551.830896ms  per_pair=551.83ms
jina fp32 b=8  s=256: RESULT batch=8  seq=256 avg=3.426224354s  per_pair=428.28ms
jina fp32 b=25 s=256: RESULT batch=25 seq=256 avg=7.144374812s  per_pair=285.77ms
```

And at the actual unit of work — one query's worth of candidates, wall clock,
clean machine:

```
=== CLEAN jina-reranker-v1-turbo-en, realistic rerank workloads
RESULT batch=20 seq=128 avg=2.478544729s  per_pair=123.93ms
RESULT batch=20 seq=256 avg=5.65574025s   per_pair=282.79ms
RESULT batch=20 seq=512 avg=17.198837833s per_pair=859.94ms
RESULT batch=50 seq=128 avg=6.171238521s  per_pair=123.42ms
RESULT batch=50 seq=256 avg=14.401982416s per_pair=288.04ms
RESULT batch=50 seq=512 avg=52.955617604s per_pair=1059.11ms
```

| N candidates | seq 128 | seq 256 | seq 512 |
|---|---|---|---|
| 20 | 2.5 s | 5.7 s | 17.2 s |
| 50 | 6.2 s | 14.4 s | 53.0 s |

**2.5 seconds to rerank 20 candidates at sequence 128 — the most favourable
configuration of the smallest credible model.** At 50 candidates and sequence
512, the configuration a code chunk actually needs, it is **53 seconds**.

For scale: D-008 measured 51 ms to embed a query. Reranking costs between
**49x** (20 candidates, seq 128) and **1,038x** (50 candidates, seq 512) the
retrieval step it is meant to refine, on the interactive path. No batching or
sequence-length trade recovers that.

---

## Results: int8 quantization

**VERIFIED, and it FALSIFIES the hopeful reading in D-008.** D-008 called int8
"the most obvious unpulled lever on both the 12x gap and the 632 MB peak RSS."
It is a lever on the memory. It is a **negative** lever on the latency, in
`simplego` specifically.

Dynamic int8 via `onnxruntime.quantization.quantize_dynamic(weight_type=QInt8)`,
`bge-small-en-v1.5`, batch 8, sequence 128 — the same shape D-008 measured:

| Backend | Precision | per text | peak RSS |
|---|---|---|---|
| Go `simplego` | fp32 | 206.13 ms | 647 MB |
| Go `simplego` | **int8** | **407.21 ms** | **290 MB** |
| ONNX Runtime | fp32 | 32.33 ms | — |
| ONNX Runtime | **int8** | **18.82 ms** | — |

```
=== bge-small fp32 (clean, b8 s128)
RESULT batch=8 seq=128 avg=1.649015208s per_pair=206.13ms
           647069696  maximum resident set size
=== bge-small int8 (clean, b8 s128)
RESULT batch=8 seq=128 avg=3.257666958s per_pair=407.21ms
           290406400  maximum resident set size

ORT small/model.onnx      batch=8 seq=128 avg=258.6ms per_text=32.33ms
ORT small/model_int8.onnx batch=8 seq=128 avg=150.5ms per_text=18.82ms
```

The clean fp32 figure, **206.13 ms**, sits 7.5% under D-008's independently
measured **222.9 ms** for the identical shape. A first, contended run of this
same benchmark gave 331.38 ms — 1.49x above D-008 — and that mismatch is what
exposed the measurement problem described under
[Measurement conditions](#measurement-conditions) above.

int8 makes ONNX Runtime **1.72x faster** and `simplego` **1.98x slower** on
the embedder. The same inversion holds on every model tested. All rows below
are **clean, paired** measurements - fp32 and int8 back-to-back in one command
on an idle machine:

| Model | Go fp32 | Go int8 | Go | Go RSS fp32 -> int8 |
|---|---|---|---|---|
| `bge-small-en-v1.5` (b8 s128) | 206 ms | 407 ms | **1.98x slower** | 647 -> 290 MB (2.23x) |
| `jina-reranker-v1-turbo-en` (b1 s512) | 691 ms | 1347 ms | **1.95x slower** | 779 -> 557 MB (1.40x) |
| `mxbai-rerank-xsmall-v1` (b1 s512) | 1724 ms | 2910 ms | **1.69x slower** | 1368 -> 853 MB (1.60x) |
| `bge-reranker-v2-m3` (b1 s512) | 9510 ms | 21658 ms | **2.28x slower** | 3601 -> 2191 MB (1.64x) |

On ONNX Runtime, paired in one process, the same graphs go the other way:

| Model | ORT fp32 | ORT int8 | ORT |
|---|---|---|---|
| `bge-small-en-v1.5` (b8 s128) | 32.3 ms | 18.8 ms | **1.72x faster** |
| `bge-reranker-v2-m3` (b1 s512) | 871.2 ms | 351.2 ms | **2.48x faster** |

```
=== CLEAN m3 fp32 vs int8, Go simplego, b1 s512
RESULT batch=1 seq=512 avg=9.509654083s  per_pair=9509.65ms
          3601399808  maximum resident set size
RESULT batch=1 seq=512 avg=21.657833583s per_pair=21657.83ms
          2190934016  maximum resident set size
=== CLEAN m3 fp32 vs int8, ONNX Runtime, b1 s512 (same process)
ORT m3 fp32 b1 s512 avg=871.2ms
ORT m3 int8 b1 s512 avg=351.2ms
```

Four models, one direction in Go; two models, the opposite direction on ONNX
Runtime. The `bge-reranker-v2-m3` row states the problem cleanly: the *same*
quantized graph runs **2.48x faster** on ONNX Runtime and **2.28x slower** in
pure Go. int8 is not a property of the model; it is a property of the backend's
kernels, and `simplego` does not have them.

**Correction.** Earlier drafts of this spike reported 2.05x / 1.92x / 1.70x /
2.63x for the Go slowdowns and **5.57x** for the m3 ORT speedup. The three
*paired* figures were close to correct - the clean re-runs give 1.98x, 1.95x
and 1.69x against the original 2.05x, 1.92x and 1.70x, all within 4%. The two
that were badly wrong
were the ones assembled from **separate commands**: m3's Go slowdown (2.63x
claimed, 2.28x actual) and m3's ORT speedup (5.57x claimed, 2.48x actual). This
is the same defect that produced the falsified "gap narrowed" finding above,
and the pattern is consistent enough to state as a rule: **paired ratios
survived contention; cross-command ratios did not.**

The mechanism is visible in `onnx-gomlx` source. Dynamic quantization rewrites
the graph — the quantized `bge-small` has 72 `MatMulInteger` and 48
`DynamicQuantizeLinear` nodes where the fp32 graph had plain `MatMul` — and
`onnx-gomlx` v0.4.2's generic handler **widens both int8 operands to int32
before multiplying**, at `internal/onnxgomlx/ops.go:2967` (`onnxMatMulInteger`):

```go
// Convert inputs to int32 to prevent overflow during matrix multiplication
aWorking := ConvertDType(a, dtypes.Int32)
bWorking := ConvertDType(b, dtypes.Int32)
```

So the int8 path is an int32 GEMM plus zero-point subtraction and rescaling —
strictly more work than the fp32 GEMM it replaced, with none of the SIMD
throughput that makes int8 fast on ONNX Runtime.

`onnx-gomlx` does ship a fusion pass that rewrites
`DynamicQuantizeLinear` + `MatMulInteger` chains into `nn.QuantizedDense`
(`internal/onnxgomlx/fusion/quantized_dense.go`), and it was **in the build
graph for every measurement here** — `go list -deps` lists
`onnx-gomlx/internal/onnxgomlx/fusion`. **UNVERIFIED: whether that fusion
actually fired on these graphs.** Either it did not match, or it did and is
still slower than fp32. Distinguishing the two would need graph-level
instrumentation and was not done. The measurement stands regardless.

Two consequences worth keeping:

- **The memory result is real and useful.** 647 MB → 290 MB on the embedder is
  a 2.23x cut in the single largest consumer against CRED's ~2 GB floor, and
  `bge-reranker-v2-m3` drops 3.60 GB → 2.19 GB (1.64x). If the constraint is
  memory rather than latency, int8 is the answer. On disk the m3 graph goes
  from 2.27 GB of external data to a single 570 MB file.
- **int8 costs almost nothing in ranking quality.** On SciFact,
  `jina-reranker-v1-turbo-en` scored NDCG@10 0.7018 fp32 and 0.7014 int8 — a
  0.0004 difference. Quantization is not why the reranker plan fails.

---

## Results: the SentencePiece tokenizer

**VERIFIED — no usable pure-Go Unigram/SentencePiece tokenizer exists.**

The reranker's pipeline, read out of `tokenizer.json` rather than assumed:

```
NORMALIZER: Sequence[
              Precompiled{precompiled_charsmap: 316720 b64 chars / 237539 bytes},
              Strip{strip_left: false, strip_right: true},
              Replace{pattern: Regex " {2,}", content: "▁"}]
PRETOK:     Metaspace{replacement: "▁", add_prefix_space: true,
                      prepend_scheme: "always"}
POST:       TemplateProcessing  <s> $A </s>  /  <s> $A </s> </s> $B </s>
MODEL_CFG:  Unigram, unk_id=3, byte_fallback=false, vocab 250,002
```

The `Precompiled` normalizer is the load-bearing difference from WordPiece. It
is not NFKC — it is SentencePiece's `nmt_nfkc` map, shipped as a **237,539-byte
Darts double-array trie blob** with a 177,152-entry trie header:

```
Precompiled {'precompiled_charsmap': 316720}
charsmap raw bytes 237539
trie_size hdr 177152
```

Confirmed against the model's own `sentencepiece.bpe.model` proto:

```
model_type: UNIGRAM
add_dummy_prefix: True remove_extra_ws: True name: nmt_nfkc
```

The prior spike's central recommendation — *probe the pinned reference and
generate the tables* — does not scale to this. A WordPiece character class is a
predicate over one codepoint, so 194,528 probes enumerate it completely. A
precompiled charsmap is a **string-to-string** map with multi-codepoint inputs
and outputs; single-codepoint probing cannot enumerate it, and the artifact
would be the 237 KB blob itself plus a correct Darts trie walker.

### Candidate 1: `gomlx/go-huggingface` — cannot load the model

`go-huggingface` v0.3.5's `tokenizers/sentencepiece` wraps
`eliben/go-sentencepiece` v0.7.0, which rejects the model outright:

```
$ ./gospm ../tok_sentencepiece.bpe.model
NewProcessorFromPath ERROR: model type UNIGRAM not supported
exit=1
```

Source confirms this is by design, not a bug —
`eliben/go-sentencepiece@v0.7.0/processor.go:76`:

```go
if tspec.GetModelType() != model.TrainerSpec_BPE {
    return nil, fmt.Errorf("model type %s not supported", tspec.GetModelType())
}
```

and `processor.go:82` additionally rejects `add_dummy_prefix` and
`remove_extra_whitespaces`, both of which XLM-RoBERTa sets. `normalize.go:12`
is a two-line function that replaces spaces and does nothing else — there is no
charsmap implementation to fix.

### Candidate 2: `sugarme/tokenizer` — loads, and is wrong

v0.3.0 has every required component (`model/unigram`, `normalizer/precompiled`,
`spm/spm-precompiled`, `pretokenizer/metaspace`) and loads `tokenizer.json`
directly. Against the reference:

```
curated: n=61     match=51     mismatch=10   panic=0    rate=83.6066%
fuzz:    n=40900  match=32025  mismatch=8875 panic=945  rate=78.3007%
probe:   n=194528 match=194512 mismatch=16   panic=3    rate=99.9918%
TOTAL:   n=235489 match=226588 mismatch=8901 panic=948  rate=96.2202%
```

**96.22%, with 948 hard panics.** Without a `recover()` per call the first run
died on input 120 of 235,489:

```
panic: runtime error: index out of range [47] with length 47
github.com/sugarme/tokenizer/normalizer.(*NormalizedString).TransformRange(...)
	normalizer/normalized.go:646
github.com/sugarme/tokenizer/normalizer.(*NormalizedString).lrstrip(...)
	normalizer/normalized.go:1353
github.com/sugarme/tokenizer/normalizer.(*Strip).Normalize(...)
	normalizer/strip.go:28
```

Four defect classes, in descending order of how much they matter to CRED:

**1. `Replace{" {2,}" → "▁"}` does not take effect. 100% failure on any input
with a run of two or more spaces.**

```
ASCII fuzz with a run of 2+ spaces:  n=342  fail=342 (100.00%)
ASCII fuzz without 2+ spaces:        n=7665 fail=650 (8.48%)
```

```
[29] "multiple     internal      spaces"
     go n=15 [0, 48716, 6, 6, 6, 6, 70796, 6, 6, 6, 6, 6, 32628, 7, 2]
     rf n=6  [0, 48716, 70796, 32628, 7, 2]
```

CRED's corpus is source code. Indentation is runs of spaces. This defect fires
on essentially every chunk the product would ever tokenize, and it does not
error — it inflates the sequence with `▁` tokens (id 6) until truncation eats
real content.

**2. Truncation is silently ignored for pairs.** Single sequences truncate
correctly; the pair path — the only path a cross-encoder uses — does not:

```
case42 single long para: go n=512  ref n=512
case60 pair   long/long: go n=5044 ref n=512
```

5,044 token IDs fed to a model with `max_position_embeddings: 514`.

**3. The precompiled charsmap is applied, but only where the mapping is
one-to-one.** The codepoint probe is 99.9918% — nearly clean. All 16 failures
are codepoints the charsmap expands, deletes, or remaps:

```
U+0307 Mn COMBINING DOT ABOVE     panic    ref=[0, 10, 247209, 71574, 2]
U+0323 Mn COMBINING DOT BELOW     panic    ref=[0, 10, 3, 71574, 2]
U+0344 Mn COMBINING GREEK DIALYTIKA TONOS
                                  mismatch go=[0,1563,3,71574,2]
                                           ref=[0,1563,246635,4868,71574,2]
U+0E33 Lo THAI CHARACTER SARA AM  mismatch go=[0,1563,3,71574,2]
                                           ref=[0,1563,73003,71574,2]
U+200D Cf ZERO WIDTH JOINER       mismatch go=[0,1563,3,71574,2]
                                           ref=[0,1563,56329,2]
U+FF9E Lm HALFWIDTH KATAKANA VOICED SOUND MARK
                                  mismatch go=[0,1563,3,71574,2]
                                           ref=[0,1563,246514,71574,2]
```

Go emits `<unk>` (3) where the reference has a real token, or panics in
`TransformRange` when the replacement length differs from the source length.
The panic is a length-bookkeeping bug in the offset-tracking code, and it is
reachable from ordinary text.

**4. Failure is broad, not confined to exotic scripts.** Fuzz failure rate by
the dominant Unicode block of each input — note that the "ASCII" bucket is
inputs whose *most common* block is ASCII, so it includes mixed-script strings:

```
block                n    fail  panic  failrate
ASCII            11424    2839    513    24.85%
Gen-Punct         1986    1116    247    56.19%
Devanagari        1975    1075     10    54.43%
CJK-Sym           1943     925     28    47.61%
Arabic            2011     566     16    28.15%
Thai              1977     370      8    18.72%
Kana              1956     270     11    13.80%
Hebrew            1958     247     14    12.61%
Greek             1935     227     20    11.73%
```

Restricting to strings that are entirely ASCII, the split is 8.48% failure
without a double space and 100% with one. So the failure is not hidden from an
English-speaking developer the way the prior spike's WordPiece defects were —
it is worse than that. It is reachable from plain indented ASCII, it degrades
quality silently rather than erroring, and on 0.4% of inputs it crashes the
process.

### The BPE path is no better

`jina-reranker-v1-turbo-en` uses a far simpler pipeline — `Sequence[NFC,
Lowercase]`, `Whitespace` pre-tokenizer, BPE, no charsmap. `sugarme` on the
same 235,489 inputs:

```
curated: n=61     match=52     mismatch=9    panic=0   rate=85.2459%
fuzz:    n=40900  match=35696  mismatch=5204 panic=156 rate=87.2763%
probe:   n=194528 match=194485 mismatch=43   panic=2   rate=99.9779%
TOTAL:   n=235489 match=230233                          rate=97.7680%
```

97.77%, 158 panics. Curated failures include accented Latin, NBSP, a
120-character token, `İ`, U+2028/U+2029, and fullwidth ASCII.

**Conclusion: the reason to hand-write the tokenizer, established in D-008 for
WordPiece, holds for both Unigram and BPE.** The difference is cost. WordPiece
was a few hundred lines plus generated range tables. Unigram requires a Darts
double-array trie walker over a 237 KB binary blob, correct offset bookkeeping
through length-changing normalization, Viterbi decoding over a 250,002-entry
lattice, and a pair-truncation path — all held to byte-identity against a
reference whose bugs must be reproduced.

Both libraries are, for what it is worth, genuinely CGO-free:

```
=== gotok (sugarme/tokenizer) cgo deps ===
(empty = none)
total: 218
```

---

## Quality evaluation

A reranker that is fast and CGO-free but does not improve NDCG is not worth
shipping. One candidate here is exactly that, which is why this section exists
at all rather than being assumed.

BEIR SciFact, 100-query sample, first stage `bge-small-en-v1.5` top-50,
reranker scoring on ONNX Runtime, `max_length=256`:

```
config: NQ=100 K=50 maxlen=256
BASELINE vector-only bge-small-en-v1.5: NDCG@10=0.6662 R@10=0.8050 R@50=0.9350
m3:        NDCG@10=0.7122 delta=+0.0460 [5000 pairs, 2353s, 471 ms/pair on ORT]
jina:      NDCG@10=0.7018 delta=+0.0356 [5000 pairs,  121s, 24 ms/pair on ORT]
jina_int8: NDCG@10=0.7014 delta=+0.0353 [5000 pairs,   61s, 12 ms/pair on ORT]
mxbai:     NDCG@10=0.6415 delta=-0.0246 [5000 pairs,  262s, 52 ms/pair on ORT]
mxbai_q:   NDCG@10=0.6380 delta=-0.0282 [5000 pairs,  166s, 33 ms/pair on ORT]
```

| Method | NDCG@10 | vs vector-only |
|---|---|---|
| Vector only (`bge-small-en-v1.5`) | 0.6662 | — |
| `bge-reranker-v2-m3` fp32 (unshippable) | **0.7122** | **+0.0460** |
| `jina-reranker-v1-turbo-en` fp32 | 0.7018 | +0.0356 |
| `jina-reranker-v1-turbo-en` int8 | 0.7014 | +0.0353 |
| ColBERT-style MaxSim, `bge-small` | 0.6955 | +0.0293 |
| `mxbai-rerank-xsmall-v1` fp32 | 0.6415 | **−0.0246** |
| `mxbai-rerank-xsmall-v1` int8 | 0.6380 | −0.0282 |

**`bge-reranker-v2-m3` is the best reranker measured, and it is still
unshippable.** It scores +0.0460 — 29% more gain than
`jina-reranker-v1-turbo-en` and 57% more than ColBERT MaxSim. Its own
evaluation is the argument against it: scoring 5,000 pairs took **2,353 s at
471 ms/pair on ONNX Runtime**, and that is the fast backend with CGO. The
quality is real; it cannot be delivered inside an interactive call on a CPU.

This is the honest shape of the trade. Cutting the cross-encoder is not free,
and this document should not be read as claiming it is.

`bge-reranker-base` remains absent — its evaluation was lost to the
truncated-download incident below and not re-run once its 3493.30 ms/pair in
pure Go had excluded it. Its NDCG is **UNVERIFIED**.

**`mxbai-rerank-xsmall-v1` makes ranking worse than not reranking at all.** It
is 1.9x the parameters of `jina-reranker-v1-turbo-en`, 2.1x the pure-Go
latency, needs 39 ONNX op types instead of 27, and it costs 0.025 NDCG@10. This
is not a hypothetical failure mode; it is a published, actively downloaded
reranker. Do not use it. More generally: model size does not predict rerank
quality — the ordering here is 567.8M > 37.8M > 70.8M — and any reranker
adopted into CRED must be measured against the vector-only baseline on a
labelled set before it ships, not selected from a leaderboard.

The headroom that reranking is competing for is small. Recall@50 is 0.9350 and
Recall@10 is 0.8050, so the first stage has already found 93.5% of the relevant
documents; reranking can only reorder within that. The gap from the vector-only
baseline to a perfect NDCG@10 of 1.0 is 0.3338. The best reranker measured
closes 0.0460 of it (13.8%); the best CPU-feasible one closes 0.0356 (10.7%);
ColBERT MaxSim closes 0.0293 (8.8%).

### ColBERT-style late interaction

The alternative that reuses the already-verified model: keep per-token
embeddings from `bge-small-en-v1.5`, score a pair as the sum over query tokens
of the maximum dot product against any document token. No new model, no new
tokenizer, no new ONNX graph, no new CGO question.

```
baseline NDCG@10 0.6662
unique docs to encode: 2852
doc token vectors in 114s
storage: 529.1 MB fp16 for 2852 docs = 181.2 KB/doc (vs 0.75 KB/doc single
         vector) => 242x
ColBERT-style MaxSim over bge-small token vectors: NDCG@10=0.6955
         delta=+0.0293  [scoring only 0.05 ms/pair]
```

**+0.0293 NDCG@10 — 82% of the best CPU-feasible cross-encoder's gain, 64% of
`bge-reranker-v2-m3`'s — at 0.05 ms per pair.**
That is **2,468x** cheaper than `jina-reranker-v1-turbo-en` on `simplego`
(123.42 ms/pair over 50 candidates, clean) and **480x** cheaper than it on ONNX
Runtime (24 ms/pair). Fifty candidates score in **2.5 ms** against 6,171 ms —
inside the recall budget rather than two orders of magnitude over it.

One caveat that belongs next to the number rather than in a footnote:
`bge-small-en-v1.5` was **not trained for late interaction**. ColBERT models are
trained with a MaxSim objective; this is MaxSim applied opportunistically to
token vectors from a model trained for CLS pooling. That it recovers most of
the cross-encoder gain anyway is the result, but it means the behaviour has no
guarantee behind it beyond this measurement, and it may not hold on a different
corpus. A purpose-trained late-interaction model would be the principled
version and would reopen the tokenizer and inference questions this option
otherwise avoids.

The cost is storage, and it is large: **242x per document**, 181 KB against
0.75 KB. On a 100,000-chunk repository that is ~17 GB of token vectors against
~73 MB of single vectors. Three mitigations are plausible and **none was tested
here** — storing token vectors only for a hot subset, quantizing the token
vectors to int8 or binary, and capping stored tokens per chunk. The design cost
is also that `tech-decisions.md`'s `halfvec` schema gains a second, much larger
vector table with its own lifecycle.

---

## Falsified along the way

**FALSIFIED: "`BAAI/bge-reranker-base`'s `tokenizer.json` is corrupt."** The
first evaluation run reported
`FAILED Exception: EOF while parsing a list at line 322241 column 22`, and the
same file failed Python's `json.load`. It was a truncated download — a
parallel `curl` loop was killed by a 2-minute command timeout mid-file, leaving
5,533,696 of 17,098,107 bytes with no error anywhere. Re-fetched and verified
against `x-linked-size`, the file parses.

Recorded because the failure mode is worth internalizing: a truncated model or
tokenizer download produces a plausible-looking parse error attributable to the
upstream project. **Size-check every downloaded artifact against
`x-linked-size` before concluding anything about it.** All other files used
here were checked this way — `tok_tokenizer.json` 17,098,273 bytes,
`cand/base/model.onnx` 1,112,459,588, `cand/jina/model.onnx` 151,296,975,
`cand/mxbai/model.onnx` 284,189,709, all matching.

---

## Results: CGO and portability

**VERIFIED.** The reranker stack adds no CGO. Checked with `go list -deps` on
the real build command, per D-008's rule, not with `nm | grep cgo`:

```
$ CGO_ENABLED=0 go list -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' -deps ./...
=== packages in build graph using cgo (goinf: onnx-gomlx + simplego) ===
(empty = none)
total packages: 218

=== cross-compile CGO_ENABLED=0 ===
  OK   linux/amd64  9992 KiB
  OK   linux/arm64  9088 KiB
  OK   darwin/arm64  9357 KiB
  OK   windows/amd64  10240 KiB
/tmp/rrk_linux-amd64: ELF 64-bit LSB executable, x86-64, version 1 (SYSV),
statically linked, ... stripped
```

`sugarme/tokenizer` is likewise CGO-free across 218 packages. **Packaging was
never the thing that broke here.** Everything in this spike compiles static and
cross-compiles clean; it is latency, memory, and tokenizer correctness that
fail.

---

## What is unverified, and why

- **UNVERIFIED: quality on code retrieval.** SciFact is scientific claim
  verification. It was chosen because it is public, labelled, small enough to
  run honestly, and closer to CRED's claim-checking shape than a web-search
  benchmark — but it is not code. `tech-decisions.md` already flags that MTEB
  is not evidence for code retrieval, and that objection applies to this
  evaluation too. The relative ordering of the methods is more likely to
  transfer than the absolute NDCG values. Checking it needs a labelled
  code-retrieval set (CoIR or CodeSearchNet) run through the same harness.
- **UNVERIFIED: 100 queries is a small sample.** The full SciFact test split is
  300. The sample was fixed by seed and used identically for every method, so
  the comparison is paired, but the confidence interval on any single NDCG
  figure is wide. No significance test was run. The `mxbai` and ColBERT results
  are the ones most worth re-running at n=300.
- **UNVERIFIED: whether `onnx-gomlx`'s `QuantizedDense` fusion fired.** The
  int32-widening generic path is confirmed from source; the fusion pass that
  might bypass it was linked into every binary but was not instrumented.
- **UNVERIFIED: ColBERT token-vector compression.** The 242x storage figure is
  measured at fp16 with no compression. Whether int8 or binary quantization of
  token vectors preserves the +0.0293 was not tested, and it is the difference
  between a ~17 GB and a low-single-digit-GB index on a 100k-chunk repository.
- **UNVERIFIED: `bge-reranker-base` quality.** Its evaluation run was lost to
  the truncated-tokenizer incident above and not re-run, because its measured
  3493.30 ms/pair in pure Go had already excluded it. Its NDCG is unknown.
- **UNVERIFIED: non-Apple-Silicon hardware.** All timings are one M1 Pro, as in
  D-008. Ratios travel better than absolutes.
- **UNVERIFIED: LLM reranking quality and cost.** Fallback (d) below is
  reasoned about, not measured.
- **UNVERIFIED: hosted reranking quality.** No hosted API was called. Fallback
  (c) is argued on packaging and privacy grounds only.
- **UNVERIFIED: ColBERT MaxSim on a purpose-trained model.** The +0.0293 comes
  from applying MaxSim to a CLS-pooled model's token vectors. A model trained
  with a late-interaction objective was not tested.

---

## Fallback analysis

No pure-Go cross-encoder is viable. The four options, with costs.

### (a) Drop cross-encoder reranking from v1

**Retrieval quality lost: 0.0460 NDCG@10 on SciFact** against the best
reranker measured, or 0.0356 against the best one that could plausibly run on a
CPU — the full gain, since nothing replaces it. Recall is untouched: reranking
reorders a fixed candidate set, so Recall@50 stays 0.9350.

This is the option's real cost and it is not negligible: it is roughly a
seventh of the distance from the current baseline to a perfect NDCG@10.

The Matryoshka `halfvec` two-stage approach in `tech-decisions.md` is *not* a
substitute and should not be described as one. It is a **cost** optimization —
truncate to 768 or 1024 dimensions for the HNSW-indexed scan, then rescore the
survivors at full width. Full-width exact rescoring recovers the precision lost
to truncation. It cannot exceed the model's own single-vector ceiling, which is
the 0.6662 baseline.

Honest framing: two-stage `halfvec` retrieval gets CRED to vector-only quality
efficiently. Whether 0.6662 is good enough is a product question that this
spike cannot answer, and it is the same question CRED would face if reranking
worked and simply were not enabled.

**Viable, and the cost is bounded and known.**

### (b) Reranking behind the D-008 accelerated build tag

Reuses machinery that already exists — no new architectural concept, and D-008
already accepted the ORT escape hatch for bulk ingestion.

It does not work here, for a reason specific to this workload. The ingestion
case put a *batch* job behind the tag: slow is annoying, users wait, the
default path stays correct. Reranking is on the **interactive** path, so the
build tag becomes a user-visible quality difference between two images — the
`:fast` image returns better results than the default, silently. That is worse
than not shipping the feature, and it contradicts D-008's own condition that
"the slow path must remain correct and complete, never a degraded mode."

And the numbers do not clear the bar even with the tag on. ORT gives 24 ms/pair
for `jina-reranker-v1-turbo-en`, so 50 candidates is 1.2 s — tolerable — but
`bge-reranker-v2-m3`, the model `tech-decisions.md` names, is 44 s for the same
work *with* ORT, and 17.6 s even quantized to int8. The build tag can therefore
only ship the small model, whose entire advantage over ColBERT MaxSim is 0.0063
NDCG@10, for the cost of glibc, QEMU, and a second image.

**Rejected. Not because the mechanism is wrong, but because it buys 0.0063
NDCG@10 for the packaging strategy.**

### (c) Hosted reranking API

Cheapest to build, and the only option not bounded by what a CPU can run
locally. The `bge-reranker-v2-m3` result gives this option a measured floor
rather than a hopeful one: a large cross-encoder is worth **+0.0460 NDCG@10**
here, 57% more than ColBERT MaxSim — and a hosted service is not constrained by
the CPU budget that makes that model unusable locally. No hosted reranker was
called in this spike, so any specific vendor's quality and latency are
**UNVERIFIED**. `tech-decisions.md` already establishes the pattern for
embeddings: both hosted and local supported, neither mandatory.

The cost is the one the positioning cannot absorb if it becomes the default.
Air-gapped operation is the sovereignty argument, and a recall path that
degrades without network access is a recall path with two behaviours. It also
sends query text and candidate documents — the retrieved evidence, which is the
sensitive part — to a third party on every call.

**Acceptable as an explicitly opt-in enhancement, on the same footing as hosted
embeddings. Never the default, and recall must be complete without it.**

### (d) LLM-based reranking

CRED already depends on an LLM, so there is no new dependency, no new model
file, no tokenizer problem, and no CGO question. Its ranking quality was not
measured here and is **UNVERIFIED**.

Costs, none measured here: latency is one additional LLM round trip on the
recall path, likely 1–5 s, which is the same order as the pure-Go cross-encoder
this spike rejected; it consumes context budget that the calling agent needs;
it is non-deterministic, so the same query can return different orderings; and
it inherits whatever the user's configured model is, making retrieval quality
vary by deployment in a way that is hard to test or support.

The determinism objection is the sharp one for CRED specifically. "A claim
lives only while its evidence does" implies reproducible evidence retrieval,
and a reranker that reorders differently across identical calls undermines
that.

**Not recommended for v1. Worth a spike of its own if vector-only quality
proves insufficient, and it should be measured on the same harness and the
same labelled set as this spike.**

### (e) ColBERT-style late interaction — the option the evidence favours

Not in the original list; it emerged as the strongest candidate.

- **+0.0293 NDCG@10** — 82% of the best CPU-feasible cross-encoder's gain,
  64% of `bge-reranker-v2-m3`'s.
- **0.05 ms/pair.** 50 candidates in 2.5 ms, versus 6,171 ms for the fastest
  pure-Go cross-encoder on the same 50 candidates at seq 128.
- **No new model, no new tokenizer, no new ONNX graph.** It reuses
  `bge-small-en-v1.5` and the WordPiece tokenizer D-008 already verified at
  242,247/242,247. The entire tokenizer risk documented above disappears.
- MaxSim is a matmul and a row-wise max. It can be a `pgvector` query or ~30
  lines of Go; it does not need `simplego` at all.

The cost is storage: **242x per document**, measured, uncompressed. That is the
whole trade, and it is a schema decision that must be made before v1 ships
rather than retrofitted — token vectors have to be written at ingest time or
the corpus needs re-embedding.

---

## Verdict

> **Cut cross-encoder reranking from v1. Replace `bge-reranker-v2-m3` with
> ColBERT-style MaxSim late interaction over the `bge-small-en-v1.5` token
> vectors CRED already produces.**
>
> `bge-reranker-v2-m3` runs in pure Go — no missing op, no OOM — and is
> unshippable regardless: **9.3 s per pair at sequence 512, 3.60 GB peak RSS**,
> and **882 ms per pair even on ONNX Runtime with CGO**. Fifty candidates cost
> **7.8 minutes** in Go and **44 seconds** with the escape hatch. This is a CPU
> feasibility failure, not a Go failure, and `tech-decisions.md`'s model choice
> is wrong independent of language.
>
> The smallest credible substitute, `jina-reranker-v1-turbo-en` at 37.8M
> parameters, needs **2.5 s to rerank 20 candidates at sequence 128** in pure
> Go — the most favourable configuration measured — and **53 s for 50 at
> sequence 512**, against 51 ms to embed the query. 49x to 1,038x over budget,
> with no batching or sequence-length trade that recovers it.
>
> ColBERT MaxSim delivers **+0.0293 NDCG@10 at 0.05 ms/pair**, using a model
> and a tokenizer that are already verified. It costs **242x storage per
> document**, measured and uncompressed, and that is the decision to argue
> about.
>
> **The cut is not free, and this verdict does not pretend otherwise.**
> `bge-reranker-v2-m3` is genuinely the best reranker measured (+0.0460), and
> ColBERT MaxSim gives up **0.0167 NDCG@10** against it and 0.0063 against
> `jina-reranker-v1-turbo-en`. That is the price of a CPU-only, air-gapped,
> statically linked deployment. It is a price worth naming rather than
> obscuring — and the moment CRED targets a GPU, this analysis should be
> reopened, because the quality is there if the compute is.

Four findings that must not be lost:

1. **A reranker can make ranking worse.** `mxbai-rerank-xsmall-v1` scored
   **−0.0246 NDCG@10** against no reranking, while being 1.9x larger and 2.1x
   slower than `jina-reranker-v1-turbo-en`, which scored +0.0356. Any reranker
   adopted into CRED must beat the vector-only baseline on a labelled set
   before it ships.
2. **int8 is a memory lever, not a latency lever — in `simplego` it is
   negative.** 1.69x to 2.28x slower across four models, while the same
   quantized graphs run 1.72x to **2.48x faster** on ONNX Runtime. It does cut
   RSS 1.4–2.2x at a 0.0004 NDCG cost. D-008's expectation that int8 would
   narrow the ~10x gap is **FALSIFIED** — it widens it.
3. **The SentencePiece tokenizer is a much bigger problem than WordPiece was,
   and the D-008 mitigation does not scale to it.** Probing enumerates a
   per-codepoint predicate; it cannot enumerate a 237 KB string-to-string Darts
   trie. `sugarme/tokenizer` scores 96.22% with 948 panics and fails **100% of
   inputs containing two consecutive spaces** — which is all source code — and
   silently ignores truncation on the pair path a cross-encoder needs.
   `eliben/go-sentencepiece` cannot load the model at all.
4. **Op coverage was never the constraint.** Every candidate ran, including
   DeBERTa-v2's 39 op types. `simplego`'s limit is throughput and memory, not
   completeness. Future model-selection spikes should measure latency and RSS
   first and stop worrying about unsupported ops.

---

## What would change this verdict

- **Measured token-vector compression.** If int8 or binary quantization of
  ColBERT token vectors preserves the +0.0293 at 8–32x less storage, the only
  real objection to (e) mostly disappears. This is the highest-value follow-up.
- **A code-retrieval evaluation that widens the gap.** SciFact is not code. If
  a cross-encoder's advantage over MaxSim is much larger on code than the
  0.0063–0.0167 measured here, option (b) or (c) gets re-argued with a real
  number behind it.
- **Vector-only quality proving insufficient in use.** The trigger for
  revisiting (c) hosted reranking or (d) LLM reranking. Both should be measured
  on this harness before either is adopted.
- **`simplego` gaining int8 GEMM kernels.** It would flip finding 2 and cut
  cross-encoder latency by roughly the 1.7x ORT sees, which is still ~50x short
  of what reranking needs. It changes the ingestion story from D-008 far more
  than it changes this one.
- **A conformant pure-Go Unigram tokenizer appearing, or being written.** It
  would not change this verdict on its own — the latency is the binding
  constraint — but it removes the second blocker if latency is ever solved.
- **Deployment moving off CPU.** Every number here is CPU-only. A GPU target
  makes `bge-reranker-v2-m3` viable immediately and makes this entire analysis
  inapplicable.

---

## Environment

| | |
|---|---|
| Machine | Apple M1 Pro, 10 cores, 16 GB, darwin/arm64, macOS 26.5.2 |
| Go | 1.26.0 |
| Python | 3.13.7 |
| `transformers` / `tokenizers` | 5.14.1 / 0.22.2 |
| `onnxruntime` / `numpy` | 1.27.0 / 2.5.1 |
| `torch` / `optimum` / `onnx` | 2.13.0 / 2.2.0 / 1.22.0 |
| `gomlx` / `onnx-gomlx` | v0.27.3 / v0.4.2 |
| `sugarme/tokenizer` | v0.3.0 |
| `eliben/go-sentencepiece` | v0.7.0 |
| `gomlx/go-huggingface` | v0.3.5 |
| Models | `BAAI/bge-reranker-v2-m3` (locally exported ONNX), `BAAI/bge-reranker-base`, `jinaai/jina-reranker-v1-turbo-en`, `mixedbread-ai/mxbai-rerank-xsmall-v1`, `BAAI/bge-small-en-v1.5` |
| Evaluation set | BEIR SciFact — 5,183 documents, 300 test queries, 100 sampled |
