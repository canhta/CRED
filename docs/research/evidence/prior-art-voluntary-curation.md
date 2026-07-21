# Prior Art — Does Voluntary Knowledge Curation Sustain?

## Provenance warning

**This document did not originate from CRED's commissioned research.** It
arrived from a background task (`Prior art voluntary contribution`) that was not
launched as part of this discovery effort.

It is retained because it addresses CRED's single largest untested assumption,
which none of the seven commissioned scans covered. The originating agent
performed its own provenance audit and downgraded several of its claims; that
grading is preserved below and must be respected.

- **Captured:** 2026-07-20
- **Status:** Third-party input, partially verified

---

## Why this matters to CRED

The PRD's core loop terminates in **"reviewed organizational learning."** That
word *reviewed* assumes humans will curate shared knowledge. This is the most
expensive assumption in the product and the least examined.

---

## Evidence against voluntary curation (verified — pages fetched)

### Stack Overflow's collapse

- **3,862 questions** posted in December 2025 — a **78% year-over-year drop**,
  and **98.1% below** the 200,000+/month peak of early 2014.
  [devclass](https://www.devclass.com/ai-ml/2026/01/05/dramatic-drop-in-stack-overflow-questions-as-devs-look-elsewhere-for-help/4079575),
  corroborated by [Wikipedia](https://en.wikipedia.org/wiki/Stack_Overflow)
- **The decline began in 2014 — well before ChatGPT.**
  [Pragmatic Engineer](https://blog.pragmaticengineer.com/stack-overflow-is-almost-dead/):
  "June 2020: questions start to decline, faster than before."
  This defeats the "it was just AI" rebuttal.
- A cited developer's diagnosis: "AI certainly accelerated the decline, but this
  is the result of consistently punishing users for trying to participate in
  your community." **Contribution friction, not competition, is the mechanism.**

### Convergent pivots away from human curation (2023–2026)

Three incumbents that spent a decade on voluntary human curation independently
converged on **automatic capture with humans demoted to validators**:

- **Stack Overflow for Teams → "Stack Internal"** — now leads with "Your AI is
  only as good as the context it's given," sells automatic capture, and reduces
  humans to "selective expert validation only where critical gaps exist."
  [stackoverflow.co/internal](https://stackoverflow.co/internal/)
- **Guru** — now sells "Knowledge Agents" that "verify and unverify information
  automatically." [getguru.com/about](https://www.getguru.com/about)
- **Notion** — AI (Feb 2023), AI agents in 3.0 (Sept 2025), and a May 2026
  developer platform letting external agents (Claude Code, Cursor, Codex) do the
  work. [Wikipedia](https://en.wikipedia.org/wiki/Notion_(productivity_software))

**Negative finding treated as a finding:** no published engagement, DAU, or
contribution-rate metric exists anywhere for Stack Overflow for Teams — six
years of marketing a contribution product with no contribution metric disclosed.

### Participation inequality baseline

Nielsen's 90-9-1 rule, primary source
([NN/g, 2006](https://www.nngroup.com/articles/participation-inequality/)):
"90% of users are lurkers… 9% contribute from time to time… 1% participate a lot
and account for most contributions." For Wikipedia he cites a far steeper
99.8 / 0.2 / 0.003 distribution.

### The graveyard

- **Google Knol** — beta 2008-07-23, closed 2012-04-30. "Two years after its
  inception, few people were aware of Knol's existence."
- **Apache Wave** — GA 2010-05-19, development suspended 2010-08-04, **under
  three months**. Google cited "a lack of interest."
- **Jive Software** — IPO Dec 2011 raising $161.3M; sold to Aurea in 2017 for
  $462M in what was described as a fire sale.

---

## The counter-evidence — stronger than expected

### Wikipedia's committed core is growing

Primary data, queried live from the Wikimedia REST API (English Wikipedia):

| Metric | Value |
|---|---|
| All editors, May 2026 | 283,193 |
| Editors with only 1–4 edits | 228,265 (80.6%) |
| Editors with 100+ edits | 5,808 (2.05%) |
| Total editors, Jan 2015 → May 2026 | 397,946 → 283,193 (**−29%**) |
| **100+ edit tier, Jan 2015 → May 2026** | **4,731 → 5,808 (+23%)** |

**The casual tier shrank by 29% while the committed core grew by 23%.**
A power law is not the same thing as decay.

### Why Google's postmortem culture sustains

[Google SRE Book](https://sre.google/sre-book/postmortem-culture/) — postmortems
survive because they are **engineered, not voluntary**:

- **Event-triggered:** "Postmortems are expected after any significant
  undesirable event," on objective triggers — not on goodwill or a schedule.
- **Privately rewarded:** a "rewarded and celebrated practice, both publicly…
  and through individual and team performance management," plus peer bonuses and
  postmortem-of-the-month.
- **Consumed by people who feel pain when it is wrong.**

Google converted a public good into a privately-rewarded, event-triggered one.

---

## Design law derived (proposed)

> The prior art condemns **ambient, voluntary maintenance by a general
> population**. It does not condemn curation as such.
>
> Therefore every review or approval step in CRED must be:
>
> 1. **Event-triggered** — fired by something that happened, never by a schedule
>    or by goodwill.
> 2. **Performed by whoever feels the pain if it is wrong** — not delegated to a
>    separate curator role.
> 3. **Cheaper than skipping it** — at the moment of the decision, not in
>    aggregate.

### Consequence for the PRD

Any design assuming teams will periodically review accumulated knowledge **will
fail**. If CRED requires human approval of knowledge, that approval must occur
**inside the task the person is already doing**, never in a separate review
queue.

This also reframes the Mem0 and Letta findings: both built governance
scaffolding and left it unwired. The likely reason is not incompetence — it is
that they placed curation outside the moment of pain, where nobody performs it.

---

## Explicitly unverified — do not cite as fact

The originating agent graded these as search-snippet only:

- Stack Overflow's Oct 2023 layoffs of 28% / 100+ staff (four outlets agree in
  snippets; none were opened).
- Almanac's shutdown (2025-01-31) and $43M raised.
- **The entire Cabrera & Cabrera "knowledge-sharing as public good" academic
  thread** — none of those papers were read. This framing is load-bearing and is
  currently unsupported.
- Slite's pivot — sourced from a competitor's blog. Discard unless confirmed.

## Found nothing

**Empirical enterprise-wiki participation concentration.** Two OpenAlex attempts
returned poor keyword relevance. Study titles exist (Arazy et al. on IBM
enterprise wikis; a Springer multi-case study; a Portsmouth study of 177 users)
but all are paywalled and **zero participation percentages were extracted**.
Flagged explicitly so no figures are invented to fill the gap. A next attempt
should query OpenAlex by DOI (`10.1007/978-3-642-19032-2_22`) rather than by
keyword.
