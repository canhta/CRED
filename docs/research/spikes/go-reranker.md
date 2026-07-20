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
| `bge-reranker-v2-m3` in pure Go | **Rejected.** Runs, but 8.6 s per pair at seq 512 |
| `bge-reranker-v2-m3` at all, any backend | **Rejected on CPU.** 1.8 s per pair on ONNX Runtime |
| Cross-encoder reranking at v1 | **Cut.** No candidate fits an interactive budget in pure Go |
| Replacement | **ColBERT-style MaxSim** over `bge-small-en-v1.5` token vectors |
| int8 in `simplego` | **Do not ship for latency.** 2.0–2.1x *slower*. Ships 2.6x less RSS |
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
executes all 26 op types in the graph and returns logits:

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

The full sweep, against ONNX Runtime on the identical graph:

| batch | seq | Go `simplego` per pair | Go peak RSS | ORT per pair | Ratio |
|---|---|---|---|---|---|
| 1 | 128 | 2369.51 ms | 3.44 GB | 473.35 ms | 5.0x |
| 8 | 128 | 3112.68 ms | 2.74 GB | 413.62 ms | 7.5x |
| 1 | 256 | 6816.06 ms | 3.59 GB | 1032.98 ms | 6.6x |
| 8 | 256 | 4268.17 ms | 4.45 GB | 754.70 ms | 5.7x |
| 1 | 512 | 8635.13 ms | 4.58 GB | 1836.34 ms | 4.7x |
| 8 | 512 | 11932.61 ms | 4.19 GB | 1346.09 ms | 8.9x |

Raw output:

```
### m3 fp32 batch=1 seq=512
RESULT batch=1 seq=512 avg=8.635128313s per_pair=8635.13ms
          4580147200  maximum resident set size
### m3 fp32 batch=8 seq=512
RESULT batch=8 seq=512 avg=1m35.460862417s per_pair=11932.61ms
          4188815360  maximum resident set size
```

```
$ ./venv/bin/python ort_bench.py onnx-m3/model.onnx
inputs ['input_ids', 'attention_mask'] outputs ['logits']
ORT batch=1 seq=128 avg=473.3ms per_pair=473.35ms
ORT batch=8 seq=128 avg=3308.9ms per_pair=413.62ms
ORT batch=1 seq=256 avg=1033.0ms per_pair=1032.98ms
ORT batch=8 seq=256 avg=6037.6ms per_pair=754.70ms
ORT batch=1 seq=512 avg=1836.3ms per_pair=1836.34ms
ORT batch=8 seq=512 avg=10768.7ms per_pair=1346.09ms
```

Three things follow, and the second is the one that matters most.

**The Go/ORT gap narrowed, not widened.** 4.7–8.9x here against 9–16x for the
embedder. Larger matmuls suit `simplego`'s kernels better. This is the one
result in the spike that is better than D-008 predicted, and it is irrelevant,
because of the next point.

**`bge-reranker-v2-m3` is not viable on CPU at all, in any language.** Fifty
pairs at sequence 512 costs **67 seconds on ONNX Runtime with CGO** and **~10
minutes in pure Go**. The build-tagged ORT escape hatch that D-008 established
for bulk ingestion does not rescue this: it converts an impossible latency into
a different impossible latency. The model choice in `tech-decisions.md` is
wrong for a CPU-only deployment independent of the Go question.

**Memory exceeds the deployment floor before anything else runs.** Peak RSS is
2.74–4.58 GB for a *single* rerank batch, against the ~2 GB floor documented for
CRED. Even at batch 1, seq 128, it is 3.44 GB.

---

## Results: substitutes on the same harness

All four candidates load and execute in `simplego` — including
`mxbai-rerank-xsmall-v1`, whose DeBERTa-v2 disentangled attention needs 39 op
types (`GatherElements`, `Tile`, `Sign`, `Ceil`, `LessOrEqual`). **Op coverage
was never the constraint.** Latency and memory are.

Pure Go, `simplego`, batch 1, sequence 512:

