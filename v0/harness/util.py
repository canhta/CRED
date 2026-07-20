"""Shared primitives: hashing, ids, jsonl io, token counting.

Stdlib-only by design. The harness must run on a clean machine with no
`pip install` step, because a dependency that fails to resolve six months from
now is a result that cannot be reproduced.
"""

from __future__ import annotations

import hashlib
import json
import os
import random
import re
import time
from dataclasses import asdict, is_dataclass
from pathlib import Path
from typing import Any, Iterable, Iterator

HARNESS_VERSION = "0.1.0"

REPO_ROOT = Path(__file__).resolve().parents[2]
V0_ROOT = REPO_ROOT / "v0"


# --------------------------------------------------------------------------
# hashing
# --------------------------------------------------------------------------


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def sha256_text(text: str) -> str:
    return sha256_bytes(text.encode("utf-8"))


def sha256_file(path: str | Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as fh:
        for chunk in iter(lambda: fh.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def sha256_json(obj: Any) -> str:
    """Hash of a canonical JSON encoding. Key order and whitespace fixed so the
    same logical object always hashes the same."""
    return sha256_text(json.dumps(obj, sort_keys=True, separators=(",", ":")))


def short_sha(full: str, n: int = 12) -> str:
    return full[:n]


# --------------------------------------------------------------------------
# ids
# --------------------------------------------------------------------------

_ULID_ALPHABET = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"


def ulid() -> str:
    """Lexicographically sortable 26-char id. Not a spec-perfect ULID (the
    random half uses `random`, not `secrets`) but monotone by time, which is
    the only property the run log needs."""
    ts = int(time.time() * 1000)
    out = []
    for _ in range(10):
        out.append(_ULID_ALPHABET[ts % 32])
        ts //= 32
    head = "".join(reversed(out))
    tail = "".join(random.choice(_ULID_ALPHABET) for _ in range(16))
    return head + tail


def utcnow() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


# --------------------------------------------------------------------------
# jsonl io
# --------------------------------------------------------------------------


def _default(o: Any) -> Any:
    if is_dataclass(o):
        return asdict(o)
    raise TypeError(f"not JSON serializable: {type(o)!r}")


def read_jsonl(path: str | Path) -> list[dict]:
    p = Path(path)
    if not p.exists():
        return []
    rows = []
    with open(p, "r", encoding="utf-8") as fh:
        for i, line in enumerate(fh, 1):
            line = line.strip()
            if not line:
                continue
            try:
                rows.append(json.loads(line))
            except json.JSONDecodeError as exc:
                raise ValueError(f"{p}:{i}: malformed JSON: {exc}") from exc
    return rows


def iter_jsonl(path: str | Path) -> Iterator[dict]:
    p = Path(path)
    if not p.exists():
        return
    with open(p, "r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if line:
                yield json.loads(line)


def write_jsonl(path: str | Path, rows: Iterable[dict]) -> Path:
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    tmp = p.with_suffix(p.suffix + ".tmp")
    with open(tmp, "w", encoding="utf-8") as fh:
        for row in rows:
            fh.write(json.dumps(row, sort_keys=True, default=_default) + "\n")
    os.replace(tmp, p)
    return p


def append_jsonl(path: str | Path, row: dict) -> None:
    """Append one record and fsync. The runner is resumable, so a half-written
    line after a crash is a corrupt log, not an inconvenience."""
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    with open(p, "a", encoding="utf-8") as fh:
        fh.write(json.dumps(row, sort_keys=True, default=_default) + "\n")
        fh.flush()
        os.fsync(fh.fileno())


def write_json(path: str | Path, obj: Any) -> Path:
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    tmp = p.with_suffix(p.suffix + ".tmp")
    with open(tmp, "w", encoding="utf-8") as fh:
        json.dump(obj, fh, indent=2, sort_keys=True, default=_default)
        fh.write("\n")
    os.replace(tmp, p)
    return p


def read_json(path: str | Path) -> Any:
    with open(path, "r", encoding="utf-8") as fh:
        return json.load(fh)


# --------------------------------------------------------------------------
# token counting
# --------------------------------------------------------------------------

_ENCODER = None
_ENCODER_NAME = "heuristic-chars-div-4"


def _encoder():
    global _ENCODER, _ENCODER_NAME
    if _ENCODER is not None:
        return _ENCODER
    try:  # optional; absent on a clean machine and that is fine
        import tiktoken  # type: ignore

        _ENCODER = tiktoken.get_encoding("cl100k_base")
        _ENCODER_NAME = "tiktoken:cl100k_base"
    except Exception:
        _ENCODER = False
    return _ENCODER


def token_estimator_name() -> str:
    _encoder()
    return _ENCODER_NAME


def count_tokens(text: str) -> int:
    """Estimated token count.

    This is used for corpus sizing and for the B/C token ceiling, never for
    billing. Billing uses provider-reported counts from the run record, which
    is why T8 is satisfied even though this estimator is approximate. The
    estimator in use is recorded in MANIFEST.json so a reader knows which one
    produced the numbers.
    """
    enc = _encoder()
    if enc:
        return len(enc.encode(text, disallowed_special=()))
    return (len(text) + 3) // 4


# --------------------------------------------------------------------------
# misc
# --------------------------------------------------------------------------


def slugify(text: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", text.lower()).strip("-")


def truncate_to_tokens(text: str, max_tokens: int) -> str:
    """Cut text to a token ceiling. Binary search on characters when a real
    encoder is present; proportional cut otherwise."""
    if count_tokens(text) <= max_tokens:
        return text
    lo, hi = 0, len(text)
    while lo < hi:
        mid = (lo + hi + 1) // 2
        if count_tokens(text[:mid]) <= max_tokens:
            lo = mid
        else:
            hi = mid - 1
    return text[:lo]


class HarnessError(RuntimeError):
    """Raised when a pre-registered invariant is violated. These are meant to
    stop the run, not to be caught and logged."""
