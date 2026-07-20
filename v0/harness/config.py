"""Experiment configuration.

The MVE (§9) and the full design (§6) are the same code path with different
JSON. §9 requires that a winning MVE mandates the full run, so the full config
must not be a rewrite — it is `config/full.json`.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from .util import V0_ROOT, HarnessError, read_json

# Arms are defined once. A config selects a subset.
ALL_ARMS = ["A0", "A", "B", "C_both", "C_doc", "C_exp"]

# Which claim evidence sources each C-arm may draw on (§8).
ARM_SOURCE_FILTER = {
    "C_doc": ["code", "committed_docs"],
    "C_exp": ["review_discussion", "issue_thread", "revert", "incident"],
    "C_both": [
        "code",
        "committed_docs",
        "review_discussion",
        "issue_thread",
        "revert",
        "incident",
    ],
}

# Which prompt template each arm uses. B and C share the "memory block" shape
# but not the template — §3 requires one fixed template per arm.
ARM_TEMPLATE = {
    "A0": "a0",
    "A": "a",
    "B": "b",
    "C_both": "c",
    "C_doc": "c",
    "C_exp": "c",
}


@dataclass(frozen=True)
class ModelSpec:
    alias: str  # local label used in run paths
    provider: str  # "anthropic" | "mock"
    model_id: str  # provider model id
    family: str  # for the two-families requirement (D5 / T3)
    tier: str  # "frontier" | "mid"
    max_input_tokens: int
    caching: bool  # D6: caching ON for arm A


@dataclass(frozen=True)
class Thresholds:
    """Every number here is pre-registered (§6). The analysis reads them from
    the frozen config; it does not hard-code them."""

    proceed_delta_points: float = 10.0
    amber_upper_points: float = 10.0
    cost_multiple_max: float = 3.0
    kappa_min: float = 0.70
    flip_rate_max: float = 0.25
    recall_at_k_min: float = 0.60
    audit_flag_rate_max: float = 0.15
    ablation_delta_points: float = 7.0  # §8 v1 gate, full design only
    bootstrap_resamples: int = 10_000
    ci_level: float = 0.95


@dataclass(frozen=True)
class Config:
    name: str
    arms: list[str]
    n_eval_tasks: int
    n_dev_tasks: int
    runs_per_task: int
    models: list[ModelSpec]
    corpus_sizes: list[str]
    primary_size: str
    memory_token_ceiling: int
    temperature: float
    seeds: list[int]
    decoy_share: float
    amortization_n: int
    family_shares: dict[str, float]
    thresholds: Thresholds
    may_greenlight: bool  # D8 / §9: the MVE may kill but not green-light
    raw: dict[str, Any] = field(default_factory=dict, repr=False)

    # ------------------------------------------------------------------
    def arm_enabled(self, arm: str) -> bool:
        return arm in self.arms

    def model(self, alias: str) -> ModelSpec:
        for m in self.models:
            if m.alias == alias:
                return m
        raise HarnessError(f"unknown model alias {alias!r}; have {[m.alias for m in self.models]}")

    def validate(self) -> None:
        unknown = set(self.arms) - set(ALL_ARMS)
        if unknown:
            raise HarnessError(f"unknown arms in config: {sorted(unknown)}")
        if "A0" not in self.arms:
            raise HarnessError("A0 is mandatory: it is the contamination floor and the task filter (§4 step 4)")
        if self.primary_size not in self.corpus_sizes:
            raise HarnessError(f"primary_size {self.primary_size!r} not in corpus_sizes {self.corpus_sizes}")
        if len(self.seeds) < self.runs_per_task:
            raise HarnessError(f"need >= {self.runs_per_task} seeds, have {len(self.seeds)}")
        if not 0.0 < self.decoy_share < 0.5:
            raise HarnessError("decoy_share must be a fraction in (0, 0.5)")
        share_sum = sum(self.family_shares.values())
        if abs(share_sum - 1.0) > 1e-6:
            raise HarnessError(f"family_shares must sum to 1.0, got {share_sum}")
        if "abstention_decoy" in self.family_shares:
            if abs(self.family_shares["abstention_decoy"] - self.decoy_share) > 1e-6:
                raise HarnessError("family_shares['abstention_decoy'] must equal decoy_share")
        # D5: two families required to green-light.
        if self.may_greenlight and len({m.family for m in self.models}) < 2:
            raise HarnessError(
                "a config that may green-light needs two model families (D5). "
                "Set may_greenlight=false or add a second family."
            )


def load_config(path: str | Path) -> Config:
    p = Path(path)
    if not p.is_absolute():
        candidate = V0_ROOT / "config" / p.name
        p = candidate if candidate.exists() else p
    raw = read_json(p)
    cfg = Config(
        name=raw["name"],
        arms=raw["arms"],
        n_eval_tasks=raw["n_eval_tasks"],
        n_dev_tasks=raw["n_dev_tasks"],
        runs_per_task=raw["runs_per_task"],
        models=[ModelSpec(**m) for m in raw["models"]],
        corpus_sizes=raw["corpus_sizes"],
        primary_size=raw["primary_size"],
        memory_token_ceiling=raw["memory_token_ceiling"],
        temperature=raw.get("temperature", 0.0),
        seeds=raw["seeds"],
        decoy_share=raw["decoy_share"],
        amortization_n=raw.get("amortization_n", 200),
        family_shares=raw["family_shares"],
        thresholds=Thresholds(**raw.get("thresholds", {})),
        may_greenlight=raw["may_greenlight"],
        raw=raw,
    )
    cfg.validate()
    return cfg


def config_sha(cfg: Config) -> str:
    from .util import sha256_json

    return sha256_json(cfg.raw)


def default_config_path() -> Path:
    return V0_ROOT / "config" / "mve.json"


if __name__ == "__main__":  # quick self-check
    import sys

    c = load_config(sys.argv[1] if len(sys.argv) > 1 else default_config_path())
    print(json.dumps({"name": c.name, "arms": c.arms, "sha": config_sha(c)}, indent=2))
