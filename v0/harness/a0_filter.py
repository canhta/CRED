"""The A0 floor filter — §4 step 4, and T6.

The design calls this "the single highest-value line in the design". It runs
arm A0 — no memory, question only — on every candidate task, three times, and
**deletes every task A0 gets right**. What survives is, by construction, not
answerable from the model's parametric knowledge alone.

Two decisions this module makes that the design leaves open, both stated rather
than buried:

1. **"Gets right" means any of the three runs.** Not majority. A task the model
   answers correctly one time in three is one it partly knows, and a partly-known
   task inflates every arm's score without discriminating between them. The
   stricter rule costs us tasks; that is the trade and it is the right side of it.

2. **Abstention decoys are exempt from deletion.** A0 has no memory, so
   abstaining is trivially correct for it on a decoy — applying the deletion rule
   literally would delete 100% of the decoys and remove the confabulation metric
   T10 exists to produce. A0's abstention rate on decoys is instead reported as a
   diagnostic: if it is low, A0 is confabulating on decoys, which is itself worth
   publishing.

This stage is re-runnable. It writes its own runs under `runs/_a0_filter/` so
that filter runs are never mixed into the eval grid, and it is idempotent —
re-running skips tasks already decided.
"""

from __future__ import annotations

from pathlib import Path

from .config import Config
from .judge import grade_run
from .runner import RUNS_ROOT, run_grid
from .tasks import CANDIDATES_PATH, load_candidates, reject
from .util import V0_ROOT, read_jsonl, utcnow, write_json, write_jsonl

FILTER_RUNS_ROOT = V0_ROOT / "runs" / "_a0_filter"
FILTER_REPORT_PATH = V0_ROOT / "tasks" / "A0_FILTER.json"


def run_filter(
    cfg: Config,
    *,
    candidates_path: Path = CANDIDATES_PATH,
    model_alias: str | None = None,
    resume: bool = True,
) -> dict:
    tasks = load_candidates(candidates_path)
    undecided = [t for t in tasks if t["filters"].get("a0_solved") is None]
    print(f"[a0-filter] {len(tasks)} candidates, {len(undecided)} undecided")

    aliases = [model_alias] if model_alias else [cfg.models[0].alias]
    run_grid(
        cfg,
        undecided,
        arms=["A0"],
        sizes=[cfg.primary_size],
        model_aliases=aliases,
        root=FILTER_RUNS_ROOT,
        resume=resume,
    )

    runs: list[dict] = []
    for p in sorted(FILTER_RUNS_ROOT.rglob("runs.jsonl")):
        runs.extend(read_jsonl(p))
    by_task: dict[str, list[dict]] = {}
    for r in runs:
        by_task.setdefault(r["task_id"], []).append(r)

    kept: list[dict] = []
    deleted = 0
    decoy_abstain_hits = 0
    decoy_runs = 0
    factual_checked = 0
    factual_solved = 0

    for t in tasks:
        rs = by_task.get(t["task_id"], [])
        usable = [r for r in rs if not r.get("error") and r.get("finish_reason") not in ("length", "max_tokens")]
        if not usable:
            # No usable A0 evidence: keep the task but mark it, so the writeup
            # can say how many tasks entered the set unfiltered.
            t["filters"]["a0_solved"] = None
            t["filters"]["a0_note"] = "no usable A0 run"
            kept.append(t)
            continue

        grades = [grade_run(r, t, cfg) for r in usable]

        if t["gold"]["abstain_expected"]:
            decoy_runs += len(grades)
            decoy_abstain_hits += sum(1 for g in grades if g["abstained"])
            t["filters"]["a0_solved"] = False
            t["filters"]["a0_note"] = "decoy: exempt from deletion (see a0_filter.py docstring)"
            t["filters"]["a0_abstain_rate"] = round(
                sum(1 for g in grades if g["abstained"]) / len(grades), 3
            )
            kept.append(t)
            continue

        factual_checked += 1
        solved = any(g["task_correct"] for g in grades)
        t["filters"]["a0_solved"] = solved
        t["filters"]["a0_correct_runs"] = sum(1 for g in grades if g["task_correct"])
        t["filters"]["a0_total_runs"] = len(grades)
        if solved:
            factual_solved += 1
            reject(
                t,
                f"A0 solved this task in {t['filters']['a0_correct_runs']}/{len(grades)} runs "
                f"with no memory: it is answerable from parametric knowledge, not from "
                f"organizational memory (§4 step 4)",
                stage="a0_filter",
            )
            deleted += 1
            continue
        kept.append(t)

    write_jsonl(candidates_path, kept)

    report = {
        "filtered_at": utcnow(),
        "rule": "delete any factual task A0 answers correctly in ANY of its runs",
        "decoy_rule": "abstention decoys are exempt from deletion; see a0_filter.py",
        "candidates_in": len(tasks),
        "factual_tasks_checked": factual_checked,
        "factual_tasks_deleted": factual_solved,
        "a0_pre_filter_solve_rate": round(factual_solved / factual_checked, 4) if factual_checked else None,
        "candidates_out": len(kept),
        "decoy_runs": decoy_runs,
        "a0_decoy_abstention_rate": round(decoy_abstain_hits / decoy_runs, 4) if decoy_runs else None,
        "runs_root": str(FILTER_RUNS_ROOT),
        "note": (
            "a0_pre_filter_solve_rate is the contamination level of the *candidate* "
            "set. T6 also requires publishing A0's residual score on the frozen eval "
            "set as the contamination floor — that comes from the eval grid, not from "
            "this stage, and is computed by analysis/analyze.py."
        ),
    }
    write_json(FILTER_REPORT_PATH, report)
    print(
        f"[a0-filter] deleted {deleted} of {factual_checked} factual tasks "
        f"(pre-filter A0 solve rate {report['a0_pre_filter_solve_rate']}); "
        f"{len(kept)} candidates remain"
    )
    return report
