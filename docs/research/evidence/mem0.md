# Evidence: Mem0 (mem0ai/mem0)

**Scope of review:** local checkout at `/Users/canh/Solo/OSS/mem0`, package version `2.0.12` (`pyproject.toml:8`), Apache-2.0 (`pyproject.toml:14`). Read of actual source: `mem0/memory/main.py` (3777 LOC), `mem0/configs/prompts.py` (1062 LOC), `mem0/memory/storage.py`, `mem0/utils/factory.py`, `mem0/utils/scoring.py`, `mem0/memory/notices.py`, `openmemory/api/app/*`, `server/`, `tests/`.

> **Provenance warning — addendum added 2026-07-21, outside the commissioned
> review above.** Triggered by a conversational question about CRED's own
> SSO/MCP auth status; a targeted follow-up check found `openmemory/` is
> **explicitly sunset**. Its README states *"⚠️ Sunsetting Notice: OpenMemory
> is being sunset... please use the Mem0 self-hosted server instead"*
> (`openmemory/README.md:3`). `server/` is the current self-hosted product.
> This re-frames every OpenMemory finding below (§5, §7): the MCP surface and
> the ACL bugs documented in this file live in the codebase mem0 itself is
> retiring, not in the one it's steering users toward. `server/` — the
> actively maintained successor — has **no MCP implementation at all**
> (`grep -rli mcp server/` → zero hits). Only the README line and that one
> grep were checked for this addendum; the rest of `server/` and
> `openmemory/` below were verified as part of the original commissioned
> review.

---

## Verdict

- **Mem0 is a personalization memory layer for consumer chat, not an organizational knowledge plane.** The canonical memory is an untyped English sentence about a *user* (`MemoryItem` — `mem0/configs/base.py:16-26`), scoped only by three opaque string IDs (`user_id`/`agent_id`/`run_id`). There is no ontology, no confidence, no review state, no validity window, no team/org.
- **v2.0 quietly deleted the governance story.** The write path is now **ADD-only** (`mem0/memory/main.py:872-1162`); the famous LLM ADD/UPDATE/DELETE reconciler (`DEFAULT_UPDATE_MEMORY_PROMPT`, `prompts.py:176-324`) is **dead code — zero call sites**. Contradiction resolution is now MD5-hash dedup plus a prompt politely asking the LLM not to re-extract. Nothing prevents poisoning or staleness.
- **Provenance is effectively zero.** A stored memory keeps `data`, `hash`, `created_at`, `updated_at`, scope IDs, and optionally `attributed_to: "user"|"assistant"` — no source message ID, no span, no run/commit/PR reference. The LLM is *asked* for `linked_memory_ids` (`prompts.py:513, 692-701, 936`) and the OSS write path **silently discards them** (`main.py:971-995` persists only `attributed_to`).
- **The reusable engineering is real and worth stealing:** the additive extraction prompt (`prompts.py:468-946`), the UUID→integer anti-hallucination remap (`main.py:889-894`), the 8-phase batched write pipeline, the transparent additive hybrid scorer with `explain=True` (`utils/scoring.py:60-139`), and a genuinely thin 11-method vector-store interface (`vector_stores/base.py`) with 24 providers.
- **The single biggest structural gap = there is no *governed shared* memory.** Every access path is a bare ID string with no authorization, no tenancy, no cross-agent write arbitration. `get`/`update`/`delete` take a memory ID and **no scope at all** (`main.py:1164`, `1771`, `1823`); the REST server authenticates properly and then ignores the principal entirely (`server/main.py:366-374`); the OSS ACL that exists is **default-open with no write path — zero rows are ever created** (`openmemory/api/app/routers/memories.py:73-74`; `grep "AccessControl("` → only the class definition).

---

## 1. Core data model

### The wire/return shape

`mem0/configs/base.py:16-26` — the entire public memory type:

```python
class MemoryItem(BaseModel):
    id: str
    memory: str      # "The memory deduced from the text data"
    hash: Optional[str]
    metadata: Optional[Dict[str, Any]]
    score: Optional[float]
    created_at: Optional[str]
    updated_at: Optional[str]
```

That is it. No type, no source, no confidence, no owner, no state.

### The stored shape

The vector-store payload written at `mem0/memory/main.py:984-995`:

| key | source | note |
|---|---|---|
| `data` | LLM-extracted text | the memory itself |
| `text_lemmatized` | `lemmatize_for_bm25(text)` | for BM25 |
| `hash` | `hashlib.md5(text.encode()).hexdigest()` (`main.py:976`) | dedup key |
| `created_at` / `updated_at` | UTC ISO | identical on create |
| `user_id`/`agent_id`/`run_id` | caller | from `_build_filters_and_metadata` |
| `attributed_to` | LLM, optional | literally `"user"` or `"assistant"` |
| `expiration_date` | caller, optional | flat `YYYY-MM-DD` |
| *(arbitrary caller metadata)* | caller | untyped passthrough |

### Scoping

`_build_filters_and_metadata` (`main.py:287-370`) is the whole scoping model. Three optional strings; at least one required (`main.py:357-363`). `actor_id` exists but is **query-time only** — explicitly *not* written to storage metadata (`main.py:308-310`), so you cannot durably say "agent B wrote this."

There is no project, org, tenant, repo, or team dimension anywhere in `mem0/`.

### Typing / ontology

