"""Grading — §5, and T5.

The design's constraints on the judge, and where each is enforced:

  - **Atomic checkpoints, not holistic quality.** The judge is only ever asked
    "does this answer assert this one fact?" (`prompts/judge.md`). There is no
    code path that asks a model to score an answer.
  - **String matching first.** Only checkpoints whose `match.type == "judge"`
    reach a model. Everything else is decided by `_string_match`, which is
    deterministic and free.
  - **Arm-blind.** `grade_run` receives the answer text and the checkpoint and
    nothing else — no arm, no model, no token counts, no run id in the prompt.
    The batch grader shuffles order with a fixed seed.
  - **Strict task correctness.** A task is correct iff *all* its checkpoints hit.
  - **κ ≥ 0.70 gate.** Implemented in `kappa.py` as a real check that raises.
"""

from __future__ import annotations

import json
import random
import re
from pathlib import Path

from . import prompts
from .config import Config
from .providers import get_provider
from .util import (
    V0_ROOT,
    HarnessError,
    append_jsonl,
    read_jsonl,
    utcnow,
)

GRADES_PATH = V0_ROOT / "grades" / "grades.jsonl"

# The literal every arm's template instructs the model to emit, plus the
# paraphrases models produce anyway. Kept narrow on purpose: a loose abstention
# detector would score confabulations as abstentions and destroy the metric
# T10 exists to produce.
ABSTAIN_PATTERNS = [
    r"^\s*the corpus does not say\.?\s*$",
    r"\bthe corpus does not say\b",
    r"\bthe (corpus|record|history) does not (say|record|contain|state)\b",
    r"\b(this|that) is not recorded (anywhere)?\b",
    r"\bno (record|mention) of (this|that) (exists|appears)\b",
]
_ABSTAIN = [re.compile(p, re.IGNORECASE) for p in ABSTAIN_PATTERNS]

_JSON_BLOCK = re.compile(r"\{.*\}", re.DOTALL)
_NORM = re.compile(r"[^a-z0-9]+")


def _normalize(text: str) -> str:
    return _NORM.sub(" ", text.lower()).strip()


def is_abstention(answer: str) -> bool:
    a = answer.strip()
    if not a:
        return False
    return any(rx.search(a) for rx in _ABSTAIN)


# ---------------------------------------------------------------------------
# deterministic checkpoint matching
# ---------------------------------------------------------------------------


def _string_match(cp: dict, answer: str) -> bool | None:
    """Returns True/False for a deterministic match rule, or None when the
    checkpoint must go to a judge."""
    m = cp.get("match", {})
    kind = m.get("type")
    norm = _normalize(answer)
    if kind == "any_of":
        return any(_normalize(v) in norm for v in m.get("values", []))
    if kind == "all_of":
        return all(_normalize(v) in norm for v in m.get("values", []))
    if kind == "regex":
        return bool(re.search(m["pattern"], answer, re.IGNORECASE))
    if kind == "judge":
        return None
    raise HarnessError(f"unknown checkpoint match type {kind!r}")


# ---------------------------------------------------------------------------
# grading one run
# ---------------------------------------------------------------------------


def grade_run(
    run: dict,
    task: dict,
    cfg: Config,
    *,
    grader_alias: str | None = None,
    grader_label: str = "judge-A",
) -> dict:
    """Grade one run record against its task. Returns a §11 grade record.

    Note what is *not* read off `run`: the arm, the model, the token counts.
    Only `run_id` (for joining) and `answer_text` are used.
    """
    answer = run.get("answer_text") or ""
    abstained = is_abstention(answer)
    expects_abstain = task["gold"]["abstain_expected"]

    results: list[dict] = []
    judge_calls = 0
    tmpl_sha = prompts.template_sha("judge")
    provider = None
    spec = None

    if not expects_abstain:
        for cp in task["gold"]["checkpoints"]:
            det = _string_match(cp, answer)
            if det is not None:
                results.append({"id": cp["id"], "hit": det, "method": "string"})
                continue
            if abstained:
                # Rule 4 of the judge prompt, applied without spending a call.
                results.append({"id": cp["id"], "hit": False, "method": "abstention"})
                continue
            if provider is None:
                spec = cfg.model(grader_alias) if grader_alias else cfg.models[0]
                provider = get_provider(spec.provider)
            rendered = prompts.render(
                "judge", {"CHECKPOINT_TEXT": cp["text"], "ANSWER": answer}
            )
            comp = provider.complete(
                model_id=spec.model_id,
                system="You grade one atomic checkpoint. Output JSON only.",
                user=rendered,
                max_tokens=300,
                temperature=cfg.temperature,
            )
            judge_calls += 1
            hit = False
            if not comp.error:
                m = _JSON_BLOCK.search(comp.text)
                if m:
                    try:
                        hit = bool(json.loads(m.group(0)).get("hit"))
                    except json.JSONDecodeError:
                        hit = False
            results.append({"id": cp["id"], "hit": hit, "method": "judge"})

    if expects_abstain:
        task_correct = abstained
        confabulated = not abstained
    else:
        task_correct = bool(results) and all(r["hit"] for r in results)
        confabulated = False

    return {
        "run_id": run["run_id"],
        "task_id": task["task_id"],
        "grader": grader_label,
        "grader_model": spec.model_id if spec else None,
        "judge_prompt_sha": tmpl_sha,
        "judge_calls": judge_calls,
        "checkpoint_results": results,
        "task_correct": task_correct,
        "abstained": abstained,
        "confabulated": confabulated,
        "graded_at": utcnow(),
    }


# ---------------------------------------------------------------------------
# batch
# ---------------------------------------------------------------------------


def grade_all(
    runs: list[dict],
    tasks: list[dict],
    cfg: Config,
    *,
    out_path: Path = GRADES_PATH,
    grader_alias: str | None = None,
    grader_label: str = "judge-A",
    seed: int = 424242,
    resume: bool = True,
) -> dict:
    """Grade a set of runs, arm-blind and in randomized order.

    The shuffle is the T5 mitigation that matters most in practice: graded in
    run order, a model sees all of arm A0's empty answers first and calibrates
    on them. A fixed seed keeps it reproducible.
    """
    by_id = {t["task_id"]: t for t in tasks}
    done: set[tuple[str, str]] = set()
    if resume:
        done = {(g["run_id"], g["grader"]) for g in read_jsonl(out_path)}

    pending = [r for r in runs if (r["run_id"], grader_label) not in done]
    rng = random.Random(seed)
    rng.shuffle(pending)

    stats = {"graded": 0, "skipped_existing": len(runs) - len(pending), "excluded_truncated": 0,
             "excluded_error": 0, "missing_task": 0, "judge_calls": 0}

    for r in pending:
        # §11: a run that ended on `length` is truncated, not wrong. Excluded
        # with a reported count rather than graded as incorrect.
        if r.get("finish_reason") in ("length", "max_tokens"):
            stats["excluded_truncated"] += 1
            continue
        if r.get("error"):
            stats["excluded_error"] += 1
            continue
        task = by_id.get(r["task_id"])
        if task is None:
            stats["missing_task"] += 1
            continue
        g = grade_run(r, task, cfg, grader_alias=grader_alias, grader_label=grader_label)
        append_jsonl(out_path, g)
        stats["graded"] += 1
        stats["judge_calls"] += g["judge_calls"]
        if stats["graded"] % 50 == 0:
            print(f"    graded {stats['graded']}/{len(pending)}")

    print(f"[grade] {stats}")
    return stats


def load_grades(path: Path = GRADES_PATH) -> list[dict]:
    return read_jsonl(path)
