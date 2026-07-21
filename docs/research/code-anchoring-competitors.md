# Code-anchoring in competitors: who expires a memory when its code changes

The question CRED's thesis stakes a claim on: *a claim lives only while its
evidence does.* For **code** specifically — does any shipped competitor anchor a
stored memory/claim to a code span (file, symbol, line) and
invalidate/expire it when that code changes semantically? This document scans
agent-memory systems, code assistants, and code-intelligence formats to place
CRED against real prior art, not roadmaps.

**Method and provenance.** Commissioned scan. Findings were gathered by fetching
primary sources — official docs, GitHub source files, protocol specs. The
load-bearing claims (Cognee's schema, Copilot Memory's validation, the SCIP
Symbol grammar, Graphiti's invalidation trigger) were fetched and read directly
for this document; the remainder were fetched during the sweep and are cited to
the exact URL so any reader can re-verify. Graphiti and mem0 also have
line-level internal repository scans already in this repo
([graphiti.md](evidence/graphiti.md), [mem0.md](evidence/mem0.md)); those are
cited by `file:line` rather than re-derived.

The distinction that runs through everything below: **"invalidate when a new
fact contradicts"** (conversational, LLM-judged — what these products do) is not
**"invalidate when the source code span changed"** (what CRED does). They are
different mechanisms with different triggers. Every row keeps them apart.

---

## 1. Agent-memory systems

Scope: mem0 / OpenMemory, Letta (MemGPT), Zep / Graphiti, Cognee, Memobase.

| Product | Code-derived memory | Anchors to file/symbol/span | Invalidates on code change |
|---|---|---|---|
| mem0 / OpenMemory | No | No | No — conversational contradiction, LLM-judged |
| Letta / MemGPT | Onboarding input only | No | No — self-edit + consolidation + git sync of markdown |
| Zep / Graphiti | No | No | No — conversational contradiction, bi-temporal, LLM-judged |
| **Cognee** | **Yes (first-class code graph)** | **Yes (file + line-span + symbol)** | **No — re-ingest/upsert; no span-level detect-and-expire** |
| Memobase | No | No | No — conversational profile-slot update, LLM-judged |

### Cognee is the only agent-memory system that anchors to code spans

**VERIFIED.** Cognee has a first-class code-graph subsystem. Two schemas exist:

- `cognee/shared/CodeGraphEntities.py` defines `FunctionDefinition` and
  `ClassDefinition`, each carrying `start_point: tuple`, `end_point: tuple`
  (line/column span), `source_code: str`, and `file_path`. `CodeFile` carries
  `file_path`, `language`, `source_code`, and edges
  `provides_function_definition` / `provides_class_definition`. Fetched
  2026-07-21:
  <https://raw.githubusercontent.com/topoteretes/cognee/main/cognee/shared/CodeGraphEntities.py>
- `cognee/tasks/code_graph/models.py` defines `CodeGraphEntity` (fields
  `file_path`, `line: Optional[int]`, `symbol_kind`) and typed nodes
  `CodeSymbol`, `ApiEndpoint`, etc. Fetched 2026-07-21:
  <https://raw.githubusercontent.com/topoteretes/cognee/main/cognee/tasks/code_graph/models.py>

So Cognee genuinely anchors memory to a file + span + symbol — the one thing the
rest of the field does not do.

**But Cognee has no span-level staleness-on-change. VERIFIED (absence).** Neither
schema carries any field for content hash, validity time, invalidation, staleness,
or version — confirmed by reading both files above. Refresh is *re-ingest and
upsert by deterministic node id*: re-running the extractor rewrites the same
nodes wholesale. There is no mechanism that detects a specific symbol's body
changed and expires the memory anchored to it, and no pruning of nodes for
deleted or renamed symbols in the code pipeline. A file-level content-hash
"reprocess if changed" path exists only in Cognee's *generic document* pipeline,
not the code pipeline (`content_changed` gate in
`cognee/tasks/ingestion/ingest_data.py`) — that is "reprocess a changed file,"
not "expire the claim bound to a changed span."

- UNVERIFIED / FALSIFIED-as-marketing: a Cognee blog claim that "memify prunes
  stale nodes" is not supported by the default memify task set
  (`cognee/memify_pipelines/memify_default_tasks.py` is
  triplet-embedding / index / session), per the sweep. Treat the pruning claim
  as marketing until a code path is shown.

### Graphiti has real temporal invalidation — but the trigger is conversational

**VERIFIED (internal scan).** Graphiti is the one product with genuine temporal
edge invalidation (`valid_at` / `invalid_at` / `expired_at`, bi-temporal). The
invalidation *decision* is deterministic interval arithmetic
([graphiti.md](evidence/graphiti.md):128-153, `edge_operations.py:538-572`) —
worth borrowing. But what *nominates* an edge for invalidation is a single LLM
call detecting that a **new NL fact contradicts an existing edge**
([graphiti.md](evidence/graphiti.md):99-121, `dedupe_edges.py`). The trigger is
a contradicting conversational fact, never a changed code span. The README's
"automatic fact invalidation" is accurate only in that narrow, LLM-in-the-loop,
conversational sense.

- Source (also fetched in sweep):
  <https://raw.githubusercontent.com/getzep/graphiti/v0.29.1/graphiti_core/prompts/dedupe_edges.py>,
  <https://arxiv.org/html/2501.13956v1> (Zep/Graphiti paper — message-type
  episodes, conversation memory).

