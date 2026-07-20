"""Arm B retrieval — chunking, BM25, fusion, recall@k.

Status, stated plainly because T2 turns on it: **the lexical half is built and
runnable; the dense half and the reranker are not.** §3 says "arm B is not a
stub" and requires a real embedding model, a tuned `k`, and a reranker. A
BM25-only arm B would be exactly the weak-retriever strawman T2 exists to
prevent, so `build_index` refuses to produce an arm-B index unless a dense
retriever is registered. Arm B is out of scope for the MVE (§9 runs A0, A,
C-both only); it is a blocker for the full design.

What is built here and used by the full pipeline regardless of arm B:
  - deterministic chunking of the corpus, shared with the C arms
  - Okapi BM25 over those chunks
  - reciprocal rank fusion
  - recall@k against gold evidence spans, with the §5 voiding rule
"""

from __future__ import annotations

import math
import re
from collections import Counter
from dataclasses import dataclass
from typing import Callable, Protocol

from .util import V0_ROOT, HarnessError, count_tokens, sha256_text, write_json, write_jsonl

INDEX_DIR = V0_ROOT / "index"

CHUNK_TOKENS = 400
CHUNK_OVERLAP_TOKENS = 80

_TOKEN = re.compile(r"[a-z0-9_]+")


def tokenize(text: str) -> list[str]:
    return _TOKEN.findall(text.lower())


# ---------------------------------------------------------------------------
# chunking
# ---------------------------------------------------------------------------


@dataclass(frozen=True)
class Chunk:
    chunk_id: str
    doc_id: str
    repo: str
    source_type: str
    path: str
    url: str
    text: str
    tokens: int


def chunk_documents(
    docs: list[dict], *, chunk_tokens: int = CHUNK_TOKENS, overlap: int = CHUNK_OVERLAP_TOKENS
) -> list[Chunk]:
    """Split on paragraph boundaries, packing to a token budget.

    Deterministic: same documents in, same chunk ids out. The C arms cite chunk
    ids as evidence pointers, so instability here would break the ablation.
    """
    if overlap >= chunk_tokens:
        raise HarnessError("overlap must be smaller than chunk size")
    out: list[Chunk] = []
    for d in docs:
        paras = [p.strip() for p in d["text"].split("\n\n") if p.strip()]
        buf: list[str] = []
        buf_tokens = 0
        idx = 0

        def flush() -> None:
            nonlocal buf, buf_tokens, idx
            if not buf:
                return
            text = "\n\n".join(buf)
            out.append(
                Chunk(
                    chunk_id=f"chk-{sha256_text(d['doc_id'] + str(idx) + text)[:12]}",
                    doc_id=d["doc_id"],
                    repo=d["repo"],
                    source_type=d["source_type"],
                    path=d["path"],
                    url=d.get("url", ""),
                    text=text,
                    tokens=count_tokens(text),
                )
            )
            idx += 1
            # Carry the tail forward as overlap.
            carry: list[str] = []
            carried = 0
            for p in reversed(buf):
                t = count_tokens(p)
                if carried + t > overlap:
                    break
                carry.insert(0, p)
                carried += t
            buf = carry
            buf_tokens = carried

        for p in paras:
            t = count_tokens(p)
            if t > chunk_tokens:
                flush()
                # A single oversized paragraph becomes its own chunk rather
                # than being dropped.
                out.append(
                    Chunk(
                        chunk_id=f"chk-{sha256_text(d['doc_id'] + str(idx) + p)[:12]}",
                        doc_id=d["doc_id"],
                        repo=d["repo"],
                        source_type=d["source_type"],
                        path=d["path"],
                        url=d.get("url", ""),
                        text=p,
                        tokens=t,
                    )
                )
                idx += 1
                continue
            if buf_tokens + t > chunk_tokens:
                flush()
            buf.append(p)
            buf_tokens += t
        flush()
    return out


# ---------------------------------------------------------------------------
# BM25
# ---------------------------------------------------------------------------


