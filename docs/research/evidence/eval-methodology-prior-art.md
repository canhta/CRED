# Evidence: evaluation methodology in memory-system research

- **Date:** 2026-07-20
- **Scope:** what the memory-system field actually does for model configuration,
  baselines, judging, cost accounting, and benchmark choice — read from
  evaluation *code* where it exists, and from papers where it does not.
- **Commissioned by:** the open question in `v0/preregistration.md` §9 on
  extended thinking. Section 8 states the decision; everything before it is the
  scan that produces it. **`v0/` was deleted 2026-07-21** (D-024 in
  `../decision-log.md`) before the experiment ran; the commissioning question
  and this document's findings are unaffected, but the source path no longer
  resolves.

This is an evidence document. It records what is there. The one place it
argues — §8, the extended-thinking decision — is marked as such, because the
commissioning question demanded a recommendation.

**Headline:** the standard practice CRED wanted to cite does not exist. Across
six repositories (2,218 Python/Rust source files) and eight papers, **zero**
enable a reasoning model or thinking mode in an evaluation, and zero mention
one. CRED cannot cite prior art for this decision. It has to make it and
justify it, which is what §8 does.

---

## 1. What was checked

### Repositories read

Every row is a checkout whose evaluation code (or absence of it) was read
directly. Commit hashes are from `git log -1` at scan time.

| Repo | Location | Commit | Date | Eval code? |
|---|---|---|---|---|
| Mem0 | `/Users/canh/Solo/OSS/mem0` | `9383e9a2556a533d289a8aa041c7a6660e806581` | 2026-07-20 | **No** — submodule, uninitialized |
| Mem0 benchmarks | scratchpad clone of the pinned submodule | `4b61c5d31b9c668a12b4f5e78064248a02c82d2b` | 2026-05-13 | Yes |
| Letta / MemGPT | `/Users/canh/Solo/OSS/letta` | `b76da9092518cbaa2d09042e52fdcbde69243e18` | 2026-07-03 | **No — none at all** |
| Graphiti (Zep) | `/Users/canh/Solo/OSS/graphiti` | `0b4bcf1284ee5fba56b77ed9961568a541e0d418` | 2026-07-17 | Yes, ~230 LOC |
| Zep | scratchpad clone | `0375d7be4a72cda6a43ecdc6fd9055846eb0fd0e` | 2026-07-17 | Yes, three harnesses |
| Cognee | scratchpad clone | `6df1eda76adbcb1111d6d45ac084a4a3aef4f908` | 2026-07-18 | Yes |
| LongMemEval | scratchpad clone | `9e0b455f4ef0e2ab8f2e582289761153549043fc` | 2026-05-11 | Yes |
| LoCoMo | scratchpad clone | `3eb6f2c585f5e1699204e3c3bdf7adc5c28cb376` | 2024-08-12 | Yes |
| MemBench | `github.com/import-myself/Membench` | `f66d8d1028d3f68627d00f77a967b93fbb8694b6` | 2025-11-27 | Partial — does not run |
| pacabench | scratchpad clone | `f6674a54d4581b76bdd0b2d450ff952ab9046c8f` | 2026-02-03 | Yes |

`langfuse`, `onyx`, and `ragflow` were not scanned. They are observability and
RAG products, not memory systems making a beat-long-context claim, and the
commissioning question is about the latter.

### Papers fetched

Mem0 (2504.19413), Zep/Graphiti (2501.13956), MemGPT (2310.08560, abstract
only), LongMemEval (2410.10813), LoCoMo (2402.17753), MemBench (2506.21605),
MemoryAgentBench (2507.05257, abstract only), and a Cognee-adjacent
hyperparameter paper (2505.24478). Plus the Mem0↔Zep dispute artifacts and one
independent LoCoMo audit.

### What could not be checked, and why

- **MemGPT and MemoryAgentBench full texts.** Abstracts only. Any claim about
  their internal configuration is UNVERIFIED and is labelled as such below.
- **LongMemEval's actual dataset files.** Not in the repo — `data/` holds only
  the sampling script (`data/custom_history/sample_haystack_and_timestamp.py`).
  The widely-quoted 115k-token figure is a README claim
  (`README.md:75-76`), not a constant in code. UNVERIFIED at the byte level.
- **MemBench's data JSONs.** 60–70 MB LFS blobs that did not fetch. Token
  statistics derive from one file that did (`ThirdAgent/comparative.json`).
- **Mem0's published LoCoMo numbers.** The result file references a harness
  feature (`merged_from_questions`) with zero call sites in the code at that
  SHA. See §6.
- **Cognee's BEAM question text.** Loaded from HuggingFace at runtime, not
  checked in.
- **Letta's published benchmark numbers.** Not reproducible from the repo,
  because the repo contains no benchmark.

---

## 2. Extended thinking and reasoning effort — the finding is absence

