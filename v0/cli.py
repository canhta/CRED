#!/usr/bin/env python3
"""CRED v0 harness CLI.

    python3 -m v0.cli <stage> [options]

Stages run in the order printed by `python3 -m v0.cli status`. Each is
independently re-runnable; the expensive ones (mine, corpus-fetch, run) are
resumable and idempotent.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from v0.harness import (a0_filter, audit, corpus, draft, freeze, judge, kappa, memory,
                        mine, retrieval, runner, tasks)
from v0.harness.config import load_config
from v0.harness.prompts import all_template_hashes
from v0.harness.providers.anthropic_provider import credentials_present
from v0.harness.util import HarnessError, read_jsonl, write_json

STAGES = [
    ("mine", "Mine backward-reference anchors from public repos (§4 steps 1-2). gh required."),
    ("corpus-fetch", "Pull pre-cutoff documents for corpus C (§2 D1). gh required."),
    ("corpus-build", "Assemble S/M/L corpora + MANIFEST.json (§3)."),
    ("index", "Chunk the corpus and build the lexical index (arm B, C assembly)."),
    ("draft", "Anchor -> candidate task, one anchor at a time (§4 step 3). MODEL."),
    ("verify-decoys", "Reject decoys whose terms appear in the corpus (§4, T10)."),
    ("verbatim-filter", "Delete tasks whose gold answer is verbatim in the tree at T (§4 step 5)."),
    ("a0-filter", "Run A0 on every candidate; delete what it solves (§4 step 4, T6). MODEL."),
    ("audit-sample", "Draw a blind 20% sample for a non-founder reviewer (§4 step 7)."),
    ("audit-score", "Score the returned audit; >15% flag rate voids the set (§4 step 7)."),
    ("freeze", "Compose eval/dev, hash eval.jsonl, write TASKSET.sha256 (§4 step 8, D4)."),
    ("extract", "Extract claims from the corpus. Refuses to run before the freeze (T4). MODEL."),
    ("partition", "Split claims into C_both / C_doc / C_exp and write build.json (§8)."),
    ("gold-leak", "Measure verbatim gold-answer overlap with the claim set (T4)."),
    ("run", "Execute the arm x task x run grid (§11). MODEL."),
    ("grade", "Grade runs: string matching first, judge only where needed (§5). MODEL."),
    ("kappa-sample", "Draw a stratified 60-run blind sample for hand-labelling (§5)."),
    ("kappa", "Compute Cohen's kappa judge-vs-human and enforce the 0.70 gate (§5)."),
    ("analyze", "Primary metric, CIs, variance, cost per correct, verdict (§5-§7)."),
]


def cmd_status(args) -> int:
    cfg = load_config(args.config)
    print(f"config: {cfg.name}  arms={cfg.arms}  n_eval={cfg.n_eval_tasks} "
          f"runs/task={cfg.runs_per_task}  may_greenlight={cfg.may_greenlight}")
    print(f"credentials detected: {credentials_present()}")
    print(f"arm B runnable: {retrieval.arm_b_available()}")
    print("\nprompt template hashes (T9):")
    for name, sha in all_template_hashes().items():
        print(f"  {name:16s} {sha}")
    print("\nartefacts:")
    for label, p in [
        ("anchors", mine.ANCHORS_PATH),
        ("corpus manifest", corpus.MANIFEST_PATH),
        ("candidates", tasks.CANDIDATES_PATH),
        ("eval.jsonl", tasks.EVAL_PATH),
        ("TASKSET.sha256", tasks.TASKSET_SHA_PATH),
        ("rejected.jsonl", tasks.REJECTED_PATH),
        ("claims (all)", memory.CLAIMS_ALL_PATH),
        ("grades", judge.GRADES_PATH),
        ("kappa", kappa.KAPPA_PATH),
    ]:
        n = ""
        if p.exists() and p.suffix == ".jsonl":
            n = f" ({len(read_jsonl(p))} rows)"
        print(f"  {'OK ' if p.exists() else '-- '} {label:16s} {p}{n}")
    print("\nstages, in order:")
    for name, desc in STAGES:
        print(f"  {name:16s} {desc}")
    return 0


def cmd_mine(args) -> int:
    specs = [s for s in mine.REPOS if s.repo in args.repo] if args.repo else mine.REPOS
    mine.mine(specs, prs_per_repo=args.prs_per_repo)
    return 0


def cmd_corpus_fetch(args) -> int:
    corpus.fetch(pr_cap=args.pr_cap)
    return 0


def cmd_corpus_build(args) -> int:
    corpus.build(args.size or None)
    return 0


def cmd_index(args) -> int:
    cfg = load_config(args.config)
    for size in (args.size or [cfg.primary_size]):
        retrieval.build_index(corpus.corpus_documents(size), size)
    return 0


def cmd_draft(args) -> int:
    cfg = load_config(args.config)
    draft.draft(cfg, model_alias=args.model, max_anchors=args.max_anchors)
    return 0


def cmd_verify_decoys(args) -> int:
    cfg = load_config(args.config)
    stats = draft.verify_decoys(args.size_one or cfg.primary_size)
    write_json(tasks.TASKS_DIR / "DECOY_VERIFICATION.json", stats)
    return 0


def cmd_verbatim_filter(args) -> int:
    """§4 step 5. The 'tree at T' is the committed-docs + code portion of the
    corpus — the part a fresh agent reading the repository would see."""
    cfg = load_config(args.config)
    size = args.size_one or cfg.primary_size
    docs = corpus.corpus_documents(size)
    tree_text = "\n".join(d["text"] for d in docs if d["source_type"] in memory.DOC_SOURCES)
    cands = tasks.load_candidates()
    kept, dropped = [], 0
    for t in cands:
        if t["gold"]["abstain_expected"]:
            t["filters"]["verbatim_in_tree_at_t"] = False
            kept.append(t)
            continue
        hit = tasks.verbatim_in_tree(t["gold"]["answer"], tree_text)
        t["filters"]["verbatim_in_tree_at_t"] = hit
        if hit:
            tasks.reject(t, "gold answer appears verbatim in the working tree at T (§4 step 5)",
                         stage="verbatim_filter")
            dropped += 1
            continue
        kept.append(t)
    from v0.harness.util import write_jsonl

    write_jsonl(tasks.CANDIDATES_PATH, kept)
    report = {"checked_against_size": size, "tree_chars": len(tree_text),
              "candidates_in": len(cands), "dropped": dropped, "candidates_out": len(kept),
              "rule": "6-gram overlap >= 0.5 between gold answer and the tree text"}
    write_json(tasks.TASKS_DIR / "VERBATIM_FILTER.json", report)
    print(f"[verbatim-filter] dropped {dropped}, kept {len(kept)}")
    return 0


def cmd_a0_filter(args) -> int:
    cfg = load_config(args.config)
    a0_filter.run_filter(cfg, model_alias=args.model)
    return 0


def cmd_audit_sample(args) -> int:
    audit.draw_sample()
    return 0


def cmd_audit_score(args) -> int:
    cfg = load_config(args.config)
    audit.score(threshold=cfg.thresholds.audit_flag_rate_max)
    return 0


def cmd_freeze(args) -> int:
    cfg = load_config(args.config)
    freeze.freeze(cfg, allow_mock=args.allow_mock, force=args.force)
    return 0


def cmd_extract(args) -> int:
    cfg = load_config(args.config)
    memory.extract(cfg, size=args.size_one, model_alias=args.model, max_chunks=args.max_chunks)
    return 0


def cmd_partition(args) -> int:
    memory.partition(load_config(args.config))
    return 0


def cmd_gold_leak(args) -> int:
    memory.measure_gold_leak(load_config(args.config))
    return 0


def cmd_run(args) -> int:
    cfg = load_config(args.config)
    freeze.verify()
    ts = tasks.load_dev() if args.dev else tasks.load_eval()
    runner.run_grid(
        cfg, ts,
        arms=args.arm or None,
        sizes=args.size or None,
        model_aliases=args.model_alias or None,
        limit=args.limit,
        dry_run=args.dry_run,
    )
    return 0


def cmd_grade(args) -> int:
    cfg = load_config(args.config)
    judge.grade_all(runner.load_runs(), tasks.load_eval(), cfg,
                    grader_alias=args.model, grader_label=args.grader)
    return 0


def cmd_kappa_sample(args) -> int:
    kappa.sample_for_human(judge.load_grades(), runner.load_runs(), n=args.n)
    return 0


def cmd_kappa(args) -> int:
    cfg = load_config(args.config)
    r = kappa.compute(threshold=cfg.thresholds.kappa_min)
    return 0 if r["passes"] else 1


def cmd_analyze(args) -> int:
    from v0.analysis.analyze import analyze

    cfg = load_config(args.config)
    analyze(cfg, model_alias=args.model, size=args.size_one,
            require_kappa=not args.skip_kappa_gate)
    return 0


HANDLERS = {
    "status": cmd_status,
    "mine": cmd_mine,
    "corpus-fetch": cmd_corpus_fetch,
    "corpus-build": cmd_corpus_build,
    "index": cmd_index,
    "draft": cmd_draft,
    "verify-decoys": cmd_verify_decoys,
    "verbatim-filter": cmd_verbatim_filter,
    "a0-filter": cmd_a0_filter,
    "audit-sample": cmd_audit_sample,
    "audit-score": cmd_audit_score,
    "freeze": cmd_freeze,
    "extract": cmd_extract,
    "partition": cmd_partition,
    "gold-leak": cmd_gold_leak,
    "run": cmd_run,
    "grade": cmd_grade,
    "kappa-sample": cmd_kappa_sample,
    "kappa": cmd_kappa,
    "analyze": cmd_analyze,
}


def main() -> int:
    ap = argparse.ArgumentParser(prog="v0.cli", description=__doc__)
    ap.add_argument("stage", choices=sorted(HANDLERS))
    ap.add_argument("--config", default="mve.json")
    ap.add_argument("--model", default=None, help="model alias for a model-calling stage")
    ap.add_argument("--model-alias", action="append", help="restrict `run` to these model aliases")
    ap.add_argument("--arm", action="append")
    ap.add_argument("--size", action="append", help="corpus sizes (repeatable)")
    ap.add_argument("--size-one", default=None, help="a single corpus size")
    ap.add_argument("--repo", action="append")
    ap.add_argument("--prs-per-repo", type=int, default=250)
    ap.add_argument("--pr-cap", type=int, default=200)
    ap.add_argument("--max-anchors", type=int, default=None)
    ap.add_argument("--max-chunks", type=int, default=None)
    ap.add_argument("--limit", type=int, default=None)
    ap.add_argument("--n", type=int, default=60)
    ap.add_argument("--grader", default="judge-A")
    ap.add_argument("--dev", action="store_true", help="run against the dev split")
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--allow-mock", action="store_true")
    ap.add_argument("--force", action="store_true")
    ap.add_argument("--skip-kappa-gate", action="store_true")
    args = ap.parse_args()

    try:
        return HANDLERS[args.stage](args)
    except HarnessError as exc:
        print(f"\nHALTED: {exc}\n", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
