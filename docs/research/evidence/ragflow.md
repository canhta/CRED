# RAGFlow — Evidence Review

**Repo:** `/Users/canh/Solo/OSS/ragflow` (InfiniFlow/ragflow)
**Commit reviewed:** `b20ca452e6583a0497b29a909c21429aa8685bd8` (2026-07-20), image tag `v0.26.4`
**License:** Apache 2.0, unmodified, no field-of-use restrictions (`LICENSE`)
**Method:** source read, not README read. All claims below cite `file:line`.

---

## Verdict

- **Do NOT depend on RAGFlow as a service, and do NOT vendor DeepDoc. Use Docling.** RAGFlow's own code settles the argument: DeepDoc is one of *eight* interchangeable parser backends in a dict at `rag/app/naive.py:373-382`, and Docling is already one of them. The maintainers do not treat DeepDoc as uniquely necessary; neither should CRED.
- **The citation data model is the thing worth stealing, and it costs ~100 lines.** A chunk cites via `position_int`: a list of `(page, x0, x1, top, bottom)` int 5-tuples (`rag/nlp/__init__.py:922-934`), rendered directly as highlight rects by the UI (`web/src/utils/document-util.ts:8-41`). Docling's native `prov[].bbox` + `page_no` maps onto it 1:1 — RAGFlow's own adapter proves it in ~90 lines (`deepdoc/parser/docling_parser.py:72-87,153-160`).
- **Citation fidelity is PDF-only.** For DOCX/MD/XLSX/PPTX-as-text, positions degrade to a fake `[[ii]*5]` sequence number, not a real bbox (`rag/nlp/__init__.py:407,429,454`). RAGFlow's "deep document understanding" gives CRED nothing beyond PDF that Docling doesn't also give.
- **Ops cost is disqualifying for a startup.** Minimum viable stack is 6 containers (Elasticsearch at a ~7.5 GiB per-container `mem_limit`, MySQL, MinIO, Valkey, deepdoc, ragflow), documented 16 GB RAM / 50 GB disk / 4 cores (`README.md:151-153`), **x86-only — no ARM64 images** (`README.md:193-195`). That is a full platform to operate in exchange for a bbox tuple.
- **`rag/` cannot be lifted out at all — the dependency on `api/` is circular.** 32 files in `rag/` import `api.*`; 28 files in `api/` import `rag.*`. `rag/svr/task_executor.py:38-83` alone pulls ~15 `api.db.*` modules. There is no "RAG core" to extract; there is one entangled application.
- **The codebase is mid-rewrite and is not a safe dependency.** ~492k lines of Go now shadow the Python backend, with its own API server, ingestor, MCP server, and agent runtime, selected by `API_PROXY_SCHEME=python|go|hybrid` (`docker/entrypoint.sh:190-205`). `CLAUDE.md:54-58` designates `internal/ingestion`, `internal/parser`, `internal/deepdoc` as "actively refactored," under a stated policy of "treat legacy code as liability" and "prefer deletion over shims." Anything vendored today is a moving target.

---

## 1. Parsing Pipeline — what actually happens to a PDF

### The pipeline

`RAGFlowPdfParser.__call__` (`deepdoc/parser/pdf_parser.py:1744-1769`) is 7 ordered stages:

```python
self.outlines = extract_pdf_outlines(fnm)
self.__images__(fnm, zoomin)          # rasterize + native char extract
self._layouts_rec(zoomin)             # layout ONNX
self._table_transformer_job(zoomin, auto_rotate=auto_rotate_tables)  # TSR ONNX
self._text_merge()
self._concat_downward()               # XGBoost line-merge
self._filter_forpages()
tbls = self._extract_table_figure(need_image, zoomin, return_html, False)
```

### Models — all bundled, all ONNX, all CPU-capable

| Model | File | Loaded at | Labels |
|---|---|---|---|
| Layout analysis | `layout.onnx` | `deepdoc/vision/layout_recognizer.py:48-66` | 11 classes: Text, Title, Figure, Figure caption, Table, Table caption, Header, Footer, Reference, Equation (`:34-46`) |
| Text detection | `det.onnx` | `deepdoc/vision/ocr.py:400` | — |
| Text recognition | `rec.onnx` | `deepdoc/vision/ocr.py:135` | — |
| Table structure | `tsr.onnx` | `deepdoc/vision/table_structure_recognizer.py:40-52` | 6 classes: table, column, row, column header, projected row header, spanning cell (`:31-38`) |
| Line-merge classifier | `updown_concat_xgb.model` | `deepdoc/parser/pdf_parser.py:92-101` | XGBoost, explicitly pinned to CPU (`:94`) |

All fetched from HuggingFace `InfiniFlow/deepdoc` via `snapshot_download` (`layout_recognizer.py:65`, `ocr.py:522`, `tsr.py:47`); the XGBoost model from `InfiniFlow/text_concat_xgb_v1.0` (`pdf_parser.py:100`). Downloads are lazy fallbacks — the Docker image pre-bakes them.

