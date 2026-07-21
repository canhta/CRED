import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { RecallPage } from './RecallPage';
import { getRecall } from '../api/client';
import type { RecallResponse } from '../api';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getRecall: vi.fn() };
});

const RESPONSE: RecallResponse = {
  query: 'acl intersection',
  claims: [
    {
      id: 'c1',
      statement: 'ACL intersection is enforced at recall',
      kind: 'behavior',
      scope: { kind: 'repo', value: 'cred' },
      status: 'live',
      source: {
        kind: 'git',
        repo: 'cred',
        path: 'internal/acl/acl.go',
        line_start: 10,
        line_end: 20,
        symbol_path: 'acl.Check',
      },
      score: 0.032,
      tokens: 40,
      contributions: [{ arm: 'dense', rank: 1, raw: 0.91, score: 0.016 }],
    },
    {
      id: 'c2',
      statement: 'Token budget omissions are reported, never silent',
      kind: 'behavior',
      scope: { kind: 'repo', value: 'cred' },
      status: 'live',
      score: 0.048,
      tokens: 55,
      contributions: [
        { arm: 'dense', rank: 4, raw: 0.62, score: 0.014 },
        { arm: 'lexical', rank: 1, raw: 8.3, score: 0.016 },
      ],
    },
  ],
  candidates: 12,
  authorized: 9,
  omitted_for_budget: 0,
  tokens_used: 95,
  token_budget: 8000,
  dominant_arm: 'lexical',
  dominant_share: 0.55,
  as_of: '2026-07-20T00:00:00Z',
  staleness_seconds: 3,
  timings: {
    embed_ms: 4,
    dense_ms: 6,
    lexical_ms: 2,
    load_ms: 1,
    total_ms: 13,
  },
};

function renderWithClient(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  );
}

describe('RecallPage', () => {
  beforeEach(() => {
    vi.mocked(getRecall).mockResolvedValue(RESPONSE);
  });

  it('renders each recalled claim and shows both arms for a claim found by two', async () => {
    const user = userEvent.setup();
    renderWithClient(<RecallPage />);

    await user.type(
      screen.getByRole('textbox', { name: 'Recall query' }),
      'acl intersection',
    );
    await user.click(screen.getByRole('button', { name: 'Search' }));

    expect(
      await screen.findByText('ACL intersection is enforced at recall'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Token budget omissions are reported, never silent'),
    ).toBeInTheDocument();

    // The single-arm claim contributes one dense badge; the two-arm claim
    // contributes both a dense and a lexical badge — the insight the view exists
    // to show. Assert both arm labels are present for the dual-arm result.
    expect(screen.getByText('dense #4')).toBeInTheDocument();
    expect(screen.getByText('lexical #1')).toBeInTheDocument();
    expect(screen.getByText('dense #1')).toBeInTheDocument();
  });
});
