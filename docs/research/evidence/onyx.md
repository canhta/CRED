# Onyx (formerly Danswer) — Evidence Review

Repo scanned: `/Users/canh/Solo/OSS/onyx` @ `d98bdb5` ("feat(connectors): Box connector (indexing + permission sync)")
Scope: source code, not docs. All citations are absolute paths + line numbers.

---

## Verdict

- **Onyx's ACL model is the single best-documented answer to "permissions must survive derivation" in OSS — and it is 100% behind the Enterprise License.** The MIT core ships a deliberately *crippled* stub (`_get_acl_for_user` returns `{user_email, PUBLIC}` only — `backend/onyx/access/access.py:114-127`); every group, external-group, and source-permission concept lives in `backend/ee/`. This is not an accident, it is the business model.
- **The core design worth stealing is three things: (1) a flattened, prefixed ACL string set (`user_email:`, `group:`, `external_group:`) computed at write time and stored as a filterable set field on every chunk; (2) Postgres as ACL source-of-truth with the index as a denormalized cache synced via a `last_modified` dirty bit; (3) a per-source `SyncConfig` plugin registry.** All three are ~200 lines of actual concept. Copy them.
- **Permission drift is the unsolved problem, and Onyx fails OPEN on it.** There is no TTL, no ACL expiry, and no "stale permissions ⇒ deny" path. A revoked Google Drive share stays queryable until the next *successful* sync — default cadence 5–30 min, unbounded if the sync job is broken. Stale-marked external group rows still grant access (`backend/ee/onyx/db/external_perm.py:219-227` does not filter `stale`). Budget for CRED to do better here; it is a real differentiator.
- **Do NOT copy Onyx's OSS/EE split.** Putting *permissions* behind the paywall means the free tier is unsafe for exactly the multi-team use case CRED targets, and it forced an ugly runtime-monkeypatch architecture (`fetch_versioned_implementation`, `backend/onyx/utils/variable_functionality.py:70-118`) that taxes every feature forever. Put security in the core; monetize admin/governance/scale.
- **CRED should not build a connector fleet.** Onyx has ~64 connector directories but only **12** sources with working permission sync (`backend/ee/onyx/external_permissions/sync_params.py:92-210`). Ingestion is commodity; permission *fidelity* is the moat. Build 3–5 deep, permission-correct connectors and integrate for the rest.

---

## 1. ACL / PERMISSION MODEL (the priority)

### 1.1 The data model

Two tiers: **Postgres is the source of truth, the search index holds a flattened denormalized copy.**

**Tier 1 — Postgres.** Three storage locations:

| What | Where | Notes |
|---|---|---|
| Per-document external permissions | `Document.external_user_emails`, `Document.external_user_group_ids`, `Document.is_public` (`backend/onyx/db/models.py:1100-1107`) | ARRAY columns. In the **MIT core schema** — the columns exist, but only EE code ever writes them. |
| User → external group membership | `User__ExternalUserGroupId` (`stale` flag, `cc_pair_id` scope) | Written by group-sync jobs |
| "This external group means everyone" | `PublicExternalUserGroup` | e.g. Drive domain-wide / anyone-with-link |
| Internal Onyx groups | `UserGroup` + assoc tables, EE-only logic (`backend/ee/onyx/db/user_group.py`) | |
| Folder/hierarchy permissions | `HierarchyNode` mirrors the same 3 fields (`backend/onyx/db/models.py:935-980`) | newer feature, same shape |

**Tier 2 — the index.** Everything collapses to one field:

```
field access_control_list type weightedset<string> {
    indexing: summary | attribute
    rank: filter
    attribute: fast-search
}
```
`backend/onyx/document_index/vespa/app_config/schemas/danswer_chunk.sd.jinja:159-163`; OpenSearch equivalent at `backend/onyx/document_index/opensearch/schema.py:46,171`.

### 1.2 The key abstraction: `ExternalAccess` → prefixed ACL strings

`backend/onyx/access/models.py` — the whole conceptual core, and it is small.

`ExternalAccess` (line 8-62) is what every connector's permission-sync function must produce:
```python
external_user_emails: set[str]
external_user_group_ids: set[str]
is_public: bool
```
Plus two named constructors that encode the security posture: `.public()` (line 41) and `.empty()` (line 49), whose docstring states the fail-closed intent explicitly:

> "This is especially helpful to use when you are performing permission-syncing, and some document's permissions aren't able to be determined (for whatever reason). Setting its `ExternalAccess` to 'private' is a feasible fallback."

`DocumentAccess.to_acl()` (`backend/onyx/access/models.py:167-192`) flattens everything into one `set[str]`:

```python
for user_email in self.user_emails:       acl_set.add(prefix_user_email(user_email))
for group_name in self.user_groups:       acl_set.add(prefix_user_group(group_name))
for e in self.external_user_emails:       acl_set.add(prefix_user_email(e))
for g in self.external_user_group_ids:    acl_set.add(prefix_external_group(g))
if self.is_public:                        acl_set.add(PUBLIC_DOC_PAT)
```

Prefixes (`backend/onyx/access/utils.py:4-27`) exist purely to prevent namespace collisions between emails, Onyx groups, and external groups. Note line 22-27 — external groups are *additionally* prefixed by source, because "engineering" in Slack ≠ "engineering" in Confluence:
```python
def build_ext_group_name_for_onyx(ext_group_name, source):
    return f"{source.value}_{ext_group_name}".lower()
```

**This is the whole trick.** Both the document ACL and the user ACL are reduced to sets of opaque strings, and authorization becomes *set intersection*, pushed into the search engine as a filter. The invariant is stated in the code (`backend/onyx/access/models.py:169-171`):

> "NOTE: When querying for documents, the supplied ACL filter strings must be formatted in the same way as this function."

That invariant is enforced only by convention — a shared helper module, not a type. It's the fragile seam of the design.

### 1.3 Query-time enforcement

The user's side of the intersection, EE version (`backend/ee/onyx/access/access.py:183-210`):
```python
def _get_acl_for_user(user, db_session) -> set[str]:
    db_user_groups     = fetch_user_groups_for_user(db_session, user.id)
    db_external_groups = fetch_external_groups_for_user(db_session, user.id)
    user_acl = {prefix_user_group(g.name) for g in db_user_groups} \
             | {prefix_external_group(g.external_user_group_id) for g in db_external_groups}
    user_acl.update(get_acl_for_user_without_groups(user, db_session))  # + email, + PUBLIC
    return user_acl
```

Path: `build_access_filters_for_user` (`backend/onyx/context/search/preprocessing/access_filters.py:8-10`) → `IndexFilters.access_control_list` (`backend/onyx/context/search/models.py:149-153`) → backend query builder.

The decision point is `_build_index_filters` (`backend/onyx/context/search/pipeline.py:100-110`):
```python
if bypass_acl:
    user_acl_filters = None
elif acl_filters is not None:
    user_acl_filters = acl_filters
else:
    if db_session is None:
        raise ValueError("Either db_session or acl_filters must be provided")
    user_acl_filters = build_access_filters_for_user(user, db_session)
```

`access_control_list=None` means **no ACL filter at all**. Security therefore rests on `bypass_acl` being false. Grep of non-test callers shows it defaults `False` everywhere and is set `True` only for system callers; the Slack bot path explicitly passes `bypass_acl=False` (`backend/onyx/onyxbot/slack/handlers/handle_regular_answer.py:308`) and the search API likewise (`backend/onyx/server/features/search/api.py:167`). The flag is documented "Use with caution!" (`backend/onyx/context/search/models.py:172-173`). It is a single boolean standing between a chat request and the entire corpus.

