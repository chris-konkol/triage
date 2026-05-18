import { get, post, put, del } from './client';
import type { Ticket, Comment } from '../types/ticket';
import type { DashboardStats } from '../types/analytics';

export const getDashboardStats = () => get<DashboardStats>('/dashboard');

export interface ListTicketsResponse {
  tickets: Ticket[];
  totalCount: number;
}

export interface ListTicketsParams {
  search?: string;
  status?: number;
  priority?: number;
  category?: number;
  page?: number;
  pageSize?: number;
}

export const listTickets = (params?: ListTicketsParams) => {
  const qs = new URLSearchParams();
  if (params?.search)   qs.set('search', params.search);
  if (params?.status)   qs.set('status', String(params.status));
  if (params?.priority) qs.set('priority', String(params.priority));
  if (params?.category) qs.set('category', String(params.category));
  if (params?.page)     qs.set('page', String(params.page));
  if (params?.pageSize) qs.set('page_size', String(params.pageSize));
  const query = qs.toString();
  return get<ListTicketsResponse>(query ? `/tickets?${query}` : '/tickets');
};

export interface GetTicketResponse {
  ticket: Ticket;
  comments: Comment[];
}

export const getTicket = (id: string) => get<GetTicketResponse>(`/tickets/${id}`);

export interface CreateTicketPayload {
  title: string;
  description: string;
  priority: number;
  category: number;
  assignedTo?: string;
  tags?: string[];
}

export const createTicket = (data: CreateTicketPayload) =>
  post<{ ticket: Ticket }>('/tickets', data);

export interface UpdateTicketPayload {
  title?: string;
  description?: string;
  priority?: number;
  status?: number;
  category?: number;
  assignedTo?: string;
  tags?: string[];
}

export const updateTicket = (id: string, data: UpdateTicketPayload) =>
  put<{ ticket: Ticket }>(`/tickets/${id}`, data);

export const deleteTicket = (id: string) =>
  del<void>(`/tickets/${id}`);

export const addComment = (ticketId: string, body: string) =>
  post<{ comment: Comment }>(`/tickets/${ticketId}/comments`, { body });
