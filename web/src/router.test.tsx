import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  RouterProvider,
  createRouter,
  createMemoryHistory,
} from '@tanstack/react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { router } from './router';
import { getClaims, getClaim, getHealth } from './api/client';
import type { ClaimList, ClaimDetail, Health } from './api';

vi.mock('./api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./api/client')>();
  return {
    ...actual,
    getClaims: vi.fn(),
    getClaim: vi.fn(),
    getHealth: vi.fn(),
  };
});

const AUTHENTICATED: Health = {
  status: 'ok',
  version: '0.1.0',
  principal: 'local',
  registration_open: false,
};

const CLAIMS: ClaimList = {
  claims: [
    {
      id: 'c1',
      statement: 'ACL intersection is enforced at recall',
      kind: 'behavior',
      scope: { kind: 'repo', value: 'cred' },
      status: 'live',
      contributed_by: 'local',
      recorded_at: '2026-07-20T00:00:00Z',
      valid_from: '2026-07-20T00:00:00Z',
      valid_until: '',
      superseded_at: '',
    },
  ],
  limit: 50,
  offset: 0,
  count: 1,
};

const DETAIL: ClaimDetail = {
  id: 'c1',
  statement: 'ACL intersection is enforced at recall',
  kind: 'behavior',
  scope: { kind: 'repo', value: 'cred' },
  status: 'live',
  confidence: 0.9,
  contributed_by: 'local',
  source_repo: 'cred',
  recorded_at: '2026-07-20T00:00:00Z',
  valid_from: '2026-07-20T00:00:00Z',
  valid_until: '',
  superseded_at: '',
  expired_reason: '',
  evidence: [],
};

function renderApp(path: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const testRouter = createRouter({
    routeTree: router.routeTree,
    history: createMemoryHistory({ initialEntries: [path] }),
  });
  return render(
    <QueryClientProvider client={client}>
      <RouterProvider router={testRouter} />
    </QueryClientProvider>,
  );
}

describe('claim navigation', () => {
  beforeEach(() => {
    vi.mocked(getClaims).mockResolvedValue(CLAIMS);
    vi.mocked(getClaim).mockResolvedValue(DETAIL);
    vi.mocked(getHealth).mockResolvedValue(AUTHENTICATED);
  });

  it('opens a claim from the list and returns to the list from its detail', async () => {
    const user = userEvent.setup();
    renderApp('/claims');

    await user.click(
      await screen.findByText('ACL intersection is enforced at recall'),
    );

    expect(await screen.findByText('Evidence (0)')).toBeInTheDocument();
    expect(getClaim).toHaveBeenCalledWith('c1');

    const breadcrumb = screen.getByRole('navigation', {
      name: 'Claim location',
    });
    await user.click(within(breadcrumb).getByText('Claims'));

    expect(
      await screen.findByRole('columnheader', { name: 'Statement' }),
    ).toBeInTheDocument();
  });
});

describe('appRoute beforeLoad guard', () => {
  beforeEach(() => {
    vi.mocked(getClaims).mockResolvedValue(CLAIMS);
    vi.mocked(getClaim).mockResolvedValue(DETAIL);
  });

  it('redirects to login when principal is empty', async () => {
    vi.mocked(getHealth).mockResolvedValue({
      status: 'ok',
      version: '0.1.0',
      principal: '',
      registration_open: false,
    });

    renderApp('/claims');

    expect(await screen.findByText('Welcome back')).toBeInTheDocument();
    expect(
      await screen.findByRole('button', { name: 'Sign in' }),
    ).toBeInTheDocument();
  });

  it('redirects to login when the health check itself throws, rather than surfacing an unhandled route error', async () => {
    vi.mocked(getHealth).mockRejectedValue(new Error('network error'));

    renderApp('/claims');

    expect(await screen.findByText('Welcome back')).toBeInTheDocument();
    expect(
      await screen.findByRole('button', { name: 'Sign in' }),
    ).toBeInTheDocument();
  });
});
