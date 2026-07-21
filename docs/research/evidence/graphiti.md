# Evidence: Graphiti (Zep) — Temporal Knowledge Graph Engine

- **Repo scanned:** `/Users/canh/Solo/OSS/graphiti`
- **Version:** `0.29.2` (`pyproject.toml:4`), HEAD `0b4bcf1`, last commit 2026-07-17
- **License:** Apache-2.0 (`pyproject.toml:11`), CLA required (`Zep-CLA.md`)
- **Method:** direct source reading. Every claim below cites file:line.
- **Caveat:** the local clone is shallow (1 commit), so git-history-based community
  health signals were not measurable locally.

---

## Verdict

1. **The temporal model is real and it is the best in the landscape — but it is
   edge-only.** `EntityEdge` carries a genuine bi-temporal quad
   (`valid_at`/`invalid_at` = valid time, `created_at`/`expired_at` = transaction
   time, `edges.py:271-282`). `EntityNode` has **only `created_at`**
   (`nodes.py:499-503`). Entities have no history. Invalidation is non-destructive:
   contradicted edges get `invalid_at`/`expired_at` stamped and are re-saved, never
   deleted (`edge_operations.py:538-572`).
2. **Provenance is genuinely reconstructable, and this is Graphiti's strongest
   asset for CRED.** Every fact stores `episodes: list[str]`
   (`edges.py:267-270`), every episode stores raw `content` (`nodes.py:321`), and
   `MENTIONS` edges link episodes to entities. `get_nodes_and_edges_by_episode()`
   (`graphiti.py:1631-1643`) closes the loop. You *can* answer "why does the system
   believe X".
3. **Ingestion is brutally expensive — this is the disqualifying constraint.**
   A single `add_episode` costs roughly **3 + 2E + N LLM calls** (E = extracted
   edges, N = entities). A modest episode with 5 entities and 4 edges is ~16 LLM
   calls. There is no per-episode cost ceiling and no batching across episodes in
   the default path.
4. **The storage abstraction is a facade.** 116 `GraphProvider.` conditionals sit
   *outside* the driver package across 13 core files, and Neo4j Cypher is the
   implicit `case _:` default in every query builder. The ABC abstracts "a thing
   that executes Cypher strings" (`query_executor.py:41`), not "a graph store".
   A non-Cypher backend is a second execution path, not a driver.
5. **Recommendation: FORK THE CONCEPTS, DO NOT DEPEND ON THE LIBRARY.** Steal the
   bi-temporal edge model, the episode-provenance link, and the MinHash/LSH dedup
   fast path. Do not inherit the Neo4j coupling, the LLM-call-per-fact ingestion
   economics, or the `group_id` string as your tenancy boundary.

---

## 1. Temporal model

### The bi-temporal quad lives on the edge

`graphiti_core/edges.py:263-285`:

```python
class EntityEdge(Edge):
    name: str = Field(description='name of the edge, relation name')
    fact: str = Field(description='fact representing the edge and nodes that it connects')
    fact_embedding: list[float] | None = ...
    episodes: list[str] = Field(default=[], description='list of episode ids ...')
    expired_at: datetime | None = Field(
        default=None, description='datetime of when the node was invalidated')
    valid_at: datetime | None = Field(
        default=None, description='datetime of when the fact became true')
    invalid_at: datetime | None = Field(
        default=None, description='datetime of when the fact stopped being true')
    reference_time: datetime | None = Field(
        default=None, description='reference timestamp from the episode that produced this edge')
```

Plus `created_at` inherited from `Edge` (`edges.py:54`). So:

| Axis | Fields | Meaning |
|---|---|---|
| **Valid time** (world) | `valid_at`, `invalid_at` | when the fact was true in reality |
| **Transaction time** (system) | `created_at`, `expired_at` | when the system learned / stopped believing it |
| **Provenance time** | `reference_time` | the source episode's timestamp |

This is a correct bi-temporal design, and the third `reference_time` axis is a
genuinely thoughtful addition — it distinguishes "the fact is valid from March"
from "we learned this from a document dated April".

### It is edge-only — entities are not temporal

`graphiti_core/nodes.py:499-503`:

```python
class EntityNode(Node):
    name_embedding: list[float] | None = ...
    summary: str = Field(description='regional summary of surrounding edges', default_factory=str)
    attributes: dict[str, Any] = ...
```

Inherits only `created_at` (`nodes.py:98`). There is **no `valid_at`,
`invalid_at`, or `expired_at` on entities**. An entity's `summary` and
`attributes` are *overwritten in place* on each episode
(`node_operations.py:762-764`: `node.attributes = attributes`). Entity state has
no history and no audit trail.

**For CRED this is a material gap.** If you want to trace "what did this service
/ owner / policy look like on date T", edge-only bi-temporality does not give it
to you. You would need to extend the model to nodes.

### Contradiction detection is a single LLM call

Detection is entirely LLM-judged. `graphiti_core/prompts/dedupe_edges.py:43-100`
sends the new fact alongside two candidate lists and asks for indices back:

```python
class EdgeDuplicate(BaseModel):
    duplicate_facts: list[int] = Field(..., description='...only from EXISTING FACTS range...')
    contradicted_facts: list[int] = Field(..., description='...from full idx range...')
```

The prompt's discriminating example (`dedupe_edges.py:90-92`):

```
EXISTING FACT: idx=1, "Alice works at Acme Corp as a software engineer"
NEW FACT: "Alice works at Acme Corp as a senior engineer"
Result: duplicate_facts=[], contradicted_facts=[1] (same relationship but updated title
        — contradiction, NOT a duplicate)
```

Candidates come from two hybrid searches per extracted edge
(`edge_operations.py:389-423`): one scoped to edges already between the same node
pair (duplicate candidates), one unscoped (invalidation candidates). Both use
`EDGE_HYBRID_SEARCH_RRF`.

The LLM's index output is defensively validated (`edge_operations.py:735-744`,
`762-770`) — out-of-range indices are logged and dropped rather than trusted.
That is good engineering worth copying.

### Invalidation is deterministic interval arithmetic — not LLM-judged

This is the part worth stealing. Once the LLM has *nominated* contradictions,
the actual expiry decision is pure code, `edge_operations.py:538-572`:

```python
def resolve_edge_contradictions(resolved_edge, invalidation_candidates) -> list[EntityEdge]:
    invalidated_edges: list[EntityEdge] = []
    for edge in invalidation_candidates:
        edge_invalid_at_utc = ensure_utc(edge.invalid_at)
        resolved_edge_valid_at_utc = ensure_utc(resolved_edge.valid_at)
        edge_valid_at_utc = ensure_utc(edge.valid_at)
        resolved_edge_invalid_at_utc = ensure_utc(resolved_edge.invalid_at)

        # non-overlapping intervals -> no contradiction
        if (edge_invalid_at_utc is not None and resolved_edge_valid_at_utc is not None
            and edge_invalid_at_utc <= resolved_edge_valid_at_utc) or \
           (edge_valid_at_utc is not None and resolved_edge_invalid_at_utc is not None
            and resolved_edge_invalid_at_utc <= edge_valid_at_utc):
            continue
        # new edge supersedes the older one
        elif (edge_valid_at_utc is not None and resolved_edge_valid_at_utc is not None
              and edge_valid_at_utc < resolved_edge_valid_at_utc):
            edge.invalid_at = resolved_edge.valid_at
            edge.expired_at = edge.expired_at if edge.expired_at is not None else utc_now()
            invalidated_edges.append(edge)
    return invalidated_edges
```