**Hardware:** CPU works. GPU is opt-in via `CUDAExecutionProvider` with a `gpu_mem_limit` (`ocr.py:108-119`), falling back to `CPUExecutionProvider` (`:121`). There are also Huawei Ascend NPU paths (`AscendLayoutRecognizer`, `layout_recognizer.py:246+`) and a YOLOv10 variant (`:168`). A remote DLA server can be pointed at via `DEEPDOC_URL`/`TENSORRT_DLA_SVR` (`layout_recognizer.py:53-61`).

### The cost driver nobody advertises

`__images__` **rasterizes every page unconditionally** at 216 DPI (`pdf_parser.py:1622`):

```python
self.page_images = [p.to_image(resolution=72 * zoomin, antialias=True).annotated for i, p in enumerate(...)]
```

and `__ocr` runs the **detection model on every page regardless of whether a native text layer exists** (`pdf_parser.py:773`): `bxs = self.ocr.detect(np.array(img), device_id)`. Native `pdfplumber` chars are then merged *into* the detected boxes (`:791-801`); recognition (`rec.onnx`) only runs where chars are missing. So the floor for any PDF is: rasterize + layout ONNX + det ONNX per page. There is no fast path for a clean digital PDF.

Genuinely good engineering worth noting: garbled-text detection triggers OCR fallback via two strategies — PUA/CID unmapped chars at threshold 0.3, and subset-font CJK→ASCII mis-encoding (`pdf_parser.py:1631-1656`), with a per-box re-check at `:826`.

### It's pluggable — and that's the headline

```python
# rag/app/naive.py:373-382
PARSERS = {
    "deepdoc": by_deepdoc, "mineru": by_mineru, "docling": by_docling,
    "opendataloader": by_opendataloader, "tcadp parser": by_tcadp,
    "paddleocr": by_paddleocr, "somark": by_somark,
    "plaintext": by_plaintext,  # default
}
```

Selected by a config string at `rag/app/naive.py:981-996`. Sizes: `docling_parser.py` 628 lines, `mineru_parser.py` 1029, `opendataloader_parser.py` 648, `somark_parser.py` 756, `tcadp_parser.py` 539. **RAGFlow ships adapters to its own competitors.** Note the default is `by_plaintext`, not DeepDoc.

---

## 2. Chunking Strategies

Templates are a flat dict of module references (`rag/svr/task_executor.py:114-131`), dispatched by `chunker = FACTORY[task["parser_id"].lower()]` (`:289`):

`general`/`naive`, `paper`, `book`, `presentation`, `manual`, `laws`, `qa`, `table`, `resume`, `picture`, `one`, `audio`, `email`, `kg`→naive, `tag`.

Sizes (`rag/app/*.py`): naive 1264, resume 2768, table 596, qa 442, paper 319, manual 301, laws 276, presentation 253, book 188, picture 184, one 176, email 129, tag 147, audio 64. **7,122 lines total.**

**How much does template choice matter? Less than the marketing implies.** `kg` is literally aliased to `naive` (`task_executor.py:129`). `book`, `laws`, `manual`, `paper` are each <320 lines of regex-and-heuristic variation on the same merge primitives (`naive_merge`, `naive_merge_docx`, `tokenize_chunks`). The real knobs are ordinary config, defaulted at `rag/app/naive.py:901`:

```python
parser_config = kwargs.get("parser_config", {"chunk_token_num": 512, "delimiter": "\n!?。；！？",
                                             "layout_recognize": "DeepDOC", "analyze_hyperlink": True})
```

The genuine outliers are `resume` (2768 lines, a bespoke entity extractor with bundled CSV/JSON gazetteers of schools, corporations, industries under `deepdoc/parser/resume/entities/res/`) and `table` (596 lines, column-type inference). Selection is manual per-knowledge-base; there is no auto-detection. As of this commit documents inherit `parser_id` from the KB rather than choosing per-upload (see the HEAD commit message).

**Relevance to CRED:** near zero. CRED ingests engineering docs, ADRs, tickets, PRs — the `naive` path with a token count and a delimiter. The template zoo is optimizing for a Chinese-enterprise document market CRED is not in.

---

## 3. Citation Mechanism — the part CRED actually needs

This is the best idea in the repo. The trick is that **position travels inline in the text as a sentinel tag**, so every chunker can slice and merge text freely without threading a parallel coordinate structure, and coordinates are recovered at the end.

### Step 1 — tag each layout box

`deepdoc/parser/pdf_parser.py:1522-1535`:

```python
def _line_tag(self, bx, ZM):
    pn = [bx["page_number"]]
    top  = bx["top"]    - self.page_cum_height[pn[0] - 1]
    bott = bx["bottom"] - self.page_cum_height[pn[0] - 1]
    ...
    while bott * ZM > self.page_images[pn[-1] - 1].size[1]:   # box spans a page break
        bott -= self.page_images[pn[-1] - 1].size[1] / ZM
        pn.append(pn[-1] + 1)
    return "@@{}\t{:.1f}\t{:.1f}\t{:.1f}\t{:.1f}##".format("-".join(map(str, pn)), bx["x0"], bx["x1"], top, bott)
```

Format: `@@<page|page-page>\t<x0>\t<x1>\t<top>\t<bottom>##`. Multi-page spans encode as `3-4`. Appended to section text at `:1591`.

