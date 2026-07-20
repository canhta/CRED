# Memco "Spark" — Paper Teardown

**Source:** Tablan, Taylor, Hurtado, Bernhem, Uhrenholt, Farei, Moilanen.
*"Smarter Together: Creating Agentic Communities of Practice through Shared
Experiential Learning."* arXiv:2511.08301v1, 11 Nov 2025.
The Memory Company (Memco) + Moonsong Labs. Local copy: `docs/2511.08301v1.pdf`

This is the **published architecture of the primary competitor** named in D-004.

---

## Verdict

- Spark's loop is **capture trace → curate → redistribute**, delivered as a
  **managed** service over **MCP**. Nothing in the paper is self-hosted.
- The headline result is real but **cannot be attributed to shared memory**:
  there is **no ablation** separating documentation retrieval from experiential
  memory.
- The experiential data is **synthetic and derived from the reference
  solutions** — the memory was built from the answers to the evaluation set.
- The paper is titled "communities of practice" but **evaluates no community**:
  no multi-agent contribution, no conflict, no poisoning, no permissions.
- **Every capability CRED proposes to differentiate on is unevaluated here.**
  That is the opening.

---

## What Spark is

Three subsystems:

1. **Knowledge base** — seeded by ingesting raw software documentation
   (~34,000 documentation blobs for 7 Python data-science libraries).
2. **Retrieval agent** — an 8-step workflow per query: analyse intent → plan
   search strategy → recall past experience → retrieve docs → generate
   recommendation → link sections → synthesise best practice → respond.
3. **Continuous learning meta-process** — curation, expansion, optimisation.

The continual-learning loop across "memory epochs":

```text
Initialization → Feedback Collection → Knowledge Extraction & Curation
              → Memory-augmented Learning → (repeat)
```

Integration surface is **MCP**. Curation is **fully managed inside Spark**,
"requiring no direct effort from any participating agents."

### Their framing of the problem

The paper opens on the same premise CRED did: traditional peer-to-peer developer
knowledge sharing is collapsing, agentic equivalents have not emerged. They cite
Stack Overflow down >75% in two years.

*(Our own research independently measured this at ~98% off peak with better
sourcing — see [knowledge-decay-and-review-capacity.md](knowledge-decay-and-review-capacity.md).)*

They explicitly position against both **provider-managed single-user memory**
(Anthropic, ChatGPT) and **user-managed single-user memory** (Letta, Zep, Mem0,
Cognee, ByteRover) — the same landscape CRED scanned.

---

## Reported results

DS-1000 benchmark (1,000 data-science problems sourced from Stack Overflow),
judged by Gemini 2.5 Pro on a 1–5 scale.

| Model | NO-SPARK | WITH-SPARK | Change |
|---|---|---|---|
| DS-1000 human reference | 4.28 | — | — |
| Qwen3-Coder-30B | 4.23 | 4.89 | **+0.66** |
| Claude Haiku 4.5 | 4.50 | 4.91 | +0.41 |
| GPT-5-Codex | 4.78 | 4.83 | **+0.05** |

Recommendation helpfulness (Claude Sonnet 3.7 judge): 76.1% "extremely
helpful", 98.2% at least "good".

---

## Six methodological weaknesses

These are the substance of the opportunity. Listed strongest first.

### 1. The experiential memory was built from the answers

Section 4.1: they generated synthetic experiential data by having GPT-4o produce
an initial solution, **then exposed it to the correct reference solution**, then
asked it to "generate realistic instructions which a user may issue to their
development agent to try to guide it towards the solution."

**The hints are derived from the reference answers to the evaluation set.** The
memory therefore contains distilled answers to the very problems being scored.
This is not leakage in the strict train/test sense, but it is close enough that
the measured lift cannot be read as evidence for organic experiential learning.

They acknowledge choosing an older model deliberately "to avoid (close to)
perfect solutions… which would materially limit the opportunities for learning."

### 2. No ablation — the central claim is unattributable

