import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get, post, put, del, ApiError } from './client';
import * as authUtils from '../utils/auth';

function mockFetch(status: number, body: unknown) {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
    text: () => Promise.resolve(JSON.stringify(body)),
  });
}

describe('api client', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(authUtils, 'getToken').mockReturnValue(null);
  });

  it('get sends a GET request and returns parsed JSON', async () => {
    global.fetch = mockFetch(200, { id: '1' });
    const result = await get<{ id: string }>('/tickets');
    expect(result).toEqual({ id: '1' });
    expect(fetch).toHaveBeenCalledWith('/api/tickets', expect.objectContaining({
      headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
    }));
  });

  it('post sends a POST request with JSON body', async () => {
    global.fetch = mockFetch(201, { ticket: { id: '1' } });
    await post('/tickets', { title: 'Bug' });
    expect(fetch).toHaveBeenCalledWith('/api/tickets', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ title: 'Bug' }),
    }));
  });

  it('put sends a PUT request', async () => {
    global.fetch = mockFetch(200, { ticket: { id: '1' } });
    await put('/tickets/1', { status: 2 });
    expect(fetch).toHaveBeenCalledWith('/api/tickets/1', expect.objectContaining({
      method: 'PUT',
    }));
  });

  it('del sends a DELETE request', async () => {
    global.fetch = mockFetch(204, undefined);
    await del('/tickets/1');
    expect(fetch).toHaveBeenCalledWith('/api/tickets/1', expect.objectContaining({
      method: 'DELETE',
    }));
  });

  it('throws ApiError with correct status on non-ok response', async () => {
    global.fetch = mockFetch(404, { error: 'not found' });
    await expect(get('/tickets/missing')).rejects.toThrow(ApiError);
    await expect(get('/tickets/missing')).rejects.toMatchObject({ status: 404 });
  });

  it('includes Authorization header when token is present', async () => {
    vi.spyOn(authUtils, 'getToken').mockReturnValue('my-token');
    global.fetch = mockFetch(200, {});
    await get('/tickets');
    expect(fetch).toHaveBeenCalledWith('/api/tickets', expect.objectContaining({
      headers: expect.objectContaining({ Authorization: 'Bearer my-token' }),
    }));
  });

  it('omits Authorization header when no token is stored', async () => {
    global.fetch = mockFetch(200, {});
    await get('/tickets');
    const headers = (fetch as ReturnType<typeof vi.fn>).mock.calls[0][1].headers;
    expect(headers).not.toHaveProperty('Authorization');
  });
});