### Step 2 — chunkers concatenate text freely

Tags ride along through `naive_merge` and friends untouched.

### Step 3 — recover coordinates at tokenization

`rag/nlp/__init__.py:391-405`:

```python
def tokenize_chunks(chunks, doc, eng, pdf_parser=None, ...):
    for ii, ck in enumerate(chunks):
        d = copy.deepcopy(doc)
        if pdf_parser:
            d["image"], poss = pdf_parser.crop(ck, need_position=True)
            add_positions(d, poss)
            ck = pdf_parser.remove_tag(ck)
        else:
            add_positions(d, [[ii] * 5])
```

Parsing back out (`pdf_parser.py:1937-1944`):

```python
@staticmethod
def extract_positions(txt):
    poss = []
    for tag in re.findall(r"@@[0-9-]+\t[0-9.\t]+##", txt):
        pn, left, right, top, bottom = tag.strip("#").strip("@").split("\t")
        poss.append(([int(p) - 1 for p in pn.split("-")], float(left), float(right), float(top), float(bottom)))
    return poss

@staticmethod
def remove_tag(txt):
    return re.sub(r"@@[\t0-9.-]+?##", "", txt)
```

### Step 4 — the stored schema

`rag/nlp/__init__.py:922-934`:

```python
def add_positions(d, poss):
    if not poss: return
    page_num_int, position_int, top_int = [], [], []
    for pn, left, right, top, bottom in poss:
        page_num_int.append(int(pn + 1))
        top_int.append(int(top))
        position_int.append((int(pn + 1), int(left), int(right), int(top), int(bottom)))
    d["page_num_int"]  = page_num_int    # [int]           — page filter / sort
    d["position_int"]  = position_int    # [(pg,x0,x1,top,bottom)] — highlight rects
    d["top_int"]       = top_int         # [int]           — reading-order sort
```

Three denormalized fields from one source, each serving a different query: `page_num_int` filters, `top_int` orders (`rag/nlp/search.py:771-779` sorts by `page_num_int`, `top_int`, `position_int`), `position_int` renders.

Alongside it, `d["image"]` holds a **cropped snapshot of the chunk region**, pushed to object storage and replaced by an `img_id` reference (`rag/svr/task_executor.py:389-398`, `image2id`). So the UI has both a vector rect *and* a raster fallback. `crop()` (`pdf_parser.py:1946+`) even pads 120px of context above and below the chunk (`:1984,1998-1999`) so the snapshot is readable.

### Step 5 — storage

Field types are inferred purely from **name suffix** via ES dynamic templates (`conf/mapping.json:25-215`): `*_int`→integer, `*_flt`→float, `*_kwd`/`*_id`→keyword, `*_tks`/`*_ltks`→analyzed text, `*_with_weight`→stored text, `*_512_vec`/`*_768_vec`/`*_1024_vec`/`*_1536_vec`→dense_vector, `*_fea`/`*_feas`→rank features. Adding a field requires zero mapping changes. Mirrored for Infinity (`conf/infinity_mapping.json:18-20`) and OceanBase (`rag/utils/ob_conn.py:78`, typed `ARRAY(ARRAY(Integer))`).

### Step 6 — retrieval and render

`rag/nlp/search.py:689,709` maps `position_int` → the API field `positions`; also at `:890,946`. `rag/prompts/generator.py:55` feeds it to the LLM prompt.

`web/src/utils/document-util.ts:8-41` — the whole UI side:

```ts
selectedChunk.positions.map((x) => {
  const boundingRect = { width: size.width, height: size.height,
                         x1: x[1], x2: x[2], y1: x[3], y2: x[4] };
  return { id: uuid(), comment: {...},
           content: { text: get(selectedChunk, 'content_with_weight') || ... },
           position: { boundingRect, rects: [boundingRect], pageNumber: x[0] } };
})
```

Coordinates are in PDF-point space (image pixels ÷ `ZM`, `pdf_parser.py:783`), so they're resolution-independent — the UI rescales against rendered page size.

### The critical caveat

**Only the PDF path produces real coordinates.** Every other route calls `add_positions(d, [[ii] * 5])` — the chunk *index* stuffed into all five slots (`rag/nlp/__init__.py:407,429,454`). So a DOCX chunk gets `position_int = [(ii+1, ii, ii, ii, ii)]`: a syntactically valid, semantically meaningless rect. PPTX is slightly better — page-level only, no intra-slide bbox (`rag/app/presentation.py:149,184,239`).

**If CRED needs sub-page citations in DOCX or Markdown, RAGFlow does not solve it and neither does copying its schema unmodified.**

---

## 4. Retrieval