There is one genuinely good hardening here (`backend/onyx/context/search/pipeline.py:54-77`): if the caller supplies `document_set` names, the pipeline re-validates each against the user's access and raises `INSUFFICIENT_PERMISSIONS`, explicitly closing "the API-layer bypass where a user could override the persona's configured document sets with arbitrary names." Steal that pattern — filter parameters supplied by a caller must themselves be authorized, not just the results.

### 1.4 Backend filter construction — and a fail-open asymmetry

**OpenSearch (the active backend)** — `backend/onyx/document_index/opensearch/search.py:933-974`:
```python
acl_visibility_filter = {"bool": {
    "should": [{"term": {PUBLIC_FIELD_NAME: {"value": True}}}],
    "minimum_should_match": 1}}
if access_control_list:
    acl_visibility_filter["bool"]["should"].append(
        {"terms": {ACCESS_CONTROL_LIST_FIELD_NAME: list(access_control_list)}})
```
An **empty** list still yields `public == true` only. Fail-closed. Their own test asserts exactly this (`backend/tests/external_dependency_unit/opensearch/test_opensearch_client.py:2047-2059`: *"Explicitly pass in an empty list to enforce private doc filtering"* → 3 of 5 docs).

**Vespa (deprecated)** — `backend/onyx/document_index/vespa/shared_utils/vespa_request_builders.py:45-60,194-199`:
```python
if not key or not vals:
    return ""          # <-- empty list => NO filter emitted
...
if filters.access_control_list is not None:
    _append(filter_parts, _build_weighted_set_filter(ACCESS_CONTROL_LIST, ...))
```
An **empty** ACL list produces an empty clause, which is dropped — i.e. *no ACL filter at all*. The `is not None` guard is defeated by `[]`. In practice `_get_acl_for_user` always returns at least `PUBLIC_DOC_PAT` so the list is never empty, but the safety property depends on a caller invariant rather than the filter builder. **Lesson for CRED: the empty-ACL case must deny, structurally, at the lowest layer.** OpenSearch's "always include the public-only clause, then OR in the user's grants" shape is the correct construction.

Scale note worth stealing: they hit Vespa HTTP 400s from OR-chaining ACL terms and moved to `weightedSet` (comment at `vespa_request_builders.py:191-193`); OpenSearch uses a `terms` query with a hard cap `MAX_NUM_TERMS_ALLOWED_IN_TERMS_QUERY` that **raises** rather than truncates (`search.py:959-964`). Truncating an ACL list would silently grant or deny; raising is right. `ExternalAccess.MAX_NUM_ENTRIES = 5000` (`backend/onyx/access/models.py:12-14`) caps the other direction — though the comment admits "not internally enforced ... the caller can check this."

Combination semantics are documented in-repo — a genuinely good artifact to imitate: `backend/onyx/document_index/FILTER_SEMANTICS.md:9-30`. ACL is `OR within, AND with rest`.

### 1.5 The sync jobs

Registry: `backend/ee/onyx/external_permissions/sync_params.py:56-210`. Each source registers a `SyncConfig` with up to three independent capabilities:
```python
class SyncConfig(BaseModel):
    doc_sync_config: DocSyncConfig | None = None      # per-document ACLs
    group_sync_config: GroupSyncConfig | None = None  # group membership
    censoring_config: CensoringConfig | None = None   # post-query filtering
```
This three-way split is the most transferable structural idea in the repo, because different SaaS APIs expose permissions in fundamentally different ways:
- **doc_sync** — source can enumerate per-object ACLs (Drive, Confluence, Slack, GitHub, Jira, SharePoint, Teams, Box, Canvas, Gmail)
- **group_sync** — source can enumerate group rosters, with `group_sync_is_cc_pair_agnostic` marking sources where groups are org-wide rather than per-credential (Confluence: `True`, line 114; Drive: `False`, line 102)
- **censoring** — source *cannot* be pre-indexed with ACLs, so documents are indexed as public and filtered **after** retrieval (Salesforce only, line 181-185)

`initial_index_should_sync` (line 59) handles the bootstrap ordering problem: for sources where the indexing run itself captures permissions, doc-sync is suppressed until one successful index exists (`backend/ee/onyx/background/celery/tasks/doc_permission_syncing/tasks.py:159-165`).

**Cadence = the staleness window.** From `backend/ee/onyx/configs/app_configs.py`: default 5 min (line 10-11); Confluence/Jira/SharePoint/Box/Canvas doc-sync 30 min (lines 24, 40, 53, 68, 119); Drive/GitHub/Slack/Teams 5 min. All env-overridable, plus a global runtime multiplier (`tasks.py:174`). Trigger condition, `tasks.py:139-180`: `access_type == AccessType.SYNC`, cc_pair ACTIVE, and `now >= last_time_perm_sync + frequency`.

`AccessType` (`backend/onyx/db/enums.py:244-247`) is the per-connector switch: `PUBLIC` / `PRIVATE` / `SYNC`. Only `SYNC` does live permission mirroring.

**Group sync uses mark-and-sweep** (`backend/ee/onyx/background/celery/tasks/external_group_syncing/tasks.py:544,551,594-606,641`):
1. `remove_stale_external_groups` (pre-clean)
2. `mark_old_external_groups_as_stale` — set `stale=True` on all rows for the cc_pair, **and commit immediately**
3. `upsert_external_groups` — `INSERT ... ON CONFLICT DO UPDATE SET stale=False`
4. `remove_stale_external_groups` — delete whatever is still `stale=True`

The immediate commit at step 2 has an excellent comment explaining why (`external_perm.py:81-86`): otherwise the connection sits idle-in-transaction during long Drive API calls and gets killed by `idle_in_transaction_session_timeout`, failing the whole sync. That is a real operational lesson.

**ACL → index propagation is a dirty bit, not a push.** `upsert_document_external_perms` (`backend/ee/onyx/db/document.py:53-103`) does three notable things:
- **Replace, not union** — docstring line 61-62: *"this will replace any existing external access, it will not do a union."* Correct: a union would make revocation impossible.
- **Permissions can be stored before the document exists** (lines 79-88) — so a doc indexed later inherits already-known ACLs, and *"the upsert function in the indexing pipeline does not overwrite the permissions fields."* This solves an ordering hazard CRED will absolutely hit.
- **Change detection** (lines 90-101): only if the email set, group set, or `is_public` differs does it write and set `document.last_modified = now()`. A separate task (`backend/onyx/background/celery/tasks/vespa/document_sync.py`) picks up modified docs and pushes the new `access_control_list` to the index.

So the write path is: source API → `ExternalAccess` → Postgres (+dirty bit) → async index update. **Two async hops between a revocation in Google Drive and enforcement at query time.**

### 1.6 Drift, staleness, and failure modes — the weak spot

Blunt assessment: **Onyx fails open on permission drift.**

