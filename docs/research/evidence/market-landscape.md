# CRED — Market & Positioning Landscape

- **Date:** 2026-07-20
- **Status:** Evidence document, adversarial framing
- **Purpose:** Find out whether CRED's thesis is wrong, early rather than late.

> **Evidence quality note.** Figures marked ✅ were pulled by me directly from the
> GitHub or npm APIs on 2026-07-20 and are verified. Other claims carry source
> URLs inline. One research thread initially returned fabricated statistics and
> self-retracted; I re-verified its load-bearing numbers independently before
> using any of them, and dropped what I could not confirm. Claims I could not
> verify are flagged ⚠️. Reddit was unreachable from this environment, so
> practitioner sentiment is drawn from HN, GitHub and blogs, which skew toward
> solo and small-team developers. Treat "no org-scale pain found" as provisional.

---

## Verdict

1. **The space is not white — it is crowded and closing.** CRED's full loop is
   already being sold. **Glen** (YC 2026) pitches "unified organizational context
   for agents and humans" that "distills work into skills and offers them back to
   the next agent" — that is CRED's evidence → knowledge → harness → learning
   loop, almost verbatim ([YC](https://www.ycombinator.com/companies/glen)).
   **Memco** ships shared agent memory with governance, SOC 2 and on-prem in
   public beta ([memco.ai](https://www.memco.ai/)). **Almanac** does the
   self-updating repo wiki for agents ([codealmanac.com](https://www.codealmanac.com/)).
   **Tessl** has $125M and is ~12 months ahead on multi-repo skill inventory,
   versioning and policy ([tessl.io](https://tessl.io/)).

2. **Harness distribution is the strongest wedge on usage evidence and the
   weakest on monetization.** ✅ `rulesync` does **807,749** npm downloads/month
   and `@intellectronica/ruler` **177,780** — roughly 985k/month of real,
   quiet demand. But every single HN launch in this category scored 1–4 points,
   and both leaders are free CLIs with no business model. Demand for the utility
   is real; demand for a *product* is unproven.

3. **Cross-agent handoff is close to dead on arrival.** ✅ `ctx-switch` does
   **87** npm downloads/month and `cli-continues` **106**, against
   `@anthropic-ai/claude-code`'s **43.6M**. That is a ~5,000x gap versus the
   rules-sync tools. No protocol covers portable session state — A2A is stateless,
   ACP binds sessions to one agent — so it is genuinely greenfield, but greenfield
   because nobody wants it, not because it is hard.

4. **Anthropic already shipped the package manager.** Claude Code has full semver
   ranges, git-tag resolution, transitive dependencies, constraint intersection,
   bundle plugins, cross-marketplace trust allowlists and `prune`
   ([docs](https://code.claude.com/docs/en/plugin-dependencies)), plus private
   marketplaces, **Required** non-removable plugins, group overrides and managed
   `claudeMd` that developers cannot override
   ([admin](https://support.claude.com/en/articles/13837433-manage-plugins-for-your-organization)).
   The "no versioning story" premise is obsolete.

5. **The premise that rules files work is under direct empirical attack.** ETH
   Zurich SRI Lab found repository context files **did not improve task success
   rates** while **increasing inference cost by 20%+**
   ([arXiv:2602.11988](https://arxiv.org/abs/2602.11988)). If the artifact does not
   measurably work, a governance plane for distributing more of it is selling into
   a contested thesis. The loudest practitioner complaints are that rules are *too
   long* and *not followed* — arguments for fewer rules and better evals, not a
   management platform.

6. **The whitest space is harness evaluation — but the market asks for it as
   *evals*, not *attribution*.** **0 of ~11** LLM observability vendors, **0 of 2**
   platform APIs and **0 of 9** DevEx vendors link config version to code outcome.
   Anthropic's Analytics API has outcomes but no config dimension; PromptLayer has
   versioned Skill Collections *and* a scoring API and has never joined them. But
   [mdarena](https://github.com/HudsonGri/mdarena) built essentially this, measured
   **+27%**, and got **65 stars / 22 HN points**. The gap is real and has been
   probed without success — which makes framing, not technology, the deciding
   variable. This is item four on CRED's differentiation list; it should be first,
   reframed as pre-merge regression testing.

7. **The single most important finding: CRED's loop inverts its own evidence
   base.** The research says the artifact that works is **small, factual,
   machine-derived and expiring**; CRED's loop is **large, narrative,
   human-curated and accumulating** — wrong on all four axes. Instruction adherence
   hits **near zero above ~6,000 words** ([AgentIF](https://arxiv.org/abs/2505.16944));
   context files grow **~4x faster than they shrink**, daily
   ([Agent READMEs](https://arxiv.org/abs/2511.12884)); **91 of 100** repos show
   configuration smells ([arXiv:2606.15828](https://arxiv.org/abs/2606.15828)); and
   ADR adoption — the closest prior art — **starves at 1–5 records in half of
   repos**. Meanwhile Vercel measured **53% → 100%** for *verifiable facts outside
   training data*. **That single line is the only evidence-backed wedge in this
   document**, and it is the opposite of "reviewed organizational learning."

---

## 1. Agent memory / context market state (2026)

### Verified traction ✅

All figures pulled from the GitHub API on 2026-07-20.

| Project | Stars | License | Notes |
|---|---|---|---|
| infiniflow/ragflow | 85,431 | Apache-2.0 | RAG engine, largest in set |
| mem0ai/mem0 | 61,245 | Apache-2.0 | Category leader for agent memory |
| khoj-ai/khoj | 35,883 | AGPL-3.0 | Personal "second brain" |
| onyx-dot-app/onyx | 31,009 | dual (MIT + EE) | Enterprise search |
| getzep/graphiti | 28,941 | Apache-2.0 | Temporal knowledge graph |
| topoteretes/cognee | 28,528 | Apache-2.0 | Graph memory |
| supermemoryai/supermemory | 28,485 | MIT | Memory/context engine |
| letta-ai/letta | 23,870 | Apache-2.0 | Stateful agents |
| airweave-ai/airweave | 6,488 | MIT | Context retrieval |
| getzep/zep | 4,766 | Apache-2.0 | Now largely a shell; see below |

**Apache 2.0 is the category default.** Eight of ten are Apache or MIT. AGPL is
the outlier. Any restrictive license would make CRED an immediate outlier in a
category where permissive licensing is the norm.

### Funding

- **Mem0** — $24M total (seed + Series A led by Basis Set, Oct 2025), YC, Peak XV,
  GitHub Fund ([TechCrunch](https://techcrunch.com/2025/10/28/mem0-raises-24m-from-yc-peak-xv-and-basis-set-to-build-the-memory-layer-for-ai-apps/)).
- **Cognee** — $7.5M seed led by Pebblebed, Feb 2026
  ([cognee.ai](https://www.cognee.ai/blog/cognee-news/cognee-raises-seven-million-five-hundred-thousand-dollars-seed)).
- **Letta** — $10M seed, Felicis, Sept 2024. No Series A announced.
- **Supermemory** — ~$3M seed (Susa, Browder Capital, SF1.vc; angels incl. Jeff Dean)
  ([supermemory.ai](https://supermemory.ai/blog/supermemory-raises-3-million-and-building-the-best-memory-engine-for-llms/)).
- **Onyx** — $10M seed, Khosla + First Round, Mar 2025
  ([TechCrunch](https://techcrunch.com/2025/03/12/why-onyx-thinks-its-open-source-solution-will-win-enterprise-search/)).

### New entrants and the ByteDance factor

- **OpenViking** (Volcano Engine / ByteDance) — open-sourced Jan 2026, ~15k stars
  within months, a filesystem-style "context database" with L0/L1/L2 tiered
  loading ([MarkTechPost](https://www.marktechpost.com/2026/03/15/meet-openviking-an-open-source-context-database-that-brings-filesystem-based-memory-and-retrieval-to-ai-agent-systems-like-openclaw/)).
  A hyperscaler-funded free competitor in the exact primitive layer.
- **ReMe** (AgentScope) — Apache 2.0, file-first memory, explicitly integrates via
  `SKILL.md` + CLI ([GitHub](https://github.com/agentscope-ai/ReMe)).

### 🚩 The two most important signals in this section

**Letta pivoted away from server-side memory to git-backed files.** On 2026-03-16
Letta announced it is refocusing around **Letta Code**, sunsetting server-side
features in favour of **client-side memory using git-backed files**, computer use,
and **"subagents and skills over legacy tools"**
([Letta](https://www.letta.com/blog/our-next-phase/)). The best-credentialed team
in agent memory (MemGPT authors, UC Berkeley) concluded that files-in-git plus
skills beat a memory service.

**Zep deprecated its self-hostable Community Edition** (2 April 2025). Their stated
reasons are the exact failure modes a solo founder faces: resource constraints
maintaining two products, discomfort with deliberately limiting features, and a
realignment so the OSS piece "doesn't compete directly with Zep's memory service"
([Zep](https://blog.getzep.com/announcing-a-new-direction-for-zeps-open-source-strategy/)).

**Has anyone moved to organizational rather than per-user memory? Yes — and they
are CRED's direct competitors.** Mem0 added agent-scope shared memory with
org isolation. Oracle shipped a "governed, unified memory core" for enterprise
agents ([Oracle](https://blogs.oracle.com/developers/oracle-ai-agent-memory-a-governed-unified-memory-core-for-enterprise-ai-agents)).
And a cluster of 2026 startups now occupies the team-memory-for-dev-agents niche
specifically:

| Company | Pitch | Overlap with CRED |
|---|---|---|
| **Glen** (YC 2026) | "Unified context for every agent and human on your team"; reads code, PRs, issues, docs, meetings; **distills work into skills and offers them to the next agent** | **Near-total.** This is CRED's loop. |
| **Memco** | Shared memory layer; captures "fixes, dead ends, decisions"; cross-IDE (Cursor, Windsurf, Copilot, Zed, VS Code, JetBrains); SOC 2, on-prem, per-seat + enterprise | **Very high.** Public beta already. |
| **Almanac** | Self-updating repo wiki; turns agent sessions into connected Markdown in-repo; agents search before changing | High on the knowledge layer |
| **Linzumi** | Team chat directing many coding agents; decisions compile into a source of truth for how agents act | High on governance |

Memco publishes benchmark claims — 50% lower cost per task, 48% faster completion,
53% fewer tokens ([memco.ai](https://www.memco.ai/)). ⚠️ Vendor-reported, unaudited.

**Read:** CRED's differentiation claims are not differentiators. They are the
category's shared pitch as of 2026.

---

## 2. The harness / rules distribution space

This was hypothesised as CRED's real wedge. It is the strongest area on usage
evidence and the weakest on monetization.

### AGENTS.md: real adoption, no spec

- 60,000+ repos; read by 20–30+ tools including Codex, Cursor, Copilot, Windsurf,
  Zed, Aider, Devin, Jules, Gemini CLI ([agents.md](https://agents.md/)).
- Donated Dec 2025 to the **Agentic AI Foundation** under the Linux Foundation,
  alongside MCP and goose; board includes Amazon, Anthropic, Block, Bloomberg,
  Cloudflare, Google, Microsoft, OpenAI
  ([Linux Foundation](https://www.linuxfoundation.org/press/linux-foundation-announces-the-formation-of-the-agentic-ai-foundation)).
- **There is no versioned schema and no formal spec.** The FAQ states it is "just
  standard Markdown. Use any headings you like." Precedence is one convention:
  nearest file wins. "AGENTS.md compliance" is therefore not a checkable property —
  which caps how much governance value can be extracted from validating it.

### 🚩 The efficacy problem

Gloaguen, Mündler, Müller, Raychev, Vechev (ETH Zurich SRI Lab / LogicStar.ai),
*"Evaluating AGENTS.md"*, [arXiv:2602.11988](https://arxiv.org/abs/2602.11988)
(submitted 2026-02-12, revised 2026-06-23): context files **did not improve task
success rates** while **increasing inference cost by over 20% on average**, across
multiple LLMs and agents, for both LLM-generated and developer-committed files.
Repository overviews — explicitly recommended by model providers — proved
unhelpful. Agents *did* follow the instructions, so compliance was not the failure
mode. Discussed at [HN 232 pts / 161 comments](https://news.ycombinator.com/item?id=47034087).

Vercel published a counter-position, "AGENTS.md outperforms skills in our agent
evals" ([HN 524 pts](https://news.ycombinator.com/item?id=46809708)). The community
is actively arguing about whether the artifact works at all.

### Fragmentation is real — but the canonical fix is a symlink

[anthropics/claude-code#6235 "Support AGENTS.md"](https://github.com/anthropics/claude-code/issues/6235)
— filed 2025-08-21, **still open**, **5,708 reactions, 342 comments**. That is the
largest single quantified demand signal in this entire document.

But: the published consensus remedy is `ln -s AGENTS.md CLAUDE.md`
([coding-with-ai.dev](https://coding-with-ai.dev/posts/sync-claude-code-codex-cursor-memory/)),
and a [widely-referenced gist](https://gist.github.com/yurukusa/d36197848911f025add142abefcde685)
concludes the no-tooling symlink "is the recommended approach — it's officially
documented, needs no tooling, **and cannot drift**." Two kill-risks follow: the
"40-line script" objection is the *prevailing advice*, not a strawman; and
Anthropic can close #6235 in one release and vaporize the category.

### Cross-tool compilers — real usage, zero commercial gravity ✅

| Tool | Stars | npm/mo ✅ | Status |
|---|---|---|---|
| dyoshikawa/rulesync | 1,245 | **807,749** | Very active |
| intellectronica/ruler | 2,811 | **177,780** | Active |
| steipete/agent-rules | 5,692 | — | **ARCHIVED** |
| block/ai-rules | 112 | — | **Stalled**, 0 commits in 4 weeks |
| PatrickJS/awesome-cursorrules | 40,364 | — | Content list, not tooling |

rulesync's 808k downloads on only 1.2k stars is the strongest genuine usage signal
in this report — downloads far outrun mindshare, implying CI and automation
embedding. But the highest-starred entry is archived, and the one corporate-backed
entrant (Block, an AAIF board member) has stalled.

### 🚩 The graveyard

Every launch attacking distribution/sync has failed on HN:

| Launch | Points | Comments |
|---|---|---|
| [Ruler — same rules to all coding agents](https://news.ycombinator.com/item?id=44062058) | 3 | 0 |
| [rulesync — single source of truth](https://news.ycombinator.com/item?id=48051242) | 1 | 1 |
| [rulesync — bulk rule management](https://news.ycombinator.com/item?id=44382989) | 1 | 0 |
| [AlignTrue — sync rules across agents, repos, teams](https://news.ycombinator.com/item?id=46193951) | 1 | 0 |
| [LynxPrompt — federated AI config manager](https://news.ycombinator.com/item?id=47231361) | 2 | 1 |
| [Rulegen — auto-generate CLAUDE.md/.cursorrules](https://news.ycombinator.com/item?id=47205953) | 4 | 1 |
| [Straion — dynamic AGENTS.md context](https://news.ycombinator.com/item?id=47112858) | 4 | 1 |
| [Agentdex — dashboard for skills, agents, rules](https://news.ycombinator.com/item?id=48723898) | 3 | 1 |

Meanwhile *authoring* and *efficacy* content dominates: [AGENTS.md open format, 837 pts](https://news.ycombinator.com/item?id=44957443),
[Claude Skills, 816 pts](https://news.ycombinator.com/item?id=45607117),
[Writing a good CLAUDE.md, 748 pts](https://news.ycombinator.com/item?id=46098838).

**Many founders have felt this itch. None has found an audience.**

### 🚩 Anthropic already built the package manager

From [plugin-dependencies](https://code.claude.com/docs/en/plugin-dependencies) and
[plugin-marketplaces](https://code.claude.com/docs/en/plugin-marketplaces):

- Full **semver ranges** (`~2.1.0`, `^2.0`, `>=1.4`) resolved against git tags
  (`{plugin}--v{version}`), via `claude plugin tag --push`
- **Transitive dependencies**, auto-installed; **constraint intersection** across
  dependents with explicit `range-conflict` / `no-matching-tag` errors
- **Bundle plugins** — a manifest of pure dependencies = a curated org-wide set
  behind one install
- **Cross-marketplace trust allowlists** (`allowCrossMarketplaceDependenciesOn`),
  deliberately non-chaining
- `claude plugin prune`, orphan cleanup, rename/removal migration maps
- Sources: `github`, `url`, `git-subdir` (sparse monorepo clone), `npm`; pin by
  `ref` or exact `sha`

Org governance ([admin docs](https://support.claude.com/en/articles/13837433-manage-plugins-for-your-organization)):
private marketplaces (GitHub sync ≤500 plugins), four per-plugin states including
**Required** (auto-provisioned, non-removable), Enterprise group-level overrides,
managed-scope plugins users cannot edit, allowlist/blocklist enforced before any
network or filesystem operation. Managed **`claudeMd`** injects org-wide
instructions into every session and **cannot be excluded or ignored** by developers
([admin-setup](https://code.claude.com/docs/en/admin-setup)). Skills have
workspace-scoped API distribution with owner-only provisioning
([enterprise skills](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/enterprise)).

The declarative-dependency request ([#27113](https://github.com/anthropics/claude-code/issues/27113),
Feb 2026) is **closed — shipped**. Note it drew only **7 reactions** versus 5,708
on the cross-tool interop issue. **Demand for governance was three orders of
magnitude weaker than demand for interop.** That is the most important asymmetry
in this document.

**Remaining Anthropic gap:** no audit logging or usage telemetry documented
anywhere in the org plugin/skill management docs.

### Incumbents own the control plane

- **Cursor** — Team/Enterprise rules created and **enforced** from the dashboard;
  enforced rules "can't be turned off" by members; server-side, auto-synced
  ([docs](https://cursor.com/help/customization/rules)). **Organizations** GA
  3 Jun 2026. Claims 64% of the Fortune 500 ([enterprise](https://cursor.com/enterprise)).
  **Cursor has acquired Continue.dev** — the nearest open pure-play (Continue Hub
  was a shared registry of rules/prompts/assistants); banner live on
  [continue.dev](https://continue.dev/). ⚠️ Terms and date unconfirmed; absence of
  a press release suggests a small acqui-hire.
- **GitHub Copilot** — organization custom instructions **GA 2 Apr 2026**
  ([changelog](https://github.blog/changelog/2026-04-02-copilot-organization-custom-instructions-are-generally-available/));
  enterprise policy in AI Controls that orgs **cannot override**
  ([docs](https://docs.github.com/en/copilot/concepts/policies)). Real temporary
  gap: org instructions apply only to Chat on github.com, code review and the cloud
  agent — **not the IDE**.
- **Hyperscalers** — [AWS Agent Registry](https://aws.amazon.com/blogs/machine-learning/the-future-of-managing-agents-at-scale-aws-agent-registry-now-in-preview/),
  [Google Agent Registry](https://docs.cloud.google.com/gemini-enterprise-agent-platform/govern/agent-registry),
  [Microsoft Agent 365 Registry](https://techcommunity.microsoft.com/blog/agent-365-blog/whats-new-in-agent-365-%E2%80%93-june-2026/4535107)
  with Purview auditing.

### Commercial pure-plays

- **Tessl** — $125M total ($25M seed boldstart/GV; **$100M Series A led by Index**,
  Nov 2024), founder **Guy Podjarny (Snyk)** — has sold dev-security governance to
  enterprises before ([TechCrunch](https://techcrunch.com/2024/11/14/tessl-raises-125m-at-at-500m-valuation-to-build-ai-that-writes-and-maintains-code/)).
  ⚠️ Valuation disputed ($500M TechCrunch vs $750M Fortune). Now explicitly a
  *"management layer for agent skills"* with policy gating, version management and
  contribution governance ([tessl.io](https://tessl.io/)). **Skill Inventory**
  (17 Jun 2026) scans GitHub *organizations* for skill sprawl, naming duplication
  and **ownership drift** across repos
  ([blog](https://tessl.io/blog/how-i-scan-my-agent-context-across-github-with-skill-inventory/)) —
  precisely CRED's multi-repo thesis, ~12 months ahead.
- **Semgrep Guardian** — org rules enforced across Claude Code, Cursor and Windsurf
  via **hooks that fire on every file write**, "ensuring a scan regardless of what
  the agent does" ([Semgrep](https://semgrep.dev/products/semgrep-guardian/)).
  **Architecturally superior to instruction injection**: it does not depend on the
  model choosing to comply. Anyone selling rule *distribution* is selling advisory
  text; Guardian sells enforcement.
- **Qodo** — ⚠️ reported $70M Series B for guardrails enforcing internal standards
  on agent output ([GovInfoSecurity](https://www.govinfosecurity.com/qodo-targets-ai-code-risks-quality-70m-series-b-raise-a-31317); source returned 403, unverified).

### MCP registries: discovery solved, governance not

PulseMCP ~15,930 servers, Smithery ~7,300, official registry ~2,000; one aggregator
counts **76,803 servers across five registries** as of 2026-07-17
([TrueFoundry](https://www.truefoundry.com/blog/best-mcp-registries)).
`modelcontextprotocol/registry` ✅ **7,049 stars**. Governance is the acknowledged
weak point: a third of servers still run pre-Dec-2025 schemas; enterprise gaps
include **no audit trail of which agent accessed which tools** and no
registry-level access governance; the OpenClaw exposure (Jan 2026) hit 42,000+
instances with MCP enabled and no auth
([NimbleBrain](https://nimblebrain.ai/blog/state-of-mcp-security-2026/)).
The official registry launch scored [19 points on HN](https://news.ycombinator.com/item?id=45176580).

`anthropics/skills` ✅ **162,779 stars** — Anthropic owns skills mindshare outright.

---

## 3. Cross-agent handoff

### The protocols genuinely do not solve this

- **A2A** (Google Apr 2025 → Linux Foundation; v1.0 Apr 2026; [150+ orgs](https://www.linuxfoundation.org/press/a2a-protocol-surpasses-150-organizations-lands-in-major-cloud-platforms-and-sees-enterprise-production-use-in-first-year))
  is **stateless**. [Only identifiers cross the wire](https://a2a-protocol.org/latest/topics/life-of-a-task/);
  `contextId` is explicitly internal and private to each agent. A2A tells Agent B
  *that* work is related; it cannot tell it *what happened*. This is enterprise
  agent interop, not coding-agent handoff.
- **ACP** (Zed, Aug 2025) is **editor↔agent, not agent↔agent**. Per the
  [spec](https://agentclientprotocol.com/protocol/session-setup), `loadSession`
  resumes *within the same agent* and the session ID is an opaque token bound to
  one agent instance. Explicitly not portable.
- **MCP is moving away from this.** Sampling is
  [deprecated as of protocol version 2026-07-28 (SEP-2577)](https://modelcontextprotocol.io/specification/draft/client/sampling) —
  "new implementations SHOULD NOT adopt it." MCP is narrowing toward tools.
- **AGNTCY / AAIF** — discovery, identity, messaging, observability. Not session
  portability.

**So there is a real standards gap.** The question is whether it is unfilled
because it is hard or because nobody wants it.

### 🚩 Revealed preference says nobody wants it ✅

npm downloads, last month, pulled directly:

| Package | Downloads/mo ✅ |
|---|---|
| `@anthropic-ai/claude-code` | **43,646,662** |
| `rulesync` | 807,749 |
| `@intellectronica/ruler` | 177,780 |
| `cli-continues` | **106** |
| `ctx-switch` | **87** |

Handoff tooling has roughly **200 downloads/month combined** against ~985k for
rules-sync tooling — a **~5,000x gap** between two categories of similar age and
similar technical difficulty. This is the cleanest natural experiment in the
document, and it is unambiguous.

### Formats are readable but unstable

Claude Code writes JSONL to `~/.claude/projects/`; Codex writes JSONL rollouts to
`~/.codex/sessions/YYYY/MM/DD/`; Cursor and Gemini CLI expose `transcript_path` via
hooks. But [openai/codex#3827](https://github.com/openai/codex/discussions/3827)
confirms **no stability or backward-compatibility guarantees**. Every converter is
built on sand — a permanent maintenance tax and a poor moat.

Most existing tools deliberately chose **markdown summaries over transcript
conversion**, suggesting legibility rather than fidelity is the binding
constraint — which further reduces the technical moat, since a markdown summary is
something an agent can already write on request.

### Vendors are solving portability inside walled gardens

Codex syncs CLI ↔ Desktop ↔ VS Code ↔ Web on a shared backend with
[no third-party integration](https://codex.danielvaughan.com/2026/04/08/cross-surface-session-sync/).
Portability is arriving intra-vendor, not cross-vendor.

### The statistic that gets over-read

[Pragmatic Engineer, 906 respondents, Jan–Feb 2026](https://newsletter.pragmaticengineer.com/p/ai-tooling-2026):
**70% use 2–4 tools simultaneously**, 15% use 5+. This establishes that developers
*own* multiple tools. It says **nothing** about moving one task between them. The
survey's own framing is that tools are *complementary*. **Multi-tool ownership is
not handoff demand** — and the unresolved crux is whether developers switch
mid-task or simply partition tasks across agents. Available evidence points to
partitioning.

---

## 4. AI engineering measurement

### DORA

**There is no 2026 DORA report.** [dora.dev/research](https://dora.dev/research/)
lists annual reports 2014–2025 only. Anyone citing "DORA 2026 findings" is citing
the companion *"ROI of AI-assisted Software Development"*
([Google Cloud](https://cloud.google.com/resources/content/dora-roi-of-ai-assisted-software-development)),
which is gated.

**DORA 2025** (~5,000 professionals, 100+ hours of interviews,
[announcement](https://cloud.google.com/blog/products/ai-machine-learning/announcing-the-2025-dora-report)):
AI adoption **90%** (+14 points YoY); **>80%** report productivity gains; **70%**
trust AI-generated code — meaning **30% have little or no trust**. The central
finding: AI adoption correlates **positively with throughput and negatively with
delivery stability**. Framing thesis: *"AI doesn't fix a team; it amplifies what's
already there."* ⚠️ Faros's writeup of the same report says 95% adoption vs
Google's 90% — use the primary source.

The **DORA AI Capabilities Model** ([dora.dev/capabilities](https://dora.dev/capabilities/))
names seven, two of which are directly relevant to CRED: **AI-accessible internal
data** ("moves it from a generic assistant to a specialized expert") and **version
control** ("ensures reproducibility and traceability… as AI accelerates the
velocity of change"). CRED's thesis has institutional backing at the capability
level, if not at the product level.

### DX — the most quotable dataset

DX's **AI Measurement Framework** ([guide](https://getdx.com/blog/ai-measurement-framework-guide/))
has three dimensions — Utilization, Impact, Cost — including agent-native metrics:
**tasks assigned to agents**, **human-equivalent hours**, **agent hourly rate**.

| Metric | Value |
|---|---|
| Developers using AI ≥ monthly | **93%** |
| Avg. time savings | **3.9 hrs/week** (daily users 4.72) |
| **AI-authored share of merged code, Q1 2026** | **27.4%** |
| PR throughput, daily AI users vs non-users | 2.4 vs 1.5/week |
| **Durable throughput improvement over one year** | **only 10–15%** |
| PR throughput lift attributable to AI | **~7.8%** |

That last pair is the under-quoted finding: **the durable organizational gain is
~8–15%**, far below headline per-developer time-savings claims.

Brian Houck's 2026 revision of **DX Core 4** ([newsletter](https://newsletter.getdx.com/p/revisiting-the-dx-core-4))
sets the governing rule: **"AI-specific telemetry should be treated as diagnostic
context rather than a replacement for outcome-oriented measurement."** DX's product
does AI-vs-human code tracking computed locally without exposing source, and **A/B
testing of AI vendors head-to-head** ([getdx.com/ai-measurement](https://getdx.com/ai-measurement/)).

### 🚩 The attribution ceiling — nobody reaches L4

| Level | Who reaches it |
|---|---|
| L0 seat/license | everyone |
| L1 tool ("this PR was Cursor-assisted") | Faros, Jellyfish, Swarmia, LinearB, Cortex, Opsera |
| L2 mode (editor vs cloud agent vs review agent) | **Swarmia**, partly Jellyfish |
| L3 manual cohort experiment | **Faros only** |
| **L4 config/harness version → outcome** | **nobody** |

- **Faros AI** — deepest telemetry; April 2026 shipped **Experiments**, an MCP
  server, and **Claude Code OpenTelemetry ingestion** (sessions, tokens, cost, tool
  acceptance rates). From **$29/contributor/module/month**
  ([platform](https://www.faros.ai/platform/ai-transformation)). Experiments
  compares "Codex vs Claude Sonnet" — but as a *human-declared cohort*, not
  per-PR provenance.
- **LinearB retreated** — deprecated its Copilot and Cursor dashboards effective
  **2026-04-02**, citing unreliable third-party API data
  ([release notes](https://linearb.helpdocs.io/article/b7okinmoom-release-notes-2026)).
  A vendor exiting AI attribution is a meaningful negative signal about data quality.
- **Swarmia** publishes the industry's detection ceiling honestly: the `Made-with:
  Cursor` git trailer, a `claude-code-assisted` label, and a low-confidence
  fallback of *"any commit by an author who used an AI tool in the previous 24
  hours"* ([docs](https://help.swarmia.com/features/ai-tools/ai-tool-detection-and-filters)).
- **🚩 Sleuth pivoted off DORA metrics into agent config governance** — "Sleuth
  Skills," a control plane for skills, rules, MCP servers and prompts with approval
  workflows and audit trails, tagline **"Define Your AGENTS.md Once. Distribute
  Everywhere"** ([sleuth.io](https://www.sleuth.io/)). It owns the *write* side and
  has abandoned the *read* side. **It is one product decision from being the
  natural incumbent for outcome tracing.**
- **Opsera** was named a Leader in the **2026 Gartner MQ for Developer Productivity
  Insight Platforms** ([PR](https://www.prnewswire.com/news-releases/opsera-named-a-leader-in-the-2026-gartner-magic-quadrant-for-developer-productivity-insight-platforms-302770434.html)) —
  the existence of that MQ category signals market maturity, and a crowded one.

### Three independent 2026 studies converge: faster, worse

| Source | N | Speed | Quality |
|---|---|---|---|
| [Faros "Acceleration Whiplash"](https://www.faros.ai/research/ai-acceleration-whiplash) | 22k devs / 4k teams | tasks/dev +34%, epics/dev +66% | bugs/dev **+54%**, incident:PR ratio **>3x**, review time **5x**, **+31% merging unreviewed** |
| [Cortex 2026 Benchmark](https://www.cortex.io/post/ai-is-making-engineering-faster-but-not-better-state-of-ai-benchmark-2026) | n/s | PRs/author +20% YoY | incidents/PR **+23.5%**, CFR **~+30%** |
| [Opsera 2026 Benchmark](https://opsera.ai/resources/report/ai-coding-impact-2026-benchmark-report/) | 250k devs / 60+ orgs | time-to-PR **−58%** | AI PRs wait **4.6x longer** in review, **+15–18% vulns** |

All three are vendor-published and each sells the remedy — discount accordingly.
But agreement across three collection methodologies plus DORA's independent
stability finding is hard to dismiss. ⚠️ **GitClear returns HTTP 403 to all
automated access**; its churn/copy-paste figures could not be verified and are
deliberately not cited here.

### 🚩 The critical question, answered honestly

**Is "which context/harness version produced accepted work" a recognized need or an
invented one? Verdict: the pain is real and quotable. CRED's framing of it is not
the one the market uses. And the closest existing product shipped and was ignored.**

**Demand is real and verbatim:**

> "everyone's writing CLAUDE.md files now but **nobody knows if theirs actually
> works**" — [HN 47655107](https://news.ycombinator.com/item?id=47655107)

> "as devs we're **overly reliant on vibes**… especially so if you're working in an
> engineering organization where **a bad edit to AGENTS.md can cause silent
> regressions for everyone in the codebase**" — [HN 48141150](https://news.ycombinator.com/item?id=48141150)

> "[I test] AGENTS.md changes by reverting and resubmitting prompts, since **it's
> not easy to write evals**" — furyofantares

That last is textbook unmet need: a painful manual workaround.

**But the counter-evidence is brutal.** [mdarena](https://github.com/HudsonGri/mdarena)
is essentially CRED's thesis, already built — it mines merged PRs, checks out
pre-PR state, runs the agent with and without each CLAUDE.md variant, and grades
against real test suites. The author measured **+27%** on a production monorepo.
Its Show HN got **22 points and 4 comments**
([HN 47655078](https://news.ycombinator.com/item?id=47655078)); **65 GitHub stars.**
Tessl sells the eval-registry version ([blog](https://tessl.io/blog/your-agentsmd-file-isnt-the-problem-your-lack-of-evals-is/)) —
42 points. **So the accurate statement is not "nobody built it." It is "it has been
built, small and free, with a credible result, and nobody showed up."**

**Clean negatives across the stack.** Of ~11 LLM observability platforms surveyed
(Langfuse, Braintrust, LangSmith, Helicone, W&B Weave, PromptLayer, Latitude,
PostHog, Statsig, Arize), **zero** ship code-outcome attribution. All have the
adjacent primitive — a score/feedback API keyed to a trace ID — but none treats
PR-merged / reverted as a first-class metric, and none has VCS integration or a
delayed-attribution model. Humanloop was **acquired by Anthropic and sunset**.

**The sharpest datapoint:** **PromptLayer already ships versioned "Skill
Collections"** — versioned folders of `SKILL.md`, `CLAUDE.md`, `AGENTS.md` with
commit messages, release labels, compare, rollback and SDK pull into `.claude/`
([docs](https://docs.promptlayer.com/features/skill-collections/overview)) — **and
a scoring API** ([docs](https://docs.promptlayer.com/features/prompt-history/scoring-requests)).
**It has never connected the two.** Config versioning is monetized; outcome
attribution is not.

**Platform vendors each hold exactly half the join:**
- [Claude Code Analytics API](https://platform.claude.com/docs/en/build-with-claude/claude-code-analytics-api) —
  metrics include `commits_by_claude_code`, `pull_requests_by_claude_code`, per-tool
  accepted/rejected. **No dimension for CLAUDE.md version, skills, or MCP config.**
  Counts PRs *created*, not merged.
- [GitHub Copilot usage metrics](https://docs.github.com/en/copilot/reference/copilot-usage-metrics/copilot-usage-metrics) —
  exposes `total_merged`, time-to-merge, sliceable by feature/ide/**model**/language.
  **No dimension for `copilot-instructions.md` version.**

Outcomes exist on one side, config versions on the other. **Nobody has built the
join.**

**🚩 The vocabulary test — a red flag.** HN Algolia returns **zero** results across
2025–2026 for phrasings like "prompt version merged PR attribution" or "config
version to code outcome." People say *"I don't know if my CLAUDE.md works."* Nobody
says *"I need config-version-to-merge-outcome attribution."* **The pain has a folk
name, not CRED's name for it.** That is a positioning problem, and it plausibly
explains mdarena's flat launch.

**Most important nuance for strategy:** the demand is articulated as **pre-merge
evals** (deterministic, replayable, "did pass rate go up?"), not post-hoc
attribution from shipped code back to config version. Those are different products —
a testing problem versus an analytics problem. **The market is asking for the
former; CRED's stated framing is the latter.**

---

## 5. OSS business model benchmarks

### The winning pattern: permissive core + thin `/ee` + cloud

**Langfuse is the closest analog and the best template.** MIT everywhere except
`/ee` directories, which ship **as source in the repo** but require a license key
to run. What is in `/ee`: SCIM, audit logs, data retention, enterprise support —
that is all. What is MIT: tracing, evals, prompt management, experiments,
annotation, playground, **and standard SSO**. In June 2025 they moved *previously
commercial* features **into** MIT — LLM-as-a-judge, annotation queues, prompt
experiments, playground — reasoning that these are "market standard at this point
and should be freely available"
([Langfuse](https://langfuse.com/blog/2025-06-04-open-sourcing-langfuse-product)).

Outcome: **$4M seed only** (Sept 2023, Lightspeed/La Famiglia/YC W23 — they had
Series A term sheets and declined), $1.1M ARR in 2024, 20k+ stars but **26M+ SDK
installs/month**, 2,000+ paying customers including 19 of the Fortune 50, acquired
by ClickHouse Jan 2026 alongside its $400M Series D at $15B
([ClickHouse](https://clickhouse.com/blog/clickhouse-acquires-langfuse-open-source-llm-observability)).
**A $4M-seed company reached a strategic acquisition with one tiny `/ee` folder.**

### Other benchmarks

| Company | License | Outcome |
|---|---|---|
| **Cal.com** | AGPLv3 + `/ee` | $1.6M → **$5.1M ARR**; ~7,000 seats at $12/mo ≈ $1M ARR ([Latka](https://getlatka.com/companies/calcom)) |
| **n8n** | Sustainable Use License | **~$40M ARR, $2.5B valuation** (Oct 2025); permanent "not open source" backlash ([issue #40](https://github.com/n8n-io/n8n/issues/40)) but **no successful fork** |
| **Supabase** | Apache 2.0 | ~$170M ARR May 2026; Series F $500M @ $10.5B ([CNBC](https://www.cnbc.com/2026/06/04/database-startup-supabase-raises-500-million-10point5-billion-valuation.html)) |
| **Sentry** | FSL (Fair Source) | **$100M+ revenue**, 100k+ cloud customers; 10,000+ orgs' compliance teams cleared it ([Sentry](https://blog.sentry.io/sentry-is-now-fair-source/)) |
| **Onyx** | MIT + EE | Netflix, Ramp, Thales ([TechCrunch](https://techcrunch.com/2025/03/12/why-onyx-thinks-its-open-source-solution-will-win-enterprise-search/)) |
| **Dify** | Modified Apache 2.0 | Anti-multi-tenant clause + logo retention ([LICENSE](https://github.com/langgenius/dify/blob/main/LICENSE)) — the most surgical restriction available |

**Supabase datapoint relevant to CRED's ICP:** AI agents, not humans, now create
the majority of new Supabase databases, with Claude Code the single largest source
([Sacra](https://sacra.com/c/supabase/)).

### What went wrong for relicensers

- **Redis → Valkey** is the worst and best-documented. BSD → SSPL (Mar 2024) → back
  to AGPLv3 (May 2025). **37.5% of Redis contributors (9 of 24) stopped
  contributing**; Valkey grew 18 → 49 contributors in 18 months and now ships
  **80 PRs/mo vs Redis's 42**
  ([Percona](https://www.percona.com/blog/community-erosion-post-license-change-quantifying-the-power-of-open-source/)).
  **Distro capture was fatal:** Fedora 42, Ubuntu 26.04 LTS, Debian 13 backports
  and Arch all default to Valkey. Going back did not work.
- **HashiCorp → OpenTofu** — BSL Aug 2023, fork within days, CNCF April 2025, 9.8M
  release downloads, ~300% YoY growth; **38% of Terraform users evaluating or
  migrating**. Notably this happened *without discernible effect on HashiCorp's
  revenue* — the relicense didn't kill the business, it killed the community and
  spawned a permanent Linux Foundation competitor.
- **Elastic → OpenSearch** — SSPL Jan 2021, returned to AGPLv3 Sept 2024. Trust
  damage persists ([Socket](https://socket.dev/blog/developers-burned-by-elasticsearch-license-change-arent-going-back)).
  The real dispute was **trademark**, not licensing — trademark policy is a cheaper
  defense than license restriction.
- **Grafana's AGPL relicense (2021) produced no fork**, because AGPL is
  OSI-approved and never triggered the "not open source anymore" narrative
  ([Grafana](https://grafana.com/blog/grafana-loki-tempo-relicensing-to-agplv3/)).
  This is the key asymmetry.

**Pattern:** moving *away from* OSI-approved licenses reliably produces a
foundation-backed fork within ~1 month, hyperscaler sponsorship, distro capture,
permanent contributor loss, and a reversal that does not restore the community.
Every disaster came from *moving the line after the community formed around a
promise*. n8n survives its backlash precisely because the restriction was there
from the start.

### Stars → ARR: no reliable multiple exists

- The only durable benchmark is Bessemer/a16z's **<5% of *users* monetize**
  ([BVP](https://www.bvp.com/atlas/roadmap-open-source),
  [a16z](https://a16z.com/open-source-from-community-to-commercialization/)).
  Note: percent of **users**, not stargazers. The gap between those is where most
  founders' projections die.
- Stars measure interest, not usage. The ROSS Index is explicit that stars signal
  interest "as opposed to a measure of usage in production," used to highlight
  momentum "**before meaningful revenue**" ([Runa](https://runacap.com/ross-index/)).
- Counterexamples are brutal: nginx serves 450M websites with 12,200 stars;
  PostCSS has 27,000 stars and **~$12,000/year** revenue.
- **Better metric for CRED:** Langfuse's headline number was not stars (20k), it
  was **26M+ SDK installs/month**. Instrument package installs and opt-in
  self-hosted telemetry from day one.
- Bessemer: **100 unique monthly contributors is already "rare territory."**

### a16z's three named failure modes

1. **Product-market fit without value-market fit** — strong adoption, no paying
   customers. The most common death.
2. **Enterprise sales outpacing community growth** — a signal of *weak* PMF.
3. **The commercial offering kills credibility.**

### "Commoditize your complement" — the trap for CRED specifically

If CRED open-sources a context/harness plane, it is commoditizing a complement to
*someone else's* product — most likely the model providers' and agent vendors'.
The question to answer honestly: **whose complement am I, and can they absorb me?**
Zep's Community Edition death and Langfuse's absorption into ClickHouse are both
this dynamic resolving against the small player. ClickHouse's stated strategy is
explicitly to "own the full observability stack for AI-native companies"
([Sacra](https://sacra.com/c/clickhouse/)).

**Zep's realignment logic is the single most actionable insight here:** open-source
the layer that **powers** your product but does not **substitute** for it. Zep
open-sourced the engine (Graphiti) and kept the service closed. That is a
structurally more stable arrangement than open-sourcing a product and holding back
features.

### Recommendation

Apache 2.0 core (patent grant; enterprise legal prefers it; it is the category
norm — 8 of 10 comparable projects). Small, clearly-marked `/ee` with exactly
Langfuse's line: **SCIM, audit logs, data retention/residency, RBAC beyond basics,
support SLAs.** Never gate anything a developer needs to evaluate whether the
product works. **Do not** start with BSL/FSL/SUL — strip-mining risk does not exist
below ~$50M ARR, and adoption is the entire game at this stage. Publish the paid
line on day one; you get exactly one chance to set it.

---

## 6. Risk and disconfirming evidence

> ⚠️ **Provenance warning.** The adversarial thread that produced much of this
> section exhausted its search budget and **fabricated several citations before
> self-retracting**. Excluded as non-existent: an "ICSA 2026 / 63% of ADRs" paper,
> GitHub 2017 "93%/60%" doc figures, Ratol & Robillard, a Storey & Barnett /
> Gartner "70% of KM projects fail" claim, Fluri 97%, and 2024 Stack Overflow
> percentages. Also flagged as likely fabricated by a second thread: a "SpaceX
> acquired Cursor" claim — **do not use it, and treat all Cursor financials as
> unverified.** Everything retained below carries a checkable source, but
> **spot-check every citation in this section before it reaches an investor deck.**

### The five strongest reasons CRED fails

**1. CRED's loop inverts its own evidence base on all four axes.** This is the
sharpest single finding in the document. The research says the artifact that works
is **small, factual, machine-derived and aggressively expiring.** CRED's loop as
specified is **large, narrative, human-curated and accumulating.** Every axis points
the wrong way:

- **Scale kills adherence.** [AgentIF](https://arxiv.org/abs/2505.16944) (50 real
  agentic apps): best model reaches instruction-satisfaction **27.2%**, and
  **"when instruction length exceeds 6,000 words, the ISR scores of all models are
  nearly 0."** [IFScale](https://arxiv.org/abs/2507.11538) (20 models): 68% accuracy
  at 500 instructions, with documented **bias toward earlier instructions** — newly
  curated learnings get weighted *least*. [Chroma Context Rot](https://www.trychroma.com/research/context-rot):
  focused prompts beat full prompts, and **coherent, logically-organized haystacks
  perform worse than shuffled ones** — a well-structured monorepo is the adversarial
  input. ~6,000 words is a *modest* org knowledge base. **Adherence hits zero
  exactly at the scale CRED is designed to reach.**
- **Curation fails as a ratchet, not as apathy** — which is worse, because the naive
  "devs won't maintain it" bear case is empirically false.
  [Agent READMEs](https://arxiv.org/abs/2511.12884) (2,303 files, 1,925 repos):
  **67.4%** of Claude Code context files are modified across multiple commits,
  median update interval **24.1 hours** — but **additions median 57 words/commit
  against deletions median <15**, a ~4x ratchet, daily, forever. Result: Flesch
  **16.6**, "very difficult," typical of dense legal documents.
  [Configuration Smells in AGENTS.md](https://arxiv.org/abs/2606.15828): **91 of
  100** popular repos had at least one smell — Context Bloat 42%, Conflicting
  Instructions 28%, **Init Fossilization 24%** (generated by `/init`, never reviewed
  again; the authors ruled out repo dormancy).
- **The closest prior art starves.** Buchgeher et al.,
  [*Using ADRs in Open Source Projects*](https://doi.org/10.1109/access.2023.3287654)
  (IEEE Access 2023): **"approximately half of all repositories using ADRs contain
  only between one and five records."** Motivated teams who deliberately adopted a
  typed, reviewed, versioned decision artifact — median outcome is abandonment after
  single digits.

**This inverts CRED's PRD risk register.** Risk #2 currently reads *"automatic
writes poison organizational memory."* The measured danger is **human writes that
are never deleted.** Retention, expiry and forced pruning are more load-bearing than
write-approval; the `validity` field must be **enforced and expiring by default**,
not merely recorded.

**2. The terminal value link has a published null result against it.**
[arXiv:2602.11988](https://arxiv.org/abs/2602.11988) found context files
**"do not generally improve task success rates"** across LLMs and agents, for both
LLM-generated and developer-committed files, while **costs rise >20%**, and
**"repository overviews, although popular and recommended by model providers, are
not helpful."** This is near-identical to **CRED's PRD acceptance criterion #3**
("a token-bounded context package outperforms naive document or repository
re-reading"). The null result is already in buyers' heads —
[232 pts / 161 comments](https://news.ycombinator.com/item?id=47034087), with the
representative reaction: *"Many of the practices in this field are mostly based on
feelings and wishful thinking, rather than any demonstrable benefit."*

**3. Every incumbent has shipped 70–90% of the loop, and the remaining gaps are
preview-stage.** The answer to "why won't Anthropic/GitHub just ship this" is:
**they mostly already have.**

| Layer | GitHub | Anthropic | Google | Cursor |
|---|---|---|---|---|
| Org typed knowledge | Org instructions **GA Apr 2026** | Skills, 4-level scoping | GEMINI.md 3-tier | Team Rules, admin-**enforced** |
| Context packages | Copilot Spaces **GA Sep 2025, free tier** | Skills progressive disclosure | Code Assist Enterprise | Project Rules |
| Harness distribution | `.github/agents/*.agent.md` | Plugins + marketplaces | CLI Extensions | Team MCP distribution |
| Outcome telemetry | Repo metrics **GA Jul 2026** | Analytics API GA Jul 2026 | — | Analytics API |
| **Reviewed learning** | **Copilot Memory (preview)** | Auto-memory (machine-local) | **Auto Memory w/ review gate** | Memories *removed* |

The three most damaging specifics:

- **[GitHub Copilot Memory](https://docs.github.com/en/copilot/concepts/agents/copilot-memory)**
  stores repo-level facts — conventions, architectural decisions, build commands —
  **shared with all users with repo access**, with **citations to supporting code,
  validation against the current codebase, and 28-day expiry of unused entries.**
  That is evidence → typed knowledge → context → execution *with provenance and
  decay*, in the platform where the code already lives. It is missing only an
  approval gate and org scope.
- **[Gemini CLI Auto Memory](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/auto-memory.md)**
  — "mines your past sessions in the background and proposes durable memory updates
  and reusable Agent Skills. **You review each candidate before it becomes
  available.**" **Google has already shipped CRED's differentiating review gate.**
  The remaining wedge lives entirely in the disclaimers: experimental, off by
  default, per-project rather than cross-workspace.
- **[OpenAI Frontier](https://techcrunch.com/2026/02/05/openai-launches-a-way-for-enterprises-to-build-and-manage-ai-agents/)**
  (Feb 2026) claimed the category name with CRED's pitch nearly verbatim: agents
  get "shared context, onboarding, hands-on learning with feedback, and clear
  permissions and boundaries," with "a feedback loop that improves them the way a
  performance review improves an employee." It manages non-OpenAI agents.

This is **core roadmap, not ecosystem leftovers**: GitHub GA'd three CRED layers in
six months — org instructions (Apr 2026), a CLI with "cross-session repository memory
that preserves learned conventions" (Feb 2026), and repo-level PR-outcome metrics
(Jul 2026) — and **Google begins billing Vertex Memory Bank 1 Sept 2026 and Semantic
Governance Policies 1 Aug 2026** at $0.30/GiB-month
([pricing](https://cloud.google.com/vertex-ai/pricing)). Nobody builds metered
billing for an afterthought. Both standards were also deliberately commoditized —
Anthropic open-sourced Skills and
[donated MCP to the Linux Foundation](https://www.anthropic.com/news/donating-the-model-context-protocol-and-establishing-of-the-agentic-ai-foundation);
OpenAI donated AGENTS.md. Textbook commoditize-your-complement.

Three further encroachment specifics worth knowing:

- **Cursor shipped auto-generated Memories in 1.0 (Jun 2025) and then removed
  them**, with the migration path being "convert them into Rules." ⚠️ Medium
  confidence on the version. This cuts both ways: it is an *opening* (nobody owns
  team memory) and a *warning* (the obvious implementation did not hold at a company
  with unlimited resources). Cursor is re-approaching via **Bugbot "Learned Rules,"
  generated automatically from the team's GitHub activity**, plus
  `@cursor remember [fact]` ([docs](https://cursor.com/docs/bugbot)) — shipping,
  team-shared, auto-learned convention memory.
- **Agent Skills is now an open standard** (Dec 2025) with ~30+ clients including
  Cursor, Copilot, Codex, Gemini CLI, JetBrains and Amp
  ([agentskills.io](https://agentskills.io/home)). Anthropic's own framing — skills
  package "company-, team-, and user-specific context into portable,
  version-controlled folders" — **is CRED's pitch, shipped and given away.**
  Building on `SKILL.md` is table stakes; owning it is impossible.
- **`github/spec-kit` has ~122k stars, MIT, GitHub-official** — it occupies the
  middle of CRED's loop for free.
- **Atlassian bought the outcome layer**: it
  [acquired DX in Sept 2025](https://www.atlassian.com/blog/announcements/atlassian-acquires-dx).
  DX is the vendor whose AI measurement framework is cited in §4 — meaning the most
  credible independent measurement player is now inside an incumbent. ⚠️ Also note:
  Atlassian's Teamwork Graph **forbids apps from both calling the API and providing
  a connector** ([developer docs](https://developer.atlassian.com/platform/teamwork-graph/)) —
  if CRED assumed it could enrich and query that graph simultaneously, that is
  prohibited today.

**4. Nobody pays for this, and the closest OSS analogue was just absorbed.**

- **Zero practitioners found reporting payment** for shared/org agent memory — no
  seat counts, no ROI posts, no procurement anecdotes. The thread asking outright,
  *"I don't really understand who is gonna pay them when we have RAG and MCP"*
  ([HN 46004834](https://news.ycombinator.com/item?id=46004834)), scored **1 point,
  zero replies**.
- **No vendor prices org memory as a SKU.** CodeRabbit "Learnings" is default-on and
  free. [Devin Knowledge](https://docs.devin.ai/product-guides/knowledge) is free.
  **Vendors monetize administration and enforcement, never the context itself.**
- **Continue.dev — 35k stars, org-shared rules/blocks Hub, the closest OSS analogue —
  [was acquired by Cursor](https://thenewstack.io/cursor-quietly-acquires-continue-an-open-source-alternative-to-github-copilot/)**
  (Jun 2026). Its final blog pivot was *away* from sharing context toward measuring
  outcomes ("Intervention Rates Are the New Build Times"). **They saw where the value
  was and ran out of runway** — which is simultaneously the strongest support for
  this document's recommendation and the clearest warning about it.
- **The single most damning datapoint:** the best public articulation of CRED's own
  problem — [*"Ask HN: Why do AI agents keep repeating mistakes your team already
  fixed?"*](https://news.ycombinator.com/item?id=47399209) — scored **2 points, zero
  replies, and the company's domain no longer resolves.** Meanwhile
  [AGENTS.md — Open format](https://news.ycombinator.com/item?id=44957443) drew
  **837 points and 382 comments.** Developers will read 380 comments about what to
  put in a text file and will not click a product that manages it for them.

**5. Files-in-git is winning, and the real competitor is a ~72-line file the agent
maintains itself.** Measured across 525 files via the GitHub API: **median 72 lines
(AGENTS.md), 95 (CLAUDE.md)**; only 3–4% exceed 500 lines. The practitioner
direction of travel is *shrinking* these files
([Ask HN, May 2026](https://news.ycombinator.com/item?id=48160604)): *"Now I only
put what amounts to a table of contents and some highlights"* / *"I personally
don't."* Letta pivoted to git-backed files
([Letta](https://www.letta.com/blog/our-next-phase/)); Zep killed its self-hosted
CE; ReMe and OpenViking are file-first; the consensus cross-tool fix is a **symlink
that "cannot drift."**

[*Prose Isn't Policy*](https://zernie.com/blog/stop-writing-claude-md-rules/)
(Jul 2026) caps the addressable surface: routing analysis puts **37% → hooks,
27% → custom lint, 20% → existing lint rules, and only 16% genuinely prose** — and a
direct prohibition stated with no competing information was ignored two-thirds of
the time. **~16% is the real TAM for curated prose knowledge.**

### 🚩 The felt pain is adherence and verification, not recall

Top exchange in the category's highest-traction thread
([HN 47491466](https://news.ycombinator.com/item?id=47491466), 225 pts):

> *"The problem I'm having with agents is **not the lack of a knowledge base. It's
> having agents follow them reliably.**"*
>
> *"**The bottleneck isn't what the agent knows, it's what the agent can verify**…
> Giving it a tool that returns ground truth works better. **No memory required, no
> drift over time.**"*

This matches the ETH finding exactly: agents *follow* instructions fine; the
information just doesn't change outcomes. "Make the model obey harder" is not the
fix. And memory products lose to no memory —
[a competitor audit of Mem0's own published numbers](https://blog.getzep.com/lies-damn-lies-statistics-is-mem0-really-sota-in-agent-memory/)
shows **"Mem0's own results show their system being outperformed by a simple
full-context baseline"** (~73% vs ~68%).

> *"At this point building a memory tool for your ai agents is like a rite of passage."*

### Context windows: the commoditizing force is the harness, not the window

**Bear side.** Anthropic charges **no long-context surcharge**, cache reads are ~10x
cheaper, and [Epoch AI](https://epoch.ai/data-insights/llm-inference-price-trends)
measures a **median 200x/year** price decline for fixed capability post-2024. In May
2025 Anthropic **deleted its embedding/vector/chunking pipeline from Claude Code and
replaced it with grep** — Boris Cherny: it "outperformed everything. By a lot."
Cursor, Windsurf, Cline and Devin followed. Vector-search-for-code was not killed by
big windows; it was killed by a model good enough to drive grep in a loop.
**If the pitch is "we manage the context window better," the competitor is an API
parameter that costs nothing.**

**Bull side (why the problem survives).** [NoLiMa](https://arxiv.org/abs/2502.05167):
**11 of 13 models drop below 50% of baseline at just 32K**; GPT-4o falls 99.3% →
69.7%. Gemini 3 Pro: 77% on MRCR@128K → **26.3% @1M**. And
[Anthropic's own context-engineering post](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
argues against its own headline spec: attention is an n² budget, and the prescribed
fixes are compaction, JIT retrieval and subagents. Their compaction default fires at
**150,000 tokens on a 1M window.**

### The graveyard

| Company | Status |
|---|---|
| **CodeSee** (closest analogue) | Shutdown Feb 2024 → GitKraken. boldstart/Uncork/Menlo/Salesforce backed |
| **Almanac** ($34M) | Dead Jan 2025 — *founders* killed it; their AI content tool grew faster |
| **Swimm** ($33.3M) | CEO May 2026: **"Swimm is becoming a service company… We own the mess."** Now sells COBOL modernization services |
| **Sourcegraph Cody** | Free/Pro/Enterprise Starter terminated Jul 2025; Amp spun out Dec 2025; no priced round since 2021 |
| **Stack Overflow** | **3,862 questions in Dec 2025, −78% YoY** vs >200k/mo peak. Prosus (paid $1.8B) cut valuation >half Jul 2026. "for Teams" renamed Stack Internal |
| **Continue.dev** | Acquired by Cursor, Jun 2026 |
| **Letta** ($10M) | Sunset its hosted memory platform; pivoted to Letta Code |
| **Pieces** | Dropped developer identity → "AI memory layer for modern work." Every HN submission scored 1–2 points |
| **CodeParrot** (YC) | Dead Jul 2025. Stated cause: **"competed with GitHub Copilot"** |

⚠️ **Tailwind Labs** reportedly laid off 75% of engineering Jan 2026 citing "the
brutal impact AI has had on our business" (headline-level, unverified) — if it
holds, it is the sharpest precedent for an OSS-knowledge thesis: great distribution,
a genuinely good paid product built on human-curated reference material, destroyed
when AI made the curated artifact generatable.

**Corrections to the graveyard — do not repeat these as failures:**

- **Augment Code is not dead; it is a live competitor in CRED's lane.** It launched
  **Cosmos** in June 2026 with a Context Engine and an **"Expert Registry" of
  reusable agents**, SOC 2 Type II, on-prem deployment, and Adobe and MongoDB as
  logos ([SiliconANGLE](https://siliconangle.com/2026/06/05/augment-code-launches-cosmos-bring-agentic-ai-software-development-teams/)).
- **Kapa.ai was not acquired** — operating and unacquired per Tracxn, PitchBook and
  Crunchbase, ~$3.7M raised. (The likely confusion is Docker, a published customer.)
- **Sourcegraph layoffs could not be substantiated** across two research passes.
  What is verified is the Cody self-serve shutdown (Jul 2025) and the Amp spinout
  (Dec 2025). Drop the layoffs claim.

**Crowding:** total disclosed funding across *all* dedicated memory startups ≈
**$50M** — less than half of LangChain's single Series B — across five projects with
23k+ stars. **Nobody discloses revenue.** And nobody successful still sells raw
memory: Letta → coding agents, Zep → compliance, Cognee → enterprise deployment.

### Economics and GTM headwinds

- AI companies see **50–60% gross margins vs 80–90% for SaaS**
  ([Bessemer](https://www.bvp.com/atlas/the-ai-pricing-and-monetization-playbook)).
  A founder to TechCrunch: *"Margins on all of the 'code gen' products are either
  neutral or negative. They're absolutely abysmal"* — variable costs within 10–15%
  across competitors means **no cost moat, only a price war.**
- **Windsurf is the clearest proof that distribution and valuation do not rescue
  negative gross margins here.** Margins were reported **"very negative"** — each
  customer interaction lost money — despite a $2.85B valuation (Feb 2025) and a
  reported $3B OpenAI offer, ending in a Google deal paying **$2.4B to
  shareholders** with the residual going to Cognition
  ([TechCrunch](https://techcrunch.com/2025/08/07/the-high-costs-and-thin-margins-threatening-ai-coding-startups/)).
- **Review capacity is already negative.** [METR's 2025 RCT](https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/) —
  16 experienced OSS developers on 246 real issues took **19% longer** with AI while
  believing they had been sped up ~20%. That perception/reality inversion is exactly
  how a second review queue gets approved and then quietly ignored.
- **Doc rot is intrinsic, and this is the floor not the ceiling.**
  [Tan, Wagner & Treude](https://arxiv.org/abs/2307.04291): **more than a quarter of
  the 1,000 most popular GitHub projects contained at least one outdated code
  reference.** That is a mechanically auto-detectable dangling reference in the
  world's most-watched repos; semantic curation of typed knowledge is strictly
  harder. Note the research field abandoned the curation-tool thesis before the
  market did — since 2024 the direction has been *generate from source on demand*,
  not *help humans curate better*.
- **Pricing is fleeing per-seat**: Augment "$100/month flat — no per-seat charge";
  Ellipsis "No per-seat fees"; Cognee "unlimited users." CRED's implied per-seat
  enterprise model sits on the axis the market is abandoning.
- **67% of enterprises pay for support and security guarantees, not features.**
- **Solo-founder compliance tax:** 79% of AI coding platforms lack public SOC 2
  Type II, forcing [90+ day verification cycles](https://www.augmentcode.com/tools/ai-coding-tools-soc2-compliance-enterprise-security-guide);
  SOC 2 Type II costs **$20–50k and 6–12 months** plus 200–400 internal hours.
  Cross-vendor governance — the most defensible wedge — is an *enterprise compliance*
  product, the least solo-founder-friendly GTM that exists.

### 🚩 Where the bear case is weakest — the honest steelman

**Facts outside the training set work, spectacularly.**
[Vercel's evals](https://vercel.com/blog/agents-md-outperforms-skills-in-our-agent-evals)
on Next.js 16 APIs absent from training data: baseline **53%** → Skills 53% →
Skills+explicit 79% → **AGENTS.md docs index 100%.** Also: agents **choose not to
invoke available skills 56% of the time** — passive context beats retrieval-on-demand
because it removes the "should I look this up?" decision. *(Caveats: single model
family, narrow API set, and Vercel concedes "different wordings produced dramatically
different results.")*

This reconciles with ETH's null result and gives the one clean line in the whole
document: **verifiable, non-obvious, outside-training-data FACTS pay off; behavioral
rules and narrative overviews do not.**

**Three genuinely unshipped gaps:** cross-machine shared learned memory (Anthropic's
own stated limitation), cross-vendor portability/governance (each vendor governs only
its own clients; GitHub's org instructions don't even reach IDEs), and
relevance-ranked retrieval at inference time instead of prepending. Cross-vendor
governance is the only position where incumbent strength becomes incumbent
*constraint* — vendors are structurally disincentivized because it de-anchors their
seat. But gaps 1 and 3 are natural extensions of existing architecture: **assume
under 12 months of daylight.**

**Capital is flowing to the verification end, not the context end:** Qodo $70M,
Cognition $1B at $25B, CodeRabbit $60M at $550M (~$40M ARR, ~700% YoY), Greptile
$25M. The money moved toward *"was the output correct."*

---

## Wedge candidates ranked

Scored 1–5, higher is better.

| Wedge | (a) Real pain | (b) Whiteness | (c) Defensibility | (d) TTFV solo | Total |
|---|---|---|---|---|---|
| **Harness evaluation & regression** | **4** | **4** | 3 | 4 | **15** |
| **Harness distribution** | **4** | 1 | 1 | **5** | 11 |
| **Cross-agent handoff** | 1 | 4 | 1 | 4 | 10 |
| **Governed org memory** | 2 | 1 | 2 | 2 | 7 |

**Harness evaluation & regression — 15 (recommended).** Note this is a *revision*
of CRED's "outcome-linked organizational learning," not an endorsement of it as
written. See the framing correction below.

Pain is the best-attested in the document (4): the ETH Zurich paper, Vercel's
rebuttal, and verbatim complaints — *"nobody knows if theirs actually works,"*
*"overly reliant on vibes,"* and an engineer manually reverting and resubmitting
prompts because *"it's not easy to write evals."* Whiteness is high but not
pristine (4): **0 of ~11 observability vendors, 0 of 2 platform APIs and 0 of 9
DevEx vendors** serve it — but mdarena and Tessl have both probed it.
Defensibility is moderate (3): the collector is not hard to copy, though
longitudinal data and cross-vendor coverage accumulate. Anthropic and Cursor each
hold half the join and neither has closed it, and this is a *complement* to them
rather than a substitute. TTFV is good (4): a replay harness over merged PRs is
buildable solo in weeks — mdarena proves it.

**Harness distribution — 11.** Best raw usage evidence (4): 5,708 reactions, 985k
npm downloads/month. But whiteness is 1 — Anthropic shipped the package manager,
Cursor and GitHub enforce org-wide, Tessl and Semgrep hold the commercial ground,
Sleuth is selling exactly "Define Your AGENTS.md Once. Distribute Everywhere," and
free OSS already works. Defensibility is 1: a symlink is the published consensus
fix and Anthropic can close the gap in one release. TTFV is excellent (5).
**This is a great feature and a terrible company.**

**Cross-agent handoff — 10.** Genuinely greenfield (4) with a real standards gap —
A2A is stateless, ACP is agent-bound, MCP is deprecating sampling. But pain is 1:
~200 npm downloads/month total, and formats carry no stability guarantees so every
converter is a maintenance tax. Greenfield because nobody wants it. **Drop it.**

**Governed org memory — 7.** Weakest overall. Pain at team scale is unproven (2);
the space is crowded with funded competitors — Glen, Memco, Almanac, Linzumi,
Oracle (1); Letta and Zep both retreated from exactly this architecture; and
time-to-value is long (2) because org memory is worthless until it has accumulated.

### Recommendation

**Build harness evaluation and regression prevention. Use rule-sync compilation as
the free acquisition surface. Drop cross-agent handoff. Defer org memory.**

Three framing corrections that the evidence forces, and which matter more than the
ranking:

**1. Sell evals, not attribution.** The market articulates this need as a
*pre-merge testing* problem — deterministic, replayable, "did pass rate go up?" —
not as *post-hoc analytics* attributing shipped code back to a config version.
Those are different products with different buyers. CRED's stated framing
("outcome-linked organizational learning") is the analytics one. **The evidence says
build the testing one.** Post-hoc attribution can come later, once the collector is
already deployed.

**2. Sell org blast radius, not individual curiosity.** Both products that probed
this space — mdarena (65 stars) and Tessl's eval pitch (42 HN points) — chose the
weak framing: *"is my markdown good?"* The strong framing is in the HN quote:
**"a bad edit to AGENTS.md can cause silent regressions for everyone in the
codebase."** That is a CI/governance problem with a named victim, a blast radius,
and a budget owner. It is also the one framing that is *inherently* org-scale,
which is where CRED wants to be. Concretely: a CI check that runs when someone
edits `AGENTS.md`/`CLAUDE.md`/a skill, replays a corpus of recent merged PRs
against old and new harness, and blocks or warns on regression. "Tests for your
agent config."

**3. Do not use CRED's vocabulary in market-facing copy.** HN Algolia returns
**zero** hits for "config version → outcome attribution" phrasings. Users say *"I
don't know if my CLAUDE.md works."* Ship in their words.

License Apache 2.0, single-command install, local-first, results visible without a
server. Give away rule-sync compilation because that is where the 985k
downloads/month already are — but never charge for it.

This still inverts CRED's current ordering, which lists outcome-linked learning
fourth. The evidence says it belongs first — but reframed as testing rather than
analytics.

**The honest burden of proof:** mdarena built a credible version of this, measured
+27%, and got 65 stars. Any plan here must answer *why succeed where mdarena
didn't* — and the only defensible answers are the org/CI framing and the
cross-vendor surface, not better technology.

**Watch Sleuth closely.** It owns governed config distribution (the write side) and
has abandoned DORA metrics (the read side). It is one product decision away from
being the natural incumbent for this exact wedge.

### The content correction — what the harness should actually contain

The evaluation wedge answers *how to sell*. This answers *what to put in the thing
being evaluated*, and it is the single highest-confidence prescription available:

**Ship verifiable facts outside the training set. Do not ship curated narrative
wisdom.** Vercel measured **53% → 100%** on exactly this class of content, and
**agents decline to invoke available skills 56% of the time**, so passive context
beats retrieval-on-demand. Concretely, the payload is: versioned internal API
signatures, service ownership, deployment topology, schema shapes, migration state,
build and test invocations — all **derivable from systems of record, auto-expiring,
and needing little human review.**

Everything ending in "reviewed organizational learning" — narrative lessons,
behavioral rules, curated wisdom — is precisely the part the evidence says will
bloat (4x ratchet), starve (ADRs at 1–5 records), and not change outcomes (ETH null
result). *Prose Isn't Policy* caps that surface at **~16%**; the other 84% belongs
in hooks and linters, where Semgrep Guardian already sits.

Three consequent design rules:

1. **Stop calling it memory.** Given ISR → 0 above 6,000 words, the correct
   architecture is a **compression and forgetting engine**, not an accumulator.
   "Organizational memory" markets the harmful property.
2. **Make expiry enforced and default-on.** GitHub Copilot Memory already ships
   28-day decay of unused entries. CRED's `validity` field must expire by default,
   not merely record a date.
3. **Size the review queue to ~16%**, not to everything teams currently write.

**And make the baseline spike a genuine kill gate.** PRD acceptance criterion #3
restates a hypothesis that [arXiv:2602.11988](https://arxiv.org/abs/2602.11988) has
already published a null result against. The paper's own recommendation — "any
attempts to improve performance should be rigorously evaluated before deployment" —
is a gift to a company selling evaluation. Take it literally, including the part
where you stop if it fails.

---

## Kill criteria

Stop or pivot if any of the following is observed.

**Immediate stop**

1. **Anthropic, Cursor or GitHub adds a config-version dimension to its analytics.**
   Each already holds half the join — Anthropic's Analytics API has
   `pull_requests_by_claude_code` but no CLAUDE.md dimension; GitHub has
   `total_merged` but no instructions-file dimension. **If either adds the missing
   dimension, the wedge closes overnight.** *Watch: Claude Code Analytics API
   changelog, GitHub Copilot usage metrics schema.*
2. **Sleuth connects Sleuth Skills to outcome data.** It owns governed config
   distribution and has already abandoned DORA metrics; joining them makes it the
   natural incumbent with an existing customer base. *Watch: sleuth.io product
   announcements.*
3. **PromptLayer joins Skill Collections to its scoring API.** Both primitives are
   already shipped and monetized; the join is a small feature for them and the
   whole product for CRED.
4. **A larger replication of [arXiv:2602.11988](https://arxiv.org/abs/2602.11988)
   confirms harness content does not affect outcomes at all.** If harness
   configuration provably does not move results, there is nothing worth testing and
   the premise dissolves. Note the current paper found +4% for human-written files —
   small but non-zero, which is what keeps the wedge alive.

**Pivot triggers**

5. **The 20-team validation fails.** If, in structured conversations with 20+
   engineering leaders running agents at team scale, fewer than 5 can name a
   specific incident where a harness change silently degraded output across the
   team — the org blast-radius framing is invented and only the weak individual
   framing remains, which mdarena already proved does not sell. **Run this before
   writing significant code.** Cheapest, highest-information test available.
6. **The mdarena conversation goes badly.** Contact the author. If the reason it
   got 65 stars was *lack of demand* rather than *lack of distribution or
   framing*, that is close to dispositive — it is the nearest thing to a natural
   experiment on this exact wedge.
7. **Nobody will run it on real repos.** If a working CI check cannot get 10 teams
   to run it against production work within 60 days of release, friction exceeds
   felt pain.
8. **Rules-file usage declines.** If AGENTS.md/CLAUDE.md adoption flattens or
   reverses through 2026–27 as models get better at inferring conventions from the
   repo itself, the substrate CRED tests is disappearing. *Watch: the 60k-repo
   figure, rulesync/ruler download trends, and whether the "delete your CLAUDE.md"
   position gains ground.*
9. **Replay proves too noisy to be a gate.** If harness A/B replay over merged PRs
   cannot produce a stable enough signal to block a PR without unacceptable false
   positives, the CI framing collapses into a dashboard — and dashboards in this
   category are a crowded, Gartner-MQ market CRED cannot win solo.
10. **GitHub scopes Copilot Memory to the org and adds an approval gate.** It
    already has repo-level shared facts, citations to supporting code, validation
    against the current codebase, and 28-day expiry
    ([docs](https://docs.github.com/en/copilot/concepts/agents/copilot-memory)).
    Two features from being CRED, inside the platform where the code lives.
11. **Google makes Gemini CLI Auto Memory org-scoped and on by default.** It has
    already shipped the review gate CRED claims as a differentiator; only
    experimental status, opt-in, and per-project scoping stand between it and
    parity.
12. **Facts-outside-training-data stops paying.** If a Vercel-style eval on
    genuinely novel internal APIs stops showing a large delta as models improve at
    reading source directly, the last evidence-backed content wedge closes and
    there is nothing left worth compiling.

**Do not treat as kill criteria**

- Low GitHub stars in the first six months. rulesync has 1.2k stars and 808k
  npm downloads/month; utility-tier infrastructure is systematically under-starred.
- Anthropic shipping more *distribution* or *package-management* features. That was
  never the recommended wedge — it is already lost.
- Competitors shipping org *memory*. The recommendation is deliberately not there.
- LinearB-style retreats from AI attribution. That reflects the unreliability of
  third-party seat-level APIs, which local replay does not depend on.

---

## Open research debt

### 🚩 Do not cite these — retracted as fabricated

A research thread fabricated the following after exhausting its search budget, then
self-retracted. **All are excluded from this document; do not reintroduce them:**
an "ICSA 2026 / 63% of ADRs bypass deliberation" paper (does not exist); GitHub
Open Source Survey "93%/60%" documentation figures; Ratol & Robillard; a
Storey & Barnett / Gartner "70% of KM projects fail" attribution; Fluri et al.
"97% of comment changes"; 2024 Stack Overflow Developer Survey percentages; and
Tan et al.'s "82.3% / 4.7 years / 55%" figures (the *headline* Tan finding cited in
§6 — "more than a quarter of the top 1,000 repos" — is from the real abstract and
is retained). A second thread flagged a **"SpaceX acquired Cursor for $60B"** claim
as likely fabricated — **do not use it, and treat all Cursor financials as
unverified.**

### Corrections applied

Kapa.ai was **not** acquired; Sourcegraph layoffs could **not** be substantiated
(the Cody self-serve shutdown and Amp spinout are verified); **Augment Code is a
live competitor, not a graveyard entry.**

### Remaining gaps

1. **Reddit, X and dev.to are entirely uncovered** — network-blocked throughout.
   This is where disillusionment content lives and where team-scale complaints would
   surface, since HN skews solo/small-team. It would most likely *strengthen* the
   negative case, but it is untested. **Treat "no org-scale pain found" as
   provisional.**
2. **Highest-value next actions, in order** — (a) contact the mdarena author;
   (b) run the 20-team validation on the org blast-radius framing; (c) close the
   Reddit gap; (d) run the ETH benchmark against CRED's own context packages
   *before* building the graph, harness registry or review workflow.
3. **Uncovered competitor cluster** — Tessl beyond its raise and repositioning,
   Tabnine, Grit.io, Moderne, Magic.dev, Sema4, Reflection.ai, and Poolside's
   reported Apr 2026 distress. **This is where the 2025–26 fire-sales concentrated
   and is the cluster most likely to hold another CodeSee-shaped precedent.** Treat
   as an open item, not as "nothing there." Also unresearched: Factory,
   Cognition/Devin, JetBrains, Greptile, Graphite, CodeRabbit; systematic YC W25→S26
   sweep (directory is JS-rendered).
4. **Blocked sources ⚠️** — GitClear returns HTTP 403 to all automated access, so
   its churn and copy-paste figures are deliberately **not** cited and need a manual
   browser visit. Stanford AI-rework studies, Google's internal AI metrics,
   Confluence/Notion decay data, and DORA 2025 instability deltas also unreached.
5. **Low confidence ⚠️, re-verify before any investor doc** — OpenAI Codex "Team
   Config" (docs 404); Unblocked's status (contradictory findings across threads);
   Atlassian Rovo adoption figures (zero verified); Prosus writedown figure; Qodo's
   $70M Series B (source 403'd); Tessl's valuation ($500M TechCrunch vs $750M
   Fortune) and registry size (3,000 vs 10,000 skills, irreconcilable across their
   own pages); Cursor Memories removal version; Tailwind Labs layoffs; Memco's
   benchmark claims (vendor-reported, unaudited); DORA adoption 90% (Google) vs 95%
   (Faros) — use 90%.

### Process note — read this before trusting any unmarked figure

The session hit its **200-call web-search cap within minutes** across five parallel
agents, pushing most verification onto direct fetches and RSS headlines rather than
article bodies. **Budget exhaustion is roughly where sourcing quality degraded and
where fabrication began.** Separately, a GitHub API rate-limit exhaustion produced
false "repo not found" results mid-research, which were caught and discarded rather
than written up as evidence of nonexistence.

The retracted items looked plausible enough to pass unexamined. Figures marked ✅
were pulled by me directly from the GitHub or npm APIs and are verified; **every
other figure should be spot-checked before it enters an investor or planning
document.** Re-run with `CLAUDE_CODE_MAX_WEB_SEARCHES_PER_SESSION` raised before
treating any of this as settled.