- **`Dealer`** (`rag/nlp/search.py:39`) over a `DocStoreConnection` ABC (`common/doc_store/doc_store_base.py:148`).
- **Two scoring layers.** Engine-side fusion weights are **hardcoded** `"0.05,0.95"` text/vector (`search.py:210`); application-side rerank defaults to `vector_similarity_weight=0.3` → term 0.7 / vector 0.3 (`search.py:549-566`, `:604`; DB defaults `api/db/db_models.py:851-852`). Citation insertion uses different weights again: `tkweight=0.1, vtweight=0.9` (`search.py:251`).
- **Backends:** Elasticsearch (default), Infinity, OpenSearch, OceanBase/SeekDB — if/elif factory on `DOC_ENGINE` (`common/settings.py:303-324`). **The abstraction leaks**: `search.py` branches on `DOC_ENGINE_INFINITY`/`DOC_ENGINE_OCEANBASE` at `:207,:625,:631`, so a new backend requires editing the retrieval layer, not just implementing the ABC.
- **Reranking is entirely API/HTTP** — 23 provider classes in `rag/llm/rerank_model.py`, no local FlagEmbedding/CrossEncoder path in this commit. Self-hostable via TEI/XInference/LocalAI, but always over HTTP.
- **Rank features are additive and can dominate**: tag cosine ×10 plus raw pagerank added on top of the 0..1 hybrid score (`search.py:330-361`, default `rank_feature={PAGERANK_FLD: 10}` at `:564`).
- **RAPTOR** (`rag/advanced_rag/knowlege_compile/raptor.py:165`) — UMAP→GMM with a BIC sweep fitting a GaussianMixture for every n in 1..max_cluster (`:256-265`), then **1 LLM call + 1 embedding call per cluster per layer** (`:901-908`). Roughly N/4–N/2 LLM calls per document; ×3 worst case with retries (`:216-231`).
- **GraphRAG** (`rag/graphrag/`) is the most expensive path in the repo. Gleaning loop `ENTITY_EXTRACTION_MAX_GLEANINGS = 2` → up to 4 LLM calls/chunk (general, `general/graph_extractor.py:94-126`) or 5 (light, `light/graph_extractor.py:74-106`); plus one call per oversized entity/relation description (`extractor.py:338-360`); plus **quadratic** entity resolution over `itertools.combinations` (`entity_resolution.py:101-145`); plus one call per community per Leiden level (`community_reports_extractor.py:58-179`). Both cost drivers default **on** (`general/index.py:256`).

---

## 5. Ingestion Cost & Ops

### Executor architecture

`rag/svr/task_executor.py` (1,960 lines). **Queue is raw Redis Streams, not a task framework** — `XADD` (`rag/utils/redis_conn.py:408`), `XREADGROUP` (`:439`), `XACK` (`:47`), `XPENDING_RANGE` (`:482`). Two priority streams `te.1.common` and `te.0.common`, group `rag_flow_svr_task_broker`; priority 1 drains first (`task_executor.py:214-217`).

Concurrency is asyncio + semaphores in a single process (`rag/svr/task_executor_limiter.py`):

| Env var | Default | Line |
|---|---|---|
| `MAX_CONCURRENT_TASKS` | 5 | `:20` |
| **`MAX_CONCURRENT_CHUNK_BUILDERS`** | **1** | `:21` |
| `MAX_CONCURRENT_MINIO` | 10 | `:22` |
| `WORKER_HEARTBEAT_TIMEOUT` | 120 s | `task_executor.py:156` |

`embed_limiter` reuses `MAX_CONCURRENT_CHUNK_BUILDERS` (`:26`), so the default of **1 serializes both chunk-building and embedding**. None of these appear in `docker/.env` — they must be injected manually. Timeouts are decorator-based and generous: `build_chunks` 80 min (`:280`), 1 h at `:1037`, 3 h at `:1385`.

**Three coexisting Python execution paths** behind `TE_RUN_MODE` (`:1730-1747`): default `"0"` runs the new `rag/svr/task_executor_refactor/`, `"1"` runs old and new side by side for diffing, else legacy `do_handle_task()`. This is a *second*, Python-internal refactor running concurrently with the Go rewrite.

### Two failure modes worth knowing

1. **Redis acks unconditionally.** `redis_msg.ack()` at `task_executor.py:1785` sits *outside* the try/except, so a failed task is acked and never redelivered via pending-entry reclaim. Retry depends entirely on a DB counter.
2. **The retry counter is incremented on every claim**, including successful ones (`api/db/services/task_service.py:230`). At `retry_count >= 3` the doc is marked `FAIL` and `get_task` returns `None` (`:223-237`).

The Go ingestor fixes #1 with real `ackOrNack` semantics (`internal/ingestion/service/ingestion_service.go:520-535`) — but see §6 for why that isn't usable yet.

### Per-chunk LLM multipliers

`auto_keywords` and `auto_questions` each fire one LLM call per chunk (`task_executor.py:424-486`); LLM content tagging adds another (`:587-612`). Enabling all three roughly triples per-chunk ingestion cost *before* RAPTOR or GraphRAG.

### Deployment footprint

`docker/docker-compose.yml` + `docker-compose-base.yml`: `ragflow-cpu`/`ragflow-gpu`, `deepdoc`, `es01`, `opensearch01`, `infinity`, `oceanbase`, `seekdb`, `mysql:8.0.40`, `minio`, **`valkey/valkey:8`** (Redis has been replaced), `nats:2.14.2` (Go profile only), `jaeger`, `clickhouse`, `kibana`, `tei-cpu`/`tei-gpu`, `sandbox-executor-manager`. Always-on: mysql, minio, valkey. Minimum useful set ≈ 6 containers.

