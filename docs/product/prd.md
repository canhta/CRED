# CRED — Product Requirements

- **Product:** CRED — Shared Harness for Intelligence, Flow & Traceability
- **Owner:** canhta
- **License:** Apache 2.0

---

## 1. What CRED is

CRED is an **open-source, self-hostable organizational memory layer for AI
coding agents**.

Agents connect over MCP. They retrieve what the organization already knows before
starting work, and contribute what they learn as they finish it. Knowledge is
shared across people, repositories, and agent tools, and every claim is traceable
to the evidence that produced it.

It runs on a single PostgreSQL database and is deployed by the team that uses it.

### The one-sentence difference

Other systems decide what to trust from **usage signal**. CRED decides from
**evidence**, so a claim dies the moment its source changes rather than when
enough people stop upvoting it.

---

## 2. Problem

Coding agents restart from zero. Every session re-derives what the team already
established, and nothing learned in one session reaches the next person, the next
repository, or the next tool.

Existing memory systems solve this for **one user**. Organizational knowledge
needs three things they do not provide:

1. **Sharing across principals** — many people, many agents, one store, with
   access control that survives derivation.
2. **Freshness** — 70.4% of developers already know where to look; only 48.8%
   trust that what they find is current. Staleness is the binding constraint,
   not discovery.
3. **Traceability** — an organization cannot act on a claim it cannot audit.

### Where the gap actually is

Roughly 80 projects claim shared or team memory. **Fewer than 15 have a real
multi-user permission model.** GitHub searches for multi-agent memory access
control return three repositories, the largest with two stars.

And across every vendor — Anthropic's server-managed instruction files, Cursor's
team rules — **learned knowledge crosses no user boundary.** That gap is
structural rather than a missing feature, which is why shipping one has not
closed it.

> Crowded at the tagline, empty at the permission gate.

---

## 3. Users

**Primary:** the person responsible for how several teams use AI agents —
platform, DevEx, or engineering enablement. They own the outcome and currently
work by copy-paste.

**Entry point:** an individual developer working across several repositories.
CRED must be worth running alone, or it never reaches the primary user.

Self-hosting is a **requirement**, because it is what makes the individual entry
point work — a local instance becomes a team instance by changing a connection
string, with no migration and no separate product. It is not the reason to
choose CRED.

---

## 4. Design laws

Non-negotiable. Every one is derived from a documented failure in a competing
system, recorded in [design-advantages.md](../research/design-advantages.md).

### L1 — No claim without evidence

Every claim carries a pointer to what produced it: a commit, a diff, a code
symbol, a document span, or an explicit human attestation. A claim with no
evidence cannot be written.

Human attestation is evidence. Orphan claims are not.

### L2 — The model nominates, code decides

An LLM may propose extractions, duplicates, and contradictions. It never mutates
state. Every write, merge, supersession, and expiry is executed by deterministic
code against validated input.

This extends to **extraction**, not only reconciliation. The extractor emits
structured proposals into a constrained schema and holds no tool that can write,
supersede, or expire.

Provider behaviour makes this mandatory rather than merely prudent: **no provider
validates structured output server-side**, several silently drop schema
constraints, and constrained decoding guarantees a valid prefix rather than valid
JSON. Every model response is validated locally and gated on
`finish_reason != "length"`.

### L3 — Invalidation is deterministic, and anchors are semantic

When evidence changes, every claim resting on it is invalidated. No inference
call, no waiting for signal.

**Anchoring is a fingerprint ladder, never a byte hash of a line range.**

| Tier | Anchor | Survives |
|---|---|---|
| 1 | Tree-sitter symbol path | line moves, reformatting, insertions above |
| 2 | Normalized hash of the enclosing AST node | formatting, headers |
| 3 | Context-window line hash | small local edits |
| 4 | Raw byte hash | nothing — diagnostic only |

An anchor is valid **iff tiers 1 and 2 agree.** Tier 4 changing while 1 and 2
hold is formatting churn and must not expire anything. Tiers 1 and 2 disagreeing
is a genuine semantic change. Ambiguous resolution expires the claim; it never
guesses.

