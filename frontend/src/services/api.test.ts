import { beforeEach, describe, expect, it } from 'vitest';
import { shouldRedirectToLoginOnUnauthorized } from './api';

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
