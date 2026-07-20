# Letta (formerly MemGPT) — Evidence Scan

**Scanned:** `/Users/canh/Solo/OSS/letta` @ `b76da90` (shallow clone, depth 1), version `0.16.8`, 2026-07-20.
**Scope:** 878 Python files, ~34 MB, 166 test files / ~98k test LOC, 167 Alembic migrations.

---

## Verdict

1. **This repo is officially dead.** `AGENTS.md:3` — "This repository is deprecated… legacy Letta server… in maintenance mode and is no longer where active development happens." Development moved to `letta-ai/letta-code` (TypeScript) and a closed App Server. Treat everything below as **a frozen reference design, not a live dependency**. Do not build on it; mine it.
2. **The context compiler is the crown jewel and is genuinely stealable.** A deterministic, pure-string `Memory.compile()` (`letta/schemas/memory.py:688`) renders blocks into XML with **self-describing budget metadata** (`chars_current` / `chars_limit` / `read_only` emitted *into the prompt*, `memory.py:161-166`). The compiled prompt is then **persisted as `message_ids[0]`** and only recompiled when the memory string actually changes (`agent_manager.py:1562`). That's a cache-friendly, diffable, auditable context plane.
3. **Memory-block sharing is fundamentally agent-centric and unsafe for organizational shared memory.** `blocks_agents` has **no permission column** (`letta/orm/blocks_agents.py:32-34`); `read_only` is a property of the *block*, not the *attachment* (`letta/orm/block.py:49`) — you cannot make one block writable by agent A and read-only for agent B. Concurrent writes are **last-writer-wins, self-documented as such**: `# update the blocks (LRW) in the DB` (`agent_manager.py:1769`).
4. **Letta built the provenance machinery CRED needs, then never wired it up.** `BlockHistory` (append-only snapshots with `actor_type`/`actor_id`, `letta/orm/block_history.py`) plus optimistic locking (`orm/block.py:56-61`) plus checkpoint/undo/redo (`block_manager.py:842-1004`) — and `checkpoint_block_async` has **zero production callers**, only tests. This is CRED's opening: the design is validated, the execution gap is the product.
5. **The `.af` agent-file format is the right idea with three disqualifying gaps:** no schema version field, archival memory silently dropped, and imports rebind tools/sources **by name**. Steal the shape, fix the gaps.

---

## 1. Context Hierarchy — what is actually in the window

### The three tiers

| Tier | Storage | In context? | Access |
|---|---|---|---|
| **Core memory** (blocks) | `block` table, always rendered | **Always, in full** | Self-edit tools |
| **Recall memory** | `message` table | Only a sliding window | `conversation_search` |
| **Archival memory** | `passage` table + pgvector | **Never** — only a *count* | `archival_memory_search` |

Out-of-context tiers appear only as a metadata teaser (`letta/prompts/prompt_generator.py:69-88`):

```
<memory_metadata>
- AGENT_ID: agent-123
- System prompt last recompiled: 2024-01-15 09:00 AM PST
- 42 previous messages between you and the user are stored in recall memory
- 156 total memories you created are stored in archival memory (use tools to access them)
- Available archival memory tags: project_x, meeting_notes, research
</memory_metadata>
```

Notable: **archival tags are surfaced but archival content is not**. Cheap, high-leverage affordance — the model learns what vocabulary exists before deciding to search. Steal this.

### The assembly path

`PromptGenerator.compile_system_message_async` (`prompt_generator.py:181`) →
`Memory.compile()` (`memory.py:688`) →
`get_system_message_from_compiled_memory` (`prompt_generator.py:107`).

`compile()` (`memory.py:688-732`) emits, in fixed order:
1. `<memory_blocks>` — one of three renderers chosen by agent type + provider (`memory.py:705-712`): standard, line-numbered (Anthropic + specific agent types only, `:697-702`), or git-backed
2. `<tool_usage_rules>` (`:718-724`)
3. `<directories>` — attached file sources (`:726-730`)

Then `get_system_message_from_compiled_memory` substitutes the whole thing into the `{CORE_MEMORY}` placeholder (`prompt_generator.py:152-169`, keyword defined `constants.py:64`), appending it if the template omitted it (`:158-162`).

**Per-block rendering** (`memory.py:149-173`) — note lines 161-166:

