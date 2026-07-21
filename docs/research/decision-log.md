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

- Emit `AGENTS.md` before `CLAUDE.md` — re-measured 2026-07-20 as **151,168 vs
  53,132**; the ratio holds at ~2.85x but the absolutes below are stale and
  GitHub's `total_count` is approximate. Originally 154,496 files vs 51,100 on
  GitHub code
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
  broken**.

  > **Refined 2026-07-20** ([go-repo-conventions.md](spikes/go-repo-conventions.md)).
  > The replacement guard has a **second vacuous-pass mode**, verified by
  > running it: with a cgo-only dependency present and `CGO_ENABLED=0`,
  > `go list` writes its complaint to **stderr** and emits nothing on stdout,
  > so `[ -z "$(... 2>/dev/null)" ]` reports clean while the build is broken.
  > That is the same failure shape this entry rejected `nm` for.
  >
  > The guard must be **two assertions**: enumerate cgo packages under
  > `CGO_ENABLED=1`, then assert `CGO_ENABLED=0 go build` succeeds. Running
  > `go list` under `CGO_ENABLED=0` alone — the intuitive reading of "against
  > the real shipping build" — is vacuous.

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
- **Status:** **Partly falsified** — see the correction below
- **Evidence:** [context7.md](evidence/context7.md),
  [how-they-operate.md](evidence/how-they-operate.md)

> **Correction, 2026-07-20.** This entry fused two separable claims, and only
> one survived.
>
> **Read-first sequencing — supported.** Reads cost zero LLM calls in Mem0,
> Graphiti and Zep. A read-only first run really is the cheap trial.
>
> **Never-automatic — not supported.** Mem0's shipped coding-agent plugin
> writes automatically at three trigger points: every 3rd user prompt, every
> Bash result, and session `Stop`. All default to **on** — `MEM0_AUTO_SAVE`
> and `MEM0_PREFETCH` are opt-*outs*. That is precisely what this entry ruled
> out, shipping in the product CRED intends to reach parity with.
>
> The error was in the derivation: this rule was inferred from Context7, and
> **Context7 has no write path at all.** A system with no writes cannot be
> evidence about when to write.
>
> What holds: reads before writes, and writes that are visible and reversible.
> What does not: forbidding automatic contribution.

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
against 51 ms to embed the query. Between **49x over budget at the most
favourable configuration and 1,038x at a realistic one**.

MaxSim delivers **+0.0293 NDCG@10** at **1/2,468th the cost**, 0.05 ms per pair.

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
  > all four back-to-back int8 ratios landed within 4% of their contended
  > values (2.05→1.98x, 1.92→1.95x, 1.70→1.69x). Only the two figures
  > assembled across commands were badly wrong.
  >
  > **Peak RSS is not reproducible to better than ~1.6x**, and this is not
  > contention: `bge-reranker-base` moved 2.65→4.19 GB while m3 moved
  > 4.58→3.60 GB — opposite directions, so allocator and GC timing. Treat every
  > RSS figure in these entries as order-of-magnitude only.

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
actively worse than no reranking, while being 1.9x larger and 2.5x slower than
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

---

## D-011 — Sovereignty is a tiebreaker, not a wedge

- **Date:** 2026-07-20
- **Status:** Decided
- **Amends:** D-004
- **Evidence:** [demand interview 001](evidence/demand/001-seta-founder.md)

### Decision

Stop treating data sovereignty as the differentiator. It is a **default
preference that trades away against speed**, not a binding constraint.

Self-hosting remains a **capability CRED must have** — it is cheap to keep and
D-003's bottom rung requires a local single-user mode anyway. What changes is
that it is no longer the reason anyone would choose CRED, and it must stop
appearing in positioning as though it were.

### Reasoning

D-004 rests on one sentence: *organizations want organizational agent memory,
and the binding constraint on adoption is that it must run on their own
infrastructure.* [demand-test.md](demand-test.md) pre-registered the test:
if the blocker named is cost or usefulness rather than data residency, the
premise fails.

The first interview — a ~300-person organization running agents across the
whole SDLC — answered Q8 **"both."** Self-hosting is the default to protect
company resources, but cloud is acceptable when a project is large enough to
justify it. Q9 named the blockers as **cost, trust, security, and whether it
is worth it** — security present, but neither first nor alone.

A preference that yields to speed cannot carry differentiation.

The evidential weight is higher than n = 1 usually earns, because of the
direction: **the most biased available source returned disconfirming
evidence.** The respondent is the founder describing their own organization,
and founders recall thesis-supporting incidents more readily than quiet weeks.
Biased sources normally confirm. This one did not.

### What this rules out

- Sovereignty as the headline in any positioning, README, or launch.
- Any roadmap item justified primarily by air-gap or on-prem operation.
- Reading D-004's competitive argument as intact. Its risk list survives; its
  differentiation claim does not.

### What this forces

CRED needs a different answer to *why choose this over a hosted competitor*,
and the honest current answer is: **there is not yet a demonstrated one.**
D-006 already demoted differentiation in favour of distribution, and this
strengthens that reading rather than creating a new problem.

The four blockers named — cost, trust, security, worth-it — are the criteria a
buyer will actually apply. Being open source addresses cost and trust
structurally. That is a weaker claim than sovereignty but it is the one the
evidence supports.

### The observation that survived, and needs a second source

The same interview reported knowledge spread across **five stores**: ADRs,
Claude memory, `docs/` folders, Confluence, and verbal exchange. That is the
strongest pro-thesis observation gathered so far, and it comes from the same
biased source. It needs a non-founder to confirm before anything is built on
it.

### Open tension (unresolved)

