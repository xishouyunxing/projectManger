import { act, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import api from '../services/api';
import { AuthProvider, useAuth } from './AuthContext';

const AuthProbe = () => {
  const { user, token, login } = useAuth();
  return (
    <div>
      <span data-testid="user-name">{user?.name || 'none'}</span>
      <span data-testid="token">{token || 'none'}</span>
      <button onClick={() => login('AUTH-1', 'secret')}>login</button>
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

  it('restores cached user state without requiring a stored JWT', () => {
    localStorage.setItem(
      'user',
      JSON.stringify({
        id: 1,
        employee_id: 'AUTH-1',
        name: 'Auth User',
        department: null,
        role: 'admin',
      }),
    );
    localStorage.setItem('lastActivity', Date.now().toString());

    render(
      <AuthProvider>
        <AuthProbe />
      </AuthProvider>,
    );

    expect(screen.getByTestId('user-name')).toHaveTextContent('Auth User');
    expect(screen.getByTestId('token')).toHaveTextContent('cookie');
  });

  it('does not persist the login response token in localStorage', async () => {
    const postSpy = vi.spyOn(api, 'post').mockResolvedValueOnce({
      data: {
        token: 'jwt-from-compat-response',
        user: {
          id: 1,
          employee_id: 'AUTH-1',
          name: 'Auth User',
          department: null,
          role: 'admin',
        },
        permissions: {
          codes: ['page:permissions'],
          lines: {},
          managed_line_ids: [],
        },
      },
    });

    render(
      <AuthProvider>
        <AuthProbe />
      </AuthProvider>,
    );

    await act(async () => {
      screen.getByRole('button', { name: 'login' }).click();
    });

    await waitFor(() => {
      expect(screen.getByTestId('user-name')).toHaveTextContent('Auth User');
    });
    expect(screen.getByTestId('token')).toHaveTextContent('cookie');
    expect(localStorage.getItem('token')).toBeNull();
    expect(postSpy).toHaveBeenCalledWith('/login', {
      employee_id: 'AUTH-1',
      password: 'secret',
    });
  });
});
