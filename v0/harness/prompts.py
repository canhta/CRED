"""Prompt template loading and hashing.

T9: one fixed template per arm, written before any result, published with
hashes. This module is the only way a template reaches a run — nothing builds
prompt text inline, so a template cannot be edited mid-experiment without the
`prompt_template_sha` on the run record changing.
"""

from __future__ import annotations

import re
from pathlib import Path

from .config import ARM_TEMPLATE
from .util import V0_ROOT, HarnessError, sha256_text

PROMPTS_DIR = V0_ROOT / "prompts"
_PLACEHOLDER = re.compile(r"\{\{([A-Z_]+)\}\}")


def template_path(name: str) -> Path:
    p = PROMPTS_DIR / f"{name}.md"
    if not p.exists():
        raise HarnessError(f"missing prompt template: {p}")
    return p


def load_template(name: str) -> str:
    return template_path(name).read_text(encoding="utf-8")


def template_sha(name: str) -> str:
    return sha256_text(load_template(name))


def arm_template_name(arm: str) -> str:
    if arm not in ARM_TEMPLATE:
        raise HarnessError(f"no template mapped for arm {arm!r}")
    return ARM_TEMPLATE[arm]


def render(name: str, values: dict[str, str]) -> str:
    """Substitute {{PLACEHOLDER}}s. Fails loudly on an unfilled placeholder —
    a prompt that silently ships the literal text `{{MEMORY_BLOCK}}` to a model
    is a void run, and finding that out during analysis is too late."""
    text = load_template(name)
    missing: list[str] = []

    def sub(m: re.Match[str]) -> str:
        key = m.group(1)
        if key not in values:
            missing.append(key)
            return m.group(0)
        return values[key]

    out = _PLACEHOLDER.sub(sub, text)
    if missing:
        raise HarnessError(f"template {name!r}: unfilled placeholders {sorted(set(missing))}")
    return out


def all_template_hashes() -> dict[str, str]:
    """Published alongside results (§12)."""
    return {p.stem: sha256_text(p.read_text(encoding="utf-8")) for p in sorted(PROMPTS_DIR.glob("*.md"))}