The split is the design insight: **LLM proposes, interval algebra disposes.** The
LLM never writes a timestamp; it only nominates candidates. Overlap logic is
deterministic, testable, and replayable.

The symmetric case is handled too — a *new* edge can be born already expired if a
contradicting candidate has a later `valid_at` (`edge_operations.py:826-840`):

```python
if resolved_edge.expired_at is None:
    invalidation_candidates.sort(key=lambda c: (c.valid_at is None, ensure_utc(c.valid_at)))
    for candidate in invalidation_candidates:
        if candidate_valid_at_utc > resolved_edge_valid_at_utc:
            # Expire new edge since we have information about more recent events
            resolved_edge.invalid_at = candidate.valid_at
            resolved_edge.expired_at = now
            break
```

Invalidated edges are then persisted alongside resolved ones —
`graphiti.py:1156`: `entity_edges = resolved_edges + invalidated_edges`. Nothing
is deleted. History is preserved.

### Timestamp extraction is its own LLM call

`edge_operations.py:576-620` (`_extract_edge_timestamps`) fires a separate
`ModelSize.small` call per new edge. The prompt
(`prompts/extract_edges.py:242-267`) is tight and anti-hallucination:

```
- Resolve relative expressions ("last week", "2 years ago", "yesterday") using REFERENCE TIME.
- If the fact is ongoing (present tense), set valid_at to REFERENCE TIME.
- Leave both null if no time is stated or resolvable.
- Do NOT hallucinate or infer dates from unrelated events.
```

System prompt: `'You extract temporal bounds from facts. NEVER hallucinate dates.'`

Note the guard at `edge_operations.py:588-592`: skipped entirely if timestamps
are already set or the episode has no `valid_at`. A `extract_timestamps_batch`
variant exists (`prompts/extract_edges.py:72-77`) but is only used in the
combined-extraction path.

### Point-in-time query: possible, but not a first-class API — and currently broken

`SearchFilters` (`search/search_filters.py:55-67`) does expose all four temporal
fields as DNF (OR-of-ANDs) date filters:

```python
    valid_at: list[list[DateFilter]] | None = Field(default=None)
    invalid_at: list[list[DateFilter]] | None = Field(default=None)
    created_at: list[list[DateFilter]] | None = Field(default=None)
    expired_at: list[list[DateFilter]] | None = Field(default=None)
```

But there is **no `as_of(T)` helper anywhere**. You must hand-assemble the
bitemporal predicate. And that exact pattern hits a real bug —
`search_filters.py:149-178`:

```python
    if filters.valid_at is not None:
        valid_at_filter = '('
        for i, or_list in enumerate(filters.valid_at):
            for j, date_filter in enumerate(or_list):
                ...
                filter_params['valid_at_' + str(j)] = date_filter.date   # <-- uses j only, not i
```

The parameter name ignores the outer OR index `i`. Two OR branches each holding
one filter both bind `$valid_at_0`, and the second silently overwrites the first.
The emitted Cypher becomes `(e.valid_at <= $valid_at_0) OR (e.valid_at > $valid_at_0)`
against a single value. The canonical "as of T" query —
`valid_at <= T AND (invalid_at > T OR invalid_at IS NULL)` — is precisely the
shape that breaks. Identical bug in the `invalid_at` (180-209), `created_at`
(211-240) and `expired_at` (242-271) blocks.

Two further defects in the same file: `valid_at=[]` emits the bare string `'('`
into the WHERE clause; and `property_filters` (`search_filters.py:67`) is declared
but never consumed by either filter constructor — setting it silently does nothing.

Scope limit: temporal filters are **edge-only**. `node_search_filter_query_constructor`
(`search_filters.py:86-104`) handles only `node_labels`. Episode search ignores
its filter argument entirely (`search_utils.py:889`, parameter named `_search_filter`).

**Verdict on temporal:** the *model* is excellent and worth copying wholesale.
The *query surface over it* is thin, untested (no test in `tests/` exercises
`SearchFilters` date fields), and has a live correctness bug on the flagship
use case.

---

## 2. Episode / provenance model

### What an Episode is

`graphiti_core/nodes.py:318-332`:

```python
class EpisodicNode(Node):
    source: EpisodeType = Field(description='source type')
    source_description: str = Field(description='description of the data source')
    content: str = Field(description='raw episode data')
    valid_at: datetime = Field(description='datetime of when the original document was created')
    entity_edges: list[str] = Field(description='list of entity edges referenced in this episode',
                                    default_factory=list)
    episode_metadata: dict[str, Any] | None = Field(
        description='customer-defined metadata key-value pairs for filtering', default=None)
```

`EpisodeType` (`nodes.py:54-87`): `message`, `json`, `text`, `fact_triple`. The
episode is the immutable unit of ingestion and the anchor of all provenance.

### The provenance graph is bidirectional

Three independent links, which is what makes reconstruction actually work:

1. **Fact → episodes.** `EntityEdge.episodes: list[str]` (`edges.py:267-270`).
   Appended on every re-observation (`edge_operations.py:696-697`,
   `edge_operations.py:751-752`) — so a fact confirmed by five episodes carries
   all five UUIDs. This is a corroboration count for free.
2. **Episode → facts.** `EpisodicNode.entity_edges` (`nodes.py:325-328`), set at
   `graphiti.py:1029`.
3. **Episode → entities.** `MENTIONS` edges via `EpisodicEdge`
   (`edges.py:143-160`), built by `build_episodic_edges`
   (`edge_operations.py:52-96`). Note the `node_episode_index_map` parameter — in
   multi-episode batches each node is linked *only* to the episodes that actually
   mentioned it (`edge_operations.py:77-84`), not blanket-linked. Precise
   attribution.

### Can you reconstruct "why does the system believe X"?

**Yes, and cleanly.** `graphiti.py:1631-1643`:

```python
    async def get_nodes_and_edges_by_episode(self, episode_uuids: list[str]) -> SearchResults:
        episodes = await EpisodicNode.get_by_uuids(self.driver, episode_uuids)
        edges_list = await semaphore_gather(
            *[EntityEdge.get_by_uuids(self.driver, episode.entity_edges) for episode in episodes],
            max_coroutines=self.max_coroutines)
        edges: list[EntityEdge] = [edge for lst in edges_list for edge in lst]
        nodes = await get_mentioned_nodes(self.driver, episodes)
        return SearchResults(edges=edges, nodes=nodes)
```

Given a fact, `edge.episodes` gives you source episode UUIDs; each episode
retains raw `content` and `source_description`; `reference_time` tells you when
the source claimed it. That is a complete provenance chain from derived fact back
to raw input.

**One critical caveat.** Raw content retention is *optional* and it is a
constructor-level switch — `graphiti.py:1026-1028`:

```python
        for ep in episodes:
            ep.entity_edges = [edge.uuid for edge in entity_edges]
            if not self.store_raw_episode_content:
                ep.content = ''
```

If `store_raw_episode_content=False`, provenance degrades to UUID pointers at
empty documents. For CRED, raw retention is non-negotiable and this flag should
not exist.

**Provenance gaps to note:**
- Entity `summary` and `attributes` have **no provenance and no history** — they
  are overwritten per episode (`node_operations.py:762-764`). You cannot ask why
  an entity summary says what it says.
- There is no record of *which model or prompt version* produced a fact. The
  `prompt_name` is threaded through for token accounting
  (`llm_client/token_tracker.py:33-44`) but never persisted onto the edge. For a
  governed system, this is a real omission — you cannot invalidate everything a
  known-bad model version wrote.

### Sagas — episode chains (new, and directly relevant to CRED)

`nodes.py:867-876` adds a `SagaNode`:

```python
class SagaNode(Node):
    summary: str = ''
    first_episode_uuid: str | None = None
    last_episode_uuid: str | None = None
    last_summarized_at: datetime | None = None
    last_summarized_episode_valid_at: datetime | None = None
```

Sagas group episodes into ordered threads via `HAS_EPISODE` edges
(`edges.py:689-725`) with consecutive episodes linked by `NEXT_EPISODE`
(`edges.py:822-860`). Wired in at `graphiti.py:1030-1078`. Incremental
re-summarization uses `last_summarized_at` as a watermark
(`graphiti.py:438`).

The `summarize_saga` prompt (`prompts/summarize_sagas.py:41-137`) is notable —
it explicitly strips conversational framing:

```
NEVER use meta-language verbs: "mentioned", "discussed", "noted", "stated", ...
The output must read as if no conversation happened — only the facts matter.
```

Its own worked example is a software-team decision log ("Deployment moved from
March 8 to March 15 because the staging environment is not ready. Priya owns
updating the client timeline."). **This is close to CRED's execution-memory use
case**, and it is the newest part of the codebase — evidence Zep is moving toward
the same territory.

---

## 3. Ontology / custom entity types

### How you define types

Plain Pydantic models passed per-call. `graphiti.py:980-998`:

```python
    async def add_episode(
        self, name, episode_body, source_description, reference_time,
        source=EpisodeType.message, group_id=None, uuid=None, update_communities=False,
        entity_types: dict[str, type[BaseModel]] | None = None,
        excluded_entity_types: list[str] | None = None,
        previous_episode_uuids: list[str] | None = None,
        edge_types: dict[str, type[BaseModel]] | None = None,
        edge_type_map: dict[tuple[str, str], list[str]] | None = None,
        custom_extraction_instructions: str | None = None,
        saga: str | SagaNode | None = None, ...
```

Effort is genuinely low — a dict of Pydantic models. But there are two sharp edges.

**The type description is the class docstring.** `node_operations.py:172-180`:

```python
        entity_types_context += [
            {
                'entity_type_id': i + 1,
                'entity_type_name': type_name,
                'entity_type_description': type_model.__doc__,
            }
            for i, (type_name, type_model) in enumerate(entity_types.items())
        ]
```

Your docstrings are load-bearing prompt text. Rewriting a docstring silently
changes extraction behaviour, and there is no versioning of that contract. There
is even a helper to trim them for summary prompts
(`node_operations.py:192-222`, `_truncate_type_description`).

**Edge types are constrained by node-pair signature.** `edge_type_map` is keyed
by `(source_label, target_label)` tuples. Resolution at
`edge_operations.py:466-486` computes the cartesian product of source and target
labels and unions the permitted edge types. Default when unspecified
(`graphiti.py:1115-1119`):

```python
                edge_type_map_default = (
                    {('Entity', 'Entity'): list(edge_types.keys())}
                    if edge_types is not None else {('Entity', 'Entity'): []}
                )
```

This is a proper typed-relation ontology — `EMPLOYS` can be restricted to
`(Organization, Person)`. Good design.

### How extraction is constrained

Weakly — by prompt, then validated in code. The entity-type list is interpolated
into the extraction prompt (`prompts/extract_nodes.py:119-120`), and the model
returns a `entity_type_id` index. The prompts are heavily defensive; the negative
constraints dominate (`prompts/extract_nodes.py:90-108`):

```
NEVER extract any of the following:
- Pronouns (you, me, I, he, she, they, we, us, it, them, this, that, those)
- Abstract concepts or feelings (joy, balance, growth, resilience, happiness, ...)
- Generic common nouns or bare object words (day, life, people, work, stuff, ...)
- Sentence fragments or clauses ("what you really care about", ...)
```

That volume of negative prompting is a tell: unconstrained LLM extraction
produces garbage entities, and this is empirical scar tissue. Copy the scar
tissue, not the approach.

Attribute extraction is separately capped in code
(`utils/maintenance/attribute_utils.py:27-32`):

```python
DEFAULT_ATTRIBUTE_MAX_LENGTH = 250
```

with per-item *and* aggregate caps on list-typed fields
(`attribute_utils.py:120-139`) to prevent "KB-scale list bleed". Over-cap fields
are dropped, not truncated (`cap_string_attributes`, `attribute_utils.py:141+`).
Sensible defensive design.

Structured output is effectively mandatory. `README.md:169-172`:

> Graphiti works best with LLM services that support Structured Output (such as
> OpenAI, Anthropic, and Gemini). Using other services may result in incorrect
> output schemas and ingestion failures.

---

## 4. Ingestion cost and latency

### The pipeline

`graphiti.py:1121-1178`, in order:

1. `extract_nodes` (`node_operations.py:70`)
2. `resolve_extracted_nodes` (`node_operations.py:627`)
3. `_extract_and_resolve_edges` → `extract_edges` + `resolve_extracted_edges`
   (`graphiti.py:1141-1153`)
4. `extract_attributes_from_nodes` (`node_operations.py:726`)
5. `_process_episode_data` → bulk write (`graphiti.py:1166-1175`)

### LLM call accounting

Eleven distinct prompts exist (`grep prompt_name=`):

```
   4 extract_edges.extract_attributes
   2 extract_nodes.extract_attributes
   1 summarize_sagas.summarize_saga
   1 summarize_nodes.summary_description
   1 summarize_nodes.summarize_pair
   1 extract_nodes_and_edges.extract_message
   1 extract_edges.extract_timestamps_batch
   1 extract_edges.extract_timestamps
   1 extract_edges.edge
   1 dedupe_nodes.nodes
   1 dedupe_edges.resolve_edge
```

Per `add_episode`, with E extracted edges and N entities:

| Stage | Calls | Evidence |
|---|---|---|
| Entity extraction | 1 (more if chunked) | `node_operations.py:244-281` |
| Node dedup | 0–1 (skipped by MinHash fast path) | `node_operations.py:552-553` |
| Edge extraction | 1 | `prompts/extract_edges.py:94` |
| Edge resolution | **E** (one per edge) | `edge_operations.py:725-731` |
| Edge timestamps | **E** (per *new* edge) | `edge_operations.py:598-604` |
| Edge attributes | up to E (if custom edge type) | `edge_operations.py:783-790` |
| Node attributes | **N** (if custom type has fields) | `node_operations.py:743-758` |
| Node summaries | ~N/batch | `node_operations.py:833` |

**≈ 3 + 2E + N calls.** Five entities and four edges ≈ 16 LLM calls for one
episode. With a custom ontology, closer to 25. The fan-out is concurrent but
unbounded in count — only in parallelism (`helpers.py:38`):

```python
SEMAPHORE_LIMIT = int(os.getenv('SEMAPHORE_LIMIT', 20))
```

There is **no cost ceiling per episode.** An entity-dense document produces
proportionally more calls with no cap.

Mitigations that do exist:
- `ModelSize.small` routing for cheap tasks (`llm_client/config.py:23,44,66`) —
  dedup, timestamps, attributes all run on the small model.
- Per-prompt token accounting (`llm_client/token_tracker.py:33-44`) —
  `PromptTokenUsage` tracks `call_count`, input/output tokens, and averages *by
  prompt name*. Excellent cost observability; worth copying directly.
- An LLM response cache (`llm_client/cache.py:27-66`), SQLite+JSON (explicitly
  replacing diskcache over a pickle CVE).
- Density-based chunking, not blanket chunking (`helpers.py:41-54`):
  `CHUNK_TOKEN_SIZE=3000`, `CHUNK_MIN_TOKENS=1000`, `CHUNK_DENSITY_THRESHOLD=0.15`.
  Comments note this targets "AWS cost data, bulk data imports, entity-dense JSON"
  while leaving "meeting transcripts, news articles, documentation" unchunked.

### Deduplication — the genuinely clever part

Before any LLM dedup call, a deterministic MinHash/LSH fast path runs
(`utils/maintenance/dedup_helpers.py`):

- `_normalize_string_exact` (39) / `_normalize_name_for_fuzzy` (45)
- `_name_entropy` (52) + `_has_high_entropy` (79) — an entropy gate that refuses
  fuzzy matching on short/low-entropy names where shingles are unreliable
- `_shingles` (88) — 3-gram shingles
- `_hash_shingle` (97) — `blake2b`, deterministic, seeded per permutation
- `_minhash_signature` (103), `_lsh_bands` (117), `_jaccard_similarity` (131)
- `_FUZZY_JACCARD_THRESHOLD = 0.9` (34)

`_resolve_with_similarity` (220) resolves exact and high-confidence fuzzy matches
with **zero LLM calls**; only the residue reaches `dedupe_nodes.nodes`. There is
a matching fast path for edges (`edge_operations.py:345-357`, exact fact+endpoint
match) and a verbatim-reuse shortcut (`edge_operations.py:688-698`).

**This is the single best cost idea in the codebase.** Deterministic resolution
first, LLM only for the ambiguous remainder.

### Latency and documented characteristics

No published benchmarks in-repo. The README is careful to attribute performance
claims to the *hosted* product, not the OSS core (`README.md:109`):

> **Retrieval & performance** | Zep: Pre-configured, production-ready retrieval
> with sub-200ms performance at scale | Graphiti: Custom implementation required;
> performance depends on your setup

The "sub-second latency" claim at `README.md:153` is in the GraphRAG comparison
table and refers to *query*, not ingestion. Ingestion latency is not
characterised anywhere.

The docstring is candid about the operational shape (`graphiti.py:1055-1060`):

> It is recommended to run this method as a background process, such as in a
> queue. It's important that each episode is added sequentially and awaited
> before adding the next one.

**Sequential per group.** Episodes cannot be safely parallelised within a
namespace because dedup and invalidation read the live graph. That is a
throughput ceiling, not just a latency one.

`add_episode_bulk` (`graphiti.py:1230`) exists with `CHUNK_SIZE = 10`
(`utils/bulk_utils.py:66`) and cross-batch dedup (`dedupe_nodes_bulk`, 374), but
carries an acknowledged O(n²) (`bulk_utils.py:416-419`):

```python
            # NOTE: this loop is O(n^2) in the number of nodes inside the batch because we rebuild
            # ... the shingle cache keeps the constant factors low for typical batch sizes
            # (<= CHUNK_SIZE), but if batches grow significantly we should switch to an
            # incremental index or chunked comparisons.
```

---

## 5. Retrieval

### What is implemented

Search methods (`search/search_config.py`):

| Enum | Line | Members |
|---|---|---|
| `EdgeSearchMethod` | 32-35 | `cosine_similarity`, `bm25`, `bfs` |
| `NodeSearchMethod` | 38-41 | `cosine_similarity`, `bm25`, `bfs` |
| `EpisodeSearchMethod` | 44-45 | **`bm25` only** |
| `CommunitySearchMethod` | 48-50 | `cosine_similarity`, `bm25` |

Rerankers: `EdgeReranker` (53-58) and `NodeReranker` (61-66) each support
`rrf`, `node_distance`, `episode_mentions`, `mmr`, `cross_encoder`;
`EpisodeReranker` (69-71) only `rrf`/`cross_encoder`.

16 recipes in `search_config_recipes.py` — `COMBINED_HYBRID_SEARCH_RRF` (34),
`..._MMR` (56), `..._CROSS_ENCODER` (81, the default for `search_()` per
`graphiti.py:1604`), plus edge-only (111-153), node-only (156-198) and
community-only (201-223) variants.

### BM25 is real

Native fulltext indexes per provider, `graph_queries.py:143-152`:

```python
    return f'CALL db.index.fulltext.queryNodes("{name}", {query}, {{limit: $limit}})'      # Neo4j
    return f"CALL db.idx.fulltext.queryNodes('{label}', {query})"                          # FalkorDB
    return f"CALL QUERY_FTS_INDEX('{label}', '{name}', {query}, TOP := $limit)"            # Kuzu
```

Neo4j's is Lucene-backed, so genuine BM25. Neptune diverges entirely to Amazon
OpenSearch (`search_utils.py:226`).

Fusion is **rank-based, not score-based** — each arm returns ordered UUID lists
and raw scores are discarded at the fusion boundary (`search.py:372`, `561`,
`701`, `803`). This correctly sidesteps the BM25/cosine normalisation problem.
Arms over-fetch `2 * limit` and truncate after rerank.

### Reranker implementations and their defects

RRF (`search_utils.py:1780-1795`) uses `rank_const=1`, not the standard 60:

```python
def rrf(results: list[list[str]], rank_const=1, min_score: float = 0):
    scores: dict[str, float] = defaultdict(float)
    for result in results:
        for i, uuid in enumerate(result):
            scores[uuid] += 1 / (i + rank_const)
```

Rank 0 contributes 1.0 and rank 1 contributes 0.5 — far steeper decay than k=60,
so a single retriever's top hit dominates fusion.

MMR (`search_utils.py:1901-1939`) is **not canonical greedy MMR** — it is a
one-shot scoring pass, so diversity is measured against all candidates rather
than the already-selected set:

```python
    for i, uuid in enumerate(uuids):
        max_sim = np.max(similarity_matrix[i, :])
        mmr = mmr_lambda * np.dot(query_array, candidate_arrays[uuid]) + (mmr_lambda - 1) * max_sim
```

Note `COMBINED_HYBRID_SEARCH_MMR` sets `mmr_lambda=1` (recipes 60/65/76), which
zeroes the diversity term — that recipe is pure cosine ranking wearing an MMR
label.

`node_distance_reranker` (1798) despite its "shortest path" comment (1828) matches
only 1-hop neighbours and returns a constant `1 AS score` (1816-1820) —
a binary adjacent/not-adjacent partition, not graded distance.

`episode_mentions_reranker` (1860) assigns un-mentioned nodes `float('inf')`
(1891) then sorts **ascending** (1894) — nodes with *no* episode mentions sort
to the top. Appears inverted.

### Cross-encoders

Three clients in `cross_encoder/`: `BGERerankerClient`
(`bge_reranker_client.py:34-54`, the only true cross-encoder,
`BAAI/bge-reranker-v2-m3`), `GeminiRerankerClient`, and `OpenAIRerankerClient`
— which is the interesting one (`openai_reranker_client.py:83-118`):

```python
                    self.client.chat.completions.create(
                        model=self.config.model or DEFAULT_MODEL,
                        messages=openai_messages, temperature=0, max_tokens=1,
                        logit_bias={'6432': 1, '7983': 1},
                        logprobs=True, top_logprobs=2,
                    )
                    ...
                norm_logprobs = np.exp(top_logprobs[0].logprob)
                if top_logprobs[0].token.strip().split(' ')[0].lower() == 'true':
                    scores.append(norm_logprobs)
                else:
                    scores.append(1 - norm_logprobs)
```

Boolean relevance classification with `max_tokens=1`, biasing the "True"/"False"
token IDs, then exponentiating the logprob into a probability. Clever, but
**one LLM call per passage**, and the hardcoded token IDs are tokenizer- and
model-specific — on a different model or `base_url` they bias arbitrary tokens.
A `continue` on empty logprobs (line 109) followed by `zip(..., strict=True)`
(116) means a single malformed response crashes the whole rerank.

### Cost of the default search

For `COMBINED_HYBRID_SEARCH_CROSS_ENCODER`: 1 embedding call, ~9 concurrent DB
queries across 4 scopes, **plus 2 sequential extra BFS roundtrips**
(`search.py:332-353`, `540-559`) that serialize the scope into two waves — and
up to ~33 concurrent LLM calls for reranking. Node and community rerankers pass
the *entire* candidate union rather than truncating to `limit` first (contrast
`search.py:397` which does `[:limit]` for edges, against 599 and 847-849 which
do not), so node reranking can issue 60+ LLM calls where edges issue 10.

**The default retrieval recipe is LLM-metered per query.** For CRED's
agent-in-the-loop retrieval this is the wrong default.

One silent-failure path worth knowing: `fulltext_query` returns `''` when the
query exceeds `MAX_QUERY_LENGTH=128` tokens (`search_utils.py:90-91, 107-109`)
and callers return `[]` — **a long query silently drops the BM25 arm** with no
warning, degrading to vector-only.

A concurrency hazard: recipes are module-level mutable singletons and
`graphiti.search()` mutates one in place (`graphiti.py:1570`:
`search_config.limit = num_results`). Process-wide side effect.

---

## 6. Storage abstraction

### The ABC is thin

`driver/driver.py:90-210`. Five abstract methods total: `execute_query` (101),
`session` (105), `close` (109), `delete_all_indexes` (113),
`build_indices_and_constraints` (127). `GraphProvider` (59-63) has exactly four
values: `NEO4J`, `FALKORDB`, `KUZU`, `NEPTUNE`.

The parameter is named `cypher_query_` (`query_executor.py:41`). **The
abstraction boundary is "a thing that executes Cypher strings", not "a graph
store."** That single naming choice tells you the whole story.

`execute_query`'s return type is unspecified (`-> Coroutine`) and the four
drivers return different shapes — Neo4j `EagerResult`, the others 3-tuples.
Callers universally destructure `records, _, _ = await driver.execute_query(...)`,
which works only because `EagerResult` happens to be a 3-namedtuple. Undocumented,
unenforced structural contract.

### Neo4j is the implicit default everywhere

116 `GraphProvider.` references outside `driver/`, across 13 files:

| File | Count |
|---|---|
| `search/search_utils.py` | 31 |
| `nodes.py` | 21 |
| `models/nodes/node_db_queries.py` | 17 |
| `models/edges/edge_db_queries.py` | 12 |
| `edges.py` | 11 |
| `graph_queries.py` | 10 |
| `utils/maintenance/graph_data_operations.py` | 3 |
| `utils/bulk_utils.py` | 3 |
| `utils/maintenance/edge_operations.py` | 2 |
| `utils/maintenance/community_operations.py` | 2 |
| `search/search_filters.py` | 2 |
| `helpers.py`, `decorators.py` | 1 each |

Kuzu accounts for ~34 branch sites, Neptune ~28, FalkorDB 5, and **Neo4j
essentially zero — because it is the `case _:` fallthrough** in every builder
(`node_db_queries.py:60, 102, 250, 320`; `edge_db_queries.py:103`;
`graph_queries.py:54-82, 131-140, 152, 163, 175`, each literally commented
`case _:  # Neo4j`).

Neo4j-only primitives hardcoded as the default branch: `db.create.setNodeVectorProperty`
(`node_db_queries.py:177, 252, 324`), `vector.similarity.cosine`
(`graph_queries.py:163`), `db.index.fulltext.queryNodes`
(`graph_queries.py:152`), `SET n:$(node.labels)` Neo4j-5 dynamic labels
(`node_db_queries.py:260`).

Neo4j driver kwargs leak into core call sites too — `routing_='r'` is passed
from `nodes.py:382` and other backends must strip it (`kuzu_driver.py:224`:
`params.pop('routing_', None)`).

`neo4j>=5.26.0` is a **hard, non-optional dependency** (`pyproject.toml:15`) even
if you only ever use FalkorDB. `helpers.py` imports `from neo4j import time as
neo4j_time` at module scope for date parsing.

### Kuzu's reification, and why it matters

`kuzu_driver.py:51-54` states the reason plainly:

```
# Kuzu requires an explicit schema.
# As Kuzu currently does not support creating full text indexes on edge properties,
# we work around this by representing (n:Entity)-[:RELATES_TO]->(m:Entity) as
# (n)-[:RELATES_TO]->(e:RelatesToNode_)-[:RELATES_TO]->(m).
```

Because facts must be full-text searchable and Kuzu can't index edge properties,
the edge becomes a node. The blast radius of that one decision reaches
`edge_db_queries.py:88,153`, `search_utils.py:207-211,325-328`, `nodes.py:562`,
`graph_queries.py:24`, and 9 of the 11 branches in `edges.py`.

**This is the single most transferable architectural lesson in the repo:** if
facts need to be first-class searchable objects with their own attributes and
temporal bounds, they *are* nodes, not edges. Kuzu forced the honest modelling.
Note also Kuzu is formally deprecated (`kuzu_driver.py:146-151`,
`pyproject.toml:31`) — so ~34 core branches exist to support a dying backend.

### Two abstraction layers, mid-migration

`driver.py:97-99` is explicit:

```python
# Legacy interfaces (kept for backwards compatibility during Phase 1)
search_interface: SearchInterface | None = None
graph_operations_interface: GraphOperationsInterface | None = None
```

A newer `driver/{neo4j,falkordb,kuzu,neptune}/operations/` layer exists at
~2,400-2,700 LOC *per provider* (~10k total) with proper ABCs
(`SearchOperations` has 15 abstract methods, `operations/search_ops.py:29-165`).
But outside the driver package it is reachable only via
`graphiti_core/namespaces/`. Three generations of abstraction coexist and the
codebase pays maintenance on all of them.

### Cost of a fifth backend

Cypher-speaking (e.g. Memgraph): ~2,500 LOC plus auditing all 116 branch sites,
because every fallthrough silently emits Neo4j Cypher at your new provider.

**Postgres/pgvector: not a driver — a second execution path.** Every inline
Cypher string in `search_utils.py`, `nodes.py`, `edges.py` and
`utils/maintenance/*` needs a SQL equivalent, and those strings are concatenated
inline inside core functions, not routed through a dispatch point. You would be
forced onto the `operations/` layer, i.e. ~2,500 LOC of ABC implementations, plus
recursive CTEs to replace variable-length path traversal, plus a `tsquery`
sanitizer, plus a datetime shim (every existing backend has one —
`neptune_driver.py:243-278` does *string rewriting of query text* to inject
`datetime(...)`).

Two footguns in the index path: both Neo4j (`neo4j_driver.py:92-101`) and
FalkorDB (`falkordb_driver.py:176-184`) **fire-and-forget**
`loop.create_task(self.build_indices_and_constraints())` in `__init__`, silently
skipping it when no loop is running. Index creation is racy relative to first
write. And `create_aoss_indices` contains a hardcoded `await asyncio.sleep(60)`
(`neptune_driver.py:327`).

---

## 7. group_id / multi-tenancy

### What exists

A single string on every node and edge. `edges.py:51`, `nodes.py:96`:

```python
    group_id: str = Field(description='partition of the graph')
```

Validation is character-class only (`helpers.py:136-157`):

```python
def validate_group_id(group_id: str | None) -> bool:
    ...
    if not re.match(r'^[a-zA-Z0-9_-]+$', group_id):
        raise GroupIdValidationError(group_id)
```

Provider-specific default (`helpers.py:68-75`) — `'_'` for FalkorDB, `''`
elsewhere. Queries filter with `WHERE e.group_id IN $group_ids`
(`edges.py:518`), and it is folded into the Lucene expression for fulltext
(`search_utils.py:95-104`).

### What is missing for real isolation

This is a **namespace, not a tenancy boundary**. For CRED specifically:

- **Flat.** A single opaque string. No hierarchy, so `org / team / project /
  repo` must be encoded by convention (`acme-platform-checkout`) with no
  structural support for "all of `acme`" or scoped inheritance.
- **No ACL anywhere.** No principal, role, permission, or grant concept exists in
  `graphiti_core`. Isolation is entirely "the caller passes the right
  `group_ids`". A caller that passes another tenant's `group_id` gets that
  tenant's data. There is no enforcement layer below the API.
- **Same physical graph by default.** All groups share one database unless you
  call `with_database()` (`driver.py:117-125`). Cross-tenant leakage is a missing
  `WHERE` clause away, and those clauses are hand-written into ~40 inline Cypher
  strings.
- **Not enforced on all paths.** Episode fulltext search ignores its filter
  argument (`search_utils.py:889`), community fulltext search takes no filter
  parameter (`search.py:795`).
- **No per-group config.** Ontology, model choice, and retention are per-call
  arguments, not per-group policy.
- **No cross-group federation.** You can query multiple `group_ids` at once, but
  there is no notion of a *shared* group with read-only visibility from
  children — which is exactly CRED's "governed shared knowledge" requirement.

`group_id` is adequate for per-user memory, which is Zep's product. It is not
adequate for organizational governance.

---

## 8. OSS project mechanics

### License and CLA

- **Apache-2.0**, verbatim 201-line text (`LICENSE`), declared at `pyproject.toml:11`.
  Copyright line is still the unfilled template `Copyright [yyyy] [name of copyright owner]`.
- **A CLA is mandatory for contributors** (`Zep-CLA.md`). It grants Zep Software,
  Inc. a "perpetual, worldwide, non-exclusive, no-charge, royalty-free,
  irrevocable copyright license … to sublicense and distribute Your
  Contributions". Contributors retain their own rights, but **Zep gains
  sublicensing rights — your upstream contributions can be relicensed into Zep's
  commercial product.** Enforced by `.github/workflows/cla.yml`.

**Implication for CRED:** Apache-2.0 means vendoring or forking is completely
free and the CLA does not bind you. The CLA only matters if you upstream.

### The OSS / Zep-cloud boundary is cleaner than the branding implies

There are **no zep-cloud API calls in `graphiti_core`**. No license-key check, no
SaaS-gated feature, no hosted-only code path. `server/graph_service/zep_graphiti.py`
defines `class ZepGraphiti(Graphiti)` but it is a plain subclass adding FastAPI
conveniences — "Zep" there is historical branding, not a boundary.

The one real coupling is **telemetry, on by default**
(`graphiti_core/telemetry/telemetry.py`):

- Fires on every `Graphiti()` construction (`graphiti.py:248`), event
  `graphiti_initialized`.
- Ships to PostHog (`https://us.i.posthog.com`, public key at `telemetry.py:18-19`).
- Payload (`graphiti.py:259-264`, `telemetry.py:106-111`): provider *class names*
  (llm/embedder/reranker/database), `graphiti_version`, `platform.machine()`.
  Anon UUID cached at `~/.cache/graphiti/telemetry_anon_id`.
- **No API keys, URIs, queries, or graph content are transmitted** — the full
  payload construction was traced.
- Opt-out: `GRAPHITI_TELEMETRY_ENABLED=false` (`telemetry.py:22,36`). Auto-disabled
  under pytest. Documented at `README.md:624-693`.
- But **`posthog>=3.0.0` is a hard runtime dependency** (`pyproject.toml:21`).
  The env var stops the calls; it does not remove the dep. `initialize_posthog()`
  returns `None` on `ImportError` (84-86), so force-uninstalling degrades gracefully.

### Repo layout — three independently versioned deliverables

| Path | What | Version |
|---|---|---|
| `graphiti_core/` | the library (22 subpackages) | 0.29.2 → PyPI `graphiti-core` |
| `server/` | FastAPI REST wrapper (`graph_service`, 16 files) | 0.1.0 → container |
| `mcp_server/` | MCP server (`src/graphiti_mcp_server.py`, 1288 LOC) | 1.0.2 |

`server/` and `mcp_server/` each have their **own `pyproject.toml` and `uv.lock`**
and consume `graphiti-core` as a *published* dependency
(`mcp_server/pyproject.toml:10` pins `graphiti-core[falkordb]>=0.29.2`). They are
not path-installed against local core. **Editing `graphiti_core/` does not affect
`mcp_server/` unless core is installed editable** — a real cross-boundary footgun.

Note `mcp_server` requires `openai>=2.41.0` while core requires `openai>=1.91.0`
— a major version ahead, a latent conflict in a shared env.

### MCP server — 13 tools

All `@mcp.tool()` in `mcp_server/src/graphiti_mcp_server.py`:

| # | Line | Tool | Purpose |
|---|---|---|---|
| 1 | 341 | `add_memory` | ingest an episode (13 params incl. `saga`, `group_id`, `custom_extraction_instructions`) |
| 2 | 477 | `search_nodes` | node search, `entity_types` + `center_node_uuid` filters |
| 3 | 554 | `search_memory_facts` | edge/fact search with `valid_at_after/before`, `invalid_at_after/before` |
| 4 | 634 | `delete_entity_edge` | delete a fact by uuid |
| 5 | 660 | `delete_episode` | delete an episode by uuid |
| 6 | 689 | `get_entity_edge` | fetch one fact |
| 7 | 716 | `get_episodes` | recent episodes for group(s) |
| 8 | 786 | `summarize_saga` | summarize an episode thread |
| 9 | 839 | `build_communities` | community detection |
| 10 | 893 | `add_triplet` | direct structured write, bypassing extraction |
| 11 | 967 | `get_episode_entities` | **provenance lookup** — entities/facts by episode uuid |
| 12 | 1003 | `clear_graph` | destructive wipe of group(s) |
| 13 | 1046 | `get_status` | DB ping |

Plus `@mcp.custom_route('/health')` at 1079.

**No `@mcp.resource()` and no `@mcp.prompt()` exist** — zero hits across
`mcp_server/src/`. Guidance is delivered as a static instruction blob
(`GRAPHITI_MCP_INSTRUCTIONS`, lines 130-168) passed to the `FastMCP` constructor
(171-174). Returns are `TypedDict`s (`models/response_types.py`), so tools emit
structured JSON schemas rather than free text — worth copying.

Tools 3 and 11 are the two most directly relevant to CRED:
`search_memory_facts` exposes temporal bounds to the agent, and
`get_episode_entities` exposes provenance.

Stale doc: `mcp_server/docs/cursor_rules.md:5` still tells agents to call
`search_facts`, which does not exist.

### MCP ingestion is fire-and-forget into a non-durable queue

`add_memory` does **not** await ingestion (`graphiti_mcp_server.py:449-470`) — it
enqueues and returns. `mcp_server/src/services/queue_service.py:15-47` keeps one
`asyncio.Queue` + one worker task per `group_id`:

```python
        if group_id not in self._episode_queues:
            self._episode_queues[group_id] = asyncio.Queue()
        await self._episode_queues[group_id].put(process_func)
        if not self._queue_workers.get(group_id, False):
            asyncio.create_task(self._process_episode_queue(group_id))
```

Consequences, all evidenced:
- **In-memory and non-durable.** Process restart silently loses every queued
  episode. No persistence, no dead-letter, no retry.
- **No backpressure** — `asyncio.Queue()` is unbounded.
- **No agent-visible status.** `get_queue_size()` (82) and `is_worker_running()`
  (88) exist but are **not exposed as tools**; `get_status` only pings the DB.
  An agent has no way to learn whether its write succeeded. Failures are
  logs-only (worker swallows exceptions at 67-70).
- The `create_task` result is not retained (45) — a GC hazard, with no drain/
  shutdown handler.

Serialization is per-group, so different `group_id`s process concurrently — which
is the correct shape, but the durability story is absent.

Transports (`1231-1272`): `stdio`, `sse` (deprecated), `http` streamable
(**default**, endpoint `/mcp/`). Config is YAML + `${VAR:default}` expansion +
env + CLI via pydantic-settings, precedence `CLI > env > yaml > dotenv`
(`config/schema.py:296-297`).

**MCP `SEMAPHORE_LIMIT` defaults to 10** (`graphiti_mcp_server.py:89`) while core
defaults to 20 (`helpers.py:38`) — same env var, two fallbacks. The MCP path
always passes `max_coroutines`, so core's 20 is shadowed.

**Security: no authorization boundary on `group_id`.** Any connected agent can
read, write, or `clear_graph` *any* group by naming it. `--destroy-graph`
(1155-1159) calls `clear_data(client.driver)` with no group filter at all.

### Tests

- Core: **33 `test_*.py`** files. `pytest.ini` defines one marker, `integration`,
  but **only 3 files actually carry `@pytest.mark.integration`** — so
  `-m "not integration"` is insufficient, which is why CI carries 10 explicit
  `--ignore` paths (`.github/workflows/unit_tests.yml`).
- Unit tier runs with `DISABLE_NEPTUNE/NEO4J/FALKORDB/KUZU=1` — no DB, no LLM.
- Integration tier spins **live FalkorDB + Neo4j 5.26** via `docker run --network host`.
- MCP live tier needs a **real OpenAI key** (`mcp-server-tests.yml`,
  `MODEL_NAME: gpt-4.1-mini`), with `cancel-in-progress` concurrency "so repeated
  pushes don't keep paying for real OpenAI calls."
- **No coverage gate anywhere**; no `pytest-cov` in any dep list.
- Of 15 MCP test files, only `test_live_falkordb_int.py` runs in CI. Treat the
  MCP test count as aspirational, not CI-enforced.

One genuinely good detail: both live workflows handle pytest exit code 5 (no
tests collected) — fork PRs without secrets self-skip and pass, but if
`OPENAI_API_KEY` *is* set and nothing collected, the job **fails loudly**. That
is a maintainer who has been burned by a vacuous green build.

### Community health

**Not measurable from this clone** — it is shallow (depth 1), so commit counts,
cadence, and contributor numbers are unobtainable. Structural signals instead:

- **15 GitHub workflows**, including CodeQL, lint, typecheck, three release
  pipelines, and four Claude-based automation workflows.
- **Supply-chain hygiene is above average**: every third-party action pinned to a
  full commit SHA with a version comment; PyPI release via **OIDC trusted
  publishing** with a tag-vs-`pyproject.toml` version assertion; CLA workflow
  mints a short-lived App token rather than a long-lived PAT.
- **`CONTRIBUTING.md` (238 lines) gates hard**: all new features and integrations
  require an **RFC issue before a PR** — explicitly naming new DB drivers and LLM
  clients — and **any PR over 500 LOC requires an RFC regardless**. PRs without a
  linked RFC get `needs-rfc` and will not be reviewed. Stated priority is that
  bug fixes get the fastest review.
- Three maintainers, all `@getzep.com` (`pyproject.toml:5-9`, matching the CLA
  allowlist). `README.md:32` is a hiring banner.

Read honestly: this is a **small maintainer team optimizing their own throughput
on a high-traffic commercial repo**. If CRED depends on this and needs an
upstream change, budget for the RFC cycle. Do not assume a drive-by PR lands.

### Version and dependency weight

`version = "0.29.2"` — **0.x, no SemVer stability guarantee**, on an active
commercial roadmap. Expect breaking changes between minors. `requires-python
>=3.10,<4`.

Seven required deps (`pyproject.toml:13-22`): `pydantic`, `neo4j`, `openai`,
`tenacity`, `numpy`, `python-dotenv`, `posthog`. **Three of those may be dead
weight** — `neo4j` and `openai` are unconditional even on a FalkorDB + Anthropic
stack, and `posthog` exists solely for telemetry. Twelve optional extras;
`kuzu` is explicitly marked for removal (`pyproject.toml:31`).

---

## Top 3 design decisions CRED should STEAL

### 1. LLM proposes, interval algebra disposes

`edge_operations.py:538-572` + `prompts/dedupe_edges.py:43-100`.

The LLM's only job is to **nominate** candidate contradictions as a list of
integer indices. It never writes a timestamp and never decides expiry. The actual
invalidation is deterministic interval arithmetic in plain Python, and the
returned indices are range-validated before use (`edge_operations.py:735-744`,
`762-770`).

Why this matters for CRED: it makes the temporal layer **testable, replayable,
and auditable without an LLM in the loop.** You can unit-test invalidation
exhaustively, and you can re-run it deterministically when a policy changes. Every
governance property CRED needs depends on the decision layer being code, not a
model. Copy this split verbatim, and go further — persist the *reason* for
invalidation (which edge superseded it) on the edge, which Graphiti does not do.

### 2. Deterministic dedup before LLM dedup

`utils/maintenance/dedup_helpers.py` — normalize → entropy gate → 3-gram shingles
→ blake2b MinHash → LSH bands → Jaccard at `_FUZZY_JACCARD_THRESHOLD = 0.9`
(line 34), with `_resolve_with_similarity` (220) resolving confident matches at
**zero LLM cost**. Only the ambiguous residue reaches `dedupe_nodes.nodes`.
Matching fast paths for edges at `edge_operations.py:345-357` and `688-698`.

The entropy gate (`_name_entropy` 52, `_has_high_entropy` 79) is the subtle part:
it *refuses* to fuzzy-match short or low-entropy names where shingles are
unreliable, falling back to the LLM instead. That is the correct failure
direction — cheap-and-confident or expensive-and-careful, never cheap-and-wrong.

For CRED, where identifiers are often structured (repo paths, service names,
ticket IDs, commit SHAs), a deterministic resolver should carry **far more** of
the load than it does here. This is the highest-leverage cost idea in the repo.

### 3. Episode as the immutable provenance anchor, with bidirectional links

`nodes.py:318-332` + `edges.py:267-270` + `edge_operations.py:52-96` +
`graphiti.py:1631-1643`.

Three links, not one: fact→episodes, episode→facts, episode→entities. Plus
`reference_time` on the edge (`edges.py:280-282`) separating "when the fact was
true" from "when the source claimed it", and `node_episode_index_map`
(`edge_operations.py:77-84`) giving precise per-episode attribution in batches
rather than blanket-linking.

Two amendments CRED should make. First, **drop the
`store_raw_episode_content=False` option** (`graphiti.py:1026-1028`) — an
execution-memory plane that discards its own sources has no provenance.
Second, **persist the producing model and prompt version on every derived fact.**
Graphiti threads `prompt_name` through for token accounting
(`token_tracker.py:33-44`) but never writes it to the graph, so you cannot
invalidate everything a known-bad model version produced. For a governed system
that is a required capability, and it is nearly free to add at write time.

Honourable mention: **per-prompt token accounting**
(`llm_client/token_tracker.py:33-44`) tracking `call_count` and input/output
tokens *by prompt name*. Copy this on day one — it is the instrument that makes
ingestion cost visible before it becomes a bill.

---

## Top 3 things CRED should NOT copy

### 1. LLM-call-per-fact ingestion economics

**≈ 3 + 2E + N LLM calls per episode**, with no per-episode ceiling
(§4). Every extracted edge costs a dedup call plus a timestamp call, each entity
costs an attribute call, and the fan-out is bounded only in parallelism
(`SEMAPHORE_LIMIT`, `helpers.py:38`), never in count. Episodes must also be
ingested **sequentially per group** (`graphiti.py:1055-1060`) because dedup and
invalidation read the live graph — a throughput ceiling, not merely a latency one.

For CRED this is disqualifying at the intended scale. A multi-repo dev-team
plane ingesting commits, PRs, CI runs, and agent traces would generate episodes
continuously, and the cost curve is linear in extracted facts with a large
constant. Structured sources (git metadata, CI results, ticket transitions) should
bypass extraction entirely via a typed write path — Graphiti's own `add_triplet`
(`graphiti.py:1645`) proves the shape exists; it is simply not the default. Budget
LLM extraction for genuinely unstructured input only, and cap calls per episode.

### 2. `group_id` as the tenancy boundary

A single flat validated-by-regex string (`helpers.py:136-157`) shared across one
physical graph, with **no principal, role, permission, or grant concept anywhere
in `graphiti_core`** (§7). Isolation is entirely "the caller passes the right
`group_ids`", enforced by hand-written `WHERE` clauses across ~40 inline Cypher
strings — and already missed on some paths (`search_utils.py:889` ignores its
filter; `search.py:795` accepts none).

The MCP surface makes it concrete: **any connected agent can `clear_graph` any
group by naming it**, and `--destroy-graph` wipes everything with no filter.

CRED's premise is *governed* shared knowledge across org/team/project. That needs
a real authorization model — hierarchical scopes, a principal on every read and
write, shared-read/private-write federation, and enforcement below the query
layer rather than in it. Retrofitting that onto a flat string is harder than
designing it in. Decide the tenancy model before the storage model.

### 3. The storage "abstraction"

116 provider conditionals outside the driver package, Neo4j Cypher as the implicit
`case _:` fallthrough in every builder, `neo4j` as a hard dependency regardless of
backend, Neo4j driver kwargs (`routing_`) leaking into core call sites, and three
generations of abstraction coexisting (`driver.py:97-99`) (§6).

Do not adopt this shape, and note what it costs: Graphiti cannot cheaply add a
non-Cypher backend, which means CRED inherits that constraint if it depends on
the library. Define your persistence boundary in terms of **domain operations**
("resolve entity", "invalidate fact", "search facts as of T"), not "execute this
query string" — the parameter name `cypher_query_` (`query_executor.py:41`) is the
whole diagnosis in one identifier.

The one thing to take from this section is the *Kuzu lesson*
(`kuzu_driver.py:51-54`): because facts need full-text indexing and their own
attributes and temporal bounds, Kuzu was forced to reify edges into
`RelatesToNode_` nodes. **A temporally-bounded, searchable, attributed fact is a
first-class object, not an edge.** Model it that way from the start and the
reification tax never appears.

---

## Recommendation

**FORK THE TEMPORAL MODEL AS A CONCEPT. DO NOT DEPEND ON GRAPHITI AS A LIBRARY.**

This is a firm recommendation, not a hedge.

**Why not depend.** The coupling risk is concentrated and not diversifiable:

- **You inherit Neo4j.** Not as a default — as a hard dependency
  (`pyproject.toml:15`) and as the implicit fallthrough in every query builder.
  CRED will want Postgres for the transactional/governance side, and adding a
  Postgres backend to Graphiti is not "writing a driver", it is implementing a
  second execution path (~2,500 LOC of ABCs plus recursive CTEs plus a `tsquery`
  sanitizer plus a datetime shim), because the abstraction boundary is a Cypher
  string executor.
- **You inherit the ingestion cost curve**, which is the single biggest threat to
  CRED's unit economics. Roughly `3 + 2E + N` LLM calls per episode is not tunable
  from outside — it is the pipeline's structure (`graphiti.py:1121-1178`), and the
  default retrieval recipe is *also* LLM-metered per query.
- **You inherit `group_id`** and would be building governance on a substrate that
  has no concept of a principal.
- **Version 0.x on a commercial roadmap** with three `@getzep.com` maintainers and
  an RFC-gated contribution process. Upstream changes CRED needs are slow by
  design, and the API is explicitly unstable. Graphiti's roadmap follows Zep's
  product needs — per-user agent memory — which diverges from CRED's
  multi-repo/multi-agent org plane. That divergence widens over time.
- The **entity model is not temporal at all** (`nodes.py:499-503`), so the headline
  reason to adopt only half-covers CRED's requirement.

**Why fork the concepts.** The genuinely hard, genuinely valuable intellectual
work here is small, well-isolated, and cheaply reimplemented:

| Steal | Source | Size |
|---|---|---|
| bi-temporal quad + `reference_time` | `edges.py:263-285` | ~20 lines of schema |
| interval-algebra invalidation | `edge_operations.py:538-572, 826-840` | ~60 lines |
| contradiction-nomination prompt + index validation | `prompts/dedupe_edges.py:43-100`, `edge_operations.py:735-770` | ~100 lines |
| MinHash/LSH dedup with entropy gate | `dedup_helpers.py` | ~300 lines |
| episode provenance triad | `nodes.py:318-332`, `edges.py:267-270`, `edge_operations.py:52-96` | ~50 lines |
| per-prompt token accounting | `token_tracker.py:33-44` | ~50 lines |
| structured MCP tool returns | `mcp_server/src/models/response_types.py` | 88 lines |

That is on the order of **600-700 lines of ideas** — a few weeks of work — against
a ~30k-line dependency that brings Neo4j, PostHog, an unbounded LLM budget, a
flat tenancy string, and an unstable 0.x API. **The leverage is in the model, not
the implementation.** Apache-2.0 means you can lift the specific algorithms
directly, with attribution, and no CLA obligation attaches unless you upstream.

**Suggested sequencing for the Spike 1 exit evidence:**

1. Implement the bi-temporal fact model **as a first-class node**, not an edge —
   take the Kuzu lesson (`kuzu_driver.py:51-54`) rather than the Neo4j shape. Extend
   bi-temporality to entities, which Graphiti does not do.
2. Port `resolve_edge_contradictions` and its interval logic, and add the
   provenance fields Graphiti lacks: producing model, prompt version, and the
   superseding fact's id on every invalidation.
3. Port `dedup_helpers.py` and measure what fraction of your *structured* dev-team
   identifiers resolve deterministically. Expect it to be much higher than
   Graphiti's conversational-text case — that ratio is your ingestion cost model,
   and it is the number that should decide the build.
4. Add a typed, extraction-free write path for structured sources (git, CI,
   tickets) as the **default**, with LLM extraction reserved for prose. Instrument
   both with per-prompt token accounting from day one.
5. Benchmark against Graphiti-on-FalkorDB as the control, using
   `tests/evals/eval_e2e_graph_building.py` as a starting harness.

Run Graphiti as a **reference implementation and benchmark control** — stand it up,
read it, measure against it. Do not put it in the dependency graph.
