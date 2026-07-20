#!/usr/bin/env python3
"""Analysis — §5, §6, §7.

Computes the pre-registered primary metric and the outcome verdict, plus the
secondary family with Holm-Bonferroni adjustment. Reads thresholds from the
frozen config rather than hard-coding them, so the numbers in `results/` and the
numbers in `preregistration.md` cannot drift apart.

Pure stdlib. The bootstrap is a loop; at 10,000 resamples over 60 tasks it takes
under a second, and a dependency that fails to install in 2027 is a result that
cannot be reproduced.

Run:
    python3 v0/analysis/analyze.py --config mve.json
"""

from __future__ import annotations

import argparse
import json
import math
import random
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from v0.harness import kappa as kappa_mod  # noqa: E402
from v0.harness.config import Config, config_sha, load_config  # noqa: E402
from v0.harness.freeze import verify as verify_freeze  # noqa: E402
from v0.harness.judge import load_grades  # noqa: E402
from v0.harness.pricing import price_table_snapshot  # noqa: E402
from v0.harness.runner import load_runs  # noqa: E402
from v0.harness.tasks import load_eval  # noqa: E402
from v0.harness.util import V0_ROOT, HarnessError, utcnow, write_json  # noqa: E402

RESULTS_DIR = V0_ROOT / "results"

# The secondary family, §7. Fixed here so it cannot be extended after seeing data.
SECONDARY_PAIRS = [
    ("C_both", "B"),
    ("B", "A"),
    ("C_both", "A0"),
    ("C_both", "C_doc"),
    ("C_exp", "A0"),
]


# ---------------------------------------------------------------------------
# assembly
# ---------------------------------------------------------------------------


def per_task_scores(runs, grades, *, arm, size, model_alias):
    """Mean score per task across the runs-per-task, for one cell of the grid.

    Returns (scores, flips, excluded) where `scores[task_id]` is in [0, 1] and
    `flips[task_id]` is True when the runs for that task disagree (§7).
    """
    grade_by_run = {g["run_id"]: g for g in grades}
    buckets: dict[str, list[bool]] = {}
    excluded = {"truncated": 0, "error": 0, "ungraded": 0}
    for r in runs:
        if r["arm"] != arm or r["corpus_size"] != size or r["model_alias"] != model_alias:
            continue
        if r.get("finish_reason") in ("length", "max_tokens"):
            excluded["truncated"] += 1
            continue
        if r.get("error"):
            excluded["error"] += 1
            continue
        g = grade_by_run.get(r["run_id"])
        if g is None:
            excluded["ungraded"] += 1
            continue
        buckets.setdefault(r["task_id"], []).append(bool(g["task_correct"]))

    scores = {t: sum(v) / len(v) for t, v in buckets.items() if v}
    flips = {t: (0 < sum(v) < len(v)) for t, v in buckets.items() if v}
    return scores, flips, excluded


def paired(a: dict[str, float], b: dict[str, float]) -> tuple[list[float], list[float], list[str]]:
    """Pair by task. Tasks missing from either arm are dropped — an unpaired
    comparison is not the pre-registered test."""
    ids = sorted(set(a) & set(b))
    return [a[i] for i in ids], [b[i] for i in ids], ids


# ---------------------------------------------------------------------------
# statistics
# ---------------------------------------------------------------------------


def paired_bootstrap(
    treat: list[float], control: list[float], *, resamples: int, ci: float, seed: int = 7
) -> dict:
    """Paired bootstrap over tasks on the per-task mean score (§6).

    Resamples *tasks*, not runs — the pairing is by task, and resampling runs
    would treat three runs of one task as three independent observations.
    """
    n = len(treat)
    if n == 0:
        raise HarnessError("no paired tasks; cannot bootstrap")
    diffs = [t - c for t, c in zip(treat, control)]
    point = sum(diffs) / n
    rng = random.Random(seed)
    boots: list[float] = []
    for _ in range(resamples):
        s = 0.0
        for _ in range(n):
            s += diffs[rng.randrange(n)]
        boots.append(s / n)
    boots.sort()
    lo_i = int((1 - ci) / 2 * resamples)
    hi_i = min(resamples - 1, int((1 + ci) / 2 * resamples))
    return {
        "n_tasks": n,
        "point_estimate": point,
        "point_estimate_points": point * 100,
        "ci_low_points": boots[lo_i] * 100,
        "ci_high_points": boots[hi_i] * 100,
        "ci_level": ci,
        "resamples": resamples,
        "ci_excludes_zero": (boots[lo_i] > 0) or (boots[hi_i] < 0),
    }


