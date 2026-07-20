"""Task records, filters, and set composition — §4 and §11.

The task record shape in §11 is the contract between every other stage. It is
constructed in exactly one place (`make_task`) so a field cannot drift.

The pipeline this module implements, in order:

    candidates.jsonl                 (draft.py writes)
      -> verbatim filter             (§4 step 5)
      -> A0 filter                   (§4 step 4, a0_filter.py)
      -> founder rejection           (§4 step 6, subtract-only, logged)
      -> compose to family shares    (§4 composition table)
      -> split eval / dev            (§6)
      -> freeze                      (§4 step 8, freeze.py)
"""

from __future__ import annotations

import random
import re
from pathlib import Path

from .config import Config
from .util import V0_ROOT, HarnessError, read_jsonl, utcnow, write_jsonl

TASKS_DIR = V0_ROOT / "tasks"
CANDIDATES_PATH = TASKS_DIR / "candidates.jsonl"
EVAL_PATH = TASKS_DIR / "eval.jsonl"
DEV_PATH = TASKS_DIR / "dev.jsonl"
REJECTED_PATH = TASKS_DIR / "rejected.jsonl"
TASKSET_SHA_PATH = TASKS_DIR / "TASKSET.sha256"

FAMILIES = [
    "decision_rationale",
    "unwritten_convention",
    "cross_cutting_context",
    "failure_recall",
    "abstention_decoy",
]

MATCH_TYPES = {"any_of", "all_of", "regex", "judge"}


# ---------------------------------------------------------------------------
# construction
# ---------------------------------------------------------------------------


def make_task(
    *,
    task_id: str,
    family: str,
    repo: str,
    cutoff_t: str,
    question: str,
    gold_answer: str,
    checkpoints: list[dict],
    abstain_expected: bool,
    anchor_url: str,
    anchor_sha: str | None,
    anchor_date: str,
    evidence_spans: list[dict],
    author: str,
) -> dict:
    """Build one §11 task record. Validates as it goes — a malformed task
    surfaces here, not three stages later inside the judge."""
    if family not in FAMILIES:
        raise HarnessError(f"unknown family {family!r}")
    if family == "abstention_decoy" and not abstain_expected:
        raise HarnessError("decoy tasks must set abstain_expected=True")
    if family != "abstention_decoy" and abstain_expected:
        raise HarnessError("only decoy tasks may set abstain_expected=True")
    if not abstain_expected:
        if not 1 <= len(checkpoints) <= 4:
            raise HarnessError(f"{task_id}: gold must decompose into 1-4 checkpoints, got {len(checkpoints)}")
        for cp in checkpoints:
            mt = cp.get("match", {}).get("type")
            if mt not in MATCH_TYPES:
                raise HarnessError(f"{task_id}: checkpoint {cp.get('id')} has bad match type {mt!r}")
            if mt in ("any_of", "all_of") and not cp["match"].get("values"):
                raise HarnessError(f"{task_id}: checkpoint {cp.get('id')} has no match values")
    return {
        "task_id": task_id,
        "family": family,
        "repo": repo,
        "cutoff_t": cutoff_t,
        "question": question.strip(),
        "gold": {
            "answer": gold_answer.strip(),
            "checkpoints": checkpoints,
            "abstain_expected": abstain_expected,
        },
        "provenance": {
            "anchor_url": anchor_url,
            "anchor_sha": anchor_sha,
            "anchor_date": anchor_date,
            "evidence_spans": evidence_spans,
        },
        "filters": {"a0_solved": None, "verbatim_in_tree_at_t": None},
        "audit": {"sampled": False, "flagged_biased": None},
        "author": author,
        "reject_reason": None,
        "set": "candidate",
    }


# ---------------------------------------------------------------------------
# rejection log — §4 step 6, the founder may only subtract, and every
# subtraction is published.
# ---------------------------------------------------------------------------


def reject(task: dict, reason: str, *, stage: str) -> dict:
    row = {**task, "reject_reason": reason, "rejected_by_stage": stage, "rejected_at": utcnow(), "set": "rejected"}
    from .util import append_jsonl

    append_jsonl(REJECTED_PATH, row)
    return row


def load_rejected() -> list[dict]:
    return read_jsonl(REJECTED_PATH)


# ---------------------------------------------------------------------------
# §4 step 5: verbatim presence filter
# ---------------------------------------------------------------------------

_WORD = re.compile(r"[A-Za-z0-9_]{4,}")