1. **No TTL on ACLs.** A grep for ACL expiry across `backend/onyx` and `backend/ee` returns nothing — `last_time_perm_sync` is used *only* to decide whether to schedule the next sync, never to invalidate or downgrade access. If the sync job is broken, credentials expire, or the connector is paused, documents remain queryable with permissions from the last successful run, indefinitely. There is no "permissions older than N ⇒ deny" circuit breaker.
2. **Stale-marked group rows still grant access.** `fetch_external_groups_for_user` (`backend/ee/onyx/db/external_perm.py:219-227`) selects on `user_id` alone — no `stale.is_(False)` predicate. Between step 2 and step 4 of the mark-and-sweep, and permanently if the job crashes in between, `stale=True` rows continue to appear in the user's ACL. This is arguably deliberate (avoids access flapping mid-sync) but it means a crashed group sync leaves over-broad access with no alarm.
3. **Two async hops** (§1.5) mean revocation latency is *sync cadence + index propagation*, i.e. minutes at best.
4. **Censoring sources are indexed as public.** Salesforce docs get `is_public` effectively true so they're retrievable by anyone, with permission applied post-retrieval (`backend/ee/onyx/access/access.py:81-108`, `is_only_censored` → `is_public_anywhere`). If the post-filter is bypassed anywhere, the data is simply exposed.

Where they *do* fail closed, and it's good:
- Censoring exceptions drop the chunks (`backend/ee/onyx/external_permissions/post_query_censoring.py:68-76`): *"Failed to censor chunks ... so throwing out all chunks for this source and continuing."*
- Anonymous users are excluded from all censoring-enabled sources outright (lines 46-48).
- Un-indexed documents get least-privilege (`backend/onyx/access/access.py:93-99`): *"in those cases the document does not exist and so we use least permissive."*
- `ExternalAccess.empty()` as the documented fallback when permissions are indeterminate.
- Documents with no resolvable source are **skipped**, not defaulted (`backend/ee/onyx/access/access.py:75-78`).

Net: the *per-document* logic is thoughtfully fail-closed; the *temporal* dimension is fail-open. CRED's opportunity is precisely there — a freshness SLA on derived permissions, with degradation to deny.

### 1.7 The EE/MIT seam inside the ACL system — precisely

This is the clearest thing in the repo and worth stating exactly:

| Component | MIT core | EE |
|---|---|---|
| `ExternalAccess` / `DocumentAccess` / `to_acl()` | ✅ `backend/onyx/access/models.py` | — |
| Prefix helpers | ✅ `backend/onyx/access/utils.py` | — |
| `access_control_list` index field + query filter | ✅ `document_index/**` | — |
| DB columns for external perms | ✅ `backend/onyx/db/models.py:1100-1107` | — |
| `_get_acl_for_user` | ⚠️ **stub**: `{user_email, PUBLIC}` only (`access.py:114-127`) | ✅ real impl w/ groups (`ee/.../access.py:183-210`) |
| `_get_access_for_documents` | ⚠️ **stub**: *"MIT version will wipe all groups and external groups on update"* (`access.py:86`) | ✅ (`ee/.../access.py:43-118`) |
| User groups | ❌ | ✅ `ee/onyx/db/user_group.py` |
| All source permission sync | ❌ | ✅ `ee/onyx/external_permissions/**` (43 files) |
| Sync Celery tasks | ❌ | ✅ `ee/onyx/background/celery/tasks/{doc_permission_syncing,external_group_syncing}` (1937 LOC) |
| Post-query censoring | ❌ | ✅ `ee/onyx/external_permissions/post_query_censoring.py` |

**Enforcement is in the core; the ability to know anything worth enforcing is in EE.** The MIT build can express only "mine or public." That is a coherent monetization line — and a bad one for a security property, see §9.

**The switch mechanism** (`backend/onyx/utils/variable_functionality.py:70-118`) is a runtime module-path swap:
```python
module_full = f"ee.{module}" if is_ee else module
try:    return getattr(importlib.import_module(module_full), attribute)
except ModuleNotFoundError as e:
    if is_ee:
        if "ee.onyx" not in str(e): raise e
        return getattr(importlib.import_module(module), attribute)  # fall back to MIT
    raise
```
`lru_cache`d, driven by `global_version.set_ee()` (lines 20-33), enabled by `ENABLE_PAID_ENTERPRISE_EDITION_FEATURES` **or** `LICENSE_ENFORCEMENT_ENABLED` (default `"true"`, line 41-43). Companion helpers: `fetch_versioned_implementation_with_fallback` (line 125) and `fetch_ee_implementation_or_noop` — the latter used at `backend/onyx/access/access.py:137-146` so the MIT build silently answers "no" to *"does this source sync permissions during indexing?"*

The convention: MIT defines `_foo` (private stub) and public `foo()` that resolves the versioned `_foo`; EE defines `_foo` at the mirrored `ee.` path, with `# NOTE: is imported ... by fetch_versioned_implementation / DO NOT REMOVE` comments (e.g. `ee/onyx/access/access.py:143-144,189-190`) because static analysis can't see the reference. The cleanest specimen is the folder-permission pair: MIT returns `[]` (`backend/onyx/access/hierarchy_access.py:7-11`), EE returns real groups (`backend/ee/onyx/access/hierarchy_access.py:7-11`).

**Cost of this pattern:** every EE-extensible call site pays an indirection with no type checking, no IDE navigation, and a fallback path that silently degrades security-relevant behavior. It also means EE source ships in the same tree and image as MIT — the paywall is a runtime flag, not a build artifact.

---

## 2. DOCUMENT SET / ORG MODEL

All ORM models live in **one 6,655-line file**: `backend/onyx/db/models.py`. There is no `ee/onyx/db/models.py` — EE tables (`UserGroup`, `License`, `ScimToken`, association tables) are declared in the *core* file. **EE-ness is enforced at the logic layer, not the schema layer.** Consequence worth internalizing: a MIT deployment has the full multi-tenant/group schema present and writable; the rows are simply ignored by the ACL computation.

Key entities: `User` (`:318`), `Connector` (`:1913`), `Credential` (`:1980`), `ConnectorCredentialPair` (`:813`, composite PK on `(connector_id, credential_id)`), `Document` (`:1029`), `DocumentSet` (`:3628`), `Persona` (`:3851`), `UserGroup` (`:4776`), `HierarchyNode` (`:929`), `UserFile` (`:5140`).

**Two-phase commit on group/set membership.** `document_set__connector_credential_pair` (`:699`) and `user_group__connector_credential_pair` (`:4662`) both carry `is_current` **inside the primary key** (`:711`, `:4672`). An edit writes new rows with `is_current=True` while old rows persist until the index ACL-sync job completes and flips `DocumentSet.is_up_to_date` (`:3638`) / `UserGroup.is_up_to_date` (`:4783`). This is the correct answer to "how do you change a group without a window where the index disagrees with Postgres," and CRED will need exactly this.

**Persona (assistant) is a content scope, not a security boundary.** Four parallel attachments: `document_sets` (`:3926`), `hierarchy_nodes` (`:3999`), `attached_documents` (`:4005`), `user_files` (`:3988`), plus a `search_start_date` floor (`:3893`). These are snapshotted into a plain pydantic `PersonaSearchInfo` (`backend/onyx/context/search/models.py:481-491`) before the DB session closes, explicitly so "SearchTool and search_pipeline never lazy-load relationships post-commit." The persona time floor is never loosened — `max(caller_start, persona_time_cutoff)` (`backend/onyx/context/search/pipeline.py:85-100`).

The separation is the important lesson: **persona restricts what is searched; the user ACL restricts what may be returned; the two are computed independently and ANDed.** A misconfigured assistant cannot leak, because the ACL filter is not derived from the assistant.