```python
s.write("<metadata>")
if getattr(block, "read_only", False):
    s.write("\n- read_only=true")
s.write(f"\n- chars_current={chars_current}")
s.write(f"\n- chars_limit={limit}\n")
s.write("</metadata>\n")
```

The agent is *told its own budget and permissions in-band*. Governance-as-prompt-text.

### Recompilation is change-gated

`rebuild_system_prompt_async` (`agent_manager.py:1523`) compiles the memory string and **compares it as a substring of the current system message** (`:1562`):

```python
if curr_memory_str in curr_system_message_openai["content"] and not force:
```

Docstring (`:1535`): *"Updates to the memory header should not trigger a rebuild, since that will simply flood recall storage with excess messages."* Good instinct — but the mechanism is substring matching, and the code even admits the flaw at `:1563`: *"NOTE: could this cause issues if a block is removed? (substring match would still work)"*. Yes, it does.

### Overflow: reactive, not proactive

- Trigger threshold: `SUMMARIZATION_TRIGGER_MULTIPLIER = 0.9` (`constants.py:83`) — 90% of the context window.
- **`thresholds.py:27-41` has a stale docstring**: it claims GPT-5 compacts at 90% and "All other models trigger at 100%", and documents a `force_proactive` parameter — but the body is a single line that ignores `force_proactive` entirely and returns `context_window * 0.9` for everything. Doc/code drift in the most safety-critical function.
- The real path is a **retry-on-exception loop**: the LLM call fires, the provider raises `ContextWindowExceededError`, and only then does compaction run (`letta/agents/letta_agent_v3.py:1218-1262`). Up to `max_summarizer_retries` attempts. So a real overflow costs a wasted API round-trip.

Two summarization modes (`services/summarizer/summarizer.py:104-119`):
- `STATIC_MESSAGE_BUFFER` (`:244`) — trim to `message_buffer_min`, summarize evicted messages in a **background fire-and-forget task** (`:124-134`).
- `PARTIAL_EVICT_MESSAGE_BUFFER` (`:136`) — the MemGPT-classic path. Evict 30% (`:49`), walk forward to the next `assistant` message to keep the message sequence valid (`:170-175`) — a nice detail, since evicting mid-tool-call corrupts the transcript — then splice a recursive summary in at index 1 as a `user` message (`:242`).

**Critical: core memory blocks are NEVER evicted.** Only messages are. Blocks grow monotonically until they blow the window, and nothing stops them (see §3).

### Token accounting

`ContextWindowOverview` (`memory.py:23-65`) is a genuinely good schema — per-section token counts *and* content for system / core_memory / memory_filesystem / tool_usage_rules / directories / summary / functions / messages / external summary.

But `ContextWindowCalculator` computes it by **regex-scraping the compiled string back apart** (`context_window_calculator.py:22-48`, `:51-83`, `:86+`) — `text.find("<memory_blocks>")` etc., with a documented "only the first occurrence is extracted" caveat (`:35`) and a fallback heuristic for custom prompts (`:74-83`). The compiler throws away structure, then the observability layer guesses it back. And the endpoint is already `deprecated=True` (`routers/v1/agents.py:588`).

---

## 2. Memory Blocks

**Schema** (`letta/schemas/block.py:13-64`): `value`, `limit` (default `CORE_MEMORY_BLOCK_CHAR_LIMIT = 100000`, `constants.py:435`), `label`, `description`, `read_only`, `metadata`, `hidden`, `tags`, plus a template family (`is_template`, `template_id`, `base_template_id`, `deployment_id`, `entity_id`, `preserve_on_migration`).

**Warning signal:** in `BlockResponse` (`block.py:88-104`), `read_only`, `hidden`, and the *entire* template family are marked `deprecated=True`. Letta is walking back its own governance primitives.

**ORM** (`letta/orm/block.py`) — the good parts:
- Optimistic locking: `version` + `__mapper_args__ = {"version_id_col": version}` (`:56-61`)
- History pointer: `current_history_entry_id` → `block_history` (`:53-55`)
- Three sharing relationships: `agents` via `blocks_agents` (`:65`), `identities` via `identities_blocks` (`:73`), `groups` via `groups_blocks` (`:80`)

**Labels are path-like.** `system/human`, `skills/foo/SKILL` — and `compile()` builds a virtual filesystem tree from them (`memory.py:243-282`, `:351-481`), rendering `<memory_filesystem>` with `tree`-style box-drawing. There's explicit handling for the case where a label is both a file and a directory (`memory.py:363-399`). This is a hierarchical namespace bolted onto a flat string column — clever, and directly relevant to CRED's multi-repo namespacing.