`mem0/configs/enums.py` is three lines of enum:

```python
class MemoryType(Enum):
    SEMANTIC = "semantic_memory"
    EPISODIC = "episodic_memory"
    PROCEDURAL = "procedural_memory"
```

Only `PROCEDURAL` is actually honored — passing anything else raises (`main.py:787-793`), and procedural memory is just "summarize the agent trajectory into one blob" (`PROCEDURAL_MEMORY_SYSTEM_PROMPT`, `prompts.py:326-403`; `_create_procedural_memory`, `main.py:1933`). SEMANTIC/EPISODIC are unreachable dead constants.

### Validity / review / confidence — absent

`grep -rni "confidence|review_state|approved|provenance|source_uri|valid_from|valid_until|superseded" mem0/ --include='*.py'` returns **five hits, all in docstrings/prompt prose**, none a field. The only temporal control is `expiration_date`: a single flat date, normalized at `main.py:388-400`, filtered at read time by `_payload_is_expired` (`main.py:403-412`). No `valid_from`. No bitemporality. No supersession chain in OSS — `delete_linked` (which walks the `linked_memory_ids` chain) exists **only on the hosted client** (`mem0/client/main.py:386-394`).

---

## 2. Write path & governance

### The pipeline (`main.py:835-1162`, sync; `2465-2798`, async — byte-for-byte parallel)

`infer=False` → raw passthrough, one memory per message (`main.py:836-870`).

`infer=True` → "V3 PHASED BATCH PIPELINE" (the comment at `main.py:872` is literal):

- **Phase 0** — pull last 10 messages for the session from SQLite (`main.py:875-877`). Session scope is a deterministic string join (`_build_session_scope`, `main.py:373-380`).
- **Phase 1** — vector-search top-10 existing memories in scope (`main.py:880-887`).
- **Anti-hallucination remap** (`main.py:889-894`) — real UUIDs are replaced with `"0"`, `"1"`, … before being shown to the LLM, and a `uuid_mapping` is kept locally. Small, cheap, effective. **Steal this.**
- **Phase 2** — one LLM call, `response_format={"type":"json_object"}`, system prompt = `ADDITIVE_EXTRACTION_PROMPT` (+ `AGENT_CONTEXT_SUFFIX` if agent-scoped and not user-scoped, `main.py:897-900`). Failures now **re-raise** as `LLMError` instead of silently returning `[]` (`main.py:919-925`) — a good 2026 fix, called out in the changelog.
- **Phase 3** — batch-embed all extracted texts, with per-item fallback (`main.py:947-959`).
- **Phase 4/5** — MD5 dedup against retrieved-neighbor hashes and within-batch (`main.py:961-995`).
- **Phase 6** — batch insert + batch history rows, each with a one-by-one fallback (`main.py:1001-1040`).
- **Phase 7** — spaCy entity extraction, global dedup, batch embed, batch search, exact-or-`score>=0.95`-semantic match, upsert into a **sidecar `<collection>_entities` collection** (`main.py:1042-1146`, collection naming at `main.py:383-385`, lazy init at `main.py:519-541`).
- **Phase 8** — persist messages (keeping only the most recent 10 per scope — `storage.py:279-291`).

### Is an LLM deciding ADD/UPDATE/DELETE? **No longer.**

```
$ grep -rn "DEFAULT_UPDATE_MEMORY_PROMPT|get_update_memory_messages" --include='*.py' .
mem0/configs/prompts.py:176   # definition
mem0/configs/prompts.py:406   # definition
mem0/configs/prompts.py:408-409  # self-reference
```

Zero call sites in `mem0/memory/`. The 150-line ADD/UPDATE/DELETE/NONE reconciler prompt everyone cites from the Mem0 paper is **vestigial**. Same for `FACT_RETRIEVAL_PROMPT` — only reachable via `get_fact_retrieval_messages_legacy` (`mem0/memory/utils.py:31-33`), itself uncalled.

What replaced it (`prompts.py:468-472`):

> "You are a Memory Extractor — a precise, evidence-bound processor... **Your sole operation is ADD**"

Consequence: **memory count only grows.** Contradiction is handled by *linking*, not resolution — and OSS drops the links (see §4). A user who says "I love X" then "I hate X" ends up with both rows, both retrievable, both equally scored. The prompt's own "Contradiction" link category (`prompts.py:699`) is written to nowhere.

### What prevents poisoning or stale data?

Concretely, in OSS:

1. **MD5 exact-text dedup** (`main.py:976-980`) — defeated by a single changed character.
2. **Prompt-level integrity rules** (`prompts.py:677-689`): "No Fabrication", "No Implicit Attribute Inference", "No Detail Contamination from Context". These are *instructions to a model*, not enforcement.
3. **`expiration_date`**, only if the caller sets it. Nothing sets it automatically.

That is the complete list. There is **no approval queue, no review state, no quarantine, no writer identity check, no conflict detection, no rate/volume guard, no trust weighting**. Any caller holding a `user_id` string can write anything into that user's memory, and it will be retrieved as fact forever.

`AGENT_CONTEXT_SUFFIX` (`prompts.py:947-957`) is the closest thing to a provenance discipline, and it is prompt prose: *"frame as agent knowledge: 'Agent was informed that [fact]'"*.

### Update / delete

Manual only, and unscoped:

- `Memory.update(memory_id, text=..., metadata=..., expiration_date=...)` — `main.py:1771-1821`. Writes history rows via `_update_memory` (`main.py:1972`).
- `Memory.delete(memory_id)` — `main.py:1823-1842`. Hard delete from the vector store.
- `Memory.delete_all(user_id=, agent_id=, run_id=)` — `main.py:1844-1884`; requires at least one filter, else tells you to use `reset()`.

### The one real governance artifact: the history log

`mem0/memory/storage.py:102-126` — SQLite `history` table:

```sql
CREATE TABLE history (
    id TEXT PRIMARY KEY, memory_id TEXT,
    old_memory TEXT, new_memory TEXT, event TEXT,
    created_at DATETIME, updated_at DATETIME,
    is_deleted INTEGER, actor_id TEXT, role TEXT
)
```

This is a real append-only change log with before/after text, exposed via `Memory.history(memory_id)` (`main.py:1886-1899`). **But**: it is local SQLite at `~/.mem0/history.db` (`configs/base.py:13, 42-45`), single-file, `check_same_thread=False` with a Python-level `threading.Lock` (`storage.py:14-15`) — it does not survive a multi-process or multi-node deployment, and it lives in a completely different store from the vectors. In the ADD-only path, `actor_id` and `role` are **never populated** (`main.py:1021-1031` passes neither).

---

## 3. Retrieval

`Memory.search()` → `_search_vector_store` (`main.py:1584-1687`). Nine steps:

1. Lemmatize query (`lemmatize_for_bm25`).
2. spaCy entity extraction from query (`utils/entity_extraction.py:751`).
3. Embed query.
4. Semantic search, **over-fetching `max(limit*4, 60)`** (`main.py:1597`).
5. `vector_store.keyword_search(...)` — returns `None` on stores that don't implement it (`vector_stores/base.py:68-83`); a warning is emitted at construction time if so (`main.py:504-511`).
6. BM25 normalization via a **query-length-adaptive logistic sigmoid** (`utils/scoring.py:16-54`) — midpoint/steepness step from `(5.0, 0.7)` at ≤3 terms to `(12.0, 0.5)` at >15.
7. Entity boosts (`_compute_entity_boosts`, `main.py:1689-1769`): query entities → entity sidecar store → for each match with `score >= 0.5`, boost its `linked_memory_ids`, damped by how many memories that entity links to (`memory_count_weight = 1/(1 + 0.001*(n-1)²)`, `main.py:1758`). Runs on a 4-thread pool.
8. `score_and_rank` (`utils/scoring.py:60-139`) — purely **additive**, with an adaptive divisor:

```python
max_possible = 1.0
if has_bm25:   max_possible += 1.0
if has_entity: max_possible += ENTITY_BOOST_WEIGHT   # 0.5
...
if semantic_score < threshold: continue     # threshold gates BEFORE combining
combined = min((semantic + bm25 + entity_boost) / max_possible, 1.0)
```

9. Format + promote payload keys.

**Reranking** is opt-in per call (`rerank=True`, `main.py:1456-1461`), configured globally, and swallows its own exceptions. Five providers (`utils/factory.py:230-239`).

**Filtering:** a decent operator language — `eq/ne/gt/gte/lt/lte/in/nin/contains/icontains`, `AND`/`OR`/`NOT`, `"*"` wildcard (documented `main.py:1358-1373`, translated at `_process_metadata_filters`, `main.py:1480-1556`). Each vector store must translate the universal operator dict itself, so real support is per-provider and uneven. At least one of `user_id`/`agent_id`/`run_id` is mandatory in filters (`main.py:1413-1417`) — this is the *only* enforced isolation in the core SDK.

### Token budgets: **none**

`search()` takes `top_k` (default 20) and `threshold`. There is no token counting, no context-window budget, no truncation-to-fit, no per-result cost accounting, no packing strategy anywhere in `mem0/`. Whatever comes back, the caller stuffs into a prompt. For CRED, this is a whole missing layer.

### Graph: **removed from OSS core**

`grep -ril graph mem0/ --include='*.py'` → 4 files, all incidental (`exceptions.py`, `neptune_analytics.py`, a prompt string, `configs/vector_stores/neptune.py`). There is **no `graph_store` field on `MemoryConfig`** (`configs/base.py:29-57`) and no `GraphMemory`/Neo4j code. `docs/platform/platform-vs-oss.mdx` marks "Graph memory: ✅ Built-in (Platform) / ✅ External graph store (OSS)" — the "external graph store" is the Neptune Analytics *vector* provider. The entity sidecar collection is the OSS replacement, and it is a flat entity→memory-id inverted index, not a graph: no edge types, no relations, no traversal.

Also platform-only per that same table: **temporal reasoning, memory decay, webhooks, memory export, dashboard, analytics** — with in-code stubs that raise and then upsell (`main.py:773-774`, `422-446`).

---

## 4. Provenance & citation

**You cannot trace a memory to its source.** What is retained per memory:

- `attributed_to` — the string `"user"` or `"assistant"`. That's the entire attribution model.
- `created_at`.
- `role` / `actor_id` — populated **only on the `infer=False` raw path** (`main.py:850-856`), never on the inferred path.
- The last ≤10 raw messages for the session, in a separate SQLite table, with **no foreign key to any memory** and hard-evicted at 10 (`storage.py:279-291`).

