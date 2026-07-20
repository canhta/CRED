"""Provider registry."""

from __future__ import annotations

from ..util import HarnessError
from .base import Completion, Provider
from .mock import MockProvider

__all__ = ["Completion", "Provider", "MockProvider", "get_provider"]

_CACHE: dict[str, Provider] = {}


def get_provider(name: str) -> Provider:
    if name in _CACHE:
        return _CACHE[name]
    if name == "mock":
        p: Provider = MockProvider()
    elif name == "anthropic":
        from .anthropic_provider import AnthropicProvider

        p = AnthropicProvider()
    else:
        raise HarnessError(f"unknown provider {name!r} (have: mock, anthropic)")
    _CACHE[name] = p
    return p
