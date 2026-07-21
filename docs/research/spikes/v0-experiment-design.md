# v0 Experiment Design — Does retrieved memory beat plain long context?

- **Date:** 2026-07-20
- **Status:** **Abandoned, unrun.** See D-024 in `../decision-log.md`.
- **Gates:** PRD §11 (first experiment), §13 (v0 → v1) — **gate dropped, see
  `CLAUDE.md`**
- **Rule:** nothing in this document may be changed after the pre-registration tag
  is cut. See §6. (Moot: the tag was never cut.)

This is the experiment PRD §13 made the first deliverable. It shipped no
product code. It was designed to produce a kill decision or a proceed
decision, and both would have been publishable.

## Verdict: abandoned, not decided

**2026-07-21.** The `v0/` harness built against this design (see §11 for its
shape) completed corpus assembly — mining, corpus-fetch, corpus-build, and
indexing all ran against three public repositories — but never made a live
model call. The `draft` stage, the first to need one, was blocked on missing
Anthropic credentials for its full run; per operator instruction the harness
was deleted rather than completed. Full reasoning in decision-log.md D-024.

This is **not** a KILL, AMBER, or PROCEED outcome under §6. None of those
outcomes require data this design never collected. The gate this document was
meant to clear is recorded as dropped, not passed, in `CLAUDE.md`.

What would change this verdict: rebuilding the harness (it is not archived,
only the design below survives) and actually running it. Nothing in this
document is disqualified by the abandonment — the critique of the fastpaca
citation in §1, the design decisions in §2, and the threats-to-validity table
in §10 remain the best available analysis of how to measure this question, if
it is ever revisited.

---

*The section below is the original design, preserved as written. It describes
a harness that was built and then deleted before producing a result — read it
as a specification, not as a report of what happened.*

---

## 0. Bottom line

The 14–77x / 31–33% citation **was located.** It is a single blog post by one
person, with an open-source harness, four HN points, and one comment — the
author's own.

It is weaker than the PRD's framing in three specific ways, one of which is
disqualifying: **the long-context arm averaged 4,232 input tokens.** An
experiment where "long context" means four thousand tokens does not test long
context. It tests whether a memory system can beat a prompt that already
contains the whole conversation, and the answer to that is obviously no.

The claim is therefore **not yet a threat to CRED, and not yet a defence of it.**
It is unmeasured. That is what this experiment fixes.

**Recommended PRD amendment** (not applied — this spike does not edit the PRD):
§11 and D-007 should stop describing the benchmark as "unrebutted." It is
unread. Substituting attention for scrutiny is the same error the document
elsewhere warns about.

---

## 1. The original benchmark — located

### What it is

| Field | Value |
|---|---|
| Title | *Universal LLM Memory Does Not Exist* |
| Author | Sebastian Lund (HN: `cpluss`), fastpaca |
| URL | https://fastpaca.com/blog/memory-isnt-one-thing |
| Published | 2025-11-21 |
| Harness | https://github.com/fastpaca/pacabench — Apache-2.0, Rust |
| Benchmark | MemBench (arXiv:2506.21605) |
| Systems under test | Mem0 (vector), Zep (Graphiti) |

**VERIFIED** — blog fetched directly; repository metadata via `gh api`
(13 stars, 1 fork, last push 2026-02-03, `examples/membench_qa_test` present).
The blog's link to `fastpaca/agentbench` redirects to `pacabench`.

### Reported results

**VERIFIED** — table quoted from the post.

| Arm | Precision | Avg input tokens | Avg latency | Total cost |
|---|---|---|---|---|
| Long context | **84.6%** | **4,232** | 7.8s | $1.98 |
| Mem0 (vector) | 49.3% | 7,319 | 154.5s | $24.88 |
| Zep (Graphiti) | 51.6% | ~1.17M | 224s | ~$152.6 |

Zep's run was **aborted at 1,730 of 4,000 cases after 9 hours** on cost alerts.
Its cost figure is extrapolated from a partial run.

### Reception

**VERIFIED** via HN Algolia API. Three submissions, all trivially scored:

