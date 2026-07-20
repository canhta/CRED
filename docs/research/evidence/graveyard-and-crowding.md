# Graveyard, Crowding, and the Commoditization of Context

## Provenance warning

**This document did not originate from CRED's commissioned research.** It
arrived from a background task (`Startup graveyard and crowding`) not launched
as part of this discovery effort. The originating agent exhausted its WebSearch
budget (200/200) and flagged several 2026 items as headline-level rather than
body-verified. That grading is preserved.

- **Captured:** 2026-07-20
- **Status:** Third-party input, mixed verification
- **This is the most disconfirming evidence gathered so far. It should be read
  before any scope is committed.**

---

## Finding 1 — The closest analogue to CRED is dead

**CodeSee** — "continuous codebase understanding" — announced shutdown
**Feb 2024** and was absorbed by GitKraken. It had tier-1 investors. It could not
stand alone as a product.

This is the nearest structural precedent to a context-understanding layer sold
as its own thing.

---

## Finding 2 — Curated artifacts die when AI can generate them

**Tailwind Labs cut 75% of engineering in Jan 2026** after a ~40% traffic drop.

This is the sharpest available precedent: enormous OSS distribution, a genuinely
good paid product built on **human-curated reference material**, destroyed when
AI made the curated artifact generatable on demand.

**Almanac** shut down Jan 2025 after ~$34M raised — its founders killed it
because their own AI content tool outgrew it within months.

**Stack Overflow** is down **~98% from peak** (3,862 questions in Dec 2025 vs
>200,000/month in early 2014). Prosus paid $1.8B and wrote it down by more than
half in July 2026.

### The pattern

A curated corpus is only valuable while it is expensive to produce. Every one of
these businesses was built on that assumption, and the assumption expired.

**Test to apply to CRED:** if a frontier model can regenerate the artifact from
the repository on demand, the artifact is not the product.

---

## Finding 3 — Organizational context is becoming a free file format

- **AGENTS.md is now Linux Foundation-stewarded**, adopted by **60,000+
  projects**, and supported by every major agent vendor.
- **Cursor and Factory were founding contributors** — funded companies that
  chose to **standardize away their own differentiator**.
- Cline's documentation explicitly describes Memory Bank as "a documentation
  methodology — **not a built-in feature**."

### Consequence

The **format** and the **compiler** are being commoditized to zero, deliberately,
by better-funded incumbents acting in concert.

This is decisive for the wedge question recorded in
the wedge analysis. **Wedge A as a compiler is not a
business.** Writing rules once and emitting `CLAUDE.md` / `AGENTS.md` / Cursor
rules is a weekend project sitting directly in the path of a Linux Foundation
standard.

---

## Finding 4 — What the survivors actually charge for

**Every funded survivor monetizes something other than the context itself:**

| Company | Gives away | Charges for |
|---|---|---|
| Devin | DeepWiki (top-of-funnel) | The agent |
| Amp | LLM cost passed through at **zero markup** | Enterprise: **+50% for SSO and managed settings** |
| Cursor | Rules format (standardized away) | **Org-level rules gated to paid tiers** |
| GitHub | Copilot instructions format | Org-level policy and enforcement |

> **Vendors monetize administration and enforcement across an organization —
> never the context itself.**

### Consequence

This simultaneously **invalidates** and **validates** parts of the current plan:

- **Invalidates rung 1 as a business.** The artifact and the compiler are free
  and standardized. They can be the adoption hook, but never the product.
- **Validates rung 3 as the business.** Org-level enforcement, administration,
  and measurement is precisely and exclusively what the market pays for. Four
  independent companies converged on this.

The adoption ladder in D-003 survives. What changes is where value is claimed:
**rung 1 must be free and must stay free**, because incumbents have already made
it free.

---

## Finding 5 — OSS traction in this category does not convert

- Total disclosed funding across **all** dedicated agent-memory startups:
  **~$50M** — less than half of LangChain's single Series B.
- **Five projects with 23k+ GitHub stars each. None disclosing revenue.**
- Gross margin ceiling of **50–60%**, set by someone else's API price list.
- Variable costs within **10–15%** across all competitors — no cost moat, only
  the conditions for a price war.
- 24 surveyed enterprise VCs converging on "more AI spend through **fewer**
  vendors."

### The direct challenge to strategy D-001

D-001 chose OSS traction first, on the reasoning that adoption is the only
durable early asset for a solo founder.

**This finding is the strongest counter-argument on record.** Five projects in
this exact category reached the traction milestone D-001 targets and converted
it into nothing measurable.

Stars are demonstrably **not** a leading indicator of revenue here. If CRED
pursues D-001, it must do so understanding that traction is a distribution
mechanism, **not** evidence of a business, and that the conversion step must be
designed deliberately rather than assumed to follow.

---

## Premises that did not survive checking

Both were in the originating brief and are corrected here so they are not reused:

- **Kapa.ai was not acquired.** Tracxn, PitchBook, and Crunchbase all show it
  operating independently at $3.7M raised. The claim likely confuses it with
  Docker, a customer with a published case study.
- **Sourcegraph layoffs could not be substantiated** across two independent
  passes (HN Algolia, Google News RSS, Bing News, layoffs.fyi, Wikipedia). No
  date or percentage should be assigned.

---

## Coverage gap

The **Windsurf / Augment / Poolside / Tessl / Tabnine / Grit.io / Moderne**
cluster is uncovered by this pass except for Windsurf's "very negative" margins
and the $2.4B Google shareholder payout. That cluster is where the 2025–26
fire-sales concentrated and is the most likely location of another
CodeSee-shaped precedent.

Partial coverage exists in
[adjacent-devtools-status.md](adjacent-devtools-status.md), which was captured
separately — including the material finding that **Tessl pivoted into harness
governance with $125M**.
