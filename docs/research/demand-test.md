# Demand test

The instrument for the cheapest disconfirmation available to this project.

D-004 rests on one sentence: *organizations want organizational agent memory,
and the binding constraint on adoption is that it must run on their own
infrastructure.* D-007 records the strongest evidence against it — roughly 20 of
30 HN results on the topic are founders promoting their own memory products, a
seller-to-buyer ratio that describes a seller-imagined market more often than an
early one. The Context7 scan landed on the same gap from a different direction:
distribution mechanics are reproducible by a small team, but nothing in that
case study speaks to whether they work without pre-existing demand.

Two independent lines of evidence now point at the same unknown. This document
is how it gets tested.

## What this test can conclude

It can **kill D-004**. If teams running agents report that they would happily
use a hosted shared-memory product and simply have not found one worth paying
for, the sovereignty premise is false and the direction is wrong — not merely
suboptimal, because sovereignty is what separates CRED from competitors that
lead on funding, compliance, and time.

It **cannot green-light the project**. Stated interest is not demand. The most
this instrument produces is the absence of a disqualifying answer, plus a list
of people who described a real problem in their own words.

## Design rules

These exist because the obvious version of this survey returns a false positive
every time.

**Ask about the past, never the future.** "Would you use a tool that shares
context across your team?" returns yes from nearly everyone and means nothing.
"What happened the last time two people on your team gave an agent contradictory
context?" returns either a story or a silence, and both are informative.

**Never describe CRED before the questions.** The moment a respondent knows what
is being sold, they answer the sales pitch instead of the question. The product
is described at the end, or not at all.

**Prefer evidence of spend and workaround over evidence of complaint.** People
complain about everything. A team that built an internal script, maintains a
shared prompt file by hand, or pays for something adjacent has revealed a
priority. A team that agrees the problem sounds annoying has revealed nothing.

**Record disconfirming answers verbatim and in full.** Per D-002 and the
evidence standard, an answer that contradicts the thesis is the valuable kind
and must survive into the written record unsummarized.

**Disclose the conflict.** The interviewer is building a product in this space.
Say so at the end, never at the start.

## Population

The test is only meaningful among teams **already running coding agents at
work**, with more than one person touching the same codebase. Everyone else is
answering a hypothetical.

Sourcing, in rough order of evidential value:

| Source | Why | Bias to disclose |
|---|---|---|
| Seta International teams | Real teams, observable behavior, follow-up possible | Employer relationship; cannot be the only source |
| Teams found via public complaints about agent context | Self-selected for having the problem | Self-selection inflates the result |
| Adjacent-tool users (harness sync, rules management) | ~985k npm downloads/month is real usage | These are individual users; team evidence is weaker |
| Cold outreach to engineering leads | Least biased | Lowest response rate |

Seta is available and is the natural first site, but a result drawn only from
Seta measures Seta. D-001 already ruled out building to one organization's
requirements; the same logic applies to measuring demand from one.

## The questions

Ordered so that the least leading come first. Conversational, not a form —
follow the interesting answer rather than the script.

### 1. Current behavior

1. Who on your team uses a coding agent, and for what kinds of work?
2. Walk me through how an agent on your team knows things about your codebase
   that are not in the code — conventions, past decisions, why an approach was
   rejected.
3. Who maintains that, and when was it last updated? *(If there is a file, ask
   to see its git history. Stale files are the finding.)*

### 2. Evidence of pain, without naming it

4. Tell me about the last time an agent confidently did something your team had
   already decided against.
5. What happened? Who caught it, and how long did it take?
6. Has anyone on your team built something to fix this? What is it, and is it
   still running? *(A dead internal tool answers more than a live complaint.)*
7. When someone joins, or leaves, what happens to what they knew?

### 3. The load-bearing question

Asked plainly, because the whole direction turns on it:

8. Suppose a product existed that gave every agent on your team the same
   accumulated knowledge about your work, and it worked well. Would you use a
   hosted version, where that knowledge lives on the vendor's servers?
9. **If no — what specifically stops you?** Do not accept "security" as a final
   answer; ask what would have to be true, who would have to approve it, and
   whether a comparable product already cleared that bar.
10. **If yes** — that is a hit against D-004. Record it verbatim and probe: is
    there a category of knowledge you would still not send? Where is the line?

### 4. Spend

11. What do you currently pay for, per developer, in this area?
12. Who signs off on a new tool at your size, and what does that process look
    like?
13. Has your team ever removed a developer tool after adopting it? Why?

### 5. Only at the end

14. Describe CRED in two sentences. Ask what is wrong with it. The objection
    raised first is the one that matters.
15. Ask for one referral to someone with the same problem.

## What gets recorded

One file per conversation under `docs/research/evidence/demand/`, with:

- Role, team size, agent tooling in use, date. No names without consent.
- Sourcing channel, so self-selection is visible in the aggregate.
- **Verbatim quotes for anything load-bearing.** Paraphrase is not evidence.
- The answer to Q8 and Q9, marked, because those are the ones D-004 turns on.
- Interviewer's own read, in a clearly separated section, so the observation and
  the interpretation never blur.

## Pre-registered reading

Written before the first conversation so the result cannot be reinterpreted
afterward. Target **n = 12 to 15** teams — enough that a lopsided split is not
noise, few enough to run in three weeks.

| Result | Reading |
|---|---|
| Most teams would use a hosted product and cite no blocker | **D-004 is falsified.** Sovereignty is not the constraint. The direction needs revisiting before engineering, not after |
| Teams describe the pain vividly but have built nothing and pay for nothing | Real annoyance, not a budget. Consistent with D-007's seller-heavy market. Proceed only as OSS with monetization deferred indefinitely |
| Teams cannot recall a concrete incident at all | The problem is founder-imagined. This is the outcome the instrument exists to be able to return |
| Teams cite a specific blocker to hosted adoption, name who must approve it, and have a workaround already running | D-004 survives its first real test |

A result where teams say they are interested but report no incident, no
workaround, and no spend is **not** a positive result, and must not be written
up as one.

## Cost

Three weeks calendar, roughly a day per week of interview time, no money. It is
the cheapest test in the project and the only one that can address the risk
D-007 names as the most likely to be wrong.

It runs in parallel with the [v0 experiment](spikes/v0-experiment-design.md),
which asks a different question: v0 asks whether the product would work, this
asks whether anyone wants it. Either can kill the project alone.
