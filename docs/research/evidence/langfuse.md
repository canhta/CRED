# Langfuse — Evidence Review for CRED

**Repo:** `/Users/canh/Solo/OSS/langfuse` @ `1cb1bbc` (shallow clone, depth 1)
**Version:** `3.221.1` (`package.json:3`)
**License:** MIT core + commercial `ee/` (narrow)
**Date of review:** 2026-07-20

---

## Verdict

- **Langfuse has nothing above the trace.** The object graph tops out at `trace` (an LLM invocation tree). `TraceSession` (`packages/shared/prisma/schema.prisma:417-430`) has exactly five fields — `id, projectId, bookmarked, public, environment` — no name, no input/output, no status, no outcome, no metadata. `dataset_runs` is the only real grouping and it is offline-experiment-scoped. **The entity CRED sells does not exist here and cannot be retrofitted cheaply.**
- **Recommendation: (c) build your own outcome store, plus (b) emit OTLP for the span tier. Never (a).** Own a small Postgres outcome/knowledge-version plane; do *not* rebuild a span store. Emit OTLP with a reserved `cred.*` attribute namespace into whatever the user already runs, and correlate on `trace_id`. Embedding Langfuse means shipping 6 containers, ClickHouse, and ~37 BullMQ queues to run a product whose root entity is the wrong shape.
- **Prompt versioning is the single best thing in this repo and is a near-exact template for harness versioning.** Immutable `(projectId, name, version)` rows, a mutable `labels text[]` pointer, and — critically — telemetry pins the *resolved integer version*, never the label. Steal this wholesale (§4).
- **Licensing is genuinely permissive and not a blocker.** Tracing, OTLP ingest, prompts, datasets, evals, scores, annotation queues, the 83-route REST API, and a ~115-file MCP server are all MIT and ungated. The self-hosted paid delta is 7 entitlements at `self-hosted:enterprise`. `self-hosted:pro` is entitlement-identical to `oss`.
- **OTLP support is deep and real, and it accepts CRED-controlled semantics today** — including `langfuse.observation.prompt.version` and a full `langfuse.experiment.*` namespace that creates dataset runs straight from span attributes (`packages/shared/src/server/otel/attributes.ts:1-54`). Option (b) is viable *now* with zero Langfuse code changes.

---

## 1. Trace Data Model

### The tables

Traces, observations, and scores live **only in ClickHouse**. Postgres retains them as `LegacyPrisma*` models (`schema.prisma:432, 463, 539`) — dead weight kept for migration.

`packages/shared/clickhouse/migrations/unclustered/0001_traces.up.sql`:
```sql
CREATE TABLE traces (
    `id` String, `timestamp` DateTime64(3), `name` String,
    `user_id` Nullable(String),
    `metadata` Map(LowCardinality(String), String),
    `release` Nullable(String),          -- deploy identifier
    `version` Nullable(String),          -- free-form app version
    `project_id` String, `public` Bool, `bookmarked` Bool,
    `tags` Array(String),
    `input` Nullable(String) CODEC(ZSTD(3)),
    `output` Nullable(String) CODEC(ZSTD(3)),
    `session_id` Nullable(String),
    `event_ts` DateTime64(3), `is_deleted` UInt8
) ENGINE = ReplacingMergeTree(event_ts, is_deleted)
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, toDate(timestamp), id);
```

`0002_observations.up.sql` adds the generation-specific columns:
`type`, `parent_observation_id`, `start_time`/`end_time`, `level`, `status_message`, `version`,
`provided_model_name`, `internal_model_id`, `model_parameters`,
`provided_usage_details`/`usage_details` `Map(LowCardinality(String), UInt64)`,
`provided_cost_details`/`cost_details` `Map(LowCardinality(String), Decimal64(12))`, `total_cost`,
`completion_start_time`, and **`prompt_id` / `prompt_name` / `prompt_version UInt16`**.

Later migrations add: `environment` (0008), score `session_id` (0012) and `dataset_run_id` (0017), observation `tool_definitions`/`tool_calls`/`tool_call_names` (0033), `usage_pricing_tier_id/name` (0031), ingestion attribution (0035).

### Observation taxonomy is richer than "LLM call"

`packages/shared/src/domain/observations.ts:5-30` — `SPAN, GENERATION, EVENT, AGENT, TOOL, CHAIN, RETRIEVER, EVALUATOR, EMBEDDING, GUARDRAIL`. Agent-shaped work is already first-class.

### How custom metadata and versions attach

- **`version`** (`Nullable(String)`) on **both** traces and observations, and **`release`** on traces. Both are first-class filterable columns (`packages/shared/src/tableDefinitions/tracesTable.ts:59-72`). This is the built-in "which code version" hook.
- **`metadata`** is `Map(LowCardinality(String), String)` — **flat, string-valued**. Nested objects are flattened to dotted paths and JSON-stringified leaves (`packages/shared/src/server/otel/utils.ts:12-48`, `flattenJsonToPathArrays`). Bloom-filter indexes exist on both keys and values. Filterable as `stringObject`.
- **`tags`** `Array(String)` on traces only.

### What is absent — be precise

| Concept CRED needs | Present? |
|---|---|
| Task / work item | **No** |
| Outcome (merged, reverted, rolled back, accepted) | **No** |
| Knowledge/context package version → outcome link | **No** |
| Anything above trace with its own metadata | **No** — `TraceSession` has 5 fields, no metadata, no name, no status |
| Multi-trace rollup entity | Only `dataset_runs`, offline-experiment-scoped |
| Session as a stored table | **No** — sessions are derived at query time by grouping traces on `session_id`; there is no `sessions` ClickHouse table in any of the 36 migrations |

Sessions *do* aggregate cost/tokens/scores across traces at query time (`packages/shared/src/tableDefinitions/sessionsView.ts`), so they are a usable weak grouping — but they carry no payload of their own. There is no place to record "this task succeeded" that isn't a `score` attached to a trace.

---

## 2. OTel Compatibility

**Deep. This is the most reusable part of Langfuse for CRED.**

### Endpoint

`web/src/pages/api/public/otel/v1/traces/index.ts` — a real OTLP/HTTP endpoint:
- Binary protobuf (`application/x-protobuf`, decoded via generated `ExportTraceServiceRequest`, L83-92) **and** OTLP-JSON (L99-107)
- gzip via `Content-Encoding` (L55-67)
- Basic auth with `pk-lf-…`/`sk-lf-…` (`createAuthedProjectAPIRoute`)
- Warns above 16 MB bodies (L113-127)
- Also `/otel/v1/metrics` (19 lines — effectively a stub)

