"""Task mining, §4 steps 1-2.

Mines *anchors* — post-cutoff text in which a maintainer refers backwards to a
decision, convention, or failure that predates the cutoff. The maintainer's own
words are the ground truth (D2). The founder does not write questions.

This module produces anchors only. Turning an anchor into a question + gold
answer is `draft.py`, and it needs a model. The split is deliberate: anchor
mining is deterministic, auditable, and free; drafting is neither.

Transport is the `gh` CLI so that authentication, pagination behaviour, and
rate-limit handling are GitHub's problem rather than ours. `gh auth status`
must be clean before running.
"""

from __future__ import annotations

import json
import re
import subprocess
import time
from dataclasses import dataclass, field
from pathlib import Path

from .util import V0_ROOT, HarnessError, sha256_text, utcnow, write_json, write_jsonl

# ---------------------------------------------------------------------------
# §4 step 2: backward-reference patterns.
# Written before any repository was queried, and frozen here so the selection
# rule is inspectable rather than tuned until it produced pleasing anchors.
# ---------------------------------------------------------------------------

ANCHOR_PATTERNS: dict[str, str] = {
    "prior_decision": r"\b(as (we|previously) (decided|agreed)|we decided|the decision was|we settled on|we standardi[sz]ed on)\b",
    "prior_attempt": r"\b(we (tried|attempted) (that|this|it)|that was tried|we already tried|has been tried before)\b",
    "moved_away": r"\b(we moved away from|we stopped using|we no longer|we dropped|we deprecated|we replaced .{0,40} with)\b",
    "convention": r"\b(the convention (here|in this repo|is)|by convention|we conventionally|the pattern here is|we('ve| have) always|we('ve| have) never)\b",
    "reverted": r"\b(was reverted|reverted because|had to revert|we reverted|this broke|caused an (outage|incident|regression))\b",
    "cross_ref": r"(?:^|\s)(?:see|per|cf\.?|refs?|context in)\s+#\d+\b",
    "historical_why": r"\b(the reason (we|this|it)|historically|originally we|the original (reason|motivation)|this exists because)\b",
}

_COMPILED = {k: re.compile(v, re.IGNORECASE) for k, v in ANCHOR_PATTERNS.items()}

# Changes to the pattern set, on the record. Published in MINING.json.
#
# Tuning a selection rule after looking at its output is normally how a task set
# gets steered, so each entry has to justify itself. The bar applied here: a
# change is allowed only if the old pattern was matching text that is not a
# backward reference *at all* — a specification error — and not because the
# anchors it produced were inconvenient. No task had been drafted and no result
# existed when these were made; the freeze that matters (§4 step 8) comes later.
PATTERN_REVISIONS = [
    {
        "date": "2026-07-20",
        "pattern": "convention",
        "was": r"...|we always|we never)",
        "now": r"...|we('ve| have) always|we('ve| have) never)",
        "reason": (
            "The bare `we always` / `we never` alternatives matched "
            "forward-looking review requests — e.g. 'assert that we never see an "
            "event that is older than the checkpoint' — which are proposals about "
            "future code, not references to an established convention. Requiring "
            "the present-perfect ('we have never', \"we've always\") restricts the "
            "match to statements about existing practice, which is what the "
            "pattern was for."
        ),
        "effect_on_existing_anchors": (
            "Anchors mined before this change are unaffected on disk. Re-run "
            "`mine` to apply it; several cockroachdb anchors matched only on this "
            "alternative and will drop out."
        ),
    },
]

# Bot authors produce template text that matches nothing useful and floods the
# sample. Excluded by suffix and by explicit list.
BOT_LOGINS = {
    "dependabot", "renovate", "github-actions", "codecov", "sonarcloud",
    "coderabbitai", "changeset-bot", "vercel", "netlify", "mergify",
    "blathers-crl", "cockroach-teamcity", "otelbot", "backstage-service",
}

MIN_ANCHOR_CHARS = 80
MAX_ANCHOR_CHARS = 4000

ANCHORS_PATH = V0_ROOT / "mining" / "anchors.jsonl"
MINING_MANIFEST_PATH = V0_ROOT / "mining" / "MINING.json"


# ---------------------------------------------------------------------------
# Repository selection — §4 source criteria, and T7 disclosure.
# ---------------------------------------------------------------------------


@dataclass(frozen=True)
class RepoSpec:
    repo: str
    cutoff_t: str  # ISO8601. Corpus C = everything authored before T.
    anchor_window_end: str
    decision_dir: str  # the `adr/` or `decisions/` equivalent §4 requires
    selection_reason: str  # T7: this choice is not blinded, so it is disclosed
    code_globs: list[str] = field(default_factory=list)


