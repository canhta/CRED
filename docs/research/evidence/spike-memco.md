# Competitive Spike: Memco (The Memory Company) / Spark

**Date:** 2026-07-20
**Method:** WebFetch against memco.ai, docs.memco.ai, github.com/memcoai, arxiv.org. **WebSearch was unavailable** (session budget exhausted at 200/200), so all third-party/social evidence — LinkedIn headcount, funding databases, HN/Reddit threads, podcasts, Moonsong Labs relationship — is **NOT VERIFIED** and marked as gaps below.

Legend: **[V]** = verified by fetching the cited URL. **[I]** = inference. **[NF]** = searched/fetched, not found.

---

## Verdict

1. **The shipped product is dramatically narrower than the marketing.** The website sells "organizational memory"; the actual documented product is a **query/share/feedback knowledge network for error messages and coding solutions**. Docs state plainly: *"Only error messages and solutions are shared."* **[V]** The gap between `memco.ai` copy and `docs.memco.ai` reality is their single biggest exposure.

2. **"On-prem" is a sales-gated promise with zero public artifacts.** The only MCP config in their public marketplace points at a hosted endpoint: `{"type":"http","url":"https://spark.memco.ai/mcp"}` **[V]**. No container images, no Helm chart, no self-host docs, no licensing terms, no hardware requirements found anywhere public **[NF]**. Their own docs page on privacy/security *does not mention on-prem, VPC, or residency at all* **[V]** — only the pricing page's Enterprise bullet does. This is CRED's clearest wedge.

3. **The SWE-bench claims are not in the paper and have no published methodology.** The arXiv abstract (2511.08301) **never mentions SWE-bench** **[V]** — it reports a 30B open-weights model matching a larger SOTA model, and a 98.2% "helpfulness band" figure. The −40%/−34%/−31%/½-variance numbers appear **only as website marketing** with no variant named, no baseline named, no harness, no scaffold, no n, no seeds, no repro artifact **[V/NF]**.

4. **Four people, founded 2025, funding undisclosed.** Careers page: *"Four people. One memory layer."* and *"We've stayed small on purpose."* **[V]** The `/investors` page is **password-gated** ("Core investors", "Friends & family") and discloses nothing **[V]**. No round, amount, or investor name is public.

5. **Traction evidence is close to nil.** GitHub org: **2 followers**, top repo `spark-cli` at **5 stars, 0 forks, 0 open issues** **[V]**. Seven repos, all MIT, all thin. No customers, case studies, logos, or testimonials found **[NF]**. No community (Discord/Slack) found **[NF]**.

6. **Pricing is public and beatable: $599/user/year, 5-seat minimum ($2,995/yr floor).** **[V]** Deliberately not usage-metered — *"No charge per memory call, retrieval, or agent run."* **[V]** An open-source self-hostable competitor attacks both the price floor and the trust-the-cloud assumption simultaneously.

---

## 1. Site Map

`sitemap.xml` returns **404** **[V]**. Structure reconstructed from homepage nav/footer **[V]**:

| Path | Status |
|---|---|
| `/how-it-works`, `/pricing`, `/blog`, `/about`, `/careers` | live |
| `/field-guide`, `/getting-started`, `/about/brand` | live |
| `/use-cases/{private-equity,customer-service,open-source-ai,engineering,consulting,enterprise-ai}` | live |
| `/solutions/{stop-agent-rework,team-rollout,governed-memory,model-portable,long-running,memory-audit}` | live |
| `/privacy`, `/terms`, `/investors` | live (`/investors` gated) |
| `docs.memco.ai`, `spark.memco.ai/auth/login`, `spark.memco.ai/dashboard` | external |

Homepage taglines **[V]**: *"Context is rented. Memory is owned."* / *"Turn agent work into institutional memory."* / *"Your memory is the asset. Models come and go."*

Homepage also exposes a product split via query param: `/?spark=private` (Private Org memory) and `/?spark=public` (Public Community) **[V]**.

---

## 2. Pricing — https://memco.ai/pricing **[V]**

| Tier | Price | Notes |
|---|---|---|
| **Developer** | Free | One developer. *"Public memory access without monthly caps"*, *"Community support — cloud only"* |
| **Team** | **$599 per user / year**, billed annually, **5-user minimum** | Private team memory, cross-tool (Claude Code, Cursor, Codex, Copilot), curation (*"Review, edit, deduplicate, and retire shared memories"*), *"Memory reuse, agent activity, and token-savings insights"* |
| **Enterprise** | Custom | *"SSO / SCIM, RBAC, and audit logs"*, *"Policy-seeded memory and provenance"*, *"Client / programme memory scoping"*, **"VPC, on-prem, and UK/EU residency"**, *"Dedicated support and paid pathfinders"* |