`ragflow-cpu` exposes **seven ports** (`docker-compose.yml:48-55`): 80, 443, 9380 (Python API), 9381 (Python admin), 9382 (MCP), 9383 (Go admin), 9384 (Go API).

- **`MEM_LIMIT=8073741824`** (~7.5 GiB) — `docker/.env:65`. Applied **per container** to elasticsearch (`:26`), opensearch (`:62`), infinity (`:89`), oceanbase (`:123`), seekdb (`:148`), clickhouse (`:360`). Enabling multiple doc engines multiplies it.
- **Minimums:** CPU ≥4 (x86), RAM ≥16 GB, Disk ≥50 GB, Docker ≥24.0.0, Python ≥3.13 (`README.md:151-155`, `docs/quickstart.mdx:30-35`).
- **`vm.max_map_count ≥ 262144`** — hard requirement; default is 65530. Full per-platform instructions at `docs/quickstart.mdx:45-181`; on macOS it needs a `--privileged --pid=host` container (`:88`). `docs/faq.mdx:429` — "If your container keeps restarting, ensure `vm.max_map_count` >= 262144."
- **x86-only.** `README.md:193-195`: "All Docker images are built for x86 platforms. We don't currently offer Docker images for ARM64." A real tax on a Mac-based team.
- **Image is ~2 GB** (`docs/develop/build_docker_image.mdx:29`). The old slim/full split is gone — a repo-wide grep for `LIGHTEN` returns zero hits. It got small by shipping **no embedding model** and **installing torch lazily at runtime** (`deepdoc/vision/ocr.py:84` `pip_install_torch()`; `task_executor.py:1917` `check_and_install_torch()`). Runtime `pip install` inside a container is an ops smell CRED should not inherit.
- **`onnxruntime-gpu` is a hard dependency on Linux x86_64**, not an extra (`pyproject.toml:78-79`). There is no `[project.optional-dependencies]` section at all.
- ONNX models are **not in git** — `rag/res/deepdoc/` is fetched on first use via `snapshot_download` from **huggingface.co** (`ragflow_deps/download_deps.py:114-122`), mirrorable via `HF_ENDPOINT` (`docker/.env:194`).
- gVisor required if the code-executor sandbox is used (`README.md:155`).

---

## 6. Agent / Workflow Canvas vs RAG Core

| Dir | Python | Go | TS/TSX |
|---|---|---|---|
| `internal/` | — | **491,961** | — |
| `web/` | — | — | **210,727** (1,279 files) |
| `test/` | 104,070 | — | — |
| `rag/` | 65,411 | — | — |
| `api/` | 47,880 | — | — |
| `common/` | 31,505 | — | — |
| `agent/` | 18,071 | — | — |
| `deepdoc/` | 15,869 | — | — |
| `sdk/` | 1,855 | — | — |
| `mcp/` | 1,011 | — | — |

**DeepDoc is ~11% of the Python backend and ~2% of the repo.** The agent canvas (`agent/`, 18k) is only ~28% the size of the RAG core — but `web/` at 210k LOC dwarfs both, and the Go agent runtime (`internal/agent/`, 40,592 src) is already **2.2× larger than the Python `agent/`**, indicating the agent subsystem is furthest along in the migration.

### Is the core extractable? No — and this is decisive.

**`rag/` and `api/` are circularly dependent.** 32 of 181 files in `rag/` import `api.*`; 28 files in `api/` import `rag.*`. `rag/svr/task_executor.py:38-83` imports ~15 `api.db.*` modules — `KnowledgebaseService`, `DocumentService`, `TaskService`, `LLMBundle`, `File2DocumentService`, `api.db.db_models.close_connection`, and more. `rag/flow/` is additionally wired into the agent canvas (4 files). There is no layered "RAG core" behind an interface; there is one application.

The chunkers (`rag/app/*`) and `deepdoc/` are the *least* entangled parts — but see below.

The clean win: `deepdoc/server/` is a **standalone LitServe microservice** with 9 dependencies, no DB, no ES, CPU-only (`deepdoc/server/pyproject.toml`), exposing `/predict/dla`, `/predict/tsr`, `/predict/ocr` on port 9390 (`deepdoc/server/README.md`). That genuinely lifts out.

The catch: it serves **raw models only** — image in, boxes out. All the value-adding orchestration (merge, downward-concat, table/figure extraction, position tagging) is in the 2,145-line `deepdoc/parser/pdf_parser.py`, which does *not* lift out cleanly. Its cross-module imports:

```
10× rag.nlp   ·  8× common.constants  ·  5× rag_tokenizer  ·  5× common.file_utils
4× rag.utils.lazy_image  ·  3× common.misc_utils  ·  2× rag.prompts.generator
1× rag.app.picture  ·  1× api.db.services.llm_service  ·  1× api.db.joint_services.tenant_model_service
```

`deepdoc/parser/figure_parser.py:22-23` imports `api.db.services.llm_service` and `api.db.joint_services.tenant_model_service` — i.e. **DeepDoc reaches into the Peewee ORM and multi-tenant model registry.** Vendoring DeepDoc means vendoring or stubbing a chunk of RAGFlow's tenancy layer.