Spec'd in Fern at `fern/apis/server/definition/opentelemetry.yml`, declaring conformance to OTLP/HTTP.

The handler does **no mapping** — it uploads the raw `resourceSpans` batch to S3 and enqueues (`OtelIngestionProcessor.publishToOtelIngestionQueue`, `packages/shared/src/server/otel/OtelIngestionProcessor.ts:206-244`). All translation happens in the worker.

### The mapping layer

`packages/shared/src/server/otel/OtelIngestionProcessor.ts` — **3,279 lines**, plus a 513-line `ObservationTypeMapper.ts`. It is a priority-ordered registry (`ObservationTypeMapperRegistry`, `ObservationTypeMapper.ts:165+`) covering, in priority order:

| Priority | Source | Key |
|---|---|---|
| 0 | Python SDK ≤3.3.0 bug workaround | — |
| 1 | Langfuse native | `langfuse.observation.type` |
| 2 | **OpenInference** | `openinference.span.kind` |
| 3 | **OTel GenAI semconv** | `gen_ai.operation.name` → `chat/text_completion/generate_content/embeddings/invoke_agent/create_agent/execute_tool` |
| 4 | Genkit | `genkit:metadata:subtype` |
| 5–6 | Vercel AI SDK | `ai.*` operation prefixes |

GenAI semconv support is current, not stale — there is an explicit `text_completion` mapping annotated `per spec as of 2025-12-04`. Input/output extraction handles `gen_ai.input.messages`/`gen_ai.output.messages`, `gen_ai.system_instructions`, `gen_ai.tool.call.arguments/result` (L2007-2033), legacy `gen_ai.prompt.*`/`gen_ai.completion.*` (L1944-1963), OpenInference `llm.input_messages` (L1965-1990), span events (`gen_ai.system.message`, `gen_ai.choice`, `gen_ai.client.inference.operation.details`), TraceLoop, Pydantic-AI/Logfire.

### Can CRED emit OTLP and have it land correctly? Yes.

`packages/shared/src/server/otel/attributes.ts` is a **reserved vendor attribute namespace** CRED can write directly:

```ts
TRACE_NAME = "langfuse.trace.name"
TRACE_USER_ID = "user.id"
TRACE_SESSION_ID = "session.id"
TRACE_METADATA = "langfuse.trace.metadata"        // dotted subkeys supported
OBSERVATION_TYPE = "langfuse.observation.type"
OBSERVATION_PROMPT_NAME = "langfuse.observation.prompt.name"
OBSERVATION_PROMPT_VERSION = "langfuse.observation.prompt.version"
RELEASE = "langfuse.release"
VERSION = "langfuse.version"
AS_ROOT = "langfuse.internal.as_root"
EXPERIMENT_ID / _NAME / _DATASET_ID / _ITEM_ID / _ITEM_VERSION / _ITEM_EXPECTED_OUTPUT / _METADATA
```

Three consequences worth acting on:

1. **Harness version rides over OTLP today.** Setting `langfuse.observation.prompt.name` + `.version` lands in the `prompt_name`/`prompt_version` ClickHouse columns and immediately participates in the per-version metrics query (§4).
2. **`langfuse.internal.as_root=true` lets CRED control trace boundaries** (`OtelIngestionProcessor.ts:819-825`) instead of inheriting OTel's parentless-span rule. OTel `trace_id` maps 1:1 to Langfuse trace id (L800).
3. **The whole `langfuse.experiment.*` namespace creates dataset runs from span attributes alone** (`extractExperimentFields`, L2884-2966). CRED can emit a "task run" as an experiment over OTLP with no server changes. `EXPERIMENT_ITEM_VERSION` must be an ISO timestamp — it's a pointer to a dataset-item `valid_from`, and `"v1"`/`"latest"` are explicitly dropped (L2969-2984).

**Caveat:** these are *Langfuse-specific* attributes. Emitting them to Phoenix/Braintrust yields no semantics. Portable option (b) means emitting standard `gen_ai.*` + a `cred.*` namespace and accepting that the outcome layer stays in CRED.

---

## 3. Ingestion & Scale Architecture

### Path

```
POST /api/public/ingestion   (4.5 MB body cap, always returns HTTP 207)
  → auth (Basic pk/sk) → rate limit (FAILS OPEN) → loose outer zod validation
  → group events by (entityType, body.id)
  → upload each group as ONE JSON blob to S3/MinIO   [BLOCKING; any failure → whole request throws]
  → enqueue 1 BullMQ job per entity; payload is a POINTER, not the data
  → 207 Multi-Status {successes, errors} per event id
                          ↓ Redis / BullMQ (sharded by `${projectId}-${eventBodyId}`)
worker:
  → Redis "recently-processed" dedup cache (5 min TTL)
  → optional redirect to SecondaryIngestionQueue (S3 SlowDown flag / per-project allowlist)
  → S3 listFiles(prefix) + download (concurrency 50), or single GET when `skipS3List`
  → IngestionService.mergeAndWrite()
      → read current row from ClickHouse: ORDER BY event_ts DESC LIMIT 1 BY id, project_id
      → merge CH row (baseline) + S3 events in sorted order, immutable keys protected
      → event_ts = now()  → model match → tokenization → cost calc
  → ClickhouseWriter: in-memory batch, flush every 1000 ms or 1000 rows, JSONEachRow
  → ReplacingMergeTree(event_ts, is_deleted) collapses versions on merge
```

Key files: `web/src/pages/api/public/ingestion.ts`, `packages/shared/src/server/ingestion/processEventBatch.ts`, `worker/src/queues/ingestionQueue.ts`, `worker/src/services/IngestionService/index.ts` (1,898 lines), `worker/src/services/ClickhouseWriter/index.ts`.

**Why S3 before the queue** (`processEventBatch.ts:279-332`): Redis payloads stay tiny so BullMQ never becomes the bottleneck; blob storage *is* the durable event log for replay/retention/deletion; and multiple partial updates to one observation accumulate as separate files under one prefix, merged once by the worker. Events are grouped by `${entityType}-${body.id}` — *"reduces infra interactions per event"* (L203-216). The bucket is a hard dependency: `LANGFUSE_S3_EVENT_UPLOAD_BUCKET` has **no default** (`packages/shared/src/env.ts:249`).

