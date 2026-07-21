import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { ClaimDetailPage } from './ClaimDetailPage';
import { ApiError, getClaim } from '../api/client';
import type { ClaimDetail } from '../api';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getClaim: vi.fn() };
});

const DETAIL: ClaimDetail = {
  id: 'c1',
  statement: 'ACL intersection is enforced at recall',
  kind: 'behavior',
  scope: { kind: 'repo', value: 'cred' },
  status: 'live',
  confidence: 0.87,
  contributed_by: 'local',
  source_repo: 'cred',
  recorded_at: '2026-07-20T00:00:00Z',
  valid_from: '2026-07-20T00:00:00Z',
  valid_until: '',
  superseded_at: '',
  expired_reason: '',
  evidence: [
    {
      id: 'e1',
      kind: 'code',
      repo: 'cred',
      path: 'internal/acl/acl.go',
      line_start: 10,
      line_end: 20,
      symbol_path: 'acl.Check',
      anchor: 'anchored',
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

describe('ClaimDetailPage', () => {
  beforeEach(() => {
    vi.mocked(getClaim).mockReset();
  });

  it('renders the claim, its metadata, and its evidence', async () => {
    vi.mocked(getClaim).mockResolvedValue(DETAIL);
    renderWithClient(<ClaimDetailPage id="c1" />);

    expect(
      await screen.findByText('ACL intersection is enforced at recall'),
    ).toBeInTheDocument();
    expect(screen.getByText('87%')).toBeInTheDocument();
    expect(
      screen.getByText('internal/acl/acl.go:10-20'),
    ).toBeInTheDocument();
    expect(screen.getByText('acl.Check')).toBeInTheDocument();
  });

  it('calls onBack when the breadcrumb is clicked', async () => {
    vi.mocked(getClaim).mockResolvedValue(DETAIL);
    const onBack = vi.fn();
    const user = userEvent.setup();
    renderWithClient(<ClaimDetailPage id="c1" onBack={onBack} />);

    await screen.findByText('ACL intersection is enforced at recall');
    await user.click(screen.getByText('Claims'));

    expect(onBack).toHaveBeenCalledOnce();
  });

  it('shows "not found" for a 404', async () => {
    vi.mocked(getClaim).mockRejectedValue(new ApiError(404, 'Not Found', ''));
    renderWithClient(<ClaimDetailPage id="missing" />);

    expect(await screen.findByText('Claim not found')).toBeInTheDocument();
  });
});
