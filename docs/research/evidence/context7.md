# Context7 — distribution scan

- **Date:** 2026-07-20
- **Status:** Evidence document
- **Purpose:** D-006 committed CRED to competing on **distribution**. Context7
  is the closest available specimen of an MCP server that won distribution in
  exactly CRED's slot. This records what it did, not whether it is a competitor
  — it is not one.

> **Evidence note.** Figures marked ✅ were pulled by me on 2026-07-20 from the
> GitHub API, the npm registry API, the HN Algolia API, or read out of a local
> clone of `upstash/context7` at `master` (pushed 2026-07-20T07:21:31Z).
> `file:line` references are to that clone. Claims sourced only to Context7's
> own documentation are labelled **claim** — they are evidence of what Upstash
> says, never of what the system does, because the component that would prove
> it is closed source. Unverified items are collected in the last section.

---

## What it is

An MCP server that injects up-to-date library documentation into a coding
agent's context. The user writes `use context7` in a prompt; the agent resolves
a library name to an ID, then queries documentation for it.

It is **not** a CRED competitor. Context7 serves **public** knowledge —
third-party library docs, authored by someone else, fetched from source. CRED
claims **private, organizational** knowledge. Same install slot, same users,
same moment; disjoint content. Whether that composes is addressed in the closing
section.

The repository is a monorepo of **clients only** — `packages/{cli,mcp,pi,sdk,
tools-ai-sdk}`. There is no crawler, no parser, no indexer, no ranker and no
classifier in the tree. Every tool call is an HTTP request to
`https://context7.com` (`packages/mcp/src/lib/constants.ts:10-11`). **The
product is the closed index; the open-source repo is the distribution
wrapper.**

---

## Adoption numbers ✅

All pulled 2026-07-20.

| Metric | Value | Source |
|---|---|---|
| Stars | **59,457** | GitHub API `/repos/upstash/context7` |
| Forks | 2,843 | same |
| Watchers | 155 | same |
| Open issues | 25 | same |
| License | **MIT** | same |
| Repo created | **2025-03-26** | same |
| First commit | 2025-03-29 ("Create README.md") | `git log --reverse` |
| Total commits | **899** | `git rev-list --count HEAD` |
| Distinct git authors | **138** (incl. bots, duplicate identities) | `git shortlog -sn` |
| GitHub contributors | **~124** | contributors API, `per_page=1`, last page |
| `@upstash/context7-mcp` | **3,702,074 downloads/month** | npm API, 2026-06-20→07-19 |
| `ctx7` (CLI) | **168,156 downloads/month** | npm API, same window |

**Commit cadence** ✅ (`git log --format=%ad | uniq -c`) — sustained, not a
spike: 115 commits in 2025-04, then between 18 and 72 every month through
2026-07. Never dormant, never heroic.

**Authorship is concentrated.** The top four identities (Enes Gules, Fahreddin
Özcan, `enesgules`, Abdusshh) account for **512 of 899 commits (57%)**, and the
top identity alone for 211. The ~124-contributor number is real but the long
tail is documentation, i18n and client-config PRs. This matters for CRED:
**59k stars did not buy a distributed engineering team.** Bessemer's "100
unique monthly contributors is rare territory" benchmark cited in
[market-landscape.md](market-landscape.md) is not met here either.

### The star-to-usage ratio is the useful number

3.7M npm downloads/month against 59k stars is a **~62:1 download-to-star
ratio**. Compare `rulesync` in [market-landscape.md](market-landscape.md):
807,749 downloads on 1,245 stars, a ~649:1 ratio. Context7 converts mindshare
*and* usage; rulesync converts usage without mindshare. Both beat the
stars-only projects. This supports that document's recommendation to instrument
installs rather than stars.

---

## The launch: 4 points on Hacker News ✅

This is the single most important finding in this document.