### Sharing — the load-bearing weakness

`letta/orm/blocks_agents.py:32-34`, the *entire* payload:

```python
agent_id: Mapped[str] = mapped_column(String, ForeignKey("agents.id", ondelete="CASCADE"), primary_key=True)
block_id: Mapped[str] = mapped_column(String, primary_key=True)
block_label: Mapped[str] = mapped_column(String, primary_key=True)
```

No permission column. No `created_at`. No actor. No role. Attachment is a bare edge.

Contrast `archives_agents` (`letta/orm/archives_agents.py:27-28`), which *does* carry `created_at` and `is_owner` — but is then crippled by `UniqueConstraint("agent_id")`, one archive per agent, with a TODO to remove it (`:19-21`).

There is **no ACL layer at all**. Access control is a single org-scoping predicate (`orm/sqlalchemy_base.py:880-901`), and the `access` parameter is literally discarded: `del access  # entrypoint for row-level permissions`. `Organization` has no roles (`orm/organization.py`), `User` has no role field (`schemas/user.py`). Same org ⇒ full permissions on everything.

### Concurrency — last-writer-wins, and they say so

The optimistic lock is real but **structurally defeated**:

1. Memory tools mutate `agent_state.memory` — the agent's **in-context snapshot**, loaded at turn start (`core_tool_executor.py:328-344`).
2. `update_memory_if_changed_async` (`agent_manager.py:1747-1787`) diffs against that stale snapshot (`:1774`) and writes unconditionally. Line 1769: `# update the blocks (LRW) in the DB`.
3. `update_block_async` (`block_manager.py:211-241`) opens a **fresh session and re-reads the row**, so `version_id_col` compares the version it just loaded against itself — it can never be stale. The lock guards microseconds inside one function, not the read-modify-write window spanning the agent's entire turn.
4. No `SELECT … FOR UPDATE`, no advisory locks, no CAS on a caller-supplied version anywhere on the block write path. `bulk_update_block_values_async` (`block_manager.py:795-833`) does plain assignment in a loop with no version check at all.

This is not theoretical. Letta's own **sleeptime** architecture runs background agents editing the *same blocks* as the foreground agent, concurrently, by design (`sleeptime_multi_agent_v4.py:170-196`). Lost updates are silent — no error, no merge, no retry.

### Provenance: built, tested, never called

`BlockHistory` (`orm/block_history.py`) is exactly the right table: full value snapshot, monotonic `sequence_number` per block (`:46-48`), and **`actor_type` / `actor_id` distinguishing agent edits from human edits** (`:36-37`, set at `block_manager.py:898-899` via `ActorType.LETTA_AGENT if agent_id else ActorType.LETTA_USER`). `checkpoint_block_async` (`:842-911`) even truncates future checkpoints to keep a linear undo stack (`:874-883`).

A repo-wide grep for `checkpoint_block_async|undo_checkpoint_block|redo_checkpoint_block` returns hits in **`tests/` only** — zero production callers, zero REST endpoints. Well-tested dead code.

---

## 3. Self-Editing — guardrails are thin and duplicated

Tools: `core_memory_append`, `core_memory_replace`, `memory_replace`, `memory_insert`, `memory_rethink`, `rethink_memory` (`letta/functions/function_sets/base.py:246-500`), executed by `LettaCoreToolExecutor` (`services/tool_executor/core_tool_executor.py`).

**Guardrails that exist:**
- `read_only` check, raising `READ_ONLY_BLOCK_EDIT_ERROR` — at 9 sites in `core_tool_executor.py` (`:320`, `:336`, `:354`, `:530`, `:665`, `:691`, `:744`, `:899`, `:959`)
- Uniqueness enforcement on replace: `occurences > 1` → refuse and report the ambiguous line numbers (`function_sets/base.py:368-373`). Good — forces precise edits over blind blanket replaces.
- Line-number-prefix rejection, so the model can't paste back the view-only line numbers it was shown (`base.py:345-356`, `core_tool_executor.py:357-374`)

**Guardrails that do NOT exist:**

1. **`limit` is advisory only.** Nothing enforces it on write. Grepping the executor for limit checks returns nothing; the only `limit`-related enforcement in the codebase is on *function output* truncation (`utils.py:926,937`) and archival token count (`server/server.py:875`). The block char limit is rendered into the prompt (`memory.py:165`) and that is the *entire* enforcement mechanism — pure trust in the model.