Not retained: source message ID, character span, document/URI, conversation ID, commit/PR/ticket, extraction model or prompt version, timestamp of the underlying event distinct from write time.

**The most damning detail.** `ADDITIVE_EXTRACTION_PROMPT` spends three separate sections instructing the model to emit `linked_memory_ids` (`prompts.py:513`, `692-701`, `936`), and the schema example at `prompts.py:926` shows it. Then `_add_to_vector_store` builds the payload:

```python
mem_metadata = deepcopy(metadata)
mem_metadata["data"] = text
mem_metadata["text_lemmatized"] = text_lemmatized
mem_metadata["hash"] = mem_hash
...
if mem.get("attributed_to"):
    mem_metadata["attributed_to"] = mem["attributed_to"]
```
— `main.py:985-993`

`linked_memory_ids` is **never read from the LLM response**. The only `linked_memory_ids` in the system are the entity sidecar's, computed independently by spaCy. So OSS pays for the tokens, the model does the linking work, and the result is dropped on the floor. Memory-to-memory provenance/supersession exists in the *prompt* and in the *hosted* client (`client/main.py:392`), not in the OSS write path.

Similarly, `generate_additive_extraction_prompt` accepts `summary=` and `recently_extracted_memories=` (`prompts.py:1016-1021`) — the OSS caller passes neither (`main.py:904-909`). These are platform-side inputs left in the ported file.

---

## 5. Permissions / ACL

### Core SDK (`mem0/`): none whatsoever

- `MemoryConfig` has no auth, tenant, or principal concept (`configs/base.py:29-57`).
- `Memory.get(memory_id)` — **`main.py:1164-1175`, takes no scope**. Hands back any memory in the collection by UUID, regardless of which `user_id` owns it.
- `Memory.update(memory_id, ...)` — `main.py:1771`, same.
- `Memory.delete(memory_id)` — `main.py:1823`, same.

Isolation is therefore *convention*: `search`/`get_all`/`delete_all` require a scope filter, `get`/`update`/`delete` do not. All memories for all users share one vector collection by default. There is no org, project, team, role, group, or grant primitive in the package.

### OpenMemory (`openmemory/api/`): a schema without an implementation

There *is* an `AccessControl` table (`openmemory/api/app/models.py:132`): `subject_type`, `subject_id`, `object_type`, `object_id`, `effect`. And an evaluator, `check_memory_access_permissions` (`openmemory/api/app/utils/permissions.py:8-53`). Precisely what it does:

```python
if memory.state != MemoryState.active: return False   # L28
if not app_id: return True                            # L32
if not app.is_active: return False                    # L41
accessible = get_accessible_memory_ids(db, app_id)
if accessible is None: return True                    # L49
return memory.id in accessible
```

And `get_accessible_memory_ids` (`openmemory/api/app/routers/memories.py:60-97`):

```python
if not app_access:
    return None          # L73-74: NO RULES == ALLOW EVERYTHING
for rule in app_access:
    if rule.effect == "allow":
        if rule.object_id: allowed.add(rule.object_id)
        else: return None                 # L86: allow-all, short-circuits any later deny
    elif rule.effect == "deny":
        if rule.object_id: denied.add(rule.object_id)
        else: return set()                # L91
if allowed_memory_ids:                    # L94: denies apply ONLY inside an allowlist
    allowed_memory_ids -= denied_memory_ids
```

Four defects, all structural:

1. **Default-open** (L73-74). Absence of policy = full access.
2. **Order-dependent, unordered.** A blanket `allow` returns before later `deny` rows are seen, and the query has no `ORDER BY` — evaluation depends on DB row order.
3. **Deny-only policies over-block.** With only `deny` rows and no `allow`, the L94 guard is false, subtraction is skipped, and `set()` is returned → everything blocked, not just the denied items.
4. **No write path exists.** `grep -rn "AccessControl(" openmemory/` returns exactly one hit — the class definition in `models.py:132`. **No code anywhere constructs an `AccessControl` row.** The ACL is dormant by construction.

`subject_type` is generic but only `"app"` is ever queried; no `user` subject is evaluated.

### Authentication in OpenMemory MCP: none

Identity is a URL path parameter (`openmemory/api/app/mcp_server.py:435`, `496`), read into a contextvar (`mcp_server.py:55-56`, `439-442`, `509-512`), and `get_user_and_app` (`app/utils/db.py:29`) **auto-creates** the user and app on first sight. Anyone who can reach the URL is any user they type. Two additional isolation bugs found:

- `search_memory` (`mcp_server.py:186-192`): `allowed = set(...) if accessible_memory_ids else None`, then `if allowed and ...`. When the user has zero accessible memories, `allowed` is `None` and the ACL check is **skipped entirely**.
- `pause_memories(global_pause=True)` (`routers/memories.py:434-441`) queries `Memory` with **no `user_id` filter** — one call pauses every memory of every user in the deployment.

### The REST server has real authentication and still no authorization

`server/` is the most mature auth surface in the repo — and it proves the point. `server/auth.py` (220 LOC) resolves four schemes in `verify_auth` (`auth.py:144-175`): HS256 JWT bearer (30-min access / 30-day refresh, `auth.py:15-18`), per-user bcrypt-hashed API keys with prefix-indexed lookup (`auth.py:126-141`, format `m0sk_<token_urlsafe(32)>`), a legacy `ADMIN_API_KEY` env var (constant-time compared, `auth.py:160`), and an `AUTH_DISABLED=1` escape hatch (`auth.py:167-169`). Single-use refresh-token JTIs (`alembic/versions/005_refresh_token_jtis.py`). That is competent authentication.