**Agents assist rather than replace, with a human reviewing output.** If that
generalizes, an agent working from stale context costs review time rather than
a production incident — lower stakes, lower willingness to pay, and a failure
mode already caught by an existing control.

This weakens the pain the product is built around, and it was volunteered
rather than probed for. It is now the second thing interviews 002+ must test,
after the fragmentation observation.

---

## D-012 — Success is parity with Mem0 plus a working team layer

- **Date:** 2026-07-20
- **Status:** Decided
- **Amends:** D-004, D-005, and the kill criteria in the PRD

### Decision

CRED is **not** a venture-scale bet and stops being evaluated as one.

Success is defined as: **do what Mem0 does, at comparable quality, and cover
teams and organizations well enough to be worth adopting.** A good open-source
project with real users clears the bar. It does not need to establish a new
category, and it does not need to prove the category is large.

### Reasoning

Every kill criterion recorded so far assumed venture stakes — that the project
must be stopped unless the market is demonstrably big. That framing produced
disproportionate machinery: pre-registered numeric kill thresholds, a κ ≥ 0.70
judge gate, blind external audit, and an experiment whose stated purpose was
to decide whether to abandon the project.

Under this decision, the existence question is already answered. Mem0, Zep,
and Letta have users. People install and run memory systems. CRED does not
have to prove that; it has to be good and to cover the team case those
products cover poorly.

### What this changes

- **v0 stops being a kill criterion and becomes a design check.** It answers
  "is retrieval pulling its weight, and where," not "should this project
  exist." The harness stays; the pre-registered abandon-threshold does not
  apply.
- **The demand test stops being existential validation** and becomes ordinary
  user research: what teams actually need, not whether anyone needs anything.
- **The evidence bar drops for existence questions and stays high for design
  questions.** Whether the category is real is settled by competitors having
  users. Whether a specific mechanism works is still measured.

### What this does not erase

Recorded because a lowered bar is a legitimate scope decision and an
illegitimate reason to discard evidence:

1. **D-011 stands.** Sovereignty is a tiebreaker, not a wedge. Lowering
   ambition does not restore a differentiator the evidence removed.
2. **The 10:1 seller-to-buyer observation in D-007 stands.** It is no longer
   existential, but it still predicts that distribution will be harder than
   the discourse suggests.
3. **The long-context question stands as a design question.** If retrieval
   does not beat a long window on a task, the honest response is to not
   retrieve for that task — not to ship retrieval anyway.

### What this rules out

- Framing any future decision as "kill the project" on market-size grounds.
- Research whose only purpose is to justify the project's existence.
- Methodology calibrated for external peer review rather than for making a
  build decision.

### What this forces

The competitive question becomes concrete and answerable: **what does Mem0 do
well, and what does it not do for teams?** That is a product question with a
findable answer, not a market question requiring a survey. It is the subject
of [why-survivors-survive.md](evidence/why-survivors-survive.md).

---

## D-013 — Distribution is integration, and pricing is metered

- **Date:** 2026-07-20
- **Status:** Decided
- **Amends:** D-006 (makes its "distribution is the edge" concrete), corrects
  the seat-price operating parameter in D-007
- **Evidence:** [why-survivors-survive.md](evidence/why-survivors-survive.md)

### Decision

**Distribution means getting vendored inside someone else's repository**, not
launching. The channel is first-party integration packages in agent
frameworks, and it is free, solo-reproducible, and captures the decision
before a user frames it as a decision.

**Pricing is metered, never per seat.** No company in this category sells
seats.

### The distribution evidence

`run-llama/llama_index` ships `llama-index-memory-mem0` as a **first-party
package inside its own monorepo**. `strands-agents/tools` ships
`mem0_memory.py`. `crewAIInc/crewAI` has **71 code hits for `mem0`** against
Zep's 9, and **zero** for Letta and Cognee.

This is the Context7 mechanic pushed one layer deeper. Two independent scans
have now converged on the same answer, which is the strongest methodological
signal in the research so far.

### The pricing correction

Five pricing pages, five metered models: Mem0 by request, Zep by ingest
credit, Cognee by token ($2.50/1M), Supermemory by credit-wrapped usage, and
Letta by **$0.10 per active agent per month**. Zep and Cognee charge **zero
for reads, storage, and users**.

D-007 recorded a **$20–40 per seat per month** anchor. That is wrong on the
**unit**, not merely the amount. Letta's per-agent meter is the tell: when one
developer runs many agents, a seat is the wrong denominator by construction.

### The parity gap is three engineers

Mem0 was **four people** at its $24M Series A. Zep says five. Under D-012,
where success is parity plus a working team layer, this is the single most
actionable number in the research: the gap is small enough to be real.

### FALSIFIED

Mem0 is **not** AWS's exclusive memory provider. `strands-agents/tools` ships
five memory tools including AWS's own `agent_core_memory` (Bedrock AgentCore
Memory). The partner became the competitor in the same namespace, and Mem0
retained the exclusivity language in fundraising materials. A concrete
instance of D-004 risk #2 — platforms price the primitive at zero.

### The correction to this project's own method

**Every demand and graveyard finding in this repository is HN-derived, and HN
is one channel that demonstrably misses companies.** Supermemory reached #1 on
Product Hunt with 705 upvotes while its **peak HN score was 5**, and it has
28.5k stars and 278k npm downloads per month. Cognee's all-time HN peak is 9.

The graveyard table in `market-landscape.md` was already annotated as a
falsified inference for MCP servers. This widens the problem: the sampling
frame itself was narrow, and Product Hunt was never examined.

### Evidence *for* the thesis, found in a competitor's launch thread

