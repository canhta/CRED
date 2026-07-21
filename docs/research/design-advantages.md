# Design Advantages — What We Know That They Didn't

Every item here comes from reading a competitor's source code and observing where
they became trapped. Each is **cheap to build in at v1** and **expensive or
impossible to retrofit** — which is exactly why the incumbents still carry them.

This is the concrete return on the discovery phase.

---

## The headline advantage: evidence-based trust, not popularity-based trust

**Memco's trust model is a Bayesian posterior updated by retrieval frequency and
feedback.** A claim becomes trusted because it gets used and upvoted.

That has a structural failure mode: **a claim that was true and became false stays
trusted until enough negative signal accumulates.** Popularity lags reality.

**CRED inverts this.** Trust derives from evidence, and invalidation is
deterministic:

```
source changes  →  claim referencing it expires immediately
```

No waiting for signal. No LLM call. A file moves, a function is deleted, a
decision is reversed — every claim resting on it is invalidated in the same
transaction.

This is **strictly better on freshness** (the pain measured at 48.8% vs 70.4%),
**cheaper** (no inference in the invalidation path), and **auditable** (the reason
is a diff, not a posterior).

It is also directly testable, which makes it a publishable result.

---

## Lessons by source

### From Graphiti — [evidence](evidence/graphiti.md)

| Lesson | Applied in CRED |
|---|---|
| **LLM nominates, deterministic code decides** (`edge_operations.py:538-572`) — LLM returns validated indices; expiry is interval arithmetic | Core architectural law. No LLM ever mutates state directly. |
| MinHash/LSH dedup with an entropy gate at **zero LLM cost** (`dedup_helpers.py`) | Adopt directly. Dedup must not cost inference. |
| Bi-temporality is **edge-only**; `EntityNode` has just `created_at`, summaries overwritten with no provenance | Bi-temporal on **every** object from day one. |
| Storage abstraction is a facade — **116 provider conditionals outside `driver/`** | Do not pretend to be portable. Commit to Postgres. |
| `group_id` is a flat string; any agent can `clear_graph` any group | Principal-aware scoping from the first migration. |
| Ingestion costs `3 + 2E + N` LLM calls per episode, **no ceiling** | Hard cost ceiling per ingest, enforced in code. |

### From Mem0 — [evidence](evidence/mem0.md)

| Lesson | Applied in CRED |
|---|---|
| A memory is an **untyped English sentence** — cannot be validated, expired by kind, or measured | Typed claims with per-type validity semantics. |
| Scope is **three opaque strings** | Real scope model with principals. |
| Write path is **ADD-only**; the reconciler has zero call sites; contradictions accumulate | Supersession in the write path from commit one. |
| Auth was built then **never read** (`server/main.py:366-374`) | Enforcement at recall, tested adversarially. |
| UUID→ordinal remap defeats ID hallucination (`main.py:889-894`) | **Steal.** |
| Explainable additive scorer (`utils/scoring.py:60-139`) | **Steal** — an inspectable score is the auditability axis. |

### From Letta — [evidence](evidence/letta.md)

| Lesson | Applied in CRED |
|---|---|
| `Memory.compile()` (`schemas/memory.py:688`) — deterministic string build, persisted, recompiled only on change | **Steal.** This is context packaging, already proven. |
| Budget and permission metadata rendered **in-band** into the prompt (`memory.py:161-166`) | **Steal.** The agent should see what it may not see. |
| `BlockHistory` — versioning, actor attribution, optimistic locking, undo/redo — with **zero production callers** | **Wire it.** They built our audit layer and never connected it. |
| Shared blocks are **last-writer-wins** (`agent_manager.py:1769`) | Optimistic locking on every shared write. |
| `.af` serialization with fail-closed referential integrity (`agent_serialization_manager.py:118-135`) | **Steal** for export and portability — the anti-lock-in guarantee. |

### From Onyx — [evidence](evidence/onyx.md)