def mcnemar_exact(treat: list[float], control: list[float]) -> dict:
    """Robustness check (§6): exact McNemar on the majority-vote binarization
    of the 3 runs."""
    b = c = 0  # b: treat wins, c: control wins
    for t, k in zip(treat, control):
        tb, cb = t >= 0.5, k >= 0.5
        if tb and not cb:
            b += 1
        elif cb and not tb:
            c += 1
    n = b + c
    if n == 0:
        return {"discordant": 0, "b": 0, "c": 0, "p_value": 1.0,
                "note": "no discordant pairs; the arms agree on every task"}
    # Two-sided exact binomial at p=0.5.
    k = min(b, c)
    tail = sum(math.comb(n, i) for i in range(0, k + 1)) / (2 ** n)
    return {"discordant": n, "b_treat_wins": b, "c_control_wins": c,
            "p_value": min(1.0, 2 * tail)}


def holm_bonferroni(pvalues: dict[str, float]) -> dict[str, float]:
    """§7: the secondary family is reported with Holm-adjusted p-values and
    labelled exploratory."""
    items = sorted(pvalues.items(), key=lambda kv: kv[1])
    m = len(items)
    adjusted: dict[str, float] = {}
    running = 0.0
    for i, (name, p) in enumerate(items):
        val = min(1.0, (m - i) * p)
        running = max(running, val)  # enforce monotonicity
        adjusted[name] = running
    return adjusted


# ---------------------------------------------------------------------------
# cost and secondary metrics
# ---------------------------------------------------------------------------


def cost_and_tokens(runs, *, arm, size, model_alias) -> dict:
    sel = [
        r for r in runs
        if r["arm"] == arm and r["corpus_size"] == size and r["model_alias"] == model_alias
        and not r.get("error")
    ]
    if not sel:
        return {}
    lat = sorted(r["latency_ms_total"] for r in sel)

    def pct(p: float) -> float:
        return lat[min(len(lat) - 1, int(p * len(lat)))]

    return {
        "runs": len(sel),
        # T8: raw token counts, so the analysis can be re-priced.
        "input_tokens_total": sum(r["input_tokens"] for r in sel),
        "cached_input_tokens_total": sum(r.get("cached_input_tokens", 0) for r in sel),
        "cache_write_tokens_total": sum(r.get("cache_write_tokens", 0) for r in sel),
        "output_tokens_total": sum(r["output_tokens"] for r in sel),
        "usd_marginal_total": sum(r["usd_marginal"] for r in sel),
        "usd_amortized_total": sum(r["usd_amortized_n200"] for r in sel),
        "usd_marginal_mean": sum(r["usd_marginal"] for r in sel) / len(sel),
        "usd_amortized_mean": sum(r["usd_amortized_n200"] for r in sel) / len(sel),
        "latency_ms_p50": pct(0.50),
        "latency_ms_p95": pct(0.95),
        "context_tokens_mean": sum(r["context_tokens"] for r in sel) / len(sel),
        "provider_model_versions": sorted({r.get("provider_model_version", "") for r in sel}),
        "truncated_for_model": sum(1 for r in sel if r.get("context_truncated_for_model")),
    }


