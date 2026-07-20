# Decision Log

Running record of scope and direction decisions made during discovery.
Each entry records the decision, the reasoning, and what it rules out.

---

## D-001 — Game selection: OSS traction first, long-horizon category bet

- **Date:** 2026-07-20
- **Status:** Decided

### Decision

CRED plays **adoption-first open source** (stars, contributors, real installs)
on a **long-horizon** timeline. Monetization is deferred, not designed away.

Seta International is available as a real-world proving ground, but CRED is
**not** being built to one organization's requirements. Seta is an evidence
source, not a spec.

### Reasoning

A solo founder cannot win this category by out-building Mem0, Onyx, or RAGFlow
on surface area. The only durable early asset is adoption, and adoption in
developer infrastructure comes from time-to-first-value, not feature count.

Long horizon means architecture decisions must not foreclose the organizational
model later, even though the first release will not ship it.

### What this rules out

- Enterprise-first sequencing: no SSO, RBAC, audit, or org hierarchy in v1.
- Heavy infrastructure as a precondition to trying the product.
- Building to a single design partner's internal workflow.
- Any first release whose value requires a team to adopt it simultaneously.

### What this forces

- Time-to-first-value measured in minutes, not a deployment project.
- The first release must be useful to **one developer, alone**, or it will not
  spread far enough to ever reach organizations.
- Data model and provenance decisions must be made correctly at v1 even though
  governance features ship later — these are the expensive things to retrofit.

### Open tension (unresolved)

The product thesis is **organizational**; the adoption path is **individual**.
These pull in opposite directions. Resolving this tension is the central design
problem of the next phase.

---

## D-002 — Evidence discipline

- **Date:** 2026-07-20
- **Status:** Decided

All competitive scans, spikes, and benchmarks are captured as durable documents
under `docs/research/`, with claims tied to concrete file references or cited
URLs. Verbal conclusions that are not written down do not count as evidence.

Repository scans live in `docs/research/evidence/`.

---

## D-003 — Adoption ladder: individual → team → organization

- **Date:** 2026-07-20
- **Status:** Decided

### Decision

CRED expands along a bottom-up ladder. An individual adopts it because it makes
their own work better. They then apply it to their team because the same
artifact already works there. The organization adopts it last, because teams
already depend on it.

This resolves the central tension between an organizational thesis and an
individual adoption path:
the thesis stays organizational while the adoption path stays individual.

### The design law this imposes

Each rung must be **independently valuable**, and climbing to the next rung must
be **nearly free**.

1. **Individual** — value with no server, no team, no coordination. If the first
   run requires anyone else's participation, the ladder has no bottom rung.
2. **Team** — reached by sharing an artifact that already exists, not by
   migrating to a different product. No re-modelling, no re-import.
3. **Organization** — governance, policy, and measurement layered *on top of*
   what teams already run, never as a precondition for running it.

A feature that only makes sense at rung 3 must not be required at rung 1.
A feature that blocks rung 1 must be removed regardless of its rung-3 value.

### Constraint on governance specifically

Evidence from the Mem0 and Letta scans shows both projects built governance
scaffolding and left it unwired, pushing the real capability into hosted
products. The most likely explanation is not incompetence: governance adds
friction (approval, review, versioning) exactly when an individual user wants
speed.

Therefore: **governance must be a free side effect of something a developer
already wants for selfish reasons — never a feature they are asked to buy into.**

If CRED markets governance as its rung-1 value proposition, it will fail the
same way.

### What this rules out

- Any onboarding step that requires an organization to exist.
- A separate "team edition" that individuals must migrate onto.
- Mandatory review or approval workflows at rung 1.

---

## D-004 — Direction: self-hostable OSS organizational memory

- **Date:** 2026-07-20
- **Status:** Decided
- **Supersedes:** the eight-item prototype scope in the current PRD

### Decision

CRED is the **open-source, self-hostable organizational memory and governance
layer** — the OSS answer to closed products like Memco and Glen.

The bet is on **data sovereignty**: organizations want shared agent memory but
will not send it to a third-party cloud.

Chosen over the alternative of entering through harness sync (where demand is
measured at ~985k downloads/month) and expanding toward memory later.

### The load-bearing assumption

> **Organizations want organizational agent memory, and the binding constraint
> on adoption is that it must run on their own infrastructure.**