---

## 7. Interfaces

**MCP server** (`mcp/server/server.py:535-648`) — exactly **three tools**, all read-only:

| Tool | Required | Notable params |
|---|---|---|
| `ragflow_retrieval` | `question` | `dataset_ids[]`, `document_ids[]`, `page`, `page_size` (≤100), `similarity_threshold` (0.2), `vector_similarity_weight` (0.3), `keyword`, `top_k` (≤1024), `rerank_id`, `force_refresh` |
| `ragflow_list_datasets` | — | `page`, `page_size` (≤1000) |
| `ragflow_list_chats` | — | `page`, `page_size` (≤100) |

**There is no ingestion tool over MCP.** Upload/parse must go through the REST API. Auth is a bearer token via middleware (`server.py:718-731`). `_map_chunk_fields` (`:417-438`) preserves all raw API fields and adds `dataset_name`/`document_name`/`document_metadata`, so `positions` does pass through to an MCP client.

There is also a **second, Go MCP server** at `internal/mcp/server.go` exposing the same three tool names (`:150`, `:222`, `:244`) — more rewrite duplication.

**REST API:** `api/apps/restful_apis/` (29 modules, 15,561 LOC). The endpoints CRED would care about:

- `POST /datasets`, `GET/PUT/DELETE /datasets/<id>` (`dataset_api.py:81,220,384,161`)
- `POST /documents/upload` (`document_api.py:99`), `POST /documents/ingest` (`:1414`)
- `POST /datasets/<id>/documents/parse` (`:1491`), `POST .../stop` (`:1604`)
- `GET/POST/PATCH/DELETE /datasets/<id>/documents/<doc_id>/chunks` (`chunk_api.py:441,842,1072,931`)
- **`POST /retrieval`** (`chunk_api.py:311`) — the main retrieval entry point
- Compatibility shims: `dify_retrieval_api.py`, `openai_api.py` (OpenAI-compatible chat completions)

**Python SDK** (`sdk/python/ragflow_sdk/`): `RAGFlow(api_key, base_url)` (`ragflow.py:27`) with `create_dataset`/`list_datasets` (`:56,98`), `retrieve` (`:187`), and module classes `DataSet.upload_documents`/`parse_documents`/`async_parse_documents` (`modules/dataset.py:54,150,144`), `Document.list_chunks`/`add_chunk` (`modules/document.py:78,90`).

**Bottom line on embedding RAGFlow as a component:** it is HTTP-only. There is no library mode, the ingestion path is not exposed over MCP, and `rag/` cannot be imported without `api/`. CRED would be operating a RAG platform and integrating over REST.

---

## 8. OSS Project Mechanics

- **License: Apache 2.0, clean.** No Commons Clause, no BSL, no user/seat caps, no telemetry gate. Grep for enterprise gating in `api/`, `rag/`, `common/` finds only three connector-level stubs — `common/data_source/confluence_connector.py:971,1006` and `common/data_source/github/utils.py:14` ("This functionality requires Enterprise Edition"). **The RAG core, DeepDoc, and retrieval are fully OSS.** For CRED's purposes the license is a non-issue; copying the citation schema is unambiguously fine (attribution + NOTICE).
- **OSS-vs-cloud boundary** sits at data-source connectors and managed hosting, not at the retrieval or parsing core — an unusually honest split.
- **Repo size:** 158 MB. `web/` 65 MB, `internal/` 20 MB, `test/` 4.8 MB, `rag/` 3.6 MB, `api/` 2.2 MB, `deepdoc/` 2.0 MB, `agent/` 2.0 MB.
- **Community health: could not be measured from this checkout** — it is a shallow clone (`.git/shallow` present, single commit in history). Commit velocity and contributor counts here are artifacts, not signal. Assess on GitHub directly before relying on this.
### Strategic risk — the Go rewrite

This is the single largest reason not to build on RAGFlow.

`internal/` is **491,961 LOC of Go**: a functionally-overlapping reimplementation of the API server, admin server, ingestion worker, agent runtime, parser, MCP server, and CLI. Both stacks ship in the same image and are selected at runtime by `API_PROXY_SCHEME` (`docker/entrypoint.sh:190-205`):

| Value | What starts |
|---|---|
| `python` (**default**, `:204`) | Python API + Python task executors (`:288`, `:321`) |
| `go` | `bin/ragflow_server --api` + `--ingestor` (`:297`) |
| `hybrid` | **Both**, routed per-endpoint via `ragflow.conf.hybrid` |

**The two ingestion paths do not even share a broker.** Python uses Redis/Valkey Streams (`te.{0,1}.common`); Go uses **NATS JetStream** (`tasks.RAGFLOW`, `internal/ingestion/service/ingestion_service.go:121`, `internal/engine/nats/nats.go:29-89`), with the `nats:2.14.2` service gated behind the `ragflow-go` compose profile. They are not interchangeable for the same workload.

**The Go ingestor is visibly incomplete**: `cmd/ragflow_server.go:501` hardcodes concurrency to 2 and the supported types to `[]string{"pdf", "docx", "txt"}`.