def confabulation(runs, grades, tasks, *, arm, size, model_alias) -> dict:
    """§5 / T10: share of decoy tasks answered with a concrete claim instead of
    an abstention. Reported as a first-class result, not a footnote."""
    decoys = {t["task_id"] for t in tasks if t["gold"]["abstain_expected"]}
    grade_by_run = {g["run_id"]: g for g in grades}
    total = confab = 0
    for r in runs:
        if r["arm"] != arm or r["corpus_size"] != size or r["model_alias"] != model_alias:
            continue
        if r["task_id"] not in decoys or r.get("error"):
            continue
        g = grade_by_run.get(r["run_id"])
        if not g:
            continue
        total += 1
        confab += int(g["confabulated"])
    return {"decoy_runs": total, "confabulated": confab,
            "confabulation_rate": (confab / total) if total else None}


# ---------------------------------------------------------------------------
# verdict
# ---------------------------------------------------------------------------


def verdict(primary: dict, cost_ratio: float | None, cfg: Config) -> dict:
    """The three pre-registered outcomes, §6. No fourth outcome exists."""
    th = cfg.thresholds
    delta = primary["point_estimate_points"]
    excludes_zero = primary["ci_excludes_zero"] and primary["ci_low_points"] > 0

    if delta <= 0:
        outcome, why = "KILL", (
            f"point estimate of (C_both - A) is {delta:+.1f} points, which is <= 0. "
            f"Per §6: publish the finding and stop. No reinterpretation, no rerun."
        )
    elif delta >= th.proceed_delta_points and excludes_zero and (
        cost_ratio is not None and cost_ratio <= th.cost_multiple_max
    ):
        outcome, why = "PROCEED", (
            f"{delta:+.1f} points >= {th.proceed_delta_points}, bootstrap CI excludes zero, "
            f"and cost per correct answer is {cost_ratio:.2f}x arm A (<= {th.cost_multiple_max}x)."
        )
    else:
        reasons = []
        if delta < th.proceed_delta_points:
            reasons.append(f"{delta:+.1f} points < {th.proceed_delta_points}")
        if not excludes_zero:
            reasons.append("bootstrap 95% CI includes zero")
        if cost_ratio is None:
            reasons.append("cost ratio not computable")
        elif cost_ratio > th.cost_multiple_max:
            reasons.append(f"cost per correct answer {cost_ratio:.2f}x > {th.cost_multiple_max}x")
        outcome, why = "AMBER", (
            "no demonstrated advantage at this scale: " + "; ".join(reasons) +
            ". Does not authorize v1. One pre-specified extension only: rerun the "
            "primary comparison at corpus size L with the same thresholds."
        )

    out = {"outcome": outcome, "reasoning": why,
           "may_greenlight_per_config": cfg.may_greenlight}
    if outcome == "PROCEED" and not cfg.may_greenlight:
        out["outcome"] = "PROCEED-BLOCKED"
        out["reasoning"] = (
            why + "\n\nHOWEVER: this configuration is pre-registered as unable to "
            "green-light (D8, §9 — 'the MVE may kill the project; it may not "
            "green-light it'). A win at this n has a confidence interval too wide to "
            "license six months of engineering, and this configuration has no "
            "ablation, so it cannot speak to the v1 gate at all. The correct next "
            "step is the full design in config/full.json, not v1."
        )
    return out


# ---------------------------------------------------------------------------
# main
# ---------------------------------------------------------------------------