Everything in this direction rests on that sentence. It is currently
**unverified**. The supporting signal is indirect: Memco sells SOC 2 and on-prem,
which implies buyers demanded both.

**If this assumption is false, the direction is wrong** — not merely
suboptimal — because sovereignty is what differentiates CRED from products that
are otherwise ahead on funding, compliance, and time.

### Known risks accepted

Recorded so they are not rediscovered as surprises. Full evidence in
[synthesis.md](synthesis.md).

1. **The reference competitors are unvalidated.** Glen is YC 2026; Memco is
   early. Building the OSS counterpart to an unproven company means inheriting
   its hypothesis risk without its funding to survive being wrong. Langfuse and
   n8n cloned *validated* markets — this does not.
2. **Platforms price the memory primitive at zero.** Anthropic's memory tool is
   GA, MCP's memory server ships free in an 88.6k-star repo, Gemini context is
   on by default.
3. **Nobody successful sells memory as the product.** Letta pivoted to coding
   agents; Zep sells compliance; Devin gives DeepWiki away.
4. **The category shows no revenue.** Five projects at 23k+ stars, none
   disclosing revenue; ~$50M total disclosed funding across all dedicated
   agent-memory startups.
5. **Mem0 tried governance and withdrew it** — the reconciler has zero call
   sites and supersession ships only in the hosted product.

### Required test of the assumption

The sovereignty premise must be tested **before** significant engineering, and
it can be tested cheaply: ask organizations already running agents whether they
would adopt a hosted shared-memory product, and if not, why not. If the answer
is cost or usefulness rather than data residency, the premise fails and the
direction should be revisited.

Seta is the first available site for this test.

### Open contradiction to resolve

D-003 requires value at **n=1**. Organizational memory is team-shaped by
definition. The design must give a single developer a reason to run CRED alone,
using the **same artifact** that later becomes team memory — otherwise the
adoption ladder loses its bottom rung.

---

## D-005 — Compete head-on with Memco and Glen

- **Date:** 2026-07-20
- **Status:** Decided
- **Amends:** D-004

### Decision

CRED competes **directly** with Memco and Glen rather than seeking an
unoccupied adjacent position.

Founder's reasoning, accepted: the AI market is large enough that one or two
competitors is not crowding. A market with funded competitors is a **validated**
market. Avoiding it forfeits the category.

This overrides the earlier recommendation to pivot toward a community-owned
commons.

### What changed since D-004