**Deliberate delay before processing** (`processEventBatch.ts:66-92`): jobs enqueue with a 5 s delay (0 for OTel), but between **23:45–00:15 UTC** the full 15 s `LANGFUSE_INGESTION_QUEUE_DELAY_MS` is used — *"the delay around date boundaries to avoid duplicates for out-of-order processing"*. A partition-boundary hazard worth knowing about before you design anything date-partitioned.

**The merge** (`IngestionService.mergeRecords`, L1078-1099): the ClickHouse row is placed **first** as the baseline, then S3 events overwrite it in sorted order, then `event_ts = now()` makes the result win the ReplacingMergeTree race. `immutableEntityKeys` protects fields a later update may never overwrite — traces `[id, project_id, timestamp, created_at, environment]`, observations additionally `[trace_id, start_time]`. Deletes are soft: a tombstone row with `is_deleted=1` and a newer `event_ts`. High-volume projects can skip the read entirely via `ClickhouseReadSkipCache` (L1566-1571).

### Failure handling in the writer — read this before copying anything

`ClickhouseWriter/index.ts` has three bespoke retry classes: retryable network errors; JS *"invalid string length"* → split the batch in half and re-queue the remainder (L174-208); and ClickHouse *"JSON object extremely large"* → `truncateOversizedRecord` keeps the first 500 KB of oversized fields (L210-280). `Decimal64(12)` values are clamped at ±999,999.999999 (L282-356). On terminal failure rows are re-queued up to `maxAttempts` and then **silently dropped** — L518 carries an explicit `// TODO - Add to a dead letter queue in Redis rather than dropping`. **Langfuse can lose data under sustained ClickHouse failure.**

### Queues

`packages/shared/src/server/queues.ts:370-408` defines **37 queue names**; `worker/src/queues/` has 26 modules plus `shardedQueueRegistry.ts`. Sharding is SHA-256 of the key mod shard count (`redis/sharding.ts:9-20`), and only activates when `REDIS_CLUSTER_ENABLED=true`. Eight queues are excluded from the generic `getQueue` factory because they *must* be obtained with a sharding key (`redis/getQueue.ts:35-46`).

Explicit **secondary queues** isolate high-throughput or misbehaving tenants — `IngestionSecondaryQueue`, `OtelIngestionSecondaryQueue`, `EvaluationExecutionSecondaryQueue`. Routing is automatic: an S3 `SlowDown` throttle on a project sets a Redis flag that demotes that project's subsequent jobs (`processEventBatch.ts:300-312`). This is a good backpressure pattern, and it exists because they got burned.

IngestionQueue job options: `attempts: 6`, exponential backoff base 5 s, `removeOnComplete: true`, `removeOnFail: 100_000`.

### Why ClickHouse

Trace/observation/score volume is append-heavy, high-cardinality, and queried analytically (percentiles, `sumMap` over token maps, time-bucketed aggregates). `ReplacingMergeTree(event_ts, is_deleted)` gives idempotent upserts without transactions — essential when the same entity arrives as several partial events out of order. Partitioning by month with `ORDER BY (project_id, toDate(ts), id)` makes tenant+time the access path.

### Operational footprint

`docker-compose.yml` — **6 services**: `langfuse-web`, `langfuse-worker`, `postgres`, `clickhouse`, `minio`, `redis`. Three stateful datastores plus Redis. `.env.prod.example` documents ~211 env keys (only ~8 required). Six `# CHANGEME` insecure defaults ship in the compose file.

### Scale claims

**No throughput numbers are claimed anywhere in the repo** — not in README, CONTRIBUTING, or AGENTS.md. The evidence is structural (sharding, secondary queues, S3 offload, batching writer) plus a handful of honest in-code admissions:

| Knob | Default | Source |
|---|---|---|
| `LANGFUSE_INGESTION_QUEUE_PROCESSING_CONCURRENCY` | 20 | `worker/src/env.ts:89` |
| `LANGFUSE_INGESTION_CLICKHOUSE_WRITE_BATCH_SIZE` | 1000 | `worker/src/env.ts:98` |
| `LANGFUSE_INGESTION_CLICKHOUSE_WRITE_INTERVAL_MS` | 1000 | `worker/src/env.ts:102` |
| `LANGFUSE_S3_CONCURRENT_READS` | 50 | `worker/src/env.ts:369` |
| `LANGFUSE_INGESTION_QUEUE_DELAY_MS` | 15_000 | `packages/shared/src/env.ts:159` |
| `LANGFUSE_OTEL_MAX_SPAN_BYTES` | 9_500_000 | `packages/shared/src/env.ts:176` — *"just under ClickHouse's 10MB min_chunk_bytes_for_parallel_parsing default"* |

The most useful data point is a comment, not a benchmark — `worker/src/queues/ingestionQueue.ts:186-187`: *"If a user has 5k events, this will likely take 100 seconds."* That is the S3-list-and-merge path, and it is the honest shape of this architecture's tail latency.

### A v4 rewrite is in flight — schema instability risk

`ClickhouseWriter` writes to `events_full`, described as *"Primary write target - MV auto-populates events_core"* (`index.ts:611`). Its DDL is **not in the migrations folder** — it lives in `packages/shared/clickhouse/scripts/dev-tables.sh` because `enable_block_number_column` needs ClickHouse ≥24.5 and `text` indexes need ≥25.x, which would strand self-hosters on older versions. Gated by `LANGFUSE_MIGRATION_V4_WRITE_MODE` and `LANGFUSE_ENABLE_EVENTS_TABLE_*`. Repositories already carry `*FromEvents` query variants alongside the current ones.

**Implication for CRED:** the trace/observation/score schema is mid-migration to a unified `events` model. Anyone building tightly against today's ClickHouse tables is building against a moving target.

### A failed optimization worth knowing

Migration `0023_traces_aggregating_merge_trees.up.sql` built a `traces_null` trigger table feeding `traces_all_amt` / `traces_7d_amt` / `traces_30d_amt` AggregatingMergeTree rollups. Migrations **`0028` and `0029` drop all of them** (`DROP VIEW traces_all_amt_mv…`, `DROP TABLE traces_null, traces_all_amt, …`). Langfuse tried materialized trace-level rollups and reverted. Trace-level cost/latency is back to query-time aggregation (§6).

---

## 4. Prompt Management & Versioning — the harness-versioning template

**This is the section to mine.**