2. **Dual implementations that disagree.** `function_sets/base.py:246-260` (`core_memory_append`) has **no `read_only` check**; `core_tool_executor.py:319-326` does. The former is the source-code definition shipped to sandboxes; the latter is the server-side fast path. Two copies of the same tool with different security properties is a latent bypass.

3. **Nothing prevents corrupting a shared block.** `read_only` is the only control, and it's global per-block, not per-agent. An agent with write access to a shared block can `core_memory_replace(old, "")` and silently delete organizational knowledge, with no checkpoint recorded (§2) and no attribution.

---

## 4. Agent State & Persistence — the `.af` format

Directly relevant to CRED's handoff package. **Two coexisting stacks:**

- **v1 (dead):** marshmallow-based, `letta/serialize_schemas/marshmallow_*.py`. Export path removed and import rejected with HTTP 400 (`routers/v1/agents.py:322-323`, `:578-583`). 9 of 12 checked-in `.af` fixtures — including the shipped demos `deep-thought.af`, `customer_service.af`, `deep_research_agent.af` — are v1-shaped and **no longer importable**.
- **v2 (current):** pure pydantic, `letta/schemas/agent_file.py` + `services/agent_serialization_manager.py`. Note it does *not* live in `serialize_schemas/`.

### v2 schema

`AgentFileSchema` (`agent_file.py:431-445`), top-level keys:

```
agents, groups, blocks, files, sources, tools, mcp_servers, skills, metadata, created_at
```

| Content | Included | Evidence |
|---|---|---|
| Message history | Yes — nested per-agent, **capped at 50** unless `message_ids` set | `agent_file.py:134`, `:199-201` |
| Tools **incl. `source_code`** | Yes — full passthrough with `json_schema`, `pip_requirements`, `npm_requirements` | `:358-367` |
| Blocks | Yes — top-level, deduped | `:150-161` |
| Files/sources **incl. content** | Yes | `:328`, `agent_serialization_manager.py:187` |
| Tool rules | Yes | `:157` |
| LLM + embedding config | Yes | `:161-162` |
| Groups (transitively pulls siblings) | Yes | `agent_serialization_manager.py:425-440` |
| MCP servers | Yes, auth-scrubbed | `agent_file.py:413-428` |
| **Archival passages** | **NO — silently dropped** | no `passages` key; `passage_manager` never referenced |

### ID remapping — the best idea in the format

Export rewrites every DB ID to a **sequential, human-readable file-local ID**: `agent-0`, `block-1`, `message-7` (`_generate_file_id`, `agent_serialization_manager.py:118-123`, counters `:100-110`, map `:97`). All cross-references — `tool_ids`, `block_ids`, `source_ids`, `group_ids`, `in_context_message_ids`, `manager_config.manager_agent_id` — are rewritten (`:259-277`, `:363-369`).

Crucially, `allow_new=False` (`:125-135`) makes a **dangling reference raise `AgentExportIdMappingError`** rather than silently minting an ID. Export is fail-closed on referential integrity. Import validates prefix format, integer suffix, global duplicates, and referential integrity before touching the DB (`_validate_schema`, `:993-1020`), and creates in dependency order (`:546-837`).

This yields **deterministic, diffable, git-friendly** agent files. Steal it wholesale.

### Secrets — handled reasonably

Three layers: env var values blanked but keys preserved (`agent_serialization_manager.py:240-244`); MCP `token` / `custom_headers` omitted and `stdio_config.env` stripped (`agent_file.py:413-428`); re-injection at import via a separate `secrets` form field (`:733-742`, router `:479`).

### Three disqualifying gaps

1. **No schema version.** v2 has none. The only artifact is `metadata={"revision_id": <alembic hash>}` (`agent_serialization_manager.py:488`) — written on export, **never read on import**. Format detection is duck-typing with a self-aware comment: `# This is kind of hacky` (`routers/v1/agents.py:562-563`). v1 *had* a real `version` field (`marshmallow_agent.py:182`) whose mismatch handler was a bare `print()` that deleted the field and proceeded (`:223-230`) — so v2 dropped a field that never worked. Either way: no migration story.

