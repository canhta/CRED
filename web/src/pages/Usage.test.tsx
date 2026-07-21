import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { UsagePage } from './UsagePage';
import { getHealth, getUsage, getUsageOrg } from '../api/client';
import type { Health, OrgUsageResponse, UsageResponse } from '../api';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    getHealth: vi.fn(),
    getUsage: vi.fn(),
    getUsageOrg: vi.fn(),
  };
});

const ADMIN: Health = {
  status: 'ok',
  version: '0.1.0',
  principal: 'local',
  role: 'admin',
  registration_open: false,
};

const MEMBER: Health = { ...ADMIN, role: 'member' };

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
};

const SAMPLE_ORG_USAGE: OrgUsageResponse = {
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
    vi.clearAllMocks();
    vi.mocked(getUsage).mockResolvedValue(SAMPLE_USAGE);
    vi.mocked(getUsageOrg).mockResolvedValue(SAMPLE_ORG_USAGE);
  });

  it('renders the three limit stats for any authenticated role', async () => {
    vi.mocked(getHealth).mockResolvedValue(MEMBER);
    renderWithClient(<UsagePage />);

    expect(await screen.findByText('Contribution')).toBeInTheDocument();
    expect(screen.getByText('Inference cost')).toBeInTheDocument();
    expect(screen.getByText('Recall')).toBeInTheDocument();
  });

  it('shows a warning banner when contributions have been denied', async () => {
    vi.mocked(getHealth).mockResolvedValue(MEMBER);
    renderWithClient(<UsagePage />);

    expect(
      await screen.findByText(/2 contribution\(s\) denied/),
    ).toBeInTheDocument();
  });

  it('marks an exhausted limit with a warning badge', async () => {
    vi.mocked(getHealth).mockResolvedValue(MEMBER);
    renderWithClient(<UsagePage />);

    expect(await screen.findByText('exhausted')).toBeInTheDocument();
  });

  it('renders the org-wide scope tables for an admin', async () => {
    vi.mocked(getHealth).mockResolvedValue(ADMIN);
    renderWithClient(<UsagePage />);

    expect(await screen.findByText('Cost by scope')).toBeInTheDocument();
    expect(screen.getByText('Scope growth')).toBeInTheDocument();
    expect(screen.getAllByText('repo: cred').length).toBeGreaterThan(0);
  });

  it('hides the org-wide tables for a member and never requests them', async () => {
    vi.mocked(getHealth).mockResolvedValue(MEMBER);
    renderWithClient(<UsagePage />);

    await screen.findByText('Contribution');
    expect(screen.queryByText('Cost by scope')).not.toBeInTheDocument();
    expect(screen.queryByText('Scope growth')).not.toBeInTheDocument();
    expect(getUsageOrg).not.toHaveBeenCalled();
  });

  it('shows an error state, not a perpetual spinner, when the org-wide report fails to load for an admin', async () => {
    vi.mocked(getHealth).mockResolvedValue(ADMIN);
    vi.mocked(getUsageOrg).mockRejectedValue(new Error('server error'));
    renderWithClient(<UsagePage />);

    expect(await screen.findByText('Contribution')).toBeInTheDocument();
    expect(
      await screen.findByText("Couldn't load org-wide usage"),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText('Loading org usage')).not.toBeInTheDocument();
  });
});