### Schema — `packages/shared/prisma/schema.prisma:865-892`

```prisma
model Prompt {
  id        String   @id @default(cuid())
  projectId String
  createdBy String
  prompt    Json                       // string (text) or message array (chat)
  name      String
  version   Int
  type      String   @default("text")  // immutable across versions of a name
  config    Json     @default("{}")    // arbitrary versioned config blob
  tags      String[] @default([])      // cross-version, synced to all rows
  labels    String[] @default([])      // per-version movable pointer
  commitMessage String?                // max 500 chars
  @@unique([projectId, name, version])
}
```

Design points that matter:

- **A row is a prompt *version*, not a prompt.** Logical identity is `(projectId, name)`; there is no parent table.
- **`config` is an arbitrary JSON blob versioned atomically with content.** This is exactly the "bundle" primitive — model params, tool configs, retrieval settings all version as one unit.
- **`tags` describe the prompt; `labels` select the version.** Clean separation.
- **Folders are pure naming convention** — `/` in `name`, queried with `startsWith`. `|` is forbidden in names because it delimits dependency syntax.
- `commitMessage` gives a git-like changelog per version.

Constants (`packages/shared/src/features/prompts/constants.ts:1-16`): `PRODUCTION_LABEL="production"`, `LATEST_PROMPT_LABEL="latest"`, label regex `/^[a-z0-9_\-.]+$/` max 36 chars.

### Version creation — `web/src/features/prompts/server/actions/createPrompt.ts:75-235`

- **Version assignment is read-max-then-+1** (L88-91, L128), *not* a sequence.
- `latest` is **materialized**, applied to every new version (L111) and stripped from priors in the same transaction. It is explicitly non-assignable via the API (`promptVersionHandler.ts:8-14`).
- Label migration off previous versions returns *unexecuted* Prisma promises appended to one `$transaction(array)` (L172-207) — insert + dependency rows + label strips + tag sync commit atomically.
- **Race handling:** concurrent creates both compute `N+1`; the loser hits `@@unique` and gets a `P2002` translated to *"…too many concurrent prompt creations… Please add a delay"* (`prompt-api-service.ts:52-66`). No lock, no retry. By contrast the label-only PATCH path **does** take `SELECT … FOR UPDATE` (`updatePrompts.ts:25-45`).
- Change events emitted for the created version **and every version that lost a label** (`promptChangeEventSourcing.ts:14-56`).

### Labels are the indirection layer

`GET /api/public/v2/prompts/{name}?version=|label=` — mutually exclusive; **defaults to `label=production`** when neither is given (`getPromptByName.ts:26-55`). PATCH is additive-only.

**The critical detail for CRED:** the SDK resolves a label to a concrete integer client-side, and **what lands on the observation is the resolved integer version, never the label** (`worker/src/services/IngestionService/index.ts:253-265, 334-336`). That is what makes version→outcome analytics well-defined as the pointer moves. `prompt_id` degrades to `""` on lookup failure but `prompt_name`/`prompt_version` are recorded verbatim — and there is deliberately **no FK** (`schema.prisma:498`, "no fk constraint, prompt can be deleted") so telemetry outlives the prompt.

### Caching — `packages/shared/src/server/services/PromptService/index.ts`

Epoch-token indirection instead of key deletion:
```
key   = prompt:{projectId}:{epoch}:{name}:{version:N | label:X}
epoch = prompt_cache_epoch:{projectId}   (48-bit token, 7d TTL)
invalidate() = rotate the epoch token    // O(1), no SCAN; old keys expire via TTL
```
TTL 3600 s (`packages/shared/src/env.ts:112-113`). Two subtleties: the epoch is **project-scoped, not prompt-scoped**, because resolved prompts inline other prompts; and the `version:`/`label:` selector prefix exists because a label could literally be `"3"`.

### Composition

`@@@langfusePrompt:name=X|version=3@@@` or `|label=production@@@` (`parsePromptDependencyTags.ts:3`). `MAX_PROMPT_NESTING_DEPTH = 5`, cycle detection, and a rule that a prompt may not reference any version of its own name. Dependencies persist in `PromptDependency` (`schema.prisma:894-912`) referencing child by **name + (version XOR label)** — symbolic and late-bound. Reverse lookups **gate label removal**: you cannot move a label out from under a dependent prompt.

### Promotion gate

`PromptProtectedLabels` (`schema.prisma:914-924`) + a separate RBAC scope `promptProtectedLabels:CUD`. Creating versions is cheap; pointing `production` at one is a distinct permission. (This is one of the 7 EE-gated entitlements.)

### Version → metrics

`packages/shared/src/server/repositories/observations.ts:1393-1470`:
```sql
SELECT count(*), prompt_id, prompt_version,
  medianExact(...usage_details 'input'...)  AS median_input_usage,
  medianExact(...usage_details 'output'...) AS median_output_usage,
  medianExact(cost_details['total'])        AS median_total_cost,
  medianExact(latency_ms)                   AS median_latency_ms
FROM observations FINAL
WHERE type='GENERATION' AND prompt_id IN (...) AND project_id={...}
GROUP BY prompt_id, prompt_version ORDER BY prompt_version DESC
```
Fanned in with observation- and trace-level scores by `promptRouter.ts:1278-1341`.

---

## 5. Evaluation

### Datasets are bitemporal — the other big steal

`schema.prisma:715-743`:
```prisma
model DatasetItem {
  validFrom DateTime @default(now())
  validTo   DateTime?
  isDeleted Boolean  @default(false)
  @@id([id, projectId, validFrom])     // SCD-2
}
```
Point-in-time resolution: `POST /dataset-run-items` accepts a `datasetVersion` ISO timestamp and resolves items "as they existed at or before this timestamp" (`fern/apis/server/definition/dataset-run-items.yml:47-52`). ClickHouse carries `dataset_item_version Nullable(DateTime64(3))` (migration 0032).

### Run items denormalize immutable snapshots