The top two comments on Mem0's 201-point Show HN were **staleness and
privacy** — the exact governance questions Mem0 still has not answered two
years later. Zep has meanwhile repositioned onto the word "governed," with SOC
2 Type II in hand.

This partly offsets D-009's disconfirming note that Context7's users did not
care about closed, popularity-shaped trust signals.

### Open tension — the most serious finding in this scan

**Three funded companies retreated from the governed-team version of this
product within twelve months.** Mem0 v2.0.0 (2026-04-14) **removed**
`org_id`/`project_id` from the client SDK entirely. Letta tore out its memory
server, and its PyPI downloads fell **203k → 90k in one month**. Zep killed
self-hosting.

> **SUPERSEDED by D-014, 2026-07-20.** The paragraph above was investigated
> and three of its four load-bearing claims did not survive
> ([the-three-retreats.md](evidence/the-three-retreats.md)):
>
> - **Letta's 203k → 90k download fall is a burst artifact.** Daily series:
>   April baseline ~2,777/day, June ~2,970/day — **June is higher than
>   before the announcement**. May's figure contains a 259,952-download burst
>   confined to Python 3.12, and the announcement was six weeks earlier.
>   FALSIFIED. It was cited here after surviving a prior review.
> - **Mem0's `org_id` removal was one of fifteen parameters** dropped in a
>   typed-options refactor, alongside `output_format` and `batch_size`.
>   `/v1/ping/` resolved org and project from the API key before and after,
>   byte-identical. Removing a caller-supplied override is a **security fix**,
>   not a withdrawal of governance.
> - **Zep is undetermined.** BYOC appears in the pricing table and in zero of
>   265 documentation pages, so the artifact that would settle motive does not
>   exist.
>
> "Three funded companies retreated" was the wrong reading. What actually
> happened is recorded in D-014.

---

## D-014 — The gap is a pricing vacancy; governance ships in the open engine

- **Date:** 2026-07-20
- **Status:** Decided
- **Supersedes:** the "three retreats" open tension in D-013
- **Evidence:** [the-three-retreats.md](evidence/the-three-retreats.md)

### Decision

Org, member, and role primitives **live in the binary a solo developer runs
offline**. They are not a client-side wrapper over a hosted service, and they
are not held back for a paid tier.

### What the investigation actually found

The founder's hypothesis was that the three companies moved the team layer out
of OSS into the paid product rather than abandoning it. **Two of three
confirm.**

| Company | Verdict | Deciding artifact |
|---|---|---|
| Mem0 | **Monetized, actively defended** | `client.project.add_member(email, role=...)` ships in OSS `mem0ai` 2.0.12 against a live, undeprecated hosted org/project API of 21 operations |
| Letta | **Monetized** | `git_http.py:264-269` returns HTTP 501 `"git HTTP requires memfs service"` — client open, service closed |
| Zep | **Cannot determine** | The artifact does not exist: BYOC is in the pricing table and in zero of 265 doc pages |

The distinction matters more than the count. In every determined case the
**closed half is a service and the open half is a client** — and what stayed
open is extraction and retrieval while what closed is governance and tenancy.
That is **the exact inverse of the split CRED proposed.**

### The strongest single artifact in the research

mem0 issue #4589 and PR #4590. A production user diagnosed the org-scoping
gap, wrote the patch, took review feedback, and had it closed with *"doesn't
align with the current SDK direction"* — two weeks before the parameter was
deleted. The user's reply:

> "we will directly search with the Qdrant, resulting in not using Mem0
> client.search()."

**Demand was not absent. It was declined.** Every prior reading in this
project treated the vacancy as evidence that nobody wanted the team layer.
This is a paying user writing the code, being refused, and leaving.

### What this rules out

- Reading the vacancy as absence of demand. It is a commercial choice by
  incumbents, and a user walked over it.
- Any architecture where the principal type exists only in a client package.
- Holding org/member/role back as a monetization lever.

### The test, and it is checkable in one command

> Grep the engine for the principal type. **If it only appears in a client
> package, the retreat has already happened.**

This applies to CRED's own codebase as a standing check, not only to
competitors.

### The cost, named

Governance is **the only obvious monetization lever** in this category. Putting
it in the open tier forecloses it. D-012 makes that affordable — success is a
good OSS project with real users, not a revenue ramp — but the trade is now
deliberate rather than accidental, and reversing it later means taking
something back from users who already have it.

### Correction to D-003

D-003's friction mechanism — governance adds friction exactly when an
individual wants speed — is **mis-scoped rather than confirmed or falsified**.
It describes *buyer* behaviour. Every retreat examined here is a *seller*
decision. The conclusion D-003 draws still stands; the reason it gives does not
match this evidence.

### Corrections to existing evidence documents

Recorded because they were load-bearing and wrong:

- `letta.md` states Letta is deprecating `read_only`. It is **enforced** at
  `letta/agent.py:191-196` with 55 references; the `deprecated=True` marker is
  on the wire schema only. FALSIFIED.
- The "239 vs 33" OSS/cloud endpoint gap in the same document is a unit
  mismatch: 194 vs 239 paths, giving 62 cloud-only. Corrected.

---

## D-015 — Build. Research phase closed

- **Date:** 2026-07-20
- **Status:** Decided

### Decision

CRED gets built. The discovery phase is closed and does not reopen on market
grounds.

The founder's reasoning, accepted: **not building leaves outstanding at zero.**
Under D-012 the bar is a good open-source project with real users, and no
further evidence is needed to justify attempting that.

### What this closes

- The existence question. It is not asked again.
- The demand test as a gate. It continues as ordinary user research, on the
  schedule the founder chooses, and never blocks a commit.