### mem0, Letta, Memobase: no anchoring, no code-change trigger

- **mem0. VERIFIED.** Fact extraction runs on user/assistant messages only; no
  file/symbol/span field. Its "code memory" for Cursor/Claude Code is the same
  generic engine storing NL preferences ("Auth uses NextAuth") — positioning,
  not code parsing (<https://mem0.ai/blog/claude-code-memory>, marketing). The
  famous ADD/UPDATE/DELETE reconciler is *dead code in v2.0* with zero call
  sites ([mem0.md](evidence/mem0.md):10, `prompts.py:176-324`); the surviving
  update path is MD5 dedup. Where UPDATE/DELETE ever fired, the trigger was a
  contradicting conversational fact, never a code diff.
  <https://raw.githubusercontent.com/mem0ai/mem0/main/mem0/configs/prompts.py>
- **Letta. VERIFIED.** Memory blocks are plain strings; the frontmatter schema
  has no file/symbol/line field. Letta Code's `/init` reads a repo to *bootstrap*
  memory but is designed to store generalized prose ("Generalize, don't
  memorize"). "Sync" is git storage sync of the memory markdown, never a diff of
  project source. <https://docs.letta.com/guides/agents/memory>,
  <https://github.com/letta-ai/letta-code>. (The upstream `letta` server repo is
  deprecated — [letta.md](evidence/letta.md):10.)
- **Memobase. VERIFIED.** Stores user profiles + event timeline extracted from
  chat; no code concept, anchor is user-id + profile-slot. Freshness is
  LLM re-extraction of user attributes from new conversation.
  <https://github.com/memodb-io/memobase>,
  <https://docs.memobase.io/features/profile/profile>.

**Verdict, category 1:** No agent-memory system does code-span anchoring *with*
staleness-on-change. Cognee anchors to spans but never expires on change; every
other product's "invalidation" is an LLM judging a new conversational fact
against an old one.

---

## 2. Code assistants with memory / rules / context

Scope: Cursor, Continue.dev, Windsurf, GitHub Copilot, Sourcegraph Cody, Aider,
Claude Code, Tabnine.

### GitHub Copilot Memory is the closest shipped prior art to CRED's thesis

**VERIFIED — and closer than expected.** Copilot's "agentic memory" (public
preview, Jan 2026) stores repository facts **"with citations pointing to the code
that supports them,"** and before using a fact **"it checks those citations
against the current branch to confirm the information is still accurate. Only
validated facts are used."** Unused facts are **deleted after 28 days**, and the
timer resets when a fact is validated and used. Facts are repository-scoped;
user-level preferences apply across repositories. Fetched 2026-07-21:
<https://docs.github.com/en/copilot/concepts/agents/copilot-memory>. Announcement:
<https://github.blog/changelog/2026-01-15-agentic-memory-for-github-copilot-is-in-public-preview/>.

This is citation-anchored, code-derived memory that is validated against current
code before use — the same *shape* as "a claim lives only while its evidence
does." It is the one product that ties a remembered fact to supporting code and
gates use on that code still supporting it.

Where it stops short of CRED, honestly stated:

- **Granularity is undocumented. VERIFIED (absence).** The docs say citations
  "point to the code that supports them" and validation is "against the current
  branch," but never state whether a citation is a file, a symbol, a line span,
  or a broad pattern. CRED's tier-1 relocatable *symbol path* and tier-2
  normalized AST-node hash are a specific, inspectable anchor;
  Copilot's is an opaque citation whose resolution is not published.
