"""Claim extraction and the C-arm memory builds — §8 and §11.

T4, the Memco failure, is the whole design of this module: *memory must not be
built from the answers*. Three mechanics enforce that, and none of them is a
promise:

1. `extract()` refuses to run unless `tasks/TASKSET.sha256` already exists. The
   task set must be frozen first.
2. This module never imports `tasks.py` and never reads `eval.jsonl`. The
   extractor is given one corpus chunk at a time and nothing else. It is
   structurally unable to see the tasks.
3. `taskset_sha_at_build` is written into every `build.json`. That field is the
   mechanical proof, checkable by a reader, that extraction happened after the
   freeze.

`gold_leak_rate` is measured *afterwards*, by `measure_gold_leak()`, which does
read the tasks. It runs in a separate process step so that the leak check
cannot become a channel by which task text reaches the extractor.
"""

from __future__ import annotations

import json
import re
from pathlib import Path

from . import prompts
from .config import ARM_SOURCE_FILTER, Config
from .corpus import corpus_documents, corpus_sha
from .pricing import build_usd
from .providers import get_provider
from .retrieval import chunk_documents
from .util import (
    V0_ROOT,
    HarnessError,
    count_tokens,
    read_jsonl,
    sha256_text,
    utcnow,
    write_json,
    write_jsonl,
)

MEMORY_DIR = V0_ROOT / "memory"
TASKSET_SHA_PATH = V0_ROOT / "tasks" / "TASKSET.sha256"
CLAIMS_ALL_PATH = MEMORY_DIR / "_all" / "claims.jsonl"

DOC_SOURCES = {"code", "committed_docs"}
_JSON_BLOCK = re.compile(r"\{.*\}", re.DOTALL)


def _read_taskset_sha() -> str:
    """T4 gate. Loud, and deliberately not overridable by a flag."""
    if not TASKSET_SHA_PATH.exists():
        raise HarnessError(
            "refusing to extract claims: tasks/TASKSET.sha256 does not exist.\n"
            "§8 and T4 require the task set to be frozen *before* extraction runs, "
            "so that `taskset_sha_at_build` proves the ordering to a reader. "
            "Run `freeze` first."
        )
    return TASKSET_SHA_PATH.read_text(encoding="utf-8").strip().split()[0]


def _evidence_class(source_type: str) -> str:
    """§8: C_doc is evidence a fresh agent reading the tree can see; C_exp is
    evidence it cannot."""
    return "doc" if source_type in DOC_SOURCES else "exp"