**VERIFIED, and it is a null result.** A case-insensitive grep for
`reasoning_effort|budget_tokens|extra_body|"thinking"|'thinking'` across all
scanned clones (2,218 `.py`/`.rs` files, excluding `.git`) returns **zero hits
in any evaluation code path**. The only matches anywhere are unrelated: an
ElevenLabs proxy's `elevenlabs_extra_body`
(`clones/zep/examples/python/elevenlabs-zep-example/llm-proxy/proxy_server.py:344`),
a Google-ADK callback comment
(`clones/zep/integrations/adk/python/src/zep_adk/callbacks.py:71`), and a
cognee unit test asserting an unrelated `strict` flag
(`clones/cognee/cognee/tests/unit/infrastructure/llm/test_get_llm_client.py:43`).

Per-system detail:

- **Mem0 benchmarks.** No `reasoning_effort`, no `extra_body`, no
  `budget_tokens`. The `thinking` matches are prompt-scaffolding XML tags
  (`benchmarks/longmemeval/prompts.py:53,65,155,342` use `<mem_thinking>`), not
  API parameters. The request bodies are fully visible at
  `benchmarks/common/llm_client.py:168-173` and `:268-276` and contain only
  `model`, `messages`, and token/temperature kwargs.
- **Letta.** `reasoning_effort` is a first-class *product* setting —
  `letta/schemas/model.py:74` declares
  `Literal["none","minimal","low","medium","high","xhigh"]`, and
  `letta/schemas/model.py:236` defaults `OpenAIReasoning(reasoning_effort="high")`.
  But there is no eval code to apply it in. It is a product knob, not an
  experimental control.
- **Graphiti.** Reasoning plumbing exists in the library
  (`graphiti_core/llm_client/openai_base_client.py:39` sets
  `DEFAULT_REASONING = 'auto'`, gated on `gpt-5`/`o1`/`o3` prefixes at
  `graphiti_core/llm_client/openai_client.py:77-78`) and the eval never touches
  it — both arms hardcode `gpt-4.1-mini`
  (`tests/evals/eval_e2e_graph_building.py:107` and `:127`), which is not a
  reasoning model.
- **Papers.** Not one of the eight mentions o1, o3, R1, extended thinking, or a
  reasoning mode, even in passing. This is not "tried and rejected" — it is
  total absence. The literature is built on non-reasoning models, predominantly
  `gpt-4o-mini`.

Note a second-order effect in Mem0's harness that is closer to CRED's problem
than anything the papers discuss. `benchmarks/common/llm_client.py:78-83`:

```python
def _openai_chat_temperature_kwargs(self, temperature: float) -> dict[str, Any]:
    """gpt-5 / o-series only accept the default temperature (1); omit the param for those models."""
    m = self.model.lower()
    if m.startswith(("gpt-5", "o1", "o3", "o4")):
        return {}
    return {"temperature": temperature}
```

The shipped default answerer and judge are both `gpt-5`
(`benchmarks/locomo/run.py:682-683`). So the declared `temperature=0` is
**silently dropped** and the benchmark runs at OpenAI's default of 1.0. The
field has already met CRED's exact problem — a reasoning-family model rejecting
sampling parameters — and handled it by looking away.

> **Consequence for CRED.** The pre-registration cannot cite convention here,
> because there is none. Whichever way §8 goes, it is a first, and it must be
> argued rather than referenced.

---

## 3. The long-context baseline — mostly absent, and unconfigured where present

This was the crux question, and the answer is worse than "handicapped."

**In three of the four vendor systems, the long-context baseline does not exist
in the code at all.**

- **Mem0.** No eval code for a full-context arm. Searched `full.?context`,
  `long.?context`, `baseline`, `naive`, `no.?memory`, `raw.?conversation`,
  `whole.?conversation` across the whole benchmark repo. Every hit is prose or
  a UI placeholder. Every answer path is gated behind `mem0.search(...)`
  (`benchmarks/locomo/run.py:418`) with no bypass branch. The comparison this
  harness supports is Mem0-config vs Mem0-config. Any long-context number
  attributed to Mem0 was produced by code not present at this SHA.
- **Letta.** N/A — no eval code of any kind.
- **Graphiti.** "Baseline" means *another Graphiti graph* built with a pinned
  model (`tests/evals/eval_e2e_graph_building.py:105-122`), compared against a
  candidate Graphiti graph. It is a self-regression check. The nearest thing to
  a QA arm, `qa_prompt` at `graphiti_core/prompts/eval.py:80-99`, is **dead
  code with zero call sites** — and it is memory-fed, not transcript-fed, so it
  would not be a long-context arm even if it ran.
- **Cognee.** No long-context implementation. All seven strategies in
  `beam/eval/registry.py:32-68` are cognee retrievers. The only `full_context`
  code is arithmetic with no LLM calls
  (`token_usage_analysis/cost_model.py:46-54`). Its own `REPORT.md:92,108,152`
  concedes the baseline numbers are not reproducible from the repo.