Persona ownership is enforced by a real DB constraint — `CheckConstraint("user_id IS NULL OR owner_group_id IS NULL", name="ck_persona_single_owner")` (`:4008-4011`) — unusual rigor for this codebase and worth copying.

**Two overlapping authorization systems coexist**, which is a warning:
- Legacy `UserRole` enum (`backend/onyx/auth/schemas.py:11-36`): `LIMITED, BASIC, ADMIN, CURATOR, GLOBAL_CURATOR, SLACK_USER, EXT_PERM_USER`, enforced via FastAPI deps (`backend/onyx/auth/users.py:2092-2129`) *plus* ad-hoc curator predicates re-implemented per query in at least five files (`backend/onyx/db/connector_credential_pair.py:174`, `backend/onyx/db/skill.py:130`, `backend/onyx/db/feedback.py:90`, `backend/ee/onyx/db/user_group.py:234,358,594`).
- Newer token-based `Permission` enum (`backend/onyx/db/enums.py:540-591`) granted per group via `permission_grant` (`models.py:4615`) and cached on `User.effective_permissions` JSONB (`:386`), with a `@validates` guard preventing role-implied tokens from being stored (`models.py:4653-4659`).

Onyx is mid-migration between the two and paying for it. **Pick one model on day 1.**

## 3. INDEXING & RETRIEVAL

**Backend: OpenSearch is now the active index; Vespa is deprecated.** Stated flatly in `backend/onyx/document_index/FILTER_SEMANTICS.md:1-5` ("Describes the active **OpenSearch** backend. The deprecated **Vespa** backend differs..."), prod compose runs `opensearch` (`deployment/docker_compose/docker-compose.prod.yml:243`), and there is a live migration task family (`backend/onyx/background/celery/tasks/opensearch_migration/`, queue at `backend/onyx/configs/constants.py:465`). Both sit behind a `DocumentIndex` interface (`backend/onyx/document_index/interfaces_new.py`) chosen by `factory.py`.

**Read that migration as a signal.** Onyx spent years on Vespa — a genuinely superior ranking engine — and is migrating off it. Operational burden and hiring pool beat ranking sophistication. CRED should start on OpenSearch/Postgres-pgvector and never touch Vespa.

**Hybrid search.** `backend/onyx/context/search/retrieval/search_runner.py:52-74`. Alpha defaults to `HYBRID_ALPHA` (0.5); `alpha <= 0.2` ⇒ `QueryType.KEYWORD`. Vespa YQL is a 4-way OR of dense-body ANN, dense-title ANN, sparse `weakAnd`, and a `content_summary` match (`backend/onyx/document_index/vespa/vespa_document_index.py:906-912`), with an embedding-dimension-specific rank profile (`:916`).

Sharp edge: **on the Vespa path `hybrid_alpha` is quantized to exactly `{0.2, 0.5}`** — `search_runner.py:66` collapses the float into a two-valued enum and `vespa_document_index.py:930-934` re-expands it. Passing `0.35` behaves identically to `0.5`. A tunable that isn't.

**Reranking is effectively dead code.** `RerankingModel` survives (`backend/onyx/natural_language_processing/search_nlp_models.py:1207`) but its only in-repo caller is marked `# No longer used` (`:1423`); the model-server reranker is fully commented out (`backend/model_server/legacy/reranker.py:11-48`); the DB columns were dropped by migration `backend/alembic/versions/78ebc66946a0_remove_reranking_from_search_settings.py`. Quality now rests on (a) Vespa/OpenSearch native global-phase reranking (`danswer_chunk.sd.jinja:249,318`, `rerank-count: 1000`) and (b) weighted RRF over a query fan-out. **A well-funded team concluded a cross-encoder rerank stage wasn't worth its latency and ops cost.** Take that data point seriously before building one.

**What replaced it — query fan-out + weighted RRF.** `backend/onyx/tools/tool_implementations/search/search_tool.py:593` does an LLM call producing a semantic query, keyword queries, and a scope decision; queries are weighted (`backend/onyx/tools/tool_implementations/search/constants.py:5-13` — semantic 1.3, keyword 1.0, non-custom 0.7, original 0.5) and fused via `weighted_reciprocal_rank_fusion` (`search_utils.py:28-41`, `k=50`). The candid comment at `constants.py:5`: *"Taking an opinionated stance on the weights, no chance users can do a good job customizing this."* Correct instinct — do not expose these.

Nice UX detail: expanded queries are streamed to the client *before* execution (`search_tool.py:953-960`) purely for perceived latency.