2. **Archival memory is lost.** For CRED this is fatal as-is — the accumulated searchable knowledge is precisely the handoff payload. Source-file passages are re-derived by re-chunking and re-embedding in **fire-and-forget background tasks** (`:691-697`); failures only flip the file to `ERROR` while the import still reports success (`:1057-1071`).

3. **Import rebinds by name, not ID.** Tools and sources are matched back by name after bulk upsert (`:573-577`, `:628-637`), so a name collision silently rebinds the imported agent to a **pre-existing** tool — a supply-chain hazard in any shared org workspace. Sources at least get a random 8-hex suffix on collision (`:617-619`); tools do not.

---

## 5. Multi-Agent — mostly stale, one live path

| Component | Status |
|---|---|
| `supervisor_multi_agent.py` | **Dead** — entire `step()` commented out, `:30-111`; only `__init__` remains |
| `round_robin_multi_agent.py` | Real but legacy sync; mutates the persona block in-memory to inject group instructions (`:155`) |
| `dynamic_multi_agent.py` | Real, sync; manager picks speaker via an attached `choose_next_participant` tool (`:235-244`) |
| `sleeptime_multi_agent_v4.py` | **The live path** — `agents/agent_loop.py:35` |

`load_multi_agent()` (`groups/helpers.py:15-86`) dispatches round-robin/dynamic/supervisor — but is **not called from anywhere outside `letta/groups/`**. The live dispatcher `AgentLoop.load()` only knows sleeptime v3/v4. Group chat is orphaned; **sleeptime is the only real multi-agent feature.**

### Sleeptime (real, and the most interesting idea)

`SleeptimeMultiAgentV4` wraps `step()`/`stream()`: run foreground normally, then fire `run_sleeptime_agents()` (`:78-79`, `:129`), which (a) fires only every N turns via a group counter (`:138-146`), (b) advances a `last_processed_message_id` watermark (`:152-154`), (c) spawns a detached `safe_create_task` per agent with a `Run` row for tracking (`:170-196`), (d) feeds the transcript to each sleeptime agent telling it *"You are NOT the primary agent… Your primary role is memory management"* (`:229-242`).

**Asynchronous, watermarked, background memory consolidation, decoupled from the request path.** Conceptually strong. Undermined by the LWW block writes (§2) — it is precisely the configuration that loses updates.

### Agent-to-agent messaging is HTTP loopback

`functions/function_sets/multi_agent.py` — these are **sandboxed tool source code**, not server-side executors. They construct a `letta_client.Letta` SDK client from `LETTA_API_KEY`/`LETTA_SERVER_URL` env (`:6-24`) and go back out through the **public REST API**. Not an in-process call.

- `send_message_to_agent_and_wait_for_reply` (`:60-97`) — synchronous, blocking, point-to-point
- `send_message_to_agents_matching_tags` (`:100-160`) — "broadcast" is a **sequential blocking for-loop** (`:143-158`), not concurrent. `match_all`/`match_some` are two separate `agents.list()` calls intersected client-side (`:117-137`), and the empty-`match_all` branch falls back to `list(limit=100)` — an unqualified broadcast is **silently capped at 100 agents**. Per-agent errors are swallowed into `"<error: ...>"` strings (`:157-158`). Responses are parsed with `ast.literal_eval` on a `str(dict)` payload (`:149`).
- `send_message_to_agent_async` (`:163-191`) — **hard-disabled in prod** (`:172-173`, `constants.py:155`)

**Real:** shared blocks, sleeptime consolidation, blocking point-to-point messaging.
**Aspirational:** supervisor groups, scalable broadcast, async messaging, any notion of governed org-wide knowledge.

---

## 6. Storage & Deployment

**ORM:** SQLAlchemy 2.x async declarative. `Base` (`orm/base.py:8`), `CommonSqlalchemyMetaMixins` supplying `created_at`/`updated_at`/`is_deleted` (`:13-18`) and `_created_by_id`/`_last_updated_by_id` (`:37-50`). `SqlalchemyBase` (`orm/sqlalchemy_base.py:121`) is a 1003-line active-record layer.

**Soft delete is default** — `delete_async` just sets `is_deleted = True` (`:681`). But reads only filter deleted rows if the caller opts in: `check_is_deleted: bool = False` (`:249`, `:394-395`, `:531-533`). **Soft-deleted rows leak into listings by default.** Likely a bug.

