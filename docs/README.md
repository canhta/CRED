# CRED Documentation

CRED is evidence-governed memory for AI agents. A claim lives only while its
evidence does.

## Start here

- **[Research synthesis](research/synthesis.md)** — what the discovery phase
  found, what it killed, and what survives. Read this first.
- [Decision log](research/decision-log.md) — decisions made, with reasoning and
  what each rules out.
- **[Demand test](research/demand-test.md)** — the instrument for the cheapest
  disconfirmation available: whether anyone actually wants this.

## Product

- **[Product requirements](product/prd.md)** — what to build, and the laws it
  must not violate.
- [Design advantages](research/design-advantages.md) — what competitors got
  wrong, and what it costs to avoid it.

## Implementation

- **[v0 harness](../v0/README.md)** — the built experiment, its stages, and what
  an operator must supply to start it.
- **[v0 experiment design](research/spikes/v0-experiment-design.md)** — the
  experiment that runs before any product code, its pre-registered kill
  threshold, and why the existing benchmark does not settle the question.
- [Go embeddings and tokenizer](research/spikes/go-embeddings-tokenizer.md) —
  the spike that decides whether the product can be written in Go at all.
- [Go reranker](research/spikes/go-reranker.md) — why cross-encoder reranking is
  cut from v1, and what replaces it.
- [Technical decisions](research/spikes/tech-decisions.md) — language,
  datastore, queue, and the failure modes each choice implies.
- [Language and MCP](research/spikes/tech-language-and-mcp.md) — runtime
  selection and protocol conformance.
- [Testing strategy](research/spikes/testing-strategy.md) — invariants,
  adversarial cases, and the shape of the suite.
- [Packaging and first run](research/spikes/packaging-and-first-run.md) —
  distribution, release automation, and the three-minute first run.

## Evidence

Competitive and market evidence, with claims tied to file references or cited
URLs. Repository scans read actual source code rather than documentation.

### Repository code scans

| Document | Subject | Size |
|---|---|---|
| [graphiti.md](research/evidence/graphiti.md) | Temporal knowledge graph, provenance, invalidation | 1,267 lines |
| [onyx.md](research/evidence/onyx.md) | Enterprise search, ACL model, OSS/EE boundary | 495 lines |
| [langfuse.md](research/evidence/langfuse.md) | Tracing, prompt versioning, evaluation | 593 lines |
| [ragflow.md](research/evidence/ragflow.md) | Document parsing, citation model | 434 lines |
| [context7.md](research/evidence/context7.md) | The most-installed MCP server: how it won distribution | 737 lines |
| [eval-methodology-prior-art.md](research/evidence/eval-methodology-prior-art.md) | Model config, baselines, judges, cost accounting across memory evals | 648 lines |
| [mem0.md](research/evidence/mem0.md) | Memory layer, write path, governance | 425 lines |
| [letta.md](research/evidence/letta.md) | Context hierarchy, memory blocks, agent state | 330 lines |

### Market and prior art

| Document | Subject |
|---|---|
| [spike-memco.md](research/evidence/spike-memco.md) | Memco: product surface, pricing, on-prem claims |
| [spike-glen.md](research/evidence/spike-glen.md) | Glen: product surface, pricing, team |
| [spike-competitors-sweep.md](research/evidence/spike-competitors-sweep.md) | Every org-level player; where the permission gap is |
| [spike-demand-and-buyers.md](research/evidence/spike-demand-and-buyers.md) | Demand evidence, pricing benchmarks, buyer |
| [memco-spark-paper.md](research/evidence/memco-spark-paper.md) | Teardown of the competitor's published architecture |
| [market-landscape.md](research/evidence/market-landscape.md) | Competitors, wedge ranking, kill criteria (149 cited URLs) |
| [graveyard-and-crowding.md](research/evidence/graveyard-and-crowding.md) | Failed analogues, commoditization of context |
| [knowledge-decay-and-review-capacity.md](research/evidence/knowledge-decay-and-review-capacity.md) | Freshness vs findability, review capacity limits |
| [prior-art-voluntary-curation.md](research/evidence/prior-art-voluntary-curation.md) | Whether voluntary curation sustains |
| [adjacent-devtools-status.md](research/evidence/adjacent-devtools-status.md) | Adjacent company status scan |

## Reading the evidence

Four documents carry **provenance warnings** — they arrived from background
tasks outside the commissioned research and are retained for relevance, not
because they were independently commissioned.

**One research agent fabricated citations before self-retracting.** Load-bearing
numbers were re-verified independently, but figures not explicitly marked as
verified should be spot-checked before any external use. Each document lists its
own unverified and falsified claims explicitly.