Where a real baseline **does** exist, it is unconfigured or truncating:

- **pacabench** — the harness behind the 14–77x claim CRED is answering. Its
  long-context agent is
  `examples/membench_qa_test/agents/long_context_agent.py:50-53`: a bare
  `chat.completions.create(model="gpt-4o-mini", messages=messages)`. **No
  temperature, no seed, no max_tokens, no truncation logic anywhere in the
  repo.** The system prompt is one line — `"You are a helpful assistant with a
  long memory."` (`:35`). The memory arms it is compared against are configured
  products. This is not a tuned baseline; it is a default-everything call.
- **MemBench** — `FullMemory` truncates by **whitespace word count, keeping the
  head**: `benchmark/memory/CommonMemory.py:1002-1003` is
  `' '.join(memory_context.split(' ')[:self.max_words])`. In the noise-injected
  long setting this silently drops answer-bearing messages that landed late.
  The `max_words` value is not in the repo (`benchutils.py`, `utils.py`, and
  `makenoise.py` are absent from `git ls-files` despite being imported at
  `benchmark/MembenchAgent.py:3`), so the configuration that produced the
  paper's numbers was never committed.
- **LongMemEval** — truncation is real and head-keeping
  (`src/generation/run_generation.py:266-279`), budgeted at
  `model_max_length - gen_length - 1000` (`:343`).
- **LoCoMo** — stuffs the full conversation to `MAX_LENGTH[model]`
  (`task_eval/gpt_utils.py:15,189`) at `temperature=0` (`:289,312`), but
  **batches 20 questions into a single call** (`scripts/evaluate_gpts.sh:7`,
  prompt at `gpt_utils.py:42-46`). That is a significant confound: the baseline
  answers questions under interference the memory arms do not face.
- **Zep's LongMemEval harness** is the one clean case found. A genuine
  full-context baseline with no truncation
  (`benchmarks/longmemeval/zep_longmem_eval.py:376-418`).

Two things cut *against* the "weak baseline" narrative and must be recorded,
because deleting inconvenient findings is the failure mode this project
documents:

1. **Mem0's paper baseline is not handicapped.** VERIFIED at
   [arxiv.org/html/2504.19413v1](https://arxiv.org/html/2504.19413v1): *"passing
   the entire conversation history within the context window of the LLM"*, same
   model (`gpt-4o-mini`), ~26k tokens. Fair, though not prompt-tuned.
2. **Zep's own paper reports the full-context baseline winning.** VERIFIED at
   [arxiv.org/html/2501.13956v1](https://arxiv.org/html/2501.13956v1): *"Using
   gpt-4-turbo, the full-conversation baseline achieved 94.4% accuracy"* and
   *"98.0% for full-conversation"* on DMR. A memory vendor published a baseline
   that beat its own system on one benchmark.

**Prompt caching on the baseline: nobody does it.** No `cache_control` and no
`cached_tokens` read in Mem0's harness (verified across
`benchmarks/common/llm_client.py:159-173`, `:199-206`, `:260-276`, `:326-333`),
none in Graphiti's, none in Cognee's, no mention in any paper. pacabench is the
sole partial exception: `pacabench-cli/src/pricing.rs:17-25,60-64` prices
cached input and `proxy.rs:60-64` reads `prompt_tokens_details.cached_tokens` —
but that *observes* OpenAI's automatic caching, it never requests it.

> **CRED's D6 (caching ON for arm A) has no precedent and is a genuine
> methodological contribution.** So is publishing arm A's config at all.

---

## 4. Temperature, seeds, and n-runs

| System | Temperature | Seed | Runs/item | Variance reported |
|---|---|---|---|---|
| Mem0 (code) | `0` declared (`llm_client.py:140,230`), **dropped for gpt-5 → effective 1.0** | None sent to any API | **1** | **None** |
| Mem0 (paper) | 0 | not stated | **10** | **Yes — mean ±1 SD** |
| Graphiti | **1.0** (inherited `DEFAULT_TEMPERATURE = 1`, `llm_client/config.py:20`) | None | **1** | **None** |
| Zep (locomo) | 0.0 (`benchmark_config.yaml:22-26`) | None | 1 | None |
| Cognee | not set in eval | not set | **1** (`sweeps/retriever_sweep_runner.py:27`) | None |
| LongMemEval | `temperature: 0`, `n: 1` (`run_generation.py:360-368`) | Dataset sampling only | 1 | None |
| LoCoMo | `temperature=0` | None | 1 | None |
| MemBench | from absent config file | None (unseeded `np.random` noise placement) | 1 | None |
| pacabench | **not set** (API default 1.0) | None | **1** | None |

**Mem0's paper is the field's ceiling and it is not close to CRED.** VERIFIED
quote: *"we conducted 10 independent runs for each method on the entire dataset
and report the mean scores along with ±1 standard deviation"* (e.g.
`67.13 ± 0.65`). That is the only system in the survey that reports dispersion
at all. Every other paper publishes bare point estimates.

