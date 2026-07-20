# Demand, Buyers, and Pricing — Evidence Review for CRED

**Date of review:** 2026-07-20
**Scope:** Demand evidence, current practice, self-hosting demand, pricing benchmarks, buyer identity, OSS→revenue conversion, Vietnam/SEA angle.
**Premise accepted, not re-litigated:** the decision to compete head-on with Memco and Glen is taken as given. This document is about *how to build and sell*, not *whether to*.

**Method note / limitation.** The session WebSearch budget (200 calls) was exhausted partway through. Later research ran on direct WebFetch, the HN Algolia API, DuckDuckGo-via-fetch, and the GitHub code-search API. Consequences, stated plainly:
- **Reddit is entirely absent.** WebFetch refuses `reddit.com`; `.json` endpoints return an HTML shell; redlib mirrors 403'd. **Zero quotes from r/ExperiencedDevs, r/LocalLLaMA, r/ChatGPTCoding, r/cursor, r/ClaudeAI.** This is the single largest gap.
- **No conference talks** were discoverable without search.
- Several primary sources hard-blocked: Gartner (403), Cisco (403), IBM (403), McKinsey (timeout), Reuters (blocked), fpt.com (403).

---

## Verdict

- **The pain is real but narrower than the pitch.** *Single-developer, cross-session amnesia* and *instruction-file staleness* are abundantly and vividly documented. The *organizational, multi-person* dimension — CRED's actual thesis — is **thinly documented**. Only two genuine non-vendor quotes on team divergence were found. **VERIFIED asymmetry:** of ~30 HN hits for agent memory, roughly **20 were founders announcing their own memory product**. Many sellers, few organic buyers using those words. Treat this as the most important finding in the document.