class BM25:
    """Okapi BM25. Pure stdlib, no index server, no dependency to rot."""

    def __init__(self, corpus: list[list[str]], *, k1: float = 1.5, b: float = 0.75) -> None:
        self.k1 = k1
        self.b = b
        self.n = len(corpus)
        self.doc_len = [len(d) for d in corpus]
        self.avgdl = (sum(self.doc_len) / self.n) if self.n else 0.0
        self.tf: list[Counter[str]] = [Counter(d) for d in corpus]
        df: Counter[str] = Counter()
        for d in corpus:
            df.update(set(d))
        self.idf = {
            term: math.log(1 + (self.n - freq + 0.5) / (freq + 0.5)) for term, freq in df.items()
        }

    def score(self, query: list[str], i: int) -> float:
        tf = self.tf[i]
        dl = self.doc_len[i] or 1
        total = 0.0
        for term in query:
            f = tf.get(term, 0)
            if not f:
                continue
            idf = self.idf.get(term, 0.0)
            total += idf * (f * (self.k1 + 1)) / (f + self.k1 * (1 - self.b + self.b * dl / self.avgdl))
        return total

    def rank(self, query: str, top_k: int) -> list[tuple[int, float]]:
        q = tokenize(query)
        scored = [(i, self.score(q, i)) for i in range(self.n)]
        scored.sort(key=lambda kv: (-kv[1], kv[0]))
        return [s for s in scored[:top_k] if s[1] > 0.0]


def reciprocal_rank_fusion(rankings: list[list[str]], *, k: int = 60) -> list[tuple[str, float]]:
    """RRF over several ranked id lists. `k=60` is the value from the original
    RRF paper; it is not tuned here."""
    scores: dict[str, float] = {}
    for ranking in rankings:
        for rank, cid in enumerate(ranking, start=1):
            scores[cid] = scores.get(cid, 0.0) + 1.0 / (k + rank)
    return sorted(scores.items(), key=lambda kv: (-kv[1], kv[0]))


# ---------------------------------------------------------------------------
# the missing halves
# ---------------------------------------------------------------------------


class DenseRetriever(Protocol):
    name: str

    def rank(self, query: str, top_k: int) -> list[tuple[str, float]]: ...


class Reranker(Protocol):
    name: str

    def rerank(self, query: str, chunk_ids: list[str], top_k: int) -> list[str]: ...


_DENSE_FACTORY: Callable[[list[Chunk]], DenseRetriever] | None = None
_RERANKER: Reranker | None = None


def register_dense(factory: Callable[[list[Chunk]], DenseRetriever]) -> None:
    global _DENSE_FACTORY
    _DENSE_FACTORY = factory


def register_reranker(r: Reranker) -> None:
    global _RERANKER
    _RERANKER = r


def arm_b_available() -> bool:
    return _DENSE_FACTORY is not None and _RERANKER is not None


def require_arm_b() -> None:
    if not arm_b_available():
        raise HarnessError(
            "Arm B is not runnable: no dense retriever and/or reranker is registered.\n"
            "§3 requires arm B to have a real embedding model, a tuned k, and a "
            "reranker; running it on BM25 alone would produce the weak-retriever "
            "strawman T2 exists to prevent, and a favourable C-vs-B number from "
            "such a run would be worthless.\n"
            "Register implementations with retrieval.register_dense() and "
            "retrieval.register_reranker(), or leave arm B out of the config "
            "(the MVE does)."
        )


# ---------------------------------------------------------------------------
# index + recall@k
# ---------------------------------------------------------------------------


def build_index(docs: list[dict], size: str) -> dict:
    """Build and persist the chunk index. Lexical-only indexes are allowed and
    useful (the C arms and the decoy checker use the chunks); it is *running
    arm B* that is gated."""
    chunks = chunk_documents(docs)
    out_dir = INDEX_DIR / size
    write_jsonl(out_dir / "chunks.jsonl", [c.__dict__ for c in chunks])
    meta = {
        "size": size,
        "chunks": len(chunks),
        "chunk_tokens": CHUNK_TOKENS,
        "chunk_overlap_tokens": CHUNK_OVERLAP_TOKENS,
        "lexical": "bm25(k1=1.5, b=0.75)",
        "dense": _DENSE_FACTORY.__name__ if _DENSE_FACTORY else None,
        "reranker": getattr(_RERANKER, "name", None),
        "fusion": "reciprocal_rank_fusion(k=60)",
        "arm_b_runnable": arm_b_available(),
    }
    write_json(out_dir / "index.json", meta)
    print(f"[index] {size}: {len(chunks)} chunks; arm B runnable = {meta['arm_b_runnable']}")
    return meta


def recall_at_k(
    retrieved_chunk_ids: list[str], gold_doc_ids: set[str], chunk_to_doc: dict[str, str]
) -> float:
    """Share of gold evidence documents that appear anywhere in the retrieved
    set. §5: below 0.60 the arm-B comparison is void, not favourable."""
    if not gold_doc_ids:
        return float("nan")
    hit = {chunk_to_doc.get(c) for c in retrieved_chunk_ids} & gold_doc_ids
    return len(hit) / len(gold_doc_ids)