- v0 as a precondition. It becomes a design check run against a working
  system, not a hurdle before one exists.

### What remains open, as engineering questions only

These stay live because they change what gets built, not whether:

- Whether retrieval beats a long window **for a given task shape** — if it does
  not, do not retrieve for that shape.
- Token-vector storage for MaxSim: 242x per document against L7's single
  database (D-010).
- Which framework integrations to ship into, and in what order (D-013).

### The constraints this build inherits

Carried forward so they are not rediscovered:

| Decision | Constraint |
|---|---|
| D-008 | Go, pure-Go embeddings, `CGO_ENABLED=0`, tokenizer tables probed from the reference |
| D-009 | First run reads and never writes; contribution is opt-in |
| D-010 | MaxSim, no cross-encoder |
| D-011 | Sovereignty is a capability, never the pitch |
| D-013 | Metered, never per seat; distribution is framework integration |
| D-014 | Org, member and role live in the engine, not a client |
| L1–L8 | The design laws in the PRD |

### Standing risk

Two funded competitors closed exactly the layer CRED is opening (D-014). If
CRED later needs revenue, that lever has been given away deliberately and
cannot be quietly taken back.

---

## D-016 — The first slice: seed from documentation, not repository history

- **Date:** 2026-07-20
- **Status:** Decided
- **Amends:** D-009
- **Evidence:** [how-they-operate.md](evidence/how-they-operate.md)

### Decision

The first implementation slice is a **read-only vertical**: seed from the
repository's own documentation and agent instruction files, index it, and
expose recall through one MCP tool and a CLI.

**Seeding comes from documentation, not from git history.** The proposal to
seed from commit history is withdrawn.

### Why the git-history proposal was wrong

The reason is CRED's own thesis, and it is the cleanest argument in the
research so far.

A claim anchored to `AGENTS.md:42` **expires when that file changes**. A claim
extracted from commit `abc123` has **immutable evidence and can therefore
never expire.** Seeding from history would fill the store with claims that are
permanently unfalsifiable — the exact inverse of *a claim lives only while its
evidence does.*

The tagline is not decoration. It is a constraint on what may be ingested.

### What the operational evidence changed

Cold-start seeding is **not** a vacancy. Mem0 ships it:
`integrations/mem0-plugin/scripts/auto_import.py` imports `CLAUDE.md`,
`AGENTS.md`, `.cursorrules` and `.windsurfrules` on `SessionStart`, with
SHA-256 change detection, in 374 lines. Supermemory has importers; Letta has
document upload.

What is actually true is narrower and still useful: **nobody seeds from
repository history, and where real seeding exists it is gated behind the
hosted tier.**

### The unclaimed position

**Nothing in this category ships config-free except the systems that never
call an LLM.** Since read paths cost zero LLM calls everywhere, D-008's
pure-Go in-process embeddings make a **zero-configuration read path**
achievable — no API key, no provider choice, no vector-store decision.

No competitor has one. This is the strongest unoccupied position the research
has found, and it is a consequence of a decision already made for other
reasons.

### The latency budget to meet

| Source | Figure |
|---|---|
| Mem0 hook timeout, `UserPromptSubmit` | 8 s |
| Mem0 search timeout | 5 s |
| Mem0's own stated comfort | "~150–200 ms is well within budget" |
| Zep Cloud, LoCoMo | median 241 ms, p95 576 ms, p99 933 ms |

Zep's figures are **cloud over network, concurrency 20, with a cross-encoder**
— not comparable to a local process, and not a target. The number to beat is
Mem0's 150–200 ms comfort band, which D-010's 51 ms local embed already fits
inside.

### The cost of this slice, named

This slice is close to **"Context7 with a local index."** It ships the
documentation-retrieval arm — which is the arm CRED's own ablation is betting
*against*, since D-012's thesis requires experiential memory to contribute
independently.

That is acceptable as a first slice because it is the cheapest thing that is
genuinely useful, and because it makes the ablation runnable against a real
system instead of a hypothesis. It is not acceptable as a destination.

### What this rules out

- Seeding from commit history, or from any source whose evidence cannot change.
- Requiring an API key, an LLM provider, or a vector-store choice to read.
- Treating the first slice as the product.

### Write path, deferred but not forbidden

D-009's never-automatic rule is withdrawn. When writes arrive, the shipped
precedent is automatic contribution at agent-lifecycle trigger points,
defaulting to on. The constraint that survives is that writes must be
**visible and reversible**, not that they must be manual.

---

## D-017 — Automatic write at agent-lifecycle triggers, nomination not storage

- **Date:** 2026-07-21
- **Status:** Decided
- **Completes:** D-009 (whose never-automatic half was already withdrawn by D-016)
- **Evidence:** [how-they-operate.md](evidence/how-they-operate.md)

### Decision

CRED writes automatically, at agent-lifecycle trigger points, defaulting to on
— the shipped Mem0 pattern. What is automatic is **nomination**, not storage.

### The mechanics, matched to the shipped precedent

Mem0's plugin writes at three points, all on by default, gated only by
opt-outs (`how-they-operate.md`): every 3rd user prompt, every Bash result,
and session `Stop`. One `add` is synchronous — 1 LLM call, 2 embedding calls,
~4 DB round trips, returning only after completion.

CRED matches the trigger model and the on-by-default posture. It does **not**
match "store whatever the model returns," because L2 forbids it.

### Where this differs from Mem0, and why it must

- **L2 — the model nominates, code decides.** The extractor emits candidate
  claims into a constrained schema and holds no authority to write, supersede,
  or expire. Automatic does not mean unvalidated: every response is validated
  locally and gated on `finish_reason != "length"`, because no provider
  validates structured output server-side.