def extract(
    cfg: Config,
    *,
    size: str | None = None,
    model_alias: str | None = None,
    max_chunks: int | None = None,
) -> dict:
    size = size or cfg.primary_size
    taskset_sha = _read_taskset_sha()

    spec = cfg.model(model_alias) if model_alias else cfg.models[0]
    provider = get_provider(spec.provider)
    tmpl_sha = prompts.template_sha("extractor")

    docs = corpus_documents(size)
    chunks = chunk_documents(docs)
    if max_chunks:
        chunks = chunks[:max_chunks]

    claims: list[dict] = []
    in_tok = out_tok = 0
    errors = 0
    import time

    t0 = time.monotonic()

    for i, ch in enumerate(chunks, 1):
        rendered = prompts.render(
            "extractor",
            {
                "REPO": ch.repo,
                "SOURCE_TYPE": ch.source_type,
                "PATH": ch.path,
                "DATE": "",
                "CHUNK_TEXT": ch.text,
            },
        )
        comp = provider.complete(
            model_id=spec.model_id,
            system="You extract durable claims from engineering history. Output JSON only.",
            user=rendered,
            max_tokens=2000,
            temperature=cfg.temperature,
        )
        in_tok += comp.input_tokens + comp.cache_read_tokens
        out_tok += comp.output_tokens
        if comp.error:
            errors += 1
            continue
        m = _JSON_BLOCK.search(comp.text)
        if not m:
            errors += 1
            continue
        try:
            parsed = json.loads(m.group(0))
        except json.JSONDecodeError:
            errors += 1
            continue

        for c in parsed.get("claims", [])[:8]:
            text = (c.get("text") or "").strip()
            if len(text) < 20:
                continue
            claims.append(
                {
                    "claim_id": "clm-" + sha256_text(ch.chunk_id + text)[:12],
                    "text": text,
                    "kind": c.get("kind", "unspecified"),
                    "evidence_class": _evidence_class(ch.source_type),
                    "evidence": [
                        {
                            "doc_id": ch.doc_id,
                            "chunk_id": ch.chunk_id,
                            "source_type": ch.source_type,
                            "repo": ch.repo,
                            "path": ch.path,
                            "url": ch.url,
                            "quote": (c.get("evidence_quote") or "")[:600],
                        }
                    ],
                    "tokens": count_tokens(text),
                }
            )
        if i % 25 == 0:
            print(f"    {i}/{len(chunks)} chunks, {len(claims)} claims")

    wall = time.monotonic() - t0

    # Deduplicate identical claim text across chunks, keeping every evidence
    # pointer. L1 in CLAUDE.md: no claim without evidence — merging must never
    # drop the evidence that justified the duplicate.
    merged: dict[str, dict] = {}
    for c in claims:
        key = c["text"].lower()
        if key in merged:
            merged[key]["evidence"].extend(c["evidence"])
            if merged[key]["evidence_class"] != c["evidence_class"]:
                merged[key]["evidence_class"] = "both"
        else:
            merged[key] = c
    final = sorted(merged.values(), key=lambda c: c["claim_id"])

    write_jsonl(CLAIMS_ALL_PATH, final)

    build = {
        "extracted_at": utcnow(),
        "corpus_size": size,
        "corpus_sha": corpus_sha(size),
        "taskset_sha_at_build": taskset_sha,
        "chunks_processed": len(chunks),
        "chunk_errors": errors,
        "n_claims": len(final),
        "n_evidence": sum(len(c["evidence"]) for c in final),
        "by_evidence_class": _count(final, "evidence_class"),
        "extractor_model": spec.model_id,
        "extractor_prompt_sha": tmpl_sha,
        "build_input_tokens": in_tok,
        "build_output_tokens": out_tok,
        "build_usd": build_usd(spec.model_id, input_tokens=in_tok, output_tokens=out_tok),
        "build_wall_clock_s": round(wall, 1),
    }
    write_json(MEMORY_DIR / "_all" / "build.json", build)
    print(f"[extract] {len(final)} claims -> {CLAIMS_ALL_PATH}")
    return build


def _count(rows: list[dict], key: str) -> dict[str, int]:
    out: dict[str, int] = {}
    for r in rows:
        out[r[key]] = out.get(r[key], 0) + 1
    return dict(sorted(out.items()))


def partition(cfg: Config) -> dict[str, dict]:
    """Write per-arm claim sets and build records.

    §8: the partition is by evidence source, decided at extraction time — a
    claim's `evidence_class` was fixed when it was extracted, from the source
    type of the chunk it came from. This function only selects.
    """
    all_claims = read_jsonl(CLAIMS_ALL_PATH)
    if not all_claims:
        raise HarnessError(f"no claims at {CLAIMS_ALL_PATH}: run `extract` first")
    base = json.loads((MEMORY_DIR / "_all" / "build.json").read_text(encoding="utf-8"))

    builds: dict[str, dict] = {}
    for arm in ("C_both", "C_doc", "C_exp"):
        if not cfg.arm_enabled(arm):
            continue
        want = ARM_SOURCE_FILTER[arm]
        selected = [
            c
            for c in all_claims
            if any(e["source_type"] in want for e in c["evidence"])
        ]
        out_dir = MEMORY_DIR / arm
        write_jsonl(out_dir / "claims.jsonl", selected)
        share = (len(selected) / len(all_claims)) if all_claims else 0.0
        b = {
            "build_id": f"bld-{arm}-{base['corpus_sha'][:8]}",
            "arm": arm,
            "corpus_sha": base["corpus_sha"],
            "source_filter": want,
            "n_claims": len(selected),
            "n_evidence": sum(len(c["evidence"]) for c in selected),
            "build_input_tokens": int(base["build_input_tokens"] * share),
            "build_output_tokens": int(base["build_output_tokens"] * share),
            "build_usd": round(base["build_usd"] * share, 6),
            "build_wall_clock_s": round(base["build_wall_clock_s"] * share, 1),
            "extractor_model": base["extractor_model"],
            "extractor_prompt_sha": base["extractor_prompt_sha"],
            "taskset_sha_at_build": base["taskset_sha_at_build"],
            "gold_leak_rate": None,
            "_note": (
                "Extraction runs once over the whole corpus; per-arm build cost is "
                "the extraction cost apportioned by share of claims retained. The "
                "arms are partitions of one extraction pass, so no arm pays for a "
                "pass the others do not."
            ),
        }
        write_json(out_dir / "build.json", b)
        builds[arm] = b
        print(f"[partition] {arm}: {len(selected)} claims")
    return builds