**Multi-tenancy:** `apply_access_predicate` (`:880-901`) scopes by `organization_id` or `user_id`. Defensive raise if a model has `organization_id` and `actor is None` (`:256-257`). No finer granularity — see §2 on the discarded `access` param.

**Postgres vs SQLite:** `DatabaseChoice` (`settings.py:273-275`), selected by whether `LETTA_PG_URI` is set (`:492-493`). ~40 branch sites: pgvector `Vector(MAX_EMBEDDING_DIM)` vs a `BINARY` TypeDecorator (`orm/passage.py:35-40`); native `cosine_distance` vs a numpy Python UDF with no index (`sqlite_functions.py:140-189`); `ILIKE` vs `LOWER() LIKE`; ~8 datetime-pagination coercions.

**SQLite is vestigial and broken in this tree.** `letta/server/db.py` has no SQLite path — line 21 always resolves the PG URI, line 58 is the *only* `create_async_engine` call repo-wide, and `convert_to_async_uri` unconditionally forces `asyncpg` (`database_utils.py:97-118`). An unconfigured install resolves `database_engine == SQLITE` while still dialing `postgresql+asyncpg://localhost:5432`. **Postgres + pgvector is mandatory.** Many migrations also early-return on SQLite (`bff040379479_add_block_history_tables.py:25,64` and ~15 more).

**Migrations:** 167 files, `Create Date` spanning 2024-10 → 2026-03. Cadence ~10-15/month peaking 2025-01 and 2025-07, then falling off after 2025-10 (5, 5, 9, 2, 4) — consistent with the deprecation.

**Self-hosting is heavy.** **69 core dependencies** plus 13 extras (`pyproject.toml`, `requires-python = "<3.14,>=3.11"`). Core alone pulls llama-index, matplotlib, grpcio+tools, temporalio, ddtrace, clickhouse-connect, sentry-sdk, markitdown[docx,pdf,pptx], mcp[cli], fastmcp. The `Dockerfile` bases on `pgvector/pgvector:0.8.1-pg15` for builder *and* runtime, then adds Node.js 22, redis-server, and the OTel Collector binary, `EXPOSE 8283 5432 6379 4317 4318` — one fat image running Postgres + Redis + OTel + app. `startup.sh` boots internal Redis and Postgres unless externally configured, then `alembic upgrade head` with hard-exit on failure (`:56-61`). Minimum realistic: 2 services. `.env.example` is 21 lines, entirely commented, documenting none of the PG/Redis/OTel/sandbox knobs.

**Tool sandbox — a real risk.** Three backends (`schemas/enums.py:262-265`): E2B, Modal, Local. `ToolSettings.sandbox_type` (`settings.py:60-70`) returns `E2B` **iff `e2b_api_key` is set, otherwise `LOCAL`** — and `AsyncToolSandboxLocal` runs tool code in a **subprocess on the server host** (`local_sandbox.py:139`), same filesystem. A default self-host executes LLM-authored code on the server box with no isolation, by silent default.

---

## 7. Interfaces

**REST:** 33 v1 routers (`routers/v1/__init__.py:34-68`) at `/v1` (`app.py:853`), aliased to `/latest` hidden from schema (`:857`), admin at `/v1/admin` (`:860-861`). Route counts: agents 54, tools 23, sources 14, folders 14, conversations 13, sandbox_configs 12, runs 11, groups 11, mcp_servers 10, identities 10, blocks 9, archives 9. Plus OpenAI-compat at `/openai`.

**The shipped `fern/openapi.json` is the Cloud surface, not this server** — 239 paths / 302 operations, including groups with no OSS router: `templates` (16), `feeds` (12), `pipelines` (7), `environments` (5), `projects`, `client-side-access-tokens`, `sandboxes`. That gap **is** the OSS-vs-cloud boundary.

**MCP: client only, never a server.** Every `FastMCP` reference is `fastmcp.client`. No `from mcp.server` import, no server instantiation, no stdio/SSE entrypoint. **Letta exposes zero MCP tools.** Three client transports in `services/mcp/`: `AsyncStdioMCPClient` (`stdio_client.py:14`), `AsyncSSEMCPClient` (`sse_client.py:19`), `AsyncStreamableHTTPMCPClient` (`streamable_http_client.py:16`), with FastMCP replacements now live for SSE/HTTP (`fastmcp_client.py:28`, `:185`) while stdio still uses the raw SDK (`mcp_manager.py:798-848`). Default type `STREAMABLE_HTTP` (`schemas/mcp.py:31`). Org-scoped persistence with encrypted headers (`orm/mcp_server.py:20`, `schemas/mcp.py:110-118`) and OAuth (`services/mcp/oauth_utils.py`). Rot signal: **three overlapping `MCPServerType` enums** (`schemas/mcp.py`, `functions/mcp_client/types.py:36`, `services/mcp/types.py:12`).