`0024_dataset_run_items.up.sql`:
```sql
CREATE TABLE dataset_run_items_rmt (
    id, project_id, dataset_run_id, dataset_item_id, dataset_id,
    trace_id, observation_id Nullable, error Nullable,
    -- denormalized immutable dataset run fields
    dataset_run_name, dataset_run_description,
    dataset_run_metadata Map(LowCardinality(String), String), dataset_run_created_at,
    -- denormalized dataset item fields (mutable, but snapshots are relevant)
    dataset_item_input Nullable(String) CODEC(ZSTD(3)),
    dataset_item_expected_output Nullable(String) CODEC(ZSTD(3)),
    dataset_item_metadata Map(LowCardinality(String), String),
    event_ts, is_deleted
) ENGINE = ReplacingMergeTree(event_ts, is_deleted)
ORDER BY (project_id, dataset_id, dataset_run_id, id);
```
Moved from Postgres because the comparison query joins run items × observations × scores — all ClickHouse-resident (backfills: `worker/src/backgroundMigrations/migrateDatasetRunItemsFromPostgresToClickhouseRmt.ts`).

### LLM-as-judge

- `EvalTemplate` (`schema.prisma:1022-1049`) is versioned with **the same `@@unique([projectId, name, version])` pattern as `Prompt`** — a consistent design language. `type ∈ {LLM_AS_JUDGE, CODE}`; CODE evaluators carry `sourceCode` (Python/TypeScript, up to 256 KB).
- `JobConfiguration` (`:1083-1110`) is the trigger: `filter Json`, `targetObject`, `variableMapping Json`, `sampling Decimal (0..1)`, `delay Int` ms, `timeScope String[] = ["NEW"]`.
- Trigger paths (`worker/src/queues/evalQueue.ts`): `TraceUpsert` → new traces; `DatasetRunItemUpsert` → experiment runs; `CreateEvalQueue` → UI backfill over history.
- `createEvalJobs` (`worker/src/features/evaluation/evalService.ts:185-717`): Redis negative-cache when no configs; **infinite-loop safeguard skipping traces whose `environment` starts with `"langfuse"`**; in-memory filter evaluation where possible, ClickHouse lookup otherwise; batched dedup; sampling via `Math.random() > sampling`; and **de-selection** — a re-emitted trace that no longer matches sets its pending execution to `CANCELLED`.
- Structured output (`packages/shared/src/features/evals/outputDefinition.ts:266-320`): the schema handed to the LLM is always `{ reasoning: string, score: number | boolean | enum(categories) | array(enum) }`. CATEGORICAL emits **one score row per match**; `comment` = `reasoning`.
- Providers (`packages/shared/src/server/llm/types.ts:246-253`): `anthropic, openai, azure, bedrock, google-vertex-ai, google-ai-studio`.
- Eval LLM calls are **themselves traced** into the project under environment `LLMJudge`.
- Failure handling is unusually mature: `EvaluatorBlockReason` (8 variants) blocks an evaluator on auth/billing/endpoint/model-config failure rather than retrying forever.

### Scores

`packages/shared/src/domain/scores.ts` — sources `API | EVAL | ANNOTATION` (`EVAL` reserved for internal evaluators, not settable externally); data types `NUMERIC | CATEGORICAL | BOOLEAN | CORRECTION | TEXT`.

**Attachment targets are four nullable columns** — `traceId`, `observationId`, `sessionId`, `datasetRunId` (`ScoreFoundationSchema:106-121`) — added incrementally (session in migration 0012, run in 0017; `trace_id` made nullable in 0014 to permit them). Validation funnels through one "central choke point": `packages/shared/src/server/ingestion/validateAndInflateScore.ts`.

**For CRED:** `datasetRunId`-attached scores are the closest existing analogue to an outcome-on-a-task. That column exists precisely because Langfuse hit the same "scores need to attach above the trace" wall.

### Annotation queues

`AnnotationQueue` / `AnnotationQueueItem` / `AnnotationQueueAssignment` (`schema.prisma:612-681`). Items target `TRACE | OBSERVATION | SESSION`, carry `lockedAt`/`lockedByUserId` for concurrent-annotator safety, and `scoreConfigIds String[]` denormalized on the queue.

### How CRED would benchmark context packages

Map: context package → dataset; a package version → the `validFrom` timestamp; a benchmark execution → dataset run pinned via `datasetVersion`; quality signal → scores on the run. This works today over the REST API or `langfuse.experiment.*` OTLP attributes. **The gap:** a dataset run is an offline batch, not a live production task. CRED's unit is a real task that happened once, not an item in a curated set.

---

## 6. Cost & Token Attribution

### Pricing schema — `schema.prisma:927-989`

```prisma
model Model {
  modelName    String
  matchPattern String        // regex matched against provided_model_name
  startDate    DateTime?     // time-scoped pricing
  unit         String?       // TOKENS|CHARACTERS|MILLISECONDS|SECONDS|REQUESTS|IMAGES
  tokenizerId  String?
  tokenizerConfig Json?
  projectId    String?       // NULL = built-in global model
  @@unique([projectId, modelName, startDate, unit])
}
model PricingTier {          // conditional pricing (e.g. long-context surcharge)
  name String, isDefault Boolean, priority Int, conditions Json @db.JsonB
  @@unique([modelId, priority])
  @@unique([modelId, name])
}
model Price { modelId, pricingTierId, usageType String, price Decimal
  @@unique([modelId, usageType, pricingTierId]) }
```

Three-level design: `Model` (matching + time-scoping) → `PricingTier` (conditional, priority-ordered) → `Price` (per usage-type rate). `usageType` is open-ended, which is why usage/cost are `Map` columns rather than fixed input/output pairs — cache reads, reasoning tokens, and audio tokens all slot in without migration.

### Model matching is a Postgres regex

`packages/shared/src/server/ingestion/modelMatch.ts:266-306`:
```sql
FROM models
WHERE (project_id = ${projectId} OR project_id IS NULL)
  AND ${model} ~ match_pattern          -- POSIX regex operator
ORDER BY project_id ASC, start_date DESC NULLS LAST
LIMIT 1
```
`ORDER BY project_id ASC` puts non-NULL first, so **a project's own model definition beats the built-in**; then the most recent `start_date` wins. Patterns look like `'(?i)^(claude-3-haiku-20240307|anthropic\.claude-3-haiku-20240307-v1:0|claude-3-haiku@20240307)$'`. Redis-cached with a `NOT_FOUND_TOKEN` negative cache (L180-236, L308).

### Token counting and the provided-cost short-circuit

`IngestionService.getUsageUnits` (L1239-1368): manual tokenization runs **only if** a model matched, the user supplied *zero* usage details, and `level !== ERROR`. Tokenizer comes from `model.tokenizerId`/`tokenizerConfig`.