The gap between Mem0's paper (10 runs, SD) and Mem0's shipped harness (1 run,
no variance code — grep for `stdev|variance|confidence|bootstrap` returns zero
matches in `benchmarks/`) is itself the finding. **The published protocol and
the running code are different artifacts.** This is the same pattern
`docs/research/evidence/mem0.md` found with the vestigial reconciler prompt.

Graphiti's case is the most severe: **temperature 1.0, no seed, one run, on a
pairwise-preference metric, reported as a single unreplicated float**
(`tests/evals/eval_e2e_graph_building.py:176-180`).

**On CRED's nondeterminism problem:** nobody has solved it because nobody has
hit it. Every system in the survey uses OpenAI models that still accept
`temperature`, so the parameter works for them and the question of what to do
when it 400s has not arisen — except in Mem0's harness, where it *has* arisen
(gpt-5) and the code silently drops the parameter without adjusting the
analysis. **CRED's response — log `temperature: null`, register the deviation
before the tag, and gate on a flip rate — is stricter than anything observed.**

---

## 5. The judge

| System | Judge model | Same family as SUT? | Arm-blind? | Human agreement |
|---|---|---|---|---|
| Mem0 (code) | `gpt-5`, default same provider | **Yes — gpt-5 judging gpt-5** | **Yes** | **None** |
| Mem0 (paper) | **never named** | unknown | unknown | **None** |
| Graphiti | `gpt-4.1-mini`, **same client object as the candidate builder** | **Yes — grades its own output** | **No — arms labelled `<BASELINE>`/`<CANDIDATE>`** | **None** |
| Zep (locomo) | `gpt-4o-mini` | n/a | not blinded | None |
| Zep (harness) | Gemini, **judge == response model** | Yes | — | None |
| Cognee | ambient `LLMGateway`, default `openai/gpt-5-mini` — same model that answers | **Yes** | Yes (structurally) | **None** |
| LongMemEval | `gpt-4o-2024-08-06` | — | — | **>97% raw agreement** |
| LoCoMo | **none** — F1/EM/ROUGE | — | — | n/a |
| MemBench | **none** — multiple choice | — | — | n/a |
| pacabench | `gpt-4o-mini`, opt-in only | — | No | None |

**A pre-registered κ ≥ 0.70 gate is unheard-of.** Exactly one paper validates
its judge against humans, and it reports **raw agreement, not a chance-corrected
statistic**: LongMemEval, VERIFIED at
[arxiv.org/html/2410.10813v1](https://arxiv.org/html/2410.10813v1) — *"Our
meta-evaluation study demonstrates that the evaluator achieves more than 97%
agreement with human experts"*. A grep for `kappa|cohen|inter.?rater` across
every scanned repo returns zero relevant hits. The single `kappa`-adjacent
function found — `benchmarks/common/metrics.py:145-157`, `compute_kendall_tau_b`
— is a rank correlation for event-ordering questions, not judge validation.

So κ ≥ 0.70 is not "stricter than average." It is **stronger than the field's
single best instance**, since even LongMemEval validates post hoc with an
uncorrected metric. Note the flip side honestly: raw agreement above 97% on a
skewed binary label can coexist with a mediocre κ, so LongMemEval's number is
less impressive than it reads — which strengthens rather than weakens the case
for a chance-corrected gate.

**Judge leniency is the field's live failure mode, and it is written into the
prompts.** Two VERIFIED examples, both read directly:

- Mem0, `benchmarks/locomo/prompts.py:222`: *"**PARTIAL CREDIT**: If the
  generated answer includes AT LEAST ONE correct item from the gold answer's
  list, mark CORRECT... Only mark WRONG if NONE of the gold answer items
  appear."* Plus `:228` *"Dates within 14 days of each other are CORRECT.
  Durations within 50% are CORRECT"*, and with `--with-evidence`, `:212` adds
  the one-directional ratchet *"Use evidence only to ACCEPT answers, never to
  reject them more strictly."*
- Zep, `benchmarks/locomo/prompts.py:88`: *"you should be generous with your
  grading - as long as it touches on the same topic as the gold answer, it
  should be counted as CORRECT."*

**Correction on the record.** A widely-repeated claim attributes the "be
generous with grading" instruction to *Mem0's* judge. The literature scan could
not verify that and flagged it UNVERIFIED. Direct grep resolves it: the phrase
is **Zep's**, at `clones/zep/benchmarks/locomo/prompts.py:88` and `:90`.
Mem0's judge is lenient by a different mechanism (partial credit + date
tolerance, quoted above). Attributing the sentence to Mem0 is FALSIFIED.

