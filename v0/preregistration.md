# v0 pre-registration — minimum viable version

- **Date:** 2026-07-20
- **Status:** Draft. Not yet frozen.
- **Freezes as:** git tag `v0-prereg`
- **Derived from:** `docs/research/spikes/v0-experiment-design.md` §6 and §9
- **Machine-readable twin:** `v0/config/mve.json`

This document and `v0/config/mve.json` state the same numbers. The analysis
reads the numbers from the JSON, so the two cannot drift: if a threshold here
disagrees with a threshold in `results/`, the JSON was edited and the tag will
show it.

**Nothing in this document may be changed after the tag is cut.** Only
dev-split runs are permitted before the tag.

---

## 1. What is being registered

The **minimum viable version** of the v0 experiment, per design §9. It is not
the full design. Its asymmetry is the point and is registered here as a binding
constraint:

> **The MVE may kill the project. It may not green-light it.**

A win at this sample size has a confidence interval too wide to license six
months of engineering, and this configuration runs no ablation, so it cannot
speak to the v1 gate at all. `may_greenlight` is `false` in the config, and
`analysis/analyze.py` converts a PROCEED verdict into `PROCEED-BLOCKED`
mechanically. The full design in `v0/config/full.json` is what a winning MVE
mandates.

---

## 2. Hypothesis

> **H1.** At corpus size M (~120k tokens), an agent given structured
> claims-and-evidence memory (arm C-both) answers more organizational-memory
> tasks correctly than an agent given the entire corpus in its context window
> (arm A).

**Null:** no difference, or arm A superior.

---

## 3. Primary metric

> **Task-level strict accuracy on the eval set, at corpus size M, arm C-both
> versus arm A, paired by task, averaged over 3 runs.**

Strict means a task is correct **iff every one of its 1–4 atomic checkpoints
hits**. Partial credit does not exist in this experiment.

Everything else is secondary and is labelled as such in the output.

---

## 4. Design point

| Parameter | Value |
|---|---|
| Eval tasks | **60** |
| Dev tasks (tuning only, never evaluated) | **12** |
| Runs per task per arm | **3** |
| Arms | **A0, A, C-both** |
| Models | **1** — one mid-tier frontier model |
| Model | `claude-sonnet-5` (Anthropic) |
| Corpus size | **M only (~120k tokens)** |
| Memory-block token ceiling, arm C | **4,000** |
| Total runs | **540** (60 × 3 × 3) |
| Temperature | **0**, where the provider accepts it — see §9 |
| Seeds | 1337, 1338, 1339 — logged, not sent; the API exposes no seed |
| Primary test | Paired bootstrap over tasks on per-task mean score, **10,000 resamples, 95% CI** |
| Robustness check | **McNemar exact test** on the 3-run majority-vote binarization |
| Amortization denominator | **200 queries** |

Dev split size is 12 rather than the full design's 30, scaled to the smaller
task set. It is drawn stratified by family before the eval set and is never
evaluated.

---

## 5. Task-set composition

| Family | Share | Count (of 72 drafted) |
|---|---|---|
| Decision rationale | 30% | 22 |
| Unwritten convention | 25% | 18 |
| Cross-cutting context | 20% | 14 |
| Failure recall | 10% | 7 |
| **Abstention decoys** | **15%** | **11** |

The decoys are not optional. Without them a memory arm can win by confidently
confabulating and the judge will reward it. **Confabulation rate is reported as
a first-class result alongside accuracy.**

---

## 6. The three outcomes

Computed at corpus size M, on the primary comparison only.

| Outcome | Condition | Action |
|---|---|---|
| **KILL** | Point estimate of (C-both − A) **≤ 0 points** | Publish the finding and stop. No reinterpretation, no rerun. |
| **AMBER** | **0 < (C-both − A) < 10 points**, **or** the bootstrap 95% CI includes 0 | "No demonstrated advantage at this scale." Does not authorize v1. One pre-specified extension only, below. |
| **PROCEED** | (C-both − A) **≥ 10 points** **AND** bootstrap 95% CI excludes 0 **AND** C-both cost per correct answer **≤ 3.0×** arm A's | Mandates the full design (`config/full.json`). **Does not authorize v1** — see §1. |

The 3× cost gate is a judgement call, stated as one. The citation this
experiment answers claimed 14–77×. A memory system that buys accuracy at 20×
has lost the argument it was built to win, whatever its accuracy.

**The one permitted extension in the AMBER case:** rerun the primary comparison
at corpus size L, with the same thresholds. Nothing else. This is specified now
so that "let's try one more thing" cannot become an unbounded search after
seeing data.

---

## 7. Power, stated honestly

At n = 60, paired binary outcome, α = 0.05 two-sided, 80% power, the detectable
effect is roughly **22 points**.

The PROCEED threshold is 10 points. **A 10-point effect is not reliably
detectable at n = 60.** That is not an oversight — it is why PROCEED at this
sample size mandates the full design rather than authorizing v1.

