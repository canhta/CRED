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