A byte hash of a line range fails in both directions: a formatting commit erases
a module's memory, and an insertion above silently re-anchors a claim onto
different code, which then validates. **A confidently wrong claim is worse than
no claim.**

### L4 — Confirmation happens inside the task, never in a queue

No approval backlog, no periodic review. A person confirms or rejects at the
moment they are already doing the work.

Review capacity is already oversubscribed roughly 2.6x by AI-generated code
review. A second queue will not be serviced.

### L5 — Access control is enforced at recall, fails closed, and is the intersection

Permissions are evaluated when knowledge is returned. Stale permission data
denies rather than grants, and every ACL entry carries a TTL.

A claim derived from several sources takes the **intersection** of their
permissions, enforced as a database constraint:

```
claim.acl ⊆ ⋂(evidence_i.acl)
```

Union leaks private content to readers of the public source. **Merging two
claims intersects their ACLs** — the natural implementation keeps the survivor's,
and that is the bug.

Unauthorized must be indistinguishable from nonexistent, in bytes and in timing.

### L6 — Everything is bi-temporal

Every object records when it was true in the world and when the system learned
it. Nothing is deleted; things expire. Intervals are half-open.

Supersession forms a DAG, enforced by a reachability check that rejects any edge
closing a cycle. Ties are broken by a deterministic total order, never by the
model. A challenger must exceed the incumbent by a margin to supersede.

**Supersession is evaluated per principal, not globally.**

### L7 — One database

PostgreSQL 17+ with pgvector. Relational, vector, full-text, and queueing all
live there. No second datastore may be added without removing another.

### L8 — Ingested content is untrusted

Repository content is data, never instruction. **Injecting five malicious texts
achieves roughly 90% attack success against corpora of millions of documents**,
and published defenses were found insufficient.

Required: provenance on every claim; the extractor holds no write authority (L2);
a claim from one source may not supersede a claim from another without explicit
policy — this alone defeats the memory-deletion attack; and recall output is
fenced as data, never interpolated into a system prompt.

---

## 5. Core model

### Claim

Small, typed, and independently expirable.

- `kind` — determines validity semantics
- `statement` — the assertion, kept short
- `scope` — organization, repository, path, or service
- `valid_from` / `valid_until` — when it is true in the world
- `recorded_at` / `superseded_at` — when the system knew it
- `superseded_by` — the replacing claim
- `acl` — the flattened permission set, evaluated at recall
- `confidence` — an explainable additive score, never an opaque posterior
- `source_repo`, `source_commit`, `extracted_by_model`, `prompt_version` —
  provenance, immutable, required by L8

Kinds ship as a closed set: `Convention`, `Decision`, `Constraint`,
`RejectedApproach`, `Failure`, `Reference`.

### Evidence

- `source` — repository, commit, path, symbol, document, or attestation
- `anchor` — the fingerprint ladder of L3
- `locator` — **geometric** (page, bounding box) for documents, **structural**
  (symbol path, line range) for code
- `extracted_text` — the normalized source text, **retained**
- `attested_by` / `attested_at` — set when the source is a person

**`extracted_text` is not optional.** Every system that can re-embed stores
content alongside vectors; a vectors-only store can never migrate to a new
embedding model. This is decided at ingest, long before the first model swap.

### Embeddings

`embedding_model_id` is **NOT NULL** on every vector row, with no defaults and no
backfill-by-guess. **Reads filter on it** — a model column that reads ignore is
worthless. One current and one in-progress model are enforced by partial unique
indexes, and reads never touch the index being built.

---

## 6. Interface

**MCP is the only integration surface.** No SDK, no library to embed.

| Tool | Purpose |
|---|---|
| `recall` | Retrieve claims relevant to the current task, within a token budget |
| `remember` | Contribute a claim with its evidence |
| `revise` | Supersede or enrich an existing claim |
| `confirm` | Affirm or reject a claim in place (L4) |