| Lesson | Applied in CRED |
|---|---|
| ACL as **prefixed opaque strings** flattened onto documents and users, intersected by the index (~270 lines) | **Steal.** Best OSS answer to permissions surviving derivation. |
| Postgres is truth, index is cache, reconciled with **replace-not-union** | **Steal.** |
| **No ACL TTL anywhere** — a revoked permission stays queryable indefinitely if sync breaks. Fail-open. | TTL on every ACL, **fail-closed** on staleness. |
| MIT core ships a deliberately crippled `_get_acl_for_user`; `UserGroup` tables exist but are inert | Permissions in the core, always. Monetize hosting and operations. |
| 55 connectors, only 12 with permission sync | **Permission fidelity is the moat, not connector count.** |

### From Langfuse — [evidence](evidence/langfuse.md)

| Lesson | Applied in CRED |
|---|---|
| Immutable `(project, name, version)`; labels are **movable pointers**; telemetry pins the **resolved integer** | **Steal exactly.** This is why "which version caused this" stays answerable. |
| Object graph topped out at `trace`; `session_id` and `dataset_run_id` were **bolted on by later migration**, forcing `trace_id` nullable | Model the outcome layer **from day one**, even if unused at v1. |
| ClickHouse writer **drops rows** after max attempts (`ClickhouseWriter/index.ts:518`) | Never acceptable when the records **are** the product. |
| 6 containers, 37 queues | One Postgres. |

### From RAGFlow — [evidence](evidence/ragflow.md)

| Lesson | Applied in CRED |
|---|---|
| Citation as `position_int` 5-tuple plus an inline sentinel tag, rendered in ~15 lines | **Steal** when documents are in scope. |
| Citations are **PDF-only**; other formats receive fake `[[ii]*5]` positions | Two-tier citation from day one: **geometric** (bbox) and **structural** (path, line range). |
| `rag/` and `api/` circularly dependent — no extractable core | Keep the core independently importable, enforced by test. |

### From Memco's own paper — [evidence](evidence/memco-spark-paper.md)

| Lesson | Applied in CRED |
|---|---|
| **Seed memory with documentation before any experience exists** | **Steal.** This solves cold start and the n=1 problem. |
| No ablation separating documentation retrieval from experiential memory | Publish the ablation they did not. |
| One epoch tested; compounding never measured | Publish the epoch curve. |
| LLM judge used where DS-1000 execution tests were available | Execution pass@1 first, judge second. |
| No cost, latency, poisoning, conflict, or staleness evaluation | Report all five. |

### From prior art and market research

| Lesson | Applied in CRED |
|---|---|
| Artifacts that work are **small, factual, machine-derived, expiring** | Design constraint on the claim model. |
| Adherence approaches zero above ~6,000 words | Hard token ceiling on assembled context. |
| Review must be **event-triggered**, never a queue (Google SRE; LinearB 2.6x capacity) | Confirmation happens inside the task. |
| Freshness beats findability (48.8% vs 70.4%) | Expiry default-on. |
| Context files grow 4x faster than they shrink | Pruning is a first-class operation, not a cleanup script. |

---

## The five that matter most

If only five survive scope pressure, these are they:

1. **Deterministic invalidation from evidence change** — the headline advantage,
   and strictly better than popularity-based trust.
2. **LLM nominates, code decides** — the only reliable defence against memory
   poisoning; three competitors violated it and paid.
3. **Onyx's ACL string model, with TTL and fail-closed** — permission fidelity is
   the moat, and the incumbent's version fails open.
4. **Langfuse's version-pinning discipline** — the reason attribution survives
   after pointers move.
5. **Letta's `compile()` plus in-band budget metadata** — proven deterministic
   context packaging, already written and abandoned.

## The honest caveat

None of these wins a deal on its own. They compound into a system that is
**cheaper to run, harder to poison, and possible to audit** — and they are only
free **now**, before the write path exists.

After the write path ships, most become migrations. That is precisely the trap
Mem0 and Langfuse are in.
