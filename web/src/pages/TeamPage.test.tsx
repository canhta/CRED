import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { TeamPage } from './TeamPage';
import {
  getTeamMembers,
  getInvites,
  createInvite,
  revokeInvite,
} from '../api/client';
import type { TeamMember, Invite } from '../api';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    getTeamMembers: vi.fn(),
    getInvites: vi.fn(),
    createInvite: vi.fn(),
    revokeInvite: vi.fn(),
  };
});

const MEMBERS: TeamMember[] = [
  {
    principal_id: 'p1',
    email: 'admin@b.com',
    role: 'admin',
    created_at: '2026-07-01T00:00:00Z',
  },
];

const INVITES: Invite[] = [
  {
    id: 'inv1',
    email: 'invitee@b.com',
    role: 'member',
    created_at: '2026-07-10T00:00:00Z',
    expires_at: '2026-07-17T00:00:00Z',
  },
];

function renderWithClient(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  );
}

describe('TeamPage', () => {
  beforeEach(() => {
    vi.mocked(getTeamMembers).mockResolvedValue(MEMBERS);
    vi.mocked(getInvites).mockResolvedValue(INVITES);
  });

  it('renders members and pending invites', async () => {
    renderWithClient(<TeamPage />);

    expect(await screen.findByText('admin@b.com')).toBeInTheDocument();
    expect(await screen.findByText('invitee@b.com')).toBeInTheDocument();
  });

  it('creates an invite and shows the generated link', async () => {
    vi.mocked(createInvite).mockResolvedValue({
      email: 'new@b.com',
      role: 'member',
      expires_at: '2026-07-28T00:00:00Z',
      token: 'rawtoken123',
    });
    const user = userEvent.setup();
    renderWithClient(<TeamPage />);

    await user.click(await screen.findByText('Invite'));
    await user.type(screen.getByLabelText('Email'), 'new@b.com');
    await user.click(screen.getByText('Send invite'));

    expect(createInvite).toHaveBeenCalledWith({
      email: 'new@b.com',
      role: 'member',
    });
    expect(await screen.findByText(/rawtoken123/)).toBeInTheDocument();
  });

  it('revokes a pending invite', async () => {
    vi.mocked(revokeInvite).mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderWithClient(<TeamPage />);

    await user.click(await screen.findByText('Revoke'));

    expect(revokeInvite).toHaveBeenCalledWith('inv1');
  });
});
