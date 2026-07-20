# v0 experiment harness

Experiment code for `docs/research/spikes/v0-experiment-design.md`. It is not
product code, it ships nothing, and it is exempt from the pre-implementation
freeze in `CLAUDE.md` for exactly that reason.

It answers one question: **does retrieved memory beat plain long context?** It
produces a KILL decision or a PROCEED decision, and both are publishable.

Read `preregistration.md` before running anything. The numbers in it are the
numbers the code enforces.

---

## 1. What is here

```
v0/
  preregistration.md          frozen at git tag v0-prereg
  config/{mve,full}.json      the MVE and the full design; same code path
  prompts/                    one template per arm + drafter, extractor, judge
  harness/                    the stages
  analysis/analyze.py         primary metric, CIs, variance, cost, verdict
  mining/                     anchors.jsonl, MINING.json  (T7 disclosure)
  corpus/                     {S,M,L}/ + MANIFEST.json
  tasks/                      candidates, eval, dev, rejected, TASKSET.sha256
  memory/                     {C_both,C_doc,C_exp}/{claims.jsonl,build.json}
  index/                      chunks + retrieval config
  runs/<model>/<arm>/<size>/runs.jsonl
  grades/                     grades.jsonl, human labels, KAPPA.json
  results/                    analysis output
```

`config/mve.json` is the minimum viable version: arms A0, A, C-both; 60 tasks;
3 runs; one mid-tier frontier model; corpus size M. `config/full.json` is the
full design: six arms, 150 tasks, two model families, the S/M/L sweep. Moving
between them is a `--config` flag, not a rewrite — which matters, because §9
pre-registers that a winning MVE *mandates* the full run.

---

## 2. What the operator must supply

