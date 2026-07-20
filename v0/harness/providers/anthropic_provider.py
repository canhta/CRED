"""Anthropic provider, via the official `anthropic` Python SDK.

Install: `pip install anthropic`. The import is lazy so the rest of the harness
— mining, corpus building, analysis — runs on a clean stdlib-only machine.

Three points where this file deviates from a naive reading of the design, each
deliberate and each recorded in `v0/README.md`:

1. **Temperature.** §6 pre-registers temperature 0. Current models
   (Sonnet 5, Opus 4.7/4.8) *reject* non-default sampling parameters with a
   400. Sending `temperature=0` would fail every request. We therefore omit the
   parameter on those models and log `temperature: null` on the run record. The
   pre-registered intent — no deliberate sampling variance — is preserved; the
   knob no longer exists.

2. **Seeds.** The Messages API exposes no seed. §6 says "seeds fixed where the
   API exposes them"; it does not. The `seed` on the run record is the
   harness's run-index label, used for bookkeeping and never sent. This is why
   §7's flip-rate check exists.

3. **Thinking.** The design does not specify it. Adaptive thinking is ON by
   default on Sonnet 5 when the field is omitted, which would add uncontrolled
   token spend and an extra source of run-to-run variance to a comparison that
   is supposed to be about the memory block. We set it explicitly — default
   disabled — and record the setting in the run record so the choice is
   visible rather than accidental.
"""

from __future__ import annotations

import os
import time
from typing import Any

from ..util import HarnessError
from .base import Completion

# Models that reject temperature / top_p / top_k with a 400.
NO_SAMPLING_PARAMS = {
    "claude-fable-5",
    "claude-mythos-5",
    "claude-opus-4-8",
    "claude-opus-4-7",
    "claude-sonnet-5",
}

# "disabled" | "adaptive". Override with CRED_V0_THINKING.
DEFAULT_THINKING = os.environ.get("CRED_V0_THINKING", "disabled")


class AnthropicProvider:
    name = "anthropic"

    def __init__(self, thinking: str | None = None, max_retries: int = 4) -> None:
        try:
            import anthropic  # noqa: F401
        except ImportError as exc:  # pragma: no cover - environment dependent
            raise HarnessError(
                "the `anthropic` package is not installed. Run `pip install anthropic`, "
                "or use the mock provider to exercise the harness offline."
            ) from exc
        import anthropic

        # Zero-arg construction resolves ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN,
        # or an `ant auth login` profile. An unset API key does not mean there
        # are no credentials.
        self._client = anthropic.Anthropic(max_retries=max_retries)
        self.thinking = thinking or DEFAULT_THINKING

    # ------------------------------------------------------------------
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
        import anthropic

        system_blocks: list[dict[str, Any]] = [{"type": "text", "text": system}]
        if cache_system:
            # D6: caching ON for the long-context arm. The corpus is identical
            # across queries; the breakpoint goes on the last system block so
            # the whole corpus is one cacheable prefix. The question lives in
            # the user turn, after the breakpoint, so it never invalidates it.
            system_blocks[-1]["cache_control"] = {"type": "ephemeral", "ttl": "1h"}

        kwargs: dict[str, Any] = {
            "model": model_id,
            "max_tokens": max_tokens,
            "system": system_blocks,
            "messages": [{"role": "user", "content": user}],
        }
        if model_id not in NO_SAMPLING_PARAMS:
            kwargs["temperature"] = temperature
        if self.thinking == "adaptive":
            kwargs["thinking"] = {"type": "adaptive"}
        elif self.thinking == "disabled":
            kwargs["thinking"] = {"type": "disabled"}

        t0 = time.monotonic()
        try:
            resp = self._client.messages.create(**kwargs)
        except anthropic.APIStatusError as exc:
            return Completion(
                text="",
                finish_reason="error",
                input_tokens=0,
                output_tokens=0,
                provider_model_version=model_id,
                latency_ms=int((time.monotonic() - t0) * 1000),
                error=f"{type(exc).__name__}({exc.status_code}): {exc}",
            )
        except anthropic.APIConnectionError as exc:
            return Completion(
                text="",
                finish_reason="error",
                input_tokens=0,
                output_tokens=0,
                provider_model_version=model_id,
                latency_ms=int((time.monotonic() - t0) * 1000),
                error=f"APIConnectionError: {exc}",
            )
        latency_ms = int((time.monotonic() - t0) * 1000)

        text = "".join(b.text for b in resp.content if getattr(b, "type", None) == "text")
        usage = resp.usage
        return Completion(
            text=text,
            finish_reason=resp.stop_reason or "unknown",
            input_tokens=getattr(usage, "input_tokens", 0) or 0,
            output_tokens=getattr(usage, "output_tokens", 0) or 0,
            cache_write_tokens=getattr(usage, "cache_creation_input_tokens", 0) or 0,
            cache_read_tokens=getattr(usage, "cache_read_input_tokens", 0) or 0,
            # T3: the resolved model string the provider actually served with.
            provider_model_version=resp.model,
            latency_ms=latency_ms,
            raw={
                "stop_reason": resp.stop_reason,
                "request_id": getattr(resp, "_request_id", None),
                "thinking": self.thinking,
                "temperature_sent": model_id not in NO_SAMPLING_PARAMS,
            },
        )


def credentials_present() -> bool:
    """Cheap check the CLI uses to decide whether a live run is even possible.
    Deliberately does not prompt for anything."""
    if os.environ.get("ANTHROPIC_API_KEY") or os.environ.get("ANTHROPIC_AUTH_TOKEN"):
        return True
    for candidate in (
        os.path.expanduser("~/.config/anthropic/credentials"),
        os.path.expanduser("~/.config/anthropic"),
    ):
        if os.path.exists(candidate):
            return True
    return False