**SDKs:** `fern/` holds only `openapi.json` + overrides — no `fern.config.json`, no `generators.yml`; generation lives elsewhere. `letta/client/__init__.py` is **0 bytes**; the real SDK is the external `letta-client>=1.7.12`, itself now labelled previous-generation (`README.md:74`).

---

## 8. OSS Project Mechanics

**License: clean Apache 2.0**, full text + appendix, `Copyright 2023, Letta authors`. No Commons Clause, no BSL, no non-compete rider. *Caveat:* the clone is shallow (`.git/shallow`, 1 commit), so license *history* cannot be verified from this tree — verify any prior relicensing against GitHub directly before relying on it.

**OSS-vs-cloud boundary:** drawn at the OpenAPI spec (§7). Templates, deployments, projects, feeds, pipelines, environments are cloud-only — and the `is_template`/`template_id`/`deployment_id`/`entity_id` fields littering the OSS `Block` schema (`block.py:24-30`) are vestigial hooks for that closed layer, now marked deprecated in responses (`:93-102`). `InternalTemplateBlockCreate` is explicitly labelled *"Used for Letta Cloud"* (`:203`).

**Tests: strong.** 166 files / ~98k LOC. `tests/managers/test_block_manager.py` alone covers checkpoint/undo/redo exhaustively including concurrency cases (`:922-933`) — for a feature with no production callers. 30 CI workflows including `alembic-validation`, `migration-test`, `core-unit-sqlite-test`, `model-sweep`, and provider matrices (ollama/vllm/lmstudio).

**Community:** Discord + forum + Fern-published docs; `close_stale_issues.yml`, `issue-guard.yml`, `manually_clear_old_issues.yml` — issue-triage automation typical of a high-volume repo. `AI_POLICY.md` (adapted from Ghostty) mandates AI-usage disclosure and human comprehension, with unreviewed AI output closed on sight. Worth copying verbatim for CRED.

**Health assessment: the code is well-engineered and the project is over.** Deprecation notice, `letta-code-sync.yml` in CI, migration cadence collapsing after 2025-10, `letta/client/__init__.py` emptied, best observability endpoint deprecated, governance fields deprecated. This is a carefully maintained tombstone.

---

## Recommendations for CRED

### Top 3 to STEAL

**1. The compile → persist → change-gate context pipeline.**
`Memory.compile()` (`memory.py:688`) is a **pure, deterministic, side-effect-free string builder**. The compiled result is persisted as `message_ids[0]` and recompiled only when the memory string changes (`agent_manager.py:1562`). That gives you prompt-cache stability, a diffable audit trail, and a testable compiler.

Steal it — with two fixes Letta didn't make: emit **structured segments** (list of `(section, content, token_count, source_ids)`) and render to string as a final step, so token accounting and provenance are *products of compilation* rather than regex-recovered afterwards (`context_window_calculator.py:22-48`); and make the change-gate a **content hash per block**, not `curr_memory_str in content` substring matching, which Letta itself flags as broken on block removal (`agent_manager.py:1563`).

**2. In-band budget and permission metadata.**
`memory.py:161-166` renders `read_only=true`, `chars_current`, `chars_limit` *into the prompt*, and `prompt_generator.py:84-85` surfaces **archival tags without archival content**. The model is told its budget, its permissions, and the vocabulary of what it could retrieve. For CRED this is the governance surface: render `owner`, `last_updated_by`, `provenance`, `confidence`, `staleness` the same way. Cheap, and it makes the agent a participant in governance rather than a subject of it. Letta then undercut this by never enforcing `limit` — CRED should render *and* enforce.

**3. The `.af` sequential-ID remapping discipline.**
`_generate_file_id` (`agent_serialization_manager.py:118-123`) rewriting every DB ID to `agent-0`/`block-1`/`message-7`, with all cross-references rewritten (`:259-277`) and `allow_new=False` making dangling references **fail the export** (`:125-135`), plus pre-import referential validation (`:993-1020`). This is what makes a handoff package deterministic, diffable, and reviewable in a PR — exactly CRED's requirement. Take the discipline and add the three things Letta lacks: a **real `schema_version` with an enforced migration path**, **archival/vector content included or explicitly manifested as a dependency**, and **binding by content-hash rather than name** (`:573-577`).