Philosophy quote: *"Priced around teams, not every agent call"* — *"No charge per memory call, retrieval, or agent run."*

Note: **"paid pathfinders"** is consulting-style revenue — a tell that Enterprise deployments are bespoke, not productized.

---

## 3. Technical Surface **[V]**

### Distribution
- npm: `npm install -g @memco/spark` (npmjs.com returned **403** to WebFetch, so downloads/version **[NF]**)
- curl: `curl -fsSL https://raw.githubusercontent.com/memcoai/spark-cli/main/install.sh | bash`
- Claude Code plugin: `/plugin marketplace add memcoai/marketplace` then `/plugin install spark-cli@MemCo`
- Codex: `codex plugin add spark-cli@MemCo`; Cursor via Settings → Plugins → Add Marketplace

### Marketplace (`memcoai/marketplace`, v1.3.0) — four plugins **[V]**
`spark-mcp`, `spark-team-mcp`, `spark-cli`, `spark-team-cli` — i.e. a public edition and a Teams edition of each of the MCP and CLI surfaces.

### The MCP server config — the single most important artifact found **[V]**
`plugins/spark-mcp/.mcp.json`:
```json
{ "spark-memory": { "type": "http", "url": "https://spark.memco.ai/mcp" } }
```
**Remote HTTP MCP, hard-coded to their cloud. No command, no args, no env vars, no configurable base URL.** There is no public mechanism to point the client at a self-hosted server.

### The 4 MCP tools (from /how-it-works) **[V]**
```
search(query, context)        -> Retrieve relevant memories
create_memory(content)        -> Store a new insight
enrich_memory(id, content)    -> Refine an existing memory
share_feedback(id, score)     -> Signal whether memory was useful
```
No JSON schemas, types, or response shapes are published for these **[NF]**.

### CLI reference — https://docs.memco.ai/docs/cli **[V]**
Commands: `query`, `share`, `share-task`, `feedback`, `login`, `logout`, `whoami`, `init`, `enable`, `disable`, `status`, `update`, `uninstall`, plus global flags.

`spark query` full syntax **[V]**:
```
spark [--pretty] [--no-color] [--api-key <key>] query "<task or error>" \
      [--tag TYPE:NAME[:VERSION] ...] [--xml-tag '<tag .../>' ...]
```
- `--tag TYPE:NAME[:VERSION]` — *"Add a semantic tag to scope the search. Repeatable."* e.g. `--tag language:python:3.11`, `framework:next:14.2`
- Global flags **must precede** the subcommand.

Response shape: JSON with `session_id` and `recommendations` (ranked array). Docs hedge: *"The exact shape may evolve — agents should treat the JSON as the contract and the human-readable form as a convenience."* **[V]** — **no versioned API contract, no OpenAPI spec found [NF]**.

Other commands: `spark share <session-id> --title "..." --content "..."`; `spark feedback <session-id> --feedback "<feedback>"` with `relevant` and `correct` flags.

Auth: OAuth 2.0 Authorization Code + PKCE (interactive); `SPARK_API_KEY` env var (CI); legacy API keys from `spark.memco.ai/dashboard`. Credentials at `~/.spark/settings.json` / `./.spark/settings.json`, mode `0o600`. HTTPS TLS 1.2+, *"There is no HTTP fallback."* **[V]**

**Critical:** the CLI's `session_id` model couples share/feedback to a prior query. There is no documented bulk import, backfill, or export path **[NF]** — no way to get your memory *out*.

---

## 4. Trust / Ranking Model **[V]**

`docs/concepts/how-recommendations-work` and `docs/concepts/knowledge-network` describe **four qualitative factors only**:
1. **Production verification** — *"Solutions that have been applied and confirmed working in production environments receive the strongest signal."*
2. **Recency** — *"A fix verified last week for Next.js 14.2 outranks a solution from two years ago for Next.js 12."*
3. **Tag matching** — exact match > version proximity > multi-type coverage.
4. **Community feedback** — `relevant` and `correct` flags adjust ranking.

**The public docs contain NO Bayesian posterior, no prior/update rule, no BM25 or vector-hybrid description, no weighting coefficients, no formulas** **[V, explicit absence]**. The Bayesian-trust and hybrid-ranking story exists **only** in the arXiv paper and the `/how-it-works` marketing page. A customer cannot audit, tune, or reason about ranking. No technical deep-dive blog post on the trust math was found across all 7 blog posts **[NF]**.

---

## 5. On-Prem / Self-Hosting **[NF — the standout gap]**

