import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { AuthProvider, useAuth } from './AuthContext';

const AuthProbe = () => {
  const { user, token } = useAuth();
  return (
    <div>
      <span data-testid="user-name">{user?.name || 'none'}</span>
      <span data-testid="token">{token || 'none'}</span>
    </div>
  );
};

describe('AuthContext cached state', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it('clears corrupt cached user data instead of crashing on startup', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
    localStorage.setItem('token', 'stale-token');
    localStorage.setItem('user', '{invalid-json');
    localStorage.setItem('lastActivity', Date.now().toString());

    render(
      <AuthProvider>
        <AuthProbe />
      </AuthProvider>,
    );

    expect(screen.getByTestId('user-name')).toHaveTextContent('none');
    expect(screen.getByTestId('token')).toHaveTextContent('none');
    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('user')).toBeNull();
    expect(localStorage.getItem('lastActivity')).toBeNull();
    expect(warnSpy).toHaveBeenCalled();
  });
});
