import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../test/renderWithProviders';
import DashboardPage from './DashboardPage';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock('../api/tickets', () => ({
  getDashboardStats: vi.fn(),
}));

import { getDashboardStats } from '../api/tickets';

const fakeStats = {
  totalOpen: 5,
  totalInProgress: 2,
  totalResolved: 10,
  totalClosed: 1,
  avgResolutionHours: 4.5,
  ticketsByPriority: [
    { priority: 'High', count: 3 },
    { priority: 'Medium', count: 2 },
  ],
  ticketsByCategory: [
    { category: 'Bug', count: 4 },
  ],
  ticketsPerDay: [
    { date: '2024-01-01', created: 2, resolved: 1 },
  ],
};

describe('DashboardPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getDashboardStats).mockResolvedValue(fakeStats);
  });

  it('renders the Dashboard heading', () => {
    renderWithProviders(<DashboardPage />);
    expect(screen.getByRole('heading', { name: /dashboard/i })).toBeInTheDocument();
  });

  it('renders stat cards with correct values after data loads', async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Open')).toBeInTheDocument());
    expect(screen.getByText('In Progress')).toBeInTheDocument();
    expect(screen.getByText('Resolved')).toBeInTheDocument();
    expect(screen.getByText('Closed')).toBeInTheDocument();
  });

  it('shows average resolution time when non-zero', async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() =>
      expect(screen.getByText(/avg resolution time/i)).toBeInTheDocument()
    );
    expect(screen.getByText(/4\.5 hours/i)).toBeInTheDocument();
  });

  it('hides average resolution time when zero', async () => {
    vi.mocked(getDashboardStats).mockResolvedValue({ ...fakeStats, avgResolutionHours: 0 });
    renderWithProviders(<DashboardPage />);
    await waitFor(() => screen.getByText('Open'));
    expect(screen.queryByText(/avg resolution time/i)).not.toBeInTheDocument();
  });

  it('shows an error message when the request fails', async () => {
    vi.mocked(getDashboardStats).mockRejectedValue(new Error('Network error'));
    renderWithProviders(<DashboardPage />);
    await waitFor(() =>
      expect(screen.getByText(/failed to load dashboard/i)).toBeInTheDocument()
    );
  });

  it('"View Tickets" navigates to /tickets', async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() => screen.getByText('Open'));
    screen.getByText(/view tickets/i).click();
    expect(mockNavigate).toHaveBeenCalledWith('/tickets');
  });
});
