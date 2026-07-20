"""Memory-block assembly — §3.

Every arm receives the same question, the same output-format instruction, and
the same template shape. The *only* thing that differs is the block assembled
here. That is the experiment.

Hard constraints enforced in this file, because a comment is not an enforcement:

  - B and all three C arms share one token ceiling (default 4,000). A win for C
    that comes from a larger block is not a win, so `assemble` measures the
    block it produced and refuses to return one over budget.
  - C_doc, C_exp, and C_both share that same ceiling. Otherwise C_both wins by
    volume and the ablation is worthless — the exact mistake §8 names.
  - Arm A gets the whole corpus, not a truncated sample. If the corpus exceeds
    the model's window, that is reported as arm A's failure mode
    (`truncated_for_model=True` on the returned block) rather than papered over.
"""

from __future__ import annotations

from dataclasses import dataclass, field

from .config import Config, ModelSpec
from .corpus import corpus_documents, corpus_text
from .memory import load_claims
from .retrieval import BM25, chunk_documents, require_arm_b, tokenize
from .util import HarnessError, count_tokens, sha256_text, truncate_to_tokens


@dataclass
class MemoryBlock:
    arm: str
    text: str
    tokens: int
    sha: str
    retrieved_ids: list[str] = field(default_factory=list)
    truncated_for_model: bool = False
    over_ceiling: bool = False

    @classmethod
    def build(cls, arm: str, text: str, retrieved_ids: list[str] | None = None, **kw) -> "MemoryBlock":
        return cls(
            arm=arm,
            text=text,
            tokens=count_tokens(text),
            sha=sha256_text(text),
            retrieved_ids=retrieved_ids or [],
            **kw,
        )


class ArmAssembler:
    """Caches the per-size corpus and per-arm claim index so a 540-run grid does
    not re-read and re-tokenize the corpus 540 times."""

    def __init__(self, cfg: Config, size: str) -> None:
        self.cfg = cfg
        self.size = size
        self._corpus: str | None = None
        self._arm_a_cache: dict[str, MemoryBlock] = {}
        self._bm25_claims: dict[str, tuple[list[dict], BM25]] = {}
        self._bm25_chunks: tuple[list, BM25] | None = None

    # -- arm A ---------------------------------------------------------
    def _arm_a(self, model: ModelSpec) -> MemoryBlock:
        # Arm A's block is identical for every question, so it is built once
        # per (size, model) and reused. That is not an optimisation detail: an
        # identical byte-for-byte prefix is what makes D6's prompt caching work
        # at all.
        key = model.alias
        if key in self._arm_a_cache:
            return self._arm_a_cache[key]
        if self._corpus is None:
            self._corpus = corpus_text(self.size)
        text = self._corpus
        # Leave headroom for the template, the question, and the output.
        budget = model.max_input_tokens - 8_000
        truncated = False
        if count_tokens(text) > budget:
            text = truncate_to_tokens(text, budget)
            truncated = True
        block = MemoryBlock.build("A", text, truncated_for_model=truncated)
        self._arm_a_cache[key] = block
        return block

    # -- arm B ---------------------------------------------------------
    def _arm_b(self, question: str) -> MemoryBlock:
        require_arm_b()  # refuses rather than silently running BM25-only
        if self._bm25_chunks is None:
            chunks = chunk_documents(corpus_documents(self.size))
            self._bm25_chunks = (chunks, BM25([tokenize(c.text) for c in chunks]))
        chunks, index = self._bm25_chunks
        picked, ids, used = [], [], 0
        for i, _score in index.rank(question, 100):
            c = chunks[i]
            if used + c.tokens > self.cfg.memory_token_ceiling:
                continue
            picked.append(f"<passage id=\"{c.chunk_id}\" path=\"{c.path}\">\n{c.text}\n</passage>")
            ids.append(c.chunk_id)
            used += c.tokens
        return MemoryBlock.build("B", "\n\n".join(picked), ids)

    # -- arms C ---------------------------------------------------------
    def _arm_c(self, arm: str, question: str) -> MemoryBlock:
        if arm not in self._bm25_claims:
            claims = load_claims(arm)
            if not claims:
                raise HarnessError(f"arm {arm} has no claims")
            self._bm25_claims[arm] = (claims, BM25([tokenize(c["text"]) for c in claims]))
        claims, index = self._bm25_claims[arm]

        picked, ids, used = [], [], 0
        for i, _score in index.rank(question, 200):
            c = claims[i]
            ev = c["evidence"][0]
            rendered = (
                f"<claim id=\"{c['claim_id']}\" kind=\"{c['kind']}\">\n"
                f"{c['text']}\n"
                f"  <evidence source=\"{ev['source_type']}\" path=\"{ev['path']}\" url=\"{ev['url']}\">"
                f"{ev['quote']}</evidence>\n"
                f"</claim>"
            )
            t = count_tokens(rendered)
            if used + t > self.cfg.memory_token_ceiling:
                continue
            picked.append(rendered)
            ids.append(c["claim_id"])
            used += t
        return MemoryBlock.build(arm, "\n\n".join(picked), ids)

    # -- entry point -----------------------------------------------------
    def assemble(self, arm: str, question: str, model: ModelSpec) -> MemoryBlock:
        if arm == "A0":
            block = MemoryBlock.build("A0", "")
        elif arm == "A":
            block = self._arm_a(model)
        elif arm == "B":
            block = self._arm_b(question)
        elif arm in ("C_both", "C_doc", "C_exp"):
            block = self._arm_c(arm, question)
        else:
            raise HarnessError(f"unknown arm {arm!r}")

        if arm != "A0" and arm != "A" and block.tokens > self.cfg.memory_token_ceiling:
            # Should be unreachable; if it fires, the ablation is void and the
            # run must stop rather than record a volume win as a memory win.
            raise HarnessError(
                f"arm {arm} produced a {block.tokens}-token block against a "
                f"{self.cfg.memory_token_ceiling} ceiling. §3 and §8 make the shared "
                f"ceiling a hard constraint; a run past it is not interpretable."
            )
        return block