> **Show HN: Context7 – LLM Code Snippets from Docs in Minutes**
> Submitted by `enesakar`, 2025-03-28T17:30:12Z.
> **4 points. 2 comments.**
> [HN 43507993](https://news.ycombinator.com/item?id=43507993)

Verified via the HN Algolia items API. Both comments, in full:

> **samchon:** "I wrote some of my libraries, but no reaction at all from the
> website. It's sad that I'm not famous ToT"
>
> **enesakar:** "you can add yours: https://context7.com/add-package"

An HN Algolia story search for `context7` ✅ returns no Context7 submission
above **4 points**. The other self-posts scored 3, 3, 2, 1, 1, 1, 1.

**Set this against `docs/research/evidence/spike-demand-and-buyers.md` and the
"graveyard" table in [market-landscape.md](market-landscape.md)**, which record
memory- and rules-product launches scoring 1–4 points and read those scores as
evidence of absent demand. Context7 scored **4 points with 2 comments** and went
on to 59,457 stars and 3.7M downloads/month.

**FALSIFIED, as a general inference: a low HN launch score does not predict
failure for an MCP server.** The graveyard table is still evidence — those
products did fail — but the HN score is not what shows it, and this document
supplies the counterexample. CRED should stop treating a bad HN launch as a
kill signal. It is close to uncorrelated.

What the launch score *is* evidence of: **Context7's distribution did not come
from a launch event.** It accumulated. The mechanism is below.

---

## Install path and time-to-first-value

### At launch (VERIFIED against git history)

The README at commit `13631ee` (2025-04-19) ✅ instructed users to paste into
`~/.cursor/mcp.json`:

```json
{ "mcpServers": { "context7": {
    "command": "npx", "args": ["-y", "@upstash/context7-mcp@latest"] } } }
```

**No account. No API key. No hosted signup.** One JSON block, one `npx`.
`CONTEXT7_API_KEY` does not appear in the tree until **2025-07-01** ✅
(`git log -S CONTEXT7_API_KEY --reverse`, commit `b808643`, "feat: add API key
support for authentication across all transports") — **three months and roughly
15,000 stars after launch.** There is even a commit `7cadd17` (2025-08-19)
titled *"docs: remove API key mentions until deployment"* and `7d4a2fa`
(2025-08-20) *"fix: remove the api key to clear confusion"* — they actively
fought to keep the key out of the install path.

**This is the strongest available evidence for CRED's "no API key at first run"
constraint (D-001, D-003 rung 1).** It is not a principle Context7 professed;
it is the configuration it actually shipped during the period it acquired its
users, and the key arrived only after adoption existed.

### Today

`README.md` now leads with:

```bash
npx ctx7 setup
```

which the README describes as: *"Authenticates via OAuth, generates an API key,
and installs the appropriate skill."* Steps to first value: **1 command, plus a
browser OAuth round-trip.**

Anonymous use still works. `server.json` declares `CONTEXT7_API_KEY` with
`"isRequired": false` for the npm package, the `mcpb` bundle, and the remote
streamable-HTTP endpoint. The key buys rate limit, not access —
`packages/mcp/src/lib/api.ts:27-29` returns *"Create a free API key at
https://context7.com/dashboard for higher limits"* to anonymous callers and
*"Upgrade your plan"* to keyed ones.

**So the trajectory is: zero-friction at launch → optional key at month 3 →
OAuth-first in the README at month ~15, with the zero-key path preserved
underneath.** That ordering is the transferable lesson, and the preserved
anonymous path is why the ladder still has a bottom rung.

### The rate-limit nudge, and a self-inflicted injection lesson

When an anonymous client crosses a per-IP threshold the backend sets
`X-Context7-Auth-Prompt: 1` and the server nudges the user to sign in. The
*first* implementation appended that nudge to the tool result text. From
`packages/mcp/CHANGELOG.md` (3.2.0, commit `c921c8b`), verbatim:

> "Surfaces the `npx ctx7 setup --<client> --mcp[ --stdio] -y` command in a
> client-rendered dialog rather than as model-visible text. **The previous
> text-injection approach was treated as untrusted instruction content by some
> agents**; elicitations are delivered out-of-band to the user so they bypass
> that path entirely."

They shipped an upsell inside the model's context window, agents correctly
classified their own vendor as a prompt injector, and they moved it to MCP
`elicitation/create` (`packages/mcp/src/lib/auth/auth-prompt.ts:26-50`), gated
on the client advertising the `elicitation` capability so it is a safe no-op
elsewhere. **Monetization pressure produced a prompt injection.** CRED will feel
the same pressure at the same place.

---

## Distribution channels

This is where the adoption came from, and it is unglamorous: **they packaged
natively for every ecosystem, one at a time, for fifteen months.**

Verified in-tree:

| Channel | Artifact | Path |
|---|---|---|
| npm | `@upstash/context7-mcp`, `ctx7` | `packages/` |
| Official MCP registry | `io.github.upstash/context7` | `server.json` ✅ listed |
| Claude Code plugin marketplace | `context7-marketplace` | `.claude-plugin/marketplace.json` |
| Agent-plugin marketplace | second manifest | `.agents/plugins/marketplace.json` |
| Gemini CLI extension | extension manifest | `gemini-extension.json` |
| Per-client plugins | claude, codex, copilot, cursor, context7-power | `plugins/` |
| Agent Skills | 3 skills | `skills/{context7-mcp,context7-cli,find-docs}/SKILL.md` |
| Rules files | 2 | `rules/context7-{cli,mcp}.md` |
| MCPB bundle | signed `.mcpb`, sha256 pinned | `server.json` |
| Docker | official image, + Hardened Image | commits `b9d5309`, `a29e56b` |
| Smithery | badge in README | `README.md` |
| Cursor one-click | base64 deeplink button, **line 3 of the README** | `README.md` |

✅ Confirmed live in the official MCP registry via
`registry.modelcontextprotocol.io/v0/servers?search=context7` — returns
`io.github.upstash/context7` at versions 1.0.0, 1.0.30, 1.0.31.

**Context7's own docs claim setup instructions for 51 MCP clients**
([all-clients](https://context7.com/docs/resources/all-clients), fetched
2026-07-20) — Claude Code, Cursor, Codex, VS Code, Zed, Warp, Amp, JetBrains,
LM Studio, Windows, Docker, and so on — with **5 one-click install buttons**
(Cursor, Kiro, LM Studio, Smithery, VS Code). The repo carries 8 first-class
client docs (`docs/clients/`). **Claim** for the 51; verified for the 8 and for
the plugin/extension manifests above.

Plus **15 translated READMEs** (`i18n/`) — zh-TW, zh-CN, ja, ko, es, fr, pt-BR,
it, id, de, ru, uk, tr, ar, vi. A meaningful share of the contributor tail is
i18n work.

**Read: distribution was a grind, not an event.** No single channel drove it.
The strategy was to be already-packaged wherever a user forms the intent to
install something, so the install is one click from inside the tool they are
already in. There is no growth hack here to copy — only a checklist to work.

---

## Tool surface

**Exactly two MCP tools.** ✅ Verified by grep across `packages/`, and by the
test at `packages/pi/__tests__/extension.test.ts:20` which asserts the full set:

```js
expect([...tools.keys()].sort()).toEqual(["query-docs", "resolve-library-id"]);
```

### `resolve-library-id` — `packages/mcp/src/index.ts:134-219`

```ts
inputSchema: {
  query:       z.string(),  // the user's question; used to RANK results
  libraryName: z.string(),  // e.g. 'Next.js', not 'nextjs'
}
annotations: { readOnlyHint: true, destructiveHint: false,
               openWorldHint: true, idempotentHint: true }
```

### `query-docs` — `packages/mcp/src/index.ts:221-262`

```ts
inputSchema: {
  libraryId: z.string(),  // '/vercel/next.js' or '/vercel/next.js/v14.3.0-canary.87'
  query:     z.string(),  // one concept per call
}
annotations: { readOnlyHint: true, destructiveHint: false,
               openWorldHint: true, idempotentHint: true }
```

Both are two-parameter, both strings, both read-only. There is no pagination
parameter, no token budget parameter, no filter object. The v1 tool
`get-library-docs` was replaced by `query-docs` in commit `66ea0d6` ("new arch",
2025-12-29) ✅ — a rename **and** a semantic shift from *fetch a document* to
*ask a question of a document*, moving the retrieval decision server-side.

**Against CRED's committed 4 tools (D-006): the most-installed MCP server in
this slot ships 2.** Two is evidently tolerable; 4 is not obviously a problem,
but the direction of the evidence is toward fewer. Note also that the entire
surface is `readOnlyHint: true`. CRED's `create_memory` / `enrich_memory` /
`share_feedback` are writes, which is a materially harder install — a read-only
server is trivially safe to try, and trialability is what a 62:1
download-to-star ratio is made of.

### Three details worth stealing

**1. Server-level `instructions`** (`packages/mcp/src/index.ts:128-130`) ship
inside the MCP handshake and tell the agent both when to call and **when not
to**: *"Do not use for: refactoring, writing scripts from scratch, debugging
business logic, code review, or general programming concepts."* An explicit
negative scope, in-band, not left to the user's rules file.

**2. A hard call budget in the description**, twice: *"IMPORTANT: Do not call
this tool more than 3 times per question."* Round-trip cost is managed by
instruction because the protocol offers no other lever.

**3. An alias map for hallucinated argument names** —
`packages/mcp/src/index.ts:276-292`, with the comment:

> "LLM clients often echo phrasing from tool descriptions instead of the literal
> schema keys, which trips Zod validation before the tool runs."

`query` accepts `userQuery` and `question`; `libraryId` on `query-docs` accepts
`context7CompatibleLibraryID`, `libraryID` and `libraryName`. This is hard-won
operational knowledge from millions of real calls: **agents get your argument
names wrong, and strict schema validation turns that into a failed tool call.**
CRED should ship an alias layer from day one rather than rediscovering this.

---

## The ambiguity problem: resolve-then-query

A user says "React". Which library is that? Context7's answer is a mandatory
two-step. From the `resolve-library-id` description:

> "You MUST call this function before 'Query Documentation' tool to obtain a
> valid Context7-compatible library ID **UNLESS the user explicitly provides a
> library ID in the format '/org/project' or '/org/project/version'**."

**Cost:** one extra round-trip per question, on every question, forever — plus
the resolve result itself occupying context. That is a real and permanent tax,
and they paid it deliberately.

**Why they paid it:** the escape hatch. Because IDs are stable, human-typable
paths (`/vercel/next.js`), a user who knows the ID skips the step entirely, and
the README teaches exactly that (*"use library /supabase/supabase"*). The
namespace doubles as a public, memorable addressing scheme. The two-step is the
fallback, not the design centre.

**They also cache the disambiguation upward.** `query` is passed to
`resolve-library-id` purely to rank — the server disambiguates using the user's
intent, not just the name string. Ambiguity is resolved server-side with more
information than the client has.

**For CRED:** organizational memory has a worse version of this problem — no
public namespace exists to fall back on, and no one can type
`/acme/auth-conventions` from memory on day one. If CRED adopts resolve-then-
query it pays the round-trip **without** the escape hatch that makes it
tolerable here. That argues for single-step retrieval with server-side
disambiguation.

---

## Freshness

From `docs/library-updates.mdx` (in-tree, so verified as documentation; the
implementing service is closed — **claim**):

Refresh is **lazy, demand-gated, and popularity-tiered.** On every request the
system checks the doc age; if older than the library's threshold, it fires a
background refresh and **still returns the stale copy immediately**.

| Popularity rank | Refresh threshold |
|---|---|
| Top 100 | 1 day |
| Top 1,000 | 15 days |
| Top 5,000 | 30 days |
| All others | 45 days |

> "If a library hasn't been requested recently, it won't be refreshed — keeping
> the system efficient and focused on actively used documentation."

Sources re-ingested: git repos re-parsed from branch, websites re-crawled from
base URL, OpenAPI specs re-fetched. Private libraries are **never** auto-
refreshed. Logged-in users can force a refresh; owners who claim a library get
higher refresh rate limits.

**Compare to CRED's evidence-based invalidation.** These are different
mechanisms answering different questions:

| | Context7 | CRED (stated thesis) |
|---|---|---|
| Trigger | time elapsed + demand | the cited evidence changed |
| Question | "is this old?" | "is this still true?" |
| Cost | one timestamp compare | tracking claim→evidence links |
| Failure | serves stale docs within TTL | claim survives its evidence |
| Truth source | re-read the upstream source | the evidence artifact itself |

Context7 can use a TTL because **its upstream is authoritative and
re-readable** — re-crawl the repo and you have the truth by construction. CRED
has no such upstream; an organizational claim's truth is not recoverable by
re-reading anything. **So TTL is genuinely unavailable to CRED, and the
evidence-link machinery is not over-engineering.** That is the one place CRED's
harder mechanism is clearly justified rather than merely preferred.

**But the demand gate transfers directly and cheaply.** *Never revalidate a
claim nobody has retrieved recently.* Invalidation work scales with reads, not
with corpus size. CRED should adopt this; it is the difference between a
curation worker that scales and one that does not.

---

## Ranking and trust

`resolve-library-id` returns, per its own description
(`packages/mcp/src/index.ts:142-160`), five ranking signals:

- **Code Snippets** — count of available examples
- **Source Reputation** — `High | Medium | Low | Unknown`
- **Benchmark Score** — "Quality indicator (100 is the highest score)"
- Name similarity (exact matches prioritized)
- Description relevance to query intent

**The ranking is not performed by the server. It is performed by the LLM.** The
tool returns a formatted list and the description instructs the agent how to
choose: *"Return the most relevant match based on: name similarity…
documentation coverage (prioritize libraries with higher Code Snippet counts),
source reputation…"* Selection is delegated to the model, in prose.

**How Source Reputation and Benchmark Score are computed is not in the
repository and not in the docs.** ⚠️ UNVERIFIED and, from outside, unverifiable.
Checked: no scoring code under `packages/`; `docs/` describes neither.

This is a clean instance of what D-005 identified as the auditability axis. The
authority signal that decides which documentation reaches a production coding
agent is an opaque number from a closed service. Nobody can audit it, reproduce
it, or contest a score. **Context7's users do not appear to care** — 59k stars
and 3.7M downloads/month with an unexplained "Benchmark Score: 100".

**That is disconfirming evidence for auditability as a wedge, and it should be
recorded as such.** D-006 already demoted auditability; this supports the
demotion. Users trade audit for convenience without hesitation when the content
is public and low-stakes. The open question — genuinely open — is whether that
holds when the content is *the organization's own beliefs* rather than React
docs. Higher stakes and an internal owner may change the answer, but Context7
provides no evidence that it does.

Note also that the ranking signals are **popularity-shaped** — snippet count is
coverage, reputation is authority-by-reputation. This is exactly the
popularity-based trust CRED's thesis rejects. It is what the winner shipped.

---

## OSS / hosted boundary

**License:** MIT, whole repo (`LICENSE`, GitHub API `license.spdx_id = MIT`).
No `/ee` directory, no dual license, no BSL. Compare Langfuse's thin `/ee` and
Onyx's MIT+EE in [market-landscape.md](market-landscape.md).

**They did not need a license boundary, because the boundary is architectural.**

| Open (MIT, in repo) | Closed (service) |
|---|---|
| MCP server, CLI, SDK, AI-SDK tools | The document index |
| Tool definitions and descriptions | Crawler, parser, chunker |
| Client configs, plugins, skills, rules | Search and ranking |
| Auth/OAuth client, session store | Source Reputation, Benchmark Score |
| Docs, i18n | Prompt-injection classifier |

Everything that creates value is a network call. Forking the repo yields a
client that calls `context7.com`. **This is precisely Zep's "open-source the
layer that powers your product but does not substitute for it," which
[market-landscape.md](market-landscape.md) calls "the single most actionable
insight" in its OSS-business section — and Context7 is a cleaner execution of
it than Zep's own.** Zep open-sourced an engine (Graphiti) that others could
run; Context7 open-sourced a client that others cannot run without them.

**Monetization** ([context7.com/plans](https://context7.com/plans), fetched
2026-07-20 — **claim**, vendor pricing page):

| Plan | Price | Included |
|---|---|---|
| Free | $0 | 1,000 API calls/mo, public repos only |
| Pro | **$10/seat/mo** | 5,000 calls/seat; overage $10/1k; private repo parsing $25/1M tokens |
| Enterprise | custom | from **$30/user/mo**, scaling down to **$2.50/user/mo** at size; SOC 2, SSO, self-hosted |

Two things CRED's price anchor should absorb. **Pro at $10/seat sits below the
$20–40 anchor in D-007** — a metered-usage developer tool prices under a
seat-based platform. And **Enterprise scales *down* to $2.50/user** — volume
pricing, not the enterprise-multiplier assumption. D-007's note that Memco's
$599/contributor/year is "outside the market" is reinforced: Context7's
enterprise floor is roughly **one-twentieth** of it per user at scale.

Also relevant to D-007's "never gate SSO": Context7 **does** gate SSO to
Enterprise. Langfuse does not, and landed 63 Fortune 500 logos. Two data points,
opposite directions; D-007's rule stands on the Langfuse evidence, but it is not
unanimous practice.

Enterprise on-premise is real and recent — `docs/enterprise/on-premise.mdx`,
`docs/enterprise/gitops.mdx` (commit `74dc6ba`, 2026-06-24),
`docs/enterprise/library-import.mdx`, and Okta / Enterprise-Managed Auth
(`id-jag`) validation in commit `2253765` (2026-06-22). **The self-host path
arrived ~15 months after launch, as an enterprise SKU.** Same sequencing D-007
assigns to CRED: capability, not wedge.

---

## Prompt-injection posture

CRED's L8 says ingested content is untrusted. Context7 serves third-party
content into an agent's context, so this is the same hazard. The record:

### It was exploited. ContextCrush, CVE-class, disclosed 2026-03-05.

Noma Labs,
[ContextCrush](https://noma.security/blog/contextcrush-context7-the-mcp-server-vulnerability/)
(fetched 2026-07-20):

- **Mechanism:** Context7 had a **"Custom Rules"** (a.k.a. "AI Instructions")
  feature letting a *library owner* attach instructions, set in the Context7
  dashboard. Per Noma: *"the custom rules were served verbatim through
  Context7's MCP server to every user who queried that library, with no
  sanitization, content filtering, or distinction from the legitimate
  documentation."*
- **Attack:** register a library, embed instructions in Custom Rules. Any
  developer querying it gets them injected as authoritative context.
- **PoC, three stages:** read all `.env` files → exfiltrate to an
  attacker-controlled GitHub repo → delete local folders under a "Cleanup"
  pretext. Executed with the developer's local permissions.
- **Timeline:** discovered 2026-02-18 → accepted 2026-02-19 → production fix
  2026-02-23 (**5 days**) → public disclosure 2026-03-05. No exploitation
  observed in the wild.
- Covered by Infosecurity Magazine, SC Media and BankInfoSecurity. The HN
  submission scored **2 points, 0 comments** ✅
  ([HN 47491665](https://news.ycombinator.com/item?id=47491665)).

**The design error is exactly the one CRED is exposed to.** A content-submission
surface where a third party can attach *instructions* — not just data — that are
served verbatim into an agent's context, with no channel separation between
"documentation" and "directives." CRED's contributed memories are the same
shape, with a worse blast radius: a poisoned organizational claim is retrieved
by every agent in the company and *looks* authoritative by design.

### Current posture

`docs/security/data-safety.mdx` — a **layered malicious content detection
system**: a classifier "tailored for Context7" for injection and malware
patterns, targeted validation of suspicious content, continuous monitoring of
flagged content, and updated detection logic. It concludes:

> "This ensures that documentation retrieved through Context7 is safe to consume
> by both human developers and AI coding agents."

**Label this a claim, and treat that last sentence as marketing.** The
classifier is not in the repo, there is no published evaluation, no false-
negative rate, no red-team result. It is a closed classifier asserted to be
sufficient. Post-hoc content classification is a mitigation, not a boundary.

The CLI does ship one real, readable control: `packages/cli/CHANGELOG.md`
records *"Add prompt injection detection with warning messages for blocked
skills"*, and `packages/cli/src/commands/skill.ts` implements it — skills
downloaded from GitHub are scanned before install.

**Three findings CRED should carry:**

1. **The highest-value injection vector was a feature, not a bug.** "Let library
   owners give the agent instructions" is a reasonable product idea that is
   structurally an injection endpoint. CRED will be asked for the same feature.
2. **Nobody punished them.** 2 HN points, three trade-press writeups, no
   discernible dent in downloads. Security incidents are not a distribution risk
   in this category — which is a reason to hold the line on principle, since the
   market will not enforce it.
3. **Classification is what they landed on, and it is weaker than separation.**
   CRED's L8 should specify a structural boundary — retrieved content marked as
   data, never as instruction, at the protocol layer — rather than a classifier
   that has to win an arms race.

---

## What transfers to CRED

**Transfers directly:**

- **Zero-key first run, defended.** Verified in git: no API key existed for the
  first 3 months, and two commits explicitly removed key mentions to reduce
  confusion. This is the empirical backing for D-001/D-003 rung 1.
- **Package for every ecosystem, one at a time.** MCP registry + Claude plugin
  marketplace + Gemini extension + MCPB + Docker + Smithery + per-client docs.
  Fifteen months of unglamorous checklist work, no growth hack.
- **A one-click install button as line 3 of the README**, above the title.
- **Argument-alias tolerance.** Agents hallucinate parameter names; strict Zod
  validation turns that into a failed call
  (`packages/mcp/src/index.ts:276-292`). Ship aliases from day one.
- **Server-level `instructions` with explicit negative scope** — tell the agent
  when *not* to call you, in-band.
- **A stated call budget** in the tool description.
- **Demand-gated revalidation.** Never revalidate a claim nobody has read
  recently. Invalidation cost scales with reads, not corpus size.
- **A stable, human-typable ID namespace** as the escape hatch from
  disambiguation.
- **Architectural OSS boundary over a license boundary.** MIT everything,
  because the index is the product. Cleaner than an `/ee` folder and it never
  triggers a relicensing crisis.

**Does not transfer:**

- **TTL-based freshness.** Requires an authoritative re-readable upstream. CRED
  has none. The evidence-link machinery is justified here, not merely preferred.
- **Two tools.** Context7 is read-only; CRED writes. Read-only is trivially safe
  to trial, which is a large part of why installs are cheap for them and will
  not be for CRED.
- **Popularity-based ranking.** Snippet count and Source Reputation are
  authority-by-popularity. CRED rejects this. Record honestly that the winner
  shipped it and no one objected.
- **A public namespace.** `/vercel/next.js` is memorable because it is public
  and shared. `/acme/auth-conventions` is not typable from memory on day one.
- **Prompt-injection posture.** Do not copy it. Copy the incident.

---

## Closing interpretation

*Marked as interpretation, per `.claude/rules/docs.md` §2 — everything above is
the scan.*

### 1. What drove distribution, and can a solo founder reproduce it?

**Not the launch.** 4 points, 2 comments, and no Context7 submission ever
cleared 4. Three things did the work, and they are unequally reproducible.

**(a) A universally-felt, instantly-legible problem.** "The LLM gives me
outdated API docs" needs no explanation and every user has hit it that week.
Value is verifiable in one prompt. **Reproducible in principle, but this is
where CRED is weakest** — "our agents don't share organizational memory" is a
problem a user must be *convinced* they have. D-007 already records the demand
risk (sellers outnumbering buyers ~10:1); this scan sharpens *why* it matters.
Context7 did not need demand generation because the pain was pre-existing and
pre-articulated. CRED will need it, and demand generation is expensive for a
solo founder.

**(b) Zero-friction, zero-risk trial.** Read-only, no account, one `npx`. The
decision to try costs nothing and risks nothing. **Fully reproducible, and CRED
has already committed to it.** But CRED writes, which raises the cost of trying
— worth designing a read-only first-run mode so the first install is as cheap as
Context7's.

**(c) Fifteen months of packaging every distribution channel as it appeared.**
Registry, marketplaces, extensions, bundles, one-click buttons, 51 client
configs, 15 translations. **This is the reproducible part, and it is the actual
answer.** It requires no funding, no team and no network — only sustained
attention. It is also the part a founder betting on "distribution" is most
likely to skip, because it looks like chores rather than strategy.

**The honest read on D-006.** Context7 is a genuine distribution win, and it
mostly validates that distribution is winnable solo. But its distribution rested
on (a) — a problem nobody had to be sold — and D-006's bet is that the founder's
channel access substitutes for that. Context7 does not provide evidence for the
substitution. It is the best available proof that the *mechanics* are
reproducible, and silent on whether they work without pre-existing demand.

### 2. Composes or competes?

**Composes, cleanly, and the composition is worth stating in CRED's
positioning.**

The split is exhaustive and non-overlapping. Context7 answers *"how does this
library work?"* from an authoritative, re-readable, public upstream, kept fresh
by TTL. CRED answers *"what did we learn / decide / break here?"* from a private
corpus with no authoritative upstream, kept honest by evidence links. Neither
mechanism works on the other's content: a TTL cannot validate "we chose Postgres
over Mongo because X," and an evidence graph is pointless for React docs when
re-crawling is cheaper and exact.

Concretely: an agent resolving a Next.js middleware question should call
Context7; the same agent asking why *this* repo wraps its middleware in a
tenant guard should call CRED. Both in the same session, no conflict.

Two things CRED can take from this:

- **A crisp positioning line.** "Context7 for your organization's own knowledge"
  is legible to anyone who has installed Context7 — which, at 3.7M
  downloads/month, is a large fraction of CRED's addressable users. It borrows
  the shape of a proven install without competing for the slot.
- **A real integration.** CRED's evidence model could cite a Context7 library ID
  as the public-knowledge anchor for a claim ("this convention exists because
  `/vercel/next.js` behaves this way"), making public/private provenance one
  graph. Genuinely useful, and cheap.

**The risk in the other direction:** Upstash is already indexing **private**
repositories (Pro tier, $25/1M tokens; `docs/enterprise/library-import.mdx`;
on-prem and GitOps docs shipped June 2026). They are moving from public docs
toward private organizational content. They will not build claim-level evidence
governance — that is not their mechanism — but they will own the *install slot*
and the *habit*, and they have 59k stars and a paying enterprise motion already
there. **Compose now; expect the boundary to be contested within a year.**

### 3. What did it get wrong that CRED should not repeat?

**(a) Serving third-party instructions verbatim into an agent's context.**
ContextCrush was a *feature* — "Custom Rules," owner-supplied AI instructions,
served with no sanitization and no distinction from documentation. CRED will be
asked for the identical feature ("let the team pin guidance for agents") and it
is the same vulnerability with a worse blast radius. **The fix is structural,
not classifier-based:** retrieved content must be transported and rendered as
data, never as instruction, with the boundary enforced at the protocol layer.
Context7 chose a closed classifier with no published evaluation and asserted in
its docs that this makes content "safe to consume." That sentence should never
appear in CRED's docs.

**(b) Injecting an upsell into the model's context window.** They appended a
sign-in nudge to tool results; agents flagged their own vendor as an injector;
they retreated to out-of-band elicitations. The general failure is treating the
model's context as a marketing surface. CRED will face this pressure at the same
place — the rate-limit boundary — and should adopt elicitations from the start.

**(c) Leaving the trust signal unexplained.** "Benchmark Score: 100" with no
published methodology decides which docs reach production agents. They got away
with it. CRED probably cannot: the same opacity applied to *an organization's
own beliefs* is a different proposition, and explaining the trust score is
nearly free if the mechanism is evidence links rather than a model. This is
D-005's auditability axis, correctly demoted by D-006 as a wedge, but cheap
enough to keep as a property.

**(d) Letting the README's install path drift to OAuth-first.** The zero-key
path still works and is still declared `isRequired: false` — but the README now
leads with a command that authenticates. A new user's first documented step is
now a browser round-trip that did not exist during the growth period. CRED
should keep the frictionless path as the *documented default*, not merely as a
surviving capability.

---

## Unverified items

Listed so nobody assumes they were checked.

- ⚠️ **How Source Reputation and Benchmark Score are computed.** Not in the
  repo, not in the docs. Unverifiable from outside. Would need Upstash to
  publish.
- ⚠️ **Whether the injection classifier works.** No published eval, no FN rate,
  no red-team result. Only the docs' assertion.
- ⚠️ **The "51 MCP clients" figure** — from Context7's own docs page. 8 verified
  in-tree (`docs/clients/`).
- ⚠️ **Pricing** — from the vendor pricing page, fetched once. Not cross-checked
  against an invoice or third party.
- ⚠️ **Which channel actually drove installs.** All channels are verified to
  exist; their relative contribution is not measurable from outside. The claim
  that no single channel dominated is inference from the absence of any
  high-scoring launch, not a measurement.
- ⚠️ **Stars-over-time.** Not retrieved; the GitHub API gives only a current
  total. The "~15,000 stars when the API key landed" figure in the install
  section is an **estimate** from commit-date position, not a measurement. Would
  need a star-history service to confirm.
- ⚠️ **Whether Upstash intends to move into private organizational memory.**
  Inference from the private-repo, on-prem and GitOps features shipping
  June–July 2026. No stated roadmap was found.
- ⚠️ **Revenue.** None disclosed, consistent with every other project in
  [market-landscape.md](market-landscape.md).
- **Reddit** was not searched (blocked in this environment, as in prior scans).
  X/Twitter launch activity was not retrieved; the claim that the launch was
  quiet rests on HN only.