- **L1 — no claim without evidence.** An automatic extraction that cannot point
  to the span that produced it is dropped, not stored. This is stricter than
  Mem0, whose extractions are free-floating memories.
- **D-016 — visible and reversible.** Every automatic write is inspectable
  (`cred log`) and reversible (`cred forget <id>`). The surviving constraint
  from D-009 is not that writes are manual; it is that they are never silent
  and never permanent-by-default.
- **L4 — confirmation inside the task, never a queue.** Automatic writes do not
  create an approval backlog. Reversal happens at the moment a person notices a
  wrong claim in a recall result, not in a review session.

### The latency constraint this inherits

The read path is 147.7 ms (D-016), inside Mem0's 150–200 ms comfort band. The
write path adds an LLM call and must not block the agent's turn on it. Mem0's
`add` is synchronous and its `UserPromptSubmit` hook has an 8 s budget;
CRED's extraction runs **off the turn** — the trigger enqueues, a River worker
extracts and writes, and the agent is never blocked waiting for nomination.

This is the first use of River (D-013 accepted it; nothing scheduled work until
now) and the first crossing of the LLM boundary (`internal/nominate`).

### What this rules out

- Storing model output without local validation (L2).
- Writing a claim whose evidence pointer is absent or unresolvable (L1).
- Blocking the agent's turn on extraction latency.
- Silent or irreversible writes (D-016).
- An approval queue of any kind (L4).

### What this forces

- `internal/nominate` — the LLM boundary, with a fake for tests, per the layout.
- A curation worker (dedup and supersession) on River, because automatic
  writes at every third turn will produce duplicates and contradictions by
  construction. Exact-hash dedup at v1 (D-010's neighbour decision in the
  tech-decisions spike), fuzzy deferred.
- `cred log` and `cred forget` — visibility and reversal are now product
  surface, not a future nicety.

---

## D-018 — L3 ships for text; the tree-sitter CGO gate is cleared, code anchoring gated elsewhere

- **Date:** 2026-07-21
- **Status:** Decided
- **Evidence:** [semantic-anchoring.md](spikes/semantic-anchoring.md)

### Decision

Semantic anchoring (L3) ships for **text/Markdown evidence** — the entire corpus
today (D-016). The fingerprint ladder lives in `internal/anchor` (pure, like
`temporal` and `acl`), is wired into seeding and the write path, and drives a
deterministic `cred reanchor` invalidation. **Code anchoring does not ship in
this slice**, but the reason is no longer CGO.

### The gate, and that it cleared

The question that gated the slice was the same shape as the tokenizer's and the
reranker's: **does a usable tree-sitter binding exist for Go with
`CGO_ENABLED=0`?** Tree-sitter is a C library, and `CGO_ENABLED=0` is
non-negotiable (D-008).

The spike answered it empirically. **It cleared.** `codeberg.org/hum3/gotreesitter`
v0.6.7 — a pure-Go GLR reimplementation loading tree-sitter's parse-table format —
builds under `CGO_ENABLED=0` (zero cgo packages), cross-compiles statically,
parses Go into a correct AST, and computes a tier-1 symbol path and tier-2
normalized node hash that **hold under formatting churn and an insertion above,
and change only on a semantic edit** — exactly L3's law.

The three C bindings (smacker, official `tree-sitter/go-tree-sitter`,
`alexaandru/go-sitter-forest`) are all CGO and all fail the `CGO_ENABLED=0`
build. The wazero/WASM binding (`malivvan/tree-sitter`) is CGO-free but ships no
Go grammar.

### Why code anchoring is still deferred — on a smaller gate

Not on CGO, which is disproved. On two other grounds:

1. **No producer.** Every `Evidence` row is `document` or `attestation`; nothing
   emits `code` evidence yet. A 10 MB parser dependency (the `grammars` package
   embeds all 206 grammars; the stripped test binary is 19 MB against ~9 MB
   today) for a code path with zero callers is weight for nothing.
2. **Fidelity is unverified — the D-008 tokenizer lesson.** A from-scratch Go
   reimplementation of a parser is the same risk class as a hand-written
   tokenizer: nearly-right is worthless for an anchor, because a confidently
   wrong claim is worse than no claim. `gotreesitter` is v0.6.x, single
   maintainer, with headline benchmark claims of exactly the seductive shape
   D-010 caught. Its grammar fidelity to upstream across real Go is unverified
   beyond a smoke test, and must be diffed before it is trusted.

So `internal/anchor` ships the pluggable seam `anchor.For(kind)`; the code
anchorer drops in at `SourceCode` with no caller change, gated on a code producer
plus a fidelity diff against upstream tree-sitter.

### What this rules out

- Reading "code anchoring is deferred" as "tree-sitter needs CGO". It does not
  (verified). The deferral is dependency weight and fidelity, not the build
  constraint.
- Adopting a reimplemented parser as the anchoring authority on faith. The
  fidelity diff is a precondition, per D-008.
- Expiring on tier 4 alone. Pre-existing tier-4-only evidence and attestations
  carry no tier-1/2 anchor, and re-anchoring leaves them untouched rather than
  expiring — over-expiring on a byte hash is the failure L3 exists to prevent.

### What this forces

- A grammar-fidelity spike (the tokenizer suites are the template) before
  `gotreesitter` is wired in, and a plan for the ~10 MB binary cost — trim to one
  grammar, or a pure-Go build tag, which does **not** reintroduce the CGO footgun.
