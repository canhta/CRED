import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { ClaimsPage } from './ClaimsPage';
import { getClaims } from '../api/client';
import type { ClaimList } from '../api';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getClaims: vi.fn() };
});

function makeClaim(
  id: string,
  statement: string,
  status: 'live' | 'expired',
) {
  return {
    id,
    statement,
    kind: 'behavior',
    scope: { kind: 'repo', value: 'cred' },
    status,
    source: {
      kind: 'git',
      repo: 'cred',
      path: 'internal/acl/acl.go',
      line_start: 10,
      line_end: 20,
      symbol_path: 'acl.Check',
    },
    contributed_by: 'local',
    recorded_at: '2026-07-20T00:00:00Z',
    valid_from: '2026-07-20T00:00:00Z',
    valid_until: '',
    superseded_at: '',
  };
}

const TWO_CLAIMS: ClaimList = {
  claims: [
    makeClaim('c1', 'ACL intersection is enforced at recall', 'live'),
    makeClaim('c2', 'Legacy token check ran on every request', 'expired'),
  ],
  limit: 50,
  offset: 0,
  count: 2,
};

function renderWithClient(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  );
}

describe('ClaimsPage', () => {
  beforeEach(() => {
    vi.mocked(getClaims).mockResolvedValue(TWO_CLAIMS);
  });

  it('renders a row per claim with a status indicator that differs by status', async () => {
    renderWithClient(<ClaimsPage />);

    expect(
      await screen.findByText('ACL intersection is enforced at recall'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Legacy token check ran on every request'),
    ).toBeInTheDocument();

    const liveDot = screen.getByLabelText('live');
    const expiredDot = screen.getByLabelText('expired');

    expect(liveDot).toBeInTheDocument();
    expect(expiredDot).toBeInTheDocument();
    expect(liveDot.getAttribute('data-variant')).toBe('success');
    expect(expiredDot.getAttribute('data-variant')).toBe('neutral');
    expect(liveDot.getAttribute('data-variant')).not.toBe(
      expiredDot.getAttribute('data-variant'),
    );
  });
});
