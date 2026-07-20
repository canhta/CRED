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

An unreplicated but **unrebutted** benchmark reports Mem0 and Zep as **14–77x
more expensive and 31–33% less accurate than plain long context**.

If that holds, CRED is wrong — and so are Memco, Glen, Mem0, and Zep. It is
cheap to test and existential. It runs before anything else is built.

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

## D-002 — Evidence discipline

- **Date:** 2026-07-20
- **Status:** Decided

All competitive scans, spikes, and benchmarks are captured as durable documents
under `docs/research/`, with claims tied to concrete file references or cited
URLs. Verbal conclusions that are not written down do not count as evidence.

Repository scans live in `docs/research/evidence/`.
