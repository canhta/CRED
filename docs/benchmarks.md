# Benchmarks and verified results

Reproduced in this repository, not carried over from the spikes. Each figure is
stated with its conditions, because a prior measurement on this project was taken
under CPU contention and had to be retracted.

## Correctness

**Tokenizer: 212,291 of 212,291 inputs match the reference exactly.** 43 curated
edge cases, 17,688 fuzz strings, and 194,560 single-codepoint probes, all diffed
against HuggingFace `tokenizers` 0.22.2. The character-class tables are generated
by probing that pinned release — never from Go's `unicode` package, because the
goal is byte-identity with the tokenizer that trained the model, not Unicode
correctness. Regenerate with `go generate ./internal/embed/wordpiece/...`.

**Embeddings match the recorded reference vector.** The CLS-pooled, normalized
embedding agrees with the value in
[the spike](research/spikes/go-embeddings-tokenizer.md) — itself cross-checked
against ONNX Runtime at cosine 1.00000000 — to within 1e-6.

**`depguard` genuinely fails.** A `pgx` import in `internal/temporal`,
`internal/acl`, or `internal/recall`, and a `database/sql` import in
`internal/temporal`, were each added and confirmed to fail the build, then
reverted.

**The schema applies to a real PostgreSQL 17 + pgvector 0.8.5.** Including the
`halfvec` column partitioned by model, the per-partition
`hnsw ((embedding::halfvec(384)) halfvec_ip_ops)` expression index, and the norm
`CHECK`.

## Recall latency

**Conditions.** Apple M1 Pro, 10 cores, 16 GB, darwin/arm64, Go 1.26.0,
`CGO_ENABLED=0`. PostgreSQL 17 + pgvector 0.8.5 in Docker Desktop on the same
machine. 1,247 claims seeded from this repository's own documentation. Eight
distinct queries × 10 rounds = 80 measurements, in-process, after a warm-up pass
so the figures exclude one-off graph compilation. Load average 2.5–3.6 at the
start and end of the run; **not a fully idle machine**, and the Docker VM is part
of what is measured.

| Stage | median | p95 | p99 | max |
|---|---|---|---|---|
| **total** | **123.5 ms** | **126.7 ms** | 127.3 ms | 127.4 ms |
| embed | 116.1 ms | 119.1 ms | 120.2 ms | 120.3 ms |
| dense (pgvector) | 1.7 ms | 5.4 ms | 6.4 ms | 6.5 ms |
| lexical (full-text) | 0.6 ms | 1.2 ms | 1.3 ms | 1.4 ms |

That sits inside Mem0's stated 150–200 ms comfort band and well under Zep's
published 576 ms p95 — though Zep's figure is a cloud call over a network at
concurrency 20 with a cross-encoder, so it is a reference point rather than a
like-for-like comparison.

**Embedding is 94% of it, and the retrieval CRED actually does is nearly free.**
Both database arms together are under 3 ms at the median against 1,247 claims.

### A correction to the embeddings spike

`go-embeddings-tokenizer.md` concludes *"Interactive recall is unaffected (51 ms
to embed a query)"*. That figure comes from its batch-8 latency table, and it
does not hold for a single query, which is what recall actually issues.

Measured on this machine, same model, same backend:

| Configuration | per text |
|---|---|
| seq 16, batch 8 | 26.3 ms |
| seq 16, **batch 1** | **116.1 ms** |
| seq 256, batch 8 | 503.5 ms |
| seq 256, batch 1 | 522.3 ms |

The batch-8 numbers reproduce the spike closely, so the model path is confirmed.
What is new is roughly **100 ms of fixed per-execution overhead** in the
`simplego` backend, independent of batch size — amortized away at batch 8,
the entire cost of a short query at batch 1. It is not fixed here; it is recorded
because it is the single largest available win on the recall path: eliminating it
would take recall from ~123 ms to ~25 ms.

## Seeding throughput

Seeding this repository — 40 files, 1,247 chunks — took **25 m 38 s**, or roughly
1.23 s per chunk, on the machine above **while the test suite, the linter and
Docker were also running**. Treat it as an upper bound rather than a clean
measurement.

This is the named cost of the pure-Go forward pass (9–16x slower than ONNX
Runtime): interactive recall is unaffected, and bulk ingestion is where it hurts.
The accepted answer is a build-tagged ONNX Runtime variant behind the existing
`Embedder` interface, not yet built.