def analyze(cfg: Config, *, model_alias: str | None = None, size: str | None = None,
            require_kappa: bool = True) -> dict:
    size = size or cfg.primary_size
    alias = model_alias or cfg.models[0].alias

    verify_freeze()
    tasks = load_eval()
    runs = load_runs()
    grades = load_grades()

    mock_runs = sum(1 for r in runs if r.get("provider") == "mock")
    if mock_runs:
        print(
            f"!! {mock_runs} of {len(runs)} runs came from the mock provider. "
            f"Mock runs are plumbing checks, not measurements."
        )

    kappa_result = None
    if require_kappa:
        kappa_result = kappa_mod.enforce_gate(threshold=cfg.thresholds.kappa_min)

    arms_present = sorted({r["arm"] for r in runs})
    per_arm: dict[str, dict] = {}
    for arm in arms_present:
        scores, flips, excluded = per_task_scores(runs, grades, arm=arm, size=size, model_alias=alias)
        if not scores:
            continue
        flip_rate = sum(flips.values()) / len(flips) if flips else 0.0
        per_arm[arm] = {
            "n_tasks": len(scores),
            "mean_score": sum(scores.values()) / len(scores),
            "mean_score_points": 100 * sum(scores.values()) / len(scores),
            "flip_rate": flip_rate,
            # §7: pre-registered. Above 25%, the arm is too noisy to decide on.
            "flip_rate_exceeds_threshold": flip_rate > cfg.thresholds.flip_rate_max,
            "excluded": excluded,
            "cost": cost_and_tokens(runs, arm=arm, size=size, model_alias=alias),
            "confabulation": confabulation(runs, grades, tasks, arm=arm, size=size, model_alias=alias),
            "_scores": scores,
        }

    noisy = [a for a, v in per_arm.items() if v["flip_rate_exceeds_threshold"]]

    # ---- primary comparison -------------------------------------------
    primary = None
    mcnemar = None
    cost_ratio = None
    if "C_both" in per_arm and "A" in per_arm:
        t, c, ids = paired(per_arm["C_both"]["_scores"], per_arm["A"]["_scores"])
        primary = paired_bootstrap(
            t, c, resamples=cfg.thresholds.bootstrap_resamples, ci=cfg.thresholds.ci_level
        )
        primary["paired_task_ids"] = len(ids)
        mcnemar = mcnemar_exact(t, c)

        # Cost per correct answer, at the amortized policy (§5).
        def cpc(arm: str) -> float | None:
            v = per_arm[arm]
            correct = v["mean_score"] * v["n_tasks"]
            if correct <= 0 or not v["cost"]:
                return None
            return v["cost"]["usd_amortized_total"] / correct

        cpc_c, cpc_a = cpc("C_both"), cpc("A")
        if cpc_c is not None and cpc_a not in (None, 0):
            cost_ratio = cpc_c / cpc_a

    # ---- secondary family ---------------------------------------------
    secondary: dict[str, dict] = {}
    pvals: dict[str, float] = {}
    for treat, control in SECONDARY_PAIRS:
        if treat not in per_arm or control not in per_arm:
            continue
        t, c, _ = paired(per_arm[treat]["_scores"], per_arm[control]["_scores"])
        if not t:
            continue
        name = f"{treat}_vs_{control}"
        boot = paired_bootstrap(t, c, resamples=cfg.thresholds.bootstrap_resamples,
                                ci=cfg.thresholds.ci_level, seed=11)
        mc = mcnemar_exact(t, c)
        secondary[name] = {"bootstrap": boot, "mcnemar": mc}
        pvals[name] = mc["p_value"]
    for name, adj in holm_bonferroni(pvals).items():
        secondary[name]["p_holm_adjusted"] = adj
        secondary[name]["label"] = "exploratory — may not override a KILL on the primary (§7)"

    v = verdict(primary, cost_ratio, cfg) if primary else {
        "outcome": "INCOMPLETE",
        "reasoning": "the primary comparison needs both C_both and A runs at the primary size.",
    }
    if noisy:
        v["blocked_by_flip_rate"] = (
            f"arms {noisy} exceed the pre-registered {cfg.thresholds.flip_rate_max:.0%} flip rate. "
            f"§7: raise runs-per-task to 5 and re-run before reading the result. "
            f"This verdict is not yet readable."
        )

    report = {
        "generated_at": utcnow(),
        "config": cfg.name,
        "config_sha": config_sha(cfg),
        "taskset_sha": verify_freeze()["taskset_sha"],
        "model_alias": alias,
        "corpus_size": size,
        "eval_tasks": len(tasks),
        "runs_analyzed": len(runs),
        "runs_from_mock_provider": mock_runs,
        "judge_validation": kappa_result,
        "primary_metric": {
            "definition": (
                "Task-level strict accuracy on the eval set, at corpus size "
                f"{size}, arm C_both versus arm A, paired by task, averaged over "
                f"{cfg.runs_per_task} runs."
            ),
            "bootstrap": primary,
            "mcnemar_majority_vote": mcnemar,
            "cost_per_correct_ratio_C_over_A": cost_ratio,
        },
        "per_arm": {a: {k: val for k, val in v2.items() if k != "_scores"} for a, v2 in per_arm.items()},
        "contamination_floor_T6": {
            "a0_residual_accuracy_points": per_arm.get("A0", {}).get("mean_score_points"),
            "note": (
                "A0's residual score on the frozen eval set. Published as the "
                "contamination floor per T6. Every other arm's score should be read "
                "against this, not against zero."
            ),
        },
        "secondary_family": secondary,
        "prices": price_table_snapshot(),
        "thresholds": cfg.thresholds.__dict__,
        "verdict": v,
    }

    RESULTS_DIR.mkdir(parents=True, exist_ok=True)
    out = RESULTS_DIR / f"analysis-{cfg.name}-{alias}-{size}.json"
    write_json(out, report)
    _print_summary(report)
    return report