- **Validate-before-use, not expire-on-change.** Copilot checks at *use* time
  and drops unused facts on a blanket 28-day TTL. It does not publish a
  reformat-vs-semantic-change distinction — CRED's core refinement is that a
  pure formatting change expires *zero* claims and a semantic edit expires
  *exactly* the affected ones ([semantic-anchoring.md](spikes/semantic-anchoring.md):140-148).
  Whether Copilot's validation survives a reformat is unknown and unpublished.
- **Closed and hosted.** No self-host, no open mechanism to inspect or govern.
  CRED's wedge is open-source, self-hostable, and a team/governance layer.

Do not overclaim novelty of the *idea* of validating code-derived memory against
current code: Copilot ships a version of it. CRED's defensible difference is the
*mechanism* (symbol-path + AST-node semantic hash, reformat-immune, inspectable)
and the *deployment* (OSS, self-host, governed shared memory), not the bare
concept.

### The rest: static files, re-retrieval, or re-embed — none span-anchored

| Product | Memory mechanism | Span-anchored | On code change |
|---|---|---|---|
| Cursor | Rules (`.mdc` static) + Memories (auto, project-scoped text) | No | Nothing — stays until manually edited |
| Continue.dev | Rules (static md/yaml) + vector index | No | Rules static; index re-embedded on (manual) re-index |
| Windsurf | Rules (static) + Memories (auto, workspace text) + index | No | Memories persist; index re-embedded on schedule |
| Cody | Query-time search retrieval (embeddings removed on EE v5.3) | No (nothing stored) | Re-retrieved fresh; no stored stale facts |
| Aider | Repo map (tree-sitter, per-request) + CONVENTIONS (static) | No | Repo map re-derived every request; conventions static |
| Claude Code | CLAUDE.md (static) + auto memory (markdown) | No (path globs at most) | Nothing — prune manually |
| Tabnine | Local embeddings index (Qdrant) | No | Incrementally re-embedded on change |