A CLI wraps the same operations for CI and non-interactive use.

CRED is an OAuth 2.1 **resource server** only, with an external identity
provider. It validates token audience itself, because no SDK does — making an
SDK-default server non-compliant with a specification requirement.

**Nothing is persisted keyed on MCP session identity**, since sessions are being
removed from the protocol.

### Context assembly

`recall` returns a deterministically assembled package. The build is a pure
function of the claim set, persisted and rebuilt only when inputs change. Budget
and permission metadata are rendered **in-band**, so the agent can see that
something exists which it may not read.

The package has a hard token ceiling — adherence collapses above roughly 6,000
words — and **exceeding it drops claims rather than truncating text**. Access
control is applied **before** budgeting, never after.

Every response carries `as_of` and `staleness_seconds`. Truncation is reported
explicitly with an omitted count. A silent truncation is a lie.

---

## 7. Retrieval and curation

**Retrieval** is hybrid: vector similarity, full-text, and scope proximity,
combined by reciprocal rank fusion, with each signal's contribution inspectable
in the response.

Iterative index scans are **required, not optional**. Filters apply after the
index scan, so a principal who can read 10% of claims otherwise receives roughly
four results instead of forty — silently. Raising the search parameter makes this
worse, not better.

**Curation** runs as a separate worker, never inline with a request:

1. **Deduplicate** — exact hash of normalized text at v1
2. **Reconcile** — the model nominates contradictions; code decides expiry (L2)
3. **Expire** — anchors re-checked; changed sources invalidate (L3)
4. **Prune** — claims that never surface are dropped; a first-class operation,
   because context grows roughly four times faster than it shrinks
5. **Rescore** — confidence recomputed from evidence state and usage

**Merging is a link, never a delete.** Both claims persist; one is marked a
duplicate. Unmerge becomes an update rather than an impossibility.

Fuzzy deduplication is deliberately **excluded from v1**. At v1 scale exact
hashing has a zero false-merge rate by construction, while similarity hashing
buys recall at the cost of a tunable false-merge rate on a destructive
operation — a bias that shipped as a library default for years before being
corrected in 2026.

### Cold start

A new instance is seeded from repository history and project documentation before
any agent has contributed. CRED is useful on first run, alone, with no team.

### Why there is no knowledge graph

This is a deliberate exclusion, not an oversight.

CRED already has graph structure, held in relational tables. Evidence produces
claims; claims supersede claims; anchors point at symbols. The L3 invalidation
cascade **is** a graph traversal, and the L6 supersession chain **is** a DAG.
Recursive queries handle both at v1 scale.

What is genuinely absent is **entity resolution**: recognizing that "service A"
across fifty claims refers to one thing, and answering multi-hop questions such
as "what depends on service A". Similarity is not identity, and embeddings cannot
supply that.

It is excluded at v1 on evidence rather than preference:

- The reference implementation costs `3 + 2E + N` inference calls per ingested
  episode with **no ceiling** — precisely the risk that curation costs more than
  the tokens it saves.
- **The section 11 ablation is what answers this question.** If retrieval proves
  to be the bottleneck, graph structure earns its cost. If it does not, the cost
  buys nothing.

Adding it before that measurement exists would be guessing.

---

## 8. Usage and limits

Every limit here is a **security control first** and an operational concern
second. Shared memory with unbounded per-principal write access is a poisoning
vector, not merely a capacity problem.

All four require columns on the claim table and a usage table. They are cheap now
and a migration later.

### Contribution quota

Claims accepted per principal per window, enforced server-side.

Without it, one agent in a loop can flood the store with near-duplicates sitting
just below the deduplication threshold — a threshold can always be approached
from below. This is the sybil and repetition defence, and it belongs in the
product rather than only in the test suite.

### Cost attribution

Inference calls, tokens, and wall-clock recorded per principal and per scope.

Two purposes: enforcing a hard cost ceiling in code, and answering **which teams
actually use this** — the question the primary user needs answered, and one no
competitor exposes.

### Recall budget

