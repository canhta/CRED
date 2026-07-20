"""The run loop — §11 run records.

One line per (arm x task x run). Resumable: a run whose `run_id` is already in
`runs.jsonl` is skipped, so a grid interrupted by a rate limit at run 400 of 540
resumes without re-spending the first 400.

Every field §11 specifies is populated here, including the ones that are only
useful if something goes wrong: `finish_reason`, `error`, raw token counts (T8),
and `provider_model_version` (T3).
"""

from __future__ import annotations

import time
from pathlib import Path

from . import prompts
from .arms import ArmAssembler
from .config import Config, ModelSpec, config_sha
from .corpus import corpus_sha
from .memory import build_record
from .pricing import amortized_usd, marginal_usd
from .util import (
    HARNESS_VERSION,
    V0_ROOT,
    HarnessError,
    read_jsonl,
    sha256_text,
    utcnow,
    append_jsonl,
)

RUNS_ROOT = V0_ROOT / "runs"
MAX_OUTPUT_TOKENS = 700  # answers are capped at 120 words by every template


def runs_path(model: ModelSpec, arm: str, size: str, root: Path = RUNS_ROOT) -> Path:
    return root / model.alias / arm / size / "runs.jsonl"


def run_id_for(model: ModelSpec, arm: str, size: str, task_id: str, run_index: int) -> str:
    """Deterministic, so resume works and so a re-run of the same cell is
    detectable rather than silently duplicated."""
    return "run-" + sha256_text(f"{model.alias}|{arm}|{size}|{task_id}|{run_index}")[:20]


def run_grid(
    cfg: Config,
    tasks: list[dict],
    *,
    arms: list[str] | None = None,
    sizes: list[str] | None = None,
    model_aliases: list[str] | None = None,
    root: Path = RUNS_ROOT,
    resume: bool = True,
    limit: int | None = None,
    dry_run: bool = False,
) -> dict:
    from .providers import get_provider

    arms = arms or cfg.arms
    sizes = sizes or [cfg.primary_size]
    models = [cfg.model(a) for a in model_aliases] if model_aliases else cfg.models

    taskset_sha = sha256_text("".join(sorted(t["task_id"] for t in tasks)))
    cfg_sha = config_sha(cfg)

    stats = {"planned": 0, "executed": 0, "skipped": 0, "errors": 0, "truncated": 0}

    for model in models:
        provider = get_provider(model.provider)
        for size in sizes:
            csha = corpus_sha(size) if arms != ["A0"] else ""
            assembler = ArmAssembler(cfg, size)
            for arm in arms:
                out = runs_path(model, arm, size, root)
                existing = {r["run_id"] for r in read_jsonl(out)} if resume else set()
                tmpl = prompts.arm_template_name(arm)
                tmpl_sha = prompts.template_sha(tmpl)
                memory_build_id = None
                build_total_usd = 0.0
                if arm.startswith("C_"):
                    br = build_record(arm)
                    memory_build_id = br["build_id"]
                    build_total_usd = br["build_usd"]

                print(f"[run] {model.alias} / {arm} / {size}: {len(tasks)} tasks x {cfg.runs_per_task}")
                for task in tasks:
                    for idx in range(cfg.runs_per_task):
                        stats["planned"] += 1
                        rid = run_id_for(model, arm, size, task["task_id"], idx)
                        if rid in existing:
                            stats["skipped"] += 1
                            continue
                        if limit is not None and stats["executed"] >= limit:
                            continue

                        t_retr0 = time.monotonic()
                        block = assembler.assemble(arm, task["question"], model)
                        latency_retrieval = int((time.monotonic() - t_retr0) * 1000)

                        values = {
                            "REPO": task["repo"],
                            "CUTOFF_T": task["cutoff_t"],
                            "QUESTION": task["question"],
                        }
                        if arm != "A0":
                            values["MEMORY_BLOCK"] = block.text
                        system = prompts.render(tmpl, values)
                        # The template already carries the question; the user
                        # turn restates it so that the cacheable system prefix
                        # (arm A's corpus) is identical across every query.
                        user = task["question"]

                        if dry_run:
                            stats["executed"] += 1
                            continue

                        comp = provider.complete(
                            model_id=model.model_id,
                            system=system,
                            user=user,
                            max_tokens=MAX_OUTPUT_TOKENS,
                            temperature=cfg.temperature,
                            seed=cfg.seeds[idx],
                            cache_system=(arm == "A" and model.caching),
                        )

                        marginal = marginal_usd(
                            model.model_id,
                            input_tokens=comp.input_tokens,
                            output_tokens=comp.output_tokens,
                            cache_write_tokens=comp.cache_write_tokens,
                            cache_read_tokens=comp.cache_read_tokens,
                        )
                        record = {
                            "run_id": rid,
                            "task_id": task["task_id"],
                            "arm": arm,
                            "model": model.model_id,
                            "model_alias": model.alias,
                            "provider": model.provider,
                            "provider_model_version": comp.provider_model_version,
                            "corpus_size": size,
                            "run_index": idx,
                            "seed": cfg.seeds[idx],
                            # null when the provider rejects the parameter; see
                            # providers/anthropic_provider.py for why.
                            "temperature": cfg.temperature if model.provider == "mock" else None,
                            "taskset_sha": taskset_sha,
                            "config_sha": cfg_sha,
                            "corpus_sha": csha,
                            "prompt_template": tmpl,
                            "prompt_template_sha": tmpl_sha,
                            "memory_build_id": memory_build_id,
                            "context_block_sha": block.sha,
                            "context_tokens": block.tokens,
                            "context_truncated_for_model": block.truncated_for_model,
                            "retrieved_ids": block.retrieved_ids,
                            "input_tokens": comp.input_tokens,
                            "cached_input_tokens": comp.cache_read_tokens,
                            "cache_write_tokens": comp.cache_write_tokens,
                            "output_tokens": comp.output_tokens,
                            "usd_marginal": round(marginal, 8),
                            "usd_amortized_n200": round(
                                amortized_usd(marginal, build_total_usd, cfg.amortization_n), 8
                            ),
                            "latency_ms_retrieval": latency_retrieval,
                            "latency_ms_generation": comp.latency_ms,
                            "latency_ms_total": latency_retrieval + comp.latency_ms,
                            "answer_text": comp.text,
                            "finish_reason": comp.finish_reason,
                            "error": comp.error,
                            "harness_version": HARNESS_VERSION,
                            "started_at": utcnow(),
                        }
                        append_jsonl(out, record)
                        stats["executed"] += 1
                        if comp.error:
                            stats["errors"] += 1
                        if comp.truncated:
                            stats["truncated"] += 1
                        if stats["executed"] % 25 == 0:
                            print(f"    {stats['executed']} runs executed")
    print(f"[run] {stats}")
    return stats


def load_runs(root: Path = RUNS_ROOT, *, include_filter_runs: bool = False) -> list[dict]:
    """Load the eval grid.

    A0-filter runs live under `runs/_a0_filter/` and are excluded by default.
    They are runs against *candidate* tasks, most of which were then deleted;
    letting them into the analysis would mix pre-freeze filtering evidence into
    the post-freeze measurement.
    """
    rows: list[dict] = []
    for p in sorted(root.rglob("runs.jsonl")):
        if not include_filter_runs and any(part.startswith("_") for part in p.parts):
            continue
        rows.extend(read_jsonl(p))
    if not rows:
        raise HarnessError(f"no runs found under {root}")
    return rows