- `cred reanchor <path>` as the deterministic invalidation primitive a CI step or
  a post-edit hook runs — a subcommand, not a daemon, because a file changing on
  disk is an external event a worker cannot see without watching the filesystem.

### Open tension (unresolved)

Seeding still supersedes a chunk on any raw-content change (tier 4), so a
formatting-only re-seed churns claims even though `cred reanchor` on the same
change would not. Re-anchoring is the L3-correct invalidation; unifying the two
so the seed path also decides on tiers 1–2 is future work, and until then
acceptance criterion 4 is met via `reanchor`, not via re-seed.

---

## D-019 — Usage limits are a poisoning defence, and a denial is never silent

- **Date:** 2026-07-21
- **Status:** Decided
- **Evidence:** PRD section 8; [how-they-operate.md](evidence/how-they-operate.md)

### Decision

The four section-8 limits ship on by default (zero-config), enforced
server-side, keyed per principal and per scope. They are built as security
controls first — the automatic write path (D-017) writes every third turn, so
unbounded per-principal write access is a poisoning vector, not merely a
capacity concern.

### The three decisions worth recording

- **A denial is recorded, never dropped.** Under the off-the-turn write path a
  quota denial cannot be a return value the caller sees, so it is written to the
  `usage_events` ledger as a `denied` row and logged at `warn` with the machine
  reason. A silent drop is how a poisoning attempt hides (L8); exhaustion is
  loud by construction. A recall denial is on the turn, so it is a typed error
  surfaced in-band.
- **The usage ledger has no foreign key to `principals`.** A denial must be
  recordable for *any* principal id, including one that does not resolve —
  accounting is not an access surface, and refusing to record an unknown
  principal would reopen the silent-drop hole. This is a deliberate inversion of
  the usual referential-integrity default.
- **The cost ceiling is calls and tokens, not currency.** CRED carries no
  per-model pricing, so a USD ceiling would be a number it cannot honestly
  compute. Calls and tokens are what it measures; the operator converts.

### Where the store-returns-rows law bit

Contribution counting is done in SQL against a Go-computed half-open window
cutoff, never as a `WHERE count >= quota` predicate. Every ceiling comparison
lives in pure `internal/limit` (on the depguard pure-algebra boundary with
`temporal`, `acl`, `anchor`); Postgres counts, Go decides. The contribution
count includes superseded rows on purpose, so dedup cannot discount the very
flood the quota exists to backstop.

### Honest gap

Usage is exported as structured `slog` records using OTel-named attribute
constants, but **no OTel span exporter is wired yet** — there was none before
this work either. PRD section 8 says "exported through OpenTelemetry"; that is
the attribute vocabulary today, not a running exporter. Recorded rather than
overclaimed.

## D-020 — Code anchoring ships multi-language, on a ctags-style regex registry, not tree-sitter

- **Date:** 2026-07-21
- **Status:** Decided (supersedes the "code anchoring is deferred" half of D-018)
- **Evidence:** [code-anchoring-prior-art.md](code-anchoring-prior-art.md), [code-anchoring-competitors.md](code-anchoring-competitors.md), [semantic-anchoring.md](spikes/semantic-anchoring.md)

### Decision

Code anchoring (L3 for `code` evidence) ships now, across many languages, on a
**table-driven regex registry modeled on universal-ctags' optlib parsers**:
adding a language is adding a row of patterns, not code. It runs on the stdlib
`regexp` (RE2) — pure Go, zero CGO, no per-language dependency. A regex
name-capture is tier 1 (a relocatable symbol path that survives line moves);
the enclosing definition's normalized bytes are tier 2. A line matching no
pattern, or a whole language not in the registry, degrades to tier-4-only and
never expires a claim by accident.

Default coverage: Go, TS/JS, Python, Rust, C, C++, Java, C#, Ruby, PHP, Swift,
Kotlin, Scala, CSS, HTML — kinds (function/method, class/struct/interface/
enum/trait, type/typedef, module/namespace, const/var, macro) drawn from ctags.

### Why the registry, not the pure-Go parser D-018 cleared

D-018 disproved the CGO gate: `gotreesitter` parses Go under `CGO_ENABLED=0`.
But it is the wrong default for *breadth*. It is v0.x single-maintainer, ~10 MB
of embedded grammars, and each language is one more grammar whose fidelity must
be vetted — the opposite of "be dynamic, cover everything." The real world is
C, C++, Rust, HTML, CSS and a long tail; a regex registry covers that tail as
*data* today, where a parser covers it as *a dependency per language* later.

The two are fidelity tiers, not rivals — the exact structure Sourcegraph ships
(ctags for ~40-language breadth, SCIP/tree-sitter for per-language fidelity).
Regex is the breadth default; pure-Go tree-sitter stays a documented, opt-in
per-language upgrade for a language that earns the 10 MB.

### What prior art forces us to say honestly

The concept is **not novel**. GitHub Copilot Memory (public preview, Jan 2026)
already stores repo facts with citations to code and validates those citations
against the current branch before use, with a 28-day TTL. Pitching CRED as
"first" or "unprecedented" is falsified on contact. What is genuinely unclaimed
is the *combination*: a relocatable symbol-path anchor **plus** a normalized
node hash that tells reformatting (expire nothing) from a semantic edit (expire
exactly what changed), delivered open, self-hostable, deterministic, and
inspectable. The honest frame is **reformat-immune, symbol-granular, open,
self-hostable** — every word true — not "first."

Among agent-memory systems (mem0, Letta, Zep/Graphiti, Cognee) none anchor a
claim to a code span and expire it on a code change; Cognee anchors to
file+line+symbol but carries no staleness. So CRED is ahead *of that category*
while behind Copilot's closed version on the concept alone.