`calculateUsageCosts` (L1444-1516) has one rule that surprises people:
```ts
// If user has provided any cost point, do not calculate any other cost points
if (providedCostKeys.length) { ... return { cost_details: {...provided}, total_cost }; }
```
**Any** caller-supplied cost fully short-circuits model pricing. Otherwise it's a per-usage-type join: for each `usage_details` key find the matching `Price.usageType` and `price.mul(units)` in decimal.js.

Pricing tiers (`packages/shared/src/server/pricing-tiers/matcher.ts`) evaluate conditions with a regex over usage-detail *keys*, sum all matching keys, and compare with `gt|gte|lt|lte|eq|neq`. AND semantics; any exception → `false` (fail-safe); non-default tiers by `priority` asc, first full match wins, else the default tier.

A nice defensive touch worth copying: `warnOnUsageTotalMismatch` (L1399-1442) flags when summed buckets exceed the provided `total` beyond 1%, but **deliberately refuses to auto-correct** because buckets can be genuinely additive (Bedrock cache writes). It emits a metric and a rate-limited warning instead of silently "fixing" data.

### provided_* vs computed

The `provided_usage_details`/`provided_cost_details` vs `usage_details`/`cost_details` split preserves caller-supplied values alongside Langfuse-computed ones. Resolved model id is denormalized onto the observation as `internal_model_id`, plus `usage_pricing_tier_id/name` (migration 0031) so the applied tier is auditable after the fact.

### Rollup is query-time, not materialized

**The `traces` table has no cost or usage columns at all** — verified against `0001_traces.up.sql`. Cost lives only on `observations` (`0002_observations.up.sql:21-23`).

`packages/shared/src/server/repositories/traces.ts:1758-1779`:
```sql
observation_stats AS (
  SELECT trace_id, project_id,
    sum(total_cost) as total_cost,
    date_diff('millisecond', ...) as latency_milliseconds,
    sumMap(usage_details) as usage_details,
    sumMap(cost_details) as cost_details
  FROM observations ${shouldUseFinal ? "FINAL" : ""}
  WHERE project_id = {projectId:String}
    AND start_time >= {cteFromTimeFilter} - TRACE_TO_OBSERVATIONS_INTERVAL
  GROUP BY project_id, trace_id
)
```
Note the tell-tales: `FINAL` is **conditional** (L1750-1752) — a correctness/cost trade-off toggled per query. And `TRACE_TO_OBSERVATIONS_INTERVAL` pads the time window so partition-pruned observation scans still catch a trace's children. Both are symptoms of aggregating across a boundary the schema doesn't model.

Other sites: `traces.ts:1183-1187, 1388-1399`; `observations.ts:1180-1191` (manual `LIMIT 1 BY` dedup instead of `FINAL`), `1513-1528`, `2012`; `daily-metrics.ts:32-59`. And `traces.ts:129-133` has the aggregations **commented out** with *"Remove those columns as this should only be used within the evalService and doesn't use them"* — a hand-tuned perf escape hatch.

Two competing definitions of total cost coexist — `sum(total_cost)` (scalar column) and `sumMap(cost_details)['total']` (map entry). They agree only because `calculateUsageCosts` always writes both.

There is **no materialized trace-level cost column**, and the AggregatingMergeTree attempt to add one was reverted (§3, migrations 0023 → 0028/0029). Session-level cost is a further query-time aggregation on top of that.

---

## 7. Interfaces & SDKs

- **REST:** 83 route files under `web/src/pages/api/public/`, 33 groups. Versioning is messy — unversioned v1 at root, `v2/` (prompts, datasets, scores, observations, metrics), `v3/scores`, and an explicit `unstable/` tier.
- **OpenAPI:** `fern/` holds 3 API definitions / 45 files; the official Python and JS SDKs are **generated from these specs**, and `.github/workflows/sdk-api-spec.yml` keeps them honest. The spec is the contract, not after-the-fact docs — good for a third-party emitter.
- **Auth:** `Basic base64(pk-lf-…:sk-lf-…)` for full scope; `Bearer <publicKey>` for restricted browser-side score ingestion (`web/src/features/public-api/server/apiAuth.ts:75-203`). Redis-cached hash lookup with bcrypt fallback and negative caching. Plan is resolved from the API key itself, so entitlements apply to API traffic.
- **MCP:** a genuinely substantial server — `web/src/pages/api/public/mcp/index.ts` + ~115 files under `web/src/features/mcp/`, **68 tools across 14 feature modules** (prompts, datasets, scores, evals, experiments, annotationQueues, observations, metrics, models, media, comments, dashboardWidgets, monitors, health). Streamable HTTP transport (2025-03-26 spec), stateless per-request server, project-scoped Basic auth. **MIT and not entitlement-gated.** Explicitly warns clients to refresh capabilities dynamically.

### Effort for CRED to be a first-class emitter

**Low — roughly a day for a working integration.** Three viable levels:

1. **OTLP** (lowest effort, no Langfuse dependency in CRED's code): POST protobuf or JSON to `/api/public/otel/v1/traces` with Basic auth. Set `langfuse.observation.type`, `langfuse.observation.prompt.name`/`.version`, `langfuse.trace.metadata.*`, `langfuse.release`, `session.id`, `user.id`. Use `langfuse.internal.as_root` to control trace boundaries.
2. **Native ingestion** (`POST /api/public/ingestion`) for batched trace/observation/score events with explicit ids.
3. **Task-run semantics**: `POST /dataset-run-items` (auto-creates the run by `runName`, accepts `datasetVersion` for point-in-time pinning), or the equivalent `langfuse.experiment.*` OTLP attributes.

---

## 8. OSS Project Mechanics

### License boundary — precise

`LICENSE:1-9`: content under **`ee/`, `web/src/ee/`, and `worker/src/ee/`** is governed by `ee/LICENSE`; **everything else is MIT Expat**.

`ee/LICENSE` permits copying/modifying for **development and testing without a subscription**, requires a valid license for production use, and **forbids redistribution outright**. It states explicitly that the MIT core "can be used and run without infringing" it.

> **Gap worth a written clarification:** `packages/shared/src/server/ee/` (containing `ingestionMasking/` and `licenseCheck/`) is an EE-named directory **not listed** in the root LICENSE carve-out. By the literal text it is MIT.

### What is actually commercial

`ee/` itself is ~9 files (a license-check shim). The real EE surface:
- `web/src/ee/features/`: `in-app-agent`, `billing` (Stripe), `multi-tenant-sso`, `audit-log-viewer`, `ui-customization`, `verified-domains`, `sfdc-sync` (Salesforce), `admin-api`, `sso-settings`
- `worker/src/ee/`: `dataRetention`, `cloudSpendAlerts`, `cloudUsageMetering`, `meteringDataPostgresExport`, `usageThresholds`

Much of this is Langfuse's own SaaS billing/CRM plumbing, not features a self-hoster wants.

**All 13 entitlements** (`web/src/features/entitlements/constants/entitlements.ts:6-21`), self-hosted view:

| Entitlement | oss | sh:pro | sh:enterprise |
|---|:-:|:-:|:-:|
| `trace-deletion` (dead code) | Y | Y | Y |
| `scheduled-blob-exports` | Y | Y | Y |
| `rbac-project-roles` | – | – | **Y** |
| `audit-logs` | – | – | **Y** |
| `data-retention` | – | – | **Y** |
| `prompt-protected-labels` | – | – | **Y** |
| `admin-api` | – | – | **Y** |
| `self-host-ui-customization` | – | – | **Y** |
| `self-host-allowed-organization-creators` | – | – | **Y** |
| `cloud-billing`, `cloud-spend-alerts`, `cloud-multi-tenant-sso`, `in-app-agent` | – | – | – (cloud only) |

Three findings:
- **`self-hosted:pro` is entitlement-identical to `oss`** (`entitlements.ts:140-161`). It buys support, not features.
- **`in-app-agent` is cloud-only at every tier** — unavailable self-hosted at any price.
- **Everything CRED would build on is free and ungated**: tracing, OTLP, prompts, datasets, evals, scores, annotation queues, dashboards, SCIM 2.0, MCP. Evals and annotation queues are *count*-limited only, and self-hosted gets unlimited (`model-based-evaluations-count-evaluators` is `false`/unlimited on **every** plan).

**License enforcement is `String.prototype.startsWith`** (`web/src/features/entitlements/server/getPlan.ts:57-67`): `langfuse_ee_` → enterprise, `langfuse_pro_` → pro. No signature, expiry, seat count, or phone-home. Their own CI uses `LANGFUSE_EE_LICENSE_KEY=langfuse_ee_test`. Treat the boundary as **legal, not technical**.

### Self-host footprint

6 containers (§3). ~211 documented env vars, ~8 required. Azure Blob and OCI object storage are first-class alternatives; Redis Cluster supported.

### Community health & cadence

The local clone is **shallow (depth 1)** — `git log | wc -l` = 1, no tags. Real velocity numbers require `git fetch --unshallow` or the GitHub API. Structural proxies:
- Version `3.221.1` — 221 minor releases on the v3 line.
- Head commit is PR **#15203** (Jul 2026).
- **30 GitHub workflows**, including CodeQL, Semgrep, Snyk (web + worker), `zizmor` (Actions supply-chain linting), and a `licencecheck.yml` that **fails the build on any weak/strong/network copyleft dependency**.
- Actions are SHA-pinned with `permissions: {}` defaults — mature supply-chain posture.
- **CLA required** from contributors (standard for open-core relicensing ability).
- Release process force-pushes `main` → `production` on `v3.x.y` tags; Docker images are tagged `:3`, a **floating major tag** — pin digests yourself.
- `.agents/` contains ~28 maintained skills including `langfuse-codebase-navigator` and a 25-rule `clickhouse-best-practices` — unusually mature agent tooling for an OSS repo.

---

## Recommendation

### FIRM: (c) build your own outcome store, with (b) OTLP for the span tier. Do not embed Langfuse.

**Why not (a) embed.** Two disqualifying problems.

*Operationally:* 6 containers, 3 stateful datastores, ~37 queues, ~211 env vars. If CRED is self-hosted by dev teams, that is the product's install story. If CRED is SaaS, you inherit ClickHouse ops on day one to store data that is not your differentiator.

*Structurally, and this is the real one:* Langfuse's root entity is an LLM invocation tree. CRED's is a task with an outcome. Everything above the trace is either absent (`TraceSession` has five fields and no metadata) or offline-experiment-shaped (`dataset_runs`). You would spend your first year bending someone else's ClickHouse schema to express "this task shipped and here's which context version was loaded" — and every Langfuse upgrade would fight you. Note that Langfuse itself hit this wall: `scores.session_id` (migration 0012) and `scores.dataset_run_id` (0017) were bolted on later, forcing `trace_id` to become nullable (0014), precisely because scores needed to attach above the trace.

**Why not (b) alone.** Emitting OTLP and letting users point it at Langfuse/Phoenix/Braintrust makes CRED a well-behaved instrumentation library, not a product. None of those backends model a task or an outcome, so the correlation "context package v7 → 3 of 5 tasks reverted" lives nowhere. You would have no queryable substrate for your own core claim.

**Why (c) + (b).** Split by unit of analysis:

- **CRED owns the outcome plane, in Postgres.** `task`, `outcome`, `knowledge_package`, `package_version`, `harness_version`, `task_context_binding`. This is low-volume — thousands of tasks per team per month, not millions of spans. It does not need ClickHouse, a queue fleet, or a blob store. A single Postgres instance carries it for years. This is the defensible asset.
- **CRED does not own the span plane.** Emit OTLP with standard `gen_ai.*` semconv plus a reserved `cred.*` namespace (`cred.task.id`, `cred.package.version`, `cred.harness.version`). Correlate by storing the OTel `trace_id` on the CRED task row. If the user runs Langfuse, additionally emit the `langfuse.*` attributes — `langfuse.observation.prompt.version` alone gets harness-version attribution into their existing per-version metrics query for free.
- **Ship a Langfuse adapter, not a Langfuse dependency.** Optional, one of several. The 68-tool MCP server and the Fern-generated OpenAPI spec make this cheap and stable.

The one-line test: if a customer deletes their entire observability backend, CRED must still answer "which context version contributed to this outcome". Under (a) or (b) it cannot. Under (c) it can, and the spans become enrichment.

### Top 3 things to STEAL

**1. Prompt-versioning as harness-versioning — the whole pattern.**
`packages/shared/prisma/schema.prisma:865-892`; `web/src/features/prompts/server/actions/createPrompt.ts:75-235`

- Immutable append-only rows keyed `(scope, name, version Int)`; content **and** an arbitrary `config Json` blob version atomically as one unit. A CRED harness version = prompt + tool defs + retrieval config + model params in one immutable row.
- **Labels as a mutable pointer array on the row**, with `production` as the default resolution target and `latest` system-managed and non-assignable.
- **The load-bearing detail:** consumers resolve by label, but telemetry records the *resolved integer version* (`IngestionService/index.ts:253-265, 334-336`). Late binding for humans, hard pinning for analytics. Copy this exactly — it is what makes "which version caused this outcome" answerable after the pointer moves.
- **No FK from telemetry to the version row** (`schema.prisma:498`) so outcome records survive deletion.
- **Two-tier RBAC as promotion gate**: creating a version needs `prompts:CUD`; moving a *protected* label needs `promptProtectedLabels:CUD`. That is your "who can promote a context package to production" control, already designed.
- **Epoch-token cache invalidation** (`PromptService/index.ts:179-240`): key `pkg:{project}:{epoch}:{name}:{version|label}`, invalidate by rotating a project-scoped epoch token — O(1), no SCAN, old keys expire by TTL. Project-scoped rather than package-scoped **because composition makes cached entries interdependent**. CRED will have composition, so inherit the project-scoped choice.
- Emit change events for the new version **and every version that lost a label** (`promptChangeEventSourcing.ts:14-56`) so downstream sees the full pointer transition.

**2. Bitemporal context items + denormalized run-item snapshots.**
`schema.prisma:715-743`; `packages/shared/clickhouse/migrations/unclustered/0024_dataset_run_items.up.sql`

`DatasetItem` uses SCD-2 — `validFrom`/`validTo`/`isDeleted` with `@@id([id, projectId, validFrom])` — and runs pin a `datasetVersion` timestamp to resolve items as of that instant. Do this for knowledge/context items: a task executed on 2026-03-01 must resolve the context exactly as it was then, even after the doc is edited.

Pair it with the `dataset_run_items_rmt` trick: **denormalize an immutable snapshot of the run config and item input onto the outcome row** ("denormalized immutable dataset run fields" / "mutable, but snapshots are relevant"). Your outcome records must be readable without joining live, mutating context tables — both for correctness and to avoid the join entirely.

**3. The OTel vendor-attribute namespace and priority-ordered mapper registry.**
`packages/shared/src/server/otel/attributes.ts:1-54`; `ObservationTypeMapper.ts:165+`

Reserve a `cred.*` attribute namespace **before** you ship an SDK, mirroring the `langfuse.trace.*` / `langfuse.observation.*` / `langfuse.experiment.*` split. Then adopt the priority-ordered mapper registry so CRED ingests spans from OpenInference, OTel GenAI semconv, Vercel AI SDK, and Genkit rather than demanding its own SDK. That registry is why Langfuse works with tools it never integrated with, and it is ~500 well-factored lines. Also copy `langfuse.internal.as_root` — an explicit override for "this span begins a new logical unit" is exactly what you need to bound a task across a distributed trace.

### Top 3 things to AVOID

**1. The infrastructure footprint. Do not start with ClickHouse.**
6 containers, 37 queues, S3-first ingestion with a mandatory bucket, per-queue shard/concurrency knobs, secondary queues for tenant isolation, and a date-boundary enqueue delay to dodge partition races. Langfuse needs this because it ingests every LLM call from every customer. **CRED ingests tasks** — orders of magnitude fewer. Start on Postgres; adopt a columnar store only when you can name the specific query Postgres can no longer serve.

The tail is worse than the headline: `ingestionQueue.ts:186-187` admits *"If a user has 5k events, this will likely take 100 seconds"*, and `ClickhouseWriter/index.ts:518` carries `// TODO - Add to a dead letter queue in Redis rather than dropping` — under sustained ClickHouse failure, rows are **dropped after `maxAttempts`**. If CRED's outcome records are the product, they cannot live in a pipeline that silently drops on backpressure. Postgres with a synchronous write gives you durability for free at your volume.

Also note the schema is mid-rewrite (§3, v4 `events_full`), with DDL living outside the migrations folder for ClickHouse-version reasons. Building tightly against today's tables means building against a moving target.

**2. Query-time rollups — and their failed materialization.**
Trace-level cost is `sumMap(cost_details)` at read time (`traces.ts:1762-1764`), never materialized; one code path has it commented out (`traces.ts:131-132`) as a live perf trade-off. Langfuse tried to fix this with AggregatingMergeTree MVs in migration `0023` and **dropped the entire apparatus in `0028`/`0029`**. Take the lesson in both directions: don't leave outcome-level aggregates to read time *and* don't reach for materialized-view machinery early. For CRED, denormalize rollups onto the task row at write time — you control the write path, and the volume is small enough that it is simply cheaper.

**3. Lossy metadata and soft-enforced invariants.**

- **Metadata as `Map(LowCardinality(String), String)`.** Nested structure is flattened to dotted paths with JSON-stringified leaves (`otel/utils.ts:12-48`). Round-tripping is lossy and typed queries are impossible. CRED's context bindings are structured by nature — use `jsonb` and index it. Related: oversized fields are silently truncated to the first 500 KB (`ClickhouseWriter/index.ts:210-280`), so large inputs are lossy too.
- **Label uniqueness has no DB constraint.** `labels text[]` with single-holder semantics enforced only by application code stripping the label from priors inside a transaction. Harden it: a partial unique index over an unnested labels table. "Which version is in production" must not be able to drift.
- **Version numbering races.** Read-max-then-+1 with the unique constraint as the only concurrency control, surfacing *"too many concurrent prompt creations… Please add a delay"* to the client (`prompt-api-service.ts:52-66`). Telling API consumers to sleep is not a design. Use a sequence or `SELECT … FOR UPDATE` on a parent row — note Langfuse *does* lock correctly on the label-move path (`updatePrompts.ts:25-45`), so the inconsistency is an oversight, not a considered trade-off.
- Minor but telling: their eval pipeline needs an **infinite-loop safeguard** skipping traces whose `environment` starts with `"langfuse"` (`evalService.ts:232-258`), because eval LLM calls are traced back into the same project. If CRED's agents observe CRED, design the reentrancy boundary explicitly rather than string-matching your way out.

---

## Open questions

- Real contributor/velocity numbers need `git fetch --unshallow` or the GitHub API — unavailable from this clone.
- No published throughput benchmarks exist in-repo; scale evidence is structural only.
- `packages/shared/src/server/ee/` licensing ambiguity (§8) — worth written clarification if ingestion masking ever matters.
