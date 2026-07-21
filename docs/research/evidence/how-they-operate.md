# How they operate

An operational scan of AI memory systems: cold start, the runtime loop, how
memory actually gets written, tool surfaces, required configuration, and what
users file issues about.

This is deliberately **not** a repeat of the existing scans. `mem0.md`,
`letta.md`, `graphiti.md`, `onyx.md`, `ragflow.md` and `context7.md` covered
data model, write path, retrieval internals, provenance, ACL and storage
abstraction. Those answered the questions of an earlier thesis that D-011 has
since demoted. The question here is different: **what does it take to run one of
these things, and what does a user get for it?**

The section that matters is [what CRED's first slice should
be](#what-creds-first-implementation-slice-should-be). Everything before it
exists to support that argument.

---

## 1. What was checked, and what could not be

### Checked, with evidence

| Subject | How | Pin |
|---|---|---|
| Mem0 | Local clone read + full-history clone for `git log`/`git show` | `9383e9a2556a533d289a8aa041c7a6660e806581` (2026-07-20) |
| Letta | Local clone read | `b76da9092518cbaa2d09042e52fdcbde69243e18` (2026-07-03) |
| Graphiti | Local clone read | `0b4bcf1284ee5fba56b77ed9961568a541e0d418` (2026-07-17) |
| Zep | Fresh shallow clone + benchmark artifacts re-extracted independently | `0375d7be4a72cda6a43ecdc6fd9055846eb0fd0e` (2026-07-17) |
| Supermemory | Fresh shallow clone + `gh release view` + fetched install script | `566be208981aa23ef20a85fd50a737861b1b10b2` (2026-07-18) |
| MCP memory server | WebFetch of raw `index.ts` and `README.md` on `main` | fetched, not pinned |
| Issue trackers | `gh issue list` / `gh issue view` across six repos | live retrieval |

Every issue number below was retrieved with `gh`. Every `file:line` was read.

### Could not be checked, and why

- **Wall-clock latency measured by me.** I did not run any of these systems. The
  decision log already records one incident of latency numbers taken under CPU
  contention (D-010). Rather than add a second, this document reports only
  numbers that the projects themselves committed, that users reported in issues,
  or that are structurally derivable by counting calls in source. Where a number
  is a vendor claim, it is labelled a claim.
- **`getzep/zep` issue tracker.** Issues are **disabled** on that repository —
  `gh` returns "the 'getzep/zep' repository has disabled issues." VERIFIED.
  There is therefore zero community complaint signal for Zep. Absence of
  complaints is not evidence of quality; it is evidence of a closed channel.
- **Supermemory's ingestion internals.** The engine is a prebuilt binary, not
  source (see §7). Per-add LLM call counts are not determinable. The
  queued-vs-synchronous split below comes from their docs, not from code.
- **Letta Code's cold-start behaviour.** `letta-ai/letta-code` is a separate repo
  that was not cloned. Its README (fetched) documents install and `/connect` for
  API keys but says nothing about seeding from repository files. UNVERIFIED.
- **The MCP memory server's line numbers.** WebFetch returns rendered markdown,
  not numbered source. Content is VERIFIED; line citations are deliberately
  absent rather than invented.
- **Mem0 issue #5352**, cited *inside* issue #5850's body as a 41-comment
  community workaround thread, **does not resolve**. `gh issue view -R
  mem0ai/mem0 5352` returns "Could not resolve to an issue or pull request."
  FALSIFIED as a citation; recorded here so nobody re-derives it.

### Not done, deliberately

Rule 6 of `docs.md` requires indexing a new document in `docs/README.md` in the
same change. The commissioning task explicitly forbade modifying any other file.
The task instruction wins; **this document is currently unindexed and needs a
one-line addition to `docs/README.md`.**

---

## 2. Cold start, per system

This was framed as the highest-value question. The answer is more interesting
than "nobody does it," which is what the framing anticipated.

### The short version

**Five of six systems are empty on first run.** One is not — and the exception
is the most commercially successful of them, which did it recently, and did it
in exactly the shape CRED proposed.

| System | Seeds from existing material? | What it seeds from |
|---|---|---|
| **Mem0** (plugin) | **Yes** | `CLAUDE.md`, `AGENTS.md`, `.cursorrules`, `.windsurfrules`, `mem0.md`; plus competitor configs |
| Mem0 (library / OSS server) | No | — |
| Letta | Partial | User-uploaded files into a "folder"; imported `.af` agent files |
| Graphiti | No | — |
| Zep | No | — |
| Supermemory | Partial, cloud-only | Twitter bookmarks, ChatGPT/Claude/Grok transcripts, Drive/Gmail/Notion/GitHub connectors |
| MCP memory server | No | — |

### Mem0 does cold-start seeding, and it is a coding-agent plugin

VERIFIED.
`/Users/canh/Solo/OSS/mem0/integrations/mem0-plugin/scripts/auto_import.py:1-8`:

> Auto-import declarative project files into mem0.
> Runs in the background from the SessionStart hook (startup only).
> Imports CLAUDE.md, AGENTS.md, .cursorrules, .windsurfrules, mem0.md
> into mem0 as project profile memories, skipping unchanged files via
> SHA-256 hashing.

The target list is at `auto_import.py:47`: `TARGET_FILES = ["CLAUDE.md",
"AGENTS.md", ".cursorrules", ".windsurfrules", "mem0.md"]`. Files over 100 KB
are skipped (`:46`). Content hashes live in `~/.mem0/file_hashes.json` (`:48`)
so re-import is idempotent. A lock file prevents concurrent runs (`:49`,
`_acquire_lock` at `:52`).

There is a second seeding path aimed squarely at competitors —
`/Users/canh/Solo/OSS/mem0/integrations/mem0-plugin/scripts/import_competing_tools.py:1-14`
takes sub-commands `cursorrules`, `copilot`, `cline`, `continue` and chunks each
tool's config files into mem0 as `project_profile` memories.

Three things to notice, because they cut different ways:

1. **It seeds from *declarative documentation*, not from repository history.**
   No `git log`, no commit walk, no diff analysis. It reads the files a developer
   already wrote for an agent. That is a far cheaper implementation than what
   CRED proposed, and it targets material that is already curated prose.
2. **It POSTs to `https://api.mem0.ai`** (`auto_import.py:45`) and requires a key
   (`resolve_api_key` from `_identity.py`). The seeding path is **hosted-only**.
   The OSS library has no equivalent.
3. **It is recent and it is a plugin, not a library feature.**
   `integrations/mem0-plugin/plugin.json` is at version `0.1.5`. This is not how
   Mem0 has historically worked; it is where Mem0 went.

### Letta seeds from uploaded documents, not from your repo

VERIFIED. The ingest path is folders (formerly "sources", now deprecated):
`letta/server/rest_api/routers/v1/folders.py:144` (`create_folder`), `:233`
(`upload_file_to_folder`). The old endpoints
`letta/server/rest_api/routers/v1/sources.py:123` and `:212` are both marked
`deprecated=True`. Processing machinery lives in
`letta/services/file_processor/` (`parser/`, `chunker/`, `embedder/`), and there
is a dedicated system prompt for it at
`letta/prompts/system_prompts/sleeptime_doc_ingest.py`.

A second path is importing a whole prebuilt agent:
`letta/server/rest_api/routers/v1/agents.py:464` (`import_agent`). This is
closer to "download someone else's configured memory" than to seeding from your
own material, but it is a genuine non-empty first run.

Neither reads a repository. Both require the user to decide what to upload.

### Graphiti, Zep and the MCP memory server are empty, full stop

**Graphiti.** VERIFIED by absence and by tool enumeration. `add_episode_bulk`
exists in core at `/Users/canh/Solo/OSS/graphiti/graphiti_core/graphiti.py:1230`
but is **not exposed as an MCP tool** — the thirteen tools registered in
`/Users/canh/Solo/OSS/graphiti/mcp_server/src/graphiti_mcp_server.py` (lines
342, 478, 555, 635, 661, 690, 717, 787, 840, 894, 968, 1004, 1047) contain no
bulk entrypoint. The bulk-ish paths that exist are demo scripts under
`examples/` (`podcast/`, `wizard_of_oz/`, `ecommerce/`, `quickstart/`), which
call core directly and are not product surface. Before the first `add_memory`,
search returns nothing.

**Zep.** Its MCP server cannot write at all — thirteen tools, all read-only,
registered at `mcp/zep-mcp-server/internal/server/server.go:71-87`, with
`README.md:8` stating "🔒 **Read-Only**: Safe, non-destructive operations."
Ingestion happens out of band through the Cloud SDK. No importer found.

**MCP memory server.** VERIFIED by fetched source: storage is a single JSONL
file, `loadGraph()` reads it, and there is no bootstrap, no seed, no importer.
The file format is plain JSONL, so a user *could* hand-author one, but the
server exposes no tool for it.

### Supermemory has real importers, and they do not work self-hosted

VERIFIED. `apps/browser-extension/utils/twitter-import.ts:61` implements a
`TwitterImporter` with cursor pagination and exponential backoff (`waitTime =
60000`, doubling, `:39-51`). Content scripts exist per site:
`apps/browser-extension/entrypoints/content/{chatgpt,claude,grok,t3,twitter}.ts`.
A batch endpoint is used at `apps/browser-extension/utils/api.ts:173`
(`/v3/documents/batch`). Cloud connectors — Drive, Gmail, Notion, OneDrive,
GitHub, S3, Granola, web crawler — are documented under `apps/docs/connectors/`.

The catch: `apps/docs/self-hosting/overview.mdx`'s feature table marks
**Connectors and MCP as unavailable ("—") for self-hosted**. The importers are a
cloud upsell. No browser-history importer was found despite the rumour;
UNVERIFIED at best, and searches for it surfaced only router / `window.history`
usage.

### What this means

The finding is **not** "nobody does cold-start seeding." It is sharper and less
comfortable:

> **Cold-start seeding exists, it is recent, it lives in the coding-agent
> integration rather than the library, and in both systems that have it, it is
> gated behind the hosted tier.**

For a product whose thesis requires n=1 value (D-003), that is good news and bad
news in the same sentence. Good: the approach is validated by the category
leader, so CRED is not inventing an unproven mechanism. Bad: it is no longer a
differentiator, and the cheap version of it — read the `AGENTS.md` the user
already wrote — is roughly 374 lines of Python, which means CRED's *harder*
version (walk git history, extract claims, anchor them to symbols) has to
justify the delta against something a competitor already ships.

---

## 3. The runtime loop, with real numbers

### Reads are cheap everywhere. Writes are where the money goes.

This is the single most consistent structural finding across all six systems,
and it inverts the intuition that retrieval is the expensive part.

| System | LLM calls per **search** | LLM calls per **add** |
|---|---|---|
| Mem0 (v3) | **0** | **1** |
| Mem0 (pre-v3) | 0 | 2 |
| Graphiti | **0** | **~3 + E + ⌈N/30⌉** floor; ~3 + 3E + N + ⌈N/30⌉ typical |
| Zep | 0 (cross-encoder rerank, not generative) | not in OSS |
| Letta | 0 for `archival_memory_search` | 1 agent step, + sleeptime agent every *k* turns |
| Supermemory | 0 (claimed) | unknown — binary |
| MCP memory server | **0** | **0** |

### Mem0: one LLM call on add, zero on search

VERIFIED by reading the v3 pipeline at
`/Users/canh/Solo/OSS/mem0/mem0/memory/main.py:835-1014`. The phases are
labelled in-source:

- Phase 0 — context gathering, `db.get_last_messages(..., limit=10)` (`:876`)
- Phase 1 — one embed (`:881`) + one vector search, `top_k=10` (`:882-887`)
- Phase 2 — **one** `self.llm.generate_response` (`:912-918`)
- Phase 3 — one batch embed of extracted texts (`:950`)
- Phase 5 — dedup by **MD5 hash only** (`:976-979`)
- Phase 6 — one batch insert (`:1007-1011`)

So one `add` is: **1 LLM call, 2 embedding calls, ~4 DB round trips**, fully
synchronous — `Memory.add` at `:721` returns only after
`_add_to_vector_store` completes (`:825`).

Search (`_search_vector_store`, `:1584-1663`) is **1 embed (`:1594`) + 1
semantic search + 1 keyword search + N entity-boost searches, and zero LLM
calls**. Over-fetch is `internal_limit = max(limit * 4, 60)` (`:1597`). Entity
boosts are parallelised across four threads (`:1734`).

**Mem0 halved its per-add LLM cost.** VERIFIED via `git show
a488e190^:mem0/memory/main.py`, the parent of the v3 port (`a488e190`,
"feat(oss): port v3 pipeline with hybrid search, entity extraction, and additive
scoring (#4805)", 2026-04-14): the old pipeline had `self.llm.generate_response`
at **two** points in the add path, lines 537 and 606 — extraction, then an
add/update/delete decision. v3 has one. The second call was deleted, not
optimised.

That deletion is the origin of Mem0's worst quality complaints. See §7.

### Graphiti: tens of LLM calls per episode, and it is superlinear in edges

VERIFIED by counting call sites in `add_episode`
(`/Users/canh/Solo/OSS/graphiti/graphiti_core/graphiti.py:980`, body
`:1122-1190`). For an episode yielding **N** entities and **E** edges:

| Stage | Site | Calls |
|---|---|---|
| Entity extraction | `graphiti.py:1122` → `node_operations.py:275` | 1 |
| Node dedupe | `graphiti.py:1131` → `node_operations.py:552` | 1 (batched) |
| Edge extraction | `graphiti.py:1139` → `edge_operations.py:202` | 1 |
| Edge resolution, **per edge** | `edge_operations.py:623`, `:726`, `:662`, `:598` | **E to 3E** |
| Node attributes, **per node** | `graphiti.py:1160` → `node_operations.py:804` | 0 to N |
| Node summaries, batched at `MAX_NODES = 30` | `node_operations.py:63`, `:976` | ⌈N/30⌉ |
| Communities (opt-in) | `graphiti.py:1185-1190` | N |

**A modest episode of N=8, E=10 is roughly 41 LLM calls for one `add_memory`.**

DB cost tracks it: edge resolution issues **two full hybrid searches per
extracted edge** — `edge_operations.py:392-403` (duplicate candidates) and
`:407-418` (invalidation candidates), both `EDGE_HYBRID_SEARCH_RRF`, each BM25 +
cosine. That is ≈4E index queries before node dedupe or writes.

Users measured this and it is worse than the arithmetic suggests.
`graphiti#1516` ("add_episode is impractically slow for >5KB content") reports
**~626 s for a one-sentence episode with 3 entities**, projects **30–50 min for
5 KB of markdown** and **5–10 hr for 49 KB**, and independently counts ~160 LLM
calls for a 5 KB doc. `graphiti#1262` reports **60 min for 100 records** via
`add_episode_bulk` — 36 s/record — with one batch of ten taking 348,108 ms. Both
VERIFIED via `gh`.

Graphiti's mitigation is deferral, not reduction. `add_memory`'s docstring at
`mcp_server/src/graphiti_mcp_server.py:359-361` states it "returns immediately
and processes the episode addition in the background," implemented as a
per-group asyncio queue with one worker per group
(`services/queue_service.py:18`, `:38`, `:44-45`, `:49`). **A search issued
immediately after a write can legitimately miss the data.** That is a
correctness property, not a performance detail, and it matters for any agent
that writes then reads within a turn.

Its escape hatch is `add_triplet` (`graphiti_mcp_server.py:894`), whose
docstring at `:905-907` says: "Unlike add_memory, this bypasses episode
extraction and writes the relationship directly." The existence of this tool is
an admission.

Graphiti's search, by contrast, costs **zero LLM calls**: the MCP server selects
`NODE_HYBRID_SEARCH_RRF` or `NODE_HYBRID_SEARCH_NODE_DISTANCE`
(`graphiti_mcp_server.py:522-528`), whose recipes use
`reranker=NodeReranker.rrf` / `node_distance`
(`search/search_config_recipes.py:156-161`, `:172-177`), not `cross_encoder`.
One embedding call (`search/search.py:148`) plus two index queries.

### Zep publishes the only real latency numbers in the category

This is the number CRED's design has to beat or match, and it is the only one in
this document that comes from committed measurement artifacts rather than
counting or claims.

VERIFIED — I re-extracted these myself from
`benchmarks/locomo/experiments/*/experiment_summary.json` at Zep SHA
`0375d7be4a72cda6a43ecdc6fd9055846eb0fd0e`, rather than accepting them
second-hand. Ten runs per experiment, LoCoMo, `gpt-4o-mini` responder and grader
at temperature 0.

| Experiment | edge/node limit | Accuracy | Retrieval median | p95 | p99 | Context tokens (median) |
|---|---|---|---|---|---|---|
| 20251207_215609 | 30 / 30 | 0.8032 | 0.189 s | 0.437 s | 0.885 s | 1997 |
| 20251203_171523 | 20 / 20 | 0.8006 | **0.241 s** | **0.576 s** | **0.933 s** | 1378 |
| 20251203_184539 | 15 / 5 | 0.7706 | 0.199 s | 0.503 s | 0.906 s | 756 |
| 20251204_225237 | 10 / 2 | 0.7372 | 0.161 s | 0.344 s | 0.626 s | 504 |
| 20251207_182039 | 5 / 2 | 0.6962 | 0.149 s | 0.332 s | 0.536 s | 347 |

For the 20/20 run, the full distribution is median 0.2408 s, mean 0.2877 s, p95
0.5760 s, p99 0.9326 s, min 0.1186 s, **max 5.5626 s**, std dev 0.1700 s.

**Three caveats, all load-bearing:**

1. **These are Zep Cloud numbers measured over the network.**
   `benchmarks/locomo/benchmark.py:39` instantiates
   `AsyncZep(api_key=os.getenv("ZEP_API_KEY"))` and `ingestion.py:13` imports
   from `zep_cloud.client`. This is not a local retrieval measurement. A
   self-hosted engine should be able to beat it on the same hardware simply by
   removing a network hop.
2. **Concurrency was 20** (`config.evaluation_concurrency`). These are latencies
   under load, not single-request latencies.
3. **All five configs use `cross_encoder` for both edge and node reranking.**
   D-010 cut the cross-encoder from CRED v1 in favour of MaxSim. Zep is paying
   for a reranker inside these sub-second numbers. That makes the comparison
   *more* favourable to CRED, not less — but it also means the accuracy column
   is not a like-for-like target.

The accuracy/latency/context curve is clean and worth internalising: going from
5/2 to 30/30 buys **+10.7 accuracy points for +40 ms of median retrieval and
5.7x the context tokens.** Retrieval latency is not the binding constraint at
this scale. Context budget is.

### The only published latency budgets in the category are hook timeouts

This is the most practically useful finding in this section, and it comes from
an unexpected place. Mem0's coding-agent plugin declares per-hook timeouts in
`/Users/canh/Solo/OSS/mem0/integrations/mem0-plugin/hooks.json`. Those timeouts
**are** a latency budget, stated by a shipping product, for exactly CRED's
deployment shape:

| Hook | Timeout | What it does |
|---|---|---|
| `UserPromptSubmit` | **8 s** | Prefetch memories for the current prompt — blocks the user |
| `Stop` | 30 s | Session summary capture |
| `PreToolUse` (Read) | 5 s | File-context injection |
| `PostToolUse` (Bash) | 5 s | Bash-output capture |
| `PostToolUse` (mem0 tools) | 3 s | Metadata bookkeeping |
| `SessionStart` (deps) | 60 s | Dependency bootstrap |

Inside that, the actual network budget is tighter still: `scripts/_search.py:14`
sets `SEARCH_TIMEOUT = 5`, and `:20-24` records the reranking decision in plain
terms —

> the extra ~150-200ms is well within the hook's curl budget

VERIFIED. So the operative numbers are: **~5 s hard ceiling on a blocking
retrieval, with ~150–200 ms treated as an acceptable increment.** Mem0's docs
also claim "Sub-50ms retrieval" (`docs/vibecoding.mdx:92`) — that is a marketing
claim with no methodology attached, and it should be read as a claim, not a
fact.

Separately, `mem0#5850` reports `add()` p50 degrading **0.55 ms at 100 memories
→ 31.76 ms at 10 K — 58x** with `infer=False`, i.e. storage-layer degradation
independent of any LLM cost. VERIFIED.

### Letta: the cost is context, not calls

Letta's memory is in-context by construction, so per-turn cost is the agent step
itself. Its deferred-work mechanism is the **sleeptime agent**: a second agent
that edits memory in the background, triggered on a turn counter —
`letta/groups/sleeptime_multi_agent_v2.py:118-119` fires when `turns_counter %
self.group.sleeptime_agent_frequency == 0`, dispatched via
`_issue_background_task` (`:231`, called at `:126` and `:223`). The counter
wraps at `letta/services/group_manager.py:264`.

`enable_sleeptime` defaults to `None` (`letta/schemas/agent.py:143-146`) and
`sleeptime_agent_frequency` defaults to `None`
(`letta/schemas/group.py:43`) — **off unless configured.**

Letta instruments latency thoroughly — `ttft_ns` and `total_duration_ns` on runs
(`letta/schemas/run.py:49-50`), `step_ns` per step
(`letta/schemas/step_metrics.py:23`), `latency_ms` on LLM traces
(`letta/schemas/llm_trace.py:86`), OTel histograms `hist_ttft_ms` and
`db_checkout_latency_ms` (`letta/otel/metric_registry.py:120-125`, `:403-408`) —
but **asserts no budget anywhere.** It emits numbers; it does not claim them.

Neither does Graphiti. Its one performance statement is a comparison table at
`/Users/canh/Solo/OSS/graphiti/README.md:153` — "Query Latency | Seconds to tens
of seconds | Typically sub-second latency" — versus GraphRAG, with no
methodology and no backing artifact. UNVERIFIED as a performance claim.

### Summary: the latency target

**CRED's blocking recall budget is ~5 s hard, sub-600 ms p95 to be competitive,
and sub-250 ms median to be clearly better than the best published number in the
category.** The p95 figure is Zep's, over a network, under concurrency 20, with
a cross-encoder. A local Go engine with MaxSim should clear it. If it does not,
that is a finding worth knowing early — which argues for measuring it in the
first slice rather than the last.

---

## 4. How memory actually gets written in practice

This section answers the question D-009 depends on, and the answer contradicts
D-009.

### The category has split into two camps, and the winning camp writes automatically

| System | Trigger | Default |
|---|---|---|
| **Mem0 (plugin)** | **Session start, every 3rd user prompt, every Bash tool result, every session Stop** | **ON** |
| Mem0 (library) | Explicit `add()` | n/a |
| Graphiti | Explicit `add_memory` tool call | n/a |
| Zep | Cannot write via MCP at all | n/a |
| Letta | Agent calls a memory tool; sleeptime agent every *k* turns | sleeptime OFF |
| Supermemory | Explicit `memory` tool; extraction automatic *after* ingest | auto-capture OFF |
| MCP memory server | Explicit tool call + a system-prompt convention | n/a |

### Mem0's shipped integration writes constantly, by default

VERIFIED from `/Users/canh/Solo/OSS/mem0/integrations/mem0-plugin/hooks.json`
and `scripts/on_user_prompt.sh`. This is not the OSS library and it is not the
`openmemory` MCP server; it is what Mem0 actually ships to coding agents today.

Eight hooks are registered across `SessionStart`, `UserPromptSubmit`,
`PreToolUse` (matching `Write|Edit|MultiEdit`, `mcp__mem0__.*`, and `Read`),
`Stop`, and `PostToolUse` (matching `mcp__mem0__.*` and `Bash`).

`on_user_prompt.sh` is worth reading in full; the behaviour it encodes:

- **Reads on every prompt over 20 characters.** Prefetch is on by default and
  runs synchronously in the hook: `[ "${MEM0_PREFETCH:-true}" != "false" ]`,
  searching with the raw prompt at `top_k=5` and injecting results as
  `additionalContext`. Opt-**out**, not opt-in.
- **Writes every third message, in the background.**
  `[ "${MEM0_AUTO_SAVE:-true}" != "false" ] && [ $((MSG_COUNT % 3)) -eq 0 ]` →
  `python3 "$SCRIPT_DIR/auto_capture.py" "$TRANSCRIPT_PATH" &`. Default **true**.
  `auto_capture.py:186` takes the last 4 exchanges; `:44-45` sets
  `MAX_CONTENT_CHARS = 8000`, `MIN_CONTENT_CHARS = 100`; `:141` posts with a 15 s
  timeout.
- **Nudges the model to write, every fifth message.**
  `if [ $((MSG_COUNT % 5)) -eq 0 ]` sets a save nudge, and a fallback check fires
  whenever `_ADDS < MSG_COUNT / 3`, injecting: "After responding, store any new
  decisions, learnings, or preferences from this exchange via add_memory."
- **Injects a standing retrieval rubric** once per session telling the agent to
  run "2-4 parallel calls with different metadata.type filters" and that "one
  search is rarely enough."
- **Pattern-matches the prompt with grep** for stack traces (`Traceback`,
  `panic:`, `fatal:`), file paths, session-resume phrasing, and explicit
  remember-intent — all without an API call — and branches on the results.

The tool descriptions push the same direction.
`openmemory/api/app/mcp_server.py:64`: "Add a new memory. **This method is
called everytime the user informs anything about themselves**, their
preferences, or anything that has any relevant information which can be useful
in the future conversation." And `:149`: "Search through stored memories. **This
method is called EVERYTIME the user asks anything.**" Emphasis in the original
capitalisation.

### The others are explicit, but two of them fake automaticity through prompting

The MCP memory server has no hooks at all. Its README instead instructs the user
to install a system prompt — "Always begin your chat by saying only
'Remembering...' and retrieve all relevant information from your knowledge
graph" — tracking identity, behaviours, preferences, goals and relationships.
VERIFIED by fetch. "Automatic" memory there is a convention the user installs by
hand, not a property of the system.

Supermemory's MCP tool descriptions are more aggressive still.
`apps/mcp/src/server.ts:134-135` opens the `memory` tool's description with
"**DO NOT USE ANY OTHER MEMORY TOOL ONLY USE THIS ONE.**" and `:146-147` says
the same for `recall`. That is prompt-level suppression of competing memory
tools — relevant to anyone planning to run more than one memory MCP server, CRED
included.

Supermemory's browser auto-capture is genuinely opt-in:
`apps/browser-extension/utils/storage.ts:37-41` (`autoSearchEnabled`, `fallback:
false`) and `:44-48` (`autoCapturePromptsEnabled`, `fallback: false`).

### Plainly: this contradicts D-009

D-009 states "CRED's first run reads and never writes. Contribution — the act of
storing a claim — is a deliberate, separate step the user takes after they have
already gotten value from recall," and rules out "automatic background
contribution before explicit opt-in, however useful."

**The category leader ships exactly the ruled-out behaviour, on by default, at
three separate trigger points, and treats it as the product.** Mem0's plugin
writes on the third message of the first session, before the user has evaluated
anything.

What D-009 got right and what it got wrong are separable, and the distinction
matters — see §9.

---

## 5. Tool surfaces compared

`context7.md` established the benchmark: **two read-only tools, two string
parameters each, 59,457 stars, 3.7M npm downloads/month.** Here is the
comparison it was missing.

| System | Tools | Read-only? | Notes |
|---|---|---|---|
| **Context7** | **2** | Yes | The adoption benchmark |
| Mem0 (`openmemory`) | **5** | No | `add_memories`, `search_memory`, `list_memories`, delete-by-id, `delete_all_memories` |
| Supermemory | 7 | No | 5 general + 2 app-only, plus 2 resources and 1 prompt |
| MCP memory server | 9 | No | 6 mutating, 3 reading |
| **Zep** | **13** | **Yes** | All read-only by design |
| **Graphiti** | **13** | No | Includes `clear_graph` |
| **CRED (committed)** | **4** | No | `recall`, `remember`, `revise`, `confirm` |

### The exact surfaces

**Mem0 `openmemory`**, all in
`/Users/canh/Solo/OSS/mem0/openmemory/api/app/mcp_server.py`:
`add_memories(text: str, infer: bool = True)` (`:64-65`), `search_memory(query:
str)` (`:149-150`), `list_memories()` (`:227-228`), delete-by-ID (`:296`),
`delete_all_memories()` (`:370-371`). Note that `add_memories` exposes `infer` —
the flag that switches off the LLM extraction path entirely (`main.py:836-870`).
A verbatim-store escape hatch on the primary write tool.

**Graphiti**, thirteen tools at the line numbers listed in §2. The headline
signature is long: `add_memory(name, episode_body, group_id=None, source='text',
source_description='', uuid=None, reference_time=None,
excluded_entity_types=None, custom_extraction_instructions=None,
previous_episode_uuids=None, update_communities=False, saga=None,
saga_previous_episode_uuid=None)` — `graphiti_mcp_server.py:342-356`. **Thirteen
parameters on the primary write tool**, against Context7's two-strings-per-tool.

**Zep**, thirteen read-only tools at
`mcp/zep-mcp-server/internal/server/server.go:71-87`: `search_graph`,
`get_user_context`, `get_user`, `list_threads`, `get_user_nodes`,
`get_user_edges`, `get_episodes`, `get_thread_messages`, `get_node`, `get_edge`,
`get_episode`, `get_node_edges`, `get_episode_mentions`. `search_graph` alone
takes two required and eight optional parameters (`docs/TOOLS.md:29-40`).

**MCP memory server**, nine tools: `create_entities`, `create_relations`,
`add_observations`, `delete_entities`, `delete_observations`,
`delete_relations`, `read_graph`, `search_nodes`, `open_nodes`.

**Supermemory**, in `apps/mcp/src/server.ts`: `memory` (`:132`), `recall`
(`:144`), `listMemories` (`:156`), `listProjects` (`:228`), `whoAmI` (`:291`),
plus app-only `memory-graph` and `fetch-graph-data`.

### Which tools get called most

**No system publishes call-frequency data.** I found no telemetry dashboards, no
usage breakdowns in docs, and no issues reporting per-tool call counts.
UNVERIFIED and likely unobtainable from public sources.

The best available proxy is what each project *tells the model* to call, which
is observable in tool descriptions and system prompts. On that evidence the
answer is unambiguous: **search is the tool that gets called.** Mem0 instructs
"EVERYTIME the user asks anything" for `search_memory` versus a conditional for
`add_memories`; Mem0's own rubric says "one search is rarely enough" and asks
for "2-4 parallel calls"; the MCP memory server's recommended prompt opens
*every* conversation with a retrieval. Zep shipped thirteen tools and zero write
tools.

### The relevant comparison for CRED

Tool count does not predict adoption. Zep and Graphiti both ship thirteen; the
MCP memory server ships nine and is the most-installed memory server by
distribution; Context7 ships two and won its slot.

What separates them is **whether the surface is comprehensible in one reading**.
Context7's two tools take two strings. Graphiti's `add_memory` takes thirteen
parameters. CRED's four tools sit closer to Context7 than to anything else in
this table, and `recall`/`remember`/`revise`/`confirm` are self-describing in a
way `search_memory_facts` versus `search_nodes` is not — a distinction Graphiti
users have to learn and that has no analogue in CRED's model.

**Four tools is defensible on this evidence.** D-009 said "the burden is on each
tool to justify itself." Against a category median of nine, four is not the
thing to cut. The real question is which of the four appears in the *first*
slice — see §9.

---

## 6. Required configuration, counted

Against CRED's stated goal of `docker compose up` reaching a working instance
with no additional steps (PRD acceptance criterion 11).

| System | Required decisions | Required secrets | Cold-start value without a key |
|---|---|---|---|
| **MCP memory server** | **0** | **0** | Works, but empty |
| **Context7** (benchmark) | 0 | 0 | **Works, full corpus** |
| Supermemory (self-host) | 1 | 1 LLM key | None |
| Zep MCP | 1 | `ZEP_API_KEY` | None — cloud account required |
| Mem0 (self-host server) | 2 | `OPENAI_API_KEY`, `JWT_SECRET` | None |
| Letta | 2–3 | LLM key, embedding handle, DB choice | None |
| **Graphiti** | **5** | `OPENAI_API_KEY`, `NEO4J_PASSWORD` | None |

### The details

**MCP memory server: zero.** VERIFIED. Exactly one env var exists and it is
optional — `MEMORY_FILE_PATH`, defaulting to a path beside the module. No API
key, no DB, no LLM, no embedding model, no vector store. This is the floor.

**Graphiti: five.** VERIFIED from `mcp_server/config/config.yaml`, expanded at
`mcp_server/src/config/schema.py:32`. Required with no default: `OPENAI_API_KEY`
(`config.yaml:20` for LLM, `:52` for embedder) and `NEO4J_PASSWORD` (`:85`). The
five decisions are: (1) LLM provider from {openai, azure_openai, anthropic,
gemini, groq} (`:14`); (2) LLM model; (3) embedder provider (`:45`) plus model
plus a `dimensions` value (`:48`) that must match the model; (4) Neo4j vs
FalkorDB (`:76-87`), and standing one up; (5) `SEMAPHORE_LIMIT`, which
`config.yaml:4-6` flags as load-bearing for rate limits.

**Mem0 self-hosted: two required.** VERIFIED from
`docs/open-source/setup.mdx:34-42`. `OPENAI_API_KEY` and `JWT_SECRET` are both
marked Required; the doc warns "The server refuses to start if `JWT_SECRET` is
unset once auth is enabled" and that
1.x deployments "will return `401` on every protected endpoint." Postgres is
bundled; ports 8888 and 3000 must be free. Notably, **auth defaults to on** — a
recent, deliberate move away from the frictionless install.

**Supermemory: one.** `apps/docs/self-hosting/configuration.mdx:8` — "the only
required input is one LLM provider key." Everything else defaults: port 6767,
data dir `./.supermemory`, local `Xenova/bge-base-en-v1.5` embeddings at 768
dimensions. Install is `curl -fsSL https://supermemory.ai/install | bash`. This
is the best self-hosted onboarding in the set — and see §7 for what it costs.

**Letta: two to three.** `letta/settings.py:472-493` — Postgres if
`letta_pg_uri` is set, else SQLite, so the DB decision has a working default.
`openai_api_key` (`:133`) / `anthropic_api_key` (`:180`) and
`default_embedding_handle` (`:298`) have no defaults. Letta's headline path is
now a CLI: `README.md:14-15` is two steps, `npm install -g @letta-ai/letta-code`
then `letta` — but the letta-code README (fetched) confirms `/connect` is still
required to configure LLM keys.

### The uncomfortable comparison

CRED's target is `docker compose up` with no additional steps. **Nothing in this
table achieves that, except the two systems that need no LLM at all.**

That is the actual finding. Zero-config is not a packaging achievement; it is a
consequence of not calling a model on the hot path. Context7 has no key because
it serves a pre-built corpus. The MCP memory server has no key because it does
substring matching on a JSONL file. Every system that extracts, embeds or
reranks needs a key, and none of them found a way around it.

For CRED this cuts a specific way. D-008 committed to pure-Go embeddings with
`CGO_ENABLED=0`, and D-010 cut the cross-encoder. **Embedding locally is the
only mechanism in this entire scan that removes the mandatory API key from the
read path.** That decision is worth more operationally than the
retrieval-quality argument that motivated it, and it is currently the strongest
claim CRED has to the zero-config slot — provided nothing in the *read* path
requires an LLM. Per §3, in Mem0, Graphiti and Zep, nothing does. Read paths are
LLM-free across the category. **CRED can plausibly ship a read path with zero
required configuration. No competitor has.**

---

## 7. Operational complaints from issue trackers

Sampled with `gh issue list` / `gh issue view` across six repositories. Ranked
by weight of evidence in the sample, not by census.

### 1. Ingest latency (worst: Graphiti)

Covered in §3. `graphiti#1516`: ~626 s for a one-sentence episode; 5 KB
projected at 30–50 min; 49 KB at 5–10 hr. `graphiti#1262`: 60 min for 100
records. `graphiti#1592`, `#1272`, `#1506`: FalkorDB `edge_fulltext_search`
doing a node-by-label scan; "add_episode times out at scale." VERIFIED.

Mem0's latency issues are narrow config gaps by comparison — a hardcoded 8 s
`RECALL_TIMEOUT_MS` (`#6307`), a missing OpenSearch timeout (`#5835`).

### 2. LLM cost, and O(n) context growth on the write path

`graphiti#467` ("LLM inference expenses are pretty high", 8 comments): **~$0.80
for 40 chats of 150–250 words each**, with custom entities disabled; the
reporter has "thousands of chats" and asks for a 5–10x reduction.

`graphiti#1275` is the structural version: `resolve_extracted_nodes` sends all
graph nodes to the LLM, so context grows O(n). At **~300+ episodes** it hits
`Output length exceeded max tokens 16384` (`completion_tokens=16384,
prompt_tokens=17894, total_tokens=34278`), **all five worker retries fail, and
the episode is silently dropped.** VERIFIED.

`mem0#2820` ("Include OpenAI Token Usage in All Relevant Method Responses", 8
comments) is the same problem one level down: users cannot measure their spend.

`modelcontextprotocol/servers#2415`: "Large memory files (80k+ tokens) result in
expensive queries" — the whole graph is loaded per read.

### 3. Deduplication does not work — present in every repo

The most universal theme in the sample, and the one most directly relevant to
CRED's curation worker.

`mem0#5850` is the controlled experiment: **5 unique facts in 3 wordings each
(15 inserts) produced 13 memories instead of 5 — the LLM caught 2 of 10
duplicates, a 20% catch rate.** Maintainer triage confirms the mechanism: v3
shows the LLM only the top-10 nearest existing memories, and the only hard check
is an **exact MD5 hash** match (`main.py:976-979`, which I read directly), so
paraphrases always land. VERIFIED.

`mem0#4956` is the causal explanation: the `ADDITIVE_EXTRACTION_PROMPT` says
"Your sole operation is ADD," so Ronaldo→Messi leaves **both** facts stored. The
maintainer states this is a **cost-driven** tradeoff — avoiding a per-memory
UPDATE/DELETE round-trip. This is the second LLM call that `a488e190` deleted
(§3), and the quality complaints are its direct consequence.

`letta#3116` ("Archival Memory Deduplication and Consolidation", 8 comments):
worked example of four passages all storing "user's favorite color is blue."
Core memory has `rethink`/`replace`/`insert`; **archival memory has no
consolidation mechanism at all.**

`graphiti#963`, `#875`: duplicate episodic nodes for the *exact same* episode,
following the official quickstart. Also `#1101` (duplicate edges
miscategorised), `#994` ("the edge detection of edge duplicated is bad"), `#789`
(BFS returns duplicate edges with swapped source/target).

### 4. Silent data loss — writes accepted, memories never persisted

The most dangerous class, because the API returns success.

`mem0#5245` — **18 comments, the single most-discussed open mem0 issue.** When
`embed_batch` fails and the per-item fallback throws, the text never enters
`embed_map`, is logged at WARNING, and **no exception reaches the caller.** I
verified this shape directly in source: `main.py:952-959` catches per-text embed
failures with `logger.warning` and continues; `:973` then silently skips any
text missing from `embed_map`. `mem0#5509` is the same bug in the TS SDK.
`mem0#4985`: switching embedding provider silently drops writes on dimension
mismatch.

`graphiti#566` — 13 comments — "API /messages endpoint returns **202 but does
not persist episodes in Neo4j** (even after fresh install)." `#1164`: episodes
fail silently when content contains the word `attributes`.

`supermemory#792`: **3,060+ docs and 91+ memories visible in the UI**, but v3
and v4 search, hybrid search, and MCP `recall` all return zero. The reporter's
summary: "Write path works perfectly. Read path is completely broken."
`#1300`/`#1301`/`#1302`: memory-agent timeout discards memories the model
already produced and marks the doc `done` with 0 memories; re-ingesting
unchanged content is a silent no-op, **so a doc that fails extraction can never
be repaired through the API.**

### 5. Self-host setup and upgrade breakage (worst: Supermemory)

Supermemory has seven-plus distinct upgrade-breaks-everything reports across
three point releases: `#1293` (v0.0.5 skips the `profile_buckets` migration →
`/v4/profile` 500), `#1314` (migration "schema 'observatory' already exists"),
`#1315` (v0.0.6 linux-x64 — "ingestion and search silently broken",
`@rivetkit/rivetkit-wasm` not bundled into the standalone binary), `#1291` (all
`POST /v3/documents` fail after v0.0.3→v0.0.5), `#1237` (API-key auth 401),
`#1296` (server survives `kill -9`, holds :6767 until reboot), `#1103` (search
returns `total:0` on any store upgraded from v0.0.1).

Graphiti: `#1307` ("Docker image is still old"), `#1624` (MCP Dockerfile build
fails — falkordb:latest dropped perl-base), `#1623` (docker-compose interpolates
host `$PATH` into container env, crashing on Windows), `#1059` (no
Neo4j→FalkorDB migration path without data loss), `#1108` (FalkorDB data missing
after v0.23.0).

Mem0 is milder and mostly packaging: `#4945`, `#5090`, `#5340`, `#4816`
(production Dockerfile missing libpq5, runs as root, `--reload` in prod).

### 6. Retrieval quality — right data stored, wrong data returned

`mem0#5742` (hybrid search drops keyword-only and entity-rescued memories),
`#6448` (`search(rerank=True)` reranks only the truncated top-k, so reranking
cannot improve recall), `#5909` (Elasticsearch caps at 10 — missing `size` on
the KNN query), `#4884` (**BM25 keyword search and entity extraction are
hardcoded to English**, 9 comments), `#6297` (Weaviate BM25 runs a lemmatized
query against the raw `data` field), `#5438` (same-name entities with different
meanings merge at a 0.95 similarity threshold).

`mem0#5730` deserves separate note: `custom_fact_extraction_prompt` was
**downgraded from a full override to an appended section** (#4740→#4805), i.e. a
deliberate regression in configurability, and self-hosted users have no way to
dial down over-extraction.

`graphiti#1642` (cross-encoder reranking drops most candidates in
`edge_search`), `#1302` (`lucene_sanitize()` escapes individual uppercase
letters, **breaking BM25 for most queries**), `#534` (`retrieve_episodes` always
returns no results).

`supermemory#1104` (6 comments, top open issue): hard-coded
`Xenova/bge-base-en-v1.5` gives "noticeably weaker semantic recall on our German
memories"; a commenter reverse-engineered the binary and found **two**
hard-coded layers — model ID *and* a fixed `vector(768)` pgvector column — and
is holding off backfilling to avoid embedding everything twice.

### 7. Context compaction destroying history (unique to Letta)

`letta#3270` — **14 comments, Letta's top open issue.** With
`sliding_window_percentage: 0.15` on a 45,000-token window, **compaction took
~31 minutes and left only the summary message.** Reproduced at 0.15, 0.25 and
0.50. Two independent commenters traced it: `goal_tokens` is computed against
context-window *capacity* rather than current usage, and `approx_token_count` is
initialised to the full window, so the loop escalates to `>= 1.0`, raises
`ValueError`, and `compact.py:327-328` catches it with a bare `except Exception`
and **silently falls back to `mode: all`** — a full wipe. Aggravated for
non-OpenAI models by tiktoken over-counting plus a 1.3x safety margin.

Also `letta#3279`, `#3242`, `#3247` (**OpenAIProvider ignores API-reported
`context_length`, defaults all unknown models to 30k tokens**).

### 8. Concurrency and store corruption

`modelcontextprotocol/servers#1819`: multiple tool calls from a *single LLM
response* corrupt the JSON file; "once the file is corrupted it is not possible
for the tool to recover/fix by itself, **every operation fails**."
`#4117` — **19 comments, the most-discussed memory-server issue** — audits
`@modelcontextprotocol/server-memory@2026.1.26` and finds whole-file writes with
no atomic replace or lock discipline, no mutation journal, no backup trail.
`#4457`: `create_relations` accepts relations to non-existent entities — silent
graph corruption.

`mem0#4892` — 12 comments, the #2 open issue — **20 concurrent writers
reproducibly corrupt the Qdrant HNSW index**; subsequent `search()`/`get()`
return wrong results or raise until process restart. Root cause is
`asyncio.to_thread` dispatch of non-re-entrant Qdrant upserts.

### 9. Config silently ignored

`modelcontextprotocol/servers#1018` (14 comments): repo source honours
`MEMORY_FILE_PATH`; **the npm-published build does not.** `#692` (12 comments):
memory lands in `~/.npm/_npx/<hash>/node_modules/.../dist/memory.json` —
**inside an npx temp dir that reinstalls wipe.** Combined with #4117's finding
that the default path is the package `dist` directory, that is a data-durability
bug, not a config annoyance.

Same shape elsewhere: `mem0#4279` (vector store defaults to `/tmp/{provider}`),
`mem0#5351` (plugin `rerank`/`keywordSearch` keys "are inert (never sent to
search)"), `graphiti#1116` (OpenAI provider ignores `api_base`, 8 comments),
`graphiti#763` (`LLMConfig.max_tokens` not respected, 8 comments),
`graphiti#1393` (MCP reranker hardcoded to OpenAI, requires `OPENAI_API_KEY`
even with non-OpenAI providers).

### 10. Resource exhaustion (essentially Supermemory-only)

`supermemory#1093` (fixed): server reaches **~22 GB RSS, peak ~50 GB before
being killed**, climbing 3–4 GB/min during steady ingestion on a 16 GB machine.
Emscripten `WebAssembly.Memory.grow()` is monotonic to ~4 GB per worker, so peak
≈ `POOL_SIZE × 4 GB`. `#1177` (open): past ~150 MB DB, the encrypted snapshot
base64-encodes the entire DB into one string and hits V8's `String::kMaxLength`;
`RangeError: Out of memory` **136 times in one log**, at under 2 GB RSS,
cascading into a permanent retry-cron loop. `#1203`: documents over ~64 KiB
permanently wedge the ingest queue — never `done`, never `failed`, undeletable
(409 loop).

### The causal chain that runs through all of it

The strongest pattern in the data is **cost → design shortcut → quality
complaint.** Mem0's maintainer states outright that ADD-only extraction exists
to avoid a second LLM round-trip; that shortcut directly produces the
duplicate-accumulation and stale-fact complaints in `#5850` and `#4956`.
`graphiti#1275` is the mirror image: it pays the O(n) cost and fails at ~300
episodes, and its reporter explicitly cites Mem0's vector-first/LLM-second
approach as the fix. **Neither tradeoff is escapable by configuration in either
system.**

That is the seam CRED is aimed at. Deterministic invalidation and exact
deduplication are not cheaper approximations of what these systems do — they are
a different answer to the question that is making both of them fail.

---

## 8. Multi-user at runtime

Restricted to what `mem0.md` §5 and `the-three-retreats.md` do not already
cover: how a principal is threaded through a call in the open code.

**Mem0: a filter dictionary, resolved at the call site.** `Memory.add`
(`main.py:721-733`) takes `user_id`, `agent_id`, `run_id` as keyword arguments,
which `_build_filters_and_metadata` (`:778-783`) turns into `effective_filters`.
Those become `search_filters` at `:880` — `{k: v for k, v in filters.items() if
k in ("user_id", "agent_id", "run_id") and v}` — and are passed to the vector
store. **There is no principal in the engine; there is a filter string supplied
by the caller.** Any client that passes the wrong `user_id` reads another user's
memories. Enforcement lives entirely in whoever calls the library.

In the plugin, the identity is resolved from the environment:
`scripts/_identity.py` `resolve_user_id()` / `resolve_api_key()`, with
`resolve_project_id()` and `resolve_branch()` from `_project.py`. Scoping is
per-machine-user plus per-repo plus per-branch, decided client-side.

**Letta: an actor, resolved server-side from a header.** Every v1 route resolves
`actor = await
server.user_manager.get_actor_or_default_async(actor_id=headers.actor_id)`
before doing anything — e.g. `letta/server/rest_api/routers/v1/sources.py:120`.
Managers take `actor` explicitly (`organization_manager.py`,
`identity_manager.py`, `block_manager.py`). This is a genuine server-side
principal, threaded through the service layer, not a caller-supplied filter.

**Zep and Supermemory: `user_id` / `containerTag` as a required tool
parameter.** Zep's `search_graph` requires `user_id` (`docs/TOOLS.md:29-40`),
i.e. the model supplies the principal. Supermemory's graph tools take
`containerTag`.

**The relevant comparison.** D-014 put org, member and role in the engine rather
than a client. Letta is the only system in this scan that already does that, and
it is the only one whose team story survived (`the-three-retreats.md`). Mem0's
model — a filter the caller supplies — is the one that makes recall-time ACL
enforcement structurally impossible, because there is nothing to enforce
against. **This is confirming evidence for D-014, from the runtime rather than
from the pricing pages.**

---

## 9. What CRED's first implementation slice should be

The proposal on the table: **one MCP tool, a CLI, cold-start seeding from git
history, read-only.**

Tested against the evidence above, it is **mostly right, and wrong in one
specific and fixable way.**

### What the evidence supports

**One tool, read-only, is right.** Read paths across the entire category cost
zero LLM calls (§3) — Mem0, Graphiti, Zep, all of them. That means a read-only
first slice is the only version of CRED that can plausibly ship with **zero
required configuration**, which no memory system in this scan achieves (§6). The
two systems that do achieve it — Context7 and the MCP memory server — are the
two that never call a model. D-008's pure-Go embedding decision is what puts
CRED in that category, and the first slice is where that advantage is either
realised or wasted. **Ship the read path with no API key, or the decision bought
nothing.**

**A CLI is right, and for a reason the proposal probably did not have.** The
single most reliable failure mode in the sample is **silent write acceptance
followed by empty reads** — `supermemory#792` ("write path works perfectly, read
path is completely broken", with 3,060 docs visible in the UI), `mem0#5245` (18
comments), `graphiti#566` (202 with no persistence). In every case users could
see that data went in and could not tell why nothing came out. A CLI that shows
what is stored, what matched, and what scored is the diagnostic those users did
not have. It is not developer convenience; it is the thing that makes the
failure legible.

**Deferring `remember` is right, but not for D-009's reason.** See below.

### What the evidence contradicts: seed from documentation first, not git history

The proposal says cold-start seeding from **git history**. Mem0 ships cold-start
seeding from **declarative documentation** — `CLAUDE.md`, `AGENTS.md`,
`.cursorrules`, `.windsurfrules` (`auto_import.py:47`) — in 374 lines of Python
with SHA-256 change detection.

Prefer documentation for the first slice, on three grounds:

1. **It is validated.** It is the one cold-start mechanism in this scan that a
   successful competitor actually ships. Git-history extraction is shipped by
   nobody, which is either an opportunity or a warning, and the first slice is
   the wrong place to find out which.
2. **The input is already curated prose.** `AGENTS.md` is a human-written
   statement of how a project works. Commit messages are not. Every extraction
   quality complaint in §7 — the 20% dedup catch rate in `mem0#5850`, the
   over-extraction in `mem0#5730` — comes from extracting facts out of
   unstructured conversational text. Seeding from documentation sidesteps the
   category's worst-evidenced failure mode entirely.
3. **It has a cleaner evidence chain, which is CRED's actual thesis.** A claim
   whose evidence is `AGENTS.md:42` has a live file and a byte range behind it.
   `a claim lives only while its evidence does` is directly testable: the file
   changes, the claim expires. A claim derived from commit `abc123` has evidence
   that is immutable and therefore can never expire — which is the opposite of
   what CRED is for. **Git history is the wrong substrate for an
   evidence-governed model.** That is an argument from CRED's own design, and it
   is stronger than the adoption argument.

Git history stays in the roadmap. It is a v1 item, not a first-slice item.

### What the first slice should be

1. **`recall`, and nothing else.** One tool. Zero required configuration — no
   API key, local embeddings, embedded store. Against a category median of nine
   tools and two-to-five required config decisions, this is the differentiated
   position, and it is only available before `remember` exists.
2. **Seeding from `AGENTS.md`, `CLAUDE.md`, `README.md` and `docs/`**, with
   content hashing for idempotent re-import. Anchor each claim to a file and byte
   range. Do not walk git history yet.
3. **A CLI that renders the score.** What was seeded, what matched a query, what
   each component of the hybrid score contributed, and what was dropped against
   the token ceiling. This is `mem0#6448`'s and `supermemory#792`'s missing
   instrument.
4. **A measured p95 for `recall`, published.** Nobody in this category publishes
   one except Zep, whose sub-second numbers are network-bound cloud calls under
   concurrency 20 (§3). A local engine that publishes an honest p95 on a stated
   machine is making a claim none of the competitors have made. If CRED cannot
   beat 576 ms p95 locally, that needs to surface in week two, not month six.

### What to explicitly not build first

**Not the curation worker.** Deduplication failure is the most universal
complaint in §7, so this feels urgent — but there is nothing to deduplicate
until there is a write path, and seeded documentation claims are hash-unique by
construction.

**Not the ACL model.** §8 shows Letta's server-side actor is the right design
and D-014 already committed to it. It is also unexercisable with one user and
one read tool.

**Not `remember`.** Not because writing is dangerous — that argument is examined
below — but because a write path is the expensive, LLM-bound, quality-fraught
half of every system in this scan, and shipping it before the read path is
proven means debugging both at once. Every silent-data-loss issue in §7 is a
write bug discovered through a read.

### The cost of this recommendation

Stating it plainly, per rule 4 of `docs.md`.

A read-only, documentation-seeded first slice is **very close to Context7 with a
local index**. It retrieves project documentation and cites where it came from.
That is genuinely useful and genuinely unproven as *memory* — it does not
demonstrate the thesis at all. Section 11's ablation exists precisely to test
whether experiential memory contributes independently of documentation
retrieval, and this slice ships the arm of that ablation that CRED is betting
**against**.

If the ablation then shows documentation retrieval alone accounts for the
benefit, the first slice will have been the product, and PRD §11's kill
criterion fires. That is the risk. It is worth taking, because the alternative —
building the write path first and measuring after — is the exact failure D-015's
"v0 ships no product code" already rejects for the same reason.

---

## 10. What contradicts CRED's current decisions

### D-009 is half falsified

D-009: "CRED's first run reads and never writes. Contribution — the act of
storing a claim — is a deliberate, separate step the user takes after they have
already gotten value from recall." Ruled out: "automatic background contribution
before explicit opt-in, however useful."

**The ruled-out behaviour is what the category leader ships, on by default.**
Mem0's plugin writes at three trigger points — every third user prompt via
`auto_capture.py`, session summaries on `Stop`, and file/bash context on tool
use — all defaulting to on, gated only by `MEM0_AUTO_SAVE` and `MEM0_PREFETCH`
opt-*outs* (`on_user_prompt.sh`, `hooks.json`). It writes on message three of
the first session, before any user has evaluated anything.

Two claims in D-009 need separating, because they have opposite verdicts:

| D-009 claim | Verdict | Evidence |
|---|---|---|
| *First run should read before it writes* | **Supported** | Read paths are LLM-free and config-free across the category (§3, §6); every silent-data-loss issue in §7 is a write bug found through a read |
| *Automatic contribution must be ruled out, however useful* | **FALSIFIED as a category norm** | Mem0's shipped plugin does exactly this, by default, at three trigger points |

D-009 derived a *product* rule from Context7, a read-only documentation server
with no write path at all. Context7 is evidence about install friction. It is
not evidence about whether users tolerate automatic writes, because Context7
never had the option. Mem0 is the system that actually ran that experiment.

The correct reading: **read-first is a sequencing decision and it is right.
Never-automatic is a permanent prohibition and it is not supported.** D-009
fused them. The first slice needs only the first. The prohibition should be
revisited before the write path ships — as a new decision entry, which this
document does not write.

Note also that D-009's stated consequence — "Cold-start seeding (repository
history and documentation) must carry the entire first-run value" — is the right
conclusion, and §2 shows a competitor already validated the documentation half
of it.

### D-009's tool-count caution should be relaxed

D-009 held that "the best available evidence points toward fewer" than four
tools, from Context7's two. §5 puts CRED's four against a category median of
nine: Zep 13, Graphiti 13, MCP memory 9, Supermemory 7, Mem0 5. Graphiti's
primary write tool alone takes thirteen parameters. **Four self-describing tools
is at the low end of the category, not the high end.** The burden D-009 placed
on each tool is met by the comparison. What remains true is that the *first
slice* should ship one — a sequencing point, not a ceiling.

### D-010's cross-encoder cut is confirmed, from an unexpected direction

D-010 cut the cross-encoder for MaxSim on latency grounds. Two independent
confirmations here. Zep's LoCoMo configs all use `cross_encoder` for both edge
and node reranking, and its p95 is 576 ms over a network — so the reranker is
inside a budget CRED can already meet. And `graphiti#1642` reports
"cross-encoder reranking drops most candidates in edge_search," while
`mem0#6448` reports `search(rerank=True)` reranking only the truncated top-k so
it "can't improve recall." **The cross-encoder is producing correctness
complaints in two systems, not just latency.** D-010 is stronger than when it
was written.

### The single-command deploy goal needs one qualification

PRD acceptance criterion 11: "`docker compose up` reaches a working instance
with no additional steps." §6 shows nothing in the category achieves this except
the two systems that never call an LLM. This is achievable for CRED's **read**
path and only the read path. The write path will need a model, hence a key or a
local model, hence a decision. **The acceptance criterion should be scoped to
the read path, or it will be quietly failed at the moment `remember` ships.**

Related: Mem0 moved the opposite way — self-hosted auth now defaults to **on**,
and `JWT_SECRET` is mandatory (`docs/open-source/setup.mdx:34-42`), with the
upgrade note warning that 1.x deployments "will return `401` on every protected
endpoint." Frictionless install and a defensible team story are in tension, and
the category leader chose the team story. D-014 makes the same choice.
Consistent, but it is a cost, and it should be named as one.

### D-014 gains confirming runtime evidence

§8: Letta resolves a server-side `actor` on every v1 route
(`sources.py:120`); Mem0 threads a caller-supplied filter dict
(`main.py:778-783`, `:880`). Recall-time ACL enforcement is structurally
impossible in Mem0's model because there is no principal to enforce against.
D-014's placement of org, member and role in the engine is correct on runtime
grounds, independent of the pricing argument that motivated it.

---

## 11. Unverified items

Carried forward so the next person does not re-derive them.

1. **Per-tool call frequency, all systems.** No project publishes it; no issue
   reports it. §5's ranking is inferred from tool descriptions and recommended
   system prompts, which is evidence of what projects *want* called, not of what
   is called. Would be settled by telemetry none of them publish.
2. **Letta Code's cold-start behaviour.** `letta-ai/letta-code` was not cloned.
   Its README documents install and `/connect` but is silent on repository
   seeding. Would be settled by cloning it and grepping for `AGENTS.md`.
3. **Supermemory's per-add LLM call count and DB round trips.** The engine is a
   prebuilt binary; the queued-vs-sync split in §3 is from docs, not code. Not
   determinable without reverse-engineering the binary, which one issue reporter
   in `supermemory#1104` has already partly done.
4. **Supermemory's "~50ms user profiles" and "95% Recall@15" claims**
   (`README.md:36-37`). No timing code or benchmark artifacts in the repo
   substantiate them — unlike Zep, which ships raw experiment JSON. Marketing
   claims, labelled as such.
5. **Graphiti's "typically sub-second latency"** (`README.md:153`). A comparison
   table against GraphRAG with no methodology and no backing artifact.
6. **Mem0's "Sub-50ms retrieval"** (`docs/vibecoding.mdx:92`). Same.
7. **Whether Zep Community Edition has a sunset date.** The blog post confirms
   "we've decided to stop maintaining and releasing Zep Community Edition" but
   gives no date. Shallow clones prevented dating the CE→`legacy/` move.
8. **Supermemory's browser-history importer.** Rumoured, not found. Searches
   surfaced only router and `window.history` usage. Twitter, ChatGPT, Claude and
   Grok importers are VERIFIED; browser history is not.
9. **No latency number in this document was measured by me.** Zep's are
   re-extracted from committed artifacts; Graphiti's and Mem0's are user-reported
   in cited issues; call counts are derived by reading source. CRED's own p95 is
   unmeasured and should be measured in the first slice.

### FALSIFIED, kept on the record

- **`mem0ai/mem0#5352`**, cited inside `mem0#5850`'s body as a 41-comment
  community thread on memory hygiene. `gh issue view -R mem0ai/mem0 5352` returns
  "Could not resolve to an issue or pull request with the number of 5352." The
  citation appears in a real issue body but does not resolve. Do not propagate it.
- **"Nobody does cold-start seeding."** This was the anticipated headline finding
  and it is false. Mem0 ships it (`auto_import.py`), Supermemory ships importers,
  Letta ships document upload. What is true is narrower: **no system seeds from
  repository history**, and in both systems with real seeding, it is gated behind
  the hosted tier.
- **"Supermemory is fully open source and self-hostable."** Asserted in
  `apps/docs/self-hosting/overview.mdx`. The repo contains **no backend source** —
  `find` for `go.mod`/`Cargo.toml`/`*.go`/`*.rs` returns zero results, and `apps/`
  holds only browser-extension, docs, mcp, memory-graph-playground,
  raycast-extension and web. The install script (fetched) downloads prebuilt
  binaries from `releases/download/server-v{version}/`; `gh release view
  server-v0.0.6` confirms **binaries only, no source tarball**. The self-hosted
  server is real and locally capable, but it is **proprietary freeware, not open
  source**, and it is feature-reduced — no connectors, no MCP.