### Top 3 NOT to copy

**1. Last-writer-wins on shared mutable blocks.**
`agent_manager.py:1769` (`# update the blocks (LRW) in the DB`) plus a read-modify-write window spanning the agent's whole turn (`core_tool_executor.py:328-344`) plus an optimistic lock defeated by re-reading in a fresh session (`block_manager.py:211-241`). Letta ships the exact configuration that loses updates — sleeptime agents editing foreground blocks concurrently. For CRED, where the whole premise is *multiple agents across multiple repos writing shared organizational knowledge*, this is disqualifying. Shared memory needs CAS on a caller-supplied version, or CRDT/append-with-merge, or single-writer-per-block ownership. Pick one before writing any code.

**2. Reactive, exception-driven context overflow.**
Waiting for the provider to raise `ContextWindowExceededError` and then compacting on retry (`letta_agent_v3.py:1218-1262`) burns a full round-trip on every overflow, and makes cost/latency unpredictable exactly when context is largest. Compounding it: the threshold function's docstring describes branching logic and a `force_proactive` parameter the body ignores entirely (`thresholds.py:27-41`). Budget **before** the call — compile, count, evict to fit, then send.

**3. Duplicated tool implementations with divergent guardrails.**
`function_sets/base.py:246-260` and `core_tool_executor.py:319-326` are the same tool; only the second checks `read_only`. Two copies with different security properties is a bypass waiting to be found. One implementation, one enforcement point. Related: don't put `read_only` on the *entity* — Letta's placement on `Block` (`orm/block.py:49`) rather than on `blocks_agents` is the root cause of its inability to do per-agent permissions, and it is now deprecating the flag anyway (`block.py:103`).

### Is Letta's memory-block sharing model a viable basis for organizational shared memory?

**No. It is fundamentally agent-centric, and the constraint is structural, not a missing feature.**

Three independent disqualifiers:

1. **The sharing edge carries no semantics.** `blocks_agents` is `(agent_id, block_id, block_label)` and nothing else (`orm/blocks_agents.py:32-34`) — no permission, no role, no actor, no timestamp. Permission lives on the *block* (`orm/block.py:49`), so a block is writable by everyone or no one. Organizational memory requires per-principal permissions on the *relationship* — "platform team writes, product agents read". Letta cannot express that sentence. Adding it means a new association model and rewriting every read path, at which point you are not using Letta's model.

2. **Writes are last-writer-wins with no attribution.** §2 above. The one place attribution exists — `BlockHistory.actor_type`/`actor_id` (`orm/block_history.py:36-37`), which correctly distinguishes agent from human edits — is **never invoked in production**. Provenance that isn't captured on the write path isn't provenance.

3. **There is no organizational principal.** Access control is one org-scoping predicate with the granularity hook explicitly discarded (`sqlalchemy_base.py:891`: `del access`). No roles on `Organization`, none on `User`. "Same org ⇒ all permissions" is the entire model. Blocks attach to `agents`, `identities`, and `groups` (`orm/block.py:65-86`) — but a `group` is a *conversation topology*, not an org unit.

The tell is the direction of travel: `read_only`, `hidden`, and the whole template family are marked `deprecated=True` in `BlockResponse` (`block.py:93-104`). Letta is *removing* governance primitives, because its product is a self-improving individual agent — the block is a scratchpad the agent owns, occasionally shared with its own background workers. Shared org memory needs the inverse: knowledge the **organization** owns, that agents are granted scoped, audited, revocable access to.

**The actionable read:** Letta's *primitives* — versioned blocks, append-only history with actor attribution, deterministic compilation, portable agent files — are the right vocabulary and are worth stealing at the design level. Its *sharing semantics* are the wrong shape. Letta built roughly 70% of the storage layer CRED needs (`BlockHistory`, `version_id_col`, checkpoint/undo/redo) and then left it unwired because its product didn't need it. That unwired 30% — the write path that captures provenance, the CAS that prevents lost updates, the per-attachment ACL — is precisely CRED's product. Validating that Letta built and abandoned it is a positive signal on the design, and a clear signal not to build on the repo.
