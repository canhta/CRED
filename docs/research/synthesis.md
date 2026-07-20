# Research Synthesis

- **Date:** 2026-07-20
- **Status:** Discovery complete, awaiting scope decision
- **Sources:** 10 evidence documents in [evidence/](evidence/) — six repository
  code scans plus market, graveyard, prior-art, and decay research.

---

## Bottom line

**The original thesis is substantially disconfirmed.** One wedge survives, in
revised form. The recommendation is to narrow, not to stop.

The value of this phase was finding this out before writing code.

---

## What the evidence killed

### 1. The space is not empty

| Competitor | Position |
|---|---|
| **Glen** (YC 2026) | Sells CRED's core loop nearly verbatim, including distilling work into skills for the next agent |
| **Memco** | Governed shared memory, SOC 2, on-prem |
| **Tessl** | $125M; abandoned spec-driven development for an Agent Enablement Platform with governance and a 3,000+ skill registry |
| **Augment Code** | Cosmos, with an Expert Registry |

CRED's four differentiation claims are the category's shared pitch, not a
differentiator.

### 2. Cross-agent handoff has no demand

Verified via npm and GitHub APIs:

- Rules-sync tooling: **~985,000 downloads/month**
- Handoff tooling: **~200**

This is not greenfield. It is absence of demand. **Wedge B is dead.**

### 3. Governance is not what users want

In the Claude Code repository, a governance proposal drew **7 reactions**; plain
interop drew **5,708**.

Anthropic has already shipped semver dependency resolution, private
marketplaces, and non-removable Required plugins. GitHub Copilot Memory and
Gemini Auto Memory ship most of this, and **Google already built the review
gate**.

### 4. Context is becoming a free file format

**AGENTS.md is Linux Foundation-stewarded, 60,000+ projects, supported by every
major vendor.** Cursor and Factory were founding contributors — funded companies
that standardized away their own differentiator.

**A rules compiler is not a business.** It sits directly in the path of a
standard, and it is a weekend project.

### 5. Discovery is not the pain

Stack Exchange raw survey data, 65,437 responses, distributions independently
recomputed:

- **70.4%** already know where to find answers
- **48.8%** believe what they find is up to date

The defensible pain is **freshness**, not findability. This is further evidence
against the retrieval/knowledge wedge.

---

## The sharpest finding

**CRED's core loop inverts its own evidence base on all four axes.**

| Research says the artifact that works is | The PRD describes |
|---|---|
| Small | Large |
| Factual | Narrative |
| Machine-derived | Human-curated |
| Expiring | Accumulating |

Supporting measurements:

- Adherence approaches **zero above ~6,000 words**.
- Context files grow **4x faster than they shrink**, daily.
- **ADRs — the closest prior art — starve at 1–5 records in half of repos.**
- Vercel measured **53% → 100%** for *verifiable facts outside training data* —
  the opposite of reviewed narrative learning.

---

## Two design laws derived from evidence

### Law 1 — LLM nominates, deterministic code decides

Graphiti implements this (`edge_operations.py:538-572`): the LLM only nominates
contradictions as validated integer indices; expiry is deterministic interval
arithmetic in Python, and nothing is deleted.

Mem0 violated it and had to remove its reconciler — the famous ADD/UPDATE/DELETE
prompt now has **zero call sites**, leaving an ADD-only write path where
contradictions accumulate. Letta violated it with last-writer-wins on shared
blocks (`agent_manager.py:1769`).

Moderne — the healthiest company in the adjacent scan — is built on exactly this
principle applied to refactoring.

### Law 2 — Review must be event-triggered, never a queue

Prior art condemns **ambient voluntary maintenance by a general population**, not
curation as such. Google's postmortem culture sustains because it is
event-triggered, privately rewarded, and consumed by people who feel pain when
it is wrong.

Independently confirmed by capacity data: LinearB (8.1M PRs) shows AI PRs accept
at **32.7% vs 84.4%** and wait **4.6x longer** — roughly **2.6x the review
capacity per merged unit**. A second approval queue will never be serviced.