Then:

```python
async def add_memory(memory_create: MemoryCreate, _auth = Depends(verify_auth)):
    response = get_memory_instance().add(messages=[...], **params)
```
— `server/main.py:366-374`

**The authenticated principal is bound to `_auth` and never used.** Scoping comes entirely from the caller-supplied `user_id` in the request body. Any authenticated caller can read, write, or delete any other user's memories by typing their `user_id`. Same for `POST /search` (`main.py:451-485`) and `DELETE /memories` (`main.py:526`).

Supporting facts: only 5 tables (`User`, `APIKey`, `RequestLog`, `RefreshTokenJti`, `Settings`) — **no org/project/tenant entity and no `org_id`/`project_id` FK anywhere**; `role` is a bare string with only `admin` recognized (`models.py:25`), and migration `004_unique_admin_role.py:21` enforces **exactly one admin per deployment** via a partial unique index; config is a single global row keyed `"config_overrides"` (`server_state.py:29, 49`), so one Mem0 config serves every user; and the whole data plane is one process-global `Memory` singleton behind an `RLock` (`server_state.py:9-11`).

**Bottom line for CRED:** there is nothing here to reuse for governance. There is a table you could borrow the *shape* of, a competent auth module, and a three-layer demonstration that authentication without authorization is worth nothing — mem0 built JWT + API keys + bcrypt + refresh rotation, and still ships an API where the caller declares whose memories they are touching.

---

## 6. Storage abstraction

The best-engineered part of the repo.

`mem0/utils/factory.py` — four flat registries mapping provider name → dotted class path, resolved by `importlib` at call time (`load_class`, `factory.py:27-30`) so optional deps stay optional:

- `LlmFactory` — 18 providers, each with an optional provider-specific config class (`factory.py:40-59`); also `register_provider()` (`factory.py:126-137`) for out-of-tree LLMs.
- `EmbedderFactory` — 11 providers (`factory.py:151-163`).
- `VectorStoreFactory` — **24 providers** (`factory.py:179-204`).
- `RerankerFactory` — 5 providers (`factory.py:230-239`).

**Cost of a new vector store:** `VectorStoreBase` (`mem0/vector_stores/base.py`) has **11 abstract methods** — `create_col`, `insert`, `search`, `delete`, `update`, `get`, `list_cols`, `delete_col`, `col_info`, `list`, `reset` — plus two optional hooks with working defaults: `keyword_search` (returns `None` → hybrid gracefully degrades to semantic-only, `base.py:68-83`) and `search_batch` (sequential fallback, `base.py:85-100`). Reference size: the smallest real providers are `s3_vectors.py` (213 LOC), `langchain.py` (216), `supabase.py` (237). So **~200-350 LOC + a Pydantic config class in `mem0/configs/vector_stores/` + one registry line.**

The score-normalization contract is documented in the interface itself (`base.py:16-25`): every implementation must return higher-is-better similarity, with explicit conversions given for cosine/L2/inner-product. That single docstring is why the additive scorer in §3 can be provider-agnostic. **Steal this pattern.**

Weaknesses: `VectorStoreFactory.create` just splats the config dict into the constructor (`factory.py:213`) — no interface validation, so a broken provider fails at first use, not at construction. And each provider must independently implement the filter operator language, so `contains`/`OR` support is silently uneven across the 24.

---

## 7. Interfaces

| Surface | Location | Notes |
|---|---|---|
| Python SDK (OSS) | `mem0.Memory`, `mem0.AsyncMemory` — `mem0/memory/main.py:448`, `2110` | full duplicate implementation, sync + async |
| Python SDK (hosted) | `mem0.MemoryClient` — `mem0/client/main.py` (1832 LOC) | strict superset |
| TypeScript SDK | `mem0-ts/src/oss/` | at parity — same `ADDITIVE_EXTRACTION_PROMPT` (`mem0-ts/src/oss/src/memory/index.ts:869`) |
| Self-hosted REST | `server/` (FastAPI + alembic + Next.js dashboard) | 14 memory endpoints inline in `server/main.py:321-556`; control plane in `server/routers/` (auth, api-keys, entities, requests). Thin passthrough to one global `Memory` singleton |
| MCP (OSS) | `openmemory/api/app/mcp_server.py` | SSE + streamable-HTTP, no stdio. **Lives in the sunset codebase** — `server/`, its designated successor, has no MCP surface at all (see provenance addendum at the top of this document) |
| MCP (hosted) | `https://mcp.mem0.ai/mcp/` — `integrations/mem0-plugin/.mcp.json` | requires `MEM0_API_KEY` |
| CLI | `cli/python`, `cli/node` | |
| Agent plugins | `integrations/mem0-plugin/` (Claude Code / Cursor / Codex / OpenCode), `.claude-plugin/marketplace.json`, `skills/` | see §8 |

### The exact MCP tool surface

`openmemory/api/app/mcp_server.py`, `FastMCP("mem0-mcp-server")` at line 43. **Five tools**, all with descriptions in the decorator rather than docstrings:

