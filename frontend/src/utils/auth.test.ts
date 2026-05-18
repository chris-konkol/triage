import { describe, it, expect, beforeEach } from 'vitest';
import { getToken, setToken, clearToken, isLoggedIn } from './auth';

describe('auth utils', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('getToken returns null when nothing is stored', () => {
    expect(getToken()).toBeNull();
  });

  it('setToken stores the token and getToken retrieves it', () => {
    setToken('my-jwt');
    expect(getToken()).toBe('my-jwt');
  });

  it('clearToken removes the stored token', () => {
    setToken('my-jwt');
    clearToken();
    expect(getToken()).toBeNull();
  });

  it('isLoggedIn returns false when no token is stored', () => {
    expect(isLoggedIn()).toBe(false);
  });

  it('isLoggedIn returns true when a token is stored', () => {
    setToken('my-jwt');
    expect(isLoggedIn()).toBe(true);
  });

  it('isLoggedIn returns false after clearing the token', () => {
    setToken('my-jwt');
    clearToken();
    expect(isLoggedIn()).toBe(false);
  });
});
