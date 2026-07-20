# Competitive Sweep — Organizational Agent Memory

**Date:** 2026-07-20
**Scope:** Projects positioning as SHARED / TEAM / ORG memory for AI coding agents.
**Method:** GitHub REST API (authenticated `gh`), HN Algolia API, arXiv API, and direct
document fetches of vendor docs. **WebSearch was unavailable** — the session budget was
exhausted (200/200) before this sweep began, so no keyword search engine was used.
Discovery therefore rests on GitHub/HN/arXiv API enumeration plus direct fetches.
Consequence: coverage of *non-GitHub, non-HN commercial* products is the weakest axis
here. See [Negative evidence & gaps](#negative-evidence--gaps).

Every claim is marked **[V]** (verified — read directly from the source page/API/source
code) or **[I]** (inferred). Nulls are stated as nulls.

---

## Verdict

1. **Yes. An open-source Memco already exists, and it is further along than CRED's
   premise assumes.** [caura-ai/caura-memclaw](https://github.com/caura-ai/caura-memclaw)
   — Apache-2.0, self-hostable via Docker Compose, MCP-native — ships *every* axis D-004
   proposed to differentiate on: visibility scopes (`scope_agent`/`scope_team`/`scope_org`),
   four-tier agent trust levels, keystone policy rules, full audit log, contradiction
   detection with supersession, an 8-status decay lifecycle, and provenance-tracked
   crystallization. It claims a production deployment at eToro (NASDAQ: ETOR) with 300+
   agents and 26,500+ memories. **[V]** This is the single most important finding in the
   sweep and it invalidates "no one has built the self-hostable governed org memory layer"
   as a positioning claim.

2. **But it is young and thin, and the category is a swarm, not a winner.** MemClaw is
   ~3 months old (created 2026-04-27), 341 stars, 8 contributors, 2 watchers. **[V]**
   [Commonly](https://github.com/Team-Commonly/commonly) actually owns OSS mindshare at
   **1,261 stars** ("agents and team share one memory", Apache-2.0, RBAC + scoped agent
   tokens + audit log) **[V]**, and [AKB](https://github.com/dnotitia/akb) has the most
   rigorous permission model of anyone (**PostgreSQL-native RBAC** — a real PG role per
   user, vault reader/writer/admin group roles, cross-vault access failing with PG error
   `42501`) though it is **BUSL-1.1, not OSI open source** **[V]**. Behind those sit 60+
   more 2026-vintage projects making overlapping claims, nearly all single-maintainer and
   under 200 stars. The whitespace is not conceptual — it is *execution, evidence, and
   trust*. Nobody has won.

   **Crucial nuance, and the strongest positive signal in the sweep:** GitHub searches for
   `multi-agent memory ACL` returned **3 repos total, max 2 stars**; `memory agents
   multi-tenant permissions` returned **2 results, max 20 stars**; `decision log agents
   MCP` maxed at 10 stars. **[V]** Most of the swarm says "team" and ships per-user
   cross-*tool* memory. Fewer than 15 of ~80 projects surfaced have a real multi-user
   permission model. **The category is crowded at the tagline and nearly empty at the
   permission gate.**

3. **The sovereignty bet is gone; the evidence and verification bets are wide open.**
   Self-hostable + Apache-2.0 + MCP + governance is now table stakes, matched by MemClaw,
   Commonly, memctl, cortex-hub and others. **[V]** What *no one* has published is an honest
   ablation separating experiential memory from documentation retrieval, an epoch curve
   showing compounding, or any adversarial result on poisoning/conflict/staleness. MemClaw
   benchmarks on LoCoMo/LongMemEval — **single-agent benchmarks it openly admits cannot
   measure cross-agent recall or governance** **[V]**. The methodology gap identified in
   the Memco teardown is still unfilled by the entire field. Note a benchmark now exists
   and is unclaimed: **[GateMem: Benchmarking Memory Governance in Multi-Principal
   Shared-Memory Agents](https://arxiv.org/abs/2606.18829)** (2026-06-17). **Verification
   is the thinnest column in the whole market** — only Kage (deterministic checks against
   real code), emem (signed Memory Tokens), UMP (signed bi-temporal records), and brain0
   (signed provenance attestations) attempt anything real. **[V]**

4. **The free tier now does more than expected, but it never crosses the user boundary.**
   Anthropic ships server-managed `claudeMd` pushed org-wide with hourly polling; Cursor
   ships dashboard-managed Team Rules; GitHub ships org custom instructions. **[V]**
   *Authored* text now syncs org-wide at three vendors. But **learned knowledge crosses no
   user boundary at any vendor** — Claude Code auto memory is documented as "machine-local…
   not shared across machines or cloud environments," and Claude.ai memory is explicitly
   per-user. **[V]** Also absent everywhere: ACL/team scoping (Anthropic states
   per-group config is "not yet supported"), provenance, decay, conflict resolution
   (Anthropic: "Claude may pick one arbitrarily"), and semantic retrieval. **[V]**

5. **ByteRover is source-available, not open source, and has pivoted away from vectors.**
   `campfirein/byterover-cli` (formerly Cipher) is **Elastic License 2.0** — 4,925 stars,
   453 forks, 17 contributors. **[V]** Memory unit is a *Context Tree*: hierarchical
   markdown in `.brv/context-tree/`, no vector DB, no graph DB, no embeddings. Team
   sharing is a paid cloud feature (git-semantic version control, shared spaces, RBAC at
   Enterprise). **[V]** It is the traction leader by an order of magnitude but is
   local-first-per-developer at its core, with org features behind the commercial wall.

6. **The nearest-term threat is not another memory startup — it is Tessl and the skills
   registries closing the loop from the other end.** Tessl ($125M, Snyk founder) already
   ships an agent that scans commit history and session logs, "spots a recurring error
   pattern, creates a skill to address it, and opens a PR." **[V]** That is auto-captured
   organizational memory delivered through a distribution channel it already owns. SKILL.md
   is now a cross-vendor open standard (agentskills.io, ~45 adopters incl. Cursor, Copilot,
   Codex, Gemini CLI). **[V]** Any org memory product that cannot export SKILL.md cedes
   distribution.

**Also now confirmed: the category is funded and 2026-vintage.** SageOx raised **$15M seed**
for "shared, structured team memory that agents consult before they act" — the closest
well-capitalized analogue to CRED's thesis. **[V]** YC funded at least four "company brain"
startups in F25–S26 alone (Nessie F25 "a shared context layer for you, your team, and your
agents"; Memory Store; Hyper; Shepherd; plus Osseus on "one permissioned research brain").
**[V]** YC W25/S25 contain **zero** true org-memory companies — the category emerges from
F25 onward. **[V]** Almost all of them compete on *ingestion* (Slack/Gmail/Notion
connectors), not on provenance, verification, or measurable execution.

**Remaining whitespace, honestly stated:** not sovereignty, not MCP, not the governance
feature list. It is (a) *proof* — the ablation, the epoch curve, and GateMem, none of which
anyone has run; (b) *capture without authoring* at team scale, plus the **private→shared
promotion primitive**, which is under-built everywhere (only Vault's
`propose→review→promote` and Grov's private-session→team sync attempt it seriously);
(c) *verification* — the emptiest column in the market and the one CRED's "Traceability"
maps onto; (d) *retrieval economics* past the context-injection wall; and (e)
consolidation/trust in a field of 60+ abandoned-in-6-months single-maintainer repos.

---

## Comparison table — org-level players

Legend: **OSS?** license · **SH** self-hostable · **ACL** per-memory permissions ·
**Prov** provenance/lineage · **Trust** explicit trust/verification mechanism.
Stars/dates from GitHub API on 2026-07-20 **[V]**.

| Project | What it is | OSS? | SH | Stars | Last push | MCP | ACL | Prov | Trust mechanism | Org-real? |
|---|---|---|---|---|---|---|---|---|---|---|
| **[caura-memclaw](https://github.com/caura-ai/caura-memclaw)** | Governed shared memory for agent *fleets*; Postgres+pgvector+Redis | Apache-2.0 | ✅ Docker | 341 | 2026-07-19 | ✅ native, Streamable HTTP, 12 tools | ✅ agent/team/org scopes | ✅ full audit log + crystallization lineage | 4-tier agent trust + keystone policies + contradiction supersession + outcome learning | **YES — strongest** |
| **[ByteRover](https://github.com/campfirein/byterover-cli)** | Context Tree markdown memory for coding agents (ex-Cipher) | ❌ Elastic-2.0 | Partial (local core) | 4,925 | 2026-06-25 | ✅ `brv mcp` | Paid tier (RBAC @ Enterprise) | Source tracking per entry | Importance scoring + recency decay | Partial — org behind paywall |
| **[Hyper](https://heyhyper.ai/) (YC P26)** | "Company brain" — episodes + facts KG over Docs/Slack/Email | ❌ closed | ❌ managed AWS | n/a | — | ✅ + lifecycle hooks | ✅ RBAC, per-fact tags | ✅ every fact carries provenance | Typed edges: "supersedes", "in tension with" | **YES** |
| **[Memco / Spark](https://arxiv.org/abs/2511.08301)** | Managed shared experiential memory, MCP-only | ❌ closed | ❌ | n/a | — | ✅ sole surface | ❌ unaddressed | ❌ unaddressed | Managed curation loop | Claimed, unevaluated |
| **[Commonly](https://github.com/Team-Commonly/commonly)** | OSS workspace where agents + team share one memory; pod-shared vs agent-private | Apache-2.0 | ✅ one command | **1,261** | 2026-07-20 | [I] | ✅ RBAC, scoped `cm_agent_*` tokens, per-pod | ✅ audit log of every agent action | ❌ null | **YES — OSS mindshare leader** |
| **[AKB](https://github.com/dnotitia/akb)** | Git-backed org KB over MCP; Confluence/Notion replacement for agents | ❌ **BUSL-1.1** | ✅ | 73 | 2026-07-20 | ✅ | ✅ **PostgreSQL-native RBAC** (`akb_user_<uid>` roles, vault reader/writer/admin, cross-vault → PG `42501`) | ✅ **hash-chained append-only JSONL**, WORM offload | ❌ null | **YES — strongest ACL** |
| **[SageOx](https://sageox.ai)** | "Hivemind for human–agent teams"; shared structured team memory consulted before acting | ❌ closed | ❌ | n/a | — | [I] | [I] | Records sessions as durable team artifacts, extracts decisions | ❌ null | **YES — $15M seed** |
| **[Dense-Mem](https://github.com/markhuangai/dense-mem)** | Go + Neo4j + Postgres; teams, profiles, API keys, control portal | Apache-2.0 | ✅ | 34 | 2026-07-19 | ✅ Streamable HTTP | ✅ teams + API keys | ✅ **evidence provenance, source fragments stored before claims** | Typed claims + conflict detection; "never silently rewrites facts" | **YES** |
| **[AgentMemoryOS](https://github.com/yamantaka520/Agent-Memory-OS)** | Design reference: private/agent/team/project/global scopes | Apache-2.0 | ✅ | **2** | 2026-07-10 | [I] | ✅ **hard gate before ranking; ACL-safe graph traversal; independent ACL clock for post-hoc revocation** | ✅ `context_pack_report()` explains every include/exclude | ❌ null | Design only, no users |
| **[emem](https://github.com/Vortx-AI/emem)** | Verifiable memory protocol — signed "Memory Tokens" | Apache-2.0 | ✅ | 45 | — | [I] | [I] | ✅ signed | **Cryptographic signatures on every observation** | Adjacent — verification primitive |
| **[omem](https://github.com/ourmem/omem)** | Personal/Team/Org "Spaces" sharing | Apache-2.0 (README) / NOASSERTION (API) | ✅ Docker/musl | 200 | 2026-05-24 | ✅ | Space-tier only | ✅ lineage: who shared, when | Version tracking for staleness | Partial |
| **[cortex-hub](https://github.com/lktiep/cortex-hub)** | Self-hosted memory + AST code intelligence, one MCP endpoint | MIT | ✅ | 57 | 2026-07-15 | ✅ | ❌ null | ❌ null | "quality enforcement" (unverified) | Partial |
| **[memctl](https://github.com/memctl/memctl)** | Cloud MCP server, memory scoped to projects **and organizations** | Apache-2.0 | ✅ Docker | 14 | 2026-05-28 | ✅ stdio | Org/project scope | ❌ null | ❌ null | Partial |
| **[Grov](https://github.com/TonyStef/Grov)** | Auto-captures private AI sessions → shared team memory, auto-injects | Apache-2.0 | ✅ | 193 | **2026-01-29 (stale)** | [I] | ❌ null | ❌ null | ❌ null | Capture loop is right; **dormant 6mo** |
| **[Klear-Team-Brain](https://github.com/Asklear/Klear-Team-Brain)** | Team AI sessions + GitHub + docs → self-hosted git truth store over MCP | Apache-2.0 | ✅ | 100 | 2026-07-04 | ✅ | ❌ null | git history | ❌ null | Partial (1 contributor) |
| **[Vault-Agent-Memory](https://github.com/zycaskevin/Vault-Agent-Memory)** | Local-first memory *governance*: shared, reviewable, auditable; SQLite+MCP | Apache-2.0 | ✅ | 43 | 2026-07-20 | ✅ | review workflow | ✅ auditable | Human review gate | Partial (1 contributor) |
| **[Kage](https://github.com/kage-core/Kage)** | Memory verified against actual code; plain files in repo, shared via git | GPL-3.0 | ✅ (no DB/account) | 28 | 2026-07-10 | ✅ | git perms | in-repo files | **Verification against real code + freshness** | Partial — novel trust model |
| **[basic-memory](https://github.com/basicmachines-co/basic-memory)** | Markdown+SQLite local / Neon+S3 cloud; "Teams" shared workspaces | AGPL-3.0 | ✅ | 3,462 | 2026-07-20 | ✅ | ❌ workspace-level only | ❌ null | ❌ null | Partial |
| **[agent-in-sync](https://github.com/agentinsync/agent-in-sync)** | Shared KB of battle-tested solutions across teams | AGPL-3.0 | ✅ | 5 | 2026-05-26 | ✅ | ❌ null | ❌ null | ❌ null | Concept only |
| **[krimto](https://github.com/krimto-labs/krimto)** | Team memory layer: markdown in git, **user→team→org hierarchy**, cross-vendor MCP | Apache-2.0 | ✅ | 6 | 2026-06-02 | ✅ | hierarchy | git | ❌ null | **Concept twin, abandoned** |
| **[brain0](https://github.com/Brain0-ai/brain0)** | Decision graph linking commits→agent prompts; DLP audit, signed attestations | Apache-2.0 | ✅ offline | 464 | 2026-07-19 | ✅ MCP memory | DLP audit | ✅ **signed provenance attestations** | Evidence-driven risk scoring | Adjacent — provenance-first |
| **[Tessl](https://tessl.io/)** | Skills registry + agent that auto-authors skills from failures via PR | ❌ closed | ❌ | n/a | — | — | workspace | versioned PRs | Quality/Impact/Security scores, Snyk scan | **Adjacent, high threat** |
| **[MCP official `memory`](https://github.com/modelcontextprotocol/servers/tree/main/src/memory)** | Reference KG server: entities/relations/observations | MIT | ✅ | 88,646 (repo) | 2026-07-10 | ✅ stdio | ❌ **none** | ❌ none | ❌ none | **No — toy** |

---

## 1. Is there an open-source Memco? — the definitive answer

### YES: caura-ai/caura-memclaw

**[V]** — all facts below read directly from the repo README and GitHub API on 2026-07-20.

| Attribute | Value |
|---|---|
| Repo | https://github.com/caura-ai/caura-memclaw |
| Homepage | https://memclaw.net (managed tier) — "A Caura.ai project" |
| License | **Apache-2.0** (trademark reservation on "MemClaw"/"Caura" per Apache §6) |
| Stars / forks / contributors / watchers | 341 / 45 / 8 / 2 |
| Created / last push | 2026-04-27 / 2026-07-19 |
| Open issues | 56 |
| Stack | FastAPI, PostgreSQL + pgvector, Redis, Docker Compose |
| Self-hosting | ✅ `docker compose up -d`; **air-gapped supported** (`pull_policy: never`), local embedder `BAAI/bge-m3` via HuggingFace TEI → zero external API calls |
| Integration | MCP native at `/mcp` (**Streamable HTTP**), 12 tools; also REST + Python/TS clients |
| Funding | **NULL** — not disclosed on repo or case study |

**Memory unit / data model.** A memory is a single `content` string; a single-pass LLM
enrichment classifies it into one of **14 memory types** and generates title, summary,
tags, importance weight, PII flags, and extracted entities. Entities and relations form a
live knowledge graph with semantic entity resolution (auto-merge at >0.85 cosine). A
separate JSONB document store handles exact-field lookups. **[V]**

**ACL — yes, three-level.** Every memory is stamped at write time with a visibility scope:
`scope_agent` (private), `scope_team` (fleet-wide, default), `scope_org` (cross-fleet).
"Cross-fleet recall is permissioned, not open." Row-level DB tenant isolation. Agent-scoped
credentials minted via `POST /api/v1/admin/agent-keys/provision`. **[V]**

**Provenance — yes.** Full audit log of "every write, delete, and transition logged with
tenant and scope context," exposed at `GET /audit-log`. Crystallization merges near-duplicates
"with full provenance." Contradiction chains are queryable at
`GET /memories/{id}/contradictions`. **[V]**

**How it decides what to trust — four mechanisms:** **[V]**
1. **Agent trust tiers** — four levels gating cross-fleet read/write/delete. Keystone
   authoring requires trust ≥1 for own-scope, ≥2 for fleet/tenant scope.
2. **Keystone policies** — mandatory governance rules read once per session that
   "override conflicting user instructions."
3. **Contradiction detection** — RDF triple comparison + LLM semantic analysis, with
   automatic supersession and chain tracking.
4. **Outcome-based learning ("Karpathy Loop")** — agents report success/failure after
   acting on a recalled memory; the system reinforces what works and auto-generates
   preventive `rule`-type memories on failure. Plus per-agent retrieval tuning (top_k,
   min_similarity, graph hops) learned from feedback.

**Lifecycle/decay:** 8-status automation retiring stale data. **[V]**

**Skill Factory** (opt-in, off by default) is a governed skills lifecycle on top:
`candidate → staged → active` with `rejected`/`quarantined`/`stale`/`deprecated` exits, six
automated gates plus a "Sentinel" content scan, and a **Skills Inbox** REST surface where a
human operator approves/edits/defers/rejects. "An agent write lands as `staged`, never
instantly `active`." A server-side resident, **Forge**, mines memory + outcome signals and
distills repeated successful procedures into skill candidates. **[V]**

> Note this directly collides with CRED's "Law 2 (no human review queue)" — MemClaw
> concluded the opposite for skills specifically and built the queue, while keeping the
> memory path fully automatic. Worth treating as a considered counter-position, not an
> oversight.

**Traction claim — VENDOR-STATED, NOT INDEPENDENTLY VERIFIED:** eToro (NASDAQ: ETOR) runs
300+ agents ("Claws" — DevClaw, SecurityClaw, QAClaw), 291 distinct writing agent IDs,
26,500+ memories, 1,372 shared skills, 23ms p50 search, 96–99% token savings.
Source: https://memclaw.net/use-cases/etoro-company-brain/

**Where MemClaw is weak — the actual opening:**
- **Evidence.** Benchmarks are LoCoMo 77.6% / LongMemEval 72.5% — and the README itself
  concedes these "can't measure cross-agent recall, outcome propagation between agents,
  fleet-scoped visibility, or governance-aware retrieval." **They have the same unfilled
  evidence gap as Memco.** No ablation, no epoch curve, no poisoning/conflict experiment.
- **Bus factor.** 8 contributors, 2 watchers, 56 open issues, 3 months old.
- **Single reference customer**, self-reported, no third-party confirmation.
- **Coding-agent specificity.** It is built for generic "agent fleets" (business agents at
  eToro), not for the software-engineering workflow — no code-awareness, no repo/commit
  linkage, no AST or diff grounding. Kage and brain0 attack that axis; MemClaw does not.

### Runners-up that also qualify as "open-source org memory"

- **[memctl](https://github.com/memctl/memctl)** — Apache-2.0, self-hostable via Docker,
  MCP server with memory "scoped to projects **and organizations**," `MEMCTL_ORG` env,
  `org` MCP tool, "organization defaults that apply across all projects." Explicitly
  "Your team shares one brain." **But: 14 stars, 0 forks, last push 2026-05-28.** **[V]**
- **[omem](https://github.com/ourmem/omem)** — three-tier Personal/Team/Organization Spaces
  with shared-memory lineage and version tracking. **Security concern [V]:** the identity
  model is "the returned `id` **is** your API Key" — one UUID as both tenant identifier and
  bearer credential, sharing done by passing target API keys around, no documented
  revocation. Not suitable for org-wide knowledge as designed.
- **[krimto](https://github.com/krimto-labs/krimto)** — near-verbatim CRED concept
  ("markdown files in git, user→team→org hierarchy, cross-vendor MCP server, Apache-2.0")
  but **6 stars, 0 forks, dead since 2026-06-02.** **[V]** Evidence the idea is obvious;
  also evidence that obviousness alone doesn't build a project.

### The other genuinely-organizational OSS players

**[Commonly](https://github.com/Team-Commonly/commonly)** — **1,261 stars**, Apache-2.0,
created 2025-02-03, pushed 2026-07-20. **[V]** "Open-source workspace where your agents and
team share one memory. Any runtime, your infra — self-host in one command, no per-agent
fees." Memory model is an explicit two-level split: **pod-shared vs agent-private**. RBAC
with scoped agent tokens (`cm_agent_*`), per-pod access control, audit log of every agent
action. **[V]** It is a full workspace product rather than a memory library — broader scope
than CRED, and it owns OSS mindshare in this category by star count.

**[AKB](https://github.com/dnotitia/akb)** — 73 stars, **BUSL-1.1 (not OSI open source)**,
pushed 2026-07-20. **[V]** Positions as a Confluence/Notion replacement for agents:
"organizational memory for AI agents, git-backed KB served over MCP." **The most rigorous
permission enforcement found anywhere in this sweep:** permissions live in the database
engine, not application code — each user gets a real PostgreSQL role `akb_user_<uid>`, each
vault has `reader`/`writer`/`admin` group roles, and cross-vault access fails with PG error
`42501`. Audit is a **hash-chained append-only JSONL** with WORM bucket offload, producer-only
(ships to your SIEM). MCP tools `akb_grant` / `akb_revoke` / `akb_set_public`. **[V]**

**[AgentMemoryOS](https://github.com/yamantaka520/Agent-Memory-OS)** — **2 stars**,
Apache-2.0, created 2026-07-10. **[V]** No traction whatsoever, but the **best-articulated
ACL design in the set** and worth reading as a reference: private/agent/team/project/global
scopes; "visibility is a hard gate enforced *before* ranking, never a soft score"; ACL-safe
graph traversal so invisible nodes are untraversable; an **independent ACL clock so
revocation propagates post-hoc across federated peers**; auditable context packs where
`context_pack_report()` explains every include/exclude decision. **[V]**

**[Vault-Agent-Memory](https://github.com/zycaskevin/Vault-Agent-Memory)** — 43 stars,
Apache-2.0. **[V]** Explicit lifecycle `propose → review → promote → search → bounded read →
rollback → audit`; scopes private/project/shared/public; "anon/scoped agents can submit
candidates but cannot write active memory" — the clearest **candidate-first write gate**
found. Names the exact failure mode CRED worries about: "private observations leak into
shared project memory." Positions as a backend-agnostic *governance contract*, not a store.

**[Dense-Mem](https://github.com/markhuangai/dense-mem)** — 34 stars, Apache-2.0, Go +
Neo4j + Postgres, MCP Streamable HTTP. **[V]** Teams, named profiles, API keys, audit
metadata, control portal. Differentiator is **evidence provenance + typed claims + conflict
detection** — it stores source fragments *before* claims and "never silently rewrites facts."

### The trap: "shared" that means cross-*tool*, not cross-*people*

**This is the dominant pattern by a wide margin.** Most projects using team language share
memory across *your own* tools (Claude Code + Cursor + Codex on one laptop), not across
people. Of ~80 projects surfaced, **fewer than 15 have a real multi-user permission model.**
**[V]** Examples that market as team memory but have **no ACL/permission/audit terms in the
README at all**:

| Project | Stars | Why it is NOT org memory |
|---|---|---|
| [im4codes](https://github.com/im4codes/imcodes) | 1,028 | Has audit/team/review vocabulary but self-describes as an "actively developed **personal** open-source project, no warranties, no SLA" |
| [cavemem](https://github.com/JuliusBrussee/cavemem) | 631 | "Local by default", cross-agent per-user |
| [memorix](https://github.com/AVIDS2/memorix) | 561 | Cross-agent via MCP — cross-tool, per-user |
| [guild](https://github.com/mathomhaus/guild) | 318 | "State lives strictly on local host; nothing leaves your machine." Multi-*agent*, single-*human* |
| [statewave](https://github.com/smaramwbc/statewave) | 292 | Strong provenance-tagged context bundles, weak multi-user |
| [egregore](https://github.com/egregore-labs/egregore) | 268 | "Agentic OS for organizations" — but a Claude Code wrapper, almost no permission machinery |
| [stash](https://github.com/Fergana-Labs/stash) | 242 | Markets as "shared memory for your team's coding agents"; **README has zero ACL/permission/audit terms** |
| [plur](https://github.com/plur-ai/plur) | 226 | "Your agents share the same memory" — local-first engrams, one machine, zero ACL keywords |
| [ultron](https://github.com/modelscope/ultron) | 164 | Alibaba/ModelScope. Tiered collective memories + skill crystallization; research-flavored, no ACL |
| [eion](https://github.com/eiondb/eion) | 159 | AGPL-3.0; **dead since 2025-07-02**. Was closest to true shared multi-agent memory in 2025 |
| [sibyl](https://github.com/hyperb1iss/sibyl) | 45 | "One CLI. One graph. Every AI tool you use, sharing memory" — self-hostable but per-user |

Small but on-thesis (<10 stars, all 2026): [hippocampus](https://github.com/z10-labs/hippocampus)
(decision memory — "what did we already decide and why", markdown ADRs in-repo, PR-reviewable
— an excellent primitive), [agent-in-sync](https://github.com/agentinsync/agent-in-sync)
("the 10th agent to hit the same bug pays 3K tokens not 50K"),
[engram-oss](https://github.com/ycnslh/engram-oss), [Akasha](https://github.com/chaterm/Akasha),
[lore-oss](https://github.com/re-cinq/lore-oss),
[CentralMemoryHub](https://github.com/JustinAngelson/CentralMemoryHub).

### Verdict on Task 4

**The "open-source Memco" exists (MemClaw), is credible on features, and is unproven on
evidence and adoption.** CRED cannot claim the category is empty. It can still claim the
category is unproven — no one, MemClaw and Memco included, has published an ablation
showing shared experiential memory contributes independently of documentation retrieval.

---

## 2. ByteRover — deep dive

**[V]** unless noted.

| Attribute | Value |
|---|---|
| Product site | https://www.byterover.dev/ · docs https://docs.byterover.dev/ · app https://app.byterover.dev |
| OSS repo | https://github.com/campfirein/byterover-cli (CLI `brv`, formerly **Cipher**) |
| **License** | **Elastic License 2.0** — source-available, NOT OSI open source. Forbids providing the software as a managed service. |
| Stars / forks / contributors | 4,925 / 453 / 17 |
| Created / last push | 2025-06-19 / 2026-06-25 |
| Paper | [arXiv:2604.01599](https://arxiv.org/abs/2604.01599), *"ByteRover: Agent-Native Memory Through LLM-Curated Hierarchical Context"*, Nguyen et al., 2026-04-02 |
| Funding | **NULL** — not disclosed in any source fetched |

**Memory unit / data model — the distinctive part.** A **Context Tree**: markdown files in
`.brv/context-tree/`, hierarchically organised **Domain → Topic → Subtopic → Entry**.
Human-readable, git-friendly. Entries carry explicit relationships, source tracking, and an
"Adaptive Knowledge Lifecycle" with importance scoring and recency decay. The paper's thesis
is that the *same LLM that reasons also curates and retrieves*, "inverting the memory
pipeline" to eliminate semantic drift between agent intent and stored knowledge. Retrieval
is a 5-tier progressive strategy at sub-100ms. Explicitly **"zero external infrastructure,
no vector database, no graph database, no embedding service."** Claims SOTA on LoCoMo and
competitive on LongMemEval.

**Integration surface.** MCP server via `brv mcp`; daemon-first architecture (background
daemon, CLI and agents connect directly). Plugins exist for Claude Code
(`campfirein/brv-claude-plugin`, auto-memory via hooks), OpenCode, OpenClaw, and Pi.

**Org/team story — commercial, not OSS.** Everything works offline/local by default.
**ByteRover Cloud** adds: team context sync via "Git-Semantic version control"
(`brv vc push/pull/fetch`), **shared spaces** across projects and teams, multi-machine sync,
and **team management — members, spaces, and permissions via the web app**. Search results
carry an **`Origin`** field distinguishing `local` vs `shared`. **[V]**

**Pricing.** Free: 100 cloud-synced context files with team collaboration. Pro: $15/mo
(annual), unlimited context files. Enterprise: custom — **SOC 2, SSO/SAML, RBAC, on-premise
deployment**. **[V]**

**ACL / provenance / trust.**
- ACL: **team permissions exist but only in the paid cloud web app**; no ACL model is
  documented in the OSS CLI. Enterprise adds RBAC. Detail level: **NULL**.
- Provenance: per-entry "source tracking" **[V]**, plus `Origin` local/shared. No audit log
  documented — **NULL**.
- Trust: importance scoring + recency decay + LLM curation. **No contradiction detection,
  no trust tiers, no verification against ground truth documented — NULL.**

**Traction.** Strongest of any dedicated player found: 4,925 stars, 453 forks, 17
contributors, an npm package with published weekly download counts, an arXiv paper, and
multiple third-party ecosystem plugins built by outsiders (`ian-pascoe/opencode-byterover`,
`RyanNg1403/byterover-skills`, `thelearningest/…-Collaboration`). HN presence is
persistent but low-scoring (2–8 points across ~8 posts since 2025-04). **[V]**

**Assessment.** ByteRover was correctly characterised by Memco as a memory layer for coding
agents, but Memco filed it under *user-managed single-user memory* and that remains broadly
right: the OSS core is per-developer and local-first; sharing is a cloud upsell. Its real
threat to CRED is **distribution and the Context Tree model** — markdown-in-git with no
vector store is cheap, inspectable, and diffable, and it directly undercuts the assumption
that org memory needs a database. Its weakness is that ELv2 blocks a hosted ecosystem and
the governance layer (ACL, provenance, trust) is thin-to-absent relative to MemClaw.

---

## 2b. Commercial / funded — the "company brain" cohort

### SageOx — the best-funded direct competitor
- https://sageox.ai · **Raised $15M seed** (**[V]** — stated on sageox.ai)
- "The hivemind for human–agent teams. **SageOx adds shared, structured team memory that
  agents consult before they act**" **[V]**
- Closed source. Records discussions, agent sessions, and assistant chats as **durable team
  artifacts**, extracting decisions. Anti-drift framing.
- Show HN 2026-02-19 scored **4 points** — very low HN traction despite the raise.

### Hyper (YC P26) — the most architecturally sophisticated closed competitor
Launch HN 2026-06-03, **79 points** (the highest-scoring org-memory post found). **[V]**
- https://heyhyper.ai · founders Shalin Shah, Kanyes Thaker
- **Data model:** hybrid **episodes** (raw source items) + **facts** (subject-predicate-object
  records). Facts form a knowledge graph with **typed edges including "X is in tension with
  Y" and "A supersedes B."** Retrieval is semantic + Postgres full-text via reciprocal rank
  fusion. **[V]**
- **"Every fact carries provenance back to its source and access-control tags for who is
  allowed to see it."** RBAC is load-bearing: *"two people on the same team can ask the same
  question and get different answers."* **[V]**
- **Not open source, managed AWS only** — founders explicitly prioritise vendor-managed
  security over self-hosting. **[V]**
- Prefers **lifecycle hooks injected into Claude Code / Cursor over MCP tool calls**, for
  more reliable context injection and extraction without relying on the agent to recognise a
  tool call. **[V]** — a notable architectural dissent from MCP-as-sole-surface.
- **Critical HN comments worth noting:** one commenter called this "the ultimate moat" and
  argued for open-source alternatives to avoid lock-in; a developer raised the intent problem
  (architecture notes from exploratory work "absolutely should not be referenced in every day
  work" yet may surface anyway); a regulated-industry user criticised vague "military grade
  encryption" claims and noted Hyper "is not currently a good fit for highly-regulated
  industries"; another reported Notion/Slack integrations failing to connect. **[V]**

### YC portfolio — the category emerges F25 onward **[V]** (via YC OSS API)

| Company | Batch | URL | Positioning |
|---|---|---|---|
| **Nessie** | F25 | https://nessielabs.com/ | "A shared context layer for you, your team, and your agents" — dead-on category match; site is a near-empty stub |
| **Memory Store** | Spring 2026 | https://memory.store | "Company Brain — one shared memory for your teammates, and agents." Syncs Gmail/Slack/Granola/Claude/Codex; models "what happened, who was involved, what changed" |
| **Hyper** | Spring 2026 | https://heyhyper.ai | see above |
| **Shepherd** | S26 | https://askshepherd.ai | "Ingests every tool into one unified memory your agents can act on" — memory→action, human approves |
| **Osseus** | S26 | https://osseus.ai/ | "One **permissioned** research brain" for R&D — permissions are the pitch |
| **Trace** | S25 | https://www.trace.so/ | AI orchestration layer, org-model aware. Adjacent |
| **WUPHF / Nex.ai** | S26 | https://github.com/nex-crm/wuphf (1,214★) | Agents collaborating over shared context; "prevent context drift through gossip" |

**YC W25 and S25 contain zero true org-memory companies** — confirming a clean 2026 category
emergence. **[V]** The cohort competes almost entirely on *ingestion breadth*
(Slack/Gmail/Notion connectors), not on provenance, verification, or measurable execution.

### Smaller commercial
[ContextVault](https://www.contextvault.dev/) — "scoped per user, per agent, **per tenant**"
**[V]**, Show HN 2026-07-13, 12 pts · [Kumbukum](https://kumbukum.com/) — OSS team shared
library over MCP · **[Kage](https://kage-core.com/)** — built on **Google's Open Knowledge
Format (OKF)**; verifies every memory deterministically against real code, claims a "100/100
trust benchmark"; the closest thing to a real *verification* mechanism in the commercial set ·
[xysq.ai](https://xysq.ai/) · [Wolbarg](https://wolbarg.com) — TS SDK, SQLite/Postgres ·
[Aide-Memory](https://www.aide-memory.dev) — path-scoped team memory, local-first.

---

## 2c. Protocols and standards — watch closely

| Protocol | URL | Status |
|---|---|---|
| **Universal Memory Protocol (UMP)** | https://universalmemoryprotocol.io/ | "What MCP did for tools, UMP does for memory" — transport-neutral, **portable, signed, bi-temporal record**. HN 2026-06-06, 41 pts. Signed + bi-temporal is directly relevant to provenance/trust |
| **Akashik Protocol** | https://akashikprotocol.com | Open protocol for shared memory, coordination, **and conflict resolution** between agents. MCP + A2A compatible, Levels 0–3, SDK v0.2.0 |
| **Google OKF (Open Knowledge Format)** | via https://kage-core.com/ | Google-shipped standard: agent memory as plain Markdown in-repo, no lock-in. **Deliberately omits verification** — that omission is the product opportunity. **[V as Kage's claim; OKF NOT verified directly at Google — flagged for follow-up]** |

**Standards risk:** OKF could commoditise the storage format. Building *on* OKF (as Kage
does) rather than against it is likely the correct posture. **[I]**

---

## 2d. Research — arXiv 2025–2026 **[V]** (via arXiv API)

**Directly on-thesis:**
- [Organizational Memory for Agentic Business Process Execution](https://arxiv.org/abs/2607.03228) — 2026-07-03 — newest and most on-point
- **[GateMem: Benchmarking Memory Governance in Multi-Principal Shared-Memory Agents](https://arxiv.org/abs/2606.18829)** — 2026-06-17 — "multi-principal" is exactly the org distinction. **A governance benchmark now exists and no product has reported against it. This is the cheapest available credibility win.**
- [Governed Shared Memory for Multi-Agent LLM Systems](https://arxiv.org/abs/2606.24535) — 2026-06-23
- [Collaborative Memory: Multi-User Memory Sharing in LLM Agents with Dynamic Access Control](https://arxiv.org/abs/2505.18279) — 2025-05-23 — the foundational ACL paper
- [PiSAs: Benchmarking Contextual Integrity in Multi-User Agentic Systems](https://arxiv.org/abs/2607.05318) — 2026-07-06
- [MOSS: Memory-Orchestrated Semantic System — An Auditable Agentic Memory Architecture](https://arxiv.org/abs/2607.04391) — 2026-07-05
- [ByteRover: Agent-Native Memory Through LLM-Curated Hierarchical Context](https://arxiv.org/abs/2604.01599) — 2026-04-02
- [Memco/Spark: Smarter Together](https://arxiv.org/abs/2511.08301) — 2025-11-11

**Security — the case for ACLs:**
- [Topology Matters: Measuring Memory Leakage in Multi-Agent LLMs](https://arxiv.org/abs/2512.04668)
- [Trojan Hippo: Weaponizing Agent Memory for Data Exfiltration](https://arxiv.org/abs/2605.01970)
- [AgentPoison](https://github.com/AI-secure/AgentPoison) (NeurIPS 2024, 230★)

**Architecture:** [G-Memory](https://arxiv.org/abs/2506.07398) ·
[LEGOMem](https://arxiv.org/abs/2510.04851) ·
[Multi-Agent Memory from a Computer Architecture Perspective](https://arxiv.org/abs/2603.10062) ·
survey [Memory in the Age of AI Agents](https://arxiv.org/abs/2512.13564) ·
[Emergent Collective Memory in Decentralized Multi-Agent AI](https://arxiv.org/abs/2512.10166)

---

## 2e. Trust / verification — the thinnest column in the market

Almost nobody does cryptographic or deterministic verification. The complete set of
exceptions found: **[V]**

- **[emem](https://github.com/Vortx-AI/emem)** (45★, Apache-2.0) — "verifiable memory
  protocol… every observation becomes a shared, verifiable **Memory Token**"; Zenodo
  whitepaper DOI. Domain is physical-world observation, but the *signed-fact-you-can-cite*
  primitive transfers directly.
- **[Kage](https://github.com/kage-core/Kage)** (28★, GPL-3.0) — deterministic verification
  of every memory against actual code; plain files in-repo, shared via git, no DB, no account.
- **UMP** — signed bi-temporal records.
- **[openclaw-mem](https://github.com/phenomenoner/openclaw-mem)** (27★) — "memory layer you
  can audit: citations, trust policies, trace receipts, rollback."
- **[brigade](https://github.com/escoffier-labs/brigade)** (61★) — "Your agents run loops.
  Brigade keeps the receipts… prove with file receipts."
- **[brain0](https://github.com/Brain0-ai/brain0)** (464★, Apache-2.0) — passive decision
  graph linking every commit to the agent prompts behind it; drift detection, DLP audit of
  what agents read, **signed provenance attestations**, offline by default.
- **[SoupNet](https://github.com/AndyForest/SoupNet)** — shared **append-only** memory of
  human judgment calls.

**This is the emptiest column and the one CRED's stated "Traceability" maps onto directly.**

---

## 3. MCP memory servers

### Official — `modelcontextprotocol/servers` **[V]**

Repo: https://github.com/modelcontextprotocol/servers — **88,646 stars**, last push
2026-07-10. Seven surviving reference servers: `everything`, `fetch`, `filesystem`, `git`,
**`memory`**, `sequentialthinking`, `time`. Twelve were archived (GitHub, GitLab, Postgres,
Slack, Redis, Sentry, Puppeteer, Brave Search, Google Drive/Maps, EverArt, AWS KB).
`memory` survived.

**The `memory` knowledge-graph server** (MIT):
- **Data model:** Entities (unique `name`, `entityType`, observations) · Relations
  (directed, active voice) · Observations (atomic fact strings). 9 tools.
- **Storage:** a **single flat JSONL file**, `MEMORY_FILE_PATH` default `memory.jsonl`.
  **Every mutation rewrites the entire file.** Verified from `index.ts`.
- **Multi-user / ACL / auth: NONE.** No identity, no tenancy, no permission filtering.
  No file locking or transactions → **concurrent writes lose data.** Stdio transport only,
  so the MCP OAuth profile does not even apply.
- **Toy or real? Toy.** The repo README states these are "reference implementations…
  intended to demonstrate MCP features and SDK usage," not production-ready.

> **Implication for CRED:** the default that every developer meets first is a single-user
> JSONL file with no concurrency control. That is the baseline being displaced, and it sets
> a very low bar — which is precisely why 30+ projects rushed the gap in 2026.

### Community servers **[V]** for stars/dates; capability claims are vendor-stated

| Server | Stars | Push | License | Storage | Team-shared? |
|---|---|---|---|---|---|
| [claude-mem](https://github.com/thedotmack/claude-mem) | 87,902 | 2026-07-19 | Apache-2.0 | SQLite + Chroma, hook-based auto-capture | null |
| [mem0](https://github.com/mem0ai/mem0) | 61,255 | 2026-07-20 | Apache-2.0 | pluggable vector+graph | SaaS multi-tenant [I] |
| [MemPalace](https://github.com/MemPalace/mempalace) | 57,489 | 2026-07-17 | MIT | null | null |
| [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp) | 33,029 | 2026-07-19 | MIT | SQLite+LZ4 | Partial — committable graph snapshot |
| [cognee](https://github.com/topoteretes/cognee) | 28,536 | 2026-07-19 | Apache-2.0 | knowledge graph | null |
| [engram](https://github.com/Gentleman-Programming/engram) | 5,579 | 2026-07-08 | MIT | SQLite+FTS5, Go | null |
| [Zep](https://github.com/getzep/zep) | 4,766 | 2026-07-17 | Apache-2.0 | temporal KG | user-scoped [I] |
| [basic-memory](https://github.com/basicmachines-co/basic-memory) | 3,462 | 2026-07-20 | AGPL-3.0 | MD+SQLite / Neon+S3 | **Yes — "Teams" workspaces**, no granular ACL |
| [caura-memclaw](https://github.com/caura-ai/caura-memclaw) | 341 | 2026-07-19 | Apache-2.0 | Postgres+pgvector | **Yes — full multi-tenant + trust tiers** |
| [omem](https://github.com/ourmem/omem) | 200 | 2026-05-24 | Apache-2.0/NOASSERTION | REST, tenant-isolated | **Yes — Spaces** |
| [memory-bank-mcp](https://github.com/alioshr/memory-bank-mcp) | 915 | **2025-08-20** | MIT | remote files | remote-capable; ~11mo stale |
| [memento-mcp](https://github.com/gannonh/memento-mcp) | 425 | **2025-10-27** | MIT | Neo4j | no; ~9mo stale |

**Answer to "does a team-shared permissioned MCP memory server exist?"** — **Yes,
definitively.** MemClaw is the reference example (multi-tenant, scoped, trust-tiered,
audited); omem and basic-memory are weaker variants. The per-developer-local-file pattern
the official server exemplifies is no longer state of the art. **[V]**

**Data-quality note:** several star counts are very high for the niche. I sanity-checked
star:fork ratios (claude-mem 87.9k:7.6k, mem0 61k:7.1k, servers 88.6k:11.3k) and they are
internally consistent, so the figures appear genuine rather than API artifacts. Reported as
API-returned. **[V, with caveat]**

### MCP protocol state, 2026 **[V]**

- **Current spec version: `2025-11-25`.** Formal feature lifecycle with a deprecation
  policy (features live ≥12 months, or ≥90 days expedited).
- **Auth:** MCP server = OAuth 2.1 **resource server**. MUST implement RFC 9728 Protected
  Resource Metadata; clients MUST implement RFC 8707 Resource Indicators; servers MUST
  validate token audience and MUST NOT pass tokens upstream. PKCE `S256` mandatory.
  **Client ID Metadata Documents** are now the preferred registration path; Dynamic Client
  Registration (RFC 7591) has been **demoted to backwards-compatibility**.
  **Step-up authorization** (`403` + `WWW-Authenticate: insufficient_scope`) lets a server
  demand more scopes mid-session — directly usable for read-only-by-default memory that
  escalates for org-scope writes.
  Note: **stdio servers SHOULD NOT use OAuth** (credentials come from env) — which is
  exactly why the official memory server has no auth.
- **Registry:** live at `registry.modelcontextprotocol.io`, GitHub OAuth for publishers,
  cursor-paginated `/v0/servers` against schema `2025-12-11`, reverse-DNS namespacing,
  `remotes` declaring `streamable-http`. **Total server count: NULL. GA vs preview: NULL**
  (the `v0` prefix suggests pre-1.0 — **[I]**).

**Practicality verdict [I]:** the protocol is not the constraint. Streamable HTTP + a
mandatory OAuth 2.1 resource-server profile with audience-bound tokens and step-up scopes
supplies exactly the primitives a permissioned org memory server needs. The friction is that
most MCP clients still default to launching stdio subprocesses.

---

## 4. First-party org memory — Anthropic / OpenAI / GitHub / Google / Cursor

**The headline: two vendors now ship server-side, admin-editable, org-wide instruction
delivery. This is no longer greenfield.** But no vendor propagates *learned* knowledge
between users.

### Anthropic / Claude Code **[V]** — [docs](https://code.claude.com/docs/en/memory)

`docs.claude.com/en/docs/claude-code/*` now 301-redirects to `code.claude.com/docs/en/*`.

**CLAUDE.md hierarchy — four levels, concatenated (not overridden):**

| Scope | Path | Shared with |
|---|---|---|
| Managed policy | macOS `/Library/Application Support/ClaudeCode/CLAUDE.md`; Linux `/etc/claude-code/CLAUDE.md`; Windows `C:\Program Files\ClaudeCode\CLAUDE.md` | whole org |
| User | `~/.claude/CLAUDE.md` | you, all projects |
| Project | `./CLAUDE.md` or `./.claude/CLAUDE.md` | team, via git |
| Local | `./CLAUDE.local.md` (gitignored) | you, this project |

Claude Code walks up the directory tree from cwd; content is ordered filesystem-root → cwd
so the nearest file is read last. Subdirectory CLAUDE.md files load **on demand** when
Claude reads files there. Managed-policy CLAUDE.md **cannot** be excluded by
`claudeMdExcludes`. Imports use `@path` (relative to importing file or absolute), **max
depth 4**, skipping code spans/fences; first external import triggers approval, declining
disables permanently. **Size target <200 lines — guidance, not enforced**; files load in
full regardless. `.claude/rules/*.md` supports YAML `paths:` frontmatter for glob-scoped
conditional loading and is symlink-friendly (the cheapest existing cross-repo rule-sharing
hack). **Claude Code does NOT read AGENTS.md** — workaround is `@AGENTS.md` or a symlink.

**Auto memory — the closest thing to auto-capture anyone ships.** Stored at
`~/.claude/projects/<project>/memory/`, keyed to the git repo (all worktrees share one).
`MEMORY.md` index + topic files; **only the index loads at session start — first 200 lines
or 25KB, whichever comes first**; topic files load on demand. On by default
(`autoMemoryEnabled: false` / `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`). **Critical, quoted:
"Auto memory is machine-local… Files are not shared across machines or cloud
environments."** Not inherited by subagents except forks.

**Server-managed settings — the finding that most changes the picture.** Owners configure
JSON at `claude.ai/admin-settings/claude-code`; clients fetch at startup and **poll hourly**;
Team/Enterprise only. The **`claudeMd` key injects managed CLAUDE.md content directly inside
managed settings** — no file deployment. So org-wide instruction text *is* genuinely
server-synced and centrally editable in 2026. Constraints are severe and explicit:
**"Settings apply uniformly to all users in the organization. Per-group configurations are
not yet supported."** No ACL, no team scoping, no per-repo targeting. Docs state it is
"a client-side control, not a security boundary," bypassed by Bedrock/Vertex/custom
`ANTHROPIC_BASE_URL`.
[server-managed-settings](https://code.claude.com/docs/en/server-managed-settings)

**Claude Code on the web:** carries repo CLAUDE.md, repo settings hooks, repo-declared
plugins, and org server-managed settings. Does **not** carry `~/.claude/CLAUDE.md`,
user-scoped plugins, or auto memory (machine-local).

**Claude.ai memory (Team/Enterprise, GA 2025-09-11; Pro/Max 2025-10-23):** **per-user,
per-project, explicitly not shared between users.** Admins can disable org-wide.
[claude.com/blog/memory](https://claude.com/blog/memory)

**`/memory` command** lists CLAUDE.md/CLAUDE.local.md across scopes, toggles auto memory,
opens the auto-memory folder. `/context` shows what actually loaded. `InstructionsLoaded`
hook logs which instruction files loaded and why.

### OpenAI / Codex **[V]**

`developers.openai.com/codex/*` 308-redirects to `learn.chatgpt.com/docs/*`. Codex reads
`~/.codex/AGENTS.md`, repo-root `.codex/AGENTS.md`, and nested files, merged with closer
files winning; untrusted projects skip project-scoped layers. `project_doc_max_bytes` caps
instruction size (**default: NULL**). Enterprise managed configuration is two-tier:
`requirements.toml` (non-overridable, via system files / cloud bundles / macOS MDM) and
`managed_config.toml` (changeable defaults) — covering approval/sandbox policy, permission
profiles, data residency, MCP allowlists, marketplace sources, hooks, network access.
**NULL: whether admins can push an org-wide AGENTS.md** — the managed-config docs cover
policy enforcement only. **This is a real gap versus Anthropic's `claudeMd`.**
**ChatGPT Enterprise "company knowledge": UNVERIFIED** — help.openai.com returned 403.

### GitHub Copilot **[V]**

Files: `.github/copilot-instructions.md`; `.github/instructions/NAME.instructions.md` with
`applyTo` globs and an `excludeAgent` keyword; **`AGENTS.md` anywhere (nearest wins); plus
`CLAUDE.md` and `GEMINI.md` at repo root** — the broadest filename compatibility of any
vendor. **Org instructions** (Copilot Business/Enterprise, Settings > Copilot > Custom
instructions) apply to Chat, code review, and cloud agent **on GitHub.com only, not the
IDE**. Precedence, quoted: *"Personal instructions take the highest priority. Repository
instructions come next, and then organization instructions are prioritized last."* — the
**inverse** of Anthropic, so org instructions cannot enforce anything. Size limits: NULL.

**Copilot Memory (public preview)** — the agent can "store useful details it has worked out
for itself about a repository," for Pro/Pro+/Max. Repository-scoped and tied to *individual
consumer-tier plans*, which strongly suggests per-user — but **sharing semantics, storage,
and edit/delete UX are UNVERIFIED** (dedicated doc page not found). Closest competitive move
to auto-capture outside Anthropic; worth a targeted follow-up. Hard constraint, quoted:
*"By default, Copilot can only access context in the repository specified when you start a
task."*

### Google **[V]**

**Gemini CLI:** three tiers — global `~/.gemini/GEMINI.md`, workspace + parent dirs, and a
**just-in-time tier** scanning ancestors up to a trusted root when tools touch files. All
discovered files are **concatenated and sent with every prompt**. `context.fileName` accepts
an array (e.g. `["AGENTS.md","GEMINI.md"]`) so AGENTS.md support is opt-in config. Imports
`@./path.md`. `/memory show` and `/memory reload`. Enterprise system settings at
`/etc/gemini-cli/settings.json` (+ Windows/macOS equivalents), four layers with **system
overrides highest**; enforcement is a **PATH wrapper script** — notably weaker than MDM.
No documented way to push a shared GEMINI.md org-wide.
**Gemini Code Assist: NULL** (docs 404). **Antigravity: NULL** (JS shell). **Jules: NULL.**

### Cursor **[V]**

`docs.cursor.com` 308-redirects to `cursor.com/docs`. Project rules live in
`.cursor/rules/*.mdc` — **a plain `.md` there is ignored**, the `.mdc` extension and
frontmatter are required. **Team Rules: organization-wide, managed via the Cursor dashboard,
Team/Enterprise; admins can enforce rules or let users toggle them off** — a genuine
server-side org instruction layer, with an enforce/optional toggle Anthropic lacks.
Precedence **Team → Project → User** (opposite of Copilot: the org can override the
individual). AGENTS.md supported. Application modes: Always Apply / Apply Intelligently /
Apply to Specific Files (globs) / Manual.
**Cursor Memories: UNVERIFIED** — two doc URLs 404'd; do not assume org-shared.

### AGENTS.md standard **[V]**

**60,000+ open-source projects** on GitHub. Now **stewarded by the Agentic AI Foundation
under the Linux Foundation** — no longer under OpenAI's sole control, which materially
raises its durability. Supporters include Codex, Jules, Devin, Windsurf, VS Code, Cursor,
Zed, Junie, Aider, goose, Factory, opencode, Warp, Copilot, Amp, RooCode, Gemini CLI,
Augment. **The one holdout is Claude Code.**

### What the free tier does NOT do — the gap table

| Capability | Built-in status | Gap |
|---|---|---|
| Auto-capture | Partial, 1.5 vendors | Only Claude Code auto memory + Copilot Memory (preview) write without human authoring. Everything else is manually authored *and manually maintained*. Anthropic frames the trigger as human: "Claude makes the same mistake a second time." |
| Cross-machine sync of learned knowledge | **No** | Quoted: auto memory "is machine-local… not shared across machines or cloud environments." |
| **Cross-user sharing of learned knowledge** | **No — universal gap** | Claude.ai memory is per-user/per-project; auto memory is machine-local; Copilot Memory is plan-gated per-repo. **Authored files propagate via git; learnings never do.** |
| Cross-repo | **No** | Copilot: "can only access context in the repository specified." Claude auto memory is keyed to the repo. Only symlinked rules, private plugin marketplaces, and org instruction boxes cross repos — all manual, all coarse. |
| ACL / team scoping | **No** | Anthropic, quoted: *"Per-group configurations are not yet supported."* Copilot org instructions are one text box, ranked lowest. Nothing supports "visible to the payments team only." |
| Provenance | **No** | No vendor tracks who authored a rule, when, from what incident, or whether it's still true. Anthropic's guidance is manual hygiene: "Review your CLAUDE.md files… periodically." |
| Decay / staleness | **No** | Zero automatic expiry. Claude Code's only staleness signal is *byte pressure* at 200 lines/25KB — a budget heuristic, not a truth model. |
| Conflict resolution | **No — explicitly disclaimed** | Anthropic, quoted: *"If two rules contradict each other, Claude may pick one arbitrarily."* Mitigation offered is `claudeMdExcludes` globs — exclude other teams' files rather than reconcile. |
| Retrieval beyond static injection | **Mostly no** | Dominant model is concatenate-at-launch; Gemini CLI sends context "with every prompt." Partial exceptions are **glob-triggered, not semantic**. **No vendor ships embedding/semantic retrieval over an org corpus.** |
| Telemetry | **No** | Nothing measures whether a rule was followed or helped. `InstructionsLoaded` logs *that* files loaded, never their effect. |
| Enforcement | **No, by design** | Anthropic: *"Claude treats them as context, not enforced configuration"*, "no guarantee of strict compliance." |
| Context economics | **Actively hostile** | Every authored instruction costs tokens every turn; docs warn longer files "reduce adherence," and `@path` imports "don't reduce context, since imported files load at launch." **A naive org memory layer makes the agent worse — any real product must be retrieval-gated, not injection-based.** |

---

## 5. Adjacent threat — agent skills registries

### The convergence is real and already happened at Tessl **[V]**

Tessl's post [*"Your agents keep making the same mistakes. Nobody has time to fix it"*](https://tessl.io/blog/your-agents-keep-making-the-same-mistakes-nobody-has-time-to-fix-it)
(2026-06-30) describes Tessl Agent scanning **commit history and session logs**
continuously: *"When it spots a recurring error pattern, it creates a skill to address it
and opens a PR."* That is learned, auto-captured organizational memory — differing from an
org-memory product mainly in that the artifact is a versioned SKILL.md merged via PR rather
than a row in a memory store. **A registry company built the accumulation loop and shipped
it into the distribution channel it already owned.**

Tessl: positioning "Skills are the new code"; registry of **3,000+ skills** plus private
workspaces; every skill scored on Quality, Impact (uplift multiplier), and Security (Snyk
scanning); founder **Guy Podjarny** (Snyk, $8B); investors Index, GV, boldstart, Accel;
**$125M total [I — aggregated, not confirmed against a primary announcement; valuation
figures disagree across sources]**. Not open source. Notably, Tessl launched in 2024 as
*spec-centric*; the 2026 site is entirely *skills-centric* — a pivot. **[V on both endpoints]**

### SKILL.md is now a cross-vendor open standard **[V]**

Originally Anthropic's, "released as an open standard," now governed at
[agentskills.io](https://agentskills.io/home) / [github.com/agentskills/agentskills], with
~45 adopters including **Cursor, GitHub Copilot, VS Code, OpenAI Codex, Gemini CLI,
OpenHands, goose, Amp, Factory, Kiro, Databricks, Snowflake, Pulumi, Tabnine, Letta**.
Format: `SKILL.md` with YAML frontmatter (`name` ≤64 chars, `description` ≤1024) plus body,
optional `scripts/` `references/` `assets/`; three-stage progressive disclosure (metadata
always in system prompt ~100 tokens/skill; body on trigger <5k; resources on read).

### Anthropic's own position **[V]**

Sharing scope differs sharply by surface: **claude.ai skills are individual-user only** and
"not shared organization-wide and cannot be centrally managed by admins"; the API is
workspace-wide with **max 8 skills per request**; Claude Code uses the filesystem plus
plugins. Skills **do not sync across surfaces**. Anthropic's enterprise guidance tells orgs
to "maintain an internal registry" and "implement your own synchronization process," and
notes "usage analytics are not currently available through the Skills API."
**Anthropic documents the org-knowledge gap and tells customers to fill it themselves.**

Its stated direction, quoted from [the engineering post](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills):
*"Looking further ahead, we hope to enable agents to create, edit, and evaluate Skills on
their own, letting them codify their own patterns of behavior into reusable capabilities."*
**The platform owner has declared the destination and not yet arrived** — that is both the
window and the platform risk.

**Claude Code plugin marketplaces are the real org distribution channel today** and are
mature: official `claude-plugins-official` enabled by default; community marketplace with
SHA-pinned plugins; any git repo or npm package as a private marketplace; enterprise
controls via `extraKnownMarketplaces`, `strictKnownMarketplaces` (allowlist by URL or
`hostPattern`, non-overridable), `disableSideloadFlags`, container seeding, stable/latest
channels, admin-forced `autoUpdate`. **[V]**

### Other registries **[V]** (stars via API 2026-07-20)

| Registry | Stars | Note |
|---|---|---|
| [ComposioHQ/awesome-claude-skills](https://github.com/ComposioHQ/awesome-claude-skills) | 68.1k | stale since 2026-05-22 |
| [sickn33/agentic-awesome-skills](https://github.com/sickn33/agentic-awesome-skills) | 43.6k | local agent-first control plane |
| [VoltAgent/awesome-agent-skills](https://github.com/VoltAgent/awesome-agent-skills) | 28.5k | 1,497+ skills, org-indexed |
| [tech-leads-club/agent-skills](https://github.com/tech-leads-club/agent-skills) | 4.9k | security-first; claims **"over 13% of marketplace skills contain critical vulnerabilities"**; content hashing, immutable lockfiles |
| **[iflytek/skillhub](https://github.com/iflytek/skillhub)** | 4.8k | **Apache-2.0 self-hosted enterprise skill registry: team namespaces, RBAC (Owner/Admin/Member), semver, admin review workflows, audit logging, Docker/K8s, S3/MinIO.** Commoditizes the governance story. |
| [Kamalnrf/claude-plugins](https://github.com/Kamalnrf/claude-plugins) | 528 | claims 11,989 plugins / 63,065 skills |

Hosted: [officialskills.sh](https://officialskills.sh/) (651 official skills, 55 vendor
teams); [skills.sh](https://skills.sh) leaderboard (claims ~946,763 skills all-time —
**self-reported and implausible as distinct quality skills; treat as inflated [I]**);
[index.tego.security/skills](https://index.tego.security/skills/) (security database).
**cursor.directory: NULL — HTTP 429 on both attempts.**

Standards efforts: [Mintlify "Skill.md: An open standard"](https://www.mintlify.com/blog/skill-md);
**[Universal Memory Protocol](https://universalmemoryprotocol.io/) — "a shared format for
agent memory"; the memory side is attempting its own standardization separately. [V]**

**Evidence skills work:** [SkillsBench](https://arxiv.org/abs/2602.12670) — 87 tasks, 8
domains, 18 model-harness configs; **skills raise pass rates 33.9% → 50.5% (+16.6pp)**;
"smaller models with Skills can match larger models without them"; focused skills
(≤3 modules) beat exhaustive bundles. **[V]**

### Are skills just hand-written memories? — no consensus; the record leans no

- **Letta** — skills are "reference guides your agent consults"; memory is identity. But
  "agents can write skills into their own memory systems, so that skills persist with the
  agent as part of its identity" — **convergence from the memory side.** **[V]**
- **Addy Osmani** — *"A skill is not reference documentation."* Skills are executable
  workflows; the session log is "the durable memory," a separate layer. **[V]**
- **Simon Willison** on Jesse Vincent's Superpowers — skills plus "self-updating memory
  notes" described as **two complementary components**. **[V]**
- **Anson Biggs** comes closest to the equivalence but insists authoring must be human, and
  quotes *"Self-generated Skills provide no benefit on average, showing that models cannot
  reliably author the procedural knowledge they benefit from consuming."* **[I — attribution
  to SkillsBench UNVERIFIED; not in the abstract. If it holds it is the strongest available
  counter to Tessl's auto-generation thesis. Verify before relying on it.]**

### Where they genuinely differ

| | Skills registries | Org memory |
|---|---|---|
| Origin | Authored, reviewed, PR'd | Observed, auto-captured |
| Unit | Bounded procedure | Unbounded episodic/factual accretion |
| Lifecycle | Semver, eval gates, deprecation | Decay, reinforcement, contradiction resolution |
| Direction | **Distribution** (one author → many agents) | **Accumulation** (many sessions → one store) |
| Retrieval | Description-matched at startup, **≤8/request on the API** | Query-time semantic, scales past that |
| Trust problem | Supply chain (13% critical-vuln claim) | Provenance and privacy |

**The recall ceiling is the structural difference that matters most.** Anthropic caps API
requests at 8 skills and warns that "with too many Skills active, Claude may fail to select
the right Skill." Skills are metadata-gated and do not scale to org-accumulated volume.
**A registry cannot become an org-memory system by adding more skills — that is an
architectural wall, not a roadmap item. [V on the caps; I on the conclusion]**

**Threat ranking:** 1) **Tessl** — high, direct, already shipping auto-capture.
2) **Anthropic** — high, platform; owns format, default marketplace, enterprise controls,
and has stated the intent. 3) **iflytek/skillhub** — moderate; commoditizes governance.
4) Aggregators — low. 5) **Memory-side standardization (UMP, Letta)** — watch; convergence
is bidirectional.

---

## Negative evidence & gaps

**Searches run that found nothing new / nothing relevant:**
- GitHub `org+memory+MCP+server`, `team+knowledge+coding+agents+MCP`, `agent+memory+
  provenance+audit`, `multi-tenant+agent+memory`, `governed+memory+agents` — beyond the
  named finds, results were dominated by awesome-lists and unrelated repos matching stray
  README words. **The org-memory category is small enough that keyword search barely
  resolves it.**
- GitHub `team+memory+agents` and `organizational+memory+agent` sorted by stars returned
  **essentially pure noise** (awesome-lists, ollama, caddy) — no org-memory project ranks
  by raw stars.
- `gh` searches for `mcp memory multi-tenant` and `mcp memory RBAC permissions` — noise only.
- HN Algolia, 2025-01-01→now, queries `agent memory`, `team memory`, `company brain`,
  `organizational memory`, `shared memory coding agents` — 215+ hits for "agent memory"
  alone, but **almost every Show HN scores 1–5 points**. The category has enormous supply
  and near-zero demonstrated demand signal. Notable exceptions: Launch HN Hyper (79),
  Show HN Sylph (10), Linggen (36), Grov (24), Modulus (15).
- `users/byterover` on GitHub → **exists but has 0 public repos**; the real org is
  `campfirein`.
- **`"institutional knowledge" AI agents`** (GitHub, by stars) — top result **6 stars**.
  The term is simply not used by this ecosystem.
- **`multi-agent memory ACL`** (GitHub, by updated) — **only 3 repos total, max 2 stars.**
  **ACL-aware agent memory is nearly unclaimed territory.**
- **`memory agents multi-tenant permissions`** (GitHub) — **2 results, max 20 stars.** Same
  conclusion, independently reached.
- **`decision log agents MCP`** (GitHub) — max 10 stars, all trivial. **Decision-provenance
  is unclaimed.**
- **HN `organizational memory`, 2025–2026** — **only 6 stories, max 2 points.** The exact
  phrase has essentially no HN mindshare, despite 215+ hits for "agent memory."
- **arXiv `collective memory` + `multi-agent`** — exactly **1** paper.
- **YC W25 and S25** — **zero** true org-memory companies. The category first appears in F25.
- Across HN 2025–2026, **almost every Show HN in this category scores 1–5 points.** Enormous
  supply, near-zero demonstrated demand signal. Exceptions: Launch HN Hyper (79), UMP (41),
  Linggen (36), Grov (24), Modulus (15), Sylph (10).
- **Product Hunt — ❌ blocked by Cloudflare, not searched at all. Real coverage gap.**
- **YC directory web UI** is JS-rendered and unfetchable; worked around via the yc-oss API.
  That API names batches winter/summer/fall/spring, so "X25"/"W26" codes do not resolve.
- Note: the repo `canhta/cred` (created 2026-07-20) **already appears in GitHub search
  results** — it is indexed and publicly discoverable.

**Explicit nulls:**
- **Caura/MemClaw funding — NULL.** Not disclosed anywhere fetched.
- **ByteRover funding — NULL.** Not disclosed anywhere fetched.
- **MCP registry total server count — NULL**; GA vs preview status — NULL.
- **Cursor Memories — UNVERIFIED** (2 doc URLs 404'd). Do not assume org-shared.
- **Copilot Memory sharing semantics/storage — UNVERIFIED** (dedicated doc not found).
- **ChatGPT Enterprise "company knowledge" — UNVERIFIED** (help.openai.com 403).
- **Google Antigravity / Jules / Gemini Code Assist memory — NULL** (404s and JS shells).
- **Codex `project_doc_max_bytes` default — NULL**; whether admins can push org AGENTS.md — NULL.
- **cursor.directory — NULL** (429).
- Storage/multi-user details for claude-mem, engram, MemPalace, cognee, AgentRecall-X, stash
  — stats verified, READMEs not fetched.

**Method gaps that materially limit confidence:**
1. **No WebSearch at all.** Commercial products with no GitHub repo and no HN post are
   systematically under-represented here. Enterprise-only vendors (Glean, Dust, Sourcegraph,
   Atlassian Rovo and similar) were **not swept**.
2. **Product Hunt was never searched** (Cloudflare-blocked). The YC directory was covered via
   the yc-oss API, which is good but not a substitute.
3. Vendor traction and funding claims — **MemClaw's eToro deployment, SOC 2 and "31,000
   downloads month one"; ByteRover's SOTA benchmark claims; SageOx's $15M; Kage's "100/100
   trust benchmark"; Tessl's $125M** — are **self-reported and not independently confirmed.**
   They are reported accurately *as claims*.
4. Funding/stage data is **NULL for nearly every Tier-1 OSS project**, including Caura/MemClaw.
5. Star counts and commit recency are point-in-time as of 2026-07-20.
6. Google's OKF was **not verified at Google directly** — it is reported as Kage's claim only.
7. Several Tier-2 projects were classified from README keyword density rather than full
   reads; individual classifications may be wrong at the margin.

**Recommended follow-ups, in priority order:**
1. **Run GateMem** ([arXiv:2606.18829](https://arxiv.org/abs/2606.18829)) and publish the
   result. A governance benchmark exists, no product has reported against it, and it is the
   cheapest available credibility win over MemClaw, Memco and the entire funded cohort.
2. Read [caura-memclaw](https://github.com/caura-ai/caura-memclaw) source properly — this
   sweep read the README, not the code. Verify the trust tiers and contradiction detection
   actually do what the README says.
3. Study [AKB](https://github.com/dnotitia/akb) (PG-native RBAC) and
   [AgentMemoryOS](https://github.com/yamantaka520/Agent-Memory-OS) (ACL clock with
   propagating revocation) as permission-model references before designing CRED's.
4. Verify MemClaw's eToro claim and Caura's funding independently.
5. Enumerate Product Hunt for 2026 memory launches (the one untouched surface).
6. Resolve Cursor Memories and Copilot Memory sharing semantics.
7. Confirm the "self-generated Skills provide no benefit" quote against the SkillsBench
   paper — it is load-bearing against Tessl's auto-generation thesis and currently unverified.
8. Verify Google OKF directly; decide whether to build on it.
