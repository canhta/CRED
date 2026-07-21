# Usage Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the console's `/usage` placeholder with a real page showing a principal's limit headroom, denied-contribution count, and per-scope cost/growth — all backed by data `cred usage` already computes.

**Architecture:** A new `GET /api/usage` Gin handler in `internal/api/usage.go` projects existing store queries (`internal/store/pg/usage.go`) through the existing pure decision functions (`internal/limit`) into JSON — no new engine logic. `tygo` generates the TypeScript types from the new Go structs. A new `UsagePage.tsx` renders them following the exact component conventions already established by `RecallPage.tsx` and `ClaimsPage.tsx`.

**Tech Stack:** Go 1.25, Gin, pgx (unchanged) · React 19, TanStack Query/Router, Astryx (`@astryxdesign/core`), Vitest + Testing Library.

## Global Constraints

- `CGO_ENABLED=0` for every Go build (`.claude/rules/go.md` §3).
- No hand-written TypeScript type may mirror a Go struct — regenerate with `task gen:types` (`.claude/rules/web.md` §3).
- No `<div>` for layout in React — compose Astryx components; style through component props first, tokens second, never raw hex/px (`.claude/rules/web.md` §4).
- Every API route lives once in `web/src/api/routes.ts` (`.claude/rules/web.md` §3).
- Handlers read the principal via `principalFrom(c)`, never the header directly (`.claude/rules/web.md` §5).
- Comments are sparse, explain the *why*, and never cite a PRD section, decision number, or law number (`.claude/rules/go.md` §10, `.claude/rules/web.md` §7).
- No new Go test file for the handler — this codebase's existing precedent for `internal/api` (`claims.go`, `recall.go`) is frontend tests against a mocked client plus manual end-to-end verification, not per-handler Go tests. Task 4 verifies this is still true before relying on it.

---

### Task 1: Backend — `GET /api/usage`

**Files:**
- Modify: `internal/api/types.go` (add types, append to end of file)
- Create: `internal/api/usage.go`
- Modify: `internal/api/api.go:36-42` (register the route)

**Interfaces:**
- Consumes: `s.store.PrincipalWindowState`, `s.store.DeniedInWindow`, `s.store.UsageByScope`, `s.store.ScopeSizes` (all `internal/store/pg/usage.go`, unchanged); `limit.Contribution`, `limit.Cost`, `limit.RecallRate`, `limit.PruneTarget`, `limit.WindowStart` (all `internal/limit/limit.go`, unchanged); `principalFrom(c)` and `s.fail(c, err)` (`internal/api/api.go`, `internal/api/claims.go`, unchanged).
- Produces: `UsageResponse`, `LimitStatus`, `ScopeCost`, `ScopeGrowth`, `UsageQuery` types in package `api`; route `GET /api/usage?scopes=N`. Task 2 consumes these type names via `tygo` generation — do not rename them after this task.

- [ ] **Step 1: Add the response types**

Append to `internal/api/types.go`:

```go

// UsageQuery is the query string of GET /api/usage.
type UsageQuery struct {
	Scopes int `json:"scopes" form:"scopes"`
}

// LimitStatus is one limit's window state: what has been used, the
// configured ceiling, and the remaining headroom before the control binds.
// Remaining reuses internal/limit.Decision's own sentinel (-1 means
// unlimited) and Ceiling <= 0 means disabled, rather than the API inventing
// a second "off"/"unlimited" convention on top of the one internal/limit
// already has — the frontend formats both, once, the same way
// `cred usage` already does in the terminal.
type LimitStatus struct {
	Window    string `json:"window"`
	Used      int    `json:"used"`
	Ceiling   int    `json:"ceiling"`
	Remaining int    `json:"remaining"`
	Allowed   bool   `json:"allowed"`
	Reason    string `json:"reason"`
}

// ScopeCost is one scope's inference cost since the report's cutoff — "which
// teams actually use this", the same report `cred usage` prints.
type ScopeCost struct {
	Scope        Scope `json:"scope"`
	Calls        int   `json:"calls"`
	InputTokens  int   `json:"input_tokens"`
	OutputTokens int   `json:"output_tokens"`
}

// ScopeGrowth is one scope's live-claim count against the growth ceiling, and
// how many claims the next prune pass would close.
type ScopeGrowth struct {
	Scope     Scope `json:"scope"`
	Live      int   `json:"live"`
	Ceiling   int   `json:"ceiling"`
	NextPrune int   `json:"next_prune"`
}

// UsageResponse is the body of GET /api/usage: the calling principal's limit
// headroom, its denied-contribution count, and the org-wide cost/growth
// report — the same counters and the same internal/limit decisions
// `cred usage` prints, so the console never shows a number the enforcement
// path didn't also compute.
type UsageResponse struct {
	Principal          string        `json:"principal"`
	Contribution        LimitStatus  `json:"contribution"`
	Cost                 LimitStatus `json:"cost"`
	InputTokensUsed      int         `json:"input_tokens_used"`
	InputTokensCeiling   int         `json:"input_tokens_ceiling"`
	Recall               LimitStatus `json:"recall"`
	DeniedWindow         string      `json:"denied_window"`
	Denied               int         `json:"denied"`
	CostByScope          []ScopeCost  `json:"cost_by_scope"`
	ScopeGrowth          []ScopeGrowth `json:"scope_growth"`
}
```

- [ ] **Step 2: Write the handler**

Create `internal/api/usage.go`:

```go
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/canhta/cred/internal/limit"
	"github.com/canhta/cred/internal/store/pg"
)

// usage reports the calling principal's limit headroom, its denied-
// contribution count, and the per-scope cost/growth report. Every number
// comes from the same store counts and the same internal/limit decisions the
// enforcement path uses, so the console can never show a number that
// disagrees with what a write or a recall was actually allowed to do.
func (s *server) usage(c *gin.Context) {
	var q UsageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid query parameters"})
		return
	}
	topN := q.Scopes
	if topN <= 0 {
		topN = 10
	}

	ctx := c.Request.Context()
	principal := principalFrom(c)
	now := time.Now().UTC()
	lc := s.cfg.Limits

	state, err := s.store.PrincipalWindowState(ctx, principal,
		limit.WindowStart(now, lc.ContributionWindow),
		limit.WindowStart(now, lc.CostWindow),
		limit.WindowStart(now, lc.RecallWindow))
	if err != nil {
		s.fail(c, err)
		return
	}

	denied, err := s.store.DeniedInWindow(ctx, principal, limit.WindowStart(now, lc.ContributionWindow))
	if err != nil {
		s.fail(c, err)
		return
	}

	costs, err := s.store.UsageByScope(ctx, limit.WindowStart(now, lc.CostWindow))
	if err != nil {
		s.fail(c, err)
		return
	}

	sizes, err := s.store.ScopeSizes(ctx, topN)
	if err != nil {
		s.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, usageResponse(string(principal), lc, state, denied, costs, sizes))
}

func usageResponse(principal string, lc limit.Config, state pg.ContributionState,
	denied int, costs []pg.ScopeCost, sizes []pg.ScopeSize,
) UsageResponse {
	contribution := limit.Contribution(state.Contributions, lc)
	cost := limit.Cost(state.InferenceCall, state.InputTokens, lc)
	recall := limit.RecallRate(state.Recalls, lc)

	costByScope := make([]ScopeCost, 0, len(costs))
	for _, c := range costs {
		costByScope = append(costByScope, ScopeCost{
			Scope:        Scope{Kind: string(c.Scope.Kind), Value: c.Scope.Value},
			Calls:        c.Calls,
			InputTokens:  c.InputTokens,
			OutputTokens: c.OutputTokens,
		})
	}

	scopeGrowth := make([]ScopeGrowth, 0, len(sizes))
	for _, sz := range sizes {
		scopeGrowth = append(scopeGrowth, ScopeGrowth{
			Scope:     Scope{Kind: string(sz.Scope.Kind), Value: sz.Scope.Value},
			Live:      sz.Live,
			Ceiling:   lc.ScopeClaimCeiling,
			NextPrune: limit.PruneTarget(sz.Live, lc),
		})
	}

	return UsageResponse{
		Principal: principal,
		Contribution: LimitStatus{
			Window: lc.ContributionWindow.String(), Used: state.Contributions,
			Ceiling: lc.ContributionQuota, Remaining: contribution.Remaining,
			Allowed: contribution.Allowed, Reason: string(contribution.Reason),
		},
		Cost: LimitStatus{
			Window: lc.CostWindow.String(), Used: state.InferenceCall,
			Ceiling: lc.MaxInferenceCalls, Remaining: cost.Remaining,
			Allowed: cost.Allowed, Reason: string(cost.Reason),
		},
		InputTokensUsed:    state.InputTokens,
		InputTokensCeiling: lc.MaxInputTokens,
		Recall: LimitStatus{
			Window: lc.RecallWindow.String(), Used: state.Recalls,
			Ceiling: lc.RecallRate, Remaining: recall.Remaining,
			Allowed: recall.Allowed, Reason: string(recall.Reason),
		},
		DeniedWindow: lc.ContributionWindow.String(),
		Denied:       denied,
		CostByScope:  costByScope,
		ScopeGrowth:  scopeGrowth,
	}
}
```