Evidence of absence, each verified:
- `/pricing` Enterprise: *"VPC, on-prem, and UK/EU residency"* — one bullet, no detail **[V]**
- `docs.memco.ai/docs/teams/privacy-and-security`: **does not mention** on-prem, VPC, self-hosting, residency, SOC 2, or model providers **[V]**
- `/solutions/governed-memory`: contains **no** deployment topology, licensing, or residency content — pure positioning + "book a demo" **[V]**
- `.mcp.json` hard-codes the hosted endpoint **[V]**
- No container image, compose file, Helm chart, Terraform, or install-server doc in any of the 7 public repos **[V/NF]**

**Assessment [I]:** on-prem is a sales conversation, delivered as bespoke engagements ("paid pathfinders"), not a shipped, documented, downloadable artifact. Whether it is genuinely self-hostable or a license-server arrangement is **undetermined from public sources** — but there is nothing a prospect can evaluate without talking to sales. The MIT-licensed repos are **clients only**; the server is closed.

---

## 6. The SWE-bench Claims — Skeptical Read **[V]**

Exact marketing copy from `/how-it-works`:
> "−40% LLM costs at steady state" (SWE-bench)
> "−34% Wall-clock time per task" (end-to-end)
> "−31% Agent steps to completion" (averaged across tasks)
> "½× Outcome variance — outcomes get more predictable"

Their only methodology statement, verbatim:
> "On controlled SWE-bench runs, the first task is a cold start. Every task after that benefits from accumulated knowledge. These are the steady-state numbers we measure — and the variance reduction is, in our view, the most important of them."

**The arXiv paper does not contain these claims.** Abstract of 2511.08301 (submitted 11 Nov 2025, v1 only, 32KB) reports entirely different results: a 30B open-weights model matching *"a much larger state-of-the-art model"*, and *"helpfulness levels of up to 98.2% in the top two (out of five) qualitative helpfulness bands."* **SWE-bench is not named in the abstract at all.** **[V]**

Not disclosed anywhere **[NF]**:
- Which SWE-bench variant (Verified / Lite / full / Multimodal / bash-only)
- Baseline model, scaffold, and agent harness
- Sample size, task selection, seeds, number of runs
- What "steady state" means operationally — how many warm-up tasks are discarded before measuring
- Whether the memory was populated from the *same* SWE-bench task distribution (a **contamination risk**: memories harvested from earlier SWE-bench instances of the same repos would leak repo-specific fixes into later tasks — this is exactly what "every task after that benefits from accumulated knowledge" describes)
- Cost accounting method for the −40% (their own retrieval/curation LLM calls counted or not?)
- Any repro script, config, or logs

**Attack line for CRED [I]:** the headline number is measured on a setup where the system has already seen closely related tasks, and "resolve rate" — the metric SWE-bench actually exists to measure — is conspicuously **absent** from all four claims. They report cost, time, steps, and variance. Not correctness.

---

## 7. Company **[V]**

- **Founded:** 2025. **HQ:** London, San Francisco & Stockholm.
- **Entities:** *"© 2026 Memco Labs, Inc & Memco Labs Ltd"* (US + UK dual entity)
- **Headcount:** *"Four people. One memory layer."* — *"We've stayed small on purpose."* (careers page). LinkedIn unverified **[NF]**.
- **Claims SOC 2 certified** (about page) — no report, auditor, or trust center found **[NF]**

**Team (about page, verbatim):**
- **Scott Taylor** — Co-Founder & CEO. *"Former Global Head of AI Products at AIG Investments ($320B AUM), building machine-learning trading systems through the pre-GPT era."*
- **Valentin Tablan** — Co-Founder & CTO. *"Former Lead Scientist for Amazon Alexa, with 20+ years at the cutting edge of natural-language and knowledge-based AI."*
- **Kristoffer Bernhem** — Co-Founder & Principal AI Engineer. Previously **Legora** and **Sana**; *"leads the retrieval and ranking layers."*

**Paper authors (7) — 4 more than the about page lists [V]:** Valentin Tablan, Scott Taylor, **Gabriel Hurtado**, Kristoffer Bernhem, **Anders Uhrenholt**, **Gabriele Farei**, **Karo Moilanen**. Karo Moilanen is a paper author but **not on the about page** **[V]** — likely advisor/fractional or departed **[I]**.

