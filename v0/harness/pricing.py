"""Cost accounting.

T8: the analysis must be re-priceable. Raw token counts are logged on every run
record; this module only turns them into dollars, and every price carries the
date it was read and where from. If prices move, edit `PRICES`, re-run
`analysis/analyze.py`, and every dollar figure updates. Nothing downstream
stores a price.

§5 pre-registers *both* amortization policies:
  - marginal   = per-query input + output, cache-aware
  - amortized  = marginal + (memory build cost / N queries), N = 200 by default
"""

from __future__ import annotations

from dataclasses import dataclass

from .util import HarnessError

PRICE_SOURCE = (
    "Anthropic published list prices, read 2026-07-20 via the bundled `claude-api` "
    "skill model table (cached 2026-06-24). Verify against "
    "https://platform.claude.com/docs/en/pricing before publishing dollar figures."
)


@dataclass(frozen=True)
class Price:
    """USD per 1M tokens."""

    input: float
    output: float
    cache_write_multiplier: float = 1.25  # 5-minute TTL
    cache_read_multiplier: float = 0.10


# Keyed by provider model id.
PRICES: dict[str, Price] = {
    "claude-opus-4-8": Price(input=5.00, output=25.00),
    "claude-opus-4-7": Price(input=5.00, output=25.00),
    "claude-sonnet-5": Price(input=3.00, output=15.00),
    "claude-sonnet-4-6": Price(input=3.00, output=15.00),
    "claude-haiku-4-5": Price(input=1.00, output=5.00),
    # The mock provider is free. Logged so cost code is exercised end to end.
    "mock-1": Price(input=0.0, output=0.0),
}


def price_for(model_id: str) -> Price:
    if model_id not in PRICES:
        raise HarnessError(
            f"no price entry for model {model_id!r}. Add one to harness/pricing.py "
            f"with a dated source rather than guessing — T8 requires the cost "
            f"model be auditable."
        )
    return PRICES[model_id]


def marginal_usd(
    model_id: str,
    *,
    input_tokens: int,
    output_tokens: int,
    cache_write_tokens: int = 0,
    cache_read_tokens: int = 0,
) -> float:
    """Cost of one query.

    `input_tokens` is the uncached remainder only — the provider reports cached
    reads and writes separately, and they are priced separately. Summing them
    into `input_tokens` would overcharge the cached arm, which is exactly the
    accounting error D6 exists to prevent.
    """
    p = price_for(model_id)
    per = 1_000_000.0
    return (
        input_tokens * p.input / per
        + cache_write_tokens * p.input * p.cache_write_multiplier / per
        + cache_read_tokens * p.input * p.cache_read_multiplier / per
        + output_tokens * p.output / per
    )


def build_usd(
    model_id: str, *, input_tokens: int, output_tokens: int
) -> float:
    """Cost of a memory-construction pass. Reported separately from query cost
    and never folded into latency (§5)."""
    return marginal_usd(model_id, input_tokens=input_tokens, output_tokens=output_tokens)


def amortized_usd(marginal: float, build_total_usd: float, n_queries: int) -> float:
    """Policy 2 of the two pre-registered policies. `n_queries` is the
    denominator; the design fixes it at 200 and requires both policies be
    published so the choice cannot be made after seeing the data."""
    if n_queries <= 0:
        raise HarnessError("amortization denominator must be positive")
    return marginal + build_total_usd / n_queries


def price_table_snapshot() -> dict:
    """Embedded in results so a reader can re-derive every dollar figure."""
    return {
        "source": PRICE_SOURCE,
        "usd_per_1m_tokens": {
            k: {
                "input": v.input,
                "output": v.output,
                "cache_write_multiplier": v.cache_write_multiplier,
                "cache_read_multiplier": v.cache_read_multiplier,
            }
            for k, v in PRICES.items()
        },
    }