# T7: "Repository and cutoff selection is not blinded. Disclosed." This list IS
# the disclosure. Each entry says why it was picked and what it risks.
REPOS: list[RepoSpec] = [
    RepoSpec(
        repo="backstage/backstage",
        cutoff_t="2026-01-01T00:00:00Z",
        anchor_window_end="2026-07-01T00:00:00Z",
        decision_dir="docs/architecture-decisions",
        selection_reason=(
            "Chosen for a maintained ADR directory (docs/architecture-decisions), "
            "created 2020-01 so >3 years of history, and >20k merged PRs with "
            "substantive review threads. Risk: it is a plugin framework, so much "
            "of its decision history is about extension points rather than "
            "runtime behaviour, which may bias tasks toward API-shape questions."
        ),
        code_globs=["packages/**/*.ts", "docs/**/*.md"],
    ),
    RepoSpec(
        repo="cockroachdb/cockroach",
        cutoff_t="2026-01-01T00:00:00Z",
        anchor_window_end="2026-07-01T00:00:00Z",
        decision_dir="docs/RFCS",
        selection_reason=(
            "Chosen for docs/RFCS — an unusually complete written record of "
            "rejected alternatives, which is exactly the C-exp evidence class — "
            "created 2014-02, and a strong culture of documented reverts. Risk: "
            "database internals are specialised enough that a model's parametric "
            "knowledge of CockroachDB may be thin, which could flatter every "
            "memory arm equally. The A0 filter is the control for this."
        ),
        code_globs=["pkg/**/*.go", "docs/RFCS/*.md"],
    ),
    RepoSpec(
        repo="open-telemetry/opentelemetry-collector",
        cutoff_t="2026-01-01T00:00:00Z",
        anchor_window_end="2026-07-01T00:00:00Z",
        decision_dir="docs/rfcs",
        selection_reason=(
            "Chosen as a third, smaller repository (7k stars vs 32k/34k) to test "
            "whether the effect survives outside very large projects, with "
            "docs/rfcs present and a specification-driven review culture that "
            "produces explicit 'we decided X because Y' comments. Risk: it is "
            "governed by an external specification, so some 'decisions' are "
            "inherited rather than made locally."
        ),
        code_globs=["**/*.go", "docs/**/*.md"],
    ),
]

# Cutoff rationale, recorded once rather than per repo:
# T = 2026-01-01 places the entire anchor window after the assistant model
# family's January 2026 training cutoff. That does not make the *answers*
# unmemorized — the decisions themselves predate T and may well be in training
# data — but it does mean the specific post-T discussion that supplies ground
# truth is not. T6's actual control is the A0 filter, not the date.


# ---------------------------------------------------------------------------
# gh transport
# ---------------------------------------------------------------------------


def _gh(args: list[str], *, retries: int = 3) -> object:
    """Run `gh api ...` and return parsed JSON."""
    last: Exception | None = None
    for attempt in range(retries):
        proc = subprocess.run(
            ["gh", "api", *args],
            capture_output=True,
            text=True,
        )
        if proc.returncode == 0:
            try:
                return json.loads(proc.stdout or "null")
            except json.JSONDecodeError as exc:  # pragma: no cover
                last = exc
                break
        stderr = proc.stderr.strip()
        if "rate limit" in stderr.lower() or "abuse" in stderr.lower():
            wait = 30 * (attempt + 1)
            print(f"    rate limited; sleeping {wait}s")
            time.sleep(wait)
            continue
        last = HarnessError(f"gh api {' '.join(args)} failed: {stderr}")
        break
    raise HarnessError(f"gh api call failed after {retries} attempts: {last}")


def check_gh() -> None:
    proc = subprocess.run(["gh", "auth", "status"], capture_output=True, text=True)
    if proc.returncode != 0:
        raise HarnessError(
            "`gh auth status` is not clean. Run `gh auth login` before mining — "
            "unauthenticated GitHub API access is rate-limited to 60 requests/hour."
        )


# ---------------------------------------------------------------------------
# Matching
# ---------------------------------------------------------------------------


def _is_bot(login: str | None) -> bool:
    if not login:
        return True
    low = login.lower()
    return low.endswith("[bot]") or low.endswith("-bot") or low in BOT_LOGINS


def match_patterns(text: str) -> list[str]:
    if not text:
        return []
    return sorted(name for name, rx in _COMPILED.items() if rx.search(text))


