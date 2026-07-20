# Knowledge Decay, Review Capacity, and the Real Shape of the Pain

## Provenance warning

**This document did not originate from CRED's commissioned research.** It
consolidates three background tasks (`Review fatigue and AI PR overload`,
`Doc rot and KM failure evidence`, and a related relay) that were not launched
as part of this discovery effort.

The originating agents graded their own claims and falsified several. That
grading is preserved. Claims marked unverified must not be used.

- **Captured:** 2026-07-20
- **Status:** Third-party input, substantially verified with primary sources

---

## Finding 1 — Discovery is not the pain. Freshness is.

Stack Exchange's official raw survey dataset was pulled directly
([schema](https://github.com/StackExchange/Survey/raw/refs/heads/main/packages/archive/2024/schema.csv),
[65,437 responses](https://github.com/StackExchange/Survey/raw/refs/heads/main/packages/archive/2024/results.csv))
and distributions recomputed independently. The replication matched Stack
Overflow's own published figures within rounding, which validates the method.

| Survey item | Verbatim question | Agree |
|---|---|---|
| **Knowledge_3** | "I can find **up-to-date** information within my organization to help me do my job." | **48.8%** |
| **Knowledge_5** | "I know which system or resource to use to find information and answers." | **70.4%** |
| Knowledge_6 | "I often find myself answering questions that I've already answered before." | 48.9% |
| Knowledge_7 | "Waiting on answers to questions often causes interruptions." | 53.0% |
| Knowledge_2 | "Knowledge silos prevent me from getting ideas across the organization." | 45.2% |
| TimeSearching | >30 min/day searching for answers | 63.7% |

### Why this pair is decisive

**70.4% of developers already know where to look. Only 48.8% trust that what
they find is current.**

Knowledge_3 is the most on-point statistic in the entire survey for a knowledge
product, and **Stack Overflow headlines it nowhere.**

This is direct evidence against a **discovery / search / retrieval** framing —
which is exactly the crowded wedge C the scans already warned against. The
defensible pain is **freshness and staleness**, not findability.

### Corroborating decay data (verified, primary)

Tan, Wagner & Treude, EMSE 29:5 (2024),
[doi:10.1007/s10664-023-10397-6](https://doi.org/10.1007/s10664-023-10397-6):

- **28.9%** of the most popular GitHub projects currently contain at least one
  outdated reference.
- **82.3%** were outdated at least once in their history.
- Mean staleness: **4.7 and 4.2 years**.
- After one month, **45–55%** of references that were eventually fixed were
  still outdated.
- **13.3%** of "fixes" were performed by deleting the documentation.

Also: [GitHub OSS Survey 2017](https://opensourcesurvey.org/2017/) — 93%/60%
on incomplete documentation; Ratol & Robillard,
[doi:10.1109/ase.2017.8115624](https://doi.org/10.1109/ase.2017.8115624) —
">half of identifiers had fragile comments after renaming."

---

## Finding 2 — There is no spare review capacity. Do not create a queue.

**LinearB 2026 Software Engineering Benchmarks** —
[linearb.io](https://linearb.io/resources/software-engineering-benchmarks).
Sample: **8.1M+ pull requests, 4,800 teams, 42 countries.**

> "Acceptance Rates for AI-generated PRs are significantly lower than manual PRs
> (**32.7% vs. 84.4%**)"
> "AI PRs wait **4.6x longer** before review"
> "Agentic AI PRs have a PR Pickup Time **5.3x longer** than Unassisted ones"

**Reviewing AI output costs roughly 2.6x the review capacity per merged unit of
work.** Vendor-sourced, but the sample is disclosed and is the largest found.

Non-vendor corroboration (abstracts fetched):

- [arXiv 2604.00917](https://arxiv.org/abs/2604.00917) — ~110,000 OSS PRs across
  Codex, Claude Code, Copilot, Jules, Devin: agent contributions "are associated
  with more churn over time compared to human-authored code."
- [arXiv 2607.13196](https://arxiv.org/abs/2607.13196) — 1.02M reviewed PRs, 207
  projects: agentic review gave "faster review decisions," but "these efficiency
  gains do not translate into better review quality."
- [arXiv 2603.11078](https://arxiv.org/abs/2603.11078) (CR-Bench) — review agents
  "exhibit a low signal-to-noise ratio when designed to identify all hidden
  issues."

### Direct consequence for CRED

Any design that adds a **second review queue** — a knowledge-approval or
harness-approval backlog — competes for capacity that is already 2.6x
oversubscribed by AI-generated code review. It will not be serviced.

This independently confirms the design law derived in
[prior-art-voluntary-curation.md](prior-art-voluntary-curation.md): review must
happen **inside the task**, never in a separate queue.

---

## Finding 3 — Two arguments to abandon (they are false)

Both were part of the adversarial brief and did not survive contact with
evidence. Retained explicitly so they are not re-invented later.

### "Reviewers rubber-stamp AI output"

[arXiv 2606.26505](https://arxiv.org/abs/2606.26505), *"Same Scrutiny, More
Time"* — a Wizard-of-Oz eye-tracking study:

> "while the thoroughness of code review did not change for participants, they
> spent **more** time fixating on LLM-labelled code"

Reviewers did **not** degrade into rubber stamps. They spent more time at equal
thoroughness. This supports the *capacity* argument and **refutes** the
*degradation* argument. Drop the latter entirely rather than softening it.

### "70% of knowledge management projects fail"

**Falsified at the source.** Storey & Barnett (2000), JKM 4(2):145-156,
[doi:10.1108/13673270010372279](https://doi.org/10.1108/13673270010372279) — the
paper this statistic is habitually hung on — was fetched. It says only:

> "A large proportion of such initiatives will fail."

No number, no percentage, no cited survey. It is a single case study. The
"Gartner 70%" attribution is untested. **Do not use this statistic.**

---

## Explicitly unverified — do not cite

- **All GitClear figures.** `gitclear.com` returns HTTP 403 on every path; both
  S3 mirror PDFs are redirect stubs. Zero figures verified. Substitute
  arXiv 2604.00917, which makes the churn point with a fetched dataset.
- **DORA 2025 exact figures.** Both report PDFs exceed the 10MB fetch limit.
  Every DORA number in circulation traces to Google summary blog posts, not the
  reports — no confidence intervals or model specifications seen.
- **Faros AI's "5x review time"** — no disclosed methodology or sample. Drop it;
  LinearB says something similar with 8.1M PRs behind it.
- **Wen et al. ICPC 2019** co-evolution body percentages — IEEE paywalled, no OA
  copy, all author-site paths 404. Do not reconstruct from memory.
- **Confluence/Notion decay statistics — genuine null.** Searches returned only
  startups *selling* doc-rot tools (Specsight, Drift, DocDrift). That is a
  market signal, not evidence. Assume any circulating number is vendor
  marketing until traced.

## Miscitation warning

Fluri et al. 2007 ([doi:10.1109/wcre.2007.21](https://doi.org/10.1109/wcre.2007.21)),
"97% of comment changes are done in the same revision," is a **pro**-co-evolution
finding and is routinely miscited as doc-rot evidence. The usable result from
that paper is different: new code barely gets commented at all.

## Adversarial sources not yet read

Both target this thesis directly and should be read before it faces a prepared
skeptic:

- `thenewstack.io/ai-code-bottleneck-myth/` — "AI hasn't shifted the bottleneck
  from coding to code review" (2026-07-18)
- [arXiv 2606.13175](https://arxiv.org/abs/2606.13175) — "The End of Code Review:
  Coding Agents Supersede Human Inspection"
