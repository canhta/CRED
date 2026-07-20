"""Corpus construction, §2 D1 and §3.

D1: all arms consume one identical corpus C. Only the access path differs. This
module is therefore the single source of truth for what "the record" contains,
and every arm's memory block is derived from its output — arm A gets the
concatenation, arm B indexes it, arm C extracts claims from it.

Two stages, deliberately separate:

  fetch  — pull raw documents authored strictly before the cutoff T. Slow,
           network-bound, cached to disk, idempotent.
  build  — assemble the S / M / L corpora from the cached documents in a fixed
           deterministic order, and write MANIFEST.json with per-size sha and
           token counts.

Nothing authored on or after T enters any corpus. That rule is enforced here,
in one place, by `_before_cutoff`.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path

from .mine import REPOS, RepoSpec, _gh, _is_bot, check_gh
from .util import (
    V0_ROOT,
    HarnessError,
    count_tokens,
    read_jsonl,
    sha256_text,
    token_estimator_name,
    truncate_to_tokens,
    utcnow,
    write_json,
    write_jsonl,
)

CORPUS_DIR = V0_ROOT / "corpus"
RAW_DIR = CORPUS_DIR / "_raw"
MANIFEST_PATH = CORPUS_DIR / "MANIFEST.json"

# §3: corpus sizes. Token budgets, measured with the estimator named in the
# manifest.
SIZE_BUDGETS = {"S": 15_000, "M": 120_000, "L": 800_000}

# §8: claims are partitioned by evidence source at extraction time. The
# partition starts here, on the document.
#   C_doc  <- code, committed_docs      (things a fresh agent reading the tree sees)
#   C_exp  <- review_discussion, issue_thread, revert, incident  (things it cannot)
SOURCE_TYPES = [
    "committed_docs",
    "code",
    "review_discussion",
    "issue_thread",
    "revert",
    "incident",
]

# Deterministic assembly order (§3: "concatenated in a fixed deterministic
# order"). Decision records first because they are the densest, then code, then
# discussion. Within a source type, by date then id.
_ORDER_RANK = {t: i for i, t in enumerate(SOURCE_TYPES)}

MAX_DOC_CHARS = 60_000


@dataclass(frozen=True)
class Doc:
    doc_id: str
    repo: str
    source_type: str
    path: str
    url: str
    date: str
    text: str

    def to_json(self) -> dict:
        return {
            "doc_id": self.doc_id,
            "repo": self.repo,
            "source_type": self.source_type,
            "path": self.path,
            "url": self.url,
            "date": self.date,
            "text": self.text,
            "text_sha256": sha256_text(self.text),
            "tokens": count_tokens(self.text),
        }


def _before_cutoff(date: str, spec: RepoSpec) -> bool:
    """The single enforcement point for D1. Anything without a date is dropped
    rather than assumed old — an undated document could be post-T, and a corpus
    that leaks the answer invalidates every arm at once."""
    return bool(date) and date < spec.cutoff_t


def _doc_id(repo: str, kind: str, key: str) -> str:
    return f"doc-{sha256_text(f'{repo}|{kind}|{key}')[:14]}"


# ---------------------------------------------------------------------------
# fetch
# ---------------------------------------------------------------------------


def _commit_at_cutoff(spec: RepoSpec) -> str:
    data = _gh([
        "-X", "GET", f"repos/{spec.repo}/commits",
        "-f", f"until={spec.cutoff_t}", "-f", "per_page=1",
    ])
    if not data:
        raise HarnessError(f"no commit found before {spec.cutoff_t} in {spec.repo}")
    return data[0]["sha"]


def _fetch_tree_docs(spec: RepoSpec, ref: str) -> list[Doc]:
    """Committed documentation at T: the decision directory plus top-level
    docs. This is the C_doc evidence class."""
    docs: list[Doc] = []
    for directory, source_type in ((spec.decision_dir, "committed_docs"), ("docs", "committed_docs")):
        try:
            # `-X GET` is load-bearing: `gh api` switches to POST as soon as a
            # `-f` field is present, and a POST to the contents endpoint fails
            # in a way that looks like "this directory does not exist".
            listing = _gh(["-X", "GET", f"repos/{spec.repo}/contents/{directory}", "-f", f"ref={ref}"])
        except HarnessError:
            continue
        if not isinstance(listing, list):
            continue
        for entry in listing:
            if entry.get("type") != "file":
                continue
            name = entry.get("name", "")
            if not name.endswith((".md", ".rst", ".txt")):
                continue
            try:
                blob = _gh(["-X", "GET", f"repos/{spec.repo}/contents/{entry['path']}",
                            "-f", f"ref={ref}"])
            except HarnessError:
                continue
            import base64

            try:
                text = base64.b64decode(blob.get("content", "")).decode("utf-8", "replace")
            except Exception:
                continue
            if len(text.strip()) < 200:
                continue
            docs.append(
                Doc(
                    doc_id=_doc_id(spec.repo, source_type, entry["path"]),
                    repo=spec.repo,
                    source_type=source_type,
                    path=entry["path"],
                    url=entry.get("html_url", ""),
                    # Tree files are dated by the cutoff commit; they exist at T
                    # by construction, so they are pre-T by definition.
                    date=spec.cutoff_t[:10] + "T00:00:00Z",
                    text=text[:MAX_DOC_CHARS],
                )
            )
    return docs


def _fetch_discussion_docs(spec: RepoSpec, pr_cap: int) -> list[Doc]:
    """PR review discussion, issue threads, and reverts from *before* T. This
    is the C_exp evidence class — the part of the record a fresh agent reading
    the working tree cannot see, and the thing the whole experiment is about."""
    docs: list[Doc] = []
    start = (int(spec.cutoff_t[:4]) - 2, spec.cutoff_t[5:10])
    since = f"{start[0]}-{start[1]}"
    q = (
        f"repo:{spec.repo} is:pr is:merged "
        f"merged:{since}..{spec.cutoff_t[:10]} comments:>3"
    )
    prs: list[dict] = []
    page = 1
    while len(prs) < pr_cap and page <= 10:
        data = _gh([
            "-X", "GET", "search/issues",
            "-f", f"q={q}", "-f", "sort=comments", "-f", "order=desc",
            "-f", "per_page=100", "-f", f"page={page}",
        ])
        items = (data or {}).get("items", [])
        if not items:
            break
        prs.extend(items)
        page += 1
    prs = prs[:pr_cap]

    for pr in prs:
        number = pr["number"]
        title = pr.get("title", "")
        is_revert = title.lower().startswith("revert") or " revert" in title.lower()

        parts: list[str] = [f"# PR #{number}: {title}"]
        body = (pr.get("body") or "").strip()
        if body and not _is_bot((pr.get("user") or {}).get("login")):
            parts.append(f"\n## Description (by @{(pr.get('user') or {}).get('login')})\n{body}")

        for endpoint, label in (
            (f"repos/{spec.repo}/issues/{number}/comments", "Comment"),
            (f"repos/{spec.repo}/pulls/{number}/comments", "Review comment"),
        ):
            try:
                comments = _gh([endpoint, "--paginate"]) or []
            except HarnessError:
                continue
            for c in comments:
                login = ((c.get("user") or {}).get("login")) or ""
                cbody = (c.get("body") or "").strip()
                if _is_bot(login) or len(cbody) < 40:
                    continue
                if not _before_cutoff(c.get("created_at", ""), spec):
                    continue
                loc = f" on `{c['path']}`" if c.get("path") else ""
                parts.append(f"\n## {label} by @{login}{loc}\n{cbody}")

        text = "\n".join(parts)
        if len(text) < 400:
            continue
        if not _before_cutoff(pr.get("closed_at") or pr.get("created_at", ""), spec):
            continue
        source_type = "revert" if is_revert else "review_discussion"
        docs.append(
            Doc(
                doc_id=_doc_id(spec.repo, source_type, str(number)),
                repo=spec.repo,
                source_type=source_type,
                path=f"pull/{number}",
                url=pr["html_url"],
                date=pr.get("closed_at") or pr.get("created_at", ""),
                text=text[:MAX_DOC_CHARS],
            )
        )
    return docs


def fetch(specs: list[RepoSpec] | None = None, *, pr_cap: int = 200) -> dict:
    check_gh()
    specs = specs or REPOS
    stats: dict[str, dict] = {}
    for spec in specs:
        print(f"[corpus.fetch] {spec.repo}")
        ref = _commit_at_cutoff(spec)
        print(f"    tree ref at T: {ref[:10]}")
        docs = _fetch_tree_docs(spec, ref)
        print(f"    {len(docs)} committed docs")
        disc = _fetch_discussion_docs(spec, pr_cap)
        print(f"    {len(disc)} discussion/revert docs")
        docs.extend(disc)
        out = RAW_DIR / f"{spec.repo.replace('/', '__')}.jsonl"
        write_jsonl(out, [d.to_json() for d in docs])
        stats[spec.repo] = {
            "tree_ref_at_T": ref,
            "docs": len(docs),
            "by_source_type": _count_by(docs),
            "tokens": sum(count_tokens(d.text) for d in docs),
        }
        print(f"    -> {out}")
    return stats


def _count_by(docs: list[Doc]) -> dict[str, int]:
    out: dict[str, int] = {}
    for d in docs:
        out[d.source_type] = out.get(d.source_type, 0) + 1
    return dict(sorted(out.items()))


# ---------------------------------------------------------------------------
# build
# ---------------------------------------------------------------------------


def load_raw(specs: list[RepoSpec] | None = None) -> list[dict]:
    specs = specs or REPOS
    rows: list[dict] = []
    for spec in specs:
        p = RAW_DIR / f"{spec.repo.replace('/', '__')}.jsonl"
        if not p.exists():
            raise HarnessError(f"no raw corpus for {spec.repo}: run `corpus fetch` first ({p})")
        rows.extend(read_jsonl(p))
    return rows


def _sort_key(d: dict) -> tuple:
    return (_ORDER_RANK.get(d["source_type"], 99), d.get("date", ""), d["doc_id"])


def _render_doc(d: dict) -> str:
    return (
        f"<document id=\"{d['doc_id']}\" repo=\"{d['repo']}\" "
        f"source=\"{d['source_type']}\" path=\"{d['path']}\" date=\"{d['date']}\">\n"
        f"{d['text']}\n</document>\n"
    )


def build(sizes: list[str] | None = None, specs: list[RepoSpec] | None = None) -> dict:
    """Assemble each corpus size and write MANIFEST.json.

    A larger size is a strict superset of a smaller one: the same deterministic
    order is walked and cut at a bigger budget. That matters for T1 — the
    accuracy-vs-size curve is only interpretable if size is the only thing
    changing.
    """
    sizes = sizes or list(SIZE_BUDGETS)
    docs = sorted(load_raw(specs), key=_sort_key)
    if not docs:
        raise HarnessError("no documents; run `corpus fetch` first")

    total_available = sum(d["tokens"] for d in docs)
    manifest: dict = {
        "built_at": utcnow(),
        "token_estimator": token_estimator_name(),
        "documents_available": len(docs),
        "tokens_available": total_available,
        "assembly_order": SOURCE_TYPES,
        "sizes": {},
        "repositories": [
            {
                "repo": s.repo,
                "cutoff_t": s.cutoff_t,
                "decision_dir": s.decision_dir,
                "selection_reason_T7": s.selection_reason,
            }
            for s in (specs or REPOS)
        ],
    }

    # Available tokens per source type, used to allocate each size's budget.
    by_type: dict[str, list[dict]] = {}
    for d in docs:
        by_type.setdefault(d["source_type"], []).append(d)
    avail = {t: sum(d["tokens"] for d in ds) for t, ds in by_type.items()}
    total_avail = sum(avail.values()) or 1

    for size in sizes:
        budget = SIZE_BUDGETS[size]

        # Stratified allocation, not priority truncation.
        #
        # Filling the budget by walking one global order silently produces a
        # corpus made of whichever source type sorts first. On the real mined
        # data that was 100% committed documentation at every size — which
        # would have left C_exp with no claims to extract and made the §8
        # ablation, the reason this project exists, a measurement of nothing.
        # Each size therefore preserves the source mix of the whole record.
        quota = {t: int(budget * avail[t] / total_avail) for t in avail}
        # Redistribute what a source type cannot fill to the ones that can.
        for t in list(quota):
            if quota[t] > avail[t]:
                quota[t] = avail[t]
        slack = budget - sum(quota.values())
        while slack > 0:
            hungry = [t for t in quota if avail[t] > quota[t]]
            if not hungry:
                break
            share = max(1, slack // len(hungry))
            for t in hungry:
                take = min(share, avail[t] - quota[t], slack)
                quota[t] += take
                slack -= take
                if slack <= 0:
                    break

        chosen: list[dict] = []
        used_by_type: dict[str, int] = {t: 0 for t in quota}
        for d in docs:  # already in the fixed deterministic order
            t = d["source_type"]
            remaining = quota.get(t, 0) - used_by_type.get(t, 0)
            if remaining <= 0:
                continue
            text = d["text"]
            tok = d["tokens"]
            if tok > remaining:
                text = truncate_to_tokens(text, remaining)
                tok = count_tokens(text)
                if tok < 200:
                    continue
                d = {**d, "text": text, "tokens": tok, "truncated": True}
            chosen.append(d)
            used_by_type[t] += tok
        used = sum(used_by_type.values())

        body = "".join(_render_doc(d) for d in chosen)
        out_dir = CORPUS_DIR / size
        out_dir.mkdir(parents=True, exist_ok=True)
        corpus_file = out_dir / "corpus.txt"
        corpus_file.write_text(body, encoding="utf-8")
        write_jsonl(out_dir / "documents.jsonl", chosen)

        sha = sha256_text(body)
        manifest["sizes"][size] = {
            "budget_tokens": budget,
            "corpus_sha256": sha,
            "corpus_tokens": count_tokens(body),
            "body_tokens": used,
            "documents": len(chosen),
            "by_source_type": _count_by_dicts(chosen),
            "tokens_by_source_type": dict(sorted(used_by_type.items())),
            "allocation": "stratified by source type, proportional to the whole record",
            "inventory": [
                {
                    "doc_id": d["doc_id"],
                    "repo": d["repo"],
                    "source_type": d["source_type"],
                    "path": d["path"],
                    "url": d["url"],
                    "date": d["date"],
                    "tokens": d["tokens"],
                    "text_sha256": d["text_sha256"],
                    "truncated": bool(d.get("truncated")),
                }
                for d in chosen
            ],
        }
        # T1 honesty: if the budget could not be met, say so here rather than
        # letting a short corpus be read as a full one.
        if used < budget * 0.9:
            manifest["sizes"][size]["under_budget_warning"] = (
                f"assembled {used} tokens against a {budget} budget; the mined "
                f"record is not large enough for this size. Fetch more "
                f"repositories or raise the PR cap before treating this size as "
                f"a measurement of that regime."
            )
        print(f"[corpus.build] {size}: {len(chosen)} docs, {used} tokens, sha {sha[:12]}")

    write_json(MANIFEST_PATH, manifest)
    print(f"[corpus.build] manifest -> {MANIFEST_PATH}")
    return manifest


def _count_by_dicts(docs: list[dict]) -> dict[str, int]:
    out: dict[str, int] = {}
    for d in docs:
        out[d["source_type"]] = out.get(d["source_type"], 0) + 1
    return dict(sorted(out.items()))


def corpus_text(size: str) -> str:
    p = CORPUS_DIR / size / "corpus.txt"
    if not p.exists():
        raise HarnessError(f"corpus size {size!r} not built: {p} missing")
    return p.read_text(encoding="utf-8")


def corpus_sha(size: str) -> str:
    manifest = json.loads(MANIFEST_PATH.read_text(encoding="utf-8"))
    return manifest["sizes"][size]["corpus_sha256"]


def corpus_documents(size: str) -> list[dict]:
    return read_jsonl(CORPUS_DIR / size / "documents.jsonl")


if __name__ == "__main__":
    import argparse

    ap = argparse.ArgumentParser(description="Corpus fetch/build (§2 D1, §3)")
    ap.add_argument("stage", choices=["fetch", "build"])
    ap.add_argument("--pr-cap", type=int, default=200)
    ap.add_argument("--size", action="append")
    args = ap.parse_args()
    if args.stage == "fetch":
        fetch(pr_cap=args.pr_cap)
    else:
        build(args.size)
