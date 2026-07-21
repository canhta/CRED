# Why the survivors survive — Mem0, Zep, Letta, Supermemory, Cognee

- **Date:** 2026-07-20
- **Status:** Evidence document
- **Purpose:** This project's research has been weighted toward disconfirmation
  — a graveyard of failed launches, no disclosed revenue in the category, a
  ~10:1 seller-to-buyer ratio on HN, a benchmark suggesting memory loses to long
  context. That was deliberate, but it produced an asymmetric picture: we know
  in detail why things here fail and almost nothing about why the survivors
  survive. This scan asks what is actually working.

> **Evidence note.** Figures marked ✅ were pulled by me on 2026-07-20 from the
> GitHub REST API, the npm registry API, the PyPI `pypistats.org` API, the HN
> Algolia API, or read out of a local clone of `mem0ai/mem0`. `file:line`
> references are to that clone. Claims sourced only to a company's own site,
> blog, or pricing page are labelled **claim** — evidence of what they *say*,
> never of what they *do*. Two sections were researched by delegated agents
> under the same discipline; where they could not verify something, that is
> recorded rather than smoothed. Unverified items are collected in the last
> section.

---

## Verdict

1. **Nobody in this category sells seats.** ✅ Five pricing pages, five metered
   models: Mem0 by add/retrieval request, Zep by ingest credit, Cognee by token,
   Supermemory by credit-wrapped usage, Letta by *active agent per month*. Zep
   and Cognee charge **nothing for reads, storage, or users**. This is the
   single most transferable finding and it **contradicts D-007's $20–40 per
   seat price anchor**.

2. **The distribution channel that moved the needle is being vendored inside
   someone else's repository** — not marketing, not launches. ✅
   `run-llama/llama_index` ships `llama-index-memory-mem0` as a first-party
   package in its own monorepo; `strands-agents/tools` ships
   `src/strands_tools/mem0_memory.py`; `getzep/zep` carries 10 framework
   integration directories. A developer picking LlamaIndex or Strands gets a
   memory vendor without ever running a vendor evaluation.