`internal/harness/` (41k src + 46k test) exists purely to diff Go output against Python output — a project spending 87k LOC on migration scaffolding is a project in motion.

Policy statements that disclaim stability: `CLAUDE.md:54-58` — *"Treat `internal/ingestion`, `internal/parser`, and `internal/deepdoc` as actively refactored code… Do not add or preserve deprecated Go APIs just to ease migration."* And `CLAUDE.md:6-8` — *"Treat legacy code as liability, not as a compatibility target."*

The Go side already reimplements retrieval with matching constants (`internal/service/nlp/retrieval.go:93-94,163-167,664`) and the citation model (`internal/ingestion/component/schema/chunk_types.go:125-126` — `_pdf_positions`, `positions`). Building it needs CMake, clang-20, and prebuilt static libs (pdfium, `office_oxide`, `pdf_oxide`) via CGO (`internal/development.md:44-60`).

**Counter-evidence, in fairness:** Python is still the default and `docs/` contains zero mention of the Go server — every user-facing doc still describes the Python path. The Go path is developer-only today. But "developer-only today, 492k LOC, explicitly no compatibility promise" is exactly the profile of a dependency that breaks under you in 12 months.

---

## Recommendation

### **(c) Use Docling. Steal RAGFlow's citation data model. Do not run RAGFlow.**

Firm. The reasoning:

1. **RAGFlow's own architecture is the argument.** `rag/app/naive.py:373-382` makes DeepDoc one of eight swappable backends, and ships a 628-line Docling adapter. If DeepDoc's quality were decisive, that dict would not exist.

2. **Option (a) — RAGFlow as a service — is disqualified on ops and stability.** 16 GB RAM, 6+ containers, a ~7.5 GiB-per-container search engine, MySQL, MinIO, Valkey, x86-only images, `vm.max_map_count` tuning, gVisor for sandboxing, and runtime `pip install torch` inside the container. The MCP surface is 3 read-only tools with **no ingestion tool**, so CRED couldn't even drive ingestion from an agent — it would be REST-integrating against a platform that is simultaneously mid-Python-refactor (`TE_RUN_MODE` triple path) and mid-Go-rewrite (492k LOC, different message broker).

3. **Option (b) — vendor DeepDoc — is disqualified on coupling and drift.** `deepdoc/parser/figure_parser.py:22-23` imports the tenant model registry; `pdf_parser.py` pulls `rag.nlp`, `rag_tokenizer`, `rag.prompts.generator`, `common.*` — and `rag/` is itself circularly bound to `api/` (32 files out, 28 back). The cleanly extractable part (`deepdoc/server/`, 9 deps) is *only the model server* — CRED would still have to rebuild the 2,145-line orchestration that turns boxes into positioned text. And `CLAUDE.md:54-58` marks that exact code as under active, compatibility-breaking refactor toward Go.

4. **Docling gives CRED the same citation primitive natively.** RAGFlow's own adapter is the proof: `_extract_bbox_from_prov` (`deepdoc/parser/docling_parser.py:72-87`) pulls `prov[0].page_no` and `prov[0].bbox` straight off Docling's document model, and `_make_line_tag` (`:153-160`) emits the *identical* `@@page\tx0\tx1\ttop\tbott##` tag. Tables and figures get bboxes too (`:295-308`). **Docling → `position_int` is a solved problem in ~90 lines, and someone already wrote it in this repo under Apache 2.0.**

### Docling vs DeepDoc, head to head

| | DeepDoc | Docling |
|---|---|---|
| **Quality** | 11-class layout + dedicated TSR ONNX + XGBoost line-merge + genuinely good garbled-font/CID fallback (`pdf_parser.py:1631-1656`). Strong on CJK and on scanned/degraded scans. | Comparable or better on Western digital PDFs; native structured `DoclingDocument` with reading order, tables, and provenance. Weaker CJK-scan heritage. |
| **Cost** | Rasterizes + runs layout ONNX + det ONNX on **every** page unconditionally (`pdf_parser.py:1622,773`). No fast path for clean digital PDFs. | Configurable pipeline; can skip OCR on digital PDFs. Cheaper on the common case. |
| **Ops** | 4 ONNX + 1 XGBoost from HF; CPU-capable but x86-only in packaged form; standalone server only serves raw models. | `pip install docling`. In-process library. ARM/Apple Silicon fine. |
| **Coupling** | Reaches into `api.db` tenancy (`figure_parser.py:22-23`). | None. |
| **Trajectory** | Being rewritten into Go/CGO; explicitly no API-stability promise (`CLAUDE.md:54-58`). | Independent, library-first, stable public model. |
| **Output for citations** | `@@…##` tags → `position_int`. | `prov[].bbox` + `page_no` → same thing, adapter already written. |

DeepDoc wins on scanned CJK and degraded scans. That is not CRED's corpus. CRED ingests ADRs, design docs, RFCs, runbooks, tickets — mostly digital PDFs, Markdown, and Office formats, where DeepDoc's advantage evaporates and its "OCR everything" floor becomes pure waste.