Rate and assembled-package size capped per principal, protecting tail latency
from an agent calling `recall` in a loop.

### Scope growth bound

Each scope carries a claim ceiling. Exceeding it makes pruning more aggressive
rather than letting the scope grow without limit. Context grows roughly four
times faster than it shrinks; growth must be bounded by policy, not by hope.

### What is reported

Quota state is visible to the principal before it is hit, and exhaustion is a
loud, explicit denial — never a silent drop. Usage is exported through
OpenTelemetry rather than a bundled dashboard.

---

## 9. Deployment

One container plus PostgreSQL, or a single binary with `DATABASE_URL`.

```
docker compose up
```

Self-hosting is the default path, not an enterprise upgrade. Comparable systems
require six to sixteen containers; matching that would forfeit the individual
entry point, and with it the adoption path.

Because one instance serves one organization, the hard parts of multi-tenancy —
partition-per-tenant planner limits, per-session metadata growth, lock-manager
contention — **do not apply**.

---

## 10. Scope

### In

- MCP server exposing `recall`, `remember`, `revise`, `confirm`
- Claim and evidence model with bi-temporal validity and supersession
- Semantic anchoring and deterministic invalidation
- Hybrid retrieval with an explainable score
- Curation worker: dedupe, reconcile, expire, prune, rescore
- Scope model with recall-time enforcement, TTL, and intersection semantics
- Cold-start seeding from repository history and documentation
- Single-command self-hosted deployment
- CLI for CI and non-interactive use
- Reproducible evaluation harness (section 11)

### Out

- Document ingestion beyond plain text and Markdown
- Connectors to third-party systems
- Web UI beyond a read-only inspector
- SSO gating, SCIM, audit export, organization hierarchy
- Managed hosting
- Agent orchestration, planning, or execution
- Fuzzy deduplication
- Cross-agent session handoff — measured demand is roughly 200 downloads per
  month against 985,000 for rule synchronisation
- A second database backend, a plugin system, a bundled identity provider

---

## 11. Evaluation

Evaluation is a deliverable, not a phase. It is also the kill criterion.

### First experiment, before anything is built

**Does retrieved memory beat plain long context?** Nobody has answered this for
the case CRED cares about. The one benchmark that exists — a single blog post
measuring Mem0 and Zep on conversational fact recall — reports them as far more
expensive and materially less accurate than passing the full history. It is
unreplicated, single-model, published by a vendor in the same category, and its
long-context arm averaged roughly 4,000 input tokens, which does not test long
context at all. It is unrebutted only in the sense that almost nobody read it.

So the benchmark does not settle the question against CRED. It does something
worse: it shows the question is open and unmeasured while the entire category
sells as though it were closed. That is why the experiment runs before any
product code, and why CRED runs it at a context length that can actually lose.

If retrieved memory does not win, CRED is wrong — and so is every competitor.
It is cheap to test and existential. Either outcome produces a publishable
result.

Design, pre-registered thresholds, and the full teardown of the existing
benchmark: [v0 experiment design](../research/spikes/v0-experiment-design.md).

### The evaluation the category has not published

The leading competitor published results with **no ablation** separating
documentation retrieval from experiential memory, **one epoch**, and an LLM judge
where execution tests were available. Their headline figures appear only in
marketing — never in the paper, and never alongside a resolve rate.

CRED publishes the version that survives scrutiny:

1. **Ablation** — documentation-only, experiential-only, combined. Without this,
   no claim about shared memory is supportable.
2. **Execution first** — pass rate as the primary metric; LLM judging secondary
   and labelled.
3. **Epoch curve** — does quality compound or plateau? The thesis rests on
   compounding, and nobody has published it.
4. **Cost and latency** — tokens, wall-clock, and inference calls per operation.
5. **Adversarial** — poisoning, contradiction, staleness, and permission leakage,
   each with a stated failure mode.

**Kill criterion:** if the ablation shows experiential memory adds nothing over
documentation retrieval, that is a finding about the category. Publish it and
stop.

---

## 12. Acceptance