**Not verified [NF]:** funding rounds, investors, amounts, valuation; the ex-Gartner / ex-Ontotext background hypothesis (Tablan's GATE/Sheffield NLP and Ontotext lineage is plausible given the 20+yr NLP claim but **unconfirmed here**); **Moonsong Labs relationship — no evidence found on any fetched page**.

**Hiring (3 roles, all remote, UK/EU/US overlap):** AI Engineer; Design Engineer, Agentic Interfaces; GTM Engineer. *"If you join now, you will own real surface area from day one."*

---

## 8. Traction **[V]**

GitHub org `memcoai` — bio *"building shared memory for AI agents"*, **2 followers**, members private:

| Repo | Lang | Stars | Last update |
|---|---|---|---|
| `spark-cli` | JavaScript | **5** (0 forks, 0 issues), v0.6.0 rel. 22 Jun 2026 | 24 Jun 2026 |
| `spark-tutorial` | Python | 1 (1 fork) | 18 Feb 2026 |
| `spark-cli-skills` | — | 1 | 22 Apr 2026 |
| `spark-teams-cli-skills` | — | 1 | 22 Apr 2026 |
| `spark-team-skills` | Shell | 0 | 2 Apr 2026 |
| `spark-skills` | Shell | 0 | 2 Apr 2026 |
| `marketplace` | Shell | 0 | 9 Jul 2026 |

All MIT. Combined **~8 stars across the entire org.**

**Not found [NF]:** customers, logos, case studies, testimonials, Discord/Slack, conference talks, podcasts, HN/Reddit threads, press coverage. (Caveat: HN/Reddit/podcast searching requires WebSearch, which was unavailable — treat as *unsearched*, not *absent*.)

**Blog — 7 posts, all thought-leadership, zero engineering depth [V]:** "Coding agents are the wedge. Organizational memory is the prize." (2 Jul 2026); "Knowledge Management Systems Are Always Obsolete. Agents Can Fix That" (20 May); "Why Active Agentic Memory is the Next Shift" (4 May); "Your Team Knows More Than Anyone On It" (1 May); "Your Agent's Memory Is a Markdown File. That's a Problem." (17 Apr); "Your Coding Agent Remembers Everything, Until It Doesn't" (7 Apr); "Continual Learning for Enterprise AI Needs a Memory Layer" (5 Apr).

---

## 9. Weaknesses & Gaps

| # | Weakness | Evidence |
|---|---|---|
| 1 | **Product ≠ marketing.** Sells org memory; ships an error/solution network. *"Only error messages and solutions are shared."* | **[V]** README, docs |
| 2 | **No self-host path.** MCP hard-coded to `spark.memco.ai/mcp`; no images/charts/docs; security docs silent on on-prem. | **[V/NF]** |
| 3 | **Closed server, thin OSS.** MIT repos are clients only; ~8 stars org-wide. | **[V]** |
| 4 | **Cloud-only, online-only.** *"Does Spark work offline? No."* Free tier is *"cloud only."* | **[V]** |
| 5 | **No data egress.** No export, bulk import, or backfill documented — memory is captured in their cloud. | **[NF]** |
| 6 | **Unstable API contract.** *"The exact shape may evolve"*; no OpenAPI, no MCP tool schemas, no versioning policy. | **[V]** |
| 7 | **Opaque ranking.** Zero formulas public; can't audit, tune, or explain why a memory ranked. | **[V]** |
| 8 | **Unreproducible headline benchmark**, absent from own paper, omits resolve rate, contamination risk unaddressed. | **[V]** |
| 9 | **No redaction/DLP.** *"Spark does not scrub or redact your input — you control what you share."* Real leak risk when an autonomous agent writes memories. | **[V]** |
| 10 | **Indefinite retention of shared memory.** *"Retained indefinitely as anonymized, aggregated patterns"*; usage data +24 months post-account. | **[V]** |
| 11 | **Unnamed subprocessors.** *"Cloud hosting, payment processing, analytics"* — none named; **no LLM provider disclosed**; no residency in policy despite selling UK/EU residency. | **[V]** |
| 12 | **Empty changelog.** `/docs/changelog` shows only "v0.3.0 (latest)" with *"Version history will be documented here as new releases are published"* — while GitHub is at **v0.6.0**. Docs are stale against the shipped CLI. | **[V]** |
| 13 | **Price floor.** $2,995/yr minimum (5 × $599) before any enterprise feature; SSO/RBAC/audit are Enterprise-gated. | **[V]** |
| 14 | **4 people, 3 open roles, 3 timezones**, bespoke "paid pathfinders" — enterprise delivery capacity is the bottleneck. | **[V/I]** |

---

## 10. Unresolved (needs WebSearch in a follow-up session)

Funding/investors/valuation; LinkedIn headcount vs. the "four people" claim; Moonsong Labs relationship; Gartner/Ontotext backgrounds; HN/Reddit/practitioner sentiment; npm download volume (npmjs 403'd); conference talks & podcasts; full arXiv PDF body (only the abstract page was fetched — **the paper's own evaluation section may contain the SWE-bench detail the abstract omits and should be read directly**); the gated `/investors` letters; `/field-guide` and the 6 `/use-cases/*` pages.