| Line | Signature | Description |
|---|---|---|
| 64 | `add_memories(text: str, infer: bool = True) -> str` | "Add a new memory. This method is called everytime the user informs anything about themselves... Set infer to False to store the memory verbatim without LLM fact extraction." |
| 149 | `search_memory(query: str) -> str` | "Search through stored memories. This method is called EVERYTIME the user asks anything." |
| 227 | `list_memories() -> str` | "List all memories in the user's memory" |
| 296 | `delete_memories(memory_ids: list[str]) -> str` | "Delete specific memories by their IDs" |
| 370 | `delete_all_memories() -> str` | "Delete all memories in the user's memory" |

Observations for CRED: **no `update` tool, no `get(id)` tool, no history/provenance tool, no scoping arguments** — `user_id` comes from the URL path, so an agent cannot express *which* scope it is reading or writing. Every tool returns `str`, not structured JSON. And the two write-destructive tools (`delete_memories`, `delete_all_memories`) are exposed to the model with no confirmation and no ACL.

A known-broken route: `handle_post_message` is defined twice (`mcp_server.py:470-474`); the `{client_name}/sse/{user_id}/messages/` route recurses into itself infinitely before the module-level definition shadows it.

### OpenMemory's data model is a better skeleton than the core SDK's

`openmemory/api/app/models.py` has things the core SDK lacks and CRED needs:

- `MemoryState` enum — `active | paused | archived | deleted` (`models.py:30-34`), with soft deletes + `archived_at`/`deleted_at`.
- `MemoryStatusHistory` (`models.py:161`) — every transition with `changed_by`.
- `MemoryAccessLog` (`models.py:176`) — `access_type` in `{search, list, delete, delete_all}`, read back via `GET /{memory_id}/access-log` (`routers/memories.py:488`).
- `App` with `is_active` (`models.py:60`) — a per-writer kill switch, enforced on both write (`mcp_server.py:86-87`) and read (`permissions.py:41`).
- `Category` + `memory_categories` (`models.py:112-129`), auto-assigned by an LLM.

Caveats before copying: the state machine has **no transition matrix** — `update_memory_state` (`routers/memories.py:37-57`) permits any state → any state including `deleted → active`; deletes are soft in SQL but hard in the vector store (`mcp_server.py:330`), so the two diverge permanently; and categorization runs an LLM call **synchronously inside a SQLAlchemy `after_insert` flush listener** (`models.py:230-243`), which is a latency and deadlock trap. Also `mcp_server.py:118-123` writes `old_state=None` into a `nullable=False` column on first insert.

---

## 8. OSS project mechanics

**License:** Apache-2.0, clean, no CLA-style friction, no BSL/SSPL rug-pull (`LICENSE`, `pyproject.toml:14`).

**OSS ↔ cloud boundary.** Not a code fork — the split is *inside* the OSS codebase, and it is aggressive:

- Platform-only, per `docs/platform/platform-vs-oss.mdx`: temporal reasoning, memory decay, webhooks, memory export, dashboard, analytics, "custom categories" (OSS "Limited"), "memory filters v2" (OSS "⚠️ via metadata").
- Stub-and-upsell in code: `Memory.add(timestamp=...)` raises (`main.py:773-774`); `project.update(decay=True)` raises (`main.py:430-432`); `_OSSProject.update` raises `"Project updates are not supported by the OSS Memory SDK."` (`main.py:419, 432`).
- **`mem0/memory/notices.py` is 1582 lines of instrumented in-terminal upsell.** It is telemetry-gated (`notices.py:82`), fetches remote PostHog feature flags (`notices.py:96-98`), reads the *ad copy from the flag payload* (`notices.py:103-106`), runs a **`displayed` vs `holdout` A/B split** (`notices.py:22-23, 117-123`), caps at 10 per 7-day window per notice type (`notices.py:29-30, 34-35`), and `print(copy, file=sys.stderr)` (`notices.py:143`). Trigger conditions include: >2000 memories stored, `top_k > 50`, a search taking >2s, deleting ≥5 memories, or a date-like string in your query (`notices.py:37-42, 821-843`).
- **Telemetry is on by default**, hardcoded PostHog key `phc_hgJkUVJFYtmaJqrvf6CYN67TIQ8yhXAkWzUn9AMU4yX` at `mem0/memory/telemetry.py:15-16`, 10% sampling on hot paths, 100% on lifecycle events (`telemetry.py:32, 53-56`). Opt-out only via `MEM0_TELEMETRY=false`.
- **`skills/mem0-oss-to-platform/SKILL.md`** ships in the repo: an agent skill whose sole job is to migrate a codebase off `Memory` onto `MemoryClient`. There is no skill for the reverse direction.
- The shipped Claude Code plugin points at the **hosted** MCP requiring `MEM0_API_KEY` (`integrations/mem0-plugin/.mcp.json`); OSS users get OpenMemory, which is a separate app with its own Postgres.

**Repo layout:** `mem0/` (core), `mem0-ts/` (TS SDK), `server/` (REST), `openmemory/` (self-hosted app + MCP + UI), `cli/`, `integrations/`, `skills/`, `examples/`, `tests/`, `docs/`, `evaluation/` (empty). Coherent, if sprawling.

**Release cadence:** 198 `<Update>` entries in `docs/changelog/sdk.mdx`; roughly 2-4 releases/month through 2026 (2026-07-13, 07-01, 06-27, 06-24 ×2, 06-17, 06-13, 06-10, …). Very much alive. 21 GitHub Actions workflows.