---

## What survives

> **Harness evaluation as pre-merge regression testing.**
>
> When someone changes `CLAUDE.md`, `AGENTS.md`, or a skill, run it against a
> standard task set and **block the merge if quality regresses**. Framed as
> organizational blast radius, not individual curiosity.

Why this survives when the rest does not:

- It is **CI, not a review queue** — satisfies Law 2.
- It **measures**, which is the stated pain and the only defensible layer.
- It **does not compete with AGENTS.md** — it rides it. The more standardized
  the format, the more people need to know whether a change broke something.
- Anthropic and GitHub have **not** shipped it, because it lives at the CI layer
  rather than the agent layer.

Accompanying principles: **ship facts, not prose; expiry default-on;
machine-derived, human-confirmed in place.**

---

## Where the money is, if there is money

Every funded survivor monetizes **administration and enforcement across an
organization** — never the context itself.

- Devin gives DeepWiki away as top-of-funnel.
- Amp passes LLM cost through at **zero markup**, charges enterprise **+50% for
  SSO and managed settings**.
- Cursor and GitHub gate **org-level rules** to paid tiers.

This preserves the adoption ladder in D-003 but relocates value capture: **rung 1
must be free and stay free**, because incumbents have already made it free.
Rung 3 — org-level enforcement and measurement — is what four independent
companies converged on charging for.

---

## Honest counter-evidence

Retained deliberately so it is not rediscovered as a surprise.

1. **OSS traction in this category does not convert.** Five projects with 23k+
   stars each; none disclosing revenue. Total disclosed funding across all
   dedicated agent-memory startups is ~$50M — less than half of LangChain's
   single Series B. Gross margins are capped at 50–60% by someone else's price
   list, with variable costs within 10–15% across all competitors.
2. **`mdarena` already built the recommended wedge. It has 65 stars.** Being
   directionally right does not produce adoption.
3. **CodeSee** — the closest structural analogue — shut down in Feb 2024 despite
   tier-1 investors.
4. **Tailwind Labs cut 75% of engineering in Jan 2026** after AI made its curated
   artifact generatable. Test to apply: *if a frontier model can regenerate the
   artifact from the repository on demand, the artifact is not the product.*

### The counter-counter-evidence

Wikipedia's committed core (100+ edits/month) **grew 23%** since 2015 while total
editors fell 29%. A power law is not decay. Curation sustains when it is
engineered rather than hoped for.

---

## Data integrity flags

**One research agent fabricated citations before self-retracting.** The lead
agent independently re-verified load-bearing numbers and documented the
retractions, but **figures not explicitly marked verified must be spot-checked
before external use.**

Additionally unverified or falsified:

- **All GitClear figures** — domain returns 403; zero figures confirmed.
- **"70% of KM projects fail"** — **falsified at source**. Storey & Barnett
  (2000) contains no such number.
- **Kapa.ai acquisition** — false; it operates independently.
- **Sourcegraph layoffs** — unsubstantiated across two independent passes.
- **DORA 2025 exact figures** — reports exceed fetch limits; all circulating
  numbers trace to summary blog posts.
- Several 2026 items are headline-level from news RSS after publishers blocked
  article fetches.

---

## Open decision

Four paths, presented to the founder on 2026-07-20:

1. **Narrow** — drop the current PRD; build only pre-merge harness regression
   testing. *(Recommended.)*
2. **Pivot** — keep organizational ambition, change the artifact to small,
   machine-derived, self-expiring facts.
3. **Stop** — the evidence is bad enough to seek a different problem.
4. **Proceed as planned** — valid only if there is specific founder knowledge the
   research did not capture, which should be stated and tested.

### Why Narrow is recommended

Seta provides several teams, several repositories, and real harness churn —
meaning a **real evaluation dataset**. That, not the idea, is the founder's
genuine advantage, and it is the input pre-merge regression testing requires.