Table 1 shows WITH-SPARK enables **three capabilities simultaneously**:
documentation search, experiential memory, and curated knowledge.
NO-SPARK enables none.

There is no condition isolating documentation-only. Given the knowledge base was
seeded with 34,000 doc blobs covering **exactly the seven libraries DS-1000
tests**, the lift is fully consistent with ordinary RAG over documentation.

**The paper does not demonstrate that shared experiential memory contributes
anything.** This is the single most important gap.

### 3. One epoch

Section 4.1: "We use **one epoch** of experiential data ingestion before
evaluation."

The entire thesis is *collective continual learning that compounds across
epochs*. It was tested with one. **The compounding claim — the reason the
architecture exists — is unmeasured.**

### 4. LLM judge used where ground truth was available

The paper itself states DS-1000 includes "execution-based test cases for
functional correctness" and "surface-form constraints."

They used **neither**. No pass@1, no execution results — only a 1–5 LLM judge
score.

This matters directionally: Spark injects best-practice recommendations, and the
judge rewards code that *looks* like best practice (idiomatic, complete,
well-structured). The judge and the treatment are aligned on the same axis.
Execution pass rates would have been an independent check and were available.

### 5. The strongest model gains nothing

GPT-5-Codex: **+0.05**, indistinguishable from noise. The headline "30B model
matches SOTA" reduces to: *a weak model given answer-derived hints approaches a
strong model.*

Their own discussion concedes the results "provide a lower bound" and speculates
the effect would be larger on proprietary or novel code — which is precisely the
setting they did not test.

### 6. No community was evaluated

The title claims "agentic communities of practice." The evaluation contains:

- no multiple contributing agents
- no conflicting contributions
- no poisoning or bad-write scenario
- no permissions, tenancy, or scoping
- no staleness or invalidation test
- no cost or latency measurement (despite an 8-step agentic workflow per query)

**Every dimension that makes shared memory hard is absent from the evaluation.**

---

## What this means for CRED

### It validates the direction

Memco raised money and published a paper on this exact thesis. The problem
framing, the landscape, and the MCP integration surface all match D-004. CRED is
not inventing a category.

### It reveals the opening

Spark is a **managed** service. The paper never mentions self-hosting, data
residency, tenancy, or access control. **The sovereignty bet in D-004 is aimed at
the one axis their published architecture does not address.**

### It hands over the evaluation bar — and how to clear it honestly

CRED can beat this on methodology at low cost:

| Spark | CRED should |
|---|---|
| Synthetic hints derived from answers | Real traces, or synthetic hints generated **without** sight of the reference solution |
| No ablation | Ablate: docs-only vs docs+experiential vs experiential-only |
| One epoch | Report the **epoch curve** — does quality compound or plateau? |
| LLM judge only | Report **execution pass@1 first**, LLM judge as secondary |
| No adversarial cases | Report poisoning, conflict, and staleness behaviour |
| No cost reported | Report tokens, latency, and cost per recommendation |

A credible ablation showing experiential memory contributes **independently of
documentation retrieval** would be a genuine contribution that the category
leader has not made.

### The uncomfortable possibility

If an honest ablation shows experiential memory adds little over
documentation retrieval, **that is a finding about the whole category** — and it
would be better to discover it in a spike than after building the product.

This should be an explicit early spike, and it doubles as the kill criterion
for D-004.

---

## Notable design details worth reusing

- **MCP as the sole integration surface** — agents interact by tool calling; no
  SDK integration burden on the developer. Consistent with the "one dev, no
  coordination" requirement in D-003.
- **Seeding the memory before any experience exists** solves the cold-start
  problem: the system is useful on day one from documentation alone, and
  experiential value accrues later. This directly addresses CRED's n=1 problem.
- **Curation runs as a separate autonomous process**, not inline with agent
  requests — consistent with Law 2 (no human review queue).
- Their claim that knowledge creation and knowledge use "do not need to share the
  same model, model generations or capabilities" is worth testing; if true, it
  means cheap models can harvest memory for expensive ones.
