import { beforeEach, describe, expect, it } from 'vitest';
import api, {
  DEFAULT_TIMEOUT_MS,
  UPLOAD_TIMEOUT_MS,
  shouldRedirectToLoginOnUnauthorized,
} from './api';

describe('API unauthorized redirect handling', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/dashboard');
  });

  it('does not redirect when the login request itself returns 401', () => {
    expect(
      shouldRedirectToLoginOnUnauthorized({
        response: { status: 401 },
        config: { url: '/login' },
      }),
    ).toBe(false);
  });

  it('does not redirect when already on the login page', () => {
    window.history.pushState({}, '', '/login');

    expect(
      shouldRedirectToLoginOnUnauthorized({
        response: { status: 401 },
        config: { url: '/profile' },
      }),
    ).toBe(false);
  });

  it('redirects protected requests that return 401 away from the login page', () => {
    expect(
      shouldRedirectToLoginOnUnauthorized({
        response: { status: 401 },
        config: { url: '/profile' },
      }),
    ).toBe(true);
  });
});

describe('API client defaults', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('sends cookie credentials by default', () => {
    expect(api.defaults.withCredentials).toBe(true);
  });

  it('does not inject bearer tokens from localStorage', async () => {
    localStorage.setItem('token', 'legacy-token');
    const request = await (api.interceptors.request as any).handlers[0].fulfilled({
      headers: {},
    });

    expect(request.headers.Authorization).toBeUndefined();
  });

  it('keeps the default timeout for normal requests', async () => {
    const request = await (api.interceptors.request as any).handlers[0].fulfilled({
      headers: { 'Content-Type': 'application/json' },
      timeout: DEFAULT_TIMEOUT_MS,
    });

    expect(request.timeout).toBe(DEFAULT_TIMEOUT_MS);
  });

  it('uses a long bounded timeout for browser-managed uploads', async () => {
    const request = await (api.interceptors.request as any).handlers[0].fulfilled({
      headers: { 'Content-Type': undefined },
      timeout: DEFAULT_TIMEOUT_MS,
    });

    expect(request.timeout).toBe(UPLOAD_TIMEOUT_MS);
  });
});