- [ ] **Step 3: Register the route**

In `internal/api/api.go`, change:

```go
	api := r.Group("/api")
	{
		api.GET("/health", s.health)
		api.GET("/claims", s.listClaims)
		api.GET("/claims/:id", s.getClaim)
		api.GET("/recall", s.recall)
	}
```

to:

```go
	api := r.Group("/api")
	{
		api.GET("/health", s.health)
		api.GET("/claims", s.listClaims)
		api.GET("/claims/:id", s.getClaim)
		api.GET("/recall", s.recall)
		api.GET("/usage", s.usage)
	}
```

- [ ] **Step 4: Build, vet, and run the existing test suite**

Run: `task api:build`
Expected: exits 0, produces no output (a plain `go build ./...` under `CGO_ENABLED=0`).

Run: `go vet ./...`
Expected: exits 0, no output.

Run: `task api:test`
Expected: `ok` for every package, including `internal/limit`, `internal/store/pg`, and `internal/api` (no new test file yet, so `internal/api` reports `?  	github.com/canhta/cred/internal/api	[no test files]` — that is expected, not a failure; see the Global Constraints note on this codebase's existing precedent).

- [ ] **Step 5: Commit**

```bash
git add internal/api/types.go internal/api/usage.go internal/api/api.go
git commit -m "feat: add GET /api/usage

Projects the same store counts and internal/limit decisions
\`cred usage\` already prints — no new engine logic, just a new
JSON surface over it."
```

---

### Task 2: Frontend API layer

**Files:**
- Modify (generated): `web/src/api/types.gen.ts` (via `task gen:types` — do not hand-edit)
- Modify: `web/src/api/types.ts`
- Modify: `web/src/api/routes.ts`
- Modify: `web/src/api/client.ts`
- Modify: `web/src/api/hooks.ts`
- Modify: `web/src/api/index.ts`

**Interfaces:**
- Consumes: Go types from Task 1 (`UsageResponse`, `LimitStatus`, `ScopeCost`, `ScopeGrowth`), regenerated by `tygo`.
- Produces: `getUsage(params?: UsageParams): Promise<UsageResponse>` (`client.ts`), `useUsage(params?: UsageParams)` (`hooks.ts`), `queryKeys.usage(params)` (`hooks.ts`), all re-exported from `web/src/api/index.ts`. Task 3 imports `useUsage`, `queryKeys`, and the types `UsageResponse`/`LimitStatus`/`ScopeCost`/`ScopeGrowth` from `'../api'` — do not rename any of these after this task.

- [ ] **Step 1: Regenerate the TypeScript types**

Run: `task gen:types`
Expected: exits 0. `web/src/api/types.gen.ts` now contains a `// source: usage.go` section with exported `UsageQuery`, `LimitStatus`, `ScopeCost`, `ScopeGrowth`, and `UsageResponse` interfaces mirroring the Go structs field-for-field (camelCase Go names, but JSON tag names — e.g. `input_tokens_used` — since tygo respects the `json:` tag).

Verify: `grep -c "interface UsageResponse" web/src/api/types.gen.ts`
Expected: `1`

- [ ] **Step 2: Re-export the new types**

In `web/src/api/types.ts`, change:

```ts
export type {
  ErrorResponse,
  HealthResponse as Health,
  Scope,
  Source,
  ClaimListItem,
  ClaimListResponse as ClaimList,
  EvidenceItem as Evidence,
  ClaimDetail,
  RecallResponse,
  RecalledClaim,
  Contribution as ArmContribution,
  RecallTimings,
} from './types.gen';
```

to:

```ts
export type {
  ErrorResponse,
  HealthResponse as Health,
  Scope,
  Source,
  ClaimListItem,
  ClaimListResponse as ClaimList,
  EvidenceItem as Evidence,
  ClaimDetail,
  RecallResponse,
  RecalledClaim,
  Contribution as ArmContribution,
  RecallTimings,
  UsageResponse,
  LimitStatus,
  ScopeCost,
  ScopeGrowth,
} from './types.gen';
```

- [ ] **Step 3: Add the route**

In `web/src/api/routes.ts`, change:

```ts
export const routes = {
  health: () => `${API_BASE}/health`,
  claims: () => `${API_BASE}/claims`,
  claim: (id: string) => `${API_BASE}/claims/${encodeURIComponent(id)}`,
  recall: () => `${API_BASE}/recall`,
} as const;
```

to:

```ts
export const routes = {
  health: () => `${API_BASE}/health`,
  claims: () => `${API_BASE}/claims`,
  claim: (id: string) => `${API_BASE}/claims/${encodeURIComponent(id)}`,
  recall: () => `${API_BASE}/recall`,
  usage: () => `${API_BASE}/usage`,
} as const;
```

- [ ] **Step 4: Add the client function**

In `web/src/api/client.ts`, change the type import:

```ts
import type { ClaimDetail, ClaimList, Health, RecallResponse } from './types';
```

to:

```ts
import type {
  ClaimDetail,
  ClaimList,
  Health,
  RecallResponse,
  UsageResponse,
} from './types';
```

Then append to the end of the file:

```ts

export interface UsageParams {
  scopes?: number;
}

export function getUsage(params: UsageParams = {}): Promise<UsageResponse> {
  const query = new URLSearchParams();
  if (params.scopes !== undefined) query.set('scopes', String(params.scopes));

  const qs = query.toString();
  return request<UsageResponse>(qs ? `${routes.usage()}?${qs}` : routes.usage());
}
```

- [ ] **Step 5: Add the hook**

In `web/src/api/hooks.ts`, change:

```ts
import { useQuery } from '@tanstack/react-query';
import { getClaim, getClaims, getHealth, getRecall } from './client';
import type { ClaimsParams, RecallParams } from './client';

export const queryKeys = {
  health: ['health'] as const,
  claims: (params: ClaimsParams) => ['claims', params] as const,
  claim: (id: string) => ['claim', id] as const,
  recall: (params: RecallParams) => ['recall', params] as const,
};
```

to:

```ts
import { useQuery } from '@tanstack/react-query';
import { getClaim, getClaims, getHealth, getRecall, getUsage } from './client';
import type { ClaimsParams, RecallParams, UsageParams } from './client';

export const queryKeys = {
  health: ['health'] as const,
  claims: (params: ClaimsParams) => ['claims', params] as const,
  claim: (id: string) => ['claim', id] as const,
  recall: (params: RecallParams) => ['recall', params] as const,
  usage: (params: UsageParams) => ['usage', params] as const,
};
```

Then append to the end of the file:

```ts

export function useUsage(params: UsageParams = {}) {
  return useQuery({
    queryKey: queryKeys.usage(params),
    queryFn: () => getUsage(params),
  });
}
```

- [ ] **Step 6: Re-export from the API barrel**

In `web/src/api/index.ts`, change:

```ts
export * from './types';
export { routes, API_BASE } from './routes';
export {
  ApiError,
  getHealth,
  getClaims,
  getClaim,
  getRecall,
  setPrincipal,
  getPrincipal,
} from './client';
export type { ClaimsParams, StatusFilter, RecallParams } from './client';
export { useHealth, useClaims, useClaim, useRecall, queryKeys } from './hooks';
```

to:

```ts
export * from './types';
export { routes, API_BASE } from './routes';
export {
  ApiError,
  getHealth,
  getClaims,
  getClaim,
  getRecall,
  getUsage,
  setPrincipal,
  getPrincipal,
} from './client';
export type {
  ClaimsParams,
  StatusFilter,
  RecallParams,
  UsageParams,
} from './client';
export {
  useHealth,
  useClaims,
  useClaim,
  useRecall,
  useUsage,
  queryKeys,
} from './hooks';
```

- [ ] **Step 7: Typecheck**

Run: `cd web && npm run typecheck`
Expected: exits 0, no output (this catches any mismatch between the generated types and the hand-written client/hooks code before the page even exists).

- [ ] **Step 8: Commit**

```bash
git add web/src/api/
git commit -m "feat: wire GET /api/usage into the frontend API layer

getUsage/useUsage follow the same routes.ts -> client.ts -> hooks.ts
-> index.ts path as recall and claims. Types regenerated with
task gen:types, not hand-written."
```

---

### Task 3: `UsagePage` component

**Files:**
- Create: `web/src/pages/Usage.test.tsx`
- Create: `web/src/pages/UsagePage.tsx`
- Modify: `web/src/router.tsx:79-81` (swap the `Placeholder` for `UsagePage`)

**Interfaces:**
- Consumes: `useUsage`, `queryKeys` (Task 2, `web/src/api`), `UsageResponse`, `LimitStatus`, `ScopeCost`, `ScopeGrowth` (Task 2, `web/src/api`).
- Produces: `export function UsagePage()` from `web/src/pages/UsagePage.tsx`, a zero-prop component (no injected navigation — this page has no drill-down, unlike Claims).

- [ ] **Step 1: Write the failing test**

Create `web/src/pages/Usage.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { UsagePage } from './UsagePage';
import { getUsage } from '../api/client';
import type { UsageResponse } from '../api';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getUsage: vi.fn() };
});

const SAMPLE_USAGE: UsageResponse = {
  principal: 'local',
  contribution: {
    window: '1h0m0s',
    used: 12,
    ceiling: 120,
    remaining: 108,
    allowed: true,
    reason: '',
  },
  cost: {
    window: '1h0m0s',
    used: 3,
    ceiling: 500,
    remaining: 497,
    allowed: true,
    reason: '',
  },
  input_tokens_used: 40000,
  input_tokens_ceiling: 2000000,
  recall: {
    window: '1m0s',
    used: 120,
    ceiling: 120,
    remaining: 0,
    allowed: false,
    reason: 'recall_rate',
  },
  denied_window: '1h0m0s',
  denied: 2,
  cost_by_scope: [
    {
      scope: { kind: 'repo', value: 'cred' },
      calls: 3,
      input_tokens: 40000,
      output_tokens: 900,
    },
  ],
  scope_growth: [
    {
      scope: { kind: 'repo', value: 'cred' },
      live: 4800,
      ceiling: 5000,
      next_prune: 0,
    },
  ],
};

function renderWithClient(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  );
}

describe('UsagePage', () => {
  beforeEach(() => {
    vi.mocked(getUsage).mockResolvedValue(SAMPLE_USAGE);
  });

  it('renders the three limit stats and a scope row', async () => {
    renderWithClient(<UsagePage />);

    expect(await screen.findByText('Contribution')).toBeInTheDocument();
    expect(screen.getByText('Inference cost')).toBeInTheDocument();
    expect(screen.getByText('Recall')).toBeInTheDocument();
    expect(screen.getAllByText('repo: cred').length).toBeGreaterThan(0);
  });

  it('shows a warning banner when contributions have been denied', async () => {
    renderWithClient(<UsagePage />);

    expect(
      await screen.findByText(/2 contribution\(s\) denied/),
    ).toBeInTheDocument();
  });

  it('marks an exhausted limit with a warning badge', async () => {
    renderWithClient(<UsagePage />);

    expect(await screen.findByText('exhausted')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npx vitest run src/pages/Usage.test.tsx`
Expected: FAIL — `Cannot find module './UsagePage'` (the file doesn't exist yet).

- [ ] **Step 3: Write the page**

Create `web/src/pages/UsagePage.tsx`:

```tsx
import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout';
import { HStack, VStack } from '@astryxdesign/core/Stack';
import { Center } from '@astryxdesign/core/Center';
import { Card } from '@astryxdesign/core/Card';
import { Heading, Text } from '@astryxdesign/core/Text';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Badge } from '@astryxdesign/core/Badge';
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { ProgressBar } from '@astryxdesign/core/ProgressBar';
import { Table, proportional, pixel } from '@astryxdesign/core/Table';
import type { TableColumn } from '@astryxdesign/core/Table';
import { useQueryClient } from '@tanstack/react-query';
import { useUsage, queryKeys } from '../api';
import type { LimitStatus, ScopeCost, ScopeGrowth, UsageResponse } from '../api';

type ScopeCostRow = ScopeCost & Record<string, unknown>;
type ScopeGrowthRow = ScopeGrowth & Record<string, unknown>;

const scopeCostColumns: TableColumn<ScopeCostRow>[] = [
  {
    key: 'scope',
    header: 'Scope',
    width: proportional(2),
    renderCell: (row) => (
      <Text type="body" maxLines={1}>
        {row.scope.kind}: {row.scope.value}
      </Text>
    ),
  },
  {
    key: 'calls',
    header: 'Calls',
    width: pixel(100),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.calls}
      </Text>
    ),
  },
  {
    key: 'input_tokens',
    header: 'Input tokens',
    width: pixel(140),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.input_tokens}
      </Text>
    ),
  },
  {
    key: 'output_tokens',
    header: 'Output tokens',
    width: pixel(140),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.output_tokens}
      </Text>
    ),
  },
];

const scopeGrowthColumns: TableColumn<ScopeGrowthRow>[] = [
  {
    key: 'scope',
    header: 'Scope',
    width: proportional(2),
    renderCell: (row) => (
      <Text type="body" maxLines={1}>
        {row.scope.kind}: {row.scope.value}
      </Text>
    ),
  },
  {
    key: 'live',
    header: 'Live claims',
    width: pixel(120),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.live}
      </Text>
    ),
  },
  {
    key: 'ceiling',
    header: 'Ceiling',
    width: pixel(100),
    renderCell: (row) => (
      <Text type="body" color="secondary" hasTabularNumbers>
        {row.ceiling <= 0 ? 'off' : row.ceiling}
      </Text>
    ),
  },
  {
    key: 'next_prune',
    header: 'Next prune',
    width: pixel(120),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.next_prune}
      </Text>
    ),
  },
];

export function UsagePage() {
  const usage = useUsage();
  const queryClient = useQueryClient();

  return (
    <Layout
      header={
        <LayoutHeader hasDivider>
          <HStack vAlign="center" hAlign="between">
            <Heading level={4}>Usage</Heading>
            <Button
              label="Refresh"
              variant="secondary"
              onClick={() => {
                void queryClient.invalidateQueries({
                  queryKey: queryKeys.usage({}),
                });
              }}
            />
          </HStack>
        </LayoutHeader>
      }
      content={
        <LayoutContent>
          <UsageBody
            isLoading={usage.isLoading}
            isError={usage.isError}
            data={usage.data}
          />
        </LayoutContent>
      }
    />
  );
}

function UsageBody({
  isLoading,
  isError,
  data,
}: {
  isLoading: boolean;
  isError: boolean;
  data: UsageResponse | undefined;
}) {
  if (isLoading) {
    return (
      <Center height="100%">
        <Spinner label="Loading usage" />
      </Center>
    );
  }

  if (isError || !data) {
    return (
      <Center height="100%">
        <EmptyState
          title="Couldn't load usage"
          description="The API request failed. Check that the CRED server is running."
        />
      </Center>
    );
  }

  return (
    <VStack gap={4}>
      {data.denied > 0 ? (
        <Banner
          status="warning"
          title={`${data.denied} contribution(s) denied in the last ${data.denied_window}`}
          description="Recorded and on the record — nothing was silently dropped."
        />
      ) : null}

      <HStack gap={3} wrap="wrap">
        <LimitCard label="Contribution" status={data.contribution} />
        <LimitCard
          label="Inference cost"
          status={data.cost}
          extra={`${data.input_tokens_used} / ${
            data.input_tokens_ceiling <= 0 ? 'off' : data.input_tokens_ceiling
          } input tokens`}
        />
        <LimitCard label="Recall" status={data.recall} />
      </HStack>

      <VStack gap={2}>
        <Heading level={5}>Cost by scope</Heading>
        {data.cost_by_scope.length === 0 ? (
          <Text type="body" color="secondary">
            No inference cost recorded in this window.
          </Text>
        ) : (
          <Table<ScopeCostRow>
            data={data.cost_by_scope as ScopeCostRow[]}
            columns={scopeCostColumns}
            idKey={(row) => `${row.scope.kind}:${row.scope.value}`}
            textOverflow="truncate"
          />
        )}
      </VStack>

      <VStack gap={2}>
        <Heading level={5}>Scope growth</Heading>
        {data.scope_growth.length === 0 ? (
          <Text type="body" color="secondary">
            No live claims.
          </Text>
        ) : (
          <Table<ScopeGrowthRow>
            data={data.scope_growth as ScopeGrowthRow[]}
            columns={scopeGrowthColumns}
            idKey={(row) => `${row.scope.kind}:${row.scope.value}`}
            textOverflow="truncate"
          />
        )}
      </VStack>
    </VStack>
  );
}

function LimitCard({
  label,
  status,
  extra,
}: {
  label: string;
  status: LimitStatus;
  extra?: string;
}) {
  const hasCeiling = status.ceiling > 0;
  const value = hasCeiling
    ? Math.min((status.used / status.ceiling) * 100, 100)
    : 0;

  return (
    <Card padding={3}>
      <VStack gap={2} width={220}>
        <HStack hAlign="between" vAlign="center">
          <Text type="body" weight="semibold">
            {label}
          </Text>
          {!status.allowed ? (
            <Badge variant="warning" label="exhausted" />
          ) : null}
        </HStack>
        {hasCeiling ? (
          <ProgressBar
            label={`${label} usage`}
            isLabelHidden
            value={value}
            variant={status.allowed ? 'accent' : 'error'}
          />
        ) : (
          <Text type="supporting" color="secondary">
            unlimited
          </Text>
        )}
        <Text type="supporting" color="secondary" hasTabularNumbers>
          {status.used} / {hasCeiling ? status.ceiling : 'off'} per{' '}
          {status.window}
        </Text>
        {extra ? (
          <Text type="supporting" color="secondary" hasTabularNumbers>
            {extra}
          </Text>
        ) : null}
      </VStack>
    </Card>
  );
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && npx vitest run src/pages/Usage.test.tsx`
Expected: PASS, 3 tests.

- [ ] **Step 5: Wire the page into the router**

In `web/src/router.tsx`, change the import:

```ts
import { Placeholder } from './pages/Placeholder';
```

to:

```ts
import { Placeholder } from './pages/Placeholder';
import { UsagePage } from './pages/UsagePage';
```

Then change:

```ts
const usageRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/usage',
  component: () => <Placeholder title="Usage" />,
});
```

to:

```ts
const usageRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/usage',
  component: UsagePage,
});
```

- [ ] **Step 6: Run the full frontend test suite and typecheck**

Run: `task web:test`
Expected: all suites pass, including the 3 new `UsagePage` tests and the existing `Claims`, `ClaimDetail`, `Recall`, and `router` tests (unaffected).

Run: `cd web && npm run typecheck`
Expected: exits 0.

Run: `cd web && npm run lint`
Expected: exits 0 (no unused imports, no raw `<div>`/hex/px violations).

- [ ] **Step 7: Commit**

```bash
git add web/src/pages/UsagePage.tsx web/src/pages/Usage.test.tsx web/src/router.tsx
git commit -m "feat: usage page — limit headroom, denied count, cost and growth by scope

Follows the RecallPage/ClaimsPage conventions: Stat-style limit
cards, Astryx Table for the two scope reports, a Banner when
contributions have been denied. Loads on mount, no polling."
```

---

### Task 4: End-to-end verification

This codebase's established practice (see the commit messages for the recall
inspector and claim detail features) is manual end-to-end verification
against a real Postgres-backed server, not a Go-level HTTP test — Task 1
Step 4 already confirmed `internal/api` still has no Go test files, so this
task follows that precedent rather than introducing a new one partway
through the feature.

**Files:** none (verification only).

- [ ] **Step 1: Start Postgres and migrate**

Run: `docker compose up -d db`
Expected: `db` container starts (or is already running).

Run: `go build -o cred .`
Expected: exits 0.

Run: `./cred migrate`
Expected: exits 0, reports the schema is up to date or applies pending migrations.

- [ ] **Step 2: Put some real data behind the report**

Run: `./cred remember "the usage ledger is append-only so the worker and recall paths never contend on a row"`
Expected: prints a new claim ID — this exercises the contribution counter.

Run: `./cred recall "usage ledger"`
Expected: prints at least one result with a score breakdown — this exercises the recall counter and records a `usage_events` row via `RecordRecall`.

- [ ] **Step 3: Hit the endpoint directly**

Run: `./cred web &` (background it, or run in a second terminal), then:

Run: `curl -s localhost:8080/api/usage | head -c 2000`
Expected: a JSON object with `principal`, `contribution.used >= 1`,
`recall.used >= 1`, and a non-empty `cost_by_scope`/`scope_growth` if the
`remember` call above landed in a scope — `contribution` counts unscoped
writes too, so `contribution.used` is the one field guaranteed non-zero here.

- [ ] **Step 4: Verify the page in a live browser**

Run: `task dev`
Expected: two processes start; the Vite dev server serves `:5173` and proxies
`/api` to the Go process on `:8080`.

Open `http://localhost:5173/usage` in a browser. Verify:
- Three limit cards render (Contribution, Inference cost, Recall) with real
  numbers, not placeholders.
- No errors in the browser console.
- Clicking "Refresh" re-fetches (watch the Network tab for a new `/api/usage`
  request).
- The Contribution and Cost tables show at least one row if the `remember`
  call above wrote to a named scope; otherwise verify the "No inference cost
  recorded" / "No live claims" empty-state text renders instead of a blank
  table.

Stop `task dev` (Ctrl-C) once verified.

- [ ] **Step 5: No commit for this task**

This task is verification-only; if any step above surfaces a bug, fix it
under the task where the bug originated (Task 1, 2, or 3), re-run that
task's checks, commit the fix there, and re-run this task from Step 1.