def _print_summary(rep: dict) -> None:
    print("\n" + "=" * 72)
    print(f"CRED v0 — {rep['config']} — {rep['model_alias']} — size {rep['corpus_size']}")
    print("=" * 72)
    print(f"task set sha : {rep['taskset_sha'][:32]}...")
    print(f"eval tasks   : {rep['eval_tasks']}   runs: {rep['runs_analyzed']}")
    if rep["runs_from_mock_provider"]:
        print(f"!! {rep['runs_from_mock_provider']} mock runs — this is plumbing, not a measurement")
    print("\nPer arm:")
    for arm, v in rep["per_arm"].items():
        conf = v["confabulation"].get("confabulation_rate")
        print(
            f"  {arm:8s} acc {v['mean_score_points']:6.2f}  n={v['n_tasks']:3d}  "
            f"flip {v['flip_rate']:.2f}  confab {('n/a' if conf is None else f'{conf:.2f}')}"
        )
    b = rep["primary_metric"]["bootstrap"]
    if b:
        print(
            f"\nPrimary (C_both - A): {b['point_estimate_points']:+.2f} points  "
            f"95% CI [{b['ci_low_points']:+.2f}, {b['ci_high_points']:+.2f}]  "
            f"n={b['n_tasks']}"
        )
        mc = rep["primary_metric"]["mcnemar_majority_vote"]
        print(f"McNemar (majority vote): p = {mc['p_value']:.4f}, discordant = {mc['discordant']}")
        cr = rep["primary_metric"]["cost_per_correct_ratio_C_over_A"]
        print(f"Cost per correct, C_both / A: {'n/a' if cr is None else f'{cr:.2f}x'}")
    print(f"\nVERDICT: {rep['verdict']['outcome']}")
    print(f"  {rep['verdict']['reasoning']}")
    if "blocked_by_flip_rate" in rep["verdict"]:
        print(f"  !! {rep['verdict']['blocked_by_flip_rate']}")
    print("=" * 72 + "\n")


def main() -> int:
    ap = argparse.ArgumentParser(description="CRED v0 analysis (§5, §6, §7)")
    ap.add_argument("--config", default="mve.json")
    ap.add_argument("--model", default=None, help="model alias")
    ap.add_argument("--size", default=None, help="corpus size (default: the pre-registered primary)")
    ap.add_argument(
        "--skip-kappa-gate",
        action="store_true",
        help="plumbing checks only. Produces a report that must not be published.",
    )
    args = ap.parse_args()
    cfg = load_config(args.config)
    try:
        analyze(cfg, model_alias=args.model, size=args.size, require_kappa=not args.skip_kappa_gate)
    except HarnessError as exc:
        print(f"\nANALYSIS HALTED\n{exc}\n", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
