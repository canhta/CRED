"""Anchor -> candidate task, §4 step 3.

"Draft question + gold answer from each anchor with an LLM, given the anchor
only. The founder does not write questions."

Two structural guarantees, both enforced here rather than promised:

1. The drafter sees **one anchor and nothing else**. It never sees the corpus,
   never sees other tasks, and never sees the experiment's hypothesis. The
   prompt is `prompts/drafter.md`; that file is hashed and published.
2. Decoys (§4, 15%) are drafted by a separate prompt and then *verified* by a
   lexical search over the built corpus. A decoy whose distinctive terms appear
   in the record is not a decoy — it is rejected, with its reason, to
   `tasks/rejected.jsonl`.

The verification in (2) is a presence test, not a proof of absence. A question
whose answer is present in the record but phrased with none of the terms the
drafter listed will survive it. That residual is recorded in the writeup.
"""

from __future__ import annotations

import json
import random
import re
from pathlib import Path

from . import prompts
from .config import Config
from .providers import get_provider
from .tasks import CANDIDATES_PATH, make_task, reject
from .util import (
    V0_ROOT,
    HarnessError,
    read_jsonl,
    sha256_text,
    utcnow,
    write_json,
    write_jsonl,
)

ANCHORS_PATH = V0_ROOT / "mining" / "anchors.jsonl"
DRAFT_STATS_PATH = V0_ROOT / "mining" / "DRAFT.json"

_JSON_BLOCK = re.compile(r"\{.*\}", re.DOTALL)


def _parse_json(text: str) -> dict:
    m = _JSON_BLOCK.search(text)
    if not m:
        raise ValueError("no JSON object in drafter output")
    return json.loads(m.group(0))


def _values(anchor: dict) -> dict[str, str]:
    return {
        "REPO": anchor["repo"],
        "ANCHOR_DATE": anchor["anchor_date"],
        "ANCHOR_AUTHOR": anchor["author"],
        "PR_NUMBER": str(anchor["pr_number"]),
        "PR_TITLE": anchor["pr_title"],
        "ANCHOR_TEXT": anchor["text"],
    }


def _task_id(anchor: dict, kind: str) -> str:
    return f"kn-{kind}-{sha256_text(anchor['anchor_id'] + kind)[:8]}"


def draft(
    cfg: Config,
    *,
    model_alias: str | None = None,
    max_anchors: int | None = None,
    decoy_fraction: float | None = None,
    seed: int = 20260720,
    out_path: Path = CANDIDATES_PATH,
) -> dict:
    anchors = read_jsonl(ANCHORS_PATH)
    if not anchors:
        raise HarnessError(f"no anchors at {ANCHORS_PATH}: run `mine` first")

    spec = cfg.model(model_alias) if model_alias else cfg.models[0]
    provider = get_provider(spec.provider)
    decoy_fraction = cfg.decoy_share if decoy_fraction is None else decoy_fraction

    rng = random.Random(seed)
    anchors = sorted(anchors, key=lambda a: a["anchor_id"])
    rng.shuffle(anchors)
    if max_anchors:
        anchors = anchors[:max_anchors]

    # Which anchors become decoys is decided by seeded shuffle before any model
    # is called, so the split cannot be steered by what the drafter produced.
    n_decoy = int(round(len(anchors) * decoy_fraction))
    decoy_ids = {a["anchor_id"] for a in anchors[:n_decoy]}

    tmpl_sha = {
        "drafter": prompts.template_sha("drafter"),
        "drafter_decoy": prompts.template_sha("drafter_decoy"),
    }

    out: list[dict] = []
    stats = {"attempted": 0, "usable": 0, "unusable": 0, "parse_error": 0, "api_error": 0,
             "decoy_attempted": 0, "decoy_usable": 0}
    in_tok = out_tok = 0

    for anchor in anchors:
        is_decoy = anchor["anchor_id"] in decoy_ids
        name = "drafter_decoy" if is_decoy else "drafter"
        stats["attempted"] += 1
        if is_decoy:
            stats["decoy_attempted"] += 1

        rendered = prompts.render(name, _values(anchor))
        comp = provider.complete(
            model_id=spec.model_id,
            system="You draft evaluation tasks. Output JSON only.",
            user=rendered,
            max_tokens=2000,
            temperature=cfg.temperature,
            seed=seed,
        )
        in_tok += comp.input_tokens + comp.cache_read_tokens
        out_tok += comp.output_tokens
        if comp.error:
            stats["api_error"] += 1
            print(f"    !! {anchor['anchor_id']}: {comp.error}")
            continue
        try:
            parsed = _parse_json(comp.text)
        except (ValueError, json.JSONDecodeError) as exc:
            stats["parse_error"] += 1
            print(f"    ?? {anchor['anchor_id']}: unparseable drafter output ({exc})")
            continue

        if not parsed.get("usable"):
            stats["unusable"] += 1
            continue

        try:
            if is_decoy:
                task = make_task(
                    task_id=_task_id(anchor, "d"),
                    family="abstention_decoy",
                    repo=anchor["repo"],
                    cutoff_t=anchor["cutoff_t"],
                    question=parsed["question"],
                    gold_answer="The corpus does not say.",
                    checkpoints=[],
                    abstain_expected=True,
                    anchor_url=anchor["anchor_url"],
                    anchor_sha=anchor["anchor_sha"],
                    anchor_date=anchor["anchor_date"],
                    evidence_spans=[],
                    author=f"miner-v1+drafter({spec.model_id})",
                )
                task["decoy"] = {
                    "why_unrecorded": parsed.get("why_unrecorded", ""),
                    "search_terms": parsed.get("search_terms", []),
                    "verified_absent": None,
                }
                stats["decoy_usable"] += 1
            else:
                task = make_task(
                    task_id=_task_id(anchor, "t"),
                    family=parsed["family"],
                    repo=anchor["repo"],
                    cutoff_t=anchor["cutoff_t"],
                    question=parsed["question"],
                    gold_answer=parsed["gold_answer"],
                    checkpoints=parsed["checkpoints"],
                    abstain_expected=False,
                    anchor_url=anchor["anchor_url"],
                    anchor_sha=anchor["anchor_sha"],
                    anchor_date=anchor["anchor_date"],
                    evidence_spans=[
                        {"path": anchor["anchor_url"], "quote": parsed.get("evidence_quote", "")}
                    ],
                    author=f"miner-v1+drafter({spec.model_id})",
                )
        except (HarnessError, KeyError) as exc:
            stats["parse_error"] += 1
            print(f"    ?? {anchor['anchor_id']}: malformed task ({exc})")
            continue

        task["drafter"] = {
            "model": spec.model_id,
            "provider_model_version": comp.provider_model_version,
            "prompt_template_sha": tmpl_sha[name],
            "anchor_id": anchor["anchor_id"],
            "drafted_at": utcnow(),
        }
        out.append(task)
        stats["usable"] += 1

    write_jsonl(out_path, out)
    from .pricing import build_usd

    summary = {
        "drafted_at": utcnow(),
        "anchors_considered": len(anchors),
        "decoy_fraction_requested": decoy_fraction,
        "drafter_model": spec.model_id,
        "prompt_template_sha": tmpl_sha,
        "stats": stats,
        "input_tokens": in_tok,
        "output_tokens": out_tok,
        "usd": build_usd(spec.model_id, input_tokens=in_tok, output_tokens=out_tok),
        "candidates_path": str(out_path),
    }
    write_json(DRAFT_STATS_PATH, summary)
    print(f"[draft] {stats['usable']} candidate tasks -> {out_path}")
    return summary