def _shingles(text: str, n: int = 6) -> set[tuple[str, ...]]:
    toks = [w.lower() for w in _WORD.findall(text)]
    return {tuple(toks[i : i + n]) for i in range(max(0, len(toks) - n + 1))}


def verbatim_in_tree(gold_answer: str, tree_text: str, *, n: int = 6, threshold: float = 0.5) -> bool:
    """True when the gold answer is substantially present verbatim in the
    working tree at T.

    Exact substring match is too weak — a maintainer's answer is rarely quoted
    word for word — so this is a 6-gram overlap test. `threshold` is the share
    of the gold answer's 6-grams that must appear in the tree. 0.5 is a
    judgement call, and it is stated as one; it is recorded in the filter stats
    so a reader can see how aggressive the filter was.
    """
    gold_grams = _shingles(gold_answer, n)
    if not gold_grams:
        return False
    tree_grams = _shingles(tree_text, n)
    hit = len(gold_grams & tree_grams)
    return hit / len(gold_grams) >= threshold


# ---------------------------------------------------------------------------
# composition and split
# ---------------------------------------------------------------------------


def compose(
    candidates: list[dict],
    cfg: Config,
    *,
    seed: int = 20260720,
) -> tuple[list[dict], list[dict], list[str]]:
    """Select eval and dev sets matching the §4 composition table.

    Returns (eval_tasks, dev_tasks, warnings). Under-supply in a family is a
    warning, never a silent substitution: filling a decision-rationale shortfall
    with convention tasks would change what the experiment measures while
    leaving the composition table looking satisfied.
    """
    rng = random.Random(seed)
    pool: dict[str, list[dict]] = {f: [] for f in FAMILIES}
    for t in candidates:
        if t.get("reject_reason"):
            continue
        if t["filters"].get("a0_solved"):
            continue
        if t["filters"].get("verbatim_in_tree_at_t"):
            continue
        pool[t["family"]].append(t)
    for f in pool:
        pool[f].sort(key=lambda t: t["task_id"])
        rng.shuffle(pool[f])

    warnings: list[str] = []

    def quota(n: int) -> dict[str, int]:
        """Largest-remainder allocation, so the shares are exact at the target n
        rather than drifting with independent rounding. This matters: T10's 15%
        decoy share is a gate, and naive per-family rounding misses it at small n."""
        exact = {f: cfg.family_shares.get(f, 0.0) * n for f in FAMILIES}
        base = {f: int(v) for f, v in exact.items()}
        short = n - sum(base.values())
        for f in sorted(FAMILIES, key=lambda f: -(exact[f] - base[f]))[:short]:
            base[f] += 1
        return base

    # Quotas are computed per split so that the eval set — the thing the gate
    # is checked against — hits its shares exactly.
    dev_want = quota(cfg.n_dev_tasks)
    eval_want = quota(cfg.n_eval_tasks)

    dev: list[dict] = []
    ev: list[dict] = []
    for f in FAMILIES:
        have = pool[f]
        need = dev_want[f] + eval_want[f]
        if len(have) < need:
            warnings.append(
                f"family {f}: wanted {need} ({eval_want[f]} eval + {dev_want[f]} dev), "
                f"have {len(have)}. The set is under-composed. Mine more anchors "
                f"rather than substituting another family."
            )
        # Dev is drawn first, so tuning never sees an eval task.
        dev.extend(have[: dev_want[f]])
        ev.extend(have[dev_want[f] : need])

    rng.shuffle(dev)
    rng.shuffle(ev)

    if len(ev) < cfg.n_eval_tasks:
        warnings.append(
            f"eval set is {len(ev)} tasks against a pre-registered n of "
            f"{cfg.n_eval_tasks}. Power is lower than §7 states; the writeup must "
            f"say so."
        )

    ev = [{**t, "set": "eval"} for t in ev]
    dev = [{**t, "set": "dev"} for t in dev]
    return ev, dev, warnings


def load_eval() -> list[dict]:
    tasks = read_jsonl(EVAL_PATH)
    if not tasks:
        raise HarnessError(f"no eval tasks at {EVAL_PATH}")
    return tasks


def load_dev() -> list[dict]:
    return read_jsonl(DEV_PATH)


def load_candidates(path: Path = CANDIDATES_PATH) -> list[dict]:
    tasks = read_jsonl(path)
    if not tasks:
        raise HarnessError(f"no candidate tasks at {path}: run `draft` first")
    return tasks
