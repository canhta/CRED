"""Blind external audit — §4 step 7.

"One person who is not the founder reviews a random 20% sample, blind to
results, answering one question per task: *does this task appear constructed to
favour structured memory?* Flag rate is published. A flag rate above 15% voids
the set."

The auditor sees the question, the family, the repository, and the anchor URL —
enough to judge whether the task was constructed to favour a memory system. The
auditor does **not** see the gold answer, the checkpoints, or any result: a gold
answer is a strong hint about what kind of system would produce it, and the
whole point of this step is an opinion formed without that hint.
"""

from __future__ import annotations

import random
from pathlib import Path

from .tasks import EVAL_PATH, TASKS_DIR, load_eval
from .util import HarnessError, read_jsonl, utcnow, write_json, write_jsonl

AUDIT_SAMPLE_PATH = TASKS_DIR / "audit_sample.jsonl"
AUDIT_LABELS_PATH = TASKS_DIR / "audit_labels.jsonl"
AUDIT_REPORT_PATH = TASKS_DIR / "AUDIT.json"


def draw_sample(*, fraction: float = 0.20, seed: int = 5150) -> dict:
    tasks = load_eval()
    n = max(1, round(len(tasks) * fraction))
    rng = random.Random(seed)
    picked = sorted(tasks, key=lambda t: t["task_id"])
    rng.shuffle(picked)
    picked = picked[:n]

    blind = [
        {
            "task_id": t["task_id"],
            "family": t["family"],
            "repo": t["repo"],
            "question": t["question"],
            "anchor_url": t["provenance"]["anchor_url"],
            # The auditor fills this in: true if the task appears constructed
            # to favour structured memory.
            "flagged_biased": None,
            "note": "",
        }
        for t in picked
    ]
    write_jsonl(AUDIT_SAMPLE_PATH, blind)
    print(
        f"[audit] {len(blind)} tasks ({fraction:.0%} of {len(tasks)}) -> {AUDIT_SAMPLE_PATH}\n"
        f"        Give this file to someone who is NOT the founder. One question per "
        f"task: does this task appear constructed to favour structured memory?\n"
        f"        They set flagged_biased true/false and save as {AUDIT_LABELS_PATH.name}."
    )
    return {"sampled": len(blind), "of": len(tasks), "fraction": fraction, "seed": seed}


def score(*, threshold: float = 0.15, labels_path: Path = AUDIT_LABELS_PATH) -> dict:
    labels = read_jsonl(labels_path)
    if not labels:
        raise HarnessError(f"no audit labels at {labels_path}: run `audit-sample` first")
    unlabelled = [r["task_id"] for r in labels if r.get("flagged_biased") is None]
    if unlabelled:
        raise HarnessError(
            f"{len(unlabelled)} audited tasks are unlabelled. A partial audit "
            f"produces a flag rate about whichever tasks were easy to judge."
        )

    flagged = [r["task_id"] for r in labels if r["flagged_biased"]]
    rate = len(flagged) / len(labels)
    report = {
        "audited_at": utcnow(),
        "n_audited": len(labels),
        "n_flagged": len(flagged),
        "flag_rate": round(rate, 4),
        "void_threshold": threshold,
        "set_voided": rate > threshold,
        "flagged_task_ids": flagged,
        "question_asked": "Does this task appear constructed to favour structured memory?",
        "note": (
            "Published regardless of outcome (§12). The auditor saw question, "
            "family, repository, and anchor URL — never the gold answer, the "
            "checkpoints, or any result."
        ),
    }
    write_json(AUDIT_REPORT_PATH, report)
    if report["set_voided"]:
        raise HarnessError(
            f"AUDIT FAILED: flag rate {rate:.1%} exceeds the {threshold:.0%} voiding "
            f"threshold (§4 step 7). The task set is void. It cannot be repaired by "
            f"deleting the flagged tasks — that is the founder steering the set, which "
            f"is what the audit exists to detect. Re-mine and re-draft."
        )
    print(f"[audit] flag rate {rate:.1%} (threshold {threshold:.0%}) -> PASS")
    return report