**Caveat to carry forward:** for Markdown/DOCX — likely the *majority* of CRED's corpus — neither tool gives sub-page bboxes, and RAGFlow degrades to fake positions (`rag/nlp/__init__.py:407`). CRED should design a **two-tier citation model** from day one: geometric (page + bbox) where the format supports it, structural (heading path + char offset range) everywhere else. RAGFlow never did this, and it shows.

---

## Top 3 Things to STEAL

### 1. The `position_int` citation model, end to end
`rag/nlp/__init__.py:922-934` → `rag/nlp/search.py:689,709` → `web/src/utils/document-util.ts:8-41`.

Three denormalized fields from one source, each with a distinct query role — `page_num_int` (filter), `top_int` (reading-order sort), `position_int` (render). A `[[page, x0, x1, top, bottom]]` array of int 5-tuples in PDF-point space, resolution-independent, rendered as highlight rects with ~15 lines of frontend. Multi-page spans handled by `pn` being a *list* (`pdf_parser.py:1527-1533`). Copy this schema verbatim; add a structural tier for non-PDF.

### 2. The inline sentinel-tag technique
`deepdoc/parser/pdf_parser.py:1522-1535` (emit), `:1937-1944` (`extract_positions` / `remove_tag`), `rag/nlp/__init__.py:391-405` (recover at tokenization).

Encoding position *inside the text* as `@@page\tx0\tx1\ttop\tbott##` means every chunker, splitter, and merger can treat content as plain strings — no parallel coordinate array to keep in sync through arbitrary text surgery. Coordinates are recovered once, at the end. This is the single cleverest idea in the codebase and it generalizes to any chunking strategy CRED invents.

### 3. Suffix-driven dynamic index mapping
`conf/mapping.json:25-215`.

Field type inferred from name suffix: `*_int`, `*_flt`, `*_kwd`, `*_tks`/`*_ltks`, `*_with_weight`, `*_512_vec`…`*_1536_vec`, `*_fea`/`*_feas`. Adding a field requires **zero** mapping migration. For CRED — where the chunk schema will churn hard early — this eliminates an entire class of index-migration pain.

**Honourable mention:** the chunk-snapshot fallback (`d["image"]` cropped with 120px of context, `pdf_parser.py:1984,1998-1999`, stored via `image2id` at `task_executor.py:389-398`). A raster thumbnail is a cheap, robust citation UX when bbox math goes wrong.

---

## Top 3 Things to AVOID

### 1. Running RAGFlow, or anything shaped like it, as infrastructure
16 GB RAM, ~7.5 GiB per search-engine container (`docker/.env:65`), 6+ containers, MySQL + MinIO + Valkey, seven exposed ports, x86-only (`README.md:193-195`), `vm.max_map_count` kernel tuning, gVisor for the sandbox, and `pip install torch` at container runtime. CRED's document ingestion is a *feature*, not a platform. Adopting this makes RAGFlow's operational surface CRED's on-call burden.

Two concrete reliability traps to note even if CRED only borrows the design: the unconditional `redis_msg.ack()` outside the try/except (`rag/svr/task_executor.py:1785`), which silently drops failed work; and `retry_count` incremented on *every* claim including successes (`api/db/services/task_service.py:230`), which will fail healthy documents after three touches. Whatever queue CRED builds, ack on success only, and separate "attempts" from "failures."

### 2. GraphRAG and RAPTOR at ingestion time
GraphRAG: 3–5 LLM calls/chunk from the gleaning loop, plus per-entity description summaries, plus **quadratic** entity resolution (`entity_resolution.py:101-145`), plus one call per community per Leiden level — with `with_resolution=True, with_community=True` defaulted **on** (`general/index.py:256`). RAPTOR adds ~N/4–N/2 LLM calls plus equal embeddings per document, ×3 with retries. These make per-document ingest cost unpredictable and unbounded. If CRED wants graph structure, derive it from code and ticket metadata it already owns — structured, free, and more accurate than LLM entity extraction over prose.

### 3. The chunk-template zoo, and hardcoded/duplicated scoring constants
Fifteen templates, 7,122 lines, one of which is an alias for another (`task_executor.py:129`) and one of which is a 2,768-line resume parser with bundled Chinese school and corporation gazetteers. Manual per-KB selection, no auto-detection. Ship one good configurable chunker.

Separately, resist RAGFlow's scoring sprawl: fusion weights hardcoded at `"0.05,0.95"` (`search.py:210`), rerank defaulting to 0.3/0.7, citation insertion using 0.1/0.9 (`search.py:251`), and rank features added *on top* with a ×10 multiplier (`search.py:330-361`) that can silently swamp the similarity score. Three different weightings of the same idea in one file is a tuning nightmare. Pick one blend, put it in config, log it.

**Also avoid:** the leaky doc-store abstraction. `DocStoreConnection` is a proper ABC (`common/doc_store/doc_store_base.py:148`), yet `search.py` still branches on `DOC_ENGINE_INFINITY`/`DOC_ENGINE_OCEANBASE` (`:207,625,631`) and calls `get_scores()`, which isn't on the interface. If CRED abstracts its store, enforce the boundary in tests or don't bother.