### What we borrow

SCIP's `Symbol` string — a position-free descriptor FQN — is the cleanest
analog for tier 1, and confirms the design instinct: identity is a symbol path,
never a line range. Graphiti's "a detector nominates, deterministic code
decides" split is the same L2 boundary CRED already runs, now applied to code.

### The one RE2 caveat carried forward

RE2 has no backreferences and no lookaround. Patterns ported from ctags optlib
that rely on either must be rewritten for RE2 before use; the shipped set is
authored RE2-native and precision-tuned, because a false-positive anchor
(re-validating against the wrong code) is worse than a missed one.

## D-021 — A React portal, and CRED grows from a CLI into a self-hosted console

- **Date:** 2026-07-21
- **Status:** Decided (stack chosen by the operator after research; scope stated by the operator)
- **Evidence:** [portal-monorepo-stack.md](portal-monorepo-stack.md), [portal-api-and-frontend.md](portal-api-and-frontend.md). Ten Go-with-UI projects surveyed first-hand (Gitea, Prometheus, Grafana, CockroachDB, Vault, Consul, SigNoz, Coder, Woodpecker, Syncthing).

### What the portal is for

A CLI-plus-Postgres tool cannot be measured or run for a week: there is no
surface to see a claim, why it ranked, why it expired, or who sees it. The portal
is that instrument. The scope the operator named is larger than a read-only
dashboard — usage/limit management, analytics and charts, team management, SSO,
project management, file upload. That makes it a self-hosted management console,
not a viewer, and it is what justifies a real typed SPA over a hypermedia page.

### Layout — Go-first, not a JS monorepo

The research corrected the first instinct. All ten surveyed Go+UI projects keep
the **Go module at the repo root** and the **frontend in one top-level directory**
(`web/` or `ui/`); none uses an `apps/web` + `apps/api` workspace — that is a JS
convention absent from the Go cohort. CRED follows the cohort: Go stays at root,
the SPA lives in `web/`, a thin `Taskfile.yml` orchestrates both toolchains, and
a plain `package.json` (no Nx/Turborepo/pnpm-workspace — one app is far below
where any pays off).

### Serving — the single binary survives

`cred web` serves the JSON API, the built SPA embedded via `go:embed`, and an SPA
history fallback, from one CGO-free static binary. The embed sits behind a build
tag (`-tags embed`) with a disk/stub fallback, so `go build` and `go test` run on
a fresh clone before `vite build` has produced `web/dist` — the pattern Vault,
Prometheus, Coder and Cockroach all use. The embedding `.go` file lives at the
repo root because `go:embed` forbids `..`, so it can reach `web/dist`. In dev,
the Vite server proxies `/api` to the Go process. Content-hashed assets get
`Cache-Control: immutable`; `index.html` gets `no-cache`.

### Stack — the operator chose the typed React path

- **API:** Huma on the stdlib `net/http` adapter (`humago`) — the Go 1.22+
  ServeMux is enough router, and Huma auto-generates an OpenAPI spec at
  `/openapi.json` from Go structs (code-first, one source of truth).
- **Type contract:** hey-api generates a typed TypeScript client from that spec,
  so a URL/method/shape mismatch fails at compile time rather than at runtime.
- **Frontend:** Vite + React + TypeScript, **Vitest** + @testing-library/react,
  TanStack Query (server state), TanStack Table (grids), TanStack Router (typed
  routes suit a filter-driven console), shadcn/ui + Tailwind (own the components,
  low lock-in).

The alternative the research leaned toward for a read-mostly surface — templ +
htmx — was set aside deliberately: the named scope (SSO, uploads, charts,
project management) is interactive and stateful enough that the SPA's cost buys
capacity that will be used, and the team's skill is React.

### Foundation first, features as slices

The foundation is invariant to the feature list: an app shell with typed routing
and navigation, the Huma→hey-api typed pipeline, an **auth middleware seam**
(principal via header/token now, OIDC/SSO later without re-architecting), the
embed+fallback serving, and one real vertical screen (the claims browser over the
existing store read models) to prove the pipeline end to end. Everything the
operator named — limits, analytics/charts, team, SSO, projects, uploads — lands
as a later slice against this foundation, each with its own API endpoints and
screen. Recorded so the growth is planned, not improvised.

### Honest scope note

This enlarges CRED beyond "parity with mem0 plus a team layer." The engine
thesis (a claim lives only while its evidence does) is unchanged; the console is
new surface area, sequenced behind a foundation so it does not destabilize the
verified core.

## D-022 — Layout stays Go-first; one binary; Astryx is the UI

- **Date:** 2026-07-21
- **Status:** Decided (refines D-021 after considering an apps/ monorepo)
- **Evidence:** [portal-monorepo-stack.md](portal-monorepo-stack.md), [astryx-ui.md](astryx-ui.md); real Go tools (Vault, CockroachDB, Gitea) that ship one binary with a `server` subcommand.

### Layout: Go at the root, `web/` beside it — an `apps/` split was considered and dropped

An `apps/server` + `apps/web` symmetry was weighed and rejected. All ten surveyed
Go+UI projects keep the Go module at the repo root with the frontend in one
directory; moving Go under `apps/` is aesthetics, and it has a real cost —
`go:embed` cannot cross `..`, so an `apps/server` embed would need `web/dist`
copied into it before every tagged build. At the root, the embed file reaches
`web/dist` directly. So: **Go module at repo root, SPA in `web/`**, a thin
`Taskfile.yml` orchestrates both, no JS monorepo tool.

### One binary, subcommands — CLI, MCP, and web are not split