| Requirement | Why | Check |
|---|---|---|
| Python **3.11+** | stdlib only for everything except the provider | `python3 --version` |
| `gh`, authenticated | mining and corpus fetch go through the GitHub API | `gh auth status` |
| `pip install anthropic` | the only stage that needs a package | — |
| Anthropic credentials | `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, or an `ant auth login` profile | `python3 -m v0.cli status` |
| One human, for two jobs | hand-label 60 runs blind (§5 κ gate); reject malformed tasks | — |
| One **non-founder** human | the blind 20% audit (§4 step 7) | — |

Optional: `pip install tiktoken` makes token counts exact rather than a
characters÷4 estimate. Which estimator was used is recorded in
`corpus/MANIFEST.json`; billing always uses provider-reported counts, so this
affects corpus sizing only, never cost.

No credentials were present in the environment this harness was built in, so
**no live model call has been made**. Every model-calling stage also runs
against the `mock` provider, which is deterministic, free, and produces
obviously-fake text. Nothing produced under the mock provider is a result —
`analyze.py` prints a warning and `freeze` refuses to freeze a mock-drafted
task set.

---

## 3. How to run it, in order

```sh
python3 -m v0.cli status      # what exists, what is missing, template hashes
```

### Stage 1 — mine anchors (free, ~30 min, `gh`)

```sh
python3 -m v0.cli mine --prs-per-repo 250
```

Mines post-cutoff comments and PR bodies matching the backward-reference
patterns in §4 step 2. Writes `mining/anchors.jsonl` and `mining/MINING.json`.
`MINING.json` contains the T7 disclosure: which repositories, which cutoff, why
each was chosen, and each one's known bias risk.

### Stage 2 — build the corpus (free, ~45 min, `gh`)

```sh
python3 -m v0.cli corpus-fetch --pr-cap 200
python3 -m v0.cli corpus-build
python3 -m v0.cli index
```

`corpus-fetch` pulls documents authored **strictly before** the cutoff T.
`corpus-build` assembles S/M/L in a fixed deterministic order and writes
`MANIFEST.json` with per-size sha256, token counts, and a full source
inventory. If a size cannot be filled it says so in the manifest rather than
letting a short corpus read as a full one.

### Stage 3 — draft candidate tasks (**model**, ~$5)

```sh
python3 -m v0.cli draft --config mve.json
python3 -m v0.cli verify-decoys
python3 -m v0.cli verbatim-filter
```

The drafter sees one anchor and nothing else. It never sees the corpus, other
tasks, or the hypothesis. `verify-decoys` rejects any decoy whose distinctive
terms turn up in the record. `verbatim-filter` deletes tasks whose gold answer
is present verbatim in the tree at T (§4 step 5).

Draft roughly **4× the target task count** — the A0 filter is designed to delete
a large fraction.

### Stage 4 — the A0 floor filter (**model**, ~$2)

```sh
python3 -m v0.cli a0-filter
```

The highest-value step in the design. Runs arm A0 — question only, no memory —
three times on every candidate and deletes every factual task it gets right.
What survives is not answerable from parametric knowledge. Writes
`tasks/A0_FILTER.json` with the pre-filter solve rate. Re-runnable and
idempotent.

Two decisions this stage makes, both argued in `harness/a0_filter.py`: "gets
right" means *any* of three runs, and abstention decoys are exempt from
deletion.

### Stage 5 — founder rejection, external audit, freeze

The founder may only **subtract**, and every subtraction lands in
`tasks/rejected.jsonl` with a reason and a stage. There is no code path that
edits or adds a task; `harness/tasks.py:reject` is the only mutation.

```sh
python3 -m v0.cli audit-sample   # blind 20% -> tasks/audit_sample.jsonl
#   ... a NON-FOUNDER sets flagged_biased true/false in each row and saves it
#   ... as tasks/audit_labels.jsonl
python3 -m v0.cli audit-score
```

The auditor answers one question per task: *does this task appear constructed
to favour structured memory?* They see the question, family, repository, and
anchor URL — never the gold answer, the checkpoints, or any result. **Above a
15% flag rate `audit-score` raises and the set is void**, and it says why
deleting the flagged tasks is not a repair.

```sh
python3 -m v0.cli freeze --config mve.json
git add v0/tasks v0/config v0/prompts v0/preregistration.md
git commit -m "v0: freeze task set and pre-registration"
git tag v0-prereg
```

The freeze refuses if any factual candidate has not been through the A0 filter,
if any task was drafted by the mock provider, if the decoy share is off by more
than 5 points, or if the set is already frozen.

### Stage 6 — build the memory (**model**, ~$2)

```sh
python3 -m v0.cli extract
python3 -m v0.cli partition
python3 -m v0.cli gold-leak
```

`extract` **refuses to run** unless `tasks/TASKSET.sha256` exists, and writes
`taskset_sha_at_build` into every `build.json`. `harness/memory.py` never
imports `tasks.py` at module scope — the extractor is structurally unable to
see the task set. `gold-leak` runs afterwards, as a separate step, and is the
only place task text and claim text meet.

### Stage 7 — the grid (**model**, ~$15)

```sh
python3 -m v0.cli run --config mve.json
```

540 runs for the MVE. Resumable: interrupted at run 400, it restarts at 400.
Every record carries raw token counts (T8), the provider's resolved model
version (T3), `finish_reason`, and the sha of the exact memory block used.

Prompt caching is ON for arm A (D6). Disabling it would be a strawman, and it
is the one point in the located citation that cuts against CRED.

### Stage 8 — grade and validate the judge (**model**, ~$1)

```sh
python3 -m v0.cli grade
python3 -m v0.cli kappa-sample --n 60      # writes a blind sample
#   ... hand-label human_task_correct in grades/human_sample.jsonl,
#   ... save as grades/human_labels.jsonl. Do NOT open human_sample_armkey.json.
python3 -m v0.cli kappa
```

String/alias matching is attempted first; only `match.type == "judge"`
checkpoints reach a model. The judge never sees the arm, the model, the token
counts, or the run id, and grades in shuffled order.

**The κ ≥ 0.70 gate is real.** `analyze.py` raises and exits non-zero without a
passing `grades/KAPPA.json`.

### Stage 9 — analyse

```sh
python3 v0/analysis/analyze.py --config mve.json
```

Prints the per-arm table, the primary comparison with its bootstrap CI, McNemar
on the majority-vote binarization, cost per correct answer, and the verdict.
Writes `results/analysis-*.json` with the price table embedded so a reader can
re-price it.

---

## 4. What each stage costs

Estimates for the MVE at `claude-sonnet-5` list prices, using the token counts
the design implies. They are estimates and the harness logs raw tokens so the
real figures replace them.

| Stage | Model calls | Est. USD |
|---|---|---|
| mine, corpus-fetch, corpus-build, index | none | 0 |
| draft (≈300 anchors) | ~300 | ~5 |
| a0-filter (≈150 candidates × 3) | ~450 + judge | ~2 |
| extract (≈300 chunks) | ~300 | ~2 |
| run (540 runs; arm A cached) | 540 | ~15 |
| grade | ~500 | ~1 |
| **Total** | | **~25** |

Design §9 budgets ~$120 for the MVE. The gap is mostly contingency — pilots,
a botched grid, re-runs after a flip-rate failure. Budget the $120.

The dominant single cost is arm A's input tokens. With caching on, ~180 arm-A
runs at ~120k tokens cost roughly $10 rather than roughly $65.

---

## 5. What the first real run against live repositories showed

Mining and corpus construction have been run for real against the three
repositories in §10 of `preregistration.md`. Drafting onward has not — there
were no model credentials. Four things came out of it that change how the
operator should plan the run.

**The anchor set on disk is a demonstration-scale sample, not the run.** 45 PRs
per repository, against the ~570 per repository the real run needs (below).
Check `run_complete` in `mining/MINING.json` before trusting any count in it.

**A pattern was matching forward-looking text, and it was over half the yield.**
The bare `we always` / `we never` alternatives in the `convention` pattern
matched review *requests* ("assert that we never see an event older than the
checkpoint") — proposals about future code, not references to established
practice. Tightened to the present perfect.

The correction removed **13 of the 24 anchors mined, 54%** — including 11 of
CockroachDB's 15. Before the fix CockroachDB looked like a 4x richer source than
the other two repositories; afterwards all three sit within 25% of each other.
Had that gone unnoticed, the mining effort would have been weighted toward a
repository on the strength of a broken regex.

The change, its reason, and its effect are recorded in `ANCHOR_PATTERNS` /
`PATTERN_REVISIONS` in `harness/mine.py` and published in `mining/MINING.json`.
It was made before any task existed and before any result; the freeze that
matters is still ahead. `mine.refilter()` re-applied the corrected patterns to
the anchors already on disk, so code and artefacts agree without re-spending
the API budget; its report is `mining/REFILTER.json`.

**Anchor yield is low, and fairly uniform once the false positives are gone.**
135 merged PRs scanned produced **11 anchors** — 0.081 per PR.

| Repository | PRs scanned | Anchors | Yield |
|---|---:|---:|---:|
| `cockroachdb/cockroach` | 45 | 4 | 0.089 |
| `open-telemetry/opentelemetry-collector` | 45 | 4 | 0.089 |
| `backstage/backstage` | 45 | 3 | 0.067 |

The 72-task set needs roughly 300 anchors once drafter rejections and the A0
filter have taken their share, so plan on mining **~3,700 PRs**. At the observed
~2.5 PRs/minute the miner is by far the long pole: budget **20–25 hours** of
wall clock, or parallelise across repositories. This number belongs in the plan,
not in a surprise halfway through drafting. It is also the strongest argument
for adding repositories rather than mining the same three harder.

**`gh api` silently switched to POST.** Passing `-f ref=<sha>` without
`-X GET` turns the request into a POST, and the contents endpoint then fails in
a way indistinguishable from "this directory does not exist". The first corpus
fetch therefore reported **0 committed documents** for a repository that has
189. Fixed; the affected calls now pass `-X GET` explicitly.

**Corpus assembly had to become stratified.** Filling each size by walking one
global document order produced a corpus that was **100% committed
documentation at every size** — the discussion never got reached. That would
have left `C_exp` with nothing to extract and turned the §8 ablation, the
measurement this project exists to make, into a measurement of nothing. Each
size now allocates its budget across source types in proportion to the whole
record:

| Size | Tokens | Docs | Committed docs | Review discussion |
|---|---:|---:|---:|---:|
| S | 15,264 | 6 | 10,863 | 4,137 |
| M | 121,429 | 33 | 86,897 | 33,103 |
| L | 807,355 | 171 | 579,309 | 220,691 |

All three sizes reach their §3 budgets from 1.42M tokens of available record,
so the S/M/L sweep is viable on these three repositories.

One caveat that remains: `C_exp` is currently **all `review_discussion`**. No
`revert` or `incident` documents surfaced, because the revert classifier keys
off PR titles among the most-commented PRs and none matched. Failure-recall
tasks and the revert half of the experiential evidence class will be thin until
the fetcher mines revert commits directly.

---

## 6. Not built

Listed rather than left as a silent gap.

**Arm B (naive RAG) is half-built.** Chunking, Okapi BM25, reciprocal rank
fusion, and recall@k are implemented and used by the C arms and the decoy
checker. The **dense retriever and the reranker are not implemented.** §3
requires arm B to have a real embedding model, a `k` tuned on the dev split,
and a reranker; a BM25-only arm B is exactly the weak-retriever strawman T2
exists to prevent. `harness/retrieval.py:require_arm_b` therefore refuses to
run arm B rather than quietly running it lexically-only. Arm B is out of scope
for the MVE (§9 runs A0, A, C-both) and is a **blocker for the full design**.
Register implementations via `retrieval.register_dense()` and
`retrieval.register_reranker()`.

**The second model family is a placeholder.** `config/full.json` has a
`second-family` entry with `REPLACE-ME` fields and the mock provider. D5
requires the same direction on two models from different families before
anything green-lights, and `Config.validate` refuses a `may_greenlight` config
without two distinct families — but it cannot check that the second one is
real. Wire a second provider before the full run.

**The blind external audit needs a person.** `audit-sample` and `audit-score`
draw the sample and enforce the 15% voiding threshold, but §4 step 7 requires a
reviewer who is not the founder, and the harness cannot supply one.

**Corpus size L may not be reachable** from three repositories at the current
PR caps. `corpus-build` writes an `under_budget_warning` into the manifest when
a size cannot be filled. Raise `--pr-cap` or add repositories before treating L
as a measurement of the 800k regime.

**The Seta replication (§4 secondary, T11) is not implemented.** It is a
separate, clearly-labelled confirmatory study and is never the headline.

**No live model call has been made.** There were no credentials in the build
environment, and the instruction was not to spend money. Every model-calling
stage has been exercised end to end against the mock provider only. Treat the
first real run as a pilot: run `--limit 20` before committing to the grid.

---

## 7. The rules this code enforces mechanically

Not as documentation — as code that raises.

| Design rule | Enforcement |
|---|---|
| D1 — all arms consume one identical corpus | Every arm's block is derived from `corpus/<size>/`; `corpus_sha` is on every run record |
| D3 — anything A0 solves is deleted before the freeze | `freeze` refuses while any factual candidate is unfiltered |
| D4 — metric and thresholds frozen before the first eval run | `analyze.py` reads thresholds from the config; `freeze` is one-way without `--force` |
| D6 — caching ON for arm A | `runner.py` sets `cache_system=True` for arm A only |
| D8 / §9 — the MVE may not green-light | `may_greenlight: false` turns PROCEED into PROCEED-BLOCKED |
| §3 — B and all C arms share one token ceiling | `arms.py:assemble` raises if a block exceeds it |
| §5 — κ ≥ 0.70 | `kappa.enforce_gate` raises; analysis exits 2 |
| §7 — flip rate ≤ 25% | The verdict is annotated as unreadable above it |
| T4 — memory not built from the answers | `extract` refuses before the freeze; `taskset_sha_at_build` on every build |
| T5 — judge is arm-blind | `grade_run` receives answer text and checkpoint only |
| T8 — analysis must be re-priceable | Raw token counts on every run; price table embedded in results |
| T9 — one fixed template per arm, hashed | `prompts.py` is the only path to prompt text; `status` prints the hashes |
| T10 — 15% decoys, confabulation first-class | `freeze` refuses outside 15% ± 5pp; confabulation is in the per-arm table |