**Test posture:** 95 `test_*.py` under `tests/`, ~1,706 test functions. Distribution is telling — `tests/vector_stores/` 733 tests, root 395, `tests/memory/` 224, `tests/llms/` 186, `tests/embeddings/` 77, `tests/rerankers/` 45, `tests/utils/` 43, `tests/configs/` **3**.

- **Provider adapters are heavily tested; product logic is not.** `mem0/memory/main.py` (3,777 LOC) is covered by `tests/test_main.py` (542 LOC / 40 tests) and `tests/memory/test_main.py` (1,085 LOC). Happy paths are single smoke tests (`tests/test_main.py:56` `test_add`, `:112` `test_search`); the bulk is input validation (`test_search_rejects_threshold_above_1` :485) and metadata-mutation regressions. Everything is mocked at the LLM boundary, so **no test asserts that a given conversation produces the correct memory decisions** — only that failures propagate (`test_llm_extraction_exception_is_reraised` :83).
- **Prompts are near-untested.** `prompts.py` is 1,062 LOC with ~10 prompt constants; `tests/configs/test_prompts.py` is **51 LOC / 3 tests**, all covering `get_update_memory_messages` string assembly — i.e. the one function with zero call sites. `ADDITIVE_EXTRACTION_PROMPT` and `generate_additive_extraction_prompt`, the entire live write path, have **zero coverage**.
- **No coverage gate.** `ci.yml:113` has a step *named* "Run tests and generate coverage report" that runs `make test` → plain `pytest tests/`. No `pytest-cov`, no `--cov`, no threshold, no upload anywhere in `pyproject.toml`, `Makefile`, or the workflows.
- **`server/` is in no CI path filter.** `grep -rn "server/" .github/workflows/*.yml` → nothing. Editing `server/main.py` alone triggers zero CI; the server tests (`tests/test_server_auth.py` 47 tests, `test_server_params.py` 68 tests) run only incidentally because they live under `tests/**`.
- `make lint` runs `ruff check` only — `ruff format --check` exists as a script and is never enforced (`ci.yml:112`).
- `mem0/memory/notices.py` and `telemetry.py` have three dedicated test files (`test_telemetry.py`, `test_telemetry_sampling.py`, `test_telemetry_aliasing.py`). The upsell machinery is better tested than the extraction prompts.

**CI structure** is worth copying even if the coverage posture isn't: `ci-gate.yml` is the single required PR check — it path-filters (`:51-88`), invokes only the relevant package workflows as reusable workflows, and aggregates via `jq` over `needs` in a `gate` job (`:146-171`). Python matrix is 3.10/3.11/3.12. There is a nice changelog gate: bumping the version in `pyproject.toml` requires a matching `docs/changelog/sdk.mdx` edit (`ci.yml:19-48`).

**Contribution friction is high — deliberately.** `CONTRIBUTING.md:14` "**Always open an issue before opening a pull request**"; `:23` wait for maintainer approval of the approach before non-trivial work; `:30` "**We cannot accept or merge any pull request until you have signed our Contributor License Agreement**". A CLA is the standard precondition for a company that wants to relicense or absorb contributions into a closed platform — read it alongside §8's OSS/cloud boundary. Minor rot: `CONTRIBUTING.md:44` claims Python 3.9+ and `:71` references a `dev_py_3_9` hatch env that `pyproject.toml` does not define (`requires-python = ">=3.10"`); `:68` says "Do not use `pip` or `conda`" while `Makefile:14-16` and `ci.yml:108` both use raw `pip install`. `.pre-commit-config.yaml` uses `language: system` hooks, which silently no-op outside the hatch shell.