3. **These are tiny teams.** ✅ Mem0 was **four people** at its $24M Series A
   ([TechCrunch](https://techcrunch.com/2025/10/28/mem0-raises-24m-from-yc-peak-xv-and-basis-set-to-build-the-memory-layer-for-ai-apps/)).
   Zep's YC page says 5. Cognee shows ~17. The gap between a solo founder and
   the category leader is roughly 3 engineers, not an army. Most of what these
   companies did is reproducible; the parts that are not are named in §7.

4. **Usage is real and growing, and it is not stars.** ✅ `mem0ai` on PyPI went
   **694,649 downloads in Jan 2026 → 3,218,505 in Jun 2026** (4.6x in five
   months). `graphiti-core` 151,756 → 1,246,321 (8.2x). These are trend lines,
   not vanity counts.

5. **Still zero disclosed revenue, category-wide.** Verified as an *absence*: I
   searched and found no ARR figure, paying-customer count, or retention number
   for any of the five. The strongest monetization evidence available is
   indirect — Zep bought SOC 2 Type II and a HIPAA BAA, which nobody does
   speculatively. **The market-landscape finding stands unchanged.**

6. **Letta is shrinking on the exact axis CRED is considering building.** ✅
   `letta-client` on PyPI peaked at **203k in May 2026** and fell to **90k in
   June** — a 56% single-month drop, following a March 2026 announcement
   sunsetting server-side memory in favour of git-backed client-side files. The
   best-credentialed team in agent memory is retreating from the memory server.

7. **Mem0's most-cited distribution claim is false.** **FALSIFIED** — see §3.

---

## 1. What was checked, and what could not be

**Checked directly** ✅ — GitHub REST API (stars, forks, contributors, repo
contents) for six repos; npm registry download API for five packages over
2026-06-20→2026-07-19; PyPI `pypistats.org/api/packages/{pkg}/overall` for five
packages, aggregated by month; HN Algolia story search and item API; GitHub code
search via `gh api` against four framework repos; a local clone of `mem0ai/mem0`
at `v2.0.12`; and direct fetches of the Mem0, Zep, Cognee, Supermemory and Letta
pricing pages plus the Mem0 Series A announcement, PR Newswire release,
TechCrunch coverage, AWS partnership post, Letta's next-phase post, and Zep's
OSS-strategy post.

**Could not be checked, and why:**

- **Git history of the local clones.** ✅ `/Users/canh/Solo/OSS/mem0` and
  `/Users/canh/Solo/OSS/letta` are **shallow clones with exactly 1 commit each**
  (`git rev-list --count HEAD` → `1`). The task suggested checking history for
  removals; that is unavailable locally. I substituted mem0's in-repo changelog
  (`docs/changelog/sdk.mdx`, 198 `<Update>` entries), which is a vendor artifact
  but a dated and specific one. `graphiti` was not present locally.
- **Revenue for anyone.** No company discloses it. `getlatka` figures surfaced
  for Zep (~$1M/2024) and Letta ($1.4M ARR) are **modeled estimates**, and the
  Zep page reports funding raised as "$0" — a verifiably false field on a page
  whose other fields are unverifiable. **Both discarded.**
- **Zep's funding.** No primary source exists. Three aggregators give three
  incompatible numbers ($500K pre-seed, $1.8M seed, $2.3M total). Crunchbase
  returns 403. **Recorded as unknown rather than picked.**
- **Star history over time.** The GitHub API gives only a current total. No
  growth curve was retrieved for any repo.
- **Cloud marketplace listings** (AWS/Azure/GCP) for any of the five. None
  found; absence of evidence only.
- **Reddit**, as in every prior scan in this project — blocked in this
  environment.
- **Discord size** for Mem0, Zep, Cognee, Supermemory. Only Letta's was
  obtainable ✅ (11,740 members, 1,141 online, via the Discord invite API).

**Two sources were encountered and deliberately excluded as AI-generated
fabrications:** `extruct.ai`, which asserts "$795M Revenue" for Zep, and
`ubos.tech`, which asserts a "$5 million seed round led by About UBOS's venture
arm." Named here so nobody re-derives them.

---

## 2. Per-company profiles

### 2.1 Mem0 — the category leader

| | |
|---|---|
| Funding | **$24M total**: $3.9M seed (previously unannounced) + $20M Series A ✅ |
| Announced | **2025-10-28** |
| Leads | Kindred Ventures (seed); **Basis Set Ventures** (Series A) |
| Others | Peak XV, GitHub Fund, Y Combinator |
| Angels | Scott Belsky, Dharmesh Shah, Olivier Pomel (Datadog), Paul Copplestone (Supabase), James Hawkins (PostHog), Thomas Dohmke (ex-GitHub), Lukas Biewald (W&B) |
| **Team size** | **Four people** at the raise ✅ |
| Stars | 61,301 ✅ · forks 7,125 · contributors 378 · Apache-2.0 |

Sources:
[mem0.ai/series-a](https://mem0.ai/series-a),
[PR Newswire](https://www.prnewswire.com/news-releases/mem0-raises-24m-series-a-to-build-memory-layer-for-ai-agents-302597157.html),
[TechCrunch](https://techcrunch.com/2025/10/28/mem0-raises-24m-from-yc-peak-xv-and-basis-set-to-build-the-memory-layer-for-ai-apps/).
The seed/Series A split appears only in TechCrunch; the company's own page says
"$24M across Seed and Series A" without decomposing it.

**What they sell** ([mem0.ai/pricing](https://mem0.ai/pricing), fetched
2026-07-20 — **claim**): Hobby $0 (10,000 add + 1,000 retrieval requests/mo,
1 project) · Starter **$19/mo** (50,000 add + 5,000 retrieval) · Pro
**$249/mo** (500,000 add + 50,000 retrieval, unlimited projects, graph memory,
advanced analytics) · Enterprise custom (SLA, **on-prem**, audit logs, SSO).

Two things to read off this. **The meter is requests, not seats** — "unlimited
end users" appears on every tier. And the free→paid step is 10k→50k add
requests for $19, which is self-serve developer pricing, while the features that
matter to an organization (audit logs, SSO, on-prem) are all in the unpriced
Enterprise tier. Mem0 monetizes API volume at the bottom and contracts at the
top, with a $249 gap in between.

**Usage signal** ✅ — the best trend data in this document:

| Month (2026) | `mem0ai` PyPI | `graphiti-core` | `zep-cloud` | `cognee` | `letta-client` |
|---|---|---|---|---|---|
| Jan | 694,649 | 151,756 | 41,232 | 19,358 | 22,112 |
| Feb | 1,871k | 311k | 117k | 80k | 63k |
| Mar | 2,799k | 477k | 166k | 77k | 73k |
| Apr | 2,752k | 490k | 156k | 95k | 171k |
| May | 3,207k | 592k | 216k | 99k | **203k** |
| Jun | **3,218,505** | **1,246,321** | 364,057 | 99,060 | **90,237** |
| Jul (partial) | 2,203k | 1,030k | 203k | 100k | 66k |

Pulled from `pypistats.org/api/packages/{pkg}/overall?mirrors=false`, category
`without_mirrors`, aggregated by month. **July is a partial month (through
~19th) and must not be compared to full months.** The API window is ~180 days,
which is why the series starts in January.

npm, 2026-06-20→2026-07-19 ✅: `@getzep/zep-cloud` **451,384** ·
`mem0ai` **367,696** · `supermemory` **278,341** ·
`@letta-ai/letta-client` **138,889** · `@mem0/community` 20,157.

Note the inversion: **Zep leads on npm downloads while having the *fewest*
stars** (`getzep/zep` 4,766 ✅). Stars and usage are close to uncorrelated here.

Company-disclosed metrics (**claim**): 41,000 stars, 14M Python downloads,
**API calls 35M in Q1 2025 → 186M in Q3 2025** at "30% month-over-month growth,"
80,000+ cloud signups (Series A, Oct 2025). The homepage today says **"90,000+
Developers build with Mem0"** ([mem0.ai](https://mem0.ai/), fetched 2026-07-20).

**That last pair is the most interesting number in this document.** Cloud
signups went 80,000 → 90,000 (+12.5%) between Oct 2025 and Jul 2026, while PyPI
downloads grew 4.6x in the first six months of 2026 alone. **OSS usage is
compounding; cloud signups are close to flat.** That is the classic
open-source shape a16z names as failure mode #1 — product-market fit without
value-market fit — visible in the category leader's own published numbers.

**Named customers: none.** ✅ The homepage shows no logos, no case studies, no
testimonials, against "thousands of teams, from the fastest-growing startups to
Fortune 500 companies, use us in production" (**claim**, Series A page). Compare
Zep (8 named logos), Cognee (Bayer, University of Wyoming, dltHub) and Letta
(Bilt, 11x, Kognitos, Hunt Club) — **the best-funded company in the category is
the only one that names nobody.** Compliance posture is also the weaker one:
**SOC 2 Type 1** (mem0.ai) against Zep's **Type II + HIPAA BAA**.

**Release cadence** ✅ — `docs/changelog/sdk.mdx` carries 198 `<Update>` entries;
v2.0.0 (2026-04-14) through v2.0.12 (2026-07-13) is 13 releases in three months.
Very much alive.

### 2.2 Zep — the enterprise motion, with a shell repo

**Funding: unknown.** No primary announcement exists. Verified only that Zep AI
is **Y Combinator Winter 2024**, founded 2023, San Francisco, **team size 5**
([YC](https://www.ycombinator.com/companies/zep-ai)). Founder/CEO Daniel Chalef.

**What they sell** ([getzep.com/pricing](https://www.getzep.com/pricing),
fetched 2026-07-20 — **claim**): Free $0 (10,000 credits/mo, 2 projects) ·
**Flex $1,250/yr** ($104/mo annual, 50,000 credits/mo, then $25/10,000) ·
**Flex Plus $3,750/yr** ($312/mo annual, 200,000 credits/mo) · Enterprise custom
(SOC 2 Type II, HIPAA BAA, 1-year audit logs, Cloud / BYOK / BYOC).

The credit unit is the important part: **1 credit per Episode up to 350 bytes,
+1 per additional 350 bytes.** Webhooks are ⅛ credit. **Retrieval, storage and
users are unmetered — zero credits.** Zep charges for writes and gives reads
away. Given D-009 already commits CRED to a read-first product, this is the
pricing shape that matches it.

Named logos ([getzep.com](https://www.getzep.com/) — **claim**): Twin Health,
Praktika.ai, Thrive AI Health, AGI Inc, Harper, FlockX, Aurasell, HoneyBook,
plus AWS and Samsung, plus an unnamed **"Fortune 500 Tech Co. (confidential)"**.
A confidential placeholder sitting among real logos is a tell that the marquee
reference cannot be named.

Paper: **arXiv:2501.13956**, "Zep: A Temporal Knowledge Graph Architecture for
Agent Memory," submitted 2025-01-20, **v1 only, never revised**. Claims DMR
94.8% vs MemGPT 93.4% — a 1.4-point margin on a benchmark the competitor
designed. See §3 for whether it drove anything.

### 2.3 Letta — the strongest credibility, the weakest trajectory

**Funding: $10M seed, 2024-09-24, led by Felicis**, with Sunflower Capital and
Essence VC; angels Jeff Dean, Clem Delangue, Jordan Tigani, Robert Nishihara and
others
([PR Newswire](https://www.prnewswire.com/news-releases/berkeley-ai-research-lab-spinout-letta-raises-10m-seed-financing-led-by-felicis-to-build-ai-with-memory-302257004.html)).

**No follow-on round has been announced in ~22 months.** Letta's blog index
carries 26 posts from Sept 2024 to Jun 2026 and **not one is a funding
announcement after the seed** ([letta.com/blog](https://www.letta.com/blog)).
Stated as "none announced," not "none raised" — Crunchbase 403'd and an
unannounced round cannot be ruled out. But for a company whose seed was led by a
tier-one fund on the back of a 958-citation paper, 22 months of silence is
itself evidence.

**What they sell** ([docs.letta.com/letta-code/pricing](https://docs.letta.com/letta-code/pricing)
— **claim**): Free $0 (BYO keys) · Pro **$20/mo** (up to 20 stateful agents) ·
Developer/API **$20/mo base + usage**, at **$0.10 per active agent per month**
and $0.00015/sec tool execution · Enterprise custom (SAML/OIDC, RBAC).

`letta.com/pricing` **301-redirects to the Letta Code docs** — the
reorganization around the new product is complete enough to have eaten the
pricing URL.
**Named customers, and they are the strongest in the set**
([letta.com/case-studies](https://www.letta.com/case-studies) — **claim**):
Bilt (**"over 1 million agents"** deployed, CTO and lead engineer on record by
name), Kognitos (an end-customer "approved a nearly half-million dollar
expansion"), 11x, Hunt Club.

**Community** ✅: **11,740 Discord members, 1,141 online** — verified directly
via the Discord invite API, and roughly 4x anything confirmable for the others.
Plus a **DeepLearning.AI course**, "LLMs as Operating Systems: Agent Memory"
(Nov 2024) — a distribution asset no competitor in this set has.

Against all of that: `letta-client` PyPI **203k May → 90k June** ✅, and
`letta-ai/letta` was last pushed 2026-07-03 while `letta-ai/letta-code` (2,866
stars, **151 open issues**) was pushed 2026-07-20. All current energy is in the
new repo; the 23,879-star flagship is comparatively quiet.

### 2.4 Supermemory — distribution without a launch

**Funding: the company and TechCrunch disagree, on the same day.** The blog says
**$3M, "pre-seed,"** led by Susa Ventures with Browder Capital and SF1.vc,
angels including Dane Knecht (CTO, Cloudflare), Theo Browne and David Cramer
([supermemory.ai](https://supermemory.ai/blog/supermemory-raises-3-million-and-building-the-best-memory-engine-for-llms/)).
TechCrunch, 2025-10-06, says **$2.6M, "seed,"** and names Jeff Dean and Logan
Kilpatrick among the angels
([TechCrunch](https://techcrunch.com/2025/10/06/a-19-year-old-nabs-backing-from-google-execs-for-his-ai-memory-startup-supermemory/)).
So Tracxn's $2.6M is not an aggregator error — **the discrepancy originates with
the company**, which rounded up and relabelled the round. Worth noting given how
much of this project's evidence discipline is about number hygiene.

**What they sell** ([supermemory.ai/pricing](https://supermemory.ai/pricing) —
**claim**): Free $0 (~$5/mo credit) · Pro **$19/mo** (~$20 credit, 2 teammates)
· Max **$100/mo** (~$130 credit) · Scale **$399/mo** (~$600 credit, 10
teammates, SOC 2/HIPAA BAA, self-host) · Enterprise custom (air-gapped).
Underlying meter: $0.005/1K tokens memory, $0.005/1K search queries.

A subscription that is a **credit wrapper on usage** — the seat counts are caps,
not the billing unit.

Named customers (**claim**): Cluely, Composio, Scira AI, Montra, Rube, Rets.
Case studies include **two explicit Mem0 rip-and-replace wins** — Scira claims
"37.4% lower mean retrieval latency vs. Mem0"
([supermemory.ai/case-studies](https://supermemory.ai/case-studies)).

Disclosed: "50k+ users," "billions of tokens every week" (**claim**). Chrome
extension: **9,000 users, 3.4★ from 46 ratings** — the consumer surface is weak.

### 2.5 Cognee — the best-capitalized late entrant

**Funding: $7.5M seed, announced 2026-02-19**, led by **Pebblebed** (Pamela
Vagata, OpenAI co-founder; Keith Adams, FAIR), with 42CAP and Vermilion
([cognee.ai](https://www.cognee.ai/blog/cognee-news/cognee-raises-seven-million-five-hundred-thousand-dollars-seed)).
Legal entity Topoteretes UG, Berlin. **Team ~17** per the about page. One
currency discrepancy: hyperight.com reports €7.5M; the company says USD.

**What they sell** ([cognee.ai/pricing](https://www.cognee.ai/pricing) —
**claim**): Free $0 (1M tokens, **unlimited users, unlimited API calls**) ·
Standard **$2.50 per 1M tokens** + $5 per extra workspace · Enterprise custom
(BYO cloud, SLA).

The purest consumption model in the set: no seat floor, no annual commitment,
explicitly unlimited seats and API calls on free.

Disclosed in the funding post (**claim**): **live in 70+ companies**; pipeline
volume grew from ~2,000 runs to **over 1 million during 2025**; 80+ OSS
contributors. Named: Bayer, University of Wyoming, Knowunity, dltHub,
SlideSpeak.

Two soft spots ✅: **617 open issues** and — the harder signal — only **94
watchers** against 28,613 stars. A 300:1 star-to-watcher ratio is extreme and
consistent with stars driven by promotion rather than by dependent users.

---

## 3. Distribution mechanics — the section that matters

Four distinct mechanisms are visible, and they are not equally reproducible.

### 3.1 The one that actually works: be vendored inside someone else's repo

This is the finding. Not "have an integration page" — **have your code inside
their monorepo, shipped in their release, documented in their docs.**

✅ Verified via `gh api search/code`:

| Framework repo | Hits for `mem0` | What is actually there |
|---|---|---|
| `run-llama/llama_index` | **9** | `llama-index-integrations/memory/llama-index-memory-mem0/` — a **first-party package** with `base.py`, `pyproject.toml`, tests, an API reference page, and `docs/examples/memory/Mem0Memory.ipynb` |
| `crewAIInc/crewAI` | **71** | Spread across versioned changelogs `v1.10.0`→`v1.14.3` in en/ko/pt-BR — a long-standing, continuously-shipped integration |
| `strands-agents/tools` | **4** | `src/strands_tools/mem0_memory.py` + `tests/test_mem0.py` + README + `pyproject.toml` extra |
| `langchain-ai/langchain` | 0 | not in core |

Comparison inside `crewAIInc/crewAI` ✅: `zep` → 9 hits, `letta` → **0**,
`cognee` → **0**. Mem0 has ~8x Zep's footprint and infinite times Letta's in the
single most popular agent framework.

Zep's counter-strategy is breadth, not depth ✅ —
`api.github.com/repos/getzep/zep/contents/integrations` returns **10
directories**: `adk` (Google), `ag2`, `autogen`, `crewai`, `langgraph`,
`livekit`, `mastra`, `ms-agent-framework`, `pydantic-ai`, `vercel-ai`. Cognee
has an official `langchain-cognee` package in LangChain's own docs.

Mem0's own in-repo surface ✅: `docs/integrations/` contains **30 `.mdx` files**
— agno, autogen, aws-bedrock, camel-ai, claude-code, codex, crewai, cursor,
dify, elevenlabs, flowise, google-ai-adk, langchain, langgraph, livekit,
llama-index, mastra, openai-agents-sdk, openclaw, opencode, pipecat,
raycast, vercel-ai-sdk, and more.

**Why this is the mechanism:** a developer who has chosen LlamaIndex does not
run a memory bake-off. They type `pip install llama-index-memory-mem0` because
it is the memory option in the docs they are already reading. The vendor
captures the decision *before it is framed as a decision*. This is the same
insight as Context7's "be already-packaged wherever a user forms the intent to
install something" ([context7.md](context7.md)), pushed one layer deeper: not
packaged for the ecosystem, but **packaged inside the ecosystem's own source
tree**.

**Cost, named:** every one of those is a PR into someone else's repo, reviewed
on their schedule, plus a permanent maintenance tax when their interfaces move.
Mem0 has 378 contributors and 679 open issues ✅; a meaningful share of that is
integration upkeep.

### 3.2 FALSIFIED — "Mem0 is the exclusive memory provider for AWS's Agent SDK"

This claim appears in Mem0's own Series A page and PR Newswire release
(**claim**) and was repeated by TechCrunch. It is **false as of 2026-07-20.**

✅ `strands-agents/tools` `src/strands_tools/` contains **five** memory tools:

```
agent_core_memory.py   elasticsearch_memory.py   mem0_memory.py
memory.py              mongodb_memory.py
```

And the README states the framework supports memory "with both Mem0, Amazon
Bedrock Knowledge Bases, Elasticsearch, and MongoDB Atlas"
(`raw.githubusercontent.com/strands-agents/tools/main/README.md:42`). One of
those competitors is **AWS's own** `agent_core_memory` — "Store and retrieve
memories with Amazon Bedrock Agent Core Memory service" (README:122).

Mem0's own partnership announcement never says "exclusive." Asked directly, the
post's language is *"By integrating Mem0, Strands supports a persistent memory
layer"*
([mem0.ai blog](https://mem0.ai/blog/aws-and-mem0-partner-to-bring-persistent-memory-to-next-gen-ai-agents-with-strands),
published 2025-05-19).

**Three lessons.** (a) The exclusivity was either never real or expired, and the
company kept the phrasing in its fundraising materials. (b) **The platform built
its own.** AWS shipped Bedrock AgentCore Memory into the same tool namespace —
which is exactly the "platforms price the memory primitive at zero" risk already
recorded in **D-004 risk #2**, now with a specific instance. (c) A
hyperscaler partnership is not a moat; it is a slot that the hyperscaler can
fill itself, and did.

### 3.3 The launch: sometimes it works, and the channel is not always HN

The [context7.md](context7.md) scan established that a low HN score does not
predict failure. This scan adds the other half: **a high one is achievable, and
one company here got it.**

✅ All from the HN Algolia API:

| Launch | Points | Comments | Date |
|---|---|---|---|
| **Show HN: Mem0 – open-source Memory Layer for AI apps** ([41447317](https://news.ycombinator.com/item?id=41447317)) | **201** | 61 | 2024-09-04 |
| MemGPT – LLMs with self-editing memory ([37901902](https://news.ycombinator.com/item?id=37901902)) | **363** | 85 | 2023-10-16 |
| MemGPT: Towards LLMs as Operating Systems ([37894403](https://news.ycombinator.com/item?id=37894403)) | **225** | 106 | 2023-10-15 |
| Show HN: Graphiti – LLM-Powered Temporal Knowledge Graphs | **142** | 21 | 2024-09-04 |
| Show HN: Cursor IDE now remembers your coding prefs using MCP | **109** | 39 | 2025-03-28 |
| Letta: framework for LLM services with memory ([43294974](https://news.ycombinator.com/item?id=43294974)) | 121 | 22 | 2025-03-07 |
| Letta Code ([46294274](https://news.ycombinator.com/item?id=46294274)) | 83 | 37 | 2025-12-16 |
| Show HN: Zep – Long-Term Memory Store for LLM Apps | 7 | — | 2023-05-10 |
| Show HN: Zep – Open-Source Graph Memory for AI Apps | 6 | — | 2024-09-26 |
| Zep: The Foundational Memory Layer for AI | 1 | — | 2024-11-09 |
| **Show HN: Cognee – Open-Source AI Memory Layer** | **9** | 2 | 2025-06-03 |
| Show HN: Cognee – Turn RAG/GraphRAG into semantic memory | 6 | 1 | 2025-02-13 |
| Show HN: Supermemory-mcp | 4 | 0 | 2025-06-08 |
| Supermemory: AI second brain | 5 | 3 | 2024-07-22 |
| **Letta's Next Phase** (the pivot) ([47406067](https://news.ycombinator.com/item?id=47406067)) | **2** | **0** | 2026-03-16 |

Read carefully, this table says four different things.

**(a) Mem0's launch genuinely worked.** 201 points, 61 comments. The top
comments are not congratulation — they are the exact objections CRED will face:
*"How does Mem0 handle the potential for outdated or irrelevant memories over
time? Is there a mechanism for 'forgetting'?"* and *"Over time, I can imagine
there's going to be a lot of sensitive information being stored. How are you
handling privacy?"* ✅ (thread 41447317). **Staleness and governance were the
first two questions HN asked the category leader in 2024, and per
[mem0.md](mem0.md) Mem0 still has no answer to either** — the reconciler has
zero call sites and the ACL has zero writers. That is a demand signal for CRED's
thesis, sitting in a competitor's launch thread.

**(b) Cognee's peak HN score across every submission ever is 9** ✅ — and it has
28,613 stars and a $7.5M seed. Supermemory's peak is 5, with 28,500 stars.
**Two of the five survivors got nothing from HN at all.** This reinforces
context7.md's falsification rather than softening it.

**(c) Zep's brand launches all failed; its two hits were about something else.**
142 points for Graphiti (the OSS library, not the SaaS) and 109 for a
*Cursor + MCP* post. Zep gets attention by attaching to someone else's hot
surface, never by pitching Zep.

**(d) Supermemory found a different channel entirely: Product Hunt.**
**#1 Product of the Day, 705 upvotes, 2024-07-21**, and a second launch at **#2,
440 upvotes, 2025-04-18** ([Product Hunt](https://www.producthunt.com/products/supermemory)).
**This project's demand research has never looked at Product Hunt.** A company
with 28.5k stars and 278k npm downloads/month built its audience on a channel
absent from every prior scan. That is a gap in our method, not just in our data.

### 3.4 The paper as a distribution engine — real, and non-transferable

Letta is the clearest case in the category and the least reproducible.

✅ The MemGPT paper (arXiv:2310.08560) has **958 citations** per Semantic
Scholar (`api.semanticscholar.org/graph/v1/paper/arXiv:2310.08560`, 95
influential). Two front-page HN hits within 48 hours (363 + 225 points). The
repo `cpacker/MemGPT` was created 2023-10-11, **five days before** the launch.

**Letta's distribution engine fired in October 2023, under a name the company no
longer uses, eleven months before the company existed — and has not fired
since.** Everything branded Letta peaks at 121 points; the Feb 2026 "Show HN:
Letta – Git-Based Memory for Coding Agents" scored **2**.

Zep tried the same move and it did not work. arXiv:2501.13956 is self-published,
v1 only, never revised, benchmarking against a metric its competitor designed,
and Graphiti's visible inflections (Sep 2024 Show HN, Mar 2025 Cursor/MCP post)
bracket the January 2025 paper without any attributable jump. **A paper is a
distribution channel only if it is a real research contribution that other
researchers cite.** MemGPT was; the Zep paper is a marketing artifact in a
paper's clothing.

### 3.5 Summary: which channel, per company

| Company | Channel that moved the needle | Confidence |
|---|---|---|
| **Mem0** | Framework vendoring (LlamaIndex/CrewAI/Strands) + a genuine 201-pt Show HN | High ✅ |
| **Letta** | The MemGPT paper + two front-page HN hits, Oct 2023 | High ✅ |
| **Zep** | Attaching to others' surfaces — Graphiti as standalone OSS, then MCP-in-Cursor | Medium ✅ |
| **Supermemory** | Product Hunt (#1, 705 upvotes) + founder's X audience + Cloudflare patronage | Medium |
| **Cognee** | **Undetermined.** HN peak 9; no single referral source evidenced. Consistent with many small channels (integration directories, DB-vendor blogs, MCP directories) plus funding press — and also consistent with star promotion, which cannot be ruled out | **Low** |

On Cloudflare: the confirmed relationship is that Dhravya Shah interned there,
later worked in devrel, and **CTO Dane Knecht personally asked him to turn
Supermemory into a product and then invested** (TechCrunch). But his Cloudflare
engineering blog post does not mention Supermemory and no Cloudflare marketing
of Supermemory was found. **This is patronage and credibility, not a
distribution partnership** — an important distinction, because patronage is not
purchasable and a partnership might be.

---

## 4. What each abandoned, and why

### Mem0 — deleted the graph, and deleted organizational scoping

✅ From `docs/changelog/sdk.mdx`, **v2.0.0, dated 2026-04-14**, under
"Breaking Changes":

- **`org_id` and `project_id` removed** from the `MemoryClient` constructor and
  all method signatures (PR #4740).
- **External graph store removed (OSS):** `graph_memory.py`,
  `memgraph_memory.py`, `kuzu_memory.py`, `apache_age_memory.py` and
  `mem0/graphs/` (Neo4j / Memgraph / Kuzu / Apache AGE / Neptune drivers)
  **deleted, ~4,000 lines** (PR #4805). The
  TS SDK lost a matching ~1,088 lines. "Graph memory is now a project-level
  setting on the Platform."
- **`add()` returns ADD-only events**: "No more `UPDATE` or `DELETE` events.
  Memories accumulate; nothing is overwritten" (PR #4805).

Three abandonments in one release, and each one is directly load-bearing for
CRED. They removed the **organizational scoping primitives**, moved the
**graph** behind the paywall, and gave up on **reconciliation** — which is what
[mem0.md](mem0.md) independently found in the source (the ADD/UPDATE/DELETE
reconciler has zero call sites). This is not drift; it is a deliberate,
documented, dated retreat from exactly the three things CRED proposes to build.

**The honest reading cuts both ways.** It is evidence that these are hard enough
that a funded team gave up on doing them in the open — and evidence that the
space is genuinely vacant.

### Letta — the rename, then the teardown

**(a) MemGPT → Letta, 2024-09-23.** ✅ `pymemgpt` on PyPI is frozen at
**v0.3.25, last upload 2024-09-11**. Every download statistic, install
instruction, tutorial and search result accumulated under the name that earned
363 HN points and 958 citations was reset into `letta`/`letta-client`. The
rename also decoupled the product from the paper that was the sole engine of its
recognition. **This is the most expensive branding decision visible in the
category, and its cost is measurable in a frozen PyPI page.**

**(b) The refocus, 2026-03-16** ([letta.com/blog/our-next-phase](https://www.letta.com/blog/our-next-phase/)).
Eight sunsets, each moving work from Letta's servers to the client:

| Sunset | Replaced by |
|---|---|
| Letta Filesystem | real filesystem access / context repos |
| Legacy memory tools (`core_memory_replace`) | git-backed context repo file ops ("MemFS") |
| Templates | versioned Letta Code SDK + community tooling |
| Identities | application-layer tags |
| Server-side MCP integrations | client-side skills |
| Server-side sleep-time agents | client-side subagents |
| Multi-agent tools | subagents + agent-discovery skills |
| **Tool rules** | **deprecated outright, no replacement** |

Stated reason: *"many of our older features were built for a world where
computer use was not the primary mechanism for agent action."*

Two things are worth extracting. **Letta Filesystem shipped in July 2025 and was
killed eight months later** — they deprecated their own 2025 roadmap. And
**every single item moves billable server work to the unbillable client.** A
company does not do that from strength.

**Community reaction: none.** ✅ The announcement scored **2 points, 0 comments**
on HN; `letta-ai/letta` shows 49 open issues and no visible revolt. Nobody
objected. Given a 23,879-star repo and 11,740 Discord members, **the silence is
the finding** — it suggests the server-side surface being torn out had very few
users to upset. Which is corroborated by the PyPI curve: 203k May → 90k June ✅.

### Zep — killed self-hosting, and the repo is now a husk

**Community Edition deprecated 2025-04-02**
([blog.getzep.com](https://blog.getzep.com/announcing-a-new-direction-for-zeps-open-source-strategy/)).
Stated reasons, verbatim:

> "Managing two related but different products…has presented real challenges…
> leading us to under-invest in the open-source version."
>
> "We also felt uneasy about the common practice in open-core of intentionally
> limiting features to drive users toward paid products."
>
> "For us, openness should be genuine—not just a marketing tactic or stepping
> stone to monetization."

Decision: "stop maintaining and releasing Zep Community Edition." The repo stays
Apache-2.0 but gets "no updates or active support." Confirmed in the docs: "Zep
Community Edition, which allows you to host Zep locally, is deprecated and no
longer supported" ([help.getzep.com/faq](https://help.getzep.com/faq)).

✅ The repo today: `getzep/zep` is **not archived**, pushed 2026-07-17, but its
description is now "**Zep | Examples, Integrations, & More**" and its top-level
contents are `.agents`, `.claude-plugin`, `.cursor`, `benchmarks`, `examples`,
`integrations`, **`legacy`**, `mcp`, `ontology`, `plugins`, `zep-eval-harness`.
**The server has been demoted into `legacy/`.** The README says "This is not
Zep's core product." The 4,766 stars were earned by software that no longer
lives there.

**Note the second-order effect.** They killed the self-hostable edition citing
discomfort with *feature-limiting* — and the result is that self-hosting became
*harder*, not freer. Self-hosters were redirected to raw Graphiti plus a graph
database they now operate themselves. Principle and outcome pointed opposite
ways.

Zep also repositioned twice: "The Foundational Memory Layer for AI" (Nov 2024) →
"Zep v3: Context Engineering Takes Center Stage" → today's "Agent memory, at
enterprise scale. Memory of users, the business, and work done. Managed,
**governed**, and served at scale" ([getzep.com](https://www.getzep.com/)).
**Zep has landed on governance as its message.** That is CRED's message, from a
company with SOC 2 Type II and a HIPAA BAA already in hand.

### Supermemory — a pivot the founder describes bluntly

2025-07-25, in the founder's own words
([supermemory.ai blog](https://supermemory.ai/blog/unified-memory-that-works-where-you-work-your-second-brain-with-supermemory/)):

> "We had a big community and a shit ton of users, but, frankly, our product's
> retention was terrible." … "People stopped using it" … "So, we pivoted to B2B
> and set our heads down for one year."

The abandoned thing is the original chat-with-your-bookmarks framing. The
consumer app was not killed — that same post announces a *return* to it
alongside the API. So the trajectory is consumer → B2B → both.

**This is the most valuable single quote in the scan.** A consumer memory
product with 50k+ users and a #1 Product Hunt launch had *terrible retention*.
Memory is easy to get people to try and hard to get them to keep using.

### Cognee — nothing abandoned

No deprecation, pivot or rename found. Consistently positioned as graph-based AI
memory since April 2024. The `topoteretes`/`cognee` split is a legal-entity vs
brand distinction, not a rename. Notably, **Cognee has not made Zep's move** —
the self-hostable Apache-2.0 engine is still the core product, actively
developed (v1.4.0, 2026-07-17).

---

## 5. The honest counter-case: is any of this real success?

The task asked for a three-way classification. Here it is, with the caveat that
**none of the five has disclosed revenue**, so every (a) rating is inferential.

| Company | Classification | The evidence, stated against them where it belongs |
|---|---|---|
| **Mem0** | **(c) usage without monetization, wearing (b)'s clothes** | Downloads 4.6x in 5 months ✅ and 186M API calls/quarter (**claim**) are real. But cloud signups moved 80k→90k in 9 months while downloads grew 4.6x; **zero named customers**; SOC 2 Type 1 only; $24M raised. The OSS is compounding and the business is not visibly following it. |
| **Zep** | **(a) revenue, at small scale — unverifiable** | Nobody buys SOC 2 Type II and a HIPAA BAA speculatively; 8 named logos; a real metered self-serve product; and they killed their OSS edition *to fund the paid one*, which is a revenue-driven decision. Against: team of 5, no ARR, **no verifiable funding round at all**, a Fortune 500 logo they cannot name, and three repositionings in 20 months. |
| **Letta** | **(c) drifting to (b)** | Best credibility in the set — 958 citations, 11,740 Discord ✅, a DeepLearning.AI course, Bilt's million agents. But no follow-on in 22 months, PyPI down 56% month-over-month ✅, and a March 2026 teardown of the paid server surface in favour of a client-side coding agent in a far more crowded market. The $0.10/active-agent meter against Bilt's "1 million agents" would be $100k/mo; **the absence of any revenue disclosure next to that number is conspicuous.** |
| **Supermemory** | **(b), with early (a)** | Named customers with specific migration wins, metered pricing, "billions of tokens weekly." Against: no ARR or customer count; the loud numbers (50k users, 28.5k stars) come from the phase the founder called a retention failure; consumer surface at 3.4★/9k installs; and its own funding announcement carries two different round sizes. |
| **Cognee** | **(b) funding and hype** | Real adoption exists — LangChain first-party integration, 70+ companies, ~500x pipeline growth, Bayer. But "live in 70+ companies" describes *deployment*, not payment, and their free tier grants unlimited users and unlimited API calls, so most of those 70 could be free. **94 watchers against 28,613 stars** ✅ and a peak HN score of 9 undercut the headline metric. Funded by a $7.5M seed, not by customers. |

**The aggregate read, stated plainly.** Four of the five are best described as
funded runway plus genuine developer usage that has not been shown to convert.
The one that looks most like a business — Zep — is also the smallest, the least
funded, the one that killed its open-source product, and the one whose numbers
are least verifiable. **Nothing in this scan overturns market-landscape.md's
finding that the category shows no revenue.** What it does overturn is the
implication that nothing is working: usage is real, growing fast, and
concentrated in specific reproducible channels.

**And the sharpest counter-case against CRED specifically:** the three companies
that tried the *governed, organizational* version of this — Mem0 (removed
`org_id`/`project_id` and the reconciler), Letta (tore out the server), Zep
(killed self-hosting) — all retreated from it in a 12-month window. Either they
were all wrong at once, or the thing is hard to sell. That must be held next to
§6's positive findings, not filed away from them.

---

## 6. What transfers to a solo unfunded founder

### Transfers, and is not purchased

1. **The whole framework-vendoring channel.** ✅ A PR into
   `llama-index-integrations/memory/` costs nothing but time and code quality.
   No money changed hands for Mem0's LlamaIndex package; it is an OSS
   contribution. **This is the highest-leverage reproducible action identified
   in this scan.** It is also the specific mechanic Context7 used, which two
   independent scans now converge on.
2. **The team size.** ✅ Mem0 shipped 61k stars, 3.2M monthly downloads and a
   $24M raise **with four people**. Zep's YC page says five. The gap between one
   founder and the category leader is ~3 engineers. **Nothing in this scan shows
   headcount buying distribution.**
3. **Metered, read-free pricing.** Zep charges 0 credits for retrieval, storage
   and users. Cognee gives unlimited seats and API calls on free. This costs
   nothing to adopt and aligns exactly with D-009's read-first first run.
4. **Product Hunt.** Supermemory got #1 with 705 upvotes; the channel is free
   and this project has never examined it.
5. **A real benchmark, published honestly.** MemGPT's 958 citations were earned.
   D-005's "evaluation honesty as category leadership" axis has a working
   precedent — and Zep's paper is the counterexample showing what fails: a
   self-published v1 benchmarking against a rival's own metric convinces nobody.
6. **Naming customers.** Zep, Cognee and Letta all do it; Mem0, the best-funded,
   does not. A logo costs a conversation, not a budget, and it is the single
   cheapest credibility asset in the set.
7. **Not renaming.** Letta's rename froze `pymemgpt` and cut the product from
   the paper that made it. Free to avoid; expensive to undo.

### Does not transfer, or is purchased

1. **The MemGPT paper.** ✅ 958 citations from a UC Berkeley lab with a genuine
   research contribution. Not reproducible on demand, and Zep's attempt to fake
   the shape of it produced nothing.
2. **Cloudflare patronage.** A CTO personally asking you to build the product
   and then investing is not a channel; it is luck plus prior employment.
3. **A hyperscaler partnership** — and per §3.2, it is worth less than it looks
   anyway, since AWS shipped a competing first-party memory tool into the same
   namespace.
4. **SOC 2 Type II and a HIPAA BAA.** Zep's enterprise motion rests on them.
   They cost real money and calendar time.
5. **A DeepLearning.AI course.** Letta's is an institutional relationship.
6. **Surviving a pivot.** Letta tore out its server surface and kept going
   because it had $10M. A solo founder gets fewer attempts.
7. **Funded runway itself.** Every company here is currently paid for by
   investors, not customers. That is the advantage, and it is the one that
   cannot be reproduced — which cuts both ways: CRED cannot buy time, but it
   also cannot afford to spend two years pre-revenue the way Letta has.

---

## 7. What this implies for CRED's current decisions

*Marked as interpretation, per `.claude/rules/docs.md` §2 — everything above is
the scan.*

### Answers D-012's question directly

D-012 defines success as *"do what Mem0 does, at comparable quality, and cover
teams and organizations well enough to be worth adopting,"* and names this
document as the place its question gets answered: **what does Mem0 do well, and
what does it not do for teams?**

**What Mem0 does well, and CRED must match:**

1. **Distribution by framework vendoring** (§3.1). This, not the retrieval, is
   the asset. ✅ 71 code hits in `crewAIInc/crewAI`, a first-party package in
   `run-llama/llama_index`, a tool in `strands-agents/tools`, 30 integration
   docs in-repo. Parity on features without parity on this channel produces a
   better product nobody installs.
2. **Provider portability** — 24 vector stores behind an 11-method interface, at
   ~250 LOC each ([mem0.md](mem0.md) §6).
3. **Extraction quality and explainable hybrid scoring** — already inventoried
   as the top-3 steal list in [mem0.md](mem0.md).
4. **Release cadence** ✅ — 13 releases in three months.

**What Mem0 does not do for teams — and this is sharper than expected:**

✅ Mem0 **removed** the team primitives rather than never having built them.
v2.0.0 (2026-04-14) deleted `org_id` and `project_id` from the `MemoryClient`
constructor and every method signature (PR #4740). Combined with what
[mem0.md](mem0.md) found in the source — `get`/`update`/`delete` take no scope
at all, the OpenMemory ACL has **zero writers**, the REST server binds the
authenticated principal to `_auth` and never reads it — the position is that
**Mem0 has no team layer, is not building one in the open, and actively
withdrew the two fields that would have anchored one.** It also names no
customers ✅, ships ADD-only writes so contradictions accumulate, and moved
graph and temporal reasoning behind the paid platform.

**The parity bar is lower than this project has been assuming.** ✅ Mem0 was
**four people** at its Series A. The scope D-006 already reduced to 4–8 MCP
tools plus hybrid retrieval plus a curation worker is a realistic solo target
over a long horizon — D-001's stated timeline.

**The one caveat D-012 should carry forward.** The three companies that tried
the governed-team version all retreated from it inside twelve months (§4). Under
D-012 that is no longer an existential signal, but it is a *design* signal, and
D-012 keeps the evidence bar high for design questions. The most likely reading
is that the team layer is not hard to build — it is hard to make someone adopt
at rung 1, which is exactly what D-003 already anticipated and what
[mem0.md](mem0.md)'s unwired-governance finding demonstrates in code.

### Contradicts D-007's price anchor — this is the clearest correction

D-007 sets the operating parameter: **"Price anchor is $20–40 per seat per
month."** ✅ **Every company in this scan meters usage, and two explicitly
charge nothing per user:**

| Company | Unit | Seats |
|---|---|---|
| Mem0 | add + retrieval requests | "Unlimited end users" on every tier |
| Zep | ingest credits (1 per 350 bytes) | **unmetered, free** |
| Cognee | $2.50 per 1M tokens | **unlimited, free** |
| Supermemory | credit-wrapped usage | caps, not the billing unit |
| Letta | **$0.10 per active agent per month** | not the unit |

Context7 already put a dent in this ([context7.md](context7.md): Pro at
$10/seat, Enterprise scaling *down* to $2.50/user). Five more data points now
say the same thing more strongly: **a developer memory infrastructure product is
priced by consumption, not by human.** Letta's per-active-agent meter is worth
special attention — in a world where one developer runs many agents, **seats are
the wrong denominator by construction.**

D-007's $20–40 figure came from developer-tooling seat benchmarks. Those are the
wrong comparables. **This should be revisited before any pricing appears in the
PRD or a launch page,** and the D-007 note that Memco's $599/contributor/year is
"outside the market" gets stronger, not weaker — it is outside the market on the
*unit*, not just the amount.

### Supports D-006 — distribution is the edge, and the channel is now named

D-006 bet on distribution without specifying a mechanism, and D-009's open
tension records that Context7 proved the *mechanics* reproducible while staying
silent on whether they work without pre-existing demand.

**This scan names the mechanism concretely: get vendored into the framework
repos.** ✅ It is verified in three separate monorepos, it is free, it is
solo-reproducible, and Mem0's footprint in `crewAIInc/crewAI` (71 hits) against
Letta's (0) and Cognee's (0) tracks the difference between the category leader
and the also-rans better than funding does.

It does **not** resolve D-009's open tension. Framework vendoring distributes to
developers who have already decided they want memory. It does not create the
demand. Both scans now land on the same unresolved gap.

### Supports D-009 — read-free, write-metered is the category's own answer

D-009 makes CRED's first run read-only. ✅ Zep charges **zero credits for
retrieval, storage and users**, metering only ingest. Cognee's free tier is
unlimited API calls. The category has independently converged on *reads are
cheap, writes are the product* — which means D-009's read-first design is not
just a trial-friction decision, it is the shape of the eventual business model
too. Those can be the same decision.

### Sharpens D-004's risk #2 with a specific instance

D-004 risk #2 says platforms price the memory primitive at zero. ✅ AWS shipped
`agent_core_memory` (Bedrock AgentCore Memory) into `strands-agents/tools`
alongside `mem0_memory.py`. **The partner became the competitor inside the same
tool namespace, and Mem0's "exclusive provider" language survived the change.**
This is what platform absorption looks like in practice, and it is the single
strongest reason not to build a strategy on a partnership.

### Complicates D-005's auditability axis — in CRED's favour, for once

D-009 recorded disconfirming evidence that auditability is weak as a wedge:
Context7's users did not care about an unexplained "Benchmark Score."

✅ This scan finds the opposite signal in a different place. The **top two
comments on Mem0's 201-point Show HN**, in 2024, were:

> "How does Mem0 handle the potential for outdated or irrelevant memories over
> time? Is there a mechanism for 'forgetting' or deprioritizing older
> information that may no longer be applicable?"
>
> "Over time, I can imagine there's going to be a lot of sensitive information
> being stored. How are you handling privacy?"

Staleness and governance were the first two questions the audience asked, and
per [mem0.md](mem0.md) Mem0 has answered neither in the two years since — the
reconciler has zero call sites, the ACL has zero writers, and v2.0.0 removed
`org_id`/`project_id` entirely. **Zep, meanwhile, has repositioned onto exactly
this word:** "Managed, **governed**, and served at scale."

This does not overturn D-006's demotion of differentiation. It does mean the
governance/staleness axis is asked about unprompted by real users, and is
currently unanswered by the leader. That is a stronger position than D-005
assumed and weaker than a wedge — **and Zep arriving at the same message with
SOC 2 Type II already in hand is a competitive warning that belongs alongside
it.**

### The uncomfortable finding for D-004 and D-011

Three companies retreated from the governed-organizational version of this
product inside twelve months: Mem0 deleted `org_id`/`project_id` and the
reconciler (2026-04-14 ✅), Letta tore out its server (2026-03-16), Zep killed
self-hosting (2025-04-02). D-011 already demoted sovereignty to a tiebreaker on
one interview. **This scan adds that the org-scoping primitives themselves are
being removed by funded teams who shipped them and then took them out.**

Two readings, and I cannot separate them with the evidence available. Either the
governed-org-memory product is hard to sell and they all found that out, or it
is hard to build and they all deprioritized it while the base product still
needed work. The Mem0 case leans toward the second — they moved graph and
temporal reasoning to the *paid* platform rather than deleting them outright,
which is a monetization move, not an abandonment. **But D-004's risk #5 ("Mem0
tried governance and withdrew it") should now read as a pattern of three, not an
instance of one.**

### A method gap this scan exposed

**Product Hunt has never been examined by this project.** Supermemory took #1
with 705 upvotes and #2 with 440, and reached 28.5k stars and 278k npm
downloads/month with a **peak HN score of 5**. Every demand and graveyard
finding in `spike-demand-and-buyers.md` and `market-landscape.md` is
HN-derived. That is one channel, and it demonstrably misses companies. The
"~10:1 seller-to-buyer ratio" finding in D-007 is a fact about Hacker News; it
may or may not be a fact about the market.

---

## 8. Unverified items

Listed so nobody assumes they were checked.

- ⚠️ **Zep's funding, entirely.** No primary source exists. Aggregators give
  $500K pre-seed / $1.8M seed / $2.3M total — three incompatible figures.
  Crunchbase 403'd. Would need Zep to publish.
- ⚠️ **Revenue for all five companies.** None disclosed. `getlatka`'s Zep (~$1M
  2024) and Letta ($1.4M ARR) figures are modeled estimates and the Zep page
  reports "$0 raised," which is false — both discarded, neither confirmed nor
  refuted.
- ⚠️ **Team sizes** except Mem0 (4, TechCrunch) and Zep (5, YC page — possibly
  stale). Cognee ~17 is a headcount off an about page. Letta and Supermemory
  unknown.
- ⚠️ **Whether Letta has raised since Sept 2024.** No announcement found in 26
  blog posts or in search. Reported as "none announced," not "none raised."
- ⚠️ **All pricing** — vendor pages, fetched once on 2026-07-20, not
  cross-checked against invoices or third parties.
- ⚠️ **All customer logos and case-study metrics** — vendor-published,
  unaudited. Supermemory's "37.4% lower latency vs. Mem0," Cognee's "70+
  companies," Letta's Bilt "1 million agents" and Kognitos "half-million dollar
  expansion" are all self-reported.
- ⚠️ **Whether Cognee's 70+ companies pay.** The wording describes deployment.
  Their free tier is unlimited users and unlimited API calls.
- ⚠️ **The Apple/AWS/Google/Microsoft logo strip on cognee.ai** — role
  undetermined. Do not cite as customers.
- ⚠️ **Which channel drove Cognee's 28.6k stars.** Undetermined. Star promotion
  cannot be ruled out.
- ⚠️ **Git history of any competitor.** ✅ The local clones at
  `/Users/canh/Solo/OSS/{mem0,letta}` are shallow — 1 commit each. All
  removal/deprecation evidence here comes from changelogs and blog posts, which
  are vendor artifacts. A full clone would allow checking what was removed
  *quietly*, which is the harder-to-spin evidence the task asked for and this
  scan could not obtain.
- ⚠️ **Star history over time** for any repo. GitHub gives only current totals.
- ⚠️ **Cloud marketplace listings** (AWS/Azure/GCP) for any of the five. None
  found; absence of evidence only.
- ⚠️ **Discord/community size** for Mem0, Zep, Cognee, Supermemory. Only
  Letta's (11,740) was obtainable ✅. Supermemory's invite links returned
  "Unknown Invite."
- ⚠️ **MemGPT citation count precision.** Semantic Scholar reports 958;
  OpenAlex reports 47 for the same work. OpenAlex undercounts arXiv preprints;
  958 is used here, but the spread is two orders of magnitude — cite the source.
- ⚠️ **Mem0's Q3 2025 "186M API calls"** and "30% MoM growth" — company-stated,
  never independently confirmed, and no figure has been published since.
- ⚠️ **Zep's "scaled 30x in 2 weeks"** — blog headline seen; post body not
  fetched; baseline unknown.
- ⚠️ **Zep's "February 2026 deprecation wave"** — the page exists at
  `help.getzep.com/february-2026-deprecation-wave`; contents not fetched.
- ⚠️ **Whether Cloudflare markets Supermemory.** No evidence either way. The
  confirmed relationship is employment plus an angel investment.
- **Reddit** was not searched (blocked in this environment, as in all prior
  scans). **X/Twitter** was not fetchable; Supermemory's founder-audience
  channel is inferred from Product Hunt and TechCrunch, not measured.
- **PyPI caveat:** the `overall` endpoint returns ~180 days, so no series
  predates January 2026, and **July 2026 is a partial month** (through ~19th)
  that must not be compared against full months.
