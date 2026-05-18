import { getToken } from '../utils/auth';

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const res = await fetch(`/api${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new ApiError(res.status, body.error ?? 'request failed');
  }

  // 204 No Content — nothing to parse
  if (res.status === 204) return undefined as T;
  return res.json();
}

export const get  = <T>(path: string)                    => request<T>(path);
export const post = <T>(path: string, body: unknown)     => request<T>(path, { method: 'POST',   body: JSON.stringify(body) });
export const put  = <T>(path: string, body: unknown)     => request<T>(path, { method: 'PUT',    body: JSON.stringify(body) });
export const del  = <T>(path: string)                    => request<T>(path, { method: 'DELETE' });