v1 is complete when all of the following hold on a repeatable evaluation set.

1. A single developer gets useful recall on first run, with no team and no
   contributed knowledge, from cold-start seeding alone.
2. Claude Code and Codex retrieve identical authorized claims through MCP.
3. Changing a source file invalidates every dependent claim, verified with a
   concurrent reader.
4. A pure-formatting commit expires **zero** claims. A semantic change expires
   exactly the right ones.
5. A claim's full evidence chain is reconstructible through the API.
6. Revoking access removes claims from recall immediately; stale permission data
   denies; unauthorized is indistinguishable from nonexistent.
7. The assembled package respects its token ceiling and reports what it dropped.
8. Retrieval completeness holds under realistic access-control selectivity.
9. The reconciler produces byte-identical output across five identical runs.
10. The ablation in section 11 runs end to end and produces publishable numbers.
11. `docker compose up` reaches a working instance with no additional steps.

---

---

## 13. Roadmap

Stages are gated by **evidence, not by dates**. A stage does not begin until the
previous gate is cleared.

| Stage | Contents | Gate to proceed |
|---|---|---|
| **v0** | Experiment only. Long context vs. retrieval vs. structured memory, on a real repository and a fixed task set. No product code | Memory beats long context by a meaningful margin. **If it does not: publish and stop** |
| **v1** | Claims and evidence, four MCP tools, semantic anchoring, intersection ACLs with fail-closed TTL, curation worker, exact deduplication, usage limits, single-command deploy | The ablation shows experiential memory contributes **independently** of documentation retrieval |
| **v2** | Entity resolution and multi-hop traversal, fuzzy deduplication, document ingestion, retrieval tuning | Real usage data identifies where the bottleneck actually is |

### v0 ships no product code

This ordering is deliberate. Building v1 first and measuring afterwards produces
a measurement nobody trusts, because by then there is six months of sunk cost
arguing with the result.

v0 needs a repository, a task set, three conditions, and a few hundred lines of
script. Both outcomes are publishable, and one of them saves six months.

### v2 is deliberately not planned in detail

Everything listed under v2 is a hypothesis about where the bottleneck will be,
and bottlenecks are measured rather than predicted. Specifying v2 now would
manufacture false confidence.

The one commitment: v2 items are admitted only when usage data shows they are the
constraint — never because they are the more sophisticated technique.

## 14. Risks

| Risk | Mitigation |
|---|---|
| Memory adds nothing over long context | Section 11's first experiment. Existential, cheap, runs first |
| Experiential memory adds nothing over documentation retrieval | The ablation. This is the kill criterion |
| Ingested content poisons shared memory | L8: provenance, no extractor write authority, no cross-source supersession, fenced output |
| Permission leakage through derived claims | L5 intersection as a database constraint; nightly recomputation alarming on any widening; canary claims per partition |
| Silently wrong anchoring | L3's fingerprint ladder; formatting-commit test in CI |
| Retrieval degrades silently under access filtering | Iterative scans, benchmarked against real ACL selectivity |
| Stale embeddings after a model change | Model ID required and filtered on; norm constraint; nightly provider-drift canary |
| Curation falls behind and serves stale confidence | `as_of` in every response; refuse to return confidence beyond the freshness threshold |
| Governance friction stops adoption | L4 — confirmation inside the task, never a queue |
| Curation costs more than the tokens it saves | Zero-inference dedup; hard cost ceiling per operation; cost reported in evaluation |
| Platform vendors ship this for free | No vendor moves learned knowledge across a user boundary; the gap is structural |
| **Demand may not exist** | Accepted. In roughly 30 forum results on this topic, about 20 are founders promoting their own memory products; two are genuine user reports. A market where sellers outnumber buyers 10:1 is more often imagined than early. The project proceeds on distribution access, not demonstrated organic demand |
| Open-source adoption does not convert to revenue | Accepted. Five projects in this category have 23k+ stars and no disclosed revenue; conversion runs 1–1.3%; adoption is treated as distribution, not validation |
