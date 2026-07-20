"""Judge validation — §5's κ ≥ 0.70 gate.

This is a real check that fails loudly, not a comment. `enforce_gate()` raises,
and `analysis/analyze.py` calls it before it will emit a primary metric computed
from judge labels.

The procedure §5 specifies:
  1. Draw a stratified 60-run sample, arm-blind.        -> `sample_for_human()`
  2. The founder hand-labels it, before seeing any aggregate result.
  3. Compute Cohen's κ between judge and human.          -> `cohens_kappa()`
  4. If κ < 0.70 the judge is discarded and the primary metric is computed on a
     human-graded subset of 60 tasks, with the reduced power stated.
"""

from __future__ import annotations

import json
import random
from pathlib import Path

from .util import V0_ROOT, HarnessError, read_jsonl, utcnow, write_json, write_jsonl

HUMAN_SAMPLE_PATH = V0_ROOT / "grades" / "human_sample.jsonl"
HUMAN_LABELS_PATH = V0_ROOT / "grades" / "human_labels.jsonl"
KAPPA_PATH = V0_ROOT / "grades" / "KAPPA.json"


def cohens_kappa(a: list[bool], b: list[bool]) -> float:
    """Cohen's κ for two binary raters over the same items."""
    if len(a) != len(b):
        raise HarnessError(f"rater vectors differ in length: {len(a)} vs {len(b)}")
    n = len(a)
    if n == 0:
        raise HarnessError("cannot compute κ over an empty sample")
    agree = sum(1 for x, y in zip(a, b) if x == y)
    po = agree / n
    pa1, pb1 = sum(a) / n, sum(b) / n
    pe = pa1 * pb1 + (1 - pa1) * (1 - pb1)
    if pe == 1.0:
        # Both raters are constant and identical. κ is undefined; agreement is
        # perfect but uninformative. Reporting 1.0 here would launder a
        # degenerate sample into a passing gate.
        raise HarnessError(
            "κ is undefined: both raters gave the same label to every item in the "
            "sample. Draw a sample with variation before claiming the judge is validated."
        )
    return (po - pe) / (1 - pe)


def sample_for_human(
    grades: list[dict],
    runs: list[dict],
    *,
    n: int = 60,
    seed: int = 90210,
    out_path: Path = HUMAN_SAMPLE_PATH,
) -> dict:
    """Draw a stratified, arm-blind sample for hand-labelling.

    Stratified by arm so the sample covers the range of answer quality, but the
    arm is stripped from what the labeller sees. The mapping back to arm lives
    in a separate key file the labeller does not open.
    """
    run_by_id = {r["run_id"]: r for r in runs}
    by_arm: dict[str, list[dict]] = {}
    for g in grades:
        r = run_by_id.get(g["run_id"])
        if not r:
            continue
        by_arm.setdefault(r["arm"], []).append(g)
    if not by_arm:
        raise HarnessError("no gradeable runs to sample")

    rng = random.Random(seed)
    per_arm = max(1, n // len(by_arm))
    picked: list[dict] = []
    for arm in sorted(by_arm):
        pool = sorted(by_arm[arm], key=lambda g: g["run_id"])
        rng.shuffle(pool)
        picked.extend(pool[:per_arm])
    rng.shuffle(picked)
    picked = picked[:n]

    blind = [
        {
            "run_id": g["run_id"],
            "task_id": g["task_id"],
            "answer_text": run_by_id[g["run_id"]].get("answer_text", ""),
            # The labeller fills this in. Left null deliberately.
            "human_task_correct": None,
        }
        for g in picked
    ]
    write_jsonl(out_path, blind)
    key = {g["run_id"]: run_by_id[g["run_id"]]["arm"] for g in picked}
    write_json(out_path.with_name("human_sample_armkey.json"), key)
    print(
        f"[kappa] wrote {len(blind)} blind items -> {out_path}\n"
        f"        Label `human_task_correct` true/false in each row, save as "
        f"{HUMAN_LABELS_PATH.name}, then run `kappa`.\n"
        f"        Do not open human_sample_armkey.json before labelling."
    )
    return {"sampled": len(blind), "arms": sorted(by_arm), "seed": seed}


def compute(
    *,
    grades_path: Path | None = None,
    labels_path: Path = HUMAN_LABELS_PATH,
    threshold: float = 0.70,
) -> dict:
    from .judge import GRADES_PATH

    grades = read_jsonl(grades_path or GRADES_PATH)
    labels = read_jsonl(labels_path)
    if not labels:
        raise HarnessError(
            f"no human labels at {labels_path}. §5 requires the founder to hand-label "
            f"a stratified 60-run sample, blind to arm, BEFORE looking at any "
            f"aggregate result. Run `kappa-sample` first."
        )

    judge_by_run = {g["run_id"]: g for g in grades}
    human: list[bool] = []
    judge: list[bool] = []
    missing: list[str] = []
    unlabelled = 0
    for row in labels:
        h = row.get("human_task_correct")
        if h is None:
            unlabelled += 1
            continue
        g = judge_by_run.get(row["run_id"])
        if g is None:
            missing.append(row["run_id"])
            continue
        human.append(bool(h))
        judge.append(bool(g["task_correct"]))

    if unlabelled:
        raise HarnessError(
            f"{unlabelled} sampled items still have human_task_correct = null. "
            f"Label all of them; a partially labelled sample makes κ a number "
            f"about whichever items were easy to label."
        )

    k = cohens_kappa(human, judge)
    result = {
        "computed_at": utcnow(),
        "n": len(human),
        "kappa": round(k, 4),
        "threshold": threshold,
        "passes": k >= threshold,
        "raw_agreement": round(sum(1 for a, b in zip(human, judge) if a == b) / len(human), 4),
        "runs_missing_judge_grade": missing,
        "if_failed": (
            "Pre-registered fallback (§5): the judge is discarded and the primary "
            "metric is computed on a human-graded subset of 60 tasks, with the "
            "reduced power stated in the writeup."
        ),
    }
    write_json(KAPPA_PATH, result)
    print(f"[kappa] κ = {k:.4f} (n={len(human)}, threshold {threshold}) -> {'PASS' if result['passes'] else 'FAIL'}")
    return result


def enforce_gate(*, threshold: float = 0.70) -> dict:
    """Called by the analysis before it will report a judge-derived primary
    metric. Raises rather than warning — §5 makes this a gate, and a gate that
    prints a warning is not a gate."""
    if not KAPPA_PATH.exists():
        raise HarnessError(
            "the judge has not been validated against human labels.\n"
            "§5 pre-registers Cohen's κ ≥ 0.70 between judge and human as a gate on "
            "the primary metric. Run `kappa-sample`, hand-label the sample blind to "
            "arm, then run `kappa`.\n"
            "This check is not skippable: an unvalidated judge is the Memco failure "
            "mode (T5), and the whole point of this experiment is not to repeat it."
        )
    result = json.loads(KAPPA_PATH.read_text(encoding="utf-8"))
    if not result.get("passes"):
        raise HarnessError(
            f"judge validation FAILED: κ = {result['kappa']} < {threshold}.\n"
            f"Per §5 the judge is discarded. The primary metric must be computed on "
            f"a human-graded subset of 60 tasks, and the writeup must state the "
            f"reduced power. Re-run the analysis with --human-graded-only."
        )
    return result
