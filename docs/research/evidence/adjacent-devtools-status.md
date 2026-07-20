# Adjacent AI Devtool Companies — Status Scan

## Provenance warning

**This document did not originate from CRED's commissioned research.** It
arrived from a background task (`Dead devtools research A2`) that was not
launched as part of this discovery effort, most likely leaking from an unrelated
session.

It is retained because the findings are directly relevant, but it has **not been
independently verified** by CRED's own research process. Treat every claim here
as needing confirmation before it informs a decision. Confidence markers from the
original are preserved.

- **Captured:** 2026-07-20
- **Status:** Unverified third-party input

---

## The finding that matters most: Tessl

**Tessl has pivoted directly into CRED's proposed wedge.**

- Raised **$125M** at $500M+ valuation (Nov 2024) on **spec-driven development**
  — "specs are durable, code is disposable."
  Investors: Index, GV, boldstart, Accel.
- Founder **Guy Podjarny** previously built **Snyk to $8B**, and before that
  founded Blaze (acquired by Akamai).
- As of July 2026, the phrase "spec-driven development" **no longer appears on
  the Tessl homepage**.
- Tessl now sells an **"Agent Enablement Platform"**:
  - **Platform** — governance
  - **Agent**
  - **Registry** — 3,000+ skills
  - Integrations with **Claude Code, Cursor, Copilot, Gemini**
- Shipping actively: Skill Inventory (2026-06-17), Tessl Review (2026-06-23),
  Tessl Academy preview (2026-07-03).
- No disclosed funding round since Nov 2024.
- Their own about page now concedes "agents are commoditizing coding."

Source: [tessl.io](https://tessl.io/), [Tessl blog](https://tessl.io/blog),
[TechCrunch](https://techcrunch.com/tag/tessl/)

### Why this is significant for CRED

**Confirming.** A proven founder with a $8B prior outcome and $125M in capital
**abandoned their founding thesis to move into multi-tool agent harness
governance and skill distribution.** Companies do not pivot at that cost into
imaginary markets. This is expensive, independent validation that the
enablement-lead persona and the harness-governance need are real — stronger
evidence than anything derivable from reading competitor source code.

**Disconfirming.** The space is **not empty**. The working assumption that the
enablement-lead persona is unserved is now partially refuted.

### Consequences for positioning

1. Any "nobody is doing this" framing must be **removed**. It is false and would
   be caught immediately.
2. The defensible position is **open source, local-first, artifact-lives-in-git**
   — set against a closed, top-down enterprise platform. This is the shape that
   won for Langfuse vs LangSmith and n8n vs Zapier.
3. **Kill criterion sharpened:** if Tessl open-sources its Registry and rule
   compilation, CRED's differentiation narrows substantially and the wedge
   should be re-examined.

---

## Cross-cutting patterns

1. **The autocomplete / IDE-plugin layer is dead as a standalone business.**
   Augment sunset completions (2026-03-31) and launched Cosmos; Tabnine shrank
   to 68 staff after 18% layoffs; Windsurf was dismembered. Every survivor moved
   **up** to agent orchestration or **down** to deterministic transformation.
   CRED targets the upward layer — correct direction, crowded.

2. **The AI-devtool exit is now the reverse-acquihire.** Google/Windsurf ($2.4B
   licence + talent, no equity) and Honeycomb/Grit (undisclosed, product sunset)
   are the same pattern at different scales: talent and IP get priced, the
   company does not.

3. **Apparent capital in this market is overstated.** Two of the best-funded
   names left the category — Reflection AI (~$2.13B) became a frontier lab,
   Magic.dev ($515M) has been publicly silent since Aug 2024.

4. **The healthiest company in the set is the least fashionable.** Moderne
   (~$50M raised, Gartner Visionary May 2026) does deterministic AST refactoring
   — the one area where LLMs complement rather than replace. Consistent with the
   Graphiti lesson: *LLM nominates, deterministic code decides.*

---

## Status table

| Company | Status | Funding | Key event | Date | Confidence |
|---|---|---|---|---|---|
| Codeium/Windsurf | Dismembered | ~$243M | OpenAI $3B collapsed → Google $2.4B → Cognition | Jul 11–14, 2025 | High |
| Augment Code | Alive, pivoted | ~$252M | Killed completions; launched Cosmos | Mar 31 / Jun 3, 2026 | High |
| Magic.dev | Alive, stalled | $515M | Silent since Aug 2024 | — | High |
| Poolside | Alive, capital-heavy | ~$1.6B+ | Nvidia $1B at $12B | Oct 30, 2025 | Partial |
| Tabnine | Alive, declining | ~$102M | 18% layoffs; 68 staff | 2024 / May 2026 | High |
| Grit.io | Acquired | — | Honeycomb; product sunset | Apr 9, 2025 | High |
| Moderne | Healthy | ~$50M | $30M Series B; Gartner Visionary | Feb 2025 / May 2026 | High |
| Sema4.ai | Alive, quiet | Unknown | Active repos, on-prem focus | Jul 14, 2026 | Partial |
| Reflection AI | Alive, pivoted out | ~$2.13B | $2B at $8B; now frontier lab | Oct 9, 2025 | High |
| **Tessl** | **Alive, thesis abandoned** | **$125M** | **Spec-driven → agent governance** | **Nov 2024 → 2026** | **High** |

---

## Verification gaps

- **Poolside 2026** — a claimed Series C collapse / CoreWeave exit is
  single-sourced to a low-credibility outlet and could not be corroborated.
  Requires tier-1 financial press before use.
- **Sema4.ai funding** — no figures obtainable; site blocks automated access.
- Magic.dev's ~$2M ARR and Windsurf's ~$243M total raise are secondary-source
  figures, not company-confirmed.
- **Tessl's product details should be re-verified directly** before they inform
  any final scope decision, given this document's provenance.
