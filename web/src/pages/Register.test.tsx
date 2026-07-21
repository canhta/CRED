import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { RegisterPage } from './RegisterPage';
import { register, getHealth, ApiError } from '../api/client';
import type { Health } from '../api';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, register: vi.fn(), getHealth: vi.fn() };
});

const OPEN: Health = {
  status: 'ok',
  version: '0.1.0',
  principal: '',
  role: '',
  registration_open: true,
};

const CLOSED: Health = { ...OPEN, registration_open: false };

function renderWithClient(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  );
}

describe('RegisterPage', () => {
  const onSuccess = vi.fn();
  const onNavigateToLogin = vi.fn();

  beforeEach(() => {
    onSuccess.mockClear();
    onNavigateToLogin.mockClear();
  });

  it('registers and calls onSuccess when registration is open', async () => {
    vi.mocked(getHealth).mockResolvedValue(OPEN);
    vi.mocked(register).mockResolvedValue({ principal: 'p1', role: 'admin' });
    const user = userEvent.setup();
    renderWithClient(
      <RegisterPage onSuccess={onSuccess} onNavigateToLogin={onNavigateToLogin} />,
    );

    await user.type(await screen.findByLabelText('Email'), 'a@b.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByText('Create account'));

    expect(register).toHaveBeenCalledWith({
      email: 'a@b.com',
      password: 'password123',
    });
    expect(onSuccess).toHaveBeenCalled();
  });

  it('redirects to login when registration is closed', async () => {
    vi.mocked(getHealth).mockResolvedValue(CLOSED);
    renderWithClient(
      <RegisterPage onSuccess={onSuccess} onNavigateToLogin={onNavigateToLogin} />,
    );

    expect(await screen.findByText(/registration is closed/i)).toBeInTheDocument();
  });

  it('shows a distinct error state when the health check itself fails, not "registration is closed"', async () => {
    vi.mocked(getHealth).mockRejectedValue(new Error('network error'));
    renderWithClient(
      <RegisterPage onSuccess={onSuccess} onNavigateToLogin={onNavigateToLogin} />,
    );

    expect(
      await screen.findByText(/couldn't check registration status/i),
    ).toBeInTheDocument();
    expect(screen.queryByText(/registration is closed/i)).not.toBeInTheDocument();
  });

  it('renders the invite form and skips the registration-closed check when an invite param is present', async () => {
    vi.mocked(getHealth).mockResolvedValue(CLOSED);
    renderWithClient(
      <RegisterPage
        invite="tok123"
        onSuccess={onSuccess}
        onNavigateToLogin={onNavigateToLogin}
      />,
    );

    expect(await screen.findByText('Create your account')).toBeInTheDocument();
    expect(screen.queryByText(/registration is closed/i)).not.toBeInTheDocument();
  });

  it('registers with the invite token and calls onSuccess', async () => {
    vi.mocked(register).mockResolvedValue({ principal: 'p1', role: 'member' });
    const user = userEvent.setup();
    renderWithClient(
      <RegisterPage
        invite="tok123"
        onSuccess={onSuccess}
        onNavigateToLogin={onNavigateToLogin}
      />,
    );

    await user.type(await screen.findByLabelText('Email'), 'invitee@b.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByText('Create account'));

    expect(register).toHaveBeenCalledWith({
      email: 'invitee@b.com',
      password: 'password123',
      invite: 'tok123',
    });
    expect(onSuccess).toHaveBeenCalled();
  });

  it('shows an invalid-invite error on a 400 response', async () => {
    vi.mocked(register).mockRejectedValue(new ApiError(400, 'Bad Request', ''));
    const user = userEvent.setup();
    renderWithClient(
      <RegisterPage
        invite="tok123"
        onSuccess={onSuccess}
        onNavigateToLogin={onNavigateToLogin}
      />,
    );

    await user.type(await screen.findByLabelText('Email'), 'invitee@b.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByText('Create account'));

    expect(
      await screen.findByText(/invite link is invalid or has expired/i),
    ).toBeInTheDocument();
  });
});