Third-party corroboration that this matters, weighted as a non-peer-reviewed
vendor blog: an independent LoCoMo audit
([dev.to/penfieldlabs](https://dev.to/penfieldlabs/we-audited-locomo-64-of-the-answer-key-is-wrong-and-the-judge-accepts-up-to-63-of-intentionally-33lg))
reports **6.4% of the answer key is wrong** (99 score-corrupting errors in
1,540 questions) and that deliberately wrong-but-topically-adjacent answers
were accepted by the published judge config **62.81%** of the time.

CRED's design is on the right side of all of this — atomic checkpoints, string
matching first, arm-blind grading — but see §9 for one gap in the MVE.

---

## 6. Cost accounting

**Almost nobody measures it.** In Mem0's benchmark harness,
`benchmarks/common/schema.py:32-33` declares `prompt_tokens` and
`completion_tokens` on `GenerationData` and **neither field is ever assigned
anywhere in the codebase**; `resp.usage` is never read (response handling at
`llm_client.py:176`, `:211`, `:338` reads only content). No tiktoken, no cost
math. The only instrumentation is latency, and it times *only the Mem0 search
call* (`benchmarks/locomo/run.py:418,435`) — not generation. Graphiti has a
`token_tracker.py` whose only consumer is its own unit test; the eval never
instantiates it. Cognee, LongMemEval (which sums and prints token totals but
computes no cost), LoCoMo, and MemBench have no dollar accounting.

**pacabench has the best cost model in the survey** — real pricing including
cached-input rates (`pacabench-cli/src/pricing.rs:17-25,60-64`). The 14–77x
claim comes from the one harness that actually counts. It is also the harness
whose baseline sets no temperature and whose comparisons are single-shot.

**What is counted, when anything is:** read-path only. Mem0's paper defines
token consumption as *"the number of tokens extracted during retrieval that
serve as context for answering queries"* — VERIFIED at
[arxiv.org/html/2504.19413v1](https://arxiv.org/html/2504.19413v1). The
write path (extraction, consolidation, per-turn ingestion calls) is excluded
entirely. The *"saves more than 90% token cost"* claim (1,764 vs 26,031 tokens)
therefore compares the memory system's **query** cost against the baseline's
**total** cost. No paper in the survey splits ingestion from query. No paper
mentions prompt caching.

> **CRED's dual amortization policy and cache-aware arm A both exceed every
> observed practice.** Full-lifecycle accounting could plausibly reverse the
> sign of Mem0's headline claim, which is worth stating in the writeup.

---

## 7. The benchmarks

| Benchmark | Measures | Context regime | Domain |
|---|---|---|---|
| **LoCoMo** | 1,986 QA over 10 conversations; multi-hop / temporal / open-domain / adversarial | **8,019–16,165 words per conversation** (measured directly from `data/locomo10.json`, 19–32 sessions) ≈ 10–25k tokens | Conversational |
| **LongMemEval** | 500 questions, 6 types + abstention (`src/evaluation/evaluate_qa.py:26-41`) | `_S` ~115k tokens / ~40 sessions; `_m` ~500 sessions (**README claim, `README.md:75-76`** — data not in repo) | Conversational |
| **MemBench** | 4-way MCQ over a synthetic message stream; exact letter match (`benchmark/env/Membenenv.py:71`) | Base trajectories **8–18 messages, ~260–670 tokens**; long context manufactured post hoc by distractor injection, sampling only **3–5 trajectories** at long lengths (`benchmark/load_test_data.py:96-100`) | Conversational |
| **MemoryAgentBench** | retrieval / test-time learning / long-range understanding / selective forgetting | not verified | General language |
| **BEAM** (via cognee) | rubric-scored | shipped runner truncates to **1 session** (`run_beam_eval.py:41-43`, `BEAM_MAX_BATCHES = 1`) | Conversational |

**CRED's concern is VERIFIED with one refinement.** Zero code, software-
engineering, or repository-history tasks exist in any benchmark scanned. The
only non-conversational content found anywhere is cognee's Wikipedia multi-hop
QA (HotpotQA / MuSiQue / 2WikiMultiHop) and a synthetic package-logistics world
(`cognee/eval_framework/benchmark_adapters/logistics_system_utils/`). Neither
is engineering work. **The precise claim is "none target engineering work,"
which is stronger than "all are conversational" and is the version CRED should
publish.**

Two length findings that matter for CRED's corpus sweep:

- **LoCoMo cannot test memory-vs-long-context at all.** At ~10–25k tokens the
  whole transcript fits several times over in any 2026 context window. Zep says
  so itself, and the criticism holds regardless of the score dispute:
  [blog.getzep.com](https://blog.getzep.com/lies-damn-lies-statistics-is-mem0-really-sota-in-agent-memory/)
  — *"The conversations in LoCoMo average around 16,000-26,000 tokens. While
  seemingly long, this is easily within the context window capabilities of
  modern LLMs."* This is the same defect the v0 design identified in the
  fastpaca benchmark's 4,232-token arm, in a different benchmark.
- **Three sources give three different LoCoMo token counts** — the paper says
  9,209.2 average, Mem0 says ~26k, Zep says 16–26k. Direct measurement of
  `locomo10.json` gives 8,019–16,165 *words*. UNRESOLVED; likely differing
  treatment of image captions and session metadata. Anyone quoting a LoCoMo
  token count should say how they counted.

**Bonus finding — Zep's LoCoMo harness silently drops category 5.**
`clones/zep/benchmarks/locomo/evaluation.py:87-89`:

```python
# Skip category 5 as golds are not provided for this category
if qa.get("category") == 5:
    continue
```

Cross-referenced against the data, category 5 is **446 of 1,986 questions
(22%)** — and it is precisely the adversarial/unanswerable set. The reported
0.803 accuracy
(`experiments/experiment_20251207_215609/experiment_summary.json:30-47`)
excludes the hardest slice. This is the mirror image of CRED's abstention
decoys: **the field's benchmarks contain an adversarial category, and vendors
discard it.** Independently, Mem0's co-founder filed
[getzep/zep-papers#5](https://github.com/getzep/zep-papers/issues/5) alleging
Zep's 84% claim *"incorrectly incorporated questions from the adversarial (5th)
category"* via a denominator that excluded them while the numerator included
their correct answers — corrected to **58.44% ± 0.20**, an inflation of ~25.6
points.

---

## 8. Comparison: CRED's protocol against observed practice

Interpretation, not scan. Marked accordingly.

| Dimension | CRED (pre-registered) | Mem0 | Zep/Graphiti | Cognee | LongMemEval | pacabench |
|---|---|---|---|---|---|---|
| Reasoning/thinking mode | **explicit, logged per run** | absent | absent | absent | absent | absent |
| Temperature | 0 where accepted; `null` logged, deviation registered | 0 declared, **1.0 effective** | **1.0** (graphiti) / 0.0 (zep) | unset | 0 | **unset (1.0)** |
| Seeds | logged, never sent (API exposes none) | none | none | none | dataset only | none |
| Runs per item | **3**, escalates to 5 on flip rate >25% | 1 (code) / 10 (paper) | 1 | 1 | 1 | 1 |
| Variance | **bootstrap CI + flip rate, pre-registered** | ±1 SD (paper only) | none | none | none | none |
| Long-context baseline | **arm A, full corpus, caching ON, config published** | **not in code** | not in code (graphiti); real (zep LME) | **not in code** | truncating, head-keeping | **default-everything** |
| Baseline parity | identical prompt template, differs only in memory block | n/a | n/a | n/a | n/a | **none** |
| Judge blindness | **arm-blind, shuffled** | blind (moot — one arm) | **not blind** | blind | — | not blind |
| Judge family | **same family as SUT (MVE)** — see §9 | **gpt-5 judges gpt-5** | **judge == candidate** | **judge == answerer** | independent | — |
| Judge↔human agreement | **κ ≥ 0.70, pre-registered, code-enforced** | none | none | none | 97% raw | none |
| Judge leniency | atomic checkpoints, string-match first | **partial credit, 14-day dates** | **"be generous"** | rubric | per-type prompts | MCQ default |
| Adversarial/abstention items | **15% decoys, confabulation is a headline metric** | — | **category 5 dropped (22%)** | — | has abstention | — |
| Cost: write path | **counted, dual amortization** | **excluded** | not counted | not counted | tokens only | counted |
| Prompt caching on baseline | **ON (D6)** | **off** | off | off | off | observed, not requested |
| Domain | **engineering / repo history** | conversational | conversational | wiki QA | conversational | conversational |
| Pre-registration | **hash-frozen, git-tagged** | none | none | none | none | none |

CRED is stricter than observed practice on every row it shares with the field,
and occupies three rows the field does not have at all (pre-registration,
write-path cost, engineering domain). **The risk this creates is not rigor —
it is that CRED has no comparable prior result to calibrate against.** Every
threshold in the pre-registration is a judgement made without reference class.
That should be said plainly in the writeup rather than presented as strength.

---

## 9. The extended-thinking decision

**This section is a recommendation, not a scan.**

### The evidence does not decide it

There is no prior art. Nobody has run this comparison with a reasoning model on
either arm, so there is no empirical basis for predicting the direction or size
of the effect. Any claim that thinking helps arm A more than arm C is a
mechanism story, not a measurement. **State that honestly rather than dressing
a guess as a finding.**

What the evidence *does* settle is the direction of the bias:

- The mechanism is real and asymmetric. Arm A must locate a few relevant facts
  in ~120k tokens of mostly-irrelevant context. Arm C receives 4,000 tokens
  already selected. Reasoning that is spent on retrieval-from-context helps the
  arm doing retrieval-from-context.
- Therefore **disabling thinking is the pro-CRED choice.** It removes a
  capability that disproportionately serves the arm CRED needs to beat. The
  current harness setting (`CRED_V0_THINKING=disabled`,
  `harness/providers/anthropic_provider.py:48`) is the configuration most
  favourable to CRED's own hypothesis. That is the fact that should drive the
  decision, and it is stated backwards in the current §9 deviation, which
  justifies `disabled` purely on variance-control grounds without noting which
  way the thumb is on the scale.
- Adaptive thinking is **on by default** on `claude-sonnet-5`. VERIFIED against
  the current Anthropic API reference: omitting the `thinking` field runs
  adaptive; `{"type": "disabled"}` is accepted but is an explicit opt-out.
  **"Off" is the deviation from the platform default, not "on."** A reader will
  reasonably ask why the memory-system vendor disabled the model's default
  reasoning in the experiment meant to validate memory systems.

### Recommendation

> **Turn adaptive thinking ON, identically on every arm — A0, A, and C-both.
> Set it explicitly (`thinking: {"type": "adaptive"}`), log the effort setting
> and the thinking token count on every run record, and register the change
> before the tag.**

Three reasons, in order of weight:

1. **It is the setting that can only hurt CRED.** A win for arm C under
   thinking-on is credible in a way a win under thinking-off is not. Since
   there is no convention to hide behind, the pre-registration has to
   demonstrate that the configuration was not chosen to flatter the hypothesis
   — and the only demonstration available is picking the configuration that
   disfavours it.
2. **It matches the platform default**, so it needs no defending. `disabled` is
   an intervention and would have to be argued against a reader who assumes the
   model's shipped behaviour.
3. **It is held constant across arms**, which is what the parity constraint in
   §3 of the design actually requires. The design's rule is that arms differ
   *only* in the memory block. A reasoning setting applied equally to all three
   arms satisfies that rule at either value; the question is only which value to
   fix it at, and (1) answers that.

### What it costs

Stated plainly, because a recommendation without a cost is incomplete.

- **Money: roughly $10–25 on the MVE.** Thinking adds output tokens only.
  540 runs × ~1–3k thinking tokens ≈ 0.5–1.6M extra output tokens. Sonnet 5
  output is $15/MTok ($10 introductory through 2026-08-31), giving ~$5–24. On
  the full design (~6,000 runs) it scales to roughly $110–270. Against the
  design's $800–2,000 all-in estimate this is **noise, and it is not a reason
  to choose either way.**
- **Variance: this is the real cost.** Thinking is a second stochastic process
  layered on a model where temperature cannot be set to 0. Run-to-run
  disagreement will rise. The existing guard is adequate and already
  pre-registered — flip rate > 25% in any arm forces 5 runs per task before the
  result may be read (`§8` of the pre-registration, enforced in
  `analysis/analyze.py`). **The concrete exposure is that the escalation
  triggers, and the MVE's 540 runs become 900**, adding roughly $60–120 and
  about a day of wall-clock. That is the price of this recommendation and it
  should be budgeted, not discovered.
- **Interpretability: thinking tokens are not returned by default.** On Sonnet
  5, `display` defaults to `"omitted"` — thinking blocks stream with empty
  text. Set `display: "summarized"` if the analysis wants to inspect *why* an
  arm failed. This costs nothing extra; thinking is billed identically under
  every display setting.

### The alternative, and why not

**Both-and-publish** — run the primary comparison at both settings — is the
scientifically best answer and is **not recommended for the MVE.** It doubles
the run count to 1,080 for a study whose stated purpose is to be cheap enough
to kill the project, and it introduces a second primary comparison into a
pre-registration whose main defence is having exactly one. If the MVE returns
PROCEED and the full design runs, **thinking-on vs thinking-off belongs there
as a registered secondary axis on arms A and C-both only**, reported as a
directional check under Holm-Bonferroni, never as a second primary.

### If this is rejected

The fallback is defensible but weaker: keep `disabled`, and add one sentence to
the deviation stating that disabling reasoning is expected to disadvantage arm A
and therefore favours CRED's hypothesis, with the effect size unmeasured. **A
disclosed thumb on the scale is survivable. An undisclosed one is not.** The
current §9 text does not disclose it.

---

## 10. Other protocol changes CRED should make

Ordered by how much they change the result.

1. **The MVE's judge is the same model family as the system under test.**
   `config/mve.json` runs one model, `claude-sonnet-5`, and the design's §5
   requires *"two judges from different model families."* The MVE silently drops
   that control while keeping the κ gate. Mem0 (gpt-5 judging gpt-5), Graphiti
   (judge is literally the same client object as the candidate builder), and
   Cognee (judge defaults to the answering model) all have this conflict, and
   it is one of the field's clearest weaknesses. **Either add a second judge
   from a different provider for the judge-only calls — cheap, since only
   `match.type == "judge"` checkpoints reach a model — or register the
   single-family judge as an explicit MVE deviation with its risk stated.**
   Right now it is neither.
2. **Publish the arm A prompt and the caching configuration as a first-class
   artifact, not a hash.** §12 lists prompt hashes. Given that three of four
   vendor systems ship no baseline code at all, the *contents* of CRED's
   baseline are the most novel thing it will publish. Hashes prove
   non-tampering; they do not let anyone improve the baseline.
3. **Report the abstention/decoy results even if the primary comparison
   fails.** Zep drops 22% adversarial questions; Mem0's judge awards partial
   credit. A published confabulation rate on a category the field discards is a
   contribution independent of the H1 outcome.
4. **Split cost reporting into write-path and read-path explicitly**, and say in
   the writeup that Mem0's headline 90% saving counts read-path only. CRED's
   dual amortization already computes what is needed; the framing is what
   makes it legible as a critique.
5. **Report the LoCoMo/token-count discrepancy** if CRED cites LoCoMo at all.
   Three published sources disagree by ~3x on the same dataset.
6. **Reconsider whether corpus size S is worth running.** Its stated purpose is
   to replicate the fastpaca regime on the same axis. That is still right — but
   the finding that LoCoMo, MemBench base trajectories, and BEAM-as-shipped are
   *all* in the same too-small regime makes S more valuable than the design
   assumed. It is not one point of comparison; it is where the entire published
   literature sits.

---

## 11. What remains unverified

- **The direction and size of the thinking effect on either arm.** This is the
  load-bearing unknown behind §9. Nothing in the literature measures it. It
  would be checked by a 20-task dev-split pilot running arms A and C at both
  settings — cheap, permitted before the tag, and worth doing before the
  deviation is registered.
- **MemGPT's and MemoryAgentBench's internal configuration.** Abstracts only.
- **LongMemEval's 115k-token figure.** README claim; the dataset files are not
  in the repo.
- **MemBench's `max_words`, temperature, and model.** The files that set them
  (`benchutils.py`, `utils.py`, `makenoise.py`) are imported but absent from
  the repository. The paper's numbers are not reproducible from the code.
- **Mem0's published LoCoMo result provenance.** `results/platform/locomo_results.json`
  carries a `merged_from_questions` key listing several hundred specific
  question IDs, and grep for that string across `benchmarks/*.py` returns zero
  matches — there is no such CLI flag in the argparse block at
  `benchmarks/locomo/run.py:682-716`. The shipped headline was produced by a
  harness version supporting selective merging of a named question subset, and
  that code is not in the repo. **If selection correlated with outcome this is
  a score-inflation mechanism; if it was mechanical (retry-on-API-failure) it is
  benign. Not determinable from the code.** Flagged as an open question, not an
  accusation.
- **Whether Zep ever responded to the category-5 issue.** The GitHub issue was
  reported open without reply; not independently confirmed.
- **The "judge accepts 62.81%" audit.** Non-peer-reviewed vendor blog, single
  source, not reproduced.

---

## 12. Sources

Repository citations throughout are `path:line` against the commits in §1.
Scratchpad clones live under
`/private/tmp/claude-501/-Users-canh-Solo-repos-shift/36bff501-9ae3-4392-b8f6-7ebed876f2e6/scratchpad/`
and are not part of this repository; re-clone from the commit hashes to
reproduce.

URLs fetched:

- https://arxiv.org/abs/2504.19413 · https://arxiv.org/html/2504.19413v1 (Mem0)
- https://arxiv.org/abs/2501.13956 · https://arxiv.org/html/2501.13956v1 (Zep)
- https://arxiv.org/abs/2310.08560 (MemGPT, abstract)
- https://arxiv.org/abs/2410.10813 · https://arxiv.org/html/2410.10813v1 (LongMemEval)
- https://arxiv.org/abs/2402.17753 · https://arxiv.org/html/2402.17753v1 (LoCoMo)
- https://arxiv.org/abs/2506.21605 · https://arxiv.org/html/2506.21605v1 (MemBench)
- https://arxiv.org/abs/2507.05257 (MemoryAgentBench, abstract)
- https://arxiv.org/abs/2505.24478 (Cognee-adjacent HPO)
- https://github.com/getzep/zep-papers/issues/5
- https://blog.getzep.com/lies-damn-lies-statistics-is-mem0-really-sota-in-agent-memory/
- https://dev.to/penfieldlabs/we-audited-locomo-64-of-the-answer-key-is-wrong-and-the-judge-accepts-up-to-63-of-intentionally-33lg

Anthropic API behaviour for `claude-sonnet-5` (adaptive thinking default,
`temperature` rejection, output pricing, `display` default) is from the
in-repo Claude API reference loaded at scan time, not from a fetched URL.