**Realistic cost of a new vector store provider** (correcting §6's floor with the full touchpoint count): `mem0/vector_stores/<name>.py` ~215-240 LOC minimum (median existing ~360, max `databricks.py` ~950) + `mem0/configs/vector_stores/<name>.py` 28-46 LOC + one line in `vector_stores/configs.py:32` + one line in `utils/factory.py:195` + a `pyproject.toml` optional-dep entry + `docs/components/vectordbs/dbs/<name>.mdx` + a `docs/llms.txt` registration + a test file (~27 tests average). **~400-600 LOC across 7 touchpoints.** The two registration maps are separate and manually synced with nothing validating agreement — an easy trap CRED should avoid by deriving one from the other.

**Docs:** 243 `.mdx` files, Mintlify-hosted, genuinely good. Plus `AGENTS.md` (611 lines, symlinked to `CLAUDE.md`) and `LLM.md` (1324 lines) — agent-readable repo docs, which is itself a good adoption pattern to copy.

**What makes it adoptable:** `pip install mem0ai` + one API key → working memory in 5 lines; Apache-2.0; 24 vector stores so it drops into whatever infra you have; excellent docs; agent-native distribution (MCP + Claude Code/Cursor/Codex plugins + skills marketplace). The friction is all *downstream* — you discover the missing governance after you've built on it.

---

## Top 3 to STEAL

1. **The UUID→ordinal remap before showing memories to the LLM** — `mem0/memory/main.py:889-894`. Three lines that make ID hallucination structurally impossible: the model only ever sees `"0"`, `"1"`, `"2"`, and the mapping back to real UUIDs stays in application code. CRED's write path will show agents far more existing context than mem0 does (cross-repo, cross-agent), so this matters *more* for us, not less. Apply the same trick to repo IDs, commit SHAs, and ticket keys.

2. **The provider contract: 11 abstract methods, degrade-gracefully optional hooks, and a normalization contract written into the interface docstring** — `mem0/vector_stores/base.py:16-25` (score normalization), `:68-83` (`keyword_search` returns `None` and hybrid quietly falls back), plus `mem0/utils/factory.py:27-30` (lazy `importlib` so optional deps stay optional) and `:126-137` (`register_provider` for out-of-tree providers). This is why they have 24 stores at ~250 LOC each. Copy the shape verbatim.

3. **Additive, explainable scoring with an adaptive divisor** — `mem0/utils/scoring.py:60-139` and the `explain=True` path surfacing `score_details` end-to-end (`main.py:1682-1683`, `1376`). No opaque RRF constant, no magic weights: semantic + BM25 + entity boost over a divisor that shrinks when a signal is unavailable, so scores stay comparable across stores that do and don't support keyword search. For CRED, where "why did this context get injected?" is a first-class product question, a scorer that can explain itself per-result is table stakes — and this is a working reference implementation. Take the query-length-adaptive BM25 sigmoid (`scoring.py:16-54`) too.

Honorable mention worth reading but not copying wholesale: `ADDITIVE_EXTRACTION_PROMPT` (`prompts.py:468-946`). Its *specificity-preservation* rules ("NEVER replace a specific noun, number, title, or description with a vague category", `prompts.py:640-666`) are exactly the failure mode CRED will hit when compressing engineering decisions into memories.

## Top 3 NOT to copy

1. **The ADD-only write path.** `main.py:872-1162` + `prompts.py:472` ("Your sole operation is ADD"). It makes the store monotonically growing, contradiction-tolerant, and unauditable: "I love X" and "I hate X" coexist and score identically. They shipped it because reconciliation is hard and recall benchmarks reward over-extraction — but for an org-wide plane where a stale "we use REST" outlives the gRPC migration and gets injected into every agent, ADD-only is a correctness bug, not a tradeoff. CRED needs explicit supersession with validity windows, not links the writer discards (`main.py:985-993`).

2. **Scope-as-a-bare-string, with unscoped mutators — and authentication mistaken for authorization.** `_build_filters_and_metadata` (`main.py:287-370`) plus `get`/`update`/`delete` taking no scope at all (`main.py:1164`, `1771`, `1823`). Authorization is not "forgot to add later" — it is architecturally excluded, and every retrofit attempt failed the same way: OpenMemory's ACL is default-open with **zero writers** (`openmemory/api/app/routers/memories.py:73-74`; no `AccessControl(` constructor anywhere), and the REST server built JWT + bcrypt API keys + refresh rotation and then bound the principal to `_auth` and never read it (`server/main.py:366-374`), so any authenticated caller names whose memories they touch. CRED must make the principal a **required argument** on every read and write from commit one, deny-by-default, with ordered policy evaluation. Retrofitting is what produced `global_pause` wiping every user's memories (`openmemory/api/app/routers/memories.py:434-441`) and a `search_memory` that skips its ACL check whenever the allow-set is empty (`mcp_server.py:186-192`).

3. **`notices.py` + default-on telemetry as a growth channel.** 1582 lines that fetch remote feature flags, A/B-test terminal upsell copy against a holdout group, and print to stderr when your library gets *successful enough* — >2000 memories, `top_k>50`, slow query (`notices.py:37-42`). Paired with a hardcoded PostHog key on by default (`telemetry.py:15-16`) and a bundled skill for migrating *off* the OSS build (`skills/mem0-oss-to-platform/`). CRED sells to engineering orgs where a library phoning home about repo scale is a procurement blocker. Default telemetry off, opt-in, documented, no remote-controlled copy.

## The single biggest structural gap

**There is no concept of a memory being *shared* — so there is nothing to govern, and nowhere for provenance to attach.**

Mem0's atom is a private sentence about one `user_id`, produced by one conversation, consumed by one agent. Every downstream absence follows from that one modelling choice:

- No principal on read or write → no ACL, no tenancy, no team (`main.py:1164`, `1771`, `1823`).
- No source pointer → no citation, no "who claimed this and from what evidence" (`main.py:985-993` — even the LLM's own links are dropped).
- No validity window or supersession → no way to retire a decision when the codebase moves on (`configs/base.py:16-26`, `main.py:388-412`).
- No confidence or review state → no way to distinguish a ratified architectural decision from something an agent inferred at 2am (`grep confidence|review|approved mem0/` → zero fields).
- No outcome signal → nothing connects a retrieved memory to whether the thing it informed shipped, passed CI, or got reverted. `feedback()` exists **only on the hosted client** (`mem0/client/main.py:920`).
- No token budgeting → no notion that shared context is a contended, costed resource (`search()` has `top_k` and nothing else).

That is precisely CRED's territory. Mem0 has solved the boring, real, tedious parts — provider portability, extraction quality, hybrid scoring — and has left the entire governance plane empty, while simultaneously proving the distribution channel (MCP + editor plugins + skills) works. The correct read is not "compete with mem0's retrieval," it is **"treat mem0-class extraction as a commodity component and build the governed, provenance-bearing, outcome-traced layer that mem0 structurally cannot host."**