| Model | Params | Arch | per pair | peak RSS |
|---|---|---|---|---|
| `bge-reranker-v2-m3` | 567.8M | XLM-R-large, 24L/1024 | 8635.13 ms | 4.58 GB |
| `bge-reranker-base` | 278.0M | XLM-R-base, 12L/768 | 4671.91 ms | 2.65 GB |
| `mxbai-rerank-xsmall-v1` | 70.8M | DeBERTa-v2, 12L/384 | 2280.50 ms | 1.37 GB |
| `mxbai-rerank-xsmall-v1` int8 | — | — | 3869.82 ms | 818 MB |
| `jina-reranker-v1-turbo-en` | 37.8M | JinaBERT, 6L/384 | **1073.46 ms** | **793 MB** |
| `jina-reranker-v1-turbo-en` int8 | — | — | 2059.45 ms | 488 MB |

The fastest candidate, swept over the operating range:

```
jina fp32 b=1  s=128: RESULT batch=1  seq=128 avg=176.764187ms  per_pair=176.76ms
jina fp32 b=8  s=128: RESULT batch=8  seq=128 avg=1.53412825s   per_pair=191.77ms
jina fp32 b=25 s=128: RESULT batch=25 seq=128 avg=4.72006177s   per_pair=188.80ms
jina fp32 b=1  s=256: RESULT batch=1  seq=256 avg=551.830896ms  per_pair=551.83ms
jina fp32 b=8  s=256: RESULT batch=8  seq=256 avg=3.426224354s  per_pair=428.28ms
jina fp32 b=25 s=256: RESULT batch=25 seq=256 avg=7.144374812s  per_pair=285.77ms
```

And at the actual unit of work — one query's worth of candidates, wall clock:

```
=== pure-Go simplego, jina-reranker-v1-turbo-en, realistic rerank workloads ===
N=20 pairs seq=128: RESULT batch=20 seq=128 avg=5.172552041s  per_pair=258.63ms
N=20 pairs seq=256: RESULT batch=20 seq=256 avg=7.638135021s  per_pair=381.91ms
N=20 pairs seq=512: RESULT batch=20 seq=512 avg=19.694441833s per_pair=984.72ms
N=50 pairs seq=128: RESULT batch=50 seq=128 avg=14.603065103s per_pair=292.06ms
N=50 pairs seq=256: RESULT batch=50 seq=256 avg=22.663122458s per_pair=453.26ms
N=50 pairs seq=512: RESULT batch=50 seq=512 avg=1m5.643578208s per_pair=1312.87ms
```

**5.2 seconds to rerank 20 candidates at sequence 128 — the most favourable
configuration of the smallest credible model.** At the sequence length a code
chunk actually needs, 19.7 seconds for 20 candidates. At the top of the range
`tech-decisions.md` specifies, 50 candidates, it is 14.6 s at sequence 128,
22.7 s at sequence 256, and 65.6 s at sequence 512.

For scale: D-008 measured 51 ms to embed a query. Reranking would cost between
**101x** (20 candidates, seq 128) and **1,287x** (50 candidates, seq 512) the
retrieval step it is meant to refine, on the interactive path. No batching or
sequence-length trade recovers two to three orders of magnitude.

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
| Go `simplego` | fp32 | 331.38 ms | 610 MB |
| Go `simplego` | **int8** | **680.23 ms** | **231 MB** |
| ONNX Runtime | fp32 | 32.33 ms | — |
| ONNX Runtime | **int8** | **18.82 ms** | — |

```
### bge-small model batch=8 seq=128
RESULT batch=8 seq=128 avg=2.651000986s per_pair=331.38ms
           610238464  maximum resident set size
### bge-small model_int8 batch=8 seq=128
RESULT batch=8 seq=128 avg=5.441832764s per_pair=680.23ms
           231342080  maximum resident set size

ORT small/model.onnx      batch=8 seq=128 avg=258.6ms per_text=32.33ms
ORT small/model_int8.onnx batch=8 seq=128 avg=150.5ms per_text=18.82ms
```

