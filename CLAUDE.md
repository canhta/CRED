# CRED — working instructions

**C**laims — **R**ights — **E**vidence — **D**erivation.
Evidence-governed memory for AI agents. A claim lives only while its evidence
does.

## Current phase

**Pre-implementation.** There is no product code in this repository and none
should be added yet. Discovery is complete; the next deliverables are an
experiment and two gating spikes, not features.

Before writing any implementation code, check that the gate for it has actually
cleared:

| Gate | Where it is decided | Status |
|---|---|---|
| Does retrieved memory beat plain long context? | `docs/research/spikes/v0-experiment-design.md` | Designed, not run |
| Does a pure-Go embedding path exist without CGO? | `docs/research/spikes/go-embeddings-tokenizer.md` | **Cleared**, with a throughput condition |

If either gate fails, the correct response is to change the plan and record why
— not to route around it.

## Non-negotiables

These come from `docs/product/prd.md` section 4. They are design laws, not
preferences. Code that violates one is wrong even if it passes tests.

- **L1** No claim without evidence. Human attestation counts; orphan claims do
  not.
- **L2** The model nominates, code decides. LLM output selects candidates;
  deterministic code makes the call. This covers extraction, not just
  reconciliation.
- **L3** Invalidation is deterministic, anchors are semantic.
- **L4** Confirmation happens inside the task, never in a queue.
- **L5** Access control is evaluated at recall, fails closed, and is the
  **intersection**: `claim.acl ⊆ ⋂(evidence_i.acl)`. Never the union.
- **L6** Everything is bi-temporal, and evaluated per principal.
- **L7** One database. PostgreSQL 17+ with pgvector.
- **L8** Ingested content is untrusted input, never instruction.

Read the PRD for the reasoning behind each. Do not restate them here from
memory — they are maintained in the PRD, and this list is a pointer.

## Rules

- `.claude/rules/docs.md` — documentation and evidence standards. Loads
  automatically when Claude works with files under `docs/`, `README.md`, or
  `CLAUDE.md`; not imported here to avoid double-loading it.

## Repository shape

```
docs/product/prd.md            The implementation-facing requirements
docs/research/synthesis.md     What discovery found; read first
docs/research/decision-log.md  Decisions, reasoning, what each rules out
docs/research/evidence/        Repo scans and market evidence
docs/research/spikes/          Technical spikes
```

## How to work here

**Persist every scan and spike into `docs/`.** A finding that exists only in a
conversation is a finding that will be lost and re-paid for. This is a standing
instruction from the maintainer, not a suggestion.

**Disconfirming evidence is the valuable kind.** This project has repeatedly
found evidence against its own thesis and recorded it rather than smoothing it
over — the seller/buyer asymmetry in the market, competitors at 23k+ stars with
no revenue, and the benchmark suggesting memory systems lose to long context.
That habit is the reason the documentation is worth anything. Continue it.

**Do not raise scope on your own.** The adoption ladder is
individual → team → organization, and each rung must be independently valuable.
Enterprise features are explicitly ruled out of v1 by D-001.