`cred` stays a single binary. The MCP server (`cred serve`), the web console
(`cred web`), and the curate worker (`cred curate`) are subcommands, not separate
binaries or modules. Splitting them into their own Go modules would force the
engine packages out of `internal/` into a shared public module plus a `go.work`,
and would break the one property the project sells — one static binary plus
Postgres, self-hostable. Vault (`vault server`), CockroachDB (`cockroach
start`), and Gitea (`gitea web`) all ship one binary; CRED follows them. If a
light-client/heavy-server split is ever needed, it is multiple `cmd/`
entrypoints in the *same* module, not separate modules — not now.

### UI is Astryx, not shadcn

The UI layer is **Astryx** (Meta, `@astryxdesign/core` + `@astryxdesign/theme-neutral`,
React 19), replacing the shadcn/ui + Tailwind of D-021. Path A: import Astryx's
pre-compiled CSS and use its components + tokens — **no StyleX build step**, so
`vite build` still emits ordinary static assets and the `go:embed` path is
untouched. Astryx's `Table` (with sort/filter/pagination hooks) replaces TanStack
Table; TanStack Query and Router stay. Charts use Astryx's own (canary)
`@astryxdesign/charts` directly, to keep one design system and token set; Recharts
is the fallback only if the canary package proves unusable.

Astryx is ~4 weeks old (v0.1.x, weekly releases). The operator chose latest,
unpinned versions over the research's pin-for-stability lean — accepted because
our source touches only React components and token overrides, so a revert to
shadcn would be a UI-layer swap, not an architecture change. `npx astryx upgrade`
is the ritual for the 0.1.x churn.

## D-023 — The console API is Gin + tygo, not OpenAPI

- **Date:** 2026-07-21
- **Status:** Decided (reverses the Huma/OpenAPI choice in D-021)
- **Evidence:** operator decision after weighing whether OpenAPI generation is needed for a console whose only client is our own SPA.

### Decision

The web API is plain **Gin** handlers with typed request/response structs. **No
OpenAPI, no Huma.** TypeScript types are generated from the Go structs with
**tygo** into `web/src/api/types.ts`; routes and fetch calls are hand-written and
centralized in one `web/src/api/` module over a single route-constant table.

### Why the reversal

D-021 chose Huma → OpenAPI → a generated typed client. That buys three things —
a client that type-checks *routes* as well as shapes, free request validation,
and a Swagger spec third parties can consume. All three are **integration-surface
value**, and this API's only consumer is our own React console. Paying a
framework plus a codegen pipeline for value we do not use is the ceremony this
project avoids. tygo gives the one thing we actually want here — shape safety
across the Go/TS boundary — with a single tool and no framework.

Gin is the router by operator preference: the largest Go web ecosystem and the
familiar choice, where stdlib `net/http` would also have sufficed.

### Cost, named

A mistyped route is a runtime 404, not a compile error — the one safety OpenAPI
would have added. Mitigated by centralizing every endpoint as a typed function
over a route-constant table, so a URL exists in exactly one place. If a
documented external API ever materializes (D-013's integration/metered
distribution), OpenAPI earns its keep then and can be added over the Gin handlers
without reworking the console.

## D-024 — The v0 memory-vs-long-context experiment is dropped, unrun

- **Date:** 2026-07-21
- **Status:** Decided
- **Evidence:** operator instruction to remove `v0/` after confirming it had
  produced no result. See `docs/research/spikes/v0-experiment-design.md` §13
  for the design and the state it was left in.

### Decision

The `v0/` harness is deleted from the repository. The experiment it implemented
— PRD §11/§13's gate, "does retrieved memory beat plain long context?" — is
**not going to be run.** `v0-experiment-design.md` is marked abandoned rather
than executed.

### Reasoning

The harness was built through corpus assembly (`mine`, `corpus-fetch`,
`corpus-build`, `index` all completed against three public repositories) but
never produced a single live model call. `draft` — the first stage needing a
model — was blocked on Anthropic credentials, which were never supplied. The
harness had no OpenAI-compatible provider, so the DeepSeek key later configured
in `.env` for the product's own `curate` command could not have run it either
without new provider code. Operator judgment: continuing to invest in a
harness that has sat unrun is not worth it relative to moving on.

### What this rules out

- Any claim that CRED's structured-memory approach has been measured against
  long context in its own regime. It has not. The only located outside
  evidence remains the fastpaca/MemBench citation, and §1 of the now-abandoned
  spike already found that citation too weak to lean on (4,232-token
  long-context arm, single unreplicated blog post, extrapolated Zep cost
  figure).
- Reusing `v0/`'s corpus, mining, or task-drafting machinery — it is deleted,
  not archived. Re-deriving it later is a rebuild, not a resume.

### What this forces

- PRD §11/§13 and `CLAUDE.md`'s gate table describe a gate that will not be
  cleared by measurement. `CLAUDE.md` is updated in this same change to record
  the gate as dropped rather than pending.
- If retrieved memory turns out not to beat long context in practice, CRED
  will find that out from real usage after release, not from a pre-registered
  experiment. That is a materially weaker falsification path than the one
  `v0-experiment-design.md` specified, and is accepted as a cost of this
  decision, not a hidden one.

### Open tension

This reverses the explicit purpose of `docs/research/spikes/v0-experiment-design.md`
(build and run the experiment before any product code) without the experiment
itself producing a KILL, AMBER, or PROCEED verdict — the decision is procedural
(unrun, deprioritized), not empirical. The PRD sections that motivated the gate
are not amended by this entry; they still describe an experiment that will not
happen. A reader relying on the PRD alone without also reading this entry would
be misled about project state.
