import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { LoginPage } from './LoginPage';
import { login, ApiError } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, login: vi.fn() };
});

function renderWithClient(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  );
}

describe('LoginPage', () => {
  const onSuccess = vi.fn();
  const onNavigateToRegister = vi.fn();

  beforeEach(() => {
    onSuccess.mockClear();
    onNavigateToRegister.mockClear();
  });

  it('calls onSuccess after a successful login', async () => {
    vi.mocked(login).mockResolvedValue({ principal: 'p1', role: 'admin' });
    const user = userEvent.setup();
    renderWithClient(
      <LoginPage
        onSuccess={onSuccess}
        onNavigateToRegister={onNavigateToRegister}
      />,
    );

    await user.type(screen.getByLabelText('Email'), 'a@b.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByText('Sign in'));

    expect(login).toHaveBeenCalledWith({
      email: 'a@b.com',
      password: 'password123',
    });
    expect(onSuccess).toHaveBeenCalled();
  });

  it('shows an inline error on invalid credentials, not a full-page failure', async () => {
    vi.mocked(login).mockRejectedValue(new ApiError(401, 'Unauthorized', ''));
    const user = userEvent.setup();
    renderWithClient(
      <LoginPage
        onSuccess={onSuccess}
        onNavigateToRegister={onNavigateToRegister}
      />,
    );

    await user.type(screen.getByLabelText('Email'), 'a@b.com');
    await user.type(screen.getByLabelText('Password'), 'wrong');
    await user.click(screen.getByText('Sign in'));

    expect(
      await screen.findByText('Invalid email or password.'),
    ).toBeInTheDocument();
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it('navigates to register when the link is clicked', async () => {
    const user = userEvent.setup();
    renderWithClient(
      <LoginPage
        onSuccess={onSuccess}
        onNavigateToRegister={onNavigateToRegister}
      />,
    );

    await user.click(screen.getByText('Need an account? Register'));
    expect(onNavigateToRegister).toHaveBeenCalled();
  });
});
