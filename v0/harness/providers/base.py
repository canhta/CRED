"""Provider interface.

Every arm, the extractor, and the judge go through this one interface so that
token accounting, the provider version string (T3), and `finish_reason` are
captured uniformly. A provider that does not report its resolved model version
is not usable here — T3 requires it per run.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Protocol


@dataclass
class Completion:
    text: str
    finish_reason: str  # "stop" | "length" | "refusal" | "error" | ...
    input_tokens: int
    output_tokens: int
    cache_write_tokens: int = 0
    cache_read_tokens: int = 0
    provider_model_version: str = ""  # T3: what actually served the request
    latency_ms: int = 0
    error: str | None = None
    raw: dict = field(default_factory=dict, repr=False)

    @property
    def truncated(self) -> bool:
        """§11: a run that ended on `length` is a truncated answer, not a wrong
        one. It is excluded with a reported count."""
        return self.finish_reason in ("length", "max_tokens")


class Provider(Protocol):
    name: str

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
        """One non-streaming completion.

        `cache_system` requests prompt caching on the system block. D6 requires
        it ON for arm A: the corpus is identical across queries and cacheable,
        and disabling it would be a strawman.
        """
        ...