# ---------------------------------------------------------------------------
# Decoy verification (§4, T10)
# ---------------------------------------------------------------------------

_TERM = re.compile(r"[A-Za-z0-9_.\-]{3,}")


def verify_decoys(size: str = "M", *, min_terms_present: int = 2, path: Path = CANDIDATES_PATH) -> dict:
    """Reject any decoy whose distinctive terms are present in the corpus.

    A decoy survives only if fewer than `min_terms_present` of its listed search
    terms appear in the record. Two rather than one because a single common
    token ("timeout", "retry") appearing somewhere is weak evidence that the
    *answer* is there.
    """
    from .corpus import corpus_text

    text = corpus_text(size).lower()
    tasks = read_jsonl(path)
    kept: list[dict] = []
    rejected = 0
    for t in tasks:
        if t["family"] != "abstention_decoy":
            kept.append(t)
            continue
        terms = [s.lower().strip() for s in t.get("decoy", {}).get("search_terms", []) if s.strip()]
        present = [s for s in terms if s and s in text]
        t.setdefault("decoy", {})
        t["decoy"]["terms_present_in_corpus"] = present
        t["decoy"]["corpus_size_checked"] = size
        if len(present) >= min_terms_present or not terms:
            t["decoy"]["verified_absent"] = False
            reason = (
                f"decoy rejected: {len(present)} of {len(terms)} distinctive terms present "
                f"in corpus {size} ({present[:5]})"
                if terms
                else "decoy rejected: drafter listed no search terms, so absence cannot be checked"
            )
            reject(t, reason, stage="verify_decoys")
            rejected += 1
            continue
        t["decoy"]["verified_absent"] = True
        kept.append(t)

    write_jsonl(path, kept)
    stats = {
        "checked_against_corpus": size,
        "min_terms_present_to_reject": min_terms_present,
        "decoys_rejected": rejected,
        "decoys_kept": sum(1 for t in kept if t["family"] == "abstention_decoy"),
        "tasks_remaining": len(kept),
        "limitation": (
            "This is a presence test over the corpus text, not a proof of "
            "absence. A decoy whose answer is recorded in wording the drafter "
            "did not anticipate will survive it."
        ),
    }
    print(f"[verify-decoys] rejected {rejected}, kept {stats['decoys_kept']}")
    return stats
