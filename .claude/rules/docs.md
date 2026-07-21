---
paths:
  - "docs/**/*"
  - "README.md"
  - "AGENTS.md"
  - "CLAUDE.md"
---

# Documentation rules

Applies to everything under `docs/`, plus `README.md` and `AGENTS.md`.

The documentation is currently the entire product. Treat it as the deliverable
it is.

## 1. Evidence standard

**Never state a fact you did not verify.** A research agent on this project
fabricated citations before self-retracting; the fallout is recorded in
`docs/README.md` and in every affected evidence file. That incident is why this
section is first.

Every external claim carries one of three labels:

- **VERIFIED** — you ran the command, read the source file, or fetched the URL.
  Include the command, the `file:line`, or the URL.
- **UNVERIFIED** — plausible, load-bearing, not yet checked. Say what would
  check it.
- **FALSIFIED** — checked and found false. **Keep it in the document.** Deleting
  a falsified claim means the next person re-derives it and believes it.

Rules that follow from this:

- Numbers require a source in the same document. No orphan statistics.
- Claims about a codebase cite `path/to/file.go:120`, not the project's README.
- Claims about a competitor's product cite a URL you actually fetched. Marketing
  copy is evidence of what they *claim*, never of what they *do* — label it as
  such. Memco's on-prem claim was accepted from a marketing line and had to be
  retracted; it had zero public artifacts behind it.
- If a document's findings arrived outside a commissioned task, open it with a
  **provenance warning**.
- When you correct yourself, correct on the record. Write what you previously
  claimed, what changed it, and what the claim is now.

## 2. Structure

**Decisions** (`decision-log.md`) use `D-NNN`, sequential, never renumbered, and
carry: Date · Status · Decision · Reasoning · **What this rules out** · What
this forces · Open tension, where one exists.

"What this rules out" is mandatory. A decision that forecloses nothing is not a
decision.

**Spikes** (`docs/research/spikes/`) carry: the question and why it matters ·
method, with exact commands · results, with real output · what is unverified and
why · **verdict**, stated as a decision · what would change the verdict.

A spike that ends without a verdict is unfinished.

**Evidence** (`docs/research/evidence/`) is a scan, not an argument. Record what
is there. Interpretation belongs in `synthesis.md` or `design-advantages.md`.

## 3. Versioning

**The PRD is one living version with no version history.** No "v2 of this
section", no changelog, no superseded blocks. Edit it in place so it always
reads as the current instruction to an implementer. Version history lives in
git; the decision log carries the reasoning.

The decision log is the opposite — append-only. Supersede an entry with a new
one that references it. Never edit a decision's substance after the fact.

## 4. Voice

Write for an implementer who is skeptical and short on time.

- Direct prose. State the thing, then support it.
- No marketing tone. Never "powerful", "seamless", "revolutionary",
  "game-changing", "cutting-edge", "robust".
- No hedging that carries no information. "may potentially be able to" is noise.
  Either it does or you do not know — and "I do not know" is a legitimate,
  useful sentence.
- Prefer a table when comparing more than two things on more than two axes.
  Prefer prose when the reasoning matters more than the comparison.
- Name the trade-off. Every recommendation costs something; if a section has no
  cost in it, the analysis is incomplete.
- Criticism of real companies is allowed and expected, but must be specific,
  sourced, and about the product — never about the people.

## 5. Mechanics

- Markdown, wrapped at **80 columns**. Do not reflow unrelated paragraphs when
  editing; it destroys the diff.
- Sentence case for headings.
- Every internal link is relative and must resolve. Verify after any rename or
  deletion — a previous cleanup left six broken links across four files.
- The product is **CRED**, uppercase. Env vars `CRED_*`, OTel namespace `cred.*`,
  binary `cred`. The prior name was `SHIFT`; if you find it, it is a leftover.
- One canonical description, used everywhere: **evidence-governed memory for AI
  agents**. Tagline: *a claim lives only while its evidence does.* If a
  document describes CRED differently, that is drift — fix it, do not fork it.
- Third-party papers and PDFs are never committed. `.gitignore` excludes
  `docs/*.pdf`. Cite them; do not redistribute them.

## 6. When adding a document

Add it to the index in `docs/README.md` in the same change. An unindexed
document is one nobody will find, which makes writing it wasted work.