Direct fetches of [memco.ai/how-it-works](https://www.memco.ai/how-it-works) and
[tryglen.com](https://www.tryglen.com/) materially weakened D-004's premise:

**Memco already ships on-premises.** "Self-hosted Spark instance entirely behind
customer firewall. Org memory never leaves customer infrastructure. Identical
MCP & CLI interface to cloud version." Plus SOC 2, per-tenant isolation, SSO.

**Sovereignty alone is therefore not a wedge against Memco.** It remains one
against Glen, which is cloud-only.

Memco has also moved well beyond its own paper. Published SWE-bench steady-state
results: **−40% LLM cost, −34% wall-clock, −31% agent steps, half the outcome
variance.** They corrected the methodological weaknesses identified in the
[paper teardown](evidence/memco-spark-paper.md).

They independently arrived at the same conclusion as Law 2 — "No human reviewer,
no manual approval queue" — confirming the reasoning but removing it as a
differentiator.

### The axes CRED must win on

Free-and-open is not by itself a strategy; the category shows five projects at
23k+ stars with no disclosed revenue. Competing head-on requires a defensible
axis. Two are structurally available to open source and not to a funded closed
competitor:

**1. Auditability of a system of record.**
Organizational memory governs what every agent in a company believes. Memco's
trust model is a **Bayesian posterior inside a closed product** — an
unauditable black box deciding which claims reach production agents.
Organizations have historically refused to place systems of record in
uninspectable systems. CRED can make *why the system believes X* fully
inspectable, reproducible, and testable. A closed vendor cannot match this
without opening their core.

**2. Evaluation honesty as category leadership.**
Memco's published paper has no ablation separating documentation retrieval from
experiential memory, ran one epoch, and used an LLM judge where execution tests
were available. Publishing the rigorous version — with ablations, epoch curves,
and execution-first metrics — establishes technical credibility in the category.
This is how Langfuse won mindshare against a better-funded incumbent.

**Supporting, not sufficient:** no lock-in on a system of record, forkability
and custom curation policy, and community-contributed integrations.

### Accepted disadvantages

Recorded honestly. CRED is a solo effort against a funded team of seven-plus
with SOC 2, published benchmarks, a research paper, and a shipped on-prem
product. CRED's compensating assets are Seta as a live evaluation environment
and the freedom to publish findings that a vendor cannot.

---

## D-006 — Parity over differentiation; distribution is the edge

- **Date:** 2026-07-20
- **Status:** Decided
- **Amends:** D-005

### Decision

CRED targets **feature parity** with Memco and Glen. Building a close
equivalent is explicitly accepted. The goal is to be **as good as they are**,
not different from them.

Founder's stated edge is **distribution**: access to users and revenue channels
independent of product differentiation.

### What this changes

Differentiation stops driving design. Design is now driven by **parity speed**:
the shortest path to a system that performs comparably on the same tasks.

The auditability and evaluation-honesty axes from D-005 are **retained but
demoted** — pursued where they are free, never at the cost of parity. They
remain the answer to "why you rather than Memco" once parity exists.

### The new load-bearing assumption

> **The founder can acquire users and revenue through existing channels, without
> product differentiation.**

This now carries the weight that data sovereignty carried in D-004. It replaces a
product bet with a **distribution bet**.

It is a legitimate strategy — many companies win on distribution against better
products — but it should be stated plainly: **if the distribution advantage does
not materialise, a parity product with no differentiation has no fallback**,
because the reason to choose it over a funded incumbent would be price alone.

**Recommended cheap test:** before parity is complete, confirm that at least one
organization outside Seta will commit to deploying it. Seta alone validates
usage, not distribution.

### Scope consequence — this is good news

Parity with Memco is a **much smaller scope** than the original eight-item PRD.
The observable surface is roughly:

- 4–8 MCP tools (`search`, `create_memory`, `enrich_memory`, `share_feedback`)
- Hybrid retrieval: vector + BM25 + trust weighting
- A trust/confidence model updated by usage and feedback
- An autonomous curation worker (ingest → curate → dedupe → prune → rescore)
- Two memory scopes: organization (default) and public (opt-in)
- Self-host and managed deployment from one codebase
- Observation-level access control applied at recall (Glen's model)

Everything in the original PRD not on this list is **deferred**.

---

## D-007 — Thesis correction: permission gate over sovereignty

- **Date:** 2026-07-20
- **Status:** Decided
- **Amends:** D-004. Leaves D-005 (compete head-on) intact.

### Decision

Three corrections, confirmed by the founder:

1. **Self-hosting is demoted from wedge to capability.** It remains a
   requirement — it is what makes the individual entry point and the adoption
   ladder work — but it is no longer the reason to choose CRED.
2. **The permission gate becomes the product focus.**
3. **The first experiment is no longer CRED's own ablation.** It is whether
   retrieved memory beats plain long context at all.

### Why sovereignty failed as a wedge

It was attacked from three independent directions:

- **Competitors already offer it.** Memco advertises on-prem; Glen offers
  self-hosting at Enterprise for a fee.
- **An OSS equivalent already exists.** `caura-ai/caura-memclaw` (Apache-2.0,
  air-gappable, MCP-native) ships the full governed-org-memory axis.
- **The market solved data residency a different way.** 28.1% report InfoSec
  blocks, but the answer was **regional residency, not customer hosting**.
  Samsung reversed its ban in June 2026 — to *cloud* Codex. Cursor holds 64% of
  the Fortune 500 with no on-prem offering. The EU AI Act slipped to 2027–28.

**The Vietnam legal-mandate argument for on-prem is false** and must never be
used. The defensible Asian angles are Japanese closed-network client regimes and
a 6–11x relative price burden.

### Why the permission gate is the focus

Two independent scans converged:

- Of roughly 80 surveyed projects, **fewer than 15 have a real multi-user
  permission model.** GitHub searches for `multi-agent memory ACL` return 3
  repositories, the largest with 2 stars.
- Onyx ships ~55 connectors but **only 12 with permission sync**. Permission
  fidelity, not feature count, is what nobody replicates.

And the decisive observation: **learned knowledge crosses no user boundary at
any vendor.** Anthropic ships server-managed `claudeMd` polled hourly; Cursor
ships Team Rules. Neither moves what an agent *learned* across a user boundary.

This gap is **structural, not a missing feature** — which is why platform
vendors have not closed it by shipping one.

> Crowded at the tagline, empty at the permission gate.

### Why the long-context experiment comes first

A benchmark reports Mem0 and Zep as far more expensive and materially less
accurate than plain long context. If that holds, CRED is wrong — and so are
Memco, Glen, Mem0, and Zep. It is cheap to test and existential, so it runs
before anything else is built.

**Correction, 2026-07-20.** This entry originally cited the benchmark as
"unreplicated but **unrebutted** … 14–77x more expensive and 31–33% less
accurate." That overstated it in three ways, found when the source was located
and read ([v0 experiment design](spikes/v0-experiment-design.md)):

- Its long-context arm averaged **~4,232 input tokens**. That does not test long
  context; it tests whether a memory layer can beat a prompt already containing
  the answer. The regime CRED operates in was never measured.
- "Unrebutted" was wrong. Three HN submissions scored 4, 4, and 2 points, with
  one comment, written by the author. Nobody rebutted it because nobody read it.
- The figures do not reconcile. 77x is extrapolated from a Zep run aborted at
  1,730 of 4,000 cases; Mem0's measured ratio is 12.6x, and "14" is not
  derivable from the published table. The accuracy range matches neither the
  absolute drop (33.0 / 35.3 points) nor the relative one (39.0% / 41.7%).

Also: conversational memory rather than code, cheapest model tier, no variance
reported, and the author sells memory infrastructure.

The decision is **unchanged** — the experiment still runs first. What changed is
the reason. It is not that strong evidence contradicts the thesis; it is that
**no adequate evidence exists in either direction**, while the category sells as
though the question were settled.

### The demand risk being accepted

Recorded plainly, because it is the weakest point in the plan:

**In roughly 30 HN results on this topic, about 20 are founders promoting their
own memory products.** Two genuine non-vendor quotes describe team divergence.
HN searches for "organizational memory" return 6 stories, the highest scoring 2
points.

A market where sellers outnumber buyers roughly 10:1 is more often a
seller-imagined market than an early one. **No evidence gathered so far
contradicts this reading.**

The founder proceeds on the basis of distribution access (D-006), not on
demonstrated organic demand. This is the assumption most likely to be wrong.

**Cheap disconfirmation available:** no major survey asks how teams share agent
context. Running that survey is simultaneously research, marketing, and a test
of the thesis.

### Operating parameters set by this evidence

- Emit `AGENTS.md` before `CLAUDE.md` — 154,496 files vs 51,100 on GitHub code
  search; the format contest is settled 3:1.
- Price anchor is **$20–40 per seat per month**. Memco at $599 per contributor
  per year sits outside the market — an exploitable weakness, not a benchmark.
- **Never gate SSO.** Langfuse gives it away and landed 63 Fortune 500 logos.
- Expect **~1–1.3% OSS conversion**. PostHog deliberately shed self-hosters as
  unprofitable; self-hosted users are the hardest cohort to monetize.
- Only **30.9% of developers use agents regularly** — the addressable base is
  smaller than the discourse suggests.

---

## D-008 — Go is the implementation language, with a bulk-embedding condition

- **Date:** 2026-07-20
- **Status:** Decided

### Decision

CRED is written in **Go**, with embeddings running in-process in **pure Go**
(`onnx-gomlx` on gomlx `simplego`), `CGO_ENABLED=0`, shipped as a statically
linked binary on a distroless base.

The WordPiece tokenizer is **hand-maintained, not borrowed**, and its character
class tables are **generated by probing the pinned HuggingFace Rust tokenizer**
— never derived from Go's `unicode` package.

### Reasoning

Go was previously chosen on an unverified assumption, recorded as such in
[packaging and first run](spikes/packaging-and-first-run.md): if the tokenizer
needed CGO, the project would lose cross-compilation without QEMU, distroless
static bases, and musl compatibility in a single stroke.

The [spike](spikes/go-embeddings-tokenizer.md) closed it empirically:

- **242,247 of 242,247 inputs match the reference tokenizer exactly** — 43
  curated edge cases, 47,676 fuzz strings, 194,528 single-codepoint probes.
- Forward pass cosine similarity **1.00000000** against ONNX Runtime, max
  element delta 1.4e-7.
- `go list -deps` reports **zero cgo packages** across 218 packages; the stack
  cross-compiles statically to four platforms.

The tables must be probed rather than computed because the goal is not Unicode
correctness — it is **byte-identity with the tokenizer that trained the model**.
HuggingFace's Rust implementation has frozen Unicode tables (824 codepoints
classify differently than Go's current ones) and a 256-codepoint hole in its
CJK range at U+2B820..U+2B91F that original BERT does not have. A
textbook-correct implementation scored 99.67%, and every one of its failures is
invisible on English text — the worst possible failure shape.

### What this rules out

- ONNX Runtime via CGO on the default build path, and with it Alpine/musl
  breakage and QEMU-based release builds.
- A Python or Node sidecar for embeddings, and the two-container deployment
  that would follow. Single-command deploy survives.
- Depending on a third-party Go tokenizer without a conformance suite.
  hugot's own tokenizer is **UNVERIFIED** and must be assumed non-conformant
  until run against the three suites.
- Using `go tool nm | grep cgo` as a CI guard. It reports 10 `_cgo_` symbols
  with CGO both on and off, so a check built on it **passes silently while
  broken**. The working guard is `go list -f '{{if .CgoFiles}}...' -deps ./...`,
  run against the real shipping build command.

### What this forces

Pure Go is **9–16x slower** than ONNX Runtime, and the gap widens with sequence
length (25.4ms vs 2.7ms at seq 16; 1503.7ms vs 94.2ms at seq 512). Interactive
recall is unaffected at 51ms per query. **Bulk ingestion is**: roughly 1.5 hours
versus 7 minutes for 10k chunks.

So the default path stays pure Go, and bulk ingestion gets an **optional
accelerated build** behind a build tag, plus honest progress reporting on first
ingest. The slow path must remain correct and complete, never a degraded mode.

### Open tension (unresolved)

The **reranker was not tested**. `bge-reranker-v2-m3` is a substantially larger
model and gomlx explicitly scopes `simplego` to small ones; this result does not
transfer, and reranking is in v1 scope. **int8 quantization is untested** — the
most obvious unpulled lever on both the 12x gap and the 632 MB peak RSS. All
timings come from one M1 Pro; the ratio should travel better than the absolutes.

---

## D-009 — First run is read-only; writing is opt-in

- **Date:** 2026-07-20
- **Status:** Decided
- **Evidence:** [context7.md](evidence/context7.md)

### Decision

CRED's first run reads and never writes. Contribution — the act of storing a
claim — is a deliberate, separate step the user takes after they have already
gotten value from recall.

`CONTEXT7_API_KEY` also stays out of the first run, confirming the existing
no-API-key constraint. That constraint is now evidence-backed rather than
asserted: Context7 launched with no key, added one three months later, and two
commits explicitly **remove** key mentions to cut install friction.

### Reasoning

Context7 exposes **two** MCP tools, both read-only, both taking two string
parameters. It reached 59,457 stars and 3.7M npm downloads per month. Its trial
is free in the only sense that matters — installing it costs the user nothing
they can regret, because it cannot change anything.

CRED writes. That makes its first install **structurally more expensive** than
any read-only server, and no amount of packaging polish removes that. A user
evaluating CRED is being asked to let an agent store things about their work
before they have any reason to trust the storage.

Separating the two collapses the trial cost back to Context7's level without
giving up the write path.

### What this rules out

- Any onboarding flow where the first agent interaction produces a write.
- Automatic background contribution before explicit opt-in, however useful.
- Treating the tool count as settled. Four tools is not obviously wrong, but the
  best available evidence points toward fewer, and the burden is on each tool to
  justify itself rather than on the reduction to justify itself.

### What this forces

Cold-start seeding (repository history and documentation) must carry the entire
first-run value, because nothing else will have been written yet. That was
already required by D-003's n=1 constraint; this makes it load-bearing twice.

### Disconfirming evidence recorded, not smoothed

Two findings from the same scan cut against current decisions:

1. **Auditability is weaker as a wedge than D-005 assumed.** Context7's trust
   signals ("Benchmark Score", "Source Reputation") are unexplained,
   popularity-shaped, and closed. 59k users did not care. This supports D-006's
   demotion of differentiation and should be read as evidence that
   evidence-based trust is a *correctness* argument, not a *demand* argument.
2. **"Never gate SSO" is not unanimous practice.** Context7 gates SSO to
   Enterprise, contradicting the operating parameter in D-007. Langfuse's
   evidence still carries that rule, but it is one strong case rather than a law.

### Open tension (unresolved)

Context7's distribution rested on a problem nobody had to be *sold*: "my LLM
gives outdated docs." D-006 bets that founder channel access substitutes for
pre-existing demand. Context7 proves the **mechanics** are reproducible by a
small team — fifteen months of packaging every channel, no growth hack — but is
**silent on whether those mechanics work without the demand underneath them.**

That gap is the same one D-007 already flagged as the weakest point in the plan.
Two independent scans have now landed on it. It is not a documentation problem;
it is the thing the v0 experiment and a demand survey exist to test.

---

## D-010 — Cross-encoder reranking is cut from v1; MaxSim replaces it

- **Date:** 2026-07-20
- **Status:** Decided
- **Amends:** D-008 (falsifies its int8 expectation), supersedes the reranking
  row in [tech-decisions.md](spikes/tech-decisions.md)
- **Evidence:** [go-reranker.md](spikes/go-reranker.md)

### Decision

v1 ships **no cross-encoder reranker**. Second-stage ranking is ColBERT-style
**MaxSim late interaction** over token vectors from `bge-small-en-v1.5` — the
model and tokenizer D-008 already verified.

### Reasoning

D-008 left the reranker as its largest open risk, on the theory that a larger
model might not survive pure Go. That framing was wrong in an instructive way.

`bge-reranker-v2-m3` **does** run in pure Go. No missing op, no OOM; `simplego`
executes all 26 op types. It is still unusable: **871 ms per pair on ONNX
Runtime with CGO** at batch 1 / seq 512, roughly 44 s for 50 candidates. The
model is infeasible on CPU regardless of language, so the previous choice was
wrong independently of Go.

> **RETRACTED, 2026-07-20.** This entry originally claimed the Go/ONNX Runtime
> gap **narrowed to 4.7–8.9x** from D-008's 9–16x, and cited 1.8 s per pair and
> 67 s for 50 candidates. Both came from an ORT fp32 baseline inflated 2.1–2.5x
> by CPU contention. Measured clean, both backends in one command on an idle
> machine, the ratio is **flat at 10.4–10.6x across seq 128/256/512** —
> squarely inside D-008's range. There is no narrowing.
>
> The claim is withdrawn rather than adjusted, because it was presented as a
> finding and it was an artifact. Worth noting *why* it was seductive: it
> arrived with a plausible mechanism attached — "larger matmuls suit
> `simplego`'s kernels" — which made it easier to believe rather than harder.

No substitute clears the bar. The fastest, `jina-reranker-v1-turbo-en` at 37.8M
parameters, needs **2.5 s for 20 candidates** at seq 128 and 6.2 s for 50 —
against 51 ms to embed the query. Between **49x and 282x over budget**.

MaxSim delivers **+0.0293 NDCG@10** at **1/5,841th the cost**, 0.05 ms per pair.

The price is named rather than buried. `bge-reranker-v2-m3` is genuinely the
best reranker measured, and MaxSim gives up **0.0167 NDCG@10** against it:

| Method | NDCG@10 | Δ vs vector only |
|---|---|---|
| Vector only | 0.6662 | — |
| `bge-reranker-v2-m3` | **0.7122** | **+0.0460** |
| `jina-reranker-v1-turbo-en` | 0.7018 | +0.0356 |
| ColBERT MaxSim | 0.6955 | +0.0293 |
| `mxbai-rerank-xsmall-v1` | 0.6415 | −0.0246 |

That 0.0167 is the cost of a CPU-only, air-gapped, statically linked
deployment. **It should be reopened the moment CRED targets a GPU** — this is a
constraint-driven decision, not a claim that MaxSim is better.

Size does not predict quality: the ordering was 567.8M > 37.8M > 70.8M.

### What this rules out

- Any cross-encoder in the interactive path on CPU, at any model size tested.
- **int8 quantization as a latency lever in pure Go.** D-008 flagged it as the
  obvious unpulled lever.

  > **Corrected from clean paired re-runs, 2026-07-20.** The first measurements
  > were taken while an unrelated sweep occupied the same CPU. Every multiplier
  > shrank, and every one held its direction:
  >
  > | Claim | First reported | Clean |
  > |---|---|---|
  > | int8 slower, `bge-small` | 2.05x | **1.98x** |
  > | int8 slower, m3 | 2.63x | **2.28x** |
  > | ORT int8 speedup, m3 | 5.57x | **2.48x** |
  > | RSS cut, `bge-small` | 2.64x | **2.23x** |
  > | RSS cut, m3 | 2.21x | **1.64x** |
  >
  > `bge-small` fp32 absolute fell from 331.38 ms to **206.13 ms**, now
  > consistent with D-008's independent 222.9 ms. The direction never moved:
  > `jina` and `mxbai` were measured back-to-back in a single command, so their
  > ratios were never cross-command and never distorted.

  It is **roughly 2x slower** across four models, not
  faster, because `onnx-gomlx` widens both int8 operands to int32 before
  multiplying (`ops.go:2967`). The cleanest evidence is m3: the **same quantized
  graph** runs **2.48x faster** on ONNX Runtime and **2.28x slower** in pure Go.
  Quantization therefore **widens** the Go/ORT gap rather than narrowing it. It
  remains a **memory** lever: 1.6–2.2x RSS reduction at 0.0004 NDCG cost.
- Trusting an available Go SentencePiece tokenizer. `eliben/go-sentencepiece`
  refuses the model (`model type UNIGRAM not supported`). `sugarme/tokenizer`
  scores 96.22% with 948 hard panics, fails **100% of inputs containing two
  consecutive spaces** — that is all indented code — and silently ignores
  truncation on the pair path, the only path a cross-encoder uses.
- D-008's mitigation as a general technique. Probing enumerates a per-codepoint
  predicate; it cannot enumerate a 237 KB string-to-string Darts trie. The
  WordPiece fix does not generalize to SentencePiece.

### What this forces

MaxSim costs **242x storage per document**, measured uncompressed: roughly
17 GB against 73 MB for 100k chunks. That is the real trade, and it lands on
L7's single-database constraint. Storage strategy for token vectors is now an
open design question, not a settled one.

A reranker must beat the vector-only baseline on a labelled set before shipping.
This is not a formality: `mxbai-rerank-xsmall-v1` scored **−0.0246 NDCG@10**,
actively worse than no reranking, while being 1.9x larger and 2.1x slower than
the model that scored +0.0356.

### Correction, 2026-07-20

This entry was first written from an incomplete draft: `bge-reranker-v2-m3`'s
own NDCG and its int8 benchmark were still running and were recorded here as
unverified. Both have since landed, and all three affected numbers moved — the
MaxSim quality gap (0.0063 implied → **0.0167** measured against m3), the int8
slowdown, and the ORT int8 speedup.

Then a second correction, from the spike's own re-check: those replacement
numbers were themselves measured under CPU contention. Clean paired re-runs
moved every multiplier and **retracted one finding outright** — the claimed
narrowing of the Go/ORT gap, which was an artifact of an inflated fp32
baseline. Final figures are inline above and in the spike document.

Two procedural lessons, both cheap and both now paid for:

1. **Do not write a decision from a spike that is still running.** This entry
   was authored from an incomplete draft.
2. **Pin measurement conditions before quoting a ratio.** Ratios measured
   back-to-back in one command survived the contention; cross-command
   comparisons did not. The error was caught only because D-008 had recorded an
   independent baseline for the same configuration — an apparently redundant
   number that turned out to be the control.

The decision never moved. Even with every figure corrected downward, the fastest
pure-Go cross-encoder is **49x over budget at its most favourable
configuration**, and m3 needs 44 s for 50 candidates on the *fast* backend. No
measurement error of this size reaches that. NDCG results are unaffected —
those are deterministic.

### Process note

A truncated download nearly became a fabricated upstream defect: a `curl` loop
killed by a timeout at 5.5 MB of 17 MB produced an unparseable `tokenizer.json`.
**Size-check every downloaded artifact against `x-linked-size`.** Recorded here
because the near-miss is the same failure shape as the earlier fabricated
citations, and the rule that catches it is cheap.