**Citations.** `DynamicCitationProcessor` (`backend/onyx/chat/citation_processor.py:70`) with three modes (`:27-48`): `HYPERLINK` (rewrite `[1]` → `[[1]](url)` + emit `CitationInfo`), `KEEP_MARKERS` (research agent, renumbered later), `REMOVE` (public Slack/Discord bots — **so document URLs cannot leak into a public channel**; that's an ACL-adjacent design decision worth stealing). Numbering is assigned by the *search tool* when it returns docs (`backend/onyx/chat/citation_utils.py:10-50`), not by the LLM. The processor buffers partial tokens, handles unicode bracket variants, and skips code blocks (`:59`). Frontend hides unresolvable citations rather than rendering broken ones (`web/src/app/app/message/MemoizedTextComponents.tsx:63-84`).

## 4. MULTI-TENANCY

**Schema-per-tenant in Postgres, via SQLAlchemy `schema_translate_map` — not row-level security.**

`backend/onyx/db/engine/sql_engine.py:444-472`:
```python
if not is_valid_schema_name(tenant_id):
    raise HTTPException(status_code=400, detail="Invalid tenant ID")
if not MULTI_TENANT and tenant_id == POSTGRES_DEFAULT_SCHEMA_STANDARD_VALUE:
    session = Session(bind=engine, expire_on_commit=False)   # fast path, no translation
...
schema_translate_map = {None: tenant_id}
```
`{None: tenant_id}` = "every model without an explicit schema resolves to `<tenant_id>`." Control-plane tables opt out explicitly (`UserTenantMapping.__table_args__ = ({"schema": "public"},)`, `models.py:5202`). Two Alembic trees: `backend/alembic/` (per-tenant) and `backend/alembic_tenants/` (the `public` control tables).

**Tenant resolution at request time** (`backend/ee/onyx/server/middleware/tenant_tracking.py:114-192`), in order: API-key/PAT (tenant base64-embedded in the key itself), Redis session token from the `fastapiusersauth` cookie (or bearer, for mobile), anonymous-user JWT cookie, explicit tenant cookie, default. **Not subdomain-based.** Every candidate passes `is_valid_schema_name` (`:143,158,191`) — the SQL-injection guard, since the schema name gets interpolated into `SET search_path` during migrations.

Two details worth stealing outright:
- **Contextvar reset discipline in Celery.** `TenantAwareTask.__call__` (`backend/onyx/background/celery/apps/app_base.py:100-118`) reads `tenant_id` from task **kwargs**, sets the contextvar, and resets it in `finally` — *"so it does not leak into any subsequent tasks on the same worker process."* Plus a belt-and-braces `@task_postrun.connect reset_tenant_id` (`:641-651`). Prefork workers are long-lived and reused; a leaked tenant contextvar is a cross-tenant data breach. This is the single highest-severity multi-tenancy bug class and Onyx defends it twice.
- **The `finally`-block hazard** at `tenant_tracking.py:185-188`: a `return` inside `finally` would swallow a propagating auth exception and fall back to the *wrong tenant*. Guarded by `if sys.exc_info()[0] is None`.

**Tenant provisioning uses a pre-warmed pool** (`backend/ee/onyx/server/tenants/provisioning.py:70-129`): `AvailableTenant` rows (`models.py:5213`) store their own `alembic_version` so pool members can be migrated forward before assignment; on migration failure the tenant is rolled back rather than orphaned (`:103-116`). Signup latency is a schema-create + migrate otherwise — this is how you avoid it.

Multi-tenancy *plumbing* is core (contextvar at `backend/shared_configs/contextvars.py:7-12`, session machinery, tenant tables, Celery propagation); *orchestration* (resolution middleware, provisioning, billing) is EE.

## 5. INTERFACES

~80 FastAPI routers in `backend/onyx/main.py:511-587` (chat, query, connectors, personas, tools, MCP, LLM/embedding providers, bots, admin), plus ~20 EE routers (`backend/ee/onyx/main.py:129-167`). A genuine safety net: **`check_router_auth(application)`** — every route must have auth or be explicitly marked public, with an EE mirror `check_ee_router_auth` (`ee/onyx/main.py:171`). Copy this. It converts "someone forgot `Depends(current_user)`" from a breach into a startup failure.

**MCP, both directions, both MIT:**
- **As a server**: FastMCP-based, separate ASGI process (`backend/onyx/mcp_server/api.py:37-52`, `backend/onyx/mcp_server_main.py`). Tools: `search_indexed_documents`, `search_web`, `open_urls` (`backend/onyx/mcp_server/tools/search.py:103,241,301`). Resources: indexed sources, document sets. **Auth reuses the normal stack** — `OnyxTokenVerifier` (`backend/onyx/mcp_server/auth.py:18-51`) validates a bearer token by proxying `GET /me` to the API server. Crucially this means **MCP queries are ACL-filtered like any other query**; there is no privileged MCP path.
- **As a client**: `backend/onyx/server/features/mcp/` with a dedicated `ssrf.py` guard for user-supplied MCP server URLs, and an `MCPTool` adapter (`backend/onyx/tools/tool_implementations/mcp/mcp_tool.py`).

**Chat API**: `POST /chat/send-chat-message` (`backend/onyx/server/query_and_chat/chat_backend.py:541-560`), SSE when `stream=true`, else JSON. ~50 packet types in a discriminated union on `type` (`backend/onyx/server/query_and_chat/streaming_models.py:69-400+`), including resumable streams (`GET /chat/chat-session/{id}/resume-stream`, `:1033`). The packet taxonomy is well-factored and worth reading before designing CRED's agent stream.

**The agent-identity gap — important for CRED.** API keys are backed by a **synthetic shadow user** with email domain `onyxapikey.ai` (`backend/onyx/configs/constants.py:108`, `backend/onyx/db/api_key.py:33-38`) carrying its own `UserRole` and default group memberships (`backend/onyx/db/api_key.py:170-179`). So an agent calling Onyx queries with *the API key's* ACL, not the ACL of the human it is acting for. A grep for on-behalf-of/impersonation on the query path returns nothing — impersonation exists only on the *ingestion* side (service accounts reading as users: `backend/onyx/connectors/google_utils/google_auth.py:85`, `backend/onyx/connectors/box/connector.py:238`). **There is no delegation model.** For CRED, where the whole point is agents acting inside an org's permission structure, this is the gap to fill: an agent token must carry the *acting human's* identity, and the ACL must be computed from that.

## 6. OSS PROJECT MECHANICS

**License.** Root `LICENSE`: MIT Expat everywhere except three directories — `backend/ee/`, `web/src/ee/`, `web/src/app/ee/`. Those are under the **Onyx Enterprise License** (`backend/ee/LICENSE`), whose operative clauses are:

> "may only be used **in production**, if you ... have agreed to, and are in compliance with, the Onyx Subscription Terms of Service ... and otherwise have a valid Onyx Enterprise License for the correct number of user seats."
> "**Notwithstanding the foregoing, you may copy and modify the Software for development and testing purposes, without requiring a subscription.**"
> "it is **forbidden to copy, merge, publish, distribute, sublicense, and/or sell the Software**."
> "DanswerAI ... **retain all right, title and interest in and to all such modifications and/or patches**"

Source-available, dev/test free, seat-metered in production, and your EE patches belong to them.

**Scale of the split** — smaller than the reputation suggests:

| | Total | EE | EE share |
|---|---|---|---|
| Python under `backend/onyx` + `backend/ee` | 1,332 | **203** | 6.9% |
| TS/TSX under `web/src` | 1,298 | **53** | 4.1% |
| Connector sources | ~55 | — | — |
| Sources with permission sync | — | **12** | — |

**What's EE**: all permission/group sync (45 files, the largest EE area), user groups, SCIM 2.0 (Okta/Entra), query history + analytics, usage reporting, billing, license/tier enforcement, standard answers, chat retention TTL, whitelabeling, custom analytics JS, outbound webhooks, token rate limits, multi-tenant/cloud provisioning (19 files), connector OAuth admin flows, evals, query expansion.

**What's NOT EE (and this is a change from the folklore):** **SAML and OIDC SSO are in the MIT core** — `backend/onyx/server/saml_multi.py:41`, `backend/onyx/server/oidc_multi.py:59`, admin at `backend/onyx/server/manage/sso/api.py:29` (verified directly). Only a SAML *session* table helper remains in `backend/ee/onyx/db/saml.py`. Also MIT: full chat/RAG, ~55 connectors, MCP server + client, API keys, PATs, ACL *enforcement*, Slack/Discord bots.

**Enforcement is runtime, not build-time.** EE source ships in the same Docker image (`backend/Dockerfile:157` `COPY ./ee /app/ee`; `:168` copies the license public key; the legal notice is a `LABEL` at `:61-65`). `LICENSE_ENFORCEMENT_ENABLED` defaults to `"true"` (`backend/onyx/utils/variable_functionality.py:39-43`), so **the stock image loads EE code by default** and gates features via two middlewares:
- `license_enforcement.py` — license is base64(JSON) + detached RSA-4096 PSS/SHA-256 signature verified against `backend/keys/license_public_key.pem` (`backend/ee/onyx/utils/license.py:49-104`; note the care at `:68-73` to re-serialize the *original* dict with `sort_keys=True`, since Pydantic re-serialization would break the signature). Status ladder `ACTIVE → PAYMENT_REMINDER → GRACE_PERIOD (30d, notify only) → GATED_ACCESS (hard 402)`, with an allowlist so you can always log in and pay (`backend/ee/onyx/configs/license_enforcement_config.py:37-73`). **Redis failure fails open** — deliberate.
- `tier_gate.py` — `COMMUNITY | BUSINESS | ENTERPRISE` per path prefix, longest-prefix-wins (`license_enforcement_config.py:76-95`).

**CLA: none for MIT, full assignment for EE.** `CONTRIBUTING.md:55-57` requires signing `contributor_ip_assignment/EE_Contributor_IP_Assignment_Agreement.md` for anything under an `ee/` directory. That agreement includes copyright assignment (§3.1), attorney-in-fact appointment (§3.2), moral-rights waiver (§4), and patent assignment (§5.1) — explicitly scoped to EE only (§7). `CONTRIBUTING.md:45` also requires a GitHub issue with upvotes and maintainer approval before feature contributions.

**Deployment footprint: heavy.** `deployment/docker_compose/docker-compose.prod.yml` = **13 services**: api_server, background (supervisord), web_server, postgres, **two** model servers (inference + indexing), opensearch, nginx, certbot, minio, redis, code-interpreter — plus 10 named volumes. Background alone runs **8 Celery worker processes + beat** across **~23 queues** (`backend/supervisord.conf:30-137`, `backend/onyx/configs/constants.py:420-465`). Realistic floor ≥16 GB RAM. A dev-only stack (`docker-compose.dev.yml`) trims to 7. There are 14 compose variants plus Helm, Terraform, and ECS Fargate targets.

**Project health**: 44 GitHub workflows including `zizmor.yml` (Actions security lint), Playwright/Jest/pytest/integration/helm suites, and `pr-linear-check.yml` (Linear issue linkage required). Active, professionalized, and clearly run as a company repo with a public mirror (`sync_foss.yml`).

## 7. CONNECTOR ARCHITECTURE & INDEXING INTERNALS

### 7.1 The connector interface — 12 variants, one abstract method

`backend/onyx/connectors/interfaces.py` (344 lines). `BaseConnector(abc.ABC, Generic[CT])` (`:48-119`) has exactly **one** abstract method — `load_credentials` (`:54-56`). Everything else is a capability mixin you opt into:

| Variant | Line | Abstract method |
|---|---|---|
| `LoadConnector` | 123 | `load_from_state() -> GenerateDocumentsOutput` |
| `PollConnector` | 130 | `poll_source(start, end)` |
| `SlimConnector` | 139 | `retrieve_all_slim_docs(start, end, callback)` |
| `SlimConnectorWithPermSync` | 152 | `retrieve_all_slim_docs_perm_sync(...)` |
| `OAuthConnector` | 163 | `oauth_id`, `oauth_authorization_url`, `oauth_code_to_token` |
| `CredentialsConnector` | 245 | `set_credentials_provider(...)` |
| `EventConnector` | 258 | `handle_event(event)` |
| `CheckpointedConnector[CT]` | 271 | `load_from_checkpoint`, `build_dummy_checkpoint`, `validate_checkpoint_json` |
| `CheckpointedConnectorWithPermSync[CT]` | 310 | + `load_from_checkpoint_with_perm_sync` |
| `Resolver` | 321 | `reindex(errors, include_permissions)` — retry failed docs |
| `HierarchyConnector` | 337 | `load_hierarchy(start, end)` |

The composition is real, not theoretical: `GoogleDriveConnector` implements **four** (`backend/onyx/connectors/google_drive/connector.py:256-260`).

**`validate_perm_sync` (`:84-94`) is marked "do not override"** — it dispatches to the EE implementation via `fetch_ee_implementation_or_noop`. So the permission contract is enforced centrally, not per connector. Good pattern.

### 7.2 Boilerplate: ~9 registration touchpoints across two languages

Measured by grepping every non-connector file mentioning `bitbucket`:

**Backend (4):** `DocumentSource` enum entry (`backend/onyx/configs/constants.py:273`), description string (`:716`), `CONNECTOR_CLASS_MAP` entry (`backend/onyx/connectors/registry.py:215-218`), Slack icon (`backend/onyx/onyxbot/slack/icons.py:61`).
**Frontend (4):** `ValidSources` (`web/src/lib/types.ts:601`), config schema (`web/src/lib/connectors/connectors.tsx:381+`), credential shape (`web/src/lib/connectors/credentials.ts`), display metadata (`web/src/lib/sources.ts:414-418`).
**Tests (1):** CI secret name (`backend/tests/utils/secret_names.py`).
**Plus, for permission sync:** a `_SOURCE_TO_SYNC_CONFIG` entry and a `doc_sync.py` / `group_sync.py` module under `backend/ee/onyx/external_permissions/<source>/`.

**There is no plugin or entrypoint mechanism** — every registration is a hardcoded dict entry, and the registry stores strings not classes for lazy import (`registry.py:8-10`). Adding a connector requires touching the core repo.

Actual connector sizes: Bitbucket 666 lines (smallest recent), Slack 2,335, Google Drive **4,652 across 6 files**. **Realistic range ~700–4,700 lines per connector, before permission sync.** This is the number that should decide CRED's connector strategy.

### 7.3 Checkpointing

`ConnectorCheckpoint` (`backend/onyx/connectors/models.py:519-530`) is astonishingly thin — just `has_more: bool`. Connectors subclass it with whatever state they need (`SlackCheckpoint` at `backend/onyx/connectors/slack/connector.py:80-92` carries the channel list, a per-channel cursor map, and seen thread timestamps).

Persistence (`backend/onyx/background/indexing/checkpointing_utils.py`): the checkpoint is `model_dump_json()`'d into the **file store** and only a pointer column (`IndexAttempt.checkpoint_pointer`) lands in Postgres (`:30-51`). Load re-validates through the connector's own `validate_checkpoint_json` (`:62-65`). Hard **200 MB** ceiling (`:211-217`) — with an acknowledged TODO that this should be disk-based (`models.py:520`).

Two robustness ideas worth stealing:
- **Anti-thrash guard** (`checkpointing_utils.py:88-104`): if all 50 most recent attempts made zero progress, discard the checkpoint and start clean. Prevents a poisoned checkpoint from permanently wedging a connector.
- **Checkpoint tracks delivery, not completion.** `backend/onyx/background/indexing/run_docfetching.py:810-812`: *"checkpointing is used to track which batches have been sent to the filestore, NOT which batches have been fully indexed."* On exception, batches are deliberately **not** cleaned up (`:838-844`) so the next run reuses them. Fetch and process are decoupled with the file store as the durable seam.

### 7.4 Three sync modes — the distinction CRED must copy

1. **Indexing** — full content → chunk → embed → write.
2. **Pruning** — *deletion reconciliation only*. Pulls all IDs from the source, deletes local IDs missing from that list (`backend/onyx/background/celery/tasks/pruning/tasks.py:474-483`). **This is what slim connectors exist for.** Without one, `extract_ids_from_runnable_connector` (`backend/onyx/background/celery/celery_utils.py:176-194`) falls back to `load_from_state()` — a full content re-download just to enumerate IDs. Critical safety property at `:109-138`: `ConnectorFailure` IDs are folded into the seen-set so *"failed-to-retrieve documents are not accidentally pruned."* Easy to miss, catastrophic to get wrong.
3. **Permission sync** — orthogonal to both (§1.5).

Plus a fourth hybrid, **permissions-during-indexing** (`run_docfetching.py:468-475`), used only for the first index of a `SYNC` cc_pair; doc_sync takes over afterwards.

Poll windows (`run_docfetching.py:477-526`) subtract a `POLL_CONNECTOR_OFFSET` for deliberate re-fetch overlap, and pin `window_end` across resumes because otherwise *"new slack channels could be missed (since existing slack channels are cached as part of the checkpoint)."*

### 7.5 Indexing pipeline

`backend/onyx/indexing/indexing_pipeline.py:1256-1513` (`index_doc_batch`): filter → ingestion hook → DB prepare/dedup → image processing → **chunk** (`:1340`) → contextual RAG (`:1345`) → **embed** (`:1366`) → row lock (`:1395`) → ACL/document-set enrichment (`:1396`) → chunk-count diff (`:1403`) → **write to all indices** (`:1429`) → verify → persist content hashes (`:1481`).

Note the ordering discipline: the row lock is taken only at `:1395`, *"Not needed until here, since this is when the actual race condition with vector db can occur."* And content hashes persist only **after** confirmed writes — *"prevents a failed index from storing a hash that would permanently skip the document on the next sync."*

**Chunker** (`backend/onyx/indexing/chunker.py`) uses `chonkie.SentenceChunker` with `CHUNK_OVERLAP = 0` and a candid comment (`:26-28`): *"it is unclear if overlaps actually help quality at all."* Token budget: `content_token_limit = chunk_token_limit - title_tokens - metadata_tokens - context_size` (`:250-252`) with three graceful degradation steps.

**Five enrichment tricks** worth knowing:
1. **Title prefix + metadata suffix**, with *two* renderings from one source — natural-language `"Metadata:\n\tkey - value"` for the embedding, bare space-joined values for BM25 (`chunker.py:42-72`).
2. **Separate title embedding**, deduped across the batch (`embedder.py:160-183`), with a `skip_title` bool because "no title" ≠ "empty title embedding."
3. **Mini-chunks**, stored in the *same* Vespa mapped tensor as the parent so ranking maxes over them — *"No minichunk documents in vespa"* (`indexing_utils.py:150-160`).
4. **Large chunks** — every `LARGE_CHUNK_RATIO` consecutive chunks merged, with `source_links` offsets rebased (`chunker.py:75-122`).
5. **Contextual RAG (Anthropic-style)** with **prompt caching** — `cacheable_prefix=CONTEXTUAL_RAG_PROMPT1(document)`, `suffix=...(chunk)` so the document context is cached across all chunks of that doc (`indexing_pipeline.py:946-951`). Errors degrade to `chunk_context = ""` rather than failing the batch.

**Dedup is two-gate** (`indexing_pipeline.py:328-396`): a timestamp gate, then a content-hash gate applied only when the timestamp did *not* advance. The precedence rule is the subtle bit (`:348-351`) — a timestamp advance is authoritative and overrides the hash, because Google Drive in-place image replacement leaves `image_file_id` unchanged while the bytes differ. `content_hash()` (`backend/onyx/connectors/models.py:305-336`) deliberately runs *before* image summarization since *"LLM-generated image summaries are non-deterministic."*

**Chunk shrinking**: `IndexingMetadata.doc_id_to_chunk_cnt_diff` (`interfaces_new.py:109-128`) lets the index delete only the "tail" chunks when a document gets shorter, instead of delete-all-then-reindex.

### 7.6 In-index multi-tenancy: a `tenant_id` field, not per-tenant indices

Confirmed four ways: the schema field is conditional (`danswer_chunk.sd.jinja:10-16`, `opensearch/schema.py:572-573`), the write path sets it (`indexing_utils.py:226-228`), the read path filters on it (`vespa_request_builders.py:26-27,186-188`), and the chunk UUID seed appends the tenant id (`document_index_utils.py:135-158`) — with a stern warning that changing that function without a migration breaks deletes and updates.

**Serious finding: the tenant filter has no fail-closed guard.** `vespa_request_builders.py:186-188` applies it only `if filters.tenant_id and MULTI_TENANT`, with an unresolved in-source `# TODO: add error condition if MULTI_TENANT and no tenant_id filter is set`. A missing tenant id silently yields **cross-tenant results**. Meanwhile the newer `IndexRetrievalFilters` (`interfaces_new.py:172-189`) *does* fail closed — defaulting `access_control_list` to `{PUBLIC_DOC_PAT}` — but is marked *"Currently unused."* They wrote the safe version and haven't wired it up. **For CRED: make the safe default the only constructor.**

Also note the two backends model "public" differently — Vespa folds it into the ACL weightedset as a sentinel; OpenSearch hoists it to a dedicated boolean *"such a broad and critical filter that it is its own field"* (`opensearch/schema.py:471-475`). Any filter code must branch on backend, and only a pair of comments keeps them in sync. This is exactly the kind of drift a single-backend decision avoids.

---

## 8. DIRECT ANSWERS

### 8.1 The OSS-vs-Enterprise boundary Onyx chose — and whether CRED should copy it

**Precisely what they chose.** MIT covers the entire *mechanism*: retrieval, chat, ~55 connectors, indexing, the `access_control_list` index field, the query-time ACL filter, MCP server and client, SAML/OIDC SSO, API keys, bots. The Enterprise License covers the *multi-user governance layer*: user groups, all external permission sync, SCIM, query history and analytics, billing, whitelabeling, webhooks, rate limits, and cloud multi-tenancy. It is 203 of 1,332 backend Python files (6.9%) and 53 of 1,298 frontend files (4.1%).

The line is drawn at a specific place: **"one user" is free; "many users with different permissions" is paid.** The MIT `_get_acl_for_user` returns `{your email, public}` — a perfectly good single-player product, and structurally incapable of being a multi-team one.

Enforcement is runtime, not build-time: EE source ships in the stock image (`backend/Dockerfile:157`), `LICENSE_ENFORCEMENT_ENABLED` defaults true, and two middlewares gate by signed license and tier. Contributions to `ee/` require full copyright and patent assignment; MIT contributions require nothing.

**Evaluation.** It is a commercially coherent line and it has clearly worked for them. It is nonetheless the wrong line for CRED, for three reasons:

1. **It makes the free tier unsafe for the target use case.** CRED's premise is organizational context shared across a *team*. A free tier that cannot express "this belongs to the platform group" isn't a smaller version of the product — it's a different product. Onyx can get away with this because single-user RAG is genuinely useful standalone. Execution memory for one developer is much less so.
2. **Security features behind a paywall invites the worst failure mode.** Someone runs the MIT build, sees `UserGroup` tables in the schema, populates them through an admin UI, and gets **no** access enforcement from them (`backend/onyx/access/access.py:84` hardcodes `user_groups=[]`). The tables exist and are inert. That is a footgun that will eventually become a CVE-shaped conversation.
3. **It cost them an architecture.** `fetch_versioned_implementation` (`backend/onyx/utils/variable_functionality.py:70-118`) is a runtime module-path swap with `lru_cache`, no type checking, no IDE navigation, `# DO NOT REMOVE` comments guarding invisible references, and a silent MIT fallback on `ModuleNotFoundError` **in the security path**. Every future EE-extensible feature pays this tax forever.

**Firm recommendation: do not copy it. Use this split instead.**

- **Open core (permissive — Apache 2.0, matching CRED's existing license):** everything a single team needs to run safely on their own hardware. That explicitly includes **the full permission model** — groups, external permission sync, query-time enforcement, freshness/staleness handling. Also: ingestion, indexing, retrieval, the agent/MCP API, self-hosted SSO.
- **Commercial:** the things that scale with organizational size and are painful to self-build — hosted/cloud multi-tenancy, SCIM and directory sync at enterprise scale, audit logging and compliance reporting, analytics and usage dashboards, SLA-backed support, and cross-org/enterprise administration.

The test: *if a feature's absence makes the free product insecure, it must be in the core.* Onyx fails this test; CRED should pass it. Monetize administration and scale, never authorization.

One thing to copy exactly: **the directory-scoped license** (`ee/` subtrees with a `LICENSE` at each root, declared in the root `LICENSE`). It is unambiguous, greppable, and tool-friendly. If CRED ever adds commercial code to the same repo, use this mechanism — just draw the line somewhere else.

### 8.2 Top 3 things to STEAL

**1. The flattened prefixed-ACL-string model — the entire §1.2 concept.**
`backend/onyx/access/models.py` + `backend/onyx/access/utils.py`, ~270 lines total.
Reduce both sides of the authorization question to sets of opaque namespaced strings (`user_email:`, `group:`, `external_group:`, `PUBLIC`), materialize the document's set into a filterable index field at write time, compute the user's set at query time, and let the search engine do the intersection. Prefix external groups by source to avoid cross-system collisions (`utils.py:22-27`).

Why it's the right answer: authorization becomes a pushed-down filter rather than a post-retrieval scan, so it costs nothing at scale, cannot be forgotten by a caller who remembers to search but forgets to check, and works identically for chunks, documents, and folders. This solves CRED's stated problem — permissions surviving into derived knowledge — by making the derived artifact carry the flattened permission with it.

Steal with **two fixes**: (a) make the empty-ACL case deny structurally at the filter builder (copy OpenSearch's "always emit the public-only clause, then OR in grants" shape at `backend/onyx/document_index/opensearch/search.py:933-974`, not Vespa's droppable clause); (b) wire up the fail-closed-by-default filter object they wrote and never used (`backend/onyx/document_index/interfaces_new.py:172-189`).

**2. The `SyncConfig` plugin registry with its three-way capability split.**
`backend/ee/onyx/external_permissions/sync_params.py:56-210`.
`doc_sync` (source can enumerate per-object ACLs) / `group_sync` (source can enumerate group rosters, with a `is_cc_pair_agnostic` flag) / `censoring` (source can't precompute, so filter after retrieval). Plus `initial_index_should_sync` to sequence the bootstrap.

This taxonomy is hard-won and non-obvious. It falls directly out of the fact that Google Drive, Confluence, and Salesforce expose permissions in structurally different ways, and it will be true for CRED's sources too. Copy the shape before writing a single integration.

**3. Postgres-as-truth + index-as-cache, reconciled by a change-detecting dirty bit.**
`backend/ee/onyx/db/document.py:53-103`.
Three properties in fifty lines: **replace-not-union** semantics (a union makes revocation impossible); **permissions storable before the document exists**, so an out-of-order index inherits already-known ACLs; and **change detection** — only a genuine diff in the email set, group set, or public flag sets `last_modified` and triggers reindex. Pair it with the `is_current`-in-primary-key two-phase membership swap (`backend/onyx/db/models.py:711,4672`) so group edits never leave a window where index and database disagree.

*Honorable mentions:* `check_router_auth` at startup (`backend/onyx/main.py`) — every route must have auth or be explicitly public, or the app won't boot. The double tenant-contextvar reset in Celery (`backend/onyx/background/celery/apps/app_base.py:100-118,641-651`). Re-authorizing caller-supplied *filter parameters*, not just results (`backend/onyx/context/search/pipeline.py:54-77`). And the in-repo `FILTER_SEMANTICS.md` — write that document on day one.

### 8.3 Top 3 things to AVOID

**1. Fail-open permission staleness.** No TTL, no expiry, no circuit breaker. `last_time_perm_sync` decides only *when to sync next*, never *whether to still trust*. A broken sync job means indefinitely stale grants, silently. Stale-marked group rows still resolve (`backend/ee/onyx/db/external_perm.py:219-227`). CRED should ship a **permission freshness SLA** from v1: every ACL carries `synced_at`; past a configurable horizon, access degrades to deny (or to owner-only) and the UI surfaces it. This is both the right security posture and a genuine differentiator — Onyx's own users cannot currently answer "how stale might my permissions be?"

**2. The runtime module-swap plugin mechanism.** `fetch_versioned_implementation` — string-prefix module swap, `lru_cache`d, untyped, invisible to static analysis, guarded by `# DO NOT REMOVE` comments, with a **silent fallback to the less-restrictive implementation** when an EE import fails (`variable_functionality.py:105-114`). In a security path, a `ModuleNotFoundError` should be a crash, not a downgrade. If CRED ever needs pluggable implementations, use explicit registration with typed protocols and fail loudly on a missing binding.

**3. Two authorization systems at once, and a 13-service deployment.** Two separate problems, one root cause — accreting rather than deciding.
- Onyx runs the legacy `UserRole` enum with curator scoping re-implemented in at least five query files *alongside* a newer `Permission`-token system (`backend/onyx/db/enums.py:540-591`). Nobody can now state the authorization rules from one file. **Pick one authorization model on day one and centralize it.**
- Production needs 13 containers, 8 Celery workers, ~23 queues, three datastores, MinIO, and two separate model servers (`deployment/docker_compose/docker-compose.prod.yml`, `backend/supervisord.conf:30-137`). ≥16 GB RAM floor. For a self-host-first product this is an adoption tax measured in abandoned installs. **CRED's self-hosted target should be Postgres + one app process + one worker**, with everything else opt-in. Also avoid running two search backends at once mid-migration — Onyx currently dual-writes Vespa and OpenSearch with Vespa declared authoritative on conflict (`backend/onyx/document_index/factory.py:142-145`), and the two model "public" differently.

### 8.4 Should CRED build connectors, or integrate?

**Firm answer: build a small number of deep, permission-correct connectors yourself; do not build a connector fleet; do not depend on Onyx or Airweave for ingestion.**

The evidence:

- **Volume ingestion is commodity and near-worthless.** ~55 connectors exist in MIT-licensed Onyx today. Airweave, Nango, Fivetran, and a dozen others cover the same ground. Nobody will choose CRED for its Notion connector.
- **Permission fidelity is scarce and is the actual moat.** Of ~55 Onyx sources, **12** have permission sync (`backend/ee/onyx/external_permissions/sync_params.py:92-210`) — and all 12 are behind the Enterprise License. A well-funded, focused team reached 12 in several years. That ratio is the whole strategic picture: the hard part isn't getting the bytes, it's getting the ACLs, and almost nobody has done it.
- **The cost is real but bounded.** ~700–4,700 lines per connector (Bitbucket 666, Slack 2,335, Drive 4,652), plus ~9 registration touchpoints, plus a separate `doc_sync`/`group_sync` module per source. Five deep connectors is a quarter or two of focused work — expensive, not prohibitive.
- **Integrating for ingestion doesn't solve CRED's problem.** Generic ingestion platforms hand you content, not permissions — and CRED's stated thesis is that *permissions are lost when source material becomes derived knowledge*. Outsourcing ingestion outsources exactly the layer where the value is created. You would be integrating away your differentiator and keeping the commodity.
- **Depending on Onyx specifically is a trap.** The permission sync you'd want is the Enterprise-licensed part — production use is seat-metered and you may not redistribute it. You would be building a product whose core capability sits under a competitor's commercial license.

**Concretely:** build first-party, permission-native connectors for GitHub, Slack, Google Drive, Linear/Jira, and Notion — the five sources where an AI dev team's organizational context actually lives. Design the `ExternalAccess` + `SyncConfig` abstraction (§8.2) first, so each connector is an implementation of a stable contract rather than bespoke code. Then expose a **documented ingestion API that accepts content plus an explicit permission descriptor**, so third parties and generic ETL tools can push into CRED — as long as they supply ACLs. That inverts the dependency: others do the long tail, CRED owns the permission contract everything must satisfy.

The one thing worth taking from Onyx directly is the *shape* of the abstraction (`backend/onyx/connectors/interfaces.py`, `backend/onyx/access/models.py`), which is MIT-licensed and excellent. Take the design; write the code.