# ---------------------------------------------------------------------------
# gold leak — measured after the fact, in a separate step (T4)
# ---------------------------------------------------------------------------


def measure_gold_leak(cfg: Config, *, n: int = 8, threshold: float = 0.5) -> dict:
    """Verbatim overlap between gold answers and the extracted claim set.

    This is the direct guard against building the memory from the answers. It
    reports, per arm, the share of eval tasks whose gold answer overlaps some
    claim by more than `threshold` of its n-grams.

    A non-zero rate is not automatically fatal — a claim and a gold answer can
    legitimately share wording because both describe the same maintainer
    sentence — but it is published per build so a reader can judge.
    """
    from .tasks import load_eval  # imported here, never at module scope (T4)

    tasks = [t for t in load_eval() if not t["gold"]["abstain_expected"]]
    word = re.compile(r"[A-Za-z0-9_]{3,}")

    def grams(text: str) -> set[tuple[str, ...]]:
        toks = [w.lower() for w in word.findall(text)]
        return {tuple(toks[i : i + n]) for i in range(max(0, len(toks) - n + 1))}

    out: dict[str, dict] = {}
    for arm in ("C_both", "C_doc", "C_exp"):
        arm_dir = MEMORY_DIR / arm
        claims_path = arm_dir / "claims.jsonl"
        if not claims_path.exists():
            continue
        claims = read_jsonl(claims_path)
        claim_grams: set[tuple[str, ...]] = set()
        for c in claims:
            claim_grams |= grams(c["text"])

        leaked: list[str] = []
        for t in tasks:
            g = grams(t["gold"]["answer"])
            if not g:
                continue
            if len(g & claim_grams) / len(g) >= threshold:
                leaked.append(t["task_id"])
        rate = len(leaked) / len(tasks) if tasks else 0.0

        bpath = arm_dir / "build.json"
        b = json.loads(bpath.read_text(encoding="utf-8"))
        b["gold_leak_rate"] = round(rate, 4)
        b["gold_leak_detail"] = {
            "ngram": n,
            "overlap_threshold": threshold,
            "tasks_checked": len(tasks),
            "tasks_leaked": len(leaked),
            "leaked_task_ids": leaked,
            "limitation": "Semantic leakage below the verbatim threshold is not detectable by this test (T4 residual).",
        }
        write_json(bpath, b)
        out[arm] = {"gold_leak_rate": rate, "tasks_leaked": len(leaked)}
        print(f"[gold-leak] {arm}: {rate:.4f} ({len(leaked)}/{len(tasks)})")
    return out


def load_claims(arm: str) -> list[dict]:
    p = MEMORY_DIR / arm / "claims.jsonl"
    if not p.exists():
        raise HarnessError(f"no claims for arm {arm}: run `extract` then `partition`")
    return read_jsonl(p)


def build_record(arm: str) -> dict:
    p = MEMORY_DIR / arm / "build.json"
    if not p.exists():
        raise HarnessError(f"no build record for arm {arm}")
    return json.loads(p.read_text(encoding="utf-8"))
