# The three retreats — Mem0, Letta, Zep

- **Date:** 2026-07-20
- **Status:** Evidence document
- **Extends and corrects:**
  [why-survivors-survive.md](why-survivors-survive.md) §4
- **Bears on:** D-003 (governance friction), D-004 risk #5, D-012, D-013

## Purpose

[why-survivors-survive.md](why-survivors-survive.md) §4 recorded that three
funded companies retreated from the governed/team version of agent memory
inside twelve months — Mem0 removed `org_id`/`project_id`, Letta tore out its
memory server, Zep killed self-hosting. D-013 names this the most serious open
tension in the research: it is simultaneously the vacant gap CRED targets and
evidence from three funded teams that the gap is hard or unwanted.

This document asks **why each retreat actually happened**, from code, PRs,
issues and download series rather than from changelogs and blog posts.

The primary hypothesis under test, stated by the founder, is that the three
**did not abandon** the governed/team capability — they **moved it out of OSS
into the paid product**. Because that hypothesis favours CRED, it was held to a
higher standard than the alternatives, not a lower one.

**Three of the four load-bearing claims in §4 do not survive.** The corrections
are recorded in full below rather than quietly applied.

---

## Verdict

| Company | Verdict | The artifact that decides it |
|---|---|---|
| **Mem0** | **Monetized, and actively defended** | `client.project` ships in OSS `mem0ai` 2.0.12 with `add_member(email, role="READER"\|"OWNER")` against a live hosted org/project API — while PR [#4590](https://github.com/mem0ai/mem0/pull/4590), which would have put org scoping in the OSS *engine*, was closed by a maintainer with "we're actively deprecating `org_id` across the SDK" |
| **Letta** | **Monetized** (memory server); **causal story FALSIFIED** (download fall) | `letta/server/rest_api/routers/v1/git_http.py:264-269` returns HTTP 501 `"git HTTP requires memfs service (LETTA_MEMFS_SERVICE_URL not configured)"` — the client is open, the service is not in the repo |
| **Zep** | **Cannot determine** | The deciding artifact does not exist: BYOC appears in the pricing table and nowhere in 265 documentation pages. The two measurements that would settle motive are both unrecoverable |

**Two of three retreats are monetization, not abandonment.** That is the better
answer for CRED. But it arrives with a cost attached, and §5 states it.

**D-003's friction hypothesis is not confirmed and not falsified — it is
mis-scoped.** The evidence supports a narrower and more actionable mechanism,
stated in §4.

---

## 1. What was checked, and what could not be

**Checked directly:**

- **Full git history.** `/Users/canh/Solo/OSS/{mem0,letta}` are shallow (1
  commit each), so fresh full clones were made into the scratchpad:
  `mem0ai/mem0` (2,493 commits, HEAD `8a57967`, 2026-07-20) and
  `letta-ai/letta` (7,467 commits, HEAD `b76da90`, 2026-07-03). The user's
  clones were not touched.
- **The removal commit, diffed.** `e44b46ef` in mem0, both the pre- and
  post-image of `mem0/client/main.py`.
- **The runtime, empirically.** `pip install "mem0ai>=2.0.0"` into a clean
  venv, then `inspect.signature` and a live constructor call.
- **GitHub issues and PRs** via `gh api search/issues` and
  `gh api repos/mem0ai/mem0/issues/{n}/comments`, including full comment
  threads on the two decisive items.
- **GitHub code search** via `gh api search/code` for real usage of the removed
  parameters, with the result repositories sampled by name rather than counted
  blind.
- **PyPI daily download series** (not monthly aggregates) and the
  `python_minor` breakdown for `letta-client`.
- **Live hosted API reference pages** at `docs.mem0.ai`, fetched.
- **A build.** `getzep/zep`'s `legacy/src` compiles today under
  `GOWORK=off go build`.

**Could not be checked, and why:**

- **Zep's Community Edition issue history.** `gh api repos/getzep/zep` returns
  `"has_issues": false`; `search/issues` for the repo returns `total_count: 0`.
  The repo was archived **on the same day as the announcement**, then later
  un-archived and repurposed. Wayback snapshots preserve only the *counts*
  (11 open / 115 closed on 2025-01-31; 0 open / 128 closed on 2025-04-26),
  not the threads. **This matters for interpretation: the absence of
  complaints about Zep is an artifact of archival, not a measurement.**
- **`zepai/zep` Docker pull count.** The image was deleted —
  `hub.docker.com/v2/repositories/zepai/zep/` returns 404 — and Wayback has no
  snapshot of the page. This was the single best available adoption metric for
  Zep CE and it is gone.
- **Letta's Discord.** The March 2026 post directed migration questions to
  Discord and office hours. Discord was not searched. GitHub silence is
  therefore weak evidence about Letta.
- **Letta's public commit dates.** PRs #9257 and #9315, cited in commit
  messages, **404 against `letta-ai/letta`** while #3393 resolves. The
  four-digit-higher numbers come from a private monorepo, and many commits
  share an identical `2026-02-24 10:52:06` timestamp — a bulk squashed import.
  **Treat OSS commit dates for Letta as import dates, not authorship dates.**
- **Whether Letta's App Server is open source.** Unverified.
- **Whether Mem0's hosted org/project endpoints work against a live key.** The
  API reference is documented and undeprecated; no call was made with real
  credentials.

---

## 2. Mem0 — monetized, and actively defended

### The artifact

**Commit `e44b46ef2ea559953190d0bfb5a58bf10ad799bb`**, authored by Kartik
(`kartik.labhshetwar@mem0.ai`), **2026-04-12**, merging
[PR #4740](https://github.com/mem0ai/mem0/pull/4740). Commit subject, verbatim:

> `fix(sdk): removing deprecating param from our sdk and docs changes with it (#4740)`

The diff touches 54+ files across docs, the Python client, and the TypeScript
client. On `mem0/client/main.py` the change is small and exact:

```diff
     def __init__(
         self,
         api_key: Optional[str] = None,
         host: Optional[str] = None,
-        org_id: Optional[str] = None,
-        project_id: Optional[str] = None,
         client: Optional[httpx.Client] = None,
     ):
-        self.org_id = org_id
-        self.project_id = project_id
+        self.org_id = None
+        self.project_id = None
```

### The first correction: this was not a governance-specific removal

`docs/migration/platform-v2-to-v3.mdx:218` lists the **fifteen** parameters
PR #4740 removed:

> **Removed parameters:** `org_id`, `project_id`, `api_version`,
> `output_format`, `async_mode`, `enable_graph`, `immutable`,
> `filter_memories`, `batch_size`, `force_add_only`, `includes`, `excludes`,
> `keyword_search`, `org_name`, `project_name`

The same PR introduced typed Pydantic option classes (`AddMemoryOptions`,
`SearchMemoryOptions`, …) per `docs/changelog/sdk.mdx:272`. **PR #4740 is an
SDK-surface cleanup that replaced loose `**kwargs` with typed options**, and
`org_id` was one of fifteen casualties, sitting next to `output_format` and
`batch_size`. §4 of `why-survivors-survive.md` read it as a targeted retreat
from organizational scoping. It was not.

### The second correction: `/v1/ping/` already resolved org and project

The pre-removal file (`git show e44b46ef^:mem0/client/main.py`) and HEAD are
**byte-identical** in `_validate_api_key`:

```python
response = self.client.get("/v1/ping/", params=params)
data = response.json()
if data.get("org_id") and data.get("project_id"):
    self.org_id = data.get("org_id")
    self.project_id = data.get("project_id")
```

Server-side resolution from the API key was already there and is unchanged.
What was removed is the **caller-supplied override** — the ability for a client
to *declare* which org and project it is in. The published docs describe exactly
this ([docs.mem0.ai/platform/features/organizations-projects](https://docs.mem0.ai/platform/features/organizations-projects),
fetched 2026-07-20):

> "organization and project are resolved automatically from your API key via
> `/v1/ping/`: no org or project params are accepted by
> `MemoryClient.__init__`. Use a project-specific API key to target a
> particular project."

**Read against [mem0.md](mem0.md) §5, this is a security fix, not a retreat.**
mem0.md's central criticism of Mem0's REST server is that it authenticates
properly and then lets the caller name whose memories they are touching
(`server/main.py:366-374`). Removing the constructor override moves tenancy
from caller-asserted to credential-derived. It is the direction CRED wants.

### What replaced it: the full team layer, still shipping in OSS

`mem0/client/project.py` at HEAD, verified live in the installed package
(`mem0ai 2.0.12`):

```
Project methods: ['add_member', 'create', 'delete', 'get', 'get_members',
                  'org_id', 'project_id', 'remove_member', 'update',
                  'update_member', 'user_email']
```

`project.py:506-529` — role-based membership, with validation:

```python
def add_member(self, email: str, role: str = "READER") -> Dict[str, Any]:
    """role: Role to assign ("READER" or "OWNER")"""
    if role not in ["READER", "OWNER"]:
        raise ValueError("Role must be either 'READER' or 'OWNER'")
    response = self._client.post(
        f"/api/v1/orgs/organizations/{self.config.org_id}/projects/{self.config.project_id}/members/",
        json=payload,
    )
```

The hosted surface behind it, from `docs/openapi.json` at HEAD (33 paths
total, 7 org/project-related):

```
/api/v1/orgs/organizations/                                    GET POST
/api/v1/orgs/organizations/{org_id}/                           GET DELETE
/api/v1/orgs/organizations/{org_id}/members/                   GET PUT POST DELETE
/api/v1/orgs/organizations/{org_id}/projects/                  GET POST
/api/v1/orgs/organizations/{org_id}/projects/{project_id}/     GET PATCH DELETE
/api/v1/orgs/organizations/{org_id}/projects/{project_id}/members/  GET POST PUT DELETE
/api/v1/webhooks/projects/{project_id}/                        GET POST
```

Fifteen API-reference pages exist at HEAD for these
(`docs/api-reference/organization/*.mdx` ×6,
`docs/api-reference/project/*.mdx` ×9), and the live page
[docs.mem0.ai/api-reference/project/get-projects](https://docs.mem0.ai/api-reference/project/get-projects)
documents `GET /api/v1/orgs/organizations/{org_id}/projects/` with **no
deprecation notice** (fetched 2026-07-20).

This is not marketing copy. It is a documented API with a shipped, typed,
role-validating client in the open-source package. **Mem0's team layer was not
abandoned. It was moved server-side, where it can only be reached with a paid
API key.**

### The empirical check

```
$ pip install "mem0ai>=2.0.0"
$ python -c "..."
mem0ai version: 2.0.12
sig: (self, api_key=None, host=None, client=None)
TypeError: MemoryClient.__init__() got an unexpected keyword argument 'org_id'
Project methods: ['add_member', 'create', ... 'update_member', 'user_email']
Memory.add sig has org?: False
```

The last line is the boundary: the OSS `Memory` engine has **no** org concept
at all, exactly as [mem0.md](mem0.md) §1 found. Org/project exists only on the
hosted path.

### Adoption before removal — real, and the demand was refused on the record

**This is the decisive evidence, and it cuts against "the feature was
unused."**

`gh api search/code`, 2026-07-20:

| Query | Hits |
|---|---|
| `"MemoryClient(" "org_id=" language:Python` | **225** |
| `"MemoryClient(" "project_id=" language:Python` | **282** |
| `"from mem0 import MemoryClient" language:Python` | 1,318 |
| `"client.project.add_member"` | 154 |

Sampling the `org_id=` results by repository name rather than trusting the
count: `run-llama/llama_index`, `crewAIInc/crewAI` (via
`src/crewai/memory/storage/mem0_storage.py`, appearing in several vendored
copies), `MervinPraison/PraisonAI`, `Upsonic/Upsonic`,
`supermemoryai/supermemory`, `google-gemini/workshops`, plus ~20 application
repos. **Roughly one in six `MemoryClient` users passed `org_id`.** These are
the framework-vendoring integrations D-013 identifies as *the* distribution
channel.

Then the two artifacts that settle it. On **2026-03-28** — two weeks before the
removal — user `Mukhsin0508` filed
[issue #4589](https://github.com/mem0ai/mem0/issues/4589) and
[PR #4590](https://github.com/mem0ai/mem0/pull/4590), asking for org-scoped
multi-tenant search in the **OSS engine**, with a detailed diagnosis
(`_build_filters_and_metadata` injects `user_id` as a hard `must` filter, so
`OR` clauses cannot reach org-wide memories) and a working patch.

The PR got substantive review — `SaharshPatel24` correctly flagged that adding
`org_id` to `get_all()` would let one call dump every memory in an org with no
access control, and the author agreed and narrowed the scope. Then, on
**2026-04-10**, maintainer `kartik-mem0` closed both. Verbatim:

> "Hi, thanks for the detailed writeup the use case makes sense. However,
> **we're actively deprecating `org_id` across the SDK and plugins, so adding
> it as a new first-class scope parameter would conflict with that
> direction.**
>
> For multi-tenant search across namespaces, the recommended approach is
> parallel searches with separate `user_id` scopes as you're already doing.
> **Closing this as it doesn't align with the current SDK direction.**"

The reporter's reply, verbatim:

> "Deprecating org_id is the first step to not supporting multi-tenancy. And
> regarding doing the multiple parallel separate searches on user_id is not
> cost efficient. The company will be paying for each embeddings (same search
> text) on the number of users they have inside the organization.
>
> **I think we will directly search with the Qdrant, resulting in not using
> Mem0 client.search() endpoint.**"

And earlier, before the close:

> "@SaharshPatel24 anything on this yet? **it's just our prod is already
> depending on this company-wide knowledge search**... would appreciate if we
> could publish it in the next release."

So: a production user asked for org-scoped retrieval in OSS, wrote the patch,
took review feedback, was closed as off-direction two weeks before the
parameter was deleted, and left for raw Qdrant. **The team primitive was not
removed for want of demand. Demand was declined.**

### One loose end worth recording: the flagship integration is broken

`run-llama/llama_index` shipped
[PR #21711](https://github.com/run-llama/llama_index/pull/21711), "Upgrade mem0
to 2.x", on **2026-05-20**, and `llama-index-memory-mem0` now pins
`mem0ai>=2.0.0,<3.0.0` (`pyproject.toml:37`). But `base.py` on `main` still
reads:

```python
def from_client(cls, context, api_key=None, host=None,
                org_id: Optional[str] = None,
                project_id: Optional[str] = None, ...):
    client = MemoryClient(
        api_key=api_key, host=host, org_id=org_id, project_id=project_id
    )
```

Against the pinned dependency, this raises
`TypeError: MemoryClient.__init__() got an unexpected keyword argument 'org_id'`
— verified empirically above. **The primary constructor path of Mem0's
flagship vendored integration has been broken for two months and nobody has
filed it.** Real usage of the *org* path, as distinct from checked-in code
containing it, is therefore near zero.

Both things are true at once, and neither cancels the other: **a small number
of users wanted org scoping badly enough to write patches, and the aggregate
usage was low enough that breaking it produced no visible bug reports.**

### Verdict — Mem0

**Monetized, and actively defended.** The org/project/member layer, with
READER/OWNER roles, is documented, undeprecated, and reachable from the
open-source client — but only against a paid key. The OSS engine has no org
concept, and a community patch that would have given it one was closed with an
explicit statement of direction. Mem0 did not find governance too hard or too
unwanted. It found it too valuable to give away.

---

## 3. Letta — monetized memory service, and a download fall that never happened

### The download fall is FALSIFIED as a causal story

`why-survivors-survive.md` verdict #6 reads: "`letta-client` on PyPI peaked at
**203k in May 2026** and fell to **90k in June** — a 56% single-month drop,
following a March 2026 announcement sunsetting server-side memory." The
monthly figures are correct (`pypistats.org/api/packages/letta-client/overall`:
2026-05 = 203,783; 2026-06 = 90,237). **The inference is wrong.**

The daily series shows June is not a fall — it is the baseline:

| Window | Daily downloads |
|---|---|
| 2026-04-01 → 04-24 | ~2,777 median |
| **2026-04-25 → 05-14** | **burst, peak 26,397 on 05-04, 259,952 total** |
| 2026-06 | ~2,970 median |

31 × ~2,970 ≈ 92k, which is June. Strip the burst from May and the residual is
48,370 — **June is higher than the pre-burst baseline, not lower.**

The `python_minor` endpoint corroborates the mechanism: the entire delta is
Python 3.12 (May 156,095 → June 40,666) while 3.10, 3.11, 3.13 and 3.14 are
flat. A user exodus drains interpreter versions proportionally; a single CI or
mirror environment does not. This is automated traffic appearing and stopping.

**The timeline kills causation independently.** The announcement is
**2026-03-16**. The burst *began* 2026-04-25, six weeks later, and decayed on
its own. Nothing in June corresponds to any Letta event.

**The rename check also comes back negative, and for a different reason than
expected.** No successor package absorbed the traffic: `letta` on PyPI went
2026-05 107,838 → 06 81,707 → **07 133,778, rising**; `letta-agent-sdk`,
`letta-core` and `lettaai` all 404. There *was* a real product migration — the
V1 SDK froze on 2026-06-02 (`letta-client` 1.12.1 and npm
`@letta-ai/letta-client`, same date) while `@letta-ai/letta-code` shipped
0.28.12 on 2026-07-20 — but that migration does not explain a May→June delta
that is a burst artifact.

**Correction to `why-survivors-survive.md`:** verdict #6 and §2.3's "203k May →
90k June ✅" should not be cited as evidence that Letta is shrinking on the
memory axis. The number is real; the story attached to it is not. Recorded
here rather than deleted so nobody re-derives it.

### "Tore out its memory server" is the wrong description

Memory was not removed. It was **replaced in-repo** by a git-backed
implementation whose *service half* lives outside the repo — and the
replacement landed **before** the announcement:

| Commit | Date | What |
|---|---|---|
| `50a60c139` | 2026-02-24 | `feat: git smart HTTP for agent memory repos (#9257)` — adds `block_manager_git.py`, `memory_repo/`, `git_http.py` |
| `0bdd555f3` | 2026-02-24 | `feat: add memfs-py service (#9315)` |
| `a50482e6d` | 2026-03-03 | `feat(core): sync skills from SKILL.md into memFS blocks (#9718)` |
| `b76da9092` | 2026-07-03 | AGENTS.md deprecation notice (#3393) |

*(Caveat from §1: #9257 and #9315 404 against the public repo. These are
private-monorepo PR numbers imported in bulk. Dates are import dates.)*

The old memory surface still ships in OSS: `/v1/blocks/*`,
`/v1/agents/{id}/core-memory/*`, `/v1/archives/*` are all implemented under
`letta/server/rest_api/routers/v1/`. And `letta/client/__init__.py` being 0
bytes is **not** a deprecation signal — it has been empty since `8ae1e6498`
(2024-09-23).

### The OSS/cloud boundary, measured

`letta.md` §7 records "239 paths / 302 operations" in `fern/openapi.json`
against "33 v1 routers". **That comparison is a unit mismatch** — 33 counts
router *files*, 239 counts *paths*. Counting `@router` decorators across all
OSS routers gives **194 OSS paths against 239 spec paths → 62 cloud-only**,
about a third of the assumed gap.

The cloud-only paths that are memory- or governance-relevant:

- **`/v1/agents/{id}/memory-files/{content,directory,history}`** — the read API
  for the new git-backed memory
- `/v1/agents/{id}/core-memory/variables`
- The non-internal **`/v1/templates/*`** family, ~20 paths — governance
- Platform: `/v1/projects`, `/v1/pipelines/*`, `/v1/feeds/*`, `/v1/sandboxes`,
  `/v1/environments/*`, `/v1/client-side-access-tokens`

Note `_internal_templates`, `_internal_blocks` and `_internal_agents` *are*
in OSS — so the split is not "templates are closed", it is "the customer-facing
template API is closed".

### The deciding artifact

`letta/server/rest_api/routers/v1/git_http.py:264-269`:

```
501: "git HTTP requires memfs service (LETTA_MEMFS_SERVICE_URL not configured)"
```

Supported by `letta/settings.py:343` (`memfs_service_url`),
`letta/server/server.py:436` (returns `None` without it), and
`memfs_client_base.py` — a 388-line `MemfsClient` that is an HTTP client to a
service **not present in this repository**.

**This is the pattern the primary hypothesis predicted: the client is open, the
service is closed, and the boundary is a config variable.** It is not marketing
copy — the 501 is executable, and the client that talks past it is 388 lines of
real code.

### Vestigial or real? Half the prior holds

Checked against `letta.md`'s finding that Letta built provenance machinery and
never wired it:

| Symbol | Finding |
|---|---|
| `checkpoint_block_async` | **Prior holds.** 1 definition (`block_manager.py:842`), **35 call sites in `tests/`, zero in production `letta/`** |
| `BlockHistory` | **Partly wrong.** 37 references in `letta/` — genuinely wired through the ORM and `block_manager.py`. Its checkpoint entry point has no production caller, but the table is live |
| `read_only` | **Prior is wrong.** 55 references, **actively enforced** at `letta/agent.py:191-196` (`ensure_read_only_block_not_modified`), called at `agent.py:1623`. The `deprecated=True` at `block.py:103` is on `BlockResponse` — the wire schema — while the live field at `block.py:36` on `BaseBlock` is undeprecated |

**Correction to [letta.md](letta.md) §2 and its "Top 3 NOT to copy" #3:**
Letta is *not* removing `read_only` from the engine. It deprecated it in the
API response schema while continuing to enforce it in the agent loop. The
"direction of travel is away from governance primitives" reading is too strong.

And the hosted memfs service is **not** vestigial in the way Mem0's reconciler
is. `letta-ai/letta-code`
[issue #3011](https://github.com/letta-ai/letta-code/issues/3011)
(2026-06-20, open) is a production 400 from `api.letta.com` —
`column "name" of relation "hosted_memfs_repositories" does not exist` — with
the reporter noting *"The parent agent's MemFS works perfectly — read, write,
commit, push all succeed."* A real user, a real hosted table, real call sites.

### Adoption before removal — near-zero objection

`gh` search on `letta-ai/letta`, issues created after 2026-02-01:
**"deprecat" in title: 0. "sunset": 0. "tool rules": 0.** A body-wide search
for deprecated/sunset/removed since March returns 14 results, all ordinary
bugs. Issue volume is flat across the event: Jan 24, Feb 19, **Mar 41, Apr
51**, May 26, Jun 27, Jul 7 — the Mar/Apr bump is activity, not revolt.

Nobody objected to losing Letta Filesystem, Templates, Identities, or tool
rules. Meanwhile `letta-ai/letta-code` carries **112** memory/migration issues:
the users did not leave, they migrated. `letta-ai/letta` still holds 23,880
stars and is **not archived**.

The caveat from §1 stands: the announcement directed complaints to Discord,
which was not searched.

### Verdict — Letta

**(a) The memory server: monetized.** The git-backed memory client is open;
the memfs service is closed and gated on `LETTA_MEMFS_SERVICE_URL`. It is
demonstrably live in the hosted product.

**(b) The download fall: FALSIFIED as a consequence of the retreat.** It is a
burst artifact in a single Python 3.12 environment, six weeks after the
announcement and unrelated to it.

---

## 4. Zep — cannot determine

### The announcement

[blog.getzep.com](https://blog.getzep.com/announcing-a-new-direction-for-zeps-open-source-strategy/),
`datePublished: 2025-04-02T19:25:59Z`, signed by Daniel Chalef. Same-day
commit `6a9ce060ea49b47c0931aa495ac775b93f358129` adds the deprecation notice
to the README. Stated rationale, verbatim:

> "Managing two related but different products…has presented real challenges…
> **leading us to under-invest in the open-source version.**"
>
> "We also felt uneasy about the common practice in open-core of
> **intentionally limiting features to drive users toward paid products.**"

Note also, verbatim: Graphiti "**powers Zep's commercial service**… Graphiti
doesn't compete directly with Zep's memory service."

### Repo state today

`getzep/zep` is **not archived** (it was archived 2025-04-02 and later
un-archived and repurposed), 4,766 stars, 642 forks, `pushed_at` 2026-07-17,
description now "Zep | Examples, Integrations, & More".

- `f5f56a0f97da3af2977bec2c5cf80da5776919dc`, 2025-06-29, "reorg (#394)" —
  creates `legacy/`, 110 files, +8,681 lines
- `c366ead`, 2025-06-29, "deletions (#395)" — 105 files, **8,630 deletions**,
  removing the server from the repo root

**The server still compiles.** `GOWORK=off go build` in `legacy/src` (Go
1.26.0, darwin/arm64) produces a 20,629,938-byte binary. `go.work` is broken
and must be bypassed. Dependabot has kept `legacy/src` patched as recently as
`9b3545e` (2026-06-18).

**But the deployment path is gone.** `legacy/docker-compose.ce.yaml` pins
`image: zepai/zep:latest`, and
`hub.docker.com/v2/repositories/zepai/zep/` returns **404**. The `zepai` org
today holds only `graphiti`, `knowledge-graph-mcp` and `lark`. **The image was
deleted, not left stale** — an affirmative act going beyond the announcement's
promise of merely "no updates or active support."

### Applying the artifact test to the successor

Zep's [pricing page](https://www.getzep.com/pricing/) advertises Cloud, BYOK,
and **BYOC** ("Zep deployed inside your VPC. Your network, your perimeter,
your compliance boundary"), with Enterprise listed as "Cloud · Cloud + BYOK ·
BYOC".

- **BYOK is real.** `help.getzep.com/bring-your-own-key.md` is a genuine
  deployment doc with `aws kms create-key` invocations, Terraform
  `aws_kms_key` blocks and cross-account grant setup. **But BYOK is
  explicitly not self-hosting** — the page describes "Zep's managed service
  with encryption keys you control." Compute and data stay in Zep Cloud.
- **BYOC is a pricing bullet and nothing else.** `help.getzep.com/sitemap.xml`
  contains **265 URLs**; grepping all of them for
  `byo|deploy|vpc|prem|install|infra|helm|docker|kubernetes` returns **zero
  matches**. `help.getzep.com/llms.txt` — the docs' own AI index — returns 0
  matches for `byoc|self.?host|on.?prem|helm|terraform|kubernetes|docker`. The
  BYOK page's own BYOC link goes to `getzep.com/enterprise`, whose BYOC tile's
  only call to action is "Talk to Sales →".

No image, no Helm chart, no Terraform module, no deployment guide, no API
reference. **This is precisely the failure mode the brief named: it looks
confirming and is not.** It is the same shape as Memco's on-prem claim, which
this project accepted from a marketing line and had to retract.

### Adoption before removal

Stars 2.9k → 3.2k and forks 420 → 474 across Jan–Apr 2025; 11 open / 128
closed issues lifetime; **25 watchers**. Low, for a self-hosted server product.
One pre-announcement signal survives in the Wayback issue list: **#384, "is ZEP
dead (or frozen)? We see no commits, no activity anymore."**, opened
2024-12-18 by `Morriz` — corroborating the announcement's own "under-invest"
admission three and a half months early.

The deprecation was **never submitted to HN** (Algolia: 0 relevant hits). The
CE launch itself scored 6 points and 0 comments.

### Did the outcome contradict the stated principle?

**Yes, verifiably, though it matured later.** The post objected to
"intentionally limiting features to drive users toward paid products" and said
Graphiti powers the commercial service. Zep's current docs say something
different: `help.getzep.com/zep-vs-graphiti.md` states Zep runs Graphiti "on
the **proprietary Context Graph Engine**", and its comparison table lists Zep's
storage as "Proprietary, highly scalable Context Graph Engine graph database"
against Graphiti's pluggable Neo4j/FalkorDB/Neptune, plus "proprietary
extraction LLMs, reranker, and embedding models." The
[Context Graph Engine page](https://www.getzep.com/platform/context-graph-engine/)
markets it as "The runtime underneath Zep."
[graphiti#1665](https://github.com/getzep/graphiti/issues/1665) (2026-07-17)
exists specifically to document the split, stating its objective is to make
clear "that Zep's managed offering does not depend on a third-party graph
database."

That comparison row — Deployment: "Self-hosted" (Graphiti) vs "Cloud / BYOK /
BYOC" (Zep) — **is the open-core split the post said it was uncomfortable
with, reconstituted one layer down.** The principle was not upheld; it was
relocated.

### Verdict — Zep

**Cannot determine.** The deciding artifact does not exist, and its absence is
the finding: BYOC has no deployment artifact anywhere in a 265-page
documentation set. That is not confirmation that self-hosting moved into the
paid product.

Motive is genuinely undetermined between abandonment and monetization because
**both decisive measurements are unrecoverable** — the `zepai/zep` pull count
(image deleted, never archived) and the post-announcement issue record
(archived, then issues disabled, same day).

"Monetized-but-unused" is explicitly **not** endorsed: "unused" is
unestablished, and the apparent absence of complaints is confounded by the
archival that made complaining impossible.

One leg of the primary hypothesis is affirmatively contradicted: **the Go
server was not moved into the paid product in any form.** It was frozen in
`legacy/`, its image deleted, and the commercial runtime rebuilt as the closed
Context Graph Engine, which is not CE's descendant.

---

## 5. Hypothesis test

### D-003's mechanism, as written

> "governance adds friction (approval, review, versioning) exactly when an
> individual user wants speed."

**Verdict: not confirmed, not falsified — mis-scoped.**

D-003's mechanism is about *user* behaviour: individuals reject governance
because it slows them down. Nothing in this scan shows a user rejecting
governance for speed. What it shows is **vendors withdrawing governance from
the free tier**, which is a *seller* decision, not a buyer one.

The one place a user is on the record —
[mem0#4589](https://github.com/mem0ai/mem0/issues/4589) — the user is
**demanding** org scoping, has it in production, wrote the patch, and leaves
for raw Qdrant when refused. That is the opposite of friction-aversion.

D-003's *conclusion* survives and should be kept: governance must be a free
side effect of something a developer already wants. But its stated *reason* is
not what the evidence shows.

### The alternatives, weighed

| Hypothesis | Mem0 | Letta | Zep |
|---|---|---|---|
| **Moved to paid, not abandoned** (primary) | **Supported** — hosted org/project API, 21 operations, READER/OWNER roles, undeprecated, reachable from the OSS client with a paid key | **Supported** — `git_http.py:264-269` 501s without `LETTA_MEMFS_SERVICE_URL`; hosted memfs verifiably live (letta-code#3011) | **Not supported** — BYOC has no artifact; the server was frozen, not relocated |
| **The feature was unused** | **Partly** — 225 code hits and one production user, but llama_index's org path broke for two months unnoticed | **Supported for the sunsets** — zero deprecation issues filed | **Cannot determine** — issue record destroyed, pull count deleted |
| **Maintenance burden** | Not supported — `project.py` is still maintained | Plausible, unevidenced | **Supported by their own words** — "leading us to under-invest in the open-source version", plus issue #384 |
| **Framework-integration conflict** | Not supported — it *broke* an integration | Not supported | Not applicable |
| **Unrelated pivot** | Not supported — #4740 is an SDK cleanup, but the org layer survives intact server-side | **Partly** — the pivot to computer-use agents is real and stated, and memory changed shape with it | Not supported |

### The refined mechanism

Three companies, one shared behaviour, and it is not friction:

**Governance is the part of a memory product that a company can charge for, so
it is the part that leaves the open tier first.** It is where multi-tenancy,
membership, roles and audit live; those are the features a buyer with a budget
needs and a solo developer does not. Mem0 kept the whole thing and put it
behind a key. Letta kept the client and closed the service. Zep — as far as
can be established — closed the runtime and gave away the extraction framework.

**The corollary matters more than the observation.** In all three cases the
part that stayed open is the *extraction and retrieval* layer, and the part
that closed is the *governance and tenancy* layer. That is exactly the inverse
of the split CRED proposes.

### The alternative that must not be collapsed

"Moved to paid" and "had almost no users" are both true of Mem0, and the
evidence separates them cleanly:

- **Demand existed and was articulate.** A production user, a detailed issue,
  a reviewed patch, and a stated intention to leave for Qdrant.
- **Aggregate usage was low.** llama_index's `from_client(org_id=...)` has
  raised `TypeError` since 2026-05-20 and nobody filed it.

A small number of users needed org scoping badly; the large majority did not
notice. For CRED that is neither the good news nor the bad news alone — §6
states what follows.

---

## 6. What CRED must do differently

### 1. The vacancy is a *pricing* vacancy, not a capability one

The three did not fail at governance. Two of them **succeeded and charged for
it**. So the gap CRED targets is not "nobody can build a team layer" — it is
"nobody offers a team layer you can run yourself."

This is a better answer than the one D-013 recorded, and it is a narrower one.
CRED's advantage is not that org-scoped memory is hard. It is that **CRED can
put the governance layer in the open tier and the three incumbents structurally
cannot**, because it is their revenue.

**The cost, named:** it is also CRED's revenue, if CRED ever wants revenue.
D-012 already defines success as a good open-source project with real users
rather than a business, which makes this affordable — but the decision is now
explicit rather than incidental. **If CRED later needs to monetize, the obvious
lever is the one thing it is currently giving away.** That should be decided
deliberately, not discovered.

### 2. Put the team layer in the OSS engine, not in a client that phones a server

This is the concrete, testable difference. In all three cases the closed half
is a *service* and the open half is a *client*:

- Mem0: `mem0/client/project.py` is open; `/api/v1/orgs/...` is closed.
  `mem0.Memory` — the engine you can actually run — has **no org concept at
  all**.
- Letta: `MemfsClient` (388 lines) is open; the memfs service is closed;
  `git_http.py` 501s.

**CRED's org, project, member and role primitives must live in the same binary
a solo developer runs with no network.** If CRED's governance ever becomes
reachable only through a hosted endpoint, it has reproduced the exact shape of
all three retreats. This is a design constraint, not a preference, and it is
checkable: *grep the engine for the principal type. If it only appears in a
client package, the retreat has happened.*

### 3. Reject caller-asserted tenancy — but do not mistake it for governance

Mem0's removal of `org_id` from the constructor is a **security improvement**
that this project initially read as a retreat. CRED should copy it: the
principal must be derived from the credential, never declared in the call.
[mem0.md](mem0.md)'s three-layer demonstration — JWT + bcrypt API keys +
refresh rotation, then `_auth` bound and never read — remains the anti-pattern.

But the reverse must also hold: **credential-derived tenancy is not by itself a
team layer.** Mem0 has credential-derived tenancy and still cannot answer
"which memories can this agent read" in the OSS engine.

### 4. Take the demand signal seriously, and take its size seriously too

[mem0#4589](https://github.com/mem0ai/mem0/issues/4589) is the strongest piece
of direct demand evidence this project has found for CRED's thesis — a named
user with the problem in production, who wrote the patch, and who left. It is
worth more than a survey.

It is also **one user**, and the surrounding evidence says most Mem0 users
never touched org scoping. Under D-012 the existence question is settled and
the design question is not, so the right response is neither to celebrate nor
discount it, but to **make the team layer free at rung 1 and invisible until
needed** — which is what D-003 already concluded, now on firmer ground.

### 5. Two method corrections that cost this project real accuracy

- **Never infer causation from a monthly download aggregate.** The Letta
  203k→90k claim survived one full review, was cited in a decision, and is a
  burst artifact visible in ten seconds of daily data. Pull daily series and
  the `python_minor` breakdown before attributing a download movement to any
  product event.
- **Never read a changelog line as a product decision without the diff.**
  "`org_id` and `project_id` removed" was one of fifteen parameters dropped in
  a typed-options refactor, next to `output_format` and `batch_size`. The
  changelog is accurate; the reading was not.

### 6. What does *not* change

D-013's distribution finding stands and is if anything reinforced — the
framework-vendored integrations are where the org parameters actually appeared
in the wild. D-011 stands: nothing here restores sovereignty as a wedge. Zep's
BYOC is a live demonstration of why marketing copy cannot be treated as
capability, which is the rule that already exists in
`.claude/rules/docs.md` §1.

---

## 7. Corrections to prior documents

Recorded on the record rather than silently applied, per
`.claude/rules/docs.md` §1.

| Document | Previous claim | Now |
|---|---|---|
| [why-survivors-survive.md](why-survivors-survive.md) verdict #6, §2.3 | `letta-client` 203k May → 90k June is evidence Letta is shrinking on the memory axis | **FALSIFIED as causation.** The numbers are right; June ≈ the pre-burst baseline, and the May figure is a Python-3.12 automation burst six weeks after the announcement |
| [why-survivors-survive.md](why-survivors-survive.md) §4, §7 | Mem0 v2.0.0 "deleted the organizational scoping primitives" | **Refined.** It removed a caller-supplied override from a client constructor, as one of fifteen parameters in a typed-options refactor. Server-side resolution was already in place and is unchanged. The org/project/member layer with roles is documented, undeprecated, and live |
| [why-survivors-survive.md](why-survivors-survive.md) §4 | Letta "tore out its server" | **Refined.** Memory was replaced in-repo by a git-backed implementation whose service half is closed. The client ships in OSS |
| [why-survivors-survive.md](why-survivors-survive.md) §7 | "the three companies that tried the governed version all retreated from it" | **Two monetized it. One cannot be determined.** None is shown to have found it too hard or unwanted |
| [letta.md](letta.md) §2, "NOT to copy" #3 | `read_only` is being deprecated; Letta is walking back its governance primitives | **Wrong.** `read_only` is enforced at `letta/agent.py:191-196`, called at `:1623`, 55 references. The `deprecated=True` is on `BlockResponse` only |
| [letta.md](letta.md) §7 | "239 paths / 302 operations" vs "33 v1 routers" is the OSS/cloud boundary | **Unit mismatch.** 33 counts router files. Counting `@router` decorators gives 194 OSS paths vs 239 spec paths → 62 cloud-only, about a third of the implied gap |
| [letta.md](letta.md) §8 | `letta/client/__init__.py` emptied is a deprecation signal | **Wrong.** Empty since `8ae1e6498`, 2024-09-23 |

`letta.md`'s finding that `checkpoint_block_async` has zero production callers
**stands** — 35 test call sites, zero in `letta/`. `mem0.md`'s findings on the
OSS engine (no org concept, dead reconciler, zero-writer ACL) all **stand**;
this document changes only the interpretation of the v2.0.0 client change.

---

## 8. What remains unverified

- ⚠️ **Zep's motive.** Undetermined between abandonment and monetization. The
  two measurements that would settle it — `zepai/zep` Docker pulls and the
  post-announcement issue threads — are both destroyed.
- ⚠️ **Whether Mem0's hosted org/project endpoints function.** The API
  reference is published and undeprecated; no call was made with a real key.
  Documented-and-live is established; *working* is not.
- ⚠️ **Whether Letta's App Server is open source.** AGENTS.md points to it as
  the replacement for the OSS API server. Its license was not checked.
- ⚠️ **Letta's Discord reaction** to the March 2026 sunsets. The announcement
  routed migration questions there. GitHub silence is therefore weak evidence.
- ⚠️ **Letta's commit dates.** Public history is a squashed import from a
  private monorepo; #9257 and #9315 404 against `letta-ai/letta`. Dates are
  import dates.
- ⚠️ **Real-world usage of Letta's sunset features** (Filesystem, Identities,
  tool rules, templates). No GitHub code search was run for these; only issue
  volume was checked.
- ⚠️ **Whether the 154 hits for `client.project.add_member`** are real usage or
  vendored copies of Mem0's own source and docs. The `org_id=` sample was
  inspected by repository; this one was not.
- ⚠️ **Whether anyone besides `Mukhsin0508` needed Mem0's OSS org scoping.**
  One production user is on the record. The population is unknown.
- **Reddit and X** were not searched — blocked in this environment, as in every
  prior scan.