| ID | Submitter | Date | Points | Comments |
|---|---|---|---|---|
| 46032521 | `cpluss` (author) | 2025-11-24 | 4 | 1 (author's own) |
| 46058995 | `gmays` | 2025-11-26 | 4 | 1 |
| 46047471 | `vinhnx` | 2025-11-25 | 2 | 0 |

> **"Unrebutted" means nobody read it.** Four points is not survival of scrutiny.
> The PRD's rhetorical weight on this word is unearned.

### The arithmetic does not reconcile

Checked against the published table:

| PRD claim | What the table supports |
|---|---|
| "14–77x more expensive" | $152.6 / $1.98 = **77.1x** (Zep, extrapolated from a 43%-complete run). $24.88 / $1.98 = **12.6x** (Mem0). The "14" is not derivable from the table. |
| "31–33% less accurate" | 84.6 → 51.6 = **33.0 points absolute** / 39.0% relative. 84.6 → 49.3 = **35.3 points absolute** / 41.7% relative. The stated range matches neither reading cleanly. |

**UNVERIFIED / ambiguous.** The figures are approximately right in magnitude and
imprecise in derivation. Anyone quoting "31–33% less accurate" should say whether
they mean points or percent, because the post does not.

**UNRESOLVED:** the model. The HN comment says `gpt-5-nano`; the rendered blog
read as `gpt-4-nano`. Either way it is the cheapest available tier — the model
least able to use a long context well, and least able to use retrieved claims
well. Not a neutral choice.

### Six weaknesses, strongest first

**1. The long-context arm was 4,232 tokens.** This is decisive. At 4k the entire
conversation fits trivially in the window, so there is nothing for a memory
system to do except lose information. The experiment measures compression damage
at a scale where no compression is needed. It says nothing about behaviour at
120k or 800k, which is the only regime where the question is live.

**2. Wrong domain.** MemBench is conversational agent memory — factual and
reflective recall over dialogue. CRED is claims-and-evidence over a code
repository and its decision history. The transfer from one to the other is by
analogy, not by measurement. **The PRD's phrase "comparable memory systems"
overstates the link.**

**3. Zep's headline number is extrapolated from an aborted run.** 1,730/4,000
cases. The 77x is a projection, presented as a measurement.

**4. No variance reported.** No confidence intervals, no seeds, no repeated runs,
no statement of runs-per-case. A single-shot difference on 4,000 cases with an
unstated repeat count cannot be separated from run-to-run drift.

**5. Third-party configuration of vendor systems.** This is exactly the failure
mode of the public Mem0-vs-Zep dispute, where each vendor demonstrated the other
had misconfigured their system. A solo author configuring two competitors'
products inherits that risk in full.

**6. The author is not neutral.** fastpaca sells memory infrastructure. The
post's conclusion is not "memory is useless" — it is that *semantic* memory is
"brilliant for personalization" while *working* memory should be lossless and
exact. That is a product positioning argument, and it is fine, but it is not a
disinterested finding.

### What this means for CRED

The claim is **not disconfirmed and not confirmed.** It is untested in CRED's
regime. Nothing in the located source removes the need for this experiment; what
it removes is the excuse to treat the question as settled by someone else.

One point cuts *against* CRED and must not be lost: **prompt caching.** The
corpus in a long-context arm is identical across queries and cacheable. The
original benchmark's cost model appears not to account for this. A fair v0 must
enable caching on the long-context arm, which makes that arm substantially
cheaper than the citation implies. Disabling it would be a strawman.

---

## 2. Decisions

Stated as decisions, with what each rules out.

| # | Decision | Rules out |
|---|---|---|
| D1 | **All arms consume one identical corpus `C`.** Only the access path differs | Any "our memory had better source data" explanation of a win |
| D2 | **Tasks are mined from public repository history after a cutoff date `T`; ground truth is a maintainer's own post-`T` words** | The founder authoring the answers |
| D3 | **Any task the no-memory floor arm solves is deleted before the eval set is frozen** | Tasks answerable from code or parametric knowledge; contamination inflating every arm equally |
| D4 | **Primary metric, sample size, and kill threshold are frozen in a git tag before the first eval run** | Post-hoc reinterpretation |
| D5 | **The decision requires the same direction on two models from different families** | A finding that is really a fact about one model |
| D6 | **Prompt caching is ON for the long-context arm; memory-construction cost is amortized and reported at two policies** | Winning on cost by charging the opponent unfairly |
| D7 | **Retrieval arm B is tuned on a held-out dev split and its recall@k is published** | The weak-retriever strawman |
| D8 | **The MVE (§9) may kill the project; it may not green-light it** | Using a cheap underpowered run as licence to build v1 |

---

## 3. The arms

Four base arms plus two ablation arms. Every arm receives the same question text,
the same tools, the same output format instruction, and the same fixed prompt
template — differing only in the **memory block** injected between system prompt
and question.

| Arm | Memory block contents | Purpose |
|---|---|---|
| **A0 — floor** | *(empty)* Agent gets the question and read access to the repo working tree at `T`. No history, no discussion, no docs beyond what is committed | Contamination floor and task filter. Anything A0 solves is not a memory task |
| **A — long context** | The whole of corpus `C`, concatenated in a fixed deterministic order, up to the model's limit. Prompt caching enabled | The thing the citation says wins |
| **B — naive RAG** | Top-`k` chunks from `C` by hybrid retrieval (BM25 + dense + reciprocal rank fusion + reranker), to a fixed token ceiling | The honest mid-point. Most competitors are actually this |
| **C — structured memory** | Claims + evidence pointers extracted from `C`, assembled to the *same* token ceiling as B | CRED's shape |
| **C-doc** | Arm C restricted to claims whose evidence is code or committed documentation | Ablation denominator |
| **C-exp** | Arm C restricted to claims whose evidence is discussion, review, revert, or incident — things not present in the tree | Ablation numerator |

### Hard constraints on arm parity

- **B and C share one token ceiling.** Default 4,000 tokens of memory block.
  A win for C that comes from a larger block is not a win.
- **C-doc, C-exp, and C-both share the same ceiling.** Otherwise C-both wins by
  volume, and that is the exact mistake the Memco paper made (evidence doc §2).
- **Arm B is not a stub.** It gets a real embedding model, chunk size and `k`
  tuned on the 30-task dev split, and a reranker. Its config is published.
- **Arm A gets the whole corpus, not a truncated sample.** If `C` at size L
  exceeds the model limit, that is reported as arm A's failure mode, not papered
  over by silently dropping content.

### Corpus sizes

The citation's fatal flaw was testing at one small size. This design sweeps.

| Label | Corpus tokens | Role |
|---|---|---|
| S | ~15k | Approximates the citation's regime |
| **M** | **~120k** | **Headline comparison. Pre-registered as primary** |
| L | ~800k or model max | Where long context is expected to degrade |

Only arms A and C run the S and L sweep (one model, one run each), to control
cost. The full grid runs at M.

---

## 4. The task set

This is where experiments of this kind cheat, so the construction is specified as
a procedure rather than an intention.

### Source

**Primary: public OSS repositories with rich decision history.** Selection
criteria: ≥3 years of history, ≥500 merged PRs with substantive review
discussion, an `adr/` or `decisions/` directory or equivalent, and at least one
documented reversal or rejected approach. Two to four repositories.

**Secondary: Seta repositories**, run as a confirmatory replication only, and
reported separately. Seta is not the primary because Seta results are not
externally replicable, and an unreplicable result is not publishable — which is
the whole point of doing this first (PRD §11).

### Construction procedure

1. **Pick a cutoff `T`** per repository. Corpus `C` = everything authored before
   `T`. Nothing after `T` enters any arm.
2. **Mine anchors** from the window after `T`: comments, commit messages, and
   review threads matching backward-reference patterns — *"as we decided"*,
   *"we tried that"*, *"we moved away from"*, *"see #"*, *"the convention here
   is"*, *"this was reverted because"*.
3. **Draft** question + gold answer from each anchor with an LLM, given the
   anchor only. The founder does not write questions.
4. **Filter by A0.** Run arm A0 on every candidate, 3 runs. **Any task A0 gets
   right is deleted.** This is the single highest-value line in the design: it
   removes tasks answerable from the code alone, and it removes tasks the model
   already memorized from training data.
5. **Filter by verbatim presence.** Delete any task whose gold answer appears
   verbatim in the working tree at `T`.
6. **Founder role is restricted to rejection.** The founder may delete a
   malformed task. Every deletion is written to `tasks/rejected.jsonl` with a
   reason, and that file is published. The founder may not edit, reword, or add.
7. **Blind external audit.** One person who is not the founder reviews a random
   20% sample, blind to results, answering one question per task: *does this task
   appear constructed to favour structured memory?* Flag rate is published. A
   flag rate above 15% voids the set.
8. **Freeze.** `sha256` of `eval.jsonl` is committed and tagged before any eval
   run.

### Composition

| Family | Share | Description | Grading |
|---|---|---|---|
| Decision rationale | 30% | Why an approach was chosen or rejected months ago | Checkpoints |
| Unwritten convention | 25% | Practice not expressed in code or lint config | Checkpoints |
| Cross-cutting context | 20% | Spans modules or repositories | Checkpoints |
| Failure recall | 10% | This was tried, it broke, here is why | Checkpoints |
| **Abstention decoys** | **15%** | The answer is genuinely not recorded anywhere. **Correct behaviour is to say so** | Abstain / confabulate |

The decoys are not optional. Without them a memory arm can win by confidently
confabulating, and the judge will reward it. Confabulation rate is reported as a
first-class result alongside accuracy.

### Ground truth

Each gold answer decomposes into **1–4 atomic checkpoints**. A checkpoint is a
single verifiable fact, graded independently, with a match rule that is a string
/ alias set wherever possible and a judge call only where it cannot be. A task is
correct **iff all its checkpoints hit** (strict). Every task carries a permanent
citation: anchor URL, anchor commit SHA, anchor date, and the evidence spans in
`C` that should support it.

### Named risk: the founder's thesis contaminating the set

Stated plainly because it is the most likely way this experiment produces a false
proceed. Mitigations, in order of strength:

1. Ground truth comes from maintainers' own words, not the founder's (step 2–3).
2. The founder can only subtract, and every subtraction is logged and published
   (step 6).
3. The A0 filter removes the easy tasks that would flatter every arm (step 4).
4. Blind external audit with a published flag rate and a voiding threshold
   (step 7).
5. The set is frozen by hash before any arm result exists (step 8).

Residual risk, accepted and recorded: the founder chooses the repositories and
the cutoff dates. That choice is not blinded. It is disclosed in the writeup.

---

## 5. Metrics

### Primary — pre-registered

> **Task-level strict accuracy on the eval set, at corpus size M, arm C-both
> versus arm A, paired by task, averaged over 3 runs.**

Everything else is secondary and labelled as such.

### Secondary

| Metric | Definition |
|---|---|
| Confabulation rate | Share of decoy tasks answered with a concrete claim instead of an abstention |
| Cost — marginal | Input + output tokens per query at published list prices, cache-aware |
| Cost — amortized | Marginal + (memory construction cost ÷ 200 queries) |
| Cost per correct answer | Amortized cost ÷ tasks correct |
| Latency | Query wall-clock p50 / p95. Memory construction reported separately, never folded in |
| Retrieval recall@k | Arm B only, against gold evidence spans. **If < 0.60, arm B is broken and its comparison is void, not favourable** |

**Cost amortization is pre-registered at both policies** because the choice
decides the answer. The located benchmark effectively amortized construction over
a single conversation, the most hostile possible assumption. Reporting only the
generous policy would be the mirror-image cheat.

### The judge, and how it is validated

Checkpoint grading uses string/alias matching first. Only checkpoints marked
`match.type == "judge"` reach a model.

- **Two judges from different model families.** Agreement between them is
  reported.
- **Arm identity is stripped** from every graded answer. The judge never sees
  which arm produced it, never sees token counts, and sees answers in randomized
  order.
- **Founder hand-labels a stratified 60-run sample**, blind to arm, before
  looking at any aggregate result.
- **Pre-registered gate: Cohen's κ between judge and human must be ≥ 0.70.**
  Below that, the judge is discarded and the primary metric is computed on a
  human-graded subset of 60 tasks, with the reduced power stated in the writeup.

---

## 6. Pre-registration

Frozen before the first eval run. Committed, tagged `v0-prereg`, and the tag SHA
published in the writeup. Only dev-split runs are permitted before the tag.

### Hypothesis

> **H1.** At corpus size M, an agent given structured claims-and-evidence memory
> (arm C-both) answers more organizational-memory tasks correctly than an agent
> given the entire corpus in its context window (arm A).

Null: no difference or arm A superior.

### Design point

| Parameter | Value |
|---|---|
| Eval tasks | **150** (plus 30 dev, never used for evaluation) |
| Runs per task per arm | **3** |
| Models | **2**, different families: one frontier, one mid-tier |
| Corpus size, primary | **M (~120k)** |
| Arms in full grid | A0, A, B, C-both, C-doc, C-exp |
| Temperature | 0. Seeds fixed where the API exposes them |
| Primary test | Paired bootstrap over tasks on per-task mean score, 10,000 resamples, 95% CI |
| Robustness check | McNemar exact test on the 3-run majority-vote binarization |

### The three outcomes

| Outcome | Condition | Action |
|---|---|---|
| **KILL** | Point estimate of (C-both − A) ≤ 0 | **Publish the finding and stop.** Per PRD §13. No reinterpretation, no rerun |
| **AMBER** | 0 < (C-both − A) < 10 points, **or** the bootstrap 95% CI includes 0 | "No demonstrated advantage at this scale." Does **not** authorize v1. One pre-specified extension only (below) |
| **PROCEED** | (C-both − A) **≥ 10 points** AND bootstrap 95% CI excludes 0 AND C-both cost per correct answer **≤ 3x** arm A's | Proceed to v1 as scoped in the PRD |

The 3x cost gate is a judgment call, stated as one: the citation's attack was
14–77x. A memory system that buys accuracy at 20x has lost the argument it was
built to win, whatever its accuracy.

**The one permitted extension in the AMBER case:** rerun the primary comparison
at corpus size L, with the same thresholds. Nothing else. This is specified now
so that "let's try one more thing" cannot become an unbounded search after seeing
data.

### Model disagreement

If the two models give opposite directions on the primary comparison, the result
is **"model-dependent"**, it is published as such, and it **does not authorize
v1.** This is pre-registered so that a single favourable model cannot be
selected after the fact.

---

## 7. Statistical honesty

### Power

Paired binary outcome, α = 0.05 two-sided, 80% power. `ψ` is the discordant-pair
proportion; values assumed as shown.

| Detectable effect `δ` | Assumed `ψ` | Tasks required |
|---|---|---|
| 20 points | 0.40 | 76 |
| **15 points** | 0.35 | **120** |
| 13–14 points | 0.33 | ~150 |
| 10 points | 0.30 | 233 |
| 7 points | 0.25 | 398 |

**n = 150 is the design point.** It detects roughly a 13–14 point effect.

Stated plainly: **a 10-point effect is not reliably detectable at n = 150.** That
is deliberate. An effect smaller than ~13 points is not an effect worth building
a product on, and the PROCEED threshold is set at 10 points precisely so that a
marginal result lands in AMBER rather than green-lighting v1. Averaging 3 runs
per task reduces measurement noise, so these figures are upper bounds on n.

### Run-to-run variance

Temperature 0 does not give determinism, and a stable model alias can drift under
you with no code change (see `testing-strategy.md` on why LLM-judge work stays
out of CI).

- 3 runs per task per arm, fixed seeds where exposed, provider and model version
  string logged per run.
- Report **per-run accuracy mean with bootstrap CI**, never a bare mean.
- Report the **flip rate**: share of tasks where the 3 runs disagree.
- **Pre-registered:** if flip rate exceeds 25% in any arm, that arm's measurement
  is too noisy to decide on. Raise runs-per-task to 5 and re-run before reading
  the result. This is specified now so it cannot be invoked selectively later.

### Multiple comparisons

**One primary comparison: C-both vs A.** It alone drives the decision.

The secondary family — {C vs B, B vs A, C vs A0, C-both vs C-doc, C-exp vs A0} —
is reported with **Holm-Bonferroni adjusted p-values** and labelled exploratory.
A significant secondary result may not override a KILL on the primary. The
sweep across corpus sizes and the second model are reported as curves and
directional checks, not as additional significance tests.

---

## 8. The ablation

PRD §13's v1 gate: experiential memory must contribute **independently** of
documentation retrieval. This is the measurement the Memco paper did not make
(evidence doc §2), and it is CRED's stated reason to exist.

### Construction

Claims are partitioned by evidence source at extraction time, not filtered after.

- **C-doc** — evidence is a code symbol or a file committed to the tree at `T`.
- **C-exp** — evidence is PR review discussion, an issue thread, a revert commit
  message, or an incident note. Explicitly: things a fresh agent reading the
  repository cannot see.
- **C-both** — the union.

### Controls that make the result mean anything

1. **Identical token ceiling across all three.** If C-both gets a bigger block it
   wins on volume and the ablation is worthless.
2. **Identical retrieval configuration.** Same reranker, same `k`, same fusion.
3. **Leakage check.** The extractor never sees the task set — the task set hash
   is committed before extraction runs. Afterwards, an automated check measures
   verbatim gold-answer overlap with the extracted claim set, and the
   `gold_leak_rate` is published per build. This is the direct guard against the
   Memco failure of building the memory from the answers.

### Pre-registered v1 gate

> Experiential memory contributes independently **iff** (C-both − C-doc) ≥ **7
> points** with a bootstrap 95% CI excluding zero, **and** C-exp alone beats A0
> with a CI excluding zero.

If C-both ≈ C-doc, that is a finding about the category. Publish it and stop, per
PRD §11.

---

## 9. Cost, effort, and the minimum viable version

### Full design

Run count:

| Block | Count |
|---|---|
| Core grid: 150 tasks × 3 runs × 6 arms × 2 models, at size M | 5,400 |
| Corpus sweep: 150 × 1 run × 2 arms × 1 model × 2 sizes | 600 |
| **Total** | **~6,000** |

Arm A dominates token spend. 900 arm-A runs at ~120k input each is ~108M input
tokens; the size-L sweep adds ~120M. All other arms combined are ~43M. Output is
small. Judge grading adds ~24M.

| Line | Estimate |
|---|---|
| API spend, no caching | $600 – $1,500 |
| API spend, caching on arm A | $300 – $700 |
| Contingency (pilots, reruns, a botched grid) | +30% |
| **All-in** | **$800 – $2,000** |

Engineering effort:

| Work | Days |
|---|---|
| Harness (runner, logging, cost accounting, resume) | 4 |
| Task mining, filtering, curation, audit — **the bulk** | 6 |
| Claim extraction pipeline + source partitioning | 3 |
| Arm B: retrieval, dev-split tuning, recall@k instrumentation | 2 |
| Judge + validation + human labelling | 2 |
| Analysis, bootstrap, plots | 2 |
| Writeup | 2 |
| **Total** | **~21 engineer-days** |

Wall-clock for the runs themselves: **2–4 days** with parallelism, dominated by
rate limits rather than compute.

Realistic calendar for a solo founder at partial allocation: **4–6 weeks.**

### Minimum viable version — recommended if the full design is too expensive

| Parameter | MVE |
|---|---|
| Arms | A0, A, C-both **only** |
| Tasks | 60 |
| Runs per task | 3 |
| Models | 1 (mid-tier frontier) |
| Corpus size | M only |
| Runs | 540 |
| API spend | ~$120 |
| Engineering | ~9 days |
| Detectable effect | ~22 points at 80% power |

**The MVE's asymmetry is the point, and it is pre-registered:**

> **The MVE may kill the project. It may not green-light it.**

Justification: the thesis under attack claims a *large* effect — the citation
itself reports 33 points. An effect of that size is comfortably detectable at
n = 60. So if arm C ties or loses in the MVE, that is a genuine, publishable,
sufficient kill signal. But a win at n = 60 has a confidence interval too wide to
license six months of engineering, and the MVE has no ablation, so it cannot
speak to the v1 gate at all. A winning MVE mandates the full design; it does not
replace it.

---

## 10. Threats to validity

| # | Threat | How the design handles it | Residual |
|---|---|---|---|
| T1 | **Long context wins only because the corpus is small** — the citation's fatal flaw | Corpus sweep S/M/L; headline pre-registered at M (~120k); accuracy-vs-size curve published per arm | If C wins only at L, that is a narrower finding and must be reported as one, not as a general win |
| T2 | **Weak retriever strawmans arm B** | Hybrid BM25 + dense + RRF + reranker; tuned on a 30-task dev split; config published; recall@k published; **recall@k < 0.60 voids the comparison rather than favouring C** | Arm B is still one implementation. Published config lets others improve it |
| T3 | **The base model determines the answer** | Two families, one frontier and one mid-tier. Same direction required on both, pre-registered. Disagreement is published as "model-dependent" and does not authorize v1 | Two families is not all families. Frontier model behaviour may change under a stable alias — provider version string logged per run |
| T4 | **Memory built from the answers** (the Memco failure) | Task set hash committed before extraction; extractor never sees tasks; `gold_leak_rate` measured and published per build | Semantic leakage below verbatim threshold is not fully detectable |
| T5 | **Judge aligned with the treatment** (the Memco failure) | Atomic checkpoints not holistic quality; string matching first; arm-blind; two judge families; κ ≥ 0.70 gate against human labels | Ambiguous checkpoints still route through a judge |
| T6 | **Training-data contamination on public repos** | A0 filter deletes every task A0 solves; A0's residual score is published as the contamination floor; prefer post-cutoff timeframes | Partial memorization that helps without solving outright |
| T7 | **Founder-shaped task set** | §4: maintainer-sourced ground truth, founder can only subtract, published rejection log, blind external audit with a 15% voiding threshold, hash freeze | Repository and cutoff selection is not blinded. Disclosed |
| T8 | **Cost accounting rigged** | Caching ON for arm A; both amortization policies pre-registered and both reported; construction latency reported separately | List prices move. Token counts are logged raw so the analysis can be re-priced |
| T9 | **Prompt sensitivity** | One fixed template per arm, written before any result, differing only in the memory block. All templates published with hashes | Prompts could still favour one arm. Publication is the mitigation |
| T10 | **Memory wins by confabulating confidently** | 15% abstention decoys; confabulation rate reported as a first-class result | — |
| T11 | **Result does not transfer to a real organization** | Seta replication run as a separate, clearly-labelled confirmatory study | Seta results are not externally replicable and are never the headline |

---

## 11. Harness shapes

Concrete enough to start building without asking a question.

### Layout

```
v0/
  preregistration.md            # frozen; git tag v0-prereg
  tasks/
    eval.jsonl                  # 150 tasks, hash-frozen
    dev.jsonl                   # 30 tasks, tuning only
    rejected.jsonl              # every founder deletion + reason, published
    TASKSET.sha256
  corpus/
    {S,M,L}/…                   # materialized corpora
    MANIFEST.json               # per-size: sha, token count, source inventory
  memory/
    {C_both,C_doc,C_exp}/claims.jsonl
    {C_both,C_doc,C_exp}/build.json
  index/                        # arm B: chunks, vectors, BM25, tuning curve
  prompts/{a0,a,b,c}.md         # one template per arm, hashed
  runs/<model>/<arm>/<size>/runs.jsonl
  grades/grades.jsonl
  analysis/analyze.py
  results/
```

### Task record — `tasks/eval.jsonl`

```json
{
  "task_id": "kn-0142",
  "family": "decision_rationale",
  "repo": "org/repo",
  "cutoff_t": "2025-03-01T00:00:00Z",
  "question": "Why does the ingest path buffer to disk instead of streaming?",
  "gold": {
    "answer": "Streaming was tried in #612 and reverted: back-pressure from the downstream consumer caused unbounded memory growth under replay.",
    "checkpoints": [
      { "id": "c1", "text": "streaming was previously attempted and reverted",
        "match": { "type": "judge" } },
      { "id": "c2", "text": "the cause was back-pressure / unbounded memory under replay",
        "match": { "type": "any_of", "values": ["back-pressure", "backpressure", "unbounded memory", "OOM"] } }
    ],
    "abstain_expected": false
  },
  "provenance": {
    "anchor_url": "https://github.com/org/repo/pull/812#issuecomment-2299",
    "anchor_sha": "a1b2c3d",
    "anchor_date": "2025-06-14T09:12:00Z",
    "evidence_spans": [
      { "path": "docs/adr/0007-ingest-buffering.md", "lines": [12, 31], "sha": "9f8e7d6" }
    ]
  },
  "filters": { "a0_solved": false, "verbatim_in_tree_at_t": false },
  "audit": { "sampled": true, "flagged_biased": false },
  "author": "miner-v1 + human-accept",
  "reject_reason": null,
  "set": "eval"
}
```

### Run record — `runs/…/runs.jsonl`, one line per (arm × task × run)

```json
{
  "run_id": "01JQ…",
  "task_id": "kn-0142",
  "arm": "C_both",
  "model": "…",
  "provider_model_version": "…",
  "corpus_size": "M",
  "run_index": 1,
  "seed": 1337,
  "temperature": 0.0,
  "taskset_sha": "…", "corpus_sha": "…", "prompt_template_sha": "…",
  "memory_build_id": "…",
  "context_block_sha": "…",
  "context_tokens": 3987,
  "retrieved_ids": ["clm-0031", "clm-0207"],
  "input_tokens": 12844,
  "cached_input_tokens": 0,
  "output_tokens": 311,
  "usd_marginal": 0.0041,
  "usd_amortized_n200": 0.0068,
  "latency_ms_retrieval": 180,
  "latency_ms_generation": 2240,
  "latency_ms_total": 2420,
  "answer_text": "…",
  "finish_reason": "stop",
  "error": null,
  "harness_version": "0.1.0",
  "started_at": "2026-08-02T10:14:03Z"
}
```

`finish_reason` is logged and checked. A run that ended on `length` is a
truncated answer, not a wrong one, and is excluded with a reported count — the
same discipline `tech-decisions.md` requires of the extractor.

### Grade record — `grades/grades.jsonl`

```json
{
  "run_id": "01JQ…",
  "grader": "judge-A",
  "judge_prompt_sha": "…",
  "checkpoint_results": [
    { "id": "c1", "hit": true,  "method": "judge" },
    { "id": "c2", "hit": false, "method": "string" }
  ],
  "task_correct": false,
  "abstained": false,
  "confabulated": false,
  "graded_at": "2026-08-03T08:01:00Z"
}
```

### Memory build record — `memory/<arm>/build.json`

```json
{
  "build_id": "bld-03",
  "arm": "C_both",
  "corpus_sha": "…",
  "source_filter": ["code", "committed_docs", "review_discussion", "revert", "incident"],
  "n_claims": 842,
  "n_evidence": 1130,
  "build_input_tokens": 4211903,
  "build_output_tokens": 188442,
  "build_usd": 14.82,
  "build_wall_clock_s": 5120,
  "extractor_model": "…",
  "extractor_prompt_sha": "…",
  "taskset_sha_at_build": "…",
  "gold_leak_rate": 0.004
}
```

`taskset_sha_at_build` is the mechanical proof that extraction happened after the
task set was frozen. Without that field the leakage guard is a promise; with it,
it is checkable by a reader.

---

## 12. What gets published, either way

Both outcomes are deliverables (PRD §11). The writeup includes, regardless of
direction:

- The pre-registration tag SHA and the frozen task-set hash.
- The full `rejected.jsonl` and the blind audit flag rate.
- Arm B's config and its recall@k.
- Judge-vs-human κ.
- Per-arm flip rate.
- The accuracy-vs-corpus-size curves.
- Both cost amortization policies.
- The replication of the fastpaca benchmark's regime as corpus size S, so the
  original result and this one can be read on the same axis.

That last item matters more than it looks. The located citation is the only
public measurement in this space, it is weak, and nobody has re-run it. Running
it as one point on a curve — rather than arguing with it — is the version that
survives scrutiny.