The thesis under attack claims a *large* effect; the located citation itself
reports 33 points. An effect of that size is comfortably detectable at n = 60.
So a tie or a loss here is a genuine, publishable, sufficient kill signal, and
a win here is not a licence.

---

## 8. Validity gates, all pre-registered

Each is a numeric threshold checked in code, not a intention.

| Gate | Threshold | Consequence of failure | Where enforced |
|---|---|---|---|
| Judge vs human agreement | **Cohen's κ ≥ 0.70** | Judge discarded; primary metric computed on a human-graded subset of 60 tasks, with reduced power stated | `harness/kappa.py:enforce_gate` — raises |
| Per-arm flip rate | **≤ 25%** | Arm too noisy to decide on. Raise runs-per-task to 5 and re-run before reading the result | `analysis/analyze.py` — blocks the verdict |
| Blind external audit flag rate | **≤ 15%** | Task set is void | Manual; recorded in `tasks/AUDIT.json` |
| Decoy share of eval set | **15% ± 5pp** | Freeze refuses | `harness/freeze.py` |
| Gold leak rate | Published per build; no auto-fail threshold | Reader judges | `harness/memory.py:measure_gold_leak` |
| Retrieval recall@k (arm B) | **≥ 0.60** | Arm B comparison is **void, not favourable** | Full design only; arm B is not in the MVE |

The κ gate is not skippable. `analyze.py` refuses to emit a judge-derived
primary metric without a passing `grades/KAPPA.json`.

---

## 9. Registered deviations from the design document

Recorded before the tag so they are not later mistaken for post-hoc
adjustments.

**Temperature.** §6 registers temperature 0. Current Anthropic models
(`claude-sonnet-5`, `claude-opus-4-7`, `claude-opus-4-8`) reject non-default
sampling parameters with a 400, so the parameter cannot be sent. The harness
omits it and logs `temperature: null` on every affected run record. The
registered intent — no deliberate sampling variance — holds; the knob no longer
exists. This is precisely why the flip-rate gate in §8 is there.

**Seeds.** The Messages API exposes no seed. The `seed` field on the run record
is the harness's run-index label. It is never sent.

**Extended thinking.** The design does not mention it. Adaptive thinking is on
by default on `claude-sonnet-5`, which would add uncontrolled token spend and a
second source of run-to-run variance to a comparison meant to be about the
memory block. It is set explicitly to `disabled` and recorded on every run
record. Override with `CRED_V0_THINKING=adaptive` — doing so after the tag is a
protocol change and must be reported as one.

**Dev split size.** 12, not 30 (§4 above).

**A0 filter, "gets right".** Any of the three runs correct, not majority. The
stricter rule is chosen because a task the model gets right one time in three is
partly memorized, and a partly-memorized task inflates every arm equally.

**A0 filter, decoys.** Abstention decoys are exempt from A0 deletion. A0 has no
memory, so abstaining is trivially correct for it and the literal rule would
delete every decoy. A0's abstention rate on decoys is published instead.

---

## 10. Disclosure — repository and cutoff selection is not blinded

T7 residual, disclosed as the design requires. The founder chose the
repositories and the cutoff date. The full reasoning, including each
repository's known bias risk, is in `v0/mining/MINING.json` under
`selection_reason_T7`, written before any task was drafted.

| Repository | Cutoff T | Decision record |
|---|---|---|
| `backstage/backstage` | 2026-01-01 | `docs/architecture-decisions` |
| `cockroachdb/cockroach` | 2026-01-01 | `docs/RFCS` |
| `open-telemetry/opentelemetry-collector` | 2026-01-01 | `docs/rfcs` |

The anchor window is 2026-01-01 to 2026-07-01, which places every anchor after
the assistant model family's stated January 2026 training cutoff. That does not
make the underlying *decisions* unmemorized — they predate T. The actual
control for training-data contamination is the A0 filter, not the date.

---

## 11. What gets published either way

Both outcomes are deliverables. Regardless of direction, the writeup includes:

- The `v0-prereg` tag SHA and the frozen `eval.jsonl` hash.
- The full `tasks/rejected.jsonl` and the blind audit flag rate.
- Judge-vs-human κ.
- Per-arm flip rate.
- Per-arm confabulation rate.
- A0's residual accuracy on the frozen eval set, as the contamination floor.
- Both cost amortization policies, and raw token counts so the analysis can be
  re-priced.
- Every prompt template, with hashes.

---

## 12. Freeze procedure

```sh
python3 -m v0.cli freeze --config mve.json
git add v0/tasks v0/config v0/prompts v0/preregistration.md
git commit -m "v0: freeze task set and pre-registration"
git tag v0-prereg
```

The tag must be cut **before** the first eval run and **before** claim
extraction. `harness/memory.py` refuses to extract without
`tasks/TASKSET.sha256`, and writes `taskset_sha_at_build` into every
`build.json`, so a reader can check the ordering rather than trust it.