int8 makes ONNX Runtime **1.72x faster** and `simplego` **2.05x slower**. The
same inversion holds on both rerankers that ship a quantized ONNX:
`jina-reranker-v1-turbo-en` 1073 ms → 2059 ms (1.92x slower),
`mxbai-rerank-xsmall-v1` 2280 ms → 3870 ms (1.70x slower). Three models, one
direction.

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

- **The memory result is real and useful.** 610 MB → 231 MB is a 2.6x cut in
  the single largest consumer against CRED's ~2 GB floor. If the constraint is
  memory rather than latency, int8 is the answer.
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
jina:      NDCG@10=0.7018 delta=+0.0356 [5000 pairs,  121s, 24 ms/pair on ORT]
jina_int8: NDCG@10=0.7014 delta=+0.0353 [5000 pairs,   61s, 12 ms/pair on ORT]
mxbai:     NDCG@10=0.6415 delta=-0.0246 [5000 pairs,  262s, 52 ms/pair on ORT]
mxbai_q:   NDCG@10=0.6380 delta=-0.0282 [5000 pairs,  166s, 33 ms/pair on ORT]
```

| Method | NDCG@10 | vs vector-only |
|---|---|---|
| Vector only (`bge-small-en-v1.5`) | 0.6662 | — |
| `jina-reranker-v1-turbo-en` fp32 | **0.7018** | **+0.0356** |
| `jina-reranker-v1-turbo-en` int8 | 0.7014 | +0.0353 |
| ColBERT-style MaxSim, `bge-small` | 0.6955 | +0.0293 |
| `mxbai-rerank-xsmall-v1` fp32 | 0.6415 | **−0.0246** |
| `mxbai-rerank-xsmall-v1` int8 | 0.6380 | −0.0282 |

`bge-reranker-v2-m3` and `bge-reranker-base` are absent from this table by
design: both had already been excluded on latency by an order of magnitude
before quality mattered. Scoring 5,000 pairs with `bge-reranker-v2-m3` costs
over an hour on ONNX Runtime alone, which is itself the finding. Their NDCG is
**UNVERIFIED**. Nothing in the verdict depends on it — even if
`bge-reranker-v2-m3` scored substantially above `jina-reranker-v1-turbo-en`, it
cannot be run at interactive latency on a CPU in any language.

**`mxbai-rerank-xsmall-v1` makes ranking worse than not reranking at all.** It
is 1.9x the parameters of `jina-reranker-v1-turbo-en`, 2.1x the pure-Go
latency, needs 39 ONNX op types instead of 27, and it costs 0.025 NDCG@10. This
is not a hypothetical failure mode; it is a published, actively downloaded
reranker. Do not use it. More generally: model size does not
predict rerank quality, and any reranker adopted
into CRED must be measured against the vector-only baseline on a labelled set
before it ships, not selected from a leaderboard.

The headroom that reranking is competing for is small. Recall@50 is 0.9350 and
Recall@10 is 0.8050, so the first stage has already found 93.5% of the relevant
documents; reranking can only reorder within that. The gap from the vector-only
baseline to a perfect NDCG@10 of 1.0 is 0.3338, and the best cross-encoder
closes 0.0356 of it — about 11%.

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

**+0.0293 NDCG@10 — 82% of the cross-encoder's gain — at 0.05 ms per pair.**
That is **5,841x** cheaper than `jina-reranker-v1-turbo-en` on `simplego`
(292.06 ms/pair over 50 candidates) and **480x** cheaper than it on ONNX
Runtime (24 ms/pair). Fifty candidates score in **2.5 ms** against 14,603 ms —
inside the recall budget rather than two orders of magnitude over it.

One caveat that belongs next to the number rather than in a footnote:
`bge-small-en-v1.5` was **not trained for late interaction**. ColBERT models are
trained with a MaxSim objective; this is MaxSim applied opportunistically to
token vectors from a model trained for CLS pooling. That it recovers 82% of the
cross-encoder gain anyway is the result, but it means the behaviour has no
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
  4671.91 ms/pair in pure Go had already excluded it. Its NDCG is unknown.
- **UNVERIFIED: `bge-reranker-v2-m3` int8 latency.** The quantization itself
  succeeded — `quantize_dynamic` collapsed the 2.27 GB external-data model to a
  single 569,622,599-byte file in 30.2 s — but the resulting graph was not
  benchmarked within this spike. Given the three-model result that int8 is
  slower in `simplego`, and that the fp32 model is already 4.7x over budget on
  *ONNX Runtime with CGO*, int8 would have to deliver roughly 30x to change the
  conclusion. Nothing measured here suggests it can.
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

**Retrieval quality lost: 0.0356 NDCG@10 on SciFact** — the full measured
cross-encoder gain, since nothing replaces it. Recall is untouched: reranking
reorders a fixed candidate set, so Recall@50 stays 0.9350.

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
`bge-reranker-v2-m3`, the model `tech-decisions.md` names, is 67 s for the same
work *with* ORT. The build tag can only ship the small model, whose entire
advantage over ColBERT MaxSim is 0.0063 NDCG@10, for the cost of glibc, QEMU,
and a second image.

**Rejected. Not because the mechanism is wrong, but because it buys 0.0063
NDCG@10 for the packaging strategy.**

### (c) Hosted reranking API

Cheapest to build, and the only option not bounded by what a CPU can run
locally. No hosted reranker was benchmarked in this spike, so its quality
advantage over ColBERT MaxSim is **UNVERIFIED here**. `tech-decisions.md`
already establishes the pattern for embeddings: both hosted and local
supported, neither mandatory.

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

- **+0.0293 NDCG@10**, 82% of the cross-encoder gain.
- **0.05 ms/pair.** 50 candidates in 2.5 ms, versus 14,603 ms for the fastest
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
> unshippable regardless: **8.6 s per pair at sequence 512, 4.58 GB peak RSS**,
> and **1.8 s per pair even on ONNX Runtime with CGO**. Fifty candidates cost
> ~10 minutes in Go and 67 seconds with the escape hatch. This is a CPU
> feasibility failure, not a Go failure, and `tech-decisions.md`'s model choice
> is wrong independent of language.
>
> The smallest credible substitute, `jina-reranker-v1-turbo-en` at 37.8M
> parameters, needs **5.2 s to rerank 20 candidates at sequence 128** in pure
> Go — the most favourable configuration measured — and **14.6 s for 50**,
> against 51 ms to embed the query. Two orders of magnitude over budget, with
> no batching or sequence-length trade that recovers it.
>
> ColBERT MaxSim delivers **+0.0293 NDCG@10 against +0.0356 for the best
> cross-encoder — 82% of the gain — at 0.05 ms/pair**, using a model and a
> tokenizer that are already verified. It costs **242x storage per document**,
> measured and uncompressed, and that is the decision to argue about.

Four findings that must not be lost:

1. **A reranker can make ranking worse.** `mxbai-rerank-xsmall-v1` scored
   **−0.0246 NDCG@10** against no reranking, while being larger and slower than
   the model that scored +0.0356. Any reranker adopted into CRED must beat the
   vector-only baseline on a labelled set before it ships.
2. **int8 is a memory lever, not a latency lever — in `simplego` it is
   negative.** 2.05x slower on `bge-small`, 1.92x on `jina`, 1.70x on `mxbai`,
   while ONNX Runtime gets 1.72x *faster*. It does cut RSS 2.6x (610 → 231 MB)
   at a 0.0004 NDCG cost. D-008's expectation that int8 would narrow the 12x
   gap is **FALSIFIED**.
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
- **A code-retrieval evaluation that reverses the ordering.** SciFact is not
  code. If a cross-encoder's advantage over MaxSim is much larger on code than
  the 0.0063 measured here, option (b) or (c) gets re-argued with a real
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