- **VERIFIED** highlights: Cursor's own docs concede staleness and push it onto
  the author — "Reference files instead of copying their contents … prevents
  them from becoming stale as code changes" (<https://cursor.com/docs/rules>).
  Aider's repo map is re-derived from the current AST every request via
  tree-sitter — re-derivation, not stored-then-invalidated
  (<https://aider.chat/docs/repomap.html>). Cody re-retrieves at query time; on
  Enterprise, embeddings were removed as of v5.3
  (<https://sourcegraph.com/docs/cody/core-concepts/context>).

Two freshness strategies dominate and neither is span-anchored memory:
**re-derive/re-retrieve at query time** (Aider, Cody) makes staleness structurally
impossible by keeping nothing; **re-embed on re-index** (Tabnine, Continue,
Windsurf, Cursor's index) refreshes a similarity index, not a location-bound
claim. The rules/memory layer proper (Cursor, Windsurf, Claude Code, Continue
rules) is static until a human edits it.

**Verdict, category 2:** Only GitHub Copilot ties a remembered fact to supporting
code and gates use on validation against current code — the closest analog to
CRED anywhere in this scan. It stops short at undocumented granularity,
validate-at-use rather than expire-on-semantic-change, and closed hosting.

---

## 3. Code-intelligence identity models (the "relocatable symbol path" analog)

Scope: Sourcegraph SCIP, LSIF, GitHub stack-graphs, Glean, Moderne/OpenRewrite.
CRED's tier-1 code anchor is a tree-sitter symbol path that must survive an
insertion above or a reformat and change only on a semantic edit
([semantic-anchoring.md](spikes/semantic-anchoring.md)). This is a solved problem
in code intelligence — borrow the identity model.

### SCIP Symbol string is the cleanest analog for a position-independent path

**VERIFIED.** A SCIP `Symbol` is a structured string — `scheme package
(descriptor)+` — where `package = manager name version` and descriptors form a
fully-qualified name using suffixes (`/` namespace, `#` type, `.` term, `(`
method). It **encodes no line/column/byte position**; the spec likens it to a
URI. Position lives entirely in a separate `Occurrence` message
(`single_line_range` / `multi_line_range`). Adding a function above a definition
changes its `Occurrence` range but not its `Symbol` string. Fetched 2026-07-21:
<https://raw.githubusercontent.com/sourcegraph/scip/main/scip.proto>. This is
exactly CRED's tier-1 goal — a relocatable path that survives code movement —
already specified and battle-tested. If tier-1 needs a canonical format,
SCIP's descriptor grammar is the one to copy.

Other identity models, for the record:

- **Glean symbol ID (VERIFIED).** Strongest *stated* guarantee: a stable string
  (e.g. `REPOSITORY/cpp/folly/Singleton`) that, per Meta, "doesn't change even if
  the symbol's definition moves around." Application-defined rather than a formal
  grammar. <https://engineering.fb.com/2024/12/19/developer-tools/glean-open-source-code-indexing/>.
- **OpenRewrite `JavaType.Method` (VERIFIED).** Identity is a compiler-accurate
  structural signature (declaring type + name + return + params + …),
  position-independent, format-preserving. Best analog if the path must be
  semantically precise about overloads/generics.
  <https://docs.openrewrite.org/reference/type-attribution>.
- **LSIF monikers (VERIFIED).** `{scheme, identifier}` position-independent
  identity layered over an otherwise line-based graph — durable but opaque, less
  useful as a *readable* path.
  <https://microsoft.github.io/language-server-protocol/specifications/lsif/0.4.0/specification/>.
- **stack-graphs (VERIFIED).** Name resolution as graph paths; incremental at
  *file* granularity, not symbol-body granularity.
  <https://github.blog/2021-12-09-introducing-stack-graphs/>.

### None of them expire a downstream artifact when a symbol's body changes

**VERIFIED (absence).** SCIP and LSIF separate identity from a body they never
hash. Glean and stack-graphs re-index at *file* granularity on change, not
per-symbol-body. OpenRewrite's LST is the only model that *could* derive
body-change detection (a full compiler-accurate tree is structurally diffable),
but that is an inference — no documented content-hash or expiry feature exists
(**UNVERIFIED** as a shipped capability). These systems solve *relocation* of a
reference; none solves *expiry of a claim when the referenced body changes*.

**Verdict, category 3:** The relocatable-symbol-path idea is proven prior art —
SCIP is the reference. The body-change → expire-the-claim step is a gap none of
these fill at symbol granularity; CRED's tier-2 normalized AST-node hash is the
piece to add on top of a SCIP-style identity.

---

## 4. Synthesis: how novel is "expire a memory when its code evidence changes"?

**The bare concept is not unique — Copilot ships a version of it; the specific
mechanism CRED uses appears to be unclaimed.** Honest breakdown:

- **Relocatable symbol identity (tier-1): not novel, and shouldn't pretend to
  be.** SCIP/Glean/OpenRewrite have solved position-independent symbol identity.
  Borrow SCIP's descriptor grammar rather than invent one.
- **Code-derived memory anchored to a span: rare but exists.** Cognee (agent-memory)
  anchors to file+span+symbol; Copilot Memory (assistant) anchors facts to code
  citations. CRED is not first here.
- **Validating code-derived memory against current code: shipped by Copilot,
  Jan 2026.** This is the honest ceiling on the novelty claim. "A claim lives
  only while its evidence does," applied to code, is a live, shipped idea in a
  major product. Any pitch that treats the *concept* as unprecedented is wrong
  and the founder should not make it.
- **What is genuinely unclaimed, as far as this scan found:** the *combination*
  of (a) a relocatable symbol-path anchor with (b) a normalized AST-node
  semantic hash that distinguishes a pure reformat (expire nothing) from a
  semantic edit (expire exactly the affected claims), (c) as an inspectable,
  deterministic, open-source, self-hostable mechanism, (d) with a governed team
  layer. No product in this scan combines all four. Cognee anchors but never
  expires. Copilot expires but with undocumented granularity, closed hosting,
  and no published reformat-vs-semantic distinction. Graphiti expires
  deterministically but on conversational contradiction, not code change.

**Closest analog to borrow from, ranked:**

1. **SCIP** — copy the descriptor-based Symbol grammar for tier-1 identity.
2. **Graphiti** — copy the *shape* of its invalidation: LLM (or detector)
   *nominates*, deterministic code *decides* the expiry
   ([graphiti.md](evidence/graphiti.md):128-153). CRED already mirrors this
   split; Graphiti is the proof it holds up.
3. **Copilot Memory** — study as the direct competitor to the thesis: validate-
   before-use + TTL is a real design point. CRED's differentiator must be stated
   as mechanism + openness + reformat-immunity, not concept.
4. **OpenRewrite LST** — if tier-2 ever needs to be more than an AST-node hash
   (e.g. dependency-version-aware), its compiler-accurate signature is the model.

**Where CRED is genuinely ahead:** reformat-immune semantic expiry at symbol
granularity, verified end-to-end for text and demonstrated for code
([semantic-anchoring.md](spikes/semantic-anchoring.md):140-148), delivered as an
open, self-hostable, governed mechanism. That is a defensible position. "First to
expire a memory when its code changes" is **not** — say "reformat-immune,
symbol-granular, open, and self-hostable," which is true, over "novel," which is
not.
