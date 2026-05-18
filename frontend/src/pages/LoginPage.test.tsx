import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../test/renderWithProviders';
import LoginPage from './LoginPage';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock('../api/client', async () => {
  const actual = await vi.importActual('../api/client');
  return { ...actual, post: vi.fn() };
});

import { post } from '../api/client';

// Mantine renders multiple elements that match /password/i (label + toggle button);
// selector:'input' targets only the actual <input type="password">.
const getPasswordInput = () =>
  screen.getByLabelText(/password/i, { selector: 'input' });

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('renders username, password fields and sign-in button', () => {
    renderWithProviders(<LoginPage />);
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
    expect(getPasswordInput()).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
  });

  it('switches to register mode when Register link is clicked', async () => {
    renderWithProviders(<LoginPage />);
    await userEvent.click(screen.getByText(/register/i));
    expect(screen.getByRole('button', { name: /register/i })).toBeInTheDocument();
  });

  it('navigates to /tickets on successful login', async () => {
    vi.mocked(post).mockResolvedValue({ token: 'jwt-token' });
    renderWithProviders(<LoginPage />);

    await userEvent.type(screen.getByLabelText(/username/i), 'alice');
    await userEvent.type(getPasswordInput(), 'password');
    await userEvent.click(screen.getByRole('button', { name: /sign in/i }));

    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/tickets'));
  });

  it('shows error message on failed login', async () => {
    vi.mocked(post).mockRejectedValue(new Error('Invalid credentials'));
    renderWithProviders(<LoginPage />);

    await userEvent.type(screen.getByLabelText(/username/i), 'alice');
    await userEvent.type(getPasswordInput(), 'wrong');
    await userEvent.click(screen.getByRole('button', { name: /sign in/i }));

    await waitFor(() =>
      expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument()
    );
  });
});