def _in_window(created: str, spec: RepoSpec) -> bool:
    return spec.cutoff_t <= created < spec.anchor_window_end


def _anchor(
    spec: RepoSpec,
    *,
    kind: str,
    url: str,
    created: str,
    author: str,
    body: str,
    pr_number: int,
    pr_title: str,
    merge_sha: str | None,
    patterns: list[str],
) -> dict:
    body = body.strip()
    return {
        "anchor_id": "anc-" + sha256_text(url + body)[:12],
        "repo": spec.repo,
        "cutoff_t": spec.cutoff_t,
        "kind": kind,
        "anchor_url": url,
        "anchor_sha": merge_sha,
        "anchor_date": created,
        "author": author,
        "pr_number": pr_number,
        "pr_title": pr_title,
        "matched_patterns": patterns,
        "text": body[:MAX_ANCHOR_CHARS],
        "text_sha256": sha256_text(body),
        "mined_at": utcnow(),
    }


# ---------------------------------------------------------------------------
# Mining
# ---------------------------------------------------------------------------


def search_merged_prs(spec: RepoSpec, limit: int) -> list[dict]:
    """PRs merged inside the anchor window. GitHub's search API caps at 1000
    results; we take the most-commented first, because a PR with no discussion
    cannot contain a backward reference."""
    out: list[dict] = []
    per_page = 100
    q = (
        f"repo:{spec.repo} is:pr is:merged "
        f"merged:{spec.cutoff_t[:10]}..{spec.anchor_window_end[:10]} "
        f"comments:>2"
    )
    page = 1
    while len(out) < limit and page <= 10:
        data = _gh(
            [
                "-X", "GET", "search/issues",
                "-f", f"q={q}",
                "-f", "sort=comments",
                "-f", "order=desc",
                "-f", f"per_page={per_page}",
                "-f", f"page={page}",
            ]
        )
        items = (data or {}).get("items", [])
        if not items:
            break
        out.extend(items)
        page += 1
    return out[:limit]


def mine_pr(spec: RepoSpec, pr: dict) -> list[dict]:
    number = pr["number"]
    title = pr.get("title", "")
    anchors: list[dict] = []

    detail = _gh([f"repos/{spec.repo}/pulls/{number}"])
    merge_sha = (detail or {}).get("merge_commit_sha")

    # 1. PR body
    body = (pr.get("body") or "").strip()
    created = pr.get("created_at", "")
    author = ((pr.get("user") or {}).get("login")) or ""
    if body and _in_window(created, spec) and not _is_bot(author):
        pats = match_patterns(body)
        if pats and len(body) >= MIN_ANCHOR_CHARS:
            anchors.append(
                _anchor(spec, kind="pr_body", url=pr["html_url"], created=created,
                        author=author, body=body, pr_number=number, pr_title=title,
                        merge_sha=merge_sha, patterns=pats)
            )

    # 2. Issue-style comments on the PR
    for c in _gh([f"repos/{spec.repo}/issues/{number}/comments", "--paginate"]) or []:
        login = ((c.get("user") or {}).get("login")) or ""
        cbody = (c.get("body") or "").strip()
        if _is_bot(login) or len(cbody) < MIN_ANCHOR_CHARS:
            continue
        if not _in_window(c.get("created_at", ""), spec):
            continue
        pats = match_patterns(cbody)
        if pats:
            anchors.append(
                _anchor(spec, kind="issue_comment", url=c["html_url"],
                        created=c["created_at"], author=login, body=cbody,
                        pr_number=number, pr_title=title, merge_sha=merge_sha,
                        patterns=pats)
            )

    # 3. Review comments (inline, on the diff) — the richest source of
    #    "the convention here is" statements.
    for c in _gh([f"repos/{spec.repo}/pulls/{number}/comments", "--paginate"]) or []:
        login = ((c.get("user") or {}).get("login")) or ""
        cbody = (c.get("body") or "").strip()
        if _is_bot(login) or len(cbody) < MIN_ANCHOR_CHARS:
            continue
        if not _in_window(c.get("created_at", ""), spec):
            continue
        pats = match_patterns(cbody)
        if pats:
            anchors.append(
                _anchor(spec, kind="review_comment", url=c["html_url"],
                        created=c["created_at"], author=login, body=cbody,
                        pr_number=number, pr_title=title, merge_sha=merge_sha,
                        patterns=pats)
            )

    return anchors


