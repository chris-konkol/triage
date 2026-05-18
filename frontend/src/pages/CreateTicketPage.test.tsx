import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../test/renderWithProviders';
import CreateTicketPage from './CreateTicketPage';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock('../api/tickets', () => ({
  createTicket: vi.fn(),
}));

import { createTicket } from '../api/tickets';

describe('CreateTicketPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders title, description, priority and category fields', () => {
    renderWithProviders(<CreateTicketPage />);
    expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
    expect(screen.getByText(/priority/i)).toBeInTheDocument();
    expect(screen.getByText(/category/i)).toBeInTheDocument();
  });

  it('calls createTicket with form values and navigates on success', async () => {
    vi.mocked(createTicket).mockResolvedValue({ ticket: { id: '1' } as any });
    renderWithProviders(<CreateTicketPage />);

    await userEvent.type(screen.getByLabelText(/title/i), 'New bug report');
    await userEvent.click(screen.getByRole('button', { name: /create ticket/i }));

    // TanStack Query v5 passes the variables as the first argument to mutationFn.
    await waitFor(() => {
      const firstArg = vi.mocked(createTicket).mock.calls[0]?.[0];
      expect(firstArg).toMatchObject({ title: 'New bug report' });
    });
    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/tickets'));
  });

  it('shows error message when createTicket fails', async () => {
    vi.mocked(createTicket).mockRejectedValue(new Error('Server error'));
    renderWithProviders(<CreateTicketPage />);

    await userEvent.type(screen.getByLabelText(/title/i), 'Bad ticket');
    await userEvent.click(screen.getByRole('button', { name: /create ticket/i }));

    await waitFor(() =>
      expect(screen.getByText(/server error/i)).toBeInTheDocument()
    );
  });
});
