"""Deterministic offline provider.

Purpose: make every code path runnable and testable without spending money or
holding credentials. It is NOT a model. It produces syntactically valid but
semantically empty answers, so any number produced under the mock provider is a
plumbing check, never a result. `analysis/analyze.py` refuses to emit a verdict
from mock runs.
"""

from __future__ import annotations

import hashlib
import json
import time

from ..util import count_tokens
from .base import Completion

MOCK_MODEL_ID = "mock-1"
MOCK_VERSION = "mock-1@deterministic-v1"

# The literal the abstention prompt asks for. The mock emits it for a fixed
# pseudorandom subset so the abstention/confabulation path is exercised.
ABSTAIN_LITERAL = "The corpus does not say."


class MockProvider:
    name = "mock"

    def __init__(self, abstain_rate: float = 0.15) -> None:
        self.abstain_rate = abstain_rate

    # ------------------------------------------------------------------
    def _respond(self, system: str, user: str, digest: str, bucket: float) -> str:
        """Shape-correct output for each stage that parses model output.

        The harness's JSON-consuming stages (drafter, extractor, judge) would
        otherwise all fail under the mock and the pipeline could never be
        exercised offline. The *shape* is real; the *content* is deliberately
        marked MOCK so that no output of this provider can be mistaken for a
        measurement.
        """
        if "JSON only" not in system:
            if bucket < self.abstain_rate:
                return ABSTAIN_LITERAL
            return (
                "MOCK ANSWER — no model was called. This text exists to exercise the "
                f"harness end to end. Deterministic tag: {digest[:12]}."
            )
        if "extract durable claims" in system:
            if bucket < 0.2:
                return '{"claims": []}'
            return json.dumps(
                {
                    "claims": [
                        {
                            "text": f"MOCK CLAIM {digest[:8]}: this project made a decision recorded in the excerpt.",
                            "evidence_quote": "MOCK evidence quote.",
                            "kind": "decision",
                        }
                    ]
                }
            )
        if "draft evaluation tasks" in system:
            if "abstention decoy" in user:
                return json.dumps(
                    {
                        "usable": True,
                        "question": f"MOCK DECOY {digest[:8]}: what was never recorded?",
                        "why_unrecorded": "MOCK",
                        "search_terms": [f"zzmock{digest[:8]}"],
                    }
                )
            return json.dumps(
                {
                    "usable": True,
                    # Vary the family so the composition step in `freeze` is
                    # actually exercised rather than trivially under-supplied.
                    "family": [
                        "decision_rationale", "unwritten_convention",
                        "cross_cutting_context", "failure_recall",
                    ][int(digest[8:10], 16) % 4],
                    "question": f"MOCK QUESTION {digest[:8]}: why was this approach chosen?",
                    "gold_answer": f"MOCK GOLD {digest[:8]}.",
                    "checkpoints": [
                        {"id": "c1", "text": "MOCK checkpoint",
                         "match": {"type": "any_of", "values": [f"MOCK GOLD {digest[:8]}"]}}
                    ],
                    "evidence_quote": "MOCK evidence quote.",
                }
            )
        if "grade one atomic checkpoint" in system:
            return json.dumps({"hit": bucket > 0.5, "reason": "MOCK grade"})
        return "{}"

    def complete(
        self,
        *,
        model_id: str,
        system: str,
        user: str,
        max_tokens: int,
        temperature: float,
        seed: int | None = None,
        cache_system: bool = False,
    ) -> Completion:
        t0 = time.monotonic()
        digest = hashlib.sha256(f"{model_id}|{seed}|{system}|{user}".encode()).hexdigest()
        bucket = int(digest[:8], 16) / 0xFFFFFFFF

        text = self._respond(system, user, digest, bucket)

        in_tok = count_tokens(system) + count_tokens(user)
        out_tok = count_tokens(text)
        cache_read = in_tok if cache_system else 0
        return Completion(
            text=text,
            finish_reason="stop",
            input_tokens=0 if cache_system else in_tok,
            output_tokens=out_tok,
            cache_write_tokens=0,
            cache_read_tokens=cache_read,
            provider_model_version=MOCK_VERSION,
            latency_ms=int((time.monotonic() - t0) * 1000),
            raw={"mock": True, "digest": digest},
        )
