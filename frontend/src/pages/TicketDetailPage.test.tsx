import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../test/renderWithProviders';
import TicketDetailPage from './TicketDetailPage';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ id: 'ticket-abc' }),
  };
});

vi.mock('../api/tickets', () => ({
  getTicket: vi.fn(),
  updateTicket: vi.fn(),
  addComment: vi.fn(),
  deleteTicket: vi.fn(),
}));

import { getTicket, addComment, deleteTicket } from '../api/tickets';

const fakeResponse = {
  ticket: {
    id: 'ticket-abc',
    title: 'Fix auth bug',
    description: 'Users cannot log in after password reset.',
    priority: 'PRIORITY_HIGH',
    status: 'STATUS_OPEN',
    category: 'CATEGORY_BUG',
    createdBy: 'alice',
    assignedTo: 'bob',
    tags: [],
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  },
  comments: [
    {
      id: 'c1',
      ticketId: 'ticket-abc',
      author: 'charlie',
      body: 'Reproduced on prod.',
      createdAt: '2024-01-02T00:00:00Z',
    },
  ],
};

describe('TicketDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getTicket).mockResolvedValue(fakeResponse);
  });

  it('renders the ticket title and description', async () => {
    renderWithProviders(<TicketDetailPage />);
    await waitFor(() => expect(screen.getByText('Fix auth bug')).toBeInTheDocument());
    expect(screen.getByText(/users cannot log in/i)).toBeInTheDocument();
  });

  it('renders existing comments', async () => {
    renderWithProviders(<TicketDetailPage />);
    await waitFor(() => expect(screen.getByText('Reproduced on prod.')).toBeInTheDocument());
    expect(screen.getByText('charlie')).toBeInTheDocument();
  });

  it('shows the comment count', async () => {
    renderWithProviders(<TicketDetailPage />);
    await waitFor(() => expect(screen.getByText(/comments \(1\)/i)).toBeInTheDocument());
  });

  it('submits a new comment', async () => {
    vi.mocked(addComment).mockResolvedValue({
      comment: { id: 'c2', ticketId: 'ticket-abc', author: 'me', body: 'LGTM', createdAt: '' },
    });
    vi.mocked(getTicket).mockResolvedValue({
      ...fakeResponse,
      comments: [...fakeResponse.comments, { id: 'c2', ticketId: 'ticket-abc', author: 'me', body: 'LGTM', createdAt: '' }],
    });

    renderWithProviders(<TicketDetailPage />);
    await waitFor(() => screen.getByText('Fix auth bug'));

    const textarea = screen.getByLabelText(/add a comment/i);
    await userEvent.type(textarea, 'LGTM');
    await userEvent.click(screen.getByRole('button', { name: /post comment/i }));

    await waitFor(() =>
      expect(addComment).toHaveBeenCalledWith('ticket-abc', 'LGTM')
    );
  });

  it('navigates back when delete is confirmed', async () => {
    vi.mocked(deleteTicket).mockResolvedValue(undefined);
    vi.spyOn(window, 'confirm').mockReturnValue(true);

    renderWithProviders(<TicketDetailPage />);
    await waitFor(() => screen.getByText('Fix auth bug'));

    await userEvent.click(screen.getByRole('button', { name: /delete/i }));

    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/tickets'));
  });
});
