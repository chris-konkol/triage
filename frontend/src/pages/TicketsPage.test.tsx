import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../test/renderWithProviders';
import TicketsPage from './TicketsPage';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock('../api/tickets', () => ({
  listTickets: vi.fn(),
}));

import { listTickets } from '../api/tickets';

const fakeTickets = {
  tickets: [
    {
      id: 'abc-123',
      title: 'Login broken',
      priority: 'PRIORITY_HIGH',
      status: 'STATUS_OPEN',
      category: 'CATEGORY_BUG',
      createdBy: 'alice',
      assignedTo: '',
      tags: [],
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    },
  ],
  totalCount: 1,
};

describe('TicketsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listTickets).mockResolvedValue(fakeTickets);
  });

  it('does not render the ticket table while loading', () => {
    vi.mocked(listTickets).mockReturnValue(new Promise(() => {})); // never resolves
    renderWithProviders(<TicketsPage />);
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('renders ticket rows after data loads', async () => {
    renderWithProviders(<TicketsPage />);
    await waitFor(() => expect(screen.getByText('Login broken')).toBeInTheDocument());
    expect(screen.getByText('alice')).toBeInTheDocument();
  });

  it('shows the total ticket count', async () => {
    renderWithProviders(<TicketsPage />);
    await waitFor(() => expect(screen.getByText(/1 tickets/i)).toBeInTheDocument());
  });

  it('renders filter controls', async () => {
    renderWithProviders(<TicketsPage />);
    await waitFor(() => screen.getByText('Login broken'));
    expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
  });

  it('clicking a ticket row navigates to its detail page', async () => {
    renderWithProviders(<TicketsPage />);
    await waitFor(() => screen.getByText('Login broken'));
    screen.getByText('Login broken').closest('tr')?.click();
    expect(mockNavigate).toHaveBeenCalledWith('/tickets/abc-123');
  });
});
