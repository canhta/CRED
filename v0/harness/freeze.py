"""Task-set freeze — §4 step 8, D4, and T4.

"`sha256` of `eval.jsonl` is committed and tagged before any eval run."

The freeze is the pivot of the whole design. Before it, the task set can be
subtracted from. After it, nothing about the task set may change, and every
downstream artefact carries its hash: run records carry `taskset_sha`, memory
builds carry `taskset_sha_at_build`. A reader who wants to check that the memory
was not built from the answers compares two hex strings.

`freeze()` refuses in three cases, all of which would make the experiment
unpublishable rather than merely untidy.
"""

from __future__ import annotations

from pathlib import Path

from .config import Config
from .tasks import (
    DEV_PATH,
    EVAL_PATH,
    TASKSET_SHA_PATH,
    load_candidates,
    compose,
)
from .util import HarnessError, sha256_file, utcnow, write_json, write_jsonl

FREEZE_RECORD_PATH = TASKSET_SHA_PATH.with_name("FREEZE.json")


def freeze(cfg: Config, *, allow_mock: bool = False, force: bool = False) -> dict:
    if TASKSET_SHA_PATH.exists() and not force:
        current = TASKSET_SHA_PATH.read_text(encoding="utf-8").split()[0]
        raise HarnessError(
            f"the task set is already frozen at {current[:16]}...\n"
            f"D4 makes the freeze one-way: re-freezing after seeing any result is "
            f"exactly the post-hoc reinterpretation the pre-registration exists to "
            f"prevent. If you are re-composing before any eval run, pass --force and "
            f"say so in the writeup."
        )

    candidates = load_candidates()

    # 1. Nothing undecided by the A0 filter.
    undecided = [t for t in candidates if t["filters"].get("a0_solved") is None
                 and not t["gold"]["abstain_expected"]]
    if undecided:
        raise HarnessError(
            f"{len(undecided)} factual candidates have not been through the A0 filter. "
            f"§4 step 4 runs before the freeze, not after. Run `a0-filter`."
        )

    # 2. No mock-authored tasks.
    mock_authored = [t for t in candidates if "mock" in (t.get("author") or "").lower()]
    if mock_authored and not allow_mock:
        raise HarnessError(
            f"{len(mock_authored)} tasks were drafted by the mock provider. These are "
            f"placeholder text, not tasks. Freezing them would produce a hash that "
            f"looks authoritative and means nothing. Draft with a real model, or pass "
            f"--allow-mock if you are deliberately freezing a plumbing-test set."
        )

    ev, dev, warnings = compose(candidates, cfg)
    for w in warnings:
        print(f"[freeze] WARNING: {w}")

    # 3. Decoy share must be what was pre-registered (T10).
    n_decoy = sum(1 for t in ev if t["gold"]["abstain_expected"])
    share = n_decoy / len(ev) if ev else 0.0
    if abs(share - cfg.decoy_share) > 0.05:
        raise HarnessError(
            f"decoy share is {share:.2%} against a pre-registered {cfg.decoy_share:.0%}. "
            f"T10 makes the decoys non-optional; without them a memory arm can win by "
            f"confabulating and the judge will reward it."
        )

    write_jsonl(EVAL_PATH, ev)
    write_jsonl(DEV_PATH, dev)
    sha = sha256_file(EVAL_PATH)
    TASKSET_SHA_PATH.write_text(f"{sha}  eval.jsonl\n", encoding="utf-8")

    record = {
        "frozen_at": utcnow(),
        "eval_sha256": sha,
        "dev_sha256": sha256_file(DEV_PATH),
        "n_eval": len(ev),
        "n_dev": len(dev),
        "n_decoy_eval": n_decoy,
        "decoy_share_eval": round(share, 4),
        "config": cfg.name,
        "families_eval": _count(ev),
        "repos_eval": _count(ev, key="repo"),
        "warnings": warnings,
        "next_step": (
            "Commit tasks/ and tag `v0-prereg` BEFORE running the eval grid or the "
            "claim extractor. The tag SHA is published in the writeup (§12)."
        ),
    }
    write_json(FREEZE_RECORD_PATH, record)
    print(f"[freeze] eval.jsonl sha256 = {sha}")
    print(f"[freeze] {len(ev)} eval / {len(dev)} dev / {n_decoy} decoys")
    print("[freeze] now: git add v0/tasks && git commit && git tag v0-prereg")
    return record


def _count(rows: list[dict], key: str = "family") -> dict[str, int]:
    out: dict[str, int] = {}
    for r in rows:
        out[r[key]] = out.get(r[key], 0) + 1
    return dict(sorted(out.items()))


def verify(path: Path = EVAL_PATH) -> dict:
    """Check the frozen hash still matches. Every stage that consumes the eval
    set should call this; a task set that changed after the freeze invalidates
    every result derived from it."""
    if not TASKSET_SHA_PATH.exists():
        raise HarnessError("task set is not frozen: tasks/TASKSET.sha256 missing")
    expected = TASKSET_SHA_PATH.read_text(encoding="utf-8").split()[0]
    actual = sha256_file(path)
    if expected != actual:
        raise HarnessError(
            f"FROZEN TASK SET HAS CHANGED.\n"
            f"  expected {expected}\n"
            f"  actual   {actual}\n"
            f"Every run and every grade derived from this set is now unpublishable. "
            f"Restore eval.jsonl from the v0-prereg tag, or start over and say so."
        )
    return {"taskset_sha": actual, "verified_at": utcnow()}