- **The strongest disconfirming artifact is a benchmark, and it is bad news.** Memory layers (Mem0, Zep) measured **14–77× more expensive and 31–33% less accurate** at fact recall than simply passing full history to long context ([HN 46032522](https://news.ycombinator.com/item?id=46032522)). Unreplicated and single-model. **Correction, 2026-07-20:** this entry originally
called the benchmark "unanswered," implying it had survived scrutiny. It had not
been read — three HN submissions scored 4, 4, and 2 points, with one comment, by
the author. Its long-context arm averaged ~4,232 input tokens, and the cited
ratios do not reconcile with its own published table. The figures above are
retained as the claim that was made; see
[v0 experiment design](../spikes/v0-experiment-design.md) for the teardown. A visible 2026 counter-current argues the whole customization stack is last year's idea: *"Making a huge custom setup is so 2025."*

- **State of practice is committed markdown plus executable guardrails, not memory servers.** **VERIFIED via GitHub code search (2026-07-20): AGENTS.md = 154,496 files; CLAUDE.md = 51,100; .cursorrules = 7,208; copilot-instructions.md = 4,064.** AGENTS.md now outnumbers CLAUDE.md ~3:1 — the standard is consolidating. The most-reported *fix* practitioners describe is not memory but lint rules, types, tests, and hooks. **No major survey (Stack Overflow, DORA, Octoverse, JetBrains) asks how teams share agent context.** That number does not exist publicly.

- **Self-hosting demand is real, minority, and trending away from true self-hosting.** **28.1%** of developers report InfoSec rules blocking AI agent tools (Stack Overflow 2025) — a large minority, meaning ~72% are not blocked. But the market answered with **regional data residency, zero-retention agreements, and SOC 2/FedRAMP, not customer-hosting.** Two decisive counter-facts: **Samsung reversed its famous 2023 source-code-leak ban in June 2026 and deployed ChatGPT Enterprise + Codex cloud-wide**; and **Cursor claims 64% of the Fortune 500 with no on-prem product at all.**

- **Money exists and the seat price is anchored at $20–$40.** **VERIFIED: startups spend an average $1,040/developer/month (median $577) on code generation** — 4.2% of average payroll, framed by Scale VP as a **new budget category, ~50× the historical $20/mo dev-tool ARPU**. Memco prices at **$599/contributor/year (~$50/seat/mo)**. Durable enterprise gates across the whole comparable set: **SCIM, audit logs, self-hosting/VPC, SLA, dedicated support, IP indemnity** — note SSO has largely fallen *out* of the enterprise tier.

- **OSS conversion is ~1–1.3%, and open source buys reach, not revenue.** Grafana monetizes **1% of 20M users**; n8n ~3,000 paying of 230,000 active (**~1.3%**); PostHog's self-hosters were **3.5% of users and were deliberately shed as unprofitable**. What OSS demonstrably buys is enterprise *reachability without a sales team* — Langfuse reached **63 of the Fortune 500** on a seed-stage team. For Vietnam: **the legal-mandate argument for on-prem fails on inspection** — no SEA jurisdiction mandates in-country hosting of source code. The two defensible legs are Japanese client closed-network regimes and price sensitivity (a $30 seat costs a Vietnamese employer **6–11× more as a share of salary** than a US employer).

---

## 1. Is organizational agent memory a felt pain?

### 1a. Evidence the pain is real

**The single best articulation of the thesis** — note the conflict of interest, it is the Memco founder pitching in the same comment:

> "The thread keeps circling back to memory. Agents don't learn. Everyone's building the same workarounds. CLAUDE.md files. Handoff docs. Learnings folders. Developer logs. All manual. All single-user. All solving the same problem: how do I stop re-teaching the agent things it should already know? ... The 'junior developer that never grows' framing is right. But it's worse - it's a junior who forgets everything at 5pm and shows up tomorrow needing the same onboarding. **And there's no way for your junior's hard-won knowledge to help anyone else's.**"
> — st-msl, 2026-01-22, [HN 46716295](https://news.ycombinator.com/item?id=46716295) ⚠️ **VENDOR (Memco founder)**

**Concrete repetition cost, with the failure modes named precisely:**

> "Every few sessions I'd find myself typing the same thing again. 'Auth token expiry is 900s not 3600s.' ... After the fifth time I stopped blaming myself for not writing it down... CLAUDE.md is manual — you maintain it. MEMORY.md is automatic but it only gets read once at session startup, it's scoped per repo so your cross-project preferences don't carry over, and once it gets long enough only the top N lines are read. **Older notes just silently disappear from context regardless of how important they are. I had a note about a critical config value fall off the bottom and cost me an hour.**"
> — NBenkovich, 2026-03-31, [HN 47593179](https://news.ycombinator.com/item?id=47593179) ⚠️ **VENDOR (cmr-memory author)**

**Non-vendor voices — higher evidentiary value, and notably scarcer:**

> "And thankfully we have memory MCP servers, otherwise it would be like mentoring a brand new intern every time you fire up Claude."
> — port11, 2026-04-20, [HN 47831037](https://news.ycombinator.com/item?id=47831037) — **VERIFIED, non-vendor**

> "amnesia across agent sessions continues to be frustrating. It wastes a lot of time and tokens, and often sends agents down the wrong path or into parallel development circuits that obfuscate functionality and contribute to ai slop."
> — planetceres, 2026-06-12, [HN 48499895](https://news.ycombinator.com/item?id=48499895) ⚠️ OpenDream author

**Team-level divergence — the closest thing found to CRED's exact thesis. There are only two of these:**

> "What I'm deeply skeptical of is the ability for agentic to integrate with a team maintaining+shipping a critical offering. If you're using LLMs for one-off PRs, great but then agentic seems like a band aid for memory etc. Meanwhile **if you're full CC/agentic it seems like a team would get out of sync.**"
> — spopejoy, [HN 46707311](https://news.ycombinator.com/item?id=46707311) — **VERIFIED, non-vendor**

> "Parallel agents drift into conflicting assumptions (DB/auth/API conventions). Tomorrow's session repeats yesterday's context dump."
> — sukinai, 2026-02-09, [HN 46946290](https://news.ycombinator.com/item?id=46946290) ⚠️ Nemp Memory author

**Instruction-file drift and rot, explicitly named:**

> "But subsystem-specific AGENTS.md/CLAUDE.md files are still superior and accomplish the same thing. **The problem with those is they can become stale.**"
> — EMM_386, 2026-07-12, [HN 48884568](https://news.ycombinator.com/item?id=48884568)

> "If I ask an agent to maintain agent-facing docs like AGENTS.md then similarly it ends up with low information density... **I do find LLMs lack the theory of mind to realise that other LLM instances don't know the things that the current instance knows, and make poor calls on what to include and what not to include.**"
> — wren6991, 2026-07-14, [HN 48903978](https://news.ycombinator.com/item?id=48903978)

**A failure mode CRED must design against — auto-memory actively backfiring:**

> "Currently considering disabling memories in Claude code as well. It keeps writing a note whenever it struggles with something, but then on the next task, it reads that note and misunderstands when it applies, gets confused about its current task and write the most unreadable code. **Yesterday told it to write a memory to never write new memories when it solves a problem.**"
> — black_knight, 2026-07-15, [HN 48917688](https://news.ycombinator.com/item?id=48917688)

**Model upgrades invalidate your context files — an underrated durability risk:**

> "I used to have a CLAUDE.md rule that forbid inline comments... **That used to work perfectly back in 2025 era models. Now these models refuse to honor it** and write worse comment slop."
> — jarjoura, 2026-07-14, [HN 48914195](https://news.ycombinator.com/item?id=48914195)

### 1b. Disconfirming evidence — stronger than expected

**The benchmark. This is the most serious threat in the document.**

> "I benchmarked Mem0 and Zep on MemBench as memory layers for LLM agents, using gpt-5-nano on 4,000 conversational cases and comparing against a long-context baseline. In this setup, **the memory systems were 14–77× more expensive over a full conversation and 31–33% less accurate at recalling facts than just passing the full history.** The post shows the results and argues that the shared 'LLM-on-write' architecture ... is a bad fit for working memory / execution state."
> — cpluss, 2025-11-24, [HN 46032522](https://news.ycombinator.com/item?id=46032522), post titled **"Universal LLM Memory Does Not Exist"**
> ⚠️ **Caveat, stated honestly:** the blog body could not be fetched (curl 0 bytes, WebFetch 404 on variant URL). Numbers are verbatim from the author's own HN submission comment, which *was* read directly. The thread had essentially **no discussion** — only the author's comment. **Unreplicated, single-model, single-benchmark.** But nobody has publicly rebutted it either.

**The "vanilla" counter-current:**

> "I'm starting to come to the opposite approach: **don't try to customize anything, just use it vanilla, and use the best model you can afford. No AGENTS.md, no special subagents or roles**, nothing but a few convenience skills which are really just textexpander. Use the harness that the LLM provider makes, and that's it. **Making a huge custom setup is so 2025.**"
> — edot, 2026-07-17, [HN 48942927](https://news.ycombinator.com/item?id=48942927)

**Instruction files fundamentally don't work — which implies a better memory layer won't help either:**

> "Given my own experience futilely fighting with Claude/Codex/OpenCode to follow AGENTS.MD/CLAUDE.MD/etc with different techniques that each purport to solve the problem, **I think the better explanation really is that they just don't work reliably enough to depend on to enforce rules.**"
> — toraway, 2026-02-26, [HN 47173315](https://news.ycombinator.com/item?id=47173315)

> "You can iterate, add more context to AGENTS.md or CLAUDE.md, add skills, setup hooks, and **no matter how many times you do it the agents will still make mistakes**... So they think maybe there are magical context tricks... **There aren't.**"
> — post_below, 2026-02-15, [HN 47022951](https://news.ycombinator.com/item?id=47022951)

**Long context already solved it for some users:**

> "This has not been my experience with Opus since Anthropic released the 1M token context window... I routinely push past 500k tokens, even sometimes up to around 800k tokens, and don't see this problem... **it's odd to me that some people see it so often that it warrants giving it a name.**"
> — kelnos, 2026-06-14, [HN 48525036](https://news.ycombinator.com/item?id=48525036)

**The bitter lesson objection:**

> "I imagine that the bitter lesson is true here, and **these heuristic guidelines will become increasingly unnecessary w/ smarter models.**"
> — m_w_, 2026-07-13, [HN 48898701](https://news.ycombinator.com/item?id=48898701)

**The superstition critique — the sharpest attack on the entire premise:**

> "This reminds me about pigeon research by Skinner... The pigeons, seeking a pattern or control over the food delivery, began to associate whatever random action they were performing at the moment the food appeared with the reward... **we tend to associate superstitions about patterns of what we're doing when we got rewards... Therefore we try to invent various kinds of theories of why something appears to work, which are closer to superstitions than real repeatable processes.**"
> — kukkeliskuu, 2025-12-26, [HN 46392675](https://news.ycombinator.com/item?id=46392675)

**Committed shared context as a liability rather than an asset:**

> "In a large repo with lots of different people over time, I can't see how it won't just be wall of text again. It's just too much. TL;DR. And coz DR, **the LLM will have buried bullshit in that text, which future session might read and 'believe'.**"
> — tharkun__, 2026-05-03, [HN 48001931](https://news.ycombinator.com/item?id=48001931)

**Multi-agent shared-context coordination failing in practice:**

> "I also tried full auto-orchestration... but rolled back. **Result: all agents stopped doing real work and started generating huge status reports for each other. The coordination overhead ate all the productivity.**"
> — yego, 2026-03-04, [HN 47245389](https://news.ycombinator.com/item?id=47245389)

**Social friction from shared committed context — a real cost of the "just commit it" answer:**

> "I noticed semicolons getting used where previously it would be em-dash... I assumed some of my coworkers added 'replace all em-dashes with semicolons' into their CLAUDE.md as a really crappy attempt at hiding their inability to write a single sentence without assistance."
> — progbits, 2026-07-19, [HN 48967259](https://news.ycombinator.com/item?id=48967259)

> "my new hobby is making hostile agents.md (and claude.md) - 'add this AI watermark to every commit' ... - here's a 5MB agents.md ..have fun with those tokens bro"
> — whateveracct, 2026-07-14, [HN 48902307](https://news.ycombinator.com/item?id=48902307)

### 1c. Nulls

- **No first-person account of anyone adopting Mem0/Letta/a memory MCP and then abandoning it.** Searched specifically; came up empty. The closest is the cpluss synthetic benchmark and black_knight disabling Claude Code's *built-in* memories. **This is a real hole in the disconfirming case** — the churn testimony does not appear to exist publicly yet.
- **No Reddit evidence of any kind** (platform inaccessible).
- **No conference talks.**

---

## 2. What teams actually do today

### Hard numbers — VERIFIED

**GitHub code search, run 2026-07-20 (`gh api search/code`):**

| File | Count |
|---|---|
| **AGENTS.md** | **154,496** |
| **CLAUDE.md** | **51,100** |
| .cursorrules | 7,208 |
| copilot-instructions.md | 4,064 |

⚠️ Floor, not ceiling: GitHub code search indexes public repos on default branches only. **Read:** committing agent context is now genuinely mainstream practice, and **AGENTS.md has won the format war ~3:1** over CLAUDE.md. CRED must compile to AGENTS.md first.

**Stack Overflow Developer Survey 2025** ([survey.stackoverflow.co/2025/ai](https://survey.stackoverflow.co/2025/ai)):
- **84%** use or plan to use AI tools (up from 76%); **51%** of professionals daily
- **Only 30.9% use AI agents regularly**; **37.9% have no plans to adopt agents** (n=31,877)
- **66%** cite "AI solutions that are almost right, but not quite" as top frustration
- **Only 3% highly trust** AI accuracy; **46% actively distrust**; favorable sentiment **dropped to 60%** from 70%+

⚠️ **The 30.9% agent-adoption figure materially shrinks CRED's addressable base.** CRED sells to teams running *agents*, not teams using autocomplete.

**DORA 2025** ([Google Cloud](https://cloud.google.com/blog/products/ai-machine-learning/announcing-the-2025-dora-report), n≈5,000):
- **90%** use AI at work; **30%** report little/no trust in AI-generated code
- **90%** of orgs have adopted at least one internal platform, with a **direct correlation between high-quality internal platforms and ability to unlock AI value**
- Framing: **"AI doesn't fix a team; it amplifies what's already there"**

This is the closest institutional support for CRED's thesis in any major survey — but note it says **"platforms," not "agent memory."** Do not overclaim it.

**GitHub Octoverse 2025** ([github.blog](https://github.blog/news-insights/octoverse/octoverse-a-new-developer-joins-github-every-second-as-ai-leads-typescript-to-1/)): 180M+ developers; ~**80%** of new developers use Copilot in week one; GitHub's coding agent generated **1M+ PRs** May–Sept 2025.

**JetBrains State of Developer Ecosystem 2025** (n=24,534): **71%** use at least one AI coding assistant; **78%** use AI for coding regularly. Most relevant line: **"Often, AI use isn't systematized. Developers are just using it ad hoc."**

### Qualitative practice

The dominant 2026 pattern is **committed markdown plus hand-rolled docs directories** — not memory servers:

> "What .editorconfig has in common with CLAUDE.md and similar files, is that they should be shared among the team... **the shared best-practises and, more importantly, learnings, can be comitted and pushed.**"
> — moritzwarhier, [HN 48912640](https://news.ycombinator.com/item?id=48912640)

> "You use a shared agents.md and an auto updated architecture doc but that is the one that needs to be heavily scrutinized and **everyone gets a turn to review it.**"
> — m3kw9, [HN 48910490](https://news.ycombinator.com/item?id=48910490)

**The most-reported actual fix is executable guardrails, not memory** — this is CRED's real competition:

> "For a year+ I've been gaining leverage by codifying guardrails. Insanely intricate (and fun to create) lint scripts... **It's fun codifying 'how we work around here'** and it's been great for keeping dumb AI mistakes off my radar."
> — cadamsdotcom, [HN 48904240](https://news.ycombinator.com/item?id=48904240)

> "**So I made it write a linter and now it uses the linter constantly which keeps it closer to on the rails.**"
> — furyofantares, [HN 46699490](https://news.ycombinator.com/item?id=46699490)

### Null — and it is a significant one

**No survey anywhere asks how teams share agent context or configuration.** Stack Overflow 2025, DORA 2025, Octoverse 2025, and JetBrains 2025 all measure *adoption* and *trust*; none measure CLAUDE.md/AGENTS.md practice. If CRED needs that number for a deck, **it does not exist publicly and would have to be generated** — which is also an opportunity (an original CRED survey would be genuinely novel content marketing).

---

## 3. Self-hosting demand

### Evidence for — VERIFIED

- **Stack Overflow 2025:** **28.1%** somewhat/strongly agreed that *"My company's IT and/or InfoSec teams have strict rules that do not allow me to use AI agent tools or platforms."* ~26% report company restrictions on broader AI tool use. **This is the single best hard number available.**
- **IBM Cost of a Data Breach 2025** (via [Nudge Security](https://www.nudgesecurity.com/post/shadow-ai-the-emerging-security-threat-in-ibms-2025-cost-of-a-data-breach-report); IBM's own page 403'd): shadow AI implicated in **20%** of breaches, **+$670,000** average cost; **97%** of AI-related breaches lacked proper AI access controls; **63%** of breached orgs have no AI governance policy.
- **Netskope 2025:** **47%** of genAI platform users go through personal accounts; ~223 genAI data-policy violations per org per month. **Source code is a top violation category — 62%** of genAI data-exposure incidents in Asia and **28%** in manufacturing are source code.
- **EU sovereignty is funded, not rhetorical:** the Commission's Cloud Sovereignty Framework defines measurable criteria, and in **April 2026 awarded a €180M sovereign cloud tender to four European providers** over six years.
- **Langfuse markets residency explicitly** ([langfuse.com/self-hosting](https://langfuse.com/self-hosting)): "you control data residency and can deploy in any region," "internet access is optional," ClickHouse BYOC/Private for air-gapped.

### Counter-evidence — substantial, and it damages the thesis

Ranked by strength:

1. **Samsung reversed the canonical ban.** Samsung banned generative AI in 2023 after engineers pasted source code into ChatGPT ([Forbes](https://www.forbes.com/sites/siladityaray/2023/05/02/samsung-bans-chatgpt-and-other-chatbots-for-employees-after-sensitive-code-leak/)). In **June 2026 it fully reversed, deploying ChatGPT Enterprise and Codex company-wide** ([OpenAI](https://openai.com/index/samsung-electronics-chatgpt-codex-deployment/)). **The most-cited "source code leaked, therefore self-host" story now ends in governed cloud.**
2. **Cursor: 64% of the Fortune 500, 50,000+ enterprises, 100M+ lines of enterprise code/day — with explicitly no on-premises option** ([cursor.com/enterprise](https://cursor.com/enterprise)). SOC 2 Type II on AWS plus privacy mode and zero-retention agreements is clearing enterprise procurement at massive scale.
3. **GitHub Copilot shipped regional data residency instead of self-hosting** ([docs](https://docs.github.com/en/enterprise-cloud@latest/admin/data-residency/github-copilot-with-data-residency)) — US and EU live, +10% AI credits, requires Enterprise Cloud, **explicitly no on-prem**. The market's answer to sovereignty has been *regional cloud*.
4. **Developers pick cloud models even when local is free:** Ollama **15.3%** vs OpenAI GPT **81.4%**, Claude Sonnet 42.8% (Stack Overflow 2025).
5. **Restriction is a minority position:** 28.1% blocked means **~72% are not.**
6. **Bans produce shadow AI, not self-hosted procurement:** 47% use personal accounts. Blocked employees route *around* policy; they do not buy an on-prem alternative.
7. **EU AI Act teeth were just pushed out.** The Digital Omnibus delay got **final Council green light 29 June 2026**: high-risk obligations slip from 2 Aug 2026 to **2 December 2027** (stand-alone) and **2 August 2028** (embedded). **The near-term regulatory forcing function is gone.**
8. **Even where self-host exists, conversion is ~1–3%** (see §6).
9. **PostHog actively shed self-hosters as unprofitable** (see §6) — a direct operator datapoint on the burden.

### Nulls

- **No survey gives a clean "X% of enterprises ban cloud AI *coding assistants*."** The 28.1% figure is about *agent* tools specifically.
- **No third-party adoption numbers for on-prem AI coding tools in finance, healthcare, defense, or government.** An entire cottage industry of air-gapped AI coding vendors exists (OnPremize, Jinba, GAIA Labs, TrueFoundry), but **every source asserting this demand is a vendor selling into it.** Treat the air-gapped segment as **unquantified**.
- **No vendor publishes an actual self-hosted-vs-cloud user split.** Best available proxy: Langfuse's **6M Docker pulls vs 26M SDK installs/month** ≈ 23% — but these count different things. **INFERRED, weak.**
- **Could not find a credible founder quote saying "everyone asks for self-hosted, nobody pays for it."** HN mining 2025–2026 returned nothing on point. Plausible but **unsupported** — do not cite it as if it exists.
- **No source ties EU AI Act obligations specifically to developer tooling.** Residency pressure runs through GDPR and procurement, not the AI Act.

### Honest read

The defensible version is narrower than "orgs demand self-hosted": a **genuine but unquantified air-gapped segment** (defense, some banks, EU sovereign public sector) plus a much larger **data-residency segment that regional cloud already serves**. The second is not a self-hosting market. **Self-hosting should be positioned as a wedge into a specific segment and as an OSS-distribution mechanism — not as the central demand thesis.**

---

## 4. Pricing benchmark table

All rows **VERIFIED** by direct page fetch on 2026-07-20 unless marked.

| Vendor | Public price | Annual discount | Enterprise = contact sales? | What Enterprise gates |
|---|---|---|---|---|
| **Memco** (direct competitor) | Free (public network) · **Team $599/contributor/year ≈ $50/seat/mo** · Enterprise custom | n/a | Yes | SOC 2, **on-premises**, GDPR readiness, VPC, code residency |
| **Glen** (direct competitor) | **null — no public pricing.** YC **Summer 2026**, team size **1** | — | — | — |
| **Langfuse Cloud** | Hobby $0 · Core $29/mo · Pro $199/mo · **Enterprise $2,499/mo (published)** | Startups 50% yr 1 | **No — price published** | SCIM, audit logs, custom rate limits, uptime+support SLA, dedicated engineer, HIPAA, PrivateLink. **SSO + project RBAC sold as a +$300/mo "Teams add-on" on Pro** |
| **Langfuse self-hosted** | **OSS free (MIT), unlimited usage, incl. SSO + org-level RBAC** · Enterprise custom | null | Yes | Project-level RBAC, server-side data masking, retention mgmt, audit logs, SCIM, Admin API, SOC2/ISO reports, dedicated support + SLA |
| **Onyx** | Business **$20/user/mo** (annual) · Enterprise custom | implied | Yes | OIDC/SAML SSO, **on-prem + region-specific deployment**, white-labeling, data exports, SLA. *RBAC is in Business* |
| **Mem0** | Hobby $0 · Starter $19/mo · Pro $249/mo · Enterprise custom | null | Yes | SSO, **on-prem deployment**, audit logs, SLA support, unlimited projects |
| **Zep** | Free $0 · Flex $104/mo ($1,250/yr) · Flex Plus $312/mo ($3,750/yr) · Enterprise custom | **17%** | Yes | Guaranteed rate limits + SLA, **BYOK + BYOC**, SOC 2 Type II, HIPAA BAA, 1-yr log retention, dedicated AM |
| **Cursor** | Hobby $0 · Pro $20/mo · **Teams $40/user/mo** · Enterprise custom | null | Yes | SCIM, audit logs, service accounts, pooled usage, repo/model/MCP access controls, AI-code-tracking API. **SAML/OIDC SSO is in Teams.** **No on-prem/VPC — AWS only** |
| **GitHub Copilot** | Free · Pro $10 · Pro+ $39 · Max $100 · **Business $19/seat/mo** · **Enterprise $39/seat/mo** | null | **No — published** | Business: policy mgmt, IP indemnity, pooled credits. Enterprise: SAML SSO, org-wide codebase indexing, custom model fine-tuning |
| **Tabnine** | **Code Assistant $39/user/mo · Agentic $59/user/mo** (annual only) | annual required | Partly | **Unusual — enterprise features are in the base price:** SaaS/VPC/on-prem/**air-gapped**, SSO, zero code retention, GDPR/SOC2/ISO27001, audit logs, IP indemnification. LLM tokens billed at provider cost **+5%** |
| **Sourcegraph** | Enterprise **starts at $16,000** | null | Yes | Single-tenant cloud + **self-hosted**, 24×5 support, audit logs, zero retention, **uncapped IP indemnity**, Context Filters |
| **Amp** | Megawatt $20/mo · Gigawatt $200/mo · PAYG | null | Yes | BYOK inference, SSO + directory sync, zero data retention, enterprise controls |
| **Devin / Windsurf (Cognition)** | Free · Pro $20/mo · Max $200/mo · **Teams $80/mo base + $40/seat** · Enterprise custom | null | Yes | SAML/OIDC SSO, centralized admin, **dedicated/VPC deployment**, priority support |
| **Augment Code** | **Business $100/mo flat** (not per seat, ≤50 seats, incl. $100 usage) · Enterprise custom | null | Yes | Only unlimited seats/concurrency, multi-region compute, dedicated support. **SSO, SOC2 Type II, CMEK, SIEM, data residency, audit trails are all already in Business** |
| **Qodo** | Pro Team $30/mo base (≤30 users), credits $0.012 · Enterprise custom (30+) | annual | Yes | SSO/SAML, audit logs, BYOK, **single-tenant or on-prem**, SLA, dedicated CSM |
| **Graphite** | Hobby $0 · Starter $20/user/mo · **Team $40/user/mo** · Enterprise custom | **20%** | Yes | SAML/SSO, audit log (SIEM), ACLs, GH Enterprise Server, SLA, custom MSA |
| **CodeRabbit** | Free · Pro $24/dev/mo · Pro Plus $48/dev/mo (annual) · Enterprise custom | quoted annually | Yes | Custom RBAC, SSO, audit logging, **self-hosting**, multi-org, SLA, EU SaaS |
| **Unblocked** | Code Review $19/user/mo annual · Platform $29 annual · Enterprise custom | ~17% | Yes | SSO, Data Shield permissions, **on-prem**, GH Enterprise/Jira DC, dedicated CSM |
| **Swimm** | **null — no public pricing**; priced on "lines of code you want to understand" | null | Yes, entirely | Not disclosed |
| **Pieces for Developers** | **null** — pricing renders client-side, unretrievable across two attempts | null | unknown | null |

### Patterns

- **$20 / $40 is the anchor.** Cursor Teams, Graphite Team, Onyx Business, Devin per-seat all land there. **That is the PLG land price and CRED should not fight it.** Memco's ~$50/seat/mo sits just above.
- **Enterprise-gating is moving *down-market* and SSO has largely fallen out of it.** Cursor puts SAML SSO in $40 Teams. Tabnine and Augment put SSO, audit logs, SOC 2, and data residency in their *base* paid tier. **Langfuse gives SSO and org-level RBAC away free even in self-hosted OSS.** Gating SSO in 2026 reads as rent-seeking (see [sso.tax](https://sso.tax/): Railway $20→$2,000, Mixpanel $20→$833, GitHub $4→$21).
- **The durable Enterprise-only gates are: SCIM, audit logs, self-hosting/VPC, retention policy, SLA, dedicated support, IP indemnity.** That is the open-core line to draw.
- **Seat pricing is eroding.** Augment ($100 flat), Amp, Mem0, Zep, Qodo are usage/credit-based; Devin is hybrid base+seat.
- **Self-hosting is the sharpest available differentiator** — Tabnine (air-gapped), Qodo, Unblocked, CodeRabbit, Onyx, Sourcegraph offer it; **Cursor explicitly does not.** That is a real wedge, but see §3 on how narrow the segment is.
- **Langfuse is the only comparable that publishes an Enterprise price** ($2,499/mo cloud) — useful as a public anchor when everyone else hides it.

---

## 5. Who is the buyer?

### The one strong datapoint — VERIFIED

[Scale Venture Partners, "How much startups are actually spending on code generation"](https://www.scalevp.com/blog/how-much-startups-are-actually-spending-on-code-generation) (2026-05-26, n=38 companies, trailing 6 months):
- **Average $1,040/developer/month ($12,480/yr); median $577/mo ($6,924/yr)**
- **4.2% of average payroll; 2.3% of median payroll**
- Spend split roughly evenly between **Claude and Cursor**; Codex material; **Copilot and Gemini rare**
- Scale frames this as **~50× ARPU increase** over the historical ~$20/mo dev-tool price point, and argues it is **a new budget category, not a reallocation**
- Heavy right skew (avg ≫ median)

**This is the best evidence that budget exists and is not zero-sum against existing dev tools.**

### Supporting — VERIFIED

- [a16z enterprise AI survey](https://a16z.com/ai-enterprise-2025/) (n=100 CIOs, May 2025): **innovation budgets fell from 25% of LLM spend to 7%** — AI has moved into centralized IT and business-unit budgets, i.e. core spend, not experimental. **~75% expected budget growth**. One CIO: *"what I spent in 2023 I now spend in a week."*
- [Puppet 2026 State of DevOps, Platform Engineering Edition](https://www.puppet.com/resources/2026-state-of-platform-engineering): **66%** apply AI in infrastructure workflows; **44%** achieve fully autonomous ops *with* standardized IDPs vs **31%** overall; **79%** of platform-mature orgs report mature governance vs **14%** of immature. **Establishes the platform team as the governance locus** — the most likely CRED champion.
- [Bessemer, "state of working with AI"](https://www.bvp.com/atlas/data-trends-state-of-working-with-ai) (May 2026, n=173 leaders / 113 companies): **77% use Claude Code, 50% Cursor**; **49%** delivering more without headcount.
- [Bessemer dev tooling roadmap](https://www.bvp.com/atlas/roadmap-developer-tooling-for-software-3-0): GitHub hit **$2B revenue in 2024 with Copilot driving >40% of growth**.

### Nulls — state these plainly

- **No survey found decomposing budget ownership by role** (eng manager vs platform/DevEx vs CTO vs CISO). a16z's CIO survey does not break it down. **This question is unanswered.**
- **No median ACV benchmark for dev infra**, and no PLG-land-size vs enterprise-contract-size data. Bessemer Cloud 100 Benchmarks and Scale's GTM Benchmark Tool likely hold it; not fetchable.
- **No data on enterprise dev-tool sales cycle length** or security-review gate duration.
- **Gartner's "80% of software engineering orgs will have platform teams by 2026" could NOT be verified** — gartner.com returned 403 on three URLs. **Do not cite it from memory.**

### INFERRED (flagged, not sourced)

At $577–$1,040/dev/month, a 200-developer org spends **$1.4M–$2.5M/yr** on AI coding tools — well past the threshold where procurement and security review engage. **But there is no direct evidence on that gate.** The combination of the Puppet platform-governance data and the DORA "internal platform quality determines AI value" finding makes **the platform/DevEx team the most likely champion and the eng leader the most likely signer**, with security as a gate rather than a buyer. **This is reasoning, not evidence.**

---

## 6. OSS → revenue conversion

### The benchmark you want does not exist — VERIFIED null

- **Runa Capital ROSS Index** ([runacap.com/ross-index](https://runacap.com/ross-index/)) measures **only GitHub star growth**. Zero revenue or conversion data. Anyone citing ROSS as monetization evidence is citing star counts.
- **Linux Foundation / Serena "State of Commercial Open Source 2025"** (via [LWN](https://lwn.net/Articles/1034944/)): $26.4B COSS funding in 2024 — but **85% was AI companies and 78% from just two firms**. No conversion data.
- **Heavybit / OSS Capital / Bessemer / Accel / Battery** — all null. The [COSS Conversion Playbook](https://chinstrap.community/coss-conversion-playbook/) is explicitly anti-benchmark.

**The absence is the finding:** the flagship index measures stars, the flagship report measures funding. Neither measures conversion. ⚠️ Discard financialmodelslab.com "OSS KPI benchmarks" — AI-generated, internally inconsistent ("180% trial-to-paid").

### The real number: ~1–1.3%

- **Grafana:** $270M ARR (June 2024), 69% YoY, 20M users — **"only monetizes 1% of its 20M users"** ([Sacra](https://sacra.com/chat/h/0f295770-caee-4cc2-8769-aab4196d31d2/)). Sacra's framing is directly on-thesis: a **"shallow funnel"** where **"most adoption proves product relevance, but not pricing power,"** and the paid layers (hosting, RBAC, SSO, reporting, support) are **"copyable layers rather than defensible moats."**
- **n8n:** ~$40M ARR, 3,000+ enterprise customers vs 230,000+ active users ≈ **1.3%**; **~$13,300 average revenue per customer**; revenue mix **~55% cloud, ~30% enterprise licenses, ~15% embedded/OEM** ([Sacra](https://sacra.com/c/n8n/)). [Series C](https://blog.n8n.io/series-c/): $180M at $2.5B, **6× user growth and 10× revenue growth in one year**, 162,000 stars.
- **PostHog:** when sunsetting Kubernetes self-hosting, **~3.5% of users** were affected — and they were **shed deliberately** ([blog](https://posthog.com/blog/sunsetting-helm-support-posthog)): *"our small infrastructure team is spending an outsized amount of time supporting the 3.5% of users"*; *"even something as simple as a full disk would cause their instance of PostHog to be down for hours or days"*; *"the tools to do that automation just don't exist. We kept finding new failure modes."* PostHog reached ~$50M ARR (Oct 2025) targeting **$100M by end of 2026 — while shedding self-hosters.**

### What OSS actually buys: enterprise reach, not revenue

**Langfuse is the strongest proof point and the cleanest template for CRED.** At [ClickHouse acquisition](https://clickhouse.com/blog/clickhouse-acquires-langfuse-open-source-llm-observability) (Jan 16 2026): **20,000+ stars, 23–26M SDK installs/month, 6M Docker pulls, 19 of the Fortune 50, 63 of the Fortune 500** (Intuit, Twilio, 7-Eleven, Merck). Founder ([PostHog spotlight](https://posthog.com/spotlight/startup-langfuse)):

> "Open source is a great way to get more enterprise adoption."

> "We wouldn't have expected to reach large Fortune 500 companies without a dedicated sales team, but we've since realized that **teams in those companies frequently adopt open source tools because they give them more visibility and control over data.**"

**No ARR was ever disclosed** — a notable null given the acquisition.

**Supabase** ([Sacra](https://sacra.com/c/supabase/)): ~$101M ARR end-2025 → **$170M ARR May 2026**; 1M+ databases, 2,500 new/day, **60%+ of new databases created by AI tools**. Critically, **the growth driver is not self-host conversion** — it is AI codegen tools auto-provisioning Supabase *Cloud*. **Open source bought the default-choice position; the cloud captured it.**

### The gating line — consistent across every open-core devinfra company

| Company | Free/OSS includes | Paid gate |
|---|---|---|
| **Langfuse** | MIT, all core features, unlimited usage, **SSO + org RBAC free** | Data masking, project RBAC, retention, SCIM, audit logs, Admin API, SOC2/ISO, SLA |
| **n8n** | Self-host free (Sustainable Use License) | SSO/SAML/LDAP (Business €667/mo); Enterprise: audit logging, log streaming, external secrets, SLA |
| **Metabase** | Unlimited users, questions, dashboards, AI SQL | Pro $517.50/mo: row/column permissions, SSO, audit, whitelabel. Enterprise ~$20k/yr min: success engineer, 1-day SLA, **air gap** |

**Pattern: the paywall is almost never the core product.** It is **identity (SSO/SAML/SCIM), authorization (fine-grained RBAC), auditability (audit logs, retention), compliance artifacts (SOC2/ISO), and support SLA.**

### Cautionary tales — take these seriously

**Cal.com went closed source, April 2026 — a brand-new argument category.** ([cal.com/blog/cal-com-goes-closed-source-why](https://cal.com/blog/cal-com-goes-closed-source-why)):

> "After five years as open source champions, Cal.com is going closed source."
> "AI can be pointed at an open source codebase and systematically scan it for vulnerabilities."
> "Being open source is increasingly like **giving attackers the blueprints to the vault.**" — CEO Bailey Pumfleet

Context: [CVE-2026-23478](https://www.implicator.ai/cal-com-goes-private-as-self-hosted-calendly-choices-narrow-in-2026/), a critical auth flaw. Cal.com was at **$7M ARR (March 2026)**, up from $6M six months earlier — note the deceleration from 3×/yr. **This is not the free-rider argument; it is "AI makes open source a security liability." Expect more of it, and expect CRED to be asked about it.**

**CockroachDB → fully proprietary (Aug 2024).** Spencer Kimball ([TechCrunch](https://techcrunch.com/2024/08/15/cockroach-labs-shakes-up-its-licensing-to-force-bigger-companies-to-pay/)):

> "**Our 'core' [free] offering has become one of our savviest competitors.**"

Mechanism: as product quality improved, support became less necessary, so enterprises had less reason to buy. **Better open source → worse conversion.**

**RethinkDB** ([post-mortem](https://gist.github.com/ramalho/93b87e961b6e019be8e1f6f82864b6f9)):

> "Developers love building developer tools, often for free. So while there is massive demand, **the supply vastly outstrips it.**"
> Thousands used it commercially, yet **"most were willing to pay less for the lifetime of usage than the price of a single Starbucks coffee."**

*Honest caveat:* RethinkDB also failed on product-market fit. Not a pure OSS indictment.

**License restriction is not the fix — both major SSPL experiments reversed.** Redis (BSD→SSPL 2024→**AGPLv3 May 2025**): *"This achieved our goal—AWS and Google now maintain their own fork—but the change hurt our relationship with the Redis community. SSPL is not truly open source."* Elastic (SSPL 2021→AGPL 2024): Banon notes three years later *"Amazon is fully invested in their fork"* — SSPL permanently ceded OpenSearch. **Sentry's FSL** is the better-designed alternative ([blog](https://blog.sentry.io/introducing-the-functional-source-license-freedom-without-free-riding/)): 2-year reversion to Apache/MIT, standardized, no custom grants — explicitly because BSL had *"too many parameters"* making each implementation a bespoke legal review.

**Skeptical takes worth internalizing.** Matt Asay, ["Open source isn't going to save AI"](https://www.infoworld.com/article/3548263/open-source-isnt-going-to-save-ai.html):

> "**Open source may enable big markets, but it doesn't enable widespread spoils from those markets.**"
> "The answer isn't going to be 'lots of open source vendors,' because, by definition, **that would simply exacerbate the complexity that customers want removed.**"

His mechanism is underrated: **OSS abundance increases the value of the consolidator**, structurally transferring margin to whoever reduces complexity — never to the project author. (Note affiliation: VP DevRel at MongoDB.) Peter Levine's ["Why There Will Never Be Another RedHat"](https://stanfordreview.org/why-there-will-never-be-another-redhat-the-economics-of-open-source/) makes the same case — **but date it accurately as 2014, not recent.**

**AI-era warning — distribution priced far ahead of conversion.** LangChain: **$12–16M ARR (June 2025)** at a **$1.25B valuation** (~90× ARR), not profitable. Ollama, vLLM, Flowise: **massive adoption, zero public revenue.**

---

## 7. Vietnam / SEA angle

### The legal-mandate argument FAILS — most important finding here

**Correction first: Decree 13/2023 is no longer operative.** VERIFIED — replaced by **Law No. 91/2025/QH15** (passed 2025-06-26, **effective 2026-01-01**) plus **Decree 356/2025/ND-CP** (2025-12-31). **Any planning doc citing Decree 13 as current law is out of date.**

- **PDP Law 91/2025 penalties:** cross-border transfer violations **up to 5% of prior-year revenue** (min VND 3B ≈ $114,500); illegal data sale **10× revenue gained**; criminal liability to 7 years ([Tilleke & Gibbins](https://www.tilleke.com/insights/vietnams-new-personal-data-protection-law-a-closer-look/)).
- **Cross-border transfer is post-hoc filing, not prior approval:** prepare a Cross-Border Transfer Impact Assessment, file **within 60 days of first transfer**, review every 6 months. **Exemptions include personnel/HR management transfers.**
- **Cybersecurity Law 2018 + Decree 53/2022 localization is TRIGGER-BASED** ([trade.gov](https://www.trade.gov/market-intelligence/vietnam-cybersecurity-data-localization-requirements)): activates only on **written request from the Minister of Public Security**, issued after a cybersecurity violation or failure to cooperate. Covers ~10 consumer service categories (e-commerce, social media, payments, messaging, gaming).
- **Law on Data 60/2024/QH15** (effective 2025-07-01) restricts outbound transfer of "core" and "important" data — but the sectoral catalogue remains incompletely specified.

**Does any of it force on-prem for source code? NO.**
- Vietnam's PDP regime regulates **personal data only**. Source code is not personal data — in scope only where it embeds PII.
- Decree 53 is trigger-based and applies to consumer service categories; an internal dev tool is not one.
- **No SEA jurisdiction surveyed mandates in-country hosting of source code. Not one.**
- **Singapore:** no localization; PDPA s.26 requires only "comparable standard" abroad. **Indonesia:** GR 71/2019 Art. 21 **expressly permits Private ESOs to locate offshore**; UU PDP 27/2022 has **no localization requirement**. **Malaysia:** *liberalized* — the 2024 Amendment Act **removed** the ministerial whitelist, effective 2025-04-01.

**Where real exposure sits is telemetry, not code.** AI coding assistants transmit developer emails, usernames, IP addresses, session IDs — squarely personal data under Law 91/2025, and Decree 53 names IP addresses and emails explicitly. **The required artifact is a filing, not an on-prem deployment** — and the HR/personnel exemption may cover employee telemetry entirely.

> **If self-hosting is justified to stakeholders on Vietnamese legal-mandate grounds, that argument will not survive scrutiny.** Defensible framings: trigger-risk avoidance, telemetry-PII minimization, and — primarily — **customer contractual requirements.**

### Japanese client security requirements — the strongest leg

**VERIFIED** ([GALK](https://www.galk-jp.com/blog/offshore-development-security/)). Standard requirements Japanese clients impose on Vietnamese offshore vendors:
- 「作業用PC、OS、ソフトは新たにこちらで用意する」 — *development PCs, OS, and software provided fresh by the Japan side*
- 「有線のクローズドネットワークを構築する」 — *construct a wired closed network* — **explicitly no Wi-Fi**, dedicated lines to Japan, per-project network segmentation
- Dedicated project rooms with access cards or biometrics, monitored entry/exit, personal items banned, camera surveillance
- **USB/removable media prohibited; PCs may not leave premises.** ISO 27001 baseline

The underlying anxiety in vendors' own words:
- [Sun Asterisk](https://sun-asterisk.com/service/development/topics/offshore/1248/): 「機密情報やソースコードの漏洩が大きなリスク」 — *"leakage of confidential information and source code is a major risk"*
- [Luvina](https://note.com/luvinasoftware/n/n88f932c8c148): 「ソースコードは物理的な形がないため、持ち出しが容易である」 — *"because source code has no physical form, it is easy to remove"*

**INFERRED:** this closed-network / no-removable-media / vendor-supplied-PC regime is **structurally incompatible with cloud AI coding tools.** The inference is sound. ⚠️ **But the null matters: no source was found where a Japanese client or Vietnamese vendor explicitly says "we can't use Copilot because of our closed network." That citation does not exist yet** — which is simultaneously the risk and the opportunity (CRED could be the first to document it).

### Price sensitivity — strongest quantitative point

**VERIFIED** — [ITviec Vietnam IT Salary & Recruitment Market Report 2025-2026](https://itviec.com/report/vietnam-it-salary-and-recruitment-market), n=1,839 (VND/month): mid-level (3–4 yr) back-end **30.1M**; senior (8+ yr) **54.9M**; Tech Lead 51.8M; CTO/CIO/VPoE 101.25M. Corroborated by Levels.fyi ($22.7k–$45.5k/yr senior SWE Vietnam).

**INFERRED (own arithmetic at ~26,000 VND/USD — cite as ours, not third-party):**

| | Monthly salary | $30/seat as % of salary |
|---|---|---|
| VN mid (3–4 yr) | ~$1,160 | **2.6%** |
| VN senior (8+ yr) | ~$2,110 | **1.4%** |
| US senior (~$150k/yr) | ~$12,500 | **0.24%** |

**A $30/seat/month tool costs a Vietnamese employer 6–11× more as a share of salary than a US employer.** At mid-level it approaches 2.6% of comp — **a CFO line item, not a corporate-card swipe.** For a 33,000-person org like FPT Software, $30/seat is **~$11.9M/year**.

**NULL:** no published commentary on PPP pricing for developer tools in SEA, and **no vendor (GitHub, Cursor, Anthropic) appears to offer regional pricing.** This argument is currently ours to make, unsupported by third parties.

### Industry size — and a number NOT to use

**VERIFIED** (Ministry of Science & Technology 2025, via [Tech in Asia](https://techinasia.com/news/vietnams-digital-technology-revenue-estimated-at-198b-in-2025)): digital technology revenue **$198B** (+26%); exports **$172B**; **overseas revenue from Vietnamese-owned tech firms: $15B**; ~80,000 digital tech enterprises.

⚠️ **The $198B/$172B figures are dominated by hardware/electronics assembly (Samsung, Foxconn, Intel). The software/IT-outsourcing number is the $15B. Using $172B as a software-export figure will not survive scrutiny.**

**FPT Software:** 2025 revenue **$1.34B**, **33,000+ employees**; Japan crossed **$500M in 2024 (+32.2%)**, targeting **$1B by 2027** ([FPT](https://fptsoftware.com/newsroom/news-and-press-releases/news/fpt-hits-half-billion-usd-revenue-in-japan-market-charts-path-to-1-billion-usd-by-2027)); AI & Cloud revenue +48.4%. ⚠️ Verify before external use: the AI Factory figure ($50M in one source vs a $200M NVIDIA commitment announced April 2024 — **these conflict**), and the $1.34B/33,000 numbers (LinkedIn snippet; fpt.com 403'd).

Others (snippet-level, INFERRED): CMC Global 3,322 employees; TMA Solutions 6,607 (+12.6%); Rikkeisoft planning $30M US investment. **NULL for KMS, NashTech, VMO, Saigon Technology.**

### Nulls — state plainly

- **Zero public statements from any Vietnamese IT outsourcing firm about Copilot, Cursor, or Claude Code adoption.** A dedicated search returned nothing. **Genuine information gap.**
- **No Vietnam- or SEA-specific AI coding tool adoption percentage exists.** TopDev's report is gated/JS-rendered.
- [e-Conomy SEA 2025](https://blog.google/company-news/inside-google/around-the-globe/google-asia/sea-economy-2025/): ASEAN digital economy >$300B, **$2.3B into 680+ regional AI startups** — **but no developer-tool adoption data and no Vietnam country cut.**
- **No named SEA-origin open-source dev-infra company with a documented monetization path.** Little regional precedent — cuts both ways.

---

## Go-to-market implications

1. **Sell the org/team wedge, but validate it first — it is the least-evidenced part of the thesis.** Single-dev amnesia is proven; multi-person drift rests on two non-vendor quotes. **The cheapest de-risking move available: run an original survey asking how teams share agent context.** No major survey asks this (§2), so the number would be genuinely novel, would be cited, and doubles as content marketing. Do this before heavy build.

2. **Compile to AGENTS.md first, CLAUDE.md second.** VERIFIED 154,496 vs 51,100 files. The format war is decided ~3:1, and CRED's "one versioned rule → agent-specific formats" adapter should reflect that ordering.

3. **Price at $20–40/seat/month; do not gate SSO.** The anchor is unambiguous. Memco sits at ~$50. **Gate on the durable line: SCIM, audit logs, retention policy, self-hosting support, SLA, dedicated support, IP indemnity** — the Langfuse model, which gives SSO and org-RBAC away free and still reached 63 of the Fortune 500. Gating SSO in 2026 invites the sso.tax critique and costs more in goodwill than it earns.

4. **Budget exists — anchor against it explicitly.** At $577–$1,040/dev/month on codegen (Scale VP), a $30 seat is **~3–5% of existing AI tool spend per developer.** That is the pitch: not a new line item but a small percentage of a large, already-approved, fast-growing one. The Memco benchmark framing (50% lower cost per task) is the same move — CRED needs its own equivalent measured number, which is exactly what the evaluation baseline is for.

5. **Demote self-hosting from thesis to feature — but keep it, because it is the sharpest differentiator against Cursor.** The evidence does not support "orgs demand self-hosted" (Samsung reversed; Cursor has 64% of the F500 with no on-prem; 72% are unrestricted; AI Act slipped to 2027–28). It *does* support a narrow air-gapped segment plus **self-hosting as an OSS-distribution and trust mechanism** — Langfuse's own stated reason enterprises adopt OSS is *"more visibility and control over data."* Position it that way. **And heed PostHog: self-hosters were 3.5% of users and were shed as unprofitable. Design self-hosting to be operationally cheap to support from day one, or it becomes a tax on a 3% cohort.**

6. **Expect ~1% conversion and plan the funnel accordingly.** Grafana 1%, n8n 1.3%. **OSS buys reach, not revenue** — Supabase's $170M ARR came from cloud auto-provisioning, not self-host upgrades. **Implication: CRED needs a hosted offering to capture value even though self-hosting is the differentiator.** Also plan the license question now: use **FSL** if protection is needed (standardized, 2-year Apache/MIT reversion), never SSPL — both major SSPL experiments reversed. And prepare an answer to the Cal.com argument ("AI makes open code a security liability"), which is new in 2026 and will be raised.

7. **Champion = platform/DevEx team; signer = eng leader; security = gate, not buyer.** ⚠️ **This is INFERRED** — no survey decomposing dev-tool budget ownership was found, and Gartner's platform-engineering statistic could not be verified. Supporting: Puppet (platform-mature orgs govern better), DORA ("internal platform quality determines AI value"), a16z (AI moved from innovation budgets into centralized IT). **Treat as a hypothesis to test in the first ten sales conversations, not a finding.**

8. **For Vietnam: lead with Japanese-client contractual regimes and price sensitivity. Never lead with legal mandate.** The legal argument is false and will be caught. The Japanese closed-network regime (no Wi-Fi, no USB, client-supplied PCs) is documented, specific, and structurally incompatible with cloud AI tooling — **and nobody has publicly connected those two dots yet.** Combined with the 6–11× relative price burden, that is a real and defensible regional wedge. **But note the demand side rests entirely on inference: there is zero published evidence of Vietnamese firms' AI tool adoption.** The founder's services-company access is the fastest way to convert that inference into the first primary-source datapoint — and into design-partner #1.

---

## Source index

**Practitioner evidence:** HN items [46716295](https://news.ycombinator.com/item?id=46716295), [47593179](https://news.ycombinator.com/item?id=47593179), [46091578](https://news.ycombinator.com/item?id=46091578), [47831037](https://news.ycombinator.com/item?id=47831037), [48499895](https://news.ycombinator.com/item?id=48499895), [46707311](https://news.ycombinator.com/item?id=46707311), [46946290](https://news.ycombinator.com/item?id=46946290), [48884568](https://news.ycombinator.com/item?id=48884568), [48903978](https://news.ycombinator.com/item?id=48903978), [48917688](https://news.ycombinator.com/item?id=48917688), [48914195](https://news.ycombinator.com/item?id=48914195), [48912404](https://news.ycombinator.com/item?id=48912404), [46032522](https://news.ycombinator.com/item?id=46032522), [48942927](https://news.ycombinator.com/item?id=48942927), [47173315](https://news.ycombinator.com/item?id=47173315), [47022951](https://news.ycombinator.com/item?id=47022951), [48525036](https://news.ycombinator.com/item?id=48525036), [48898701](https://news.ycombinator.com/item?id=48898701), [46392675](https://news.ycombinator.com/item?id=46392675), [48001931](https://news.ycombinator.com/item?id=48001931), [47245389](https://news.ycombinator.com/item?id=47245389), [48902307](https://news.ycombinator.com/item?id=48902307), [48967259](https://news.ycombinator.com/item?id=48967259), [48912640](https://news.ycombinator.com/item?id=48912640), [48910490](https://news.ycombinator.com/item?id=48910490), [48904240](https://news.ycombinator.com/item?id=48904240), [46699490](https://news.ycombinator.com/item?id=46699490), [48965310](https://news.ycombinator.com/item?id=48965310)

**Surveys:** [Stack Overflow 2025 AI](https://survey.stackoverflow.co/2025/ai) · [SO 2025 Technology](https://survey.stackoverflow.co/2025/technology) · [DORA 2025](https://cloud.google.com/blog/products/ai-machine-learning/announcing-the-2025-dora-report) · [Octoverse 2025](https://github.blog/news-insights/octoverse/octoverse-a-new-developer-joins-github-every-second-as-ai-leads-typescript-to-1/) · [JetBrains 2025](https://devecosystem-2025.jetbrains.com/artificial-intelligence) · [Puppet 2026](https://www.puppet.com/resources/2026-state-of-platform-engineering) · [Netskope Shadow AI 2025](https://www.netskope.com/resources/cloud-and-threat-reports/cloud-and-threat-report-shadow-ai-and-agentic-ai-2025) · [IBM breach 2025 via Nudge](https://www.nudgesecurity.com/post/shadow-ai-the-emerging-security-threat-in-ibms-2025-cost-of-a-data-breach-report)

**Pricing:** [Memco](https://www.memco.ai/) · [Glen/YC](https://www.ycombinator.com/companies/glen) · [Langfuse](https://langfuse.com/pricing) · [Langfuse self-host](https://langfuse.com/pricing-self-host) · [Cursor](https://cursor.com/pricing) · [Cursor Enterprise](https://cursor.com/enterprise) · [Copilot](https://github.com/features/copilot/plans) · [Mem0](https://mem0.ai/pricing) · [Zep](https://www.getzep.com/pricing) · [Onyx](https://www.onyx.app/pricing) · [Tabnine](https://www.tabnine.com/pricing/) · [Sourcegraph](https://sourcegraph.com/pricing) · [Amp](https://ampcode.com/pricing) · [Devin](https://devin.ai/pricing) · [Augment](https://www.augmentcode.com/pricing) · [Qodo](https://www.qodo.ai/pricing/) · [Graphite](https://graphite.com/pricing) · [CodeRabbit](https://www.coderabbit.ai/pricing) · [Unblocked](https://getunblocked.com/pricing) · [sso.tax](https://sso.tax/)

**Buyers/budget:** [Scale VP codegen spend](https://www.scalevp.com/blog/how-much-startups-are-actually-spending-on-code-generation) · [a16z enterprise AI](https://a16z.com/ai-enterprise-2025/) · [Bessemer AI data trends](https://www.bvp.com/atlas/data-trends-state-of-working-with-ai) · [Bessemer dev tooling](https://www.bvp.com/atlas/roadmap-developer-tooling-for-software-3-0)

**OSS conversion:** [Sacra Grafana](https://sacra.com/chat/h/0f295770-caee-4cc2-8769-aab4196d31d2/) · [Sacra n8n](https://sacra.com/c/n8n/) · [Sacra Supabase](https://sacra.com/c/supabase/) · [n8n Series C](https://blog.n8n.io/series-c/) · [PostHog sunsetting Helm](https://posthog.com/blog/sunsetting-helm-support-posthog) · [Langfuse spotlight](https://posthog.com/spotlight/startup-langfuse) · [ClickHouse acquires Langfuse](https://clickhouse.com/blog/clickhouse-acquires-langfuse-open-source-llm-observability) · [Cal.com closed source](https://cal.com/blog/cal-com-goes-closed-source-why) · [Cockroach licensing](https://techcrunch.com/2024/08/15/cockroach-labs-shakes-up-its-licensing-to-force-bigger-companies-to-pay/) · [RethinkDB post-mortem](https://gist.github.com/ramalho/93b87e961b6e019be8e1f6f82864b6f9) · [Sentry FSL](https://blog.sentry.io/introducing-the-functional-source-license-freedom-without-free-riding/) · [Redis AGPLv3](https://redis.io/blog/agplv3/) · [Asay on OSS + AI](https://www.infoworld.com/article/3548263/open-source-isnt-going-to-save-ai.html) · [Levine 2014](https://stanfordreview.org/why-there-will-never-be-another-redhat-the-economics-of-open-source/) · [ROSS Index](https://runacap.com/ross-index/) · [LWN on COSS 2025](https://lwn.net/Articles/1034944/)

**Self-hosting / regulatory:** [Samsung ban 2023](https://www.forbes.com/sites/siladityaray/2023/05/02/samsung-bans-chatgpt-and-other-chatbots-for-employees-after-sensitive-code-leak/) · [Samsung reversal 2026](https://openai.com/index/samsung-electronics-chatgpt-codex-deployment/) · [Copilot data residency](https://docs.github.com/en/enterprise-cloud@latest/admin/data-residency/github-copilot-with-data-residency) · [AI Act timeline](https://artificialintelligenceact.eu/implementation-timeline/) · [Council AI Act delay 2026-06-29](https://www.consilium.europa.eu/en/press/press-releases/2026/06/29/artificial-intelligence-council-gives-final-green-light-to-simplify-and-streamline-rules/) · [Gaia-X](https://gaia-x.eu/) · [Langfuse self-hosting](https://langfuse.com/self-hosting) · [n8n deployment options](https://docs.n8n.io/choose-how-to-use-n8n/) · [Onyx deployment](https://docs.onyx.app/deployment/overview)

**Vietnam/SEA:** [Tilleke on PDP Law](https://www.tilleke.com/insights/vietnams-new-personal-data-protection-law-a-closer-look/) · [VILAF on Decree 356/2025](https://www.vilaf.com.vn/blog/vietnams-new-personal-data-protection-decree-key-compliance-requirements-effective-immediately/) · [trade.gov Vietnam localization](https://www.trade.gov/market-intelligence/vietnam-cybersecurity-data-localization-requirements) · [Freshfields Decree 53](https://www.freshfields.com/en/our-thinking/blogs/technology-quotient/data-localisation-in-vietnam-highlights-under-decree-53-and-decree-13-102iulg) · [GALK offshore security](https://www.galk-jp.com/blog/offshore-development-security/) · [Sun Asterisk](https://sun-asterisk.com/service/development/topics/offshore/1248/) · [Luvina](https://note.com/luvinasoftware/n/n88f932c8c148) · [ITviec salary report](https://itviec.com/report/vietnam-it-salary-and-recruitment-market) · [Tech in Asia VN digital revenue](https://techinasia.com/news/vietnams-digital-technology-revenue-estimated-at-198b-in-2025) · [FPT Japan $500M](https://fptsoftware.com/newsroom/news-and-press-releases/news/fpt-hits-half-billion-usd-revenue-in-japan-market-charts-path-to-1-billion-usd-by-2027) · [e-Conomy SEA 2025](https://blog.google/company-news/inside-google/around-the-globe/google-asia/sea-economy-2025/)