def mine(
    specs: list[RepoSpec] | None = None,
    *,
    prs_per_repo: int = 120,
    out_path: Path = ANCHORS_PATH,
) -> dict:
    check_gh()
    specs = specs or REPOS
    all_anchors: list[dict] = []
    per_repo_stats: dict[str, dict] = {}

    for spec in specs:
        print(f"[mine] {spec.repo}  T={spec.cutoff_t}  window<{spec.anchor_window_end}")
        prs = search_merged_prs(spec, prs_per_repo)
        print(f"    {len(prs)} merged PRs with >2 comments in window")
        found: list[dict] = []
        for i, pr in enumerate(prs, 1):
            try:
                found.extend(mine_pr(spec, pr))
            except HarnessError as exc:
                print(f"    !! PR #{pr['number']}: {exc}")
                continue
            if i % 20 == 0:
                print(f"    {i}/{len(prs)} PRs scanned, {len(found)} anchors")
        # Dedupe within a repo by anchor_id.
        seen: set[str] = set()
        deduped = []
        for a in found:
            if a["anchor_id"] in seen:
                continue
            seen.add(a["anchor_id"])
            deduped.append(a)
        per_repo_stats[spec.repo] = {
            "prs_scanned": len(prs),
            "anchors": len(deduped),
            # Yield varies enormously by repository culture. Recorded so the
            # operator can size the mining run rather than discover mid-draft
            # that the anchor pool cannot support the pre-registered n.
            "anchors_per_pr": round(len(deduped) / len(prs), 3) if prs else None,
            "by_kind": _count(deduped, "kind"),
            "by_pattern": _count_patterns(deduped),
        }
        print(f"    -> {len(deduped)} anchors")
        all_anchors.extend(deduped)
        # Written after every repository, not only at the end: mining is slow
        # enough that an interrupted run must not lose an hour of work. The
        # manifest is written on the same beat — a manifest that describes a
        # different run than anchors.jsonl is worse than no manifest, because
        # it looks authoritative.
        write_jsonl(out_path, all_anchors)
        write_json(
            MINING_MANIFEST_PATH,
            _manifest(specs, per_repo_stats, all_anchors, out_path, prs_per_repo, complete=False),
        )

    write_jsonl(out_path, all_anchors)
    manifest = _manifest(specs, per_repo_stats, all_anchors, out_path, prs_per_repo, complete=True)
    write_json(MINING_MANIFEST_PATH, manifest)
    print(f"[mine] {len(all_anchors)} anchors -> {out_path}")
    return manifest


def _manifest(specs, per_repo_stats, all_anchors, out_path, prs_per_repo, *, complete: bool) -> dict:
    return {
        "mined_at": utcnow(),
        "run_complete": complete,
        "repos_finished": sorted(per_repo_stats),
        "anchors_path": str(out_path.relative_to(V0_ROOT.parent)),
        "anchors_total": len(all_anchors),
        "anchor_patterns": ANCHOR_PATTERNS,
        "pattern_revisions": PATTERN_REVISIONS,
        "min_anchor_chars": MIN_ANCHOR_CHARS,
        "prs_per_repo_cap": prs_per_repo,
        "repositories": [
            {
                "repo": s.repo,
                "cutoff_t": s.cutoff_t,
                "anchor_window_end": s.anchor_window_end,
                "decision_dir": s.decision_dir,
                "selection_reason_T7": s.selection_reason,
                **per_repo_stats.get(s.repo, {}),
            }
            for s in specs
        ],
        "disclosure_T7": (
            "Repository and cutoff selection is not blinded. The founder chose "
            "these repositories and this cutoff. The reasons are recorded above, "
            "including the known bias risk of each choice."
        ),
    }


def _count(rows: list[dict], key: str) -> dict[str, int]:
    out: dict[str, int] = {}
    for r in rows:
        out[r[key]] = out.get(r[key], 0) + 1
    return dict(sorted(out.items(), key=lambda kv: -kv[1]))


def _count_patterns(rows: list[dict]) -> dict[str, int]:
    out: dict[str, int] = {}
    for r in rows:
        for p in r["matched_patterns"]:
            out[p] = out.get(p, 0) + 1
    return dict(sorted(out.items(), key=lambda kv: -kv[1]))


if __name__ == "__main__":
    import argparse

    ap = argparse.ArgumentParser(description="Mine backward-reference anchors (§4 steps 1-2)")
    ap.add_argument("--prs-per-repo", type=int, default=120)
    ap.add_argument("--repo", action="append", help="restrict to these repos")
    args = ap.parse_args()
    specs = REPOS
    if args.repo:
        specs = [s for s in REPOS if s.repo in args.repo]
    mine(specs, prs_per_repo=args.prs_per_repo)
