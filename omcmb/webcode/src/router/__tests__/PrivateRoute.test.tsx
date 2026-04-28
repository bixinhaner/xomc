import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';

interface UserStoreShape {
  isAuthenticated: boolean;
  accessToken: string | null;
  refreshToken: string | null;
  isTokenExpired: () => boolean;
}

let userStoreState: UserStoreShape;

vi.mock('@core/store/userStore', () => ({
  useUserStore: () => userStoreState,
}));

import PrivateRoute from '../PrivateRoute';

function renderRoute(initialPath = '/protected') {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route
          path="/protected"
          element={
            <PrivateRoute>
              <div data-testid="protected">secret</div>
            </PrivateRoute>
          }
        />
        <Route path="/login" element={<div data-testid="login">login page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  userStoreState = {
    isAuthenticated: false,
    accessToken: null,
    refreshToken: null,
    isTokenExpired: () => false,
  };
});

describe('PrivateRoute', () => {
  it('redirects to /login when not authenticated', () => {
    renderRoute();
    expect(screen.getByTestId('login')).toBeInTheDocument();
    expect(screen.queryByTestId('protected')).not.toBeInTheDocument();
  });

  it('renders children when authenticated and tokens are valid', () => {
    userStoreState = {
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      isTokenExpired: () => false,
    };
    renderRoute();
    expect(screen.getByTestId('protected')).toBeInTheDocument();
  });

  it('redirects to /login when token expired and no refresh token', () => {
    userStoreState = {
      isAuthenticated: true,
      accessToken: 'old',
      refreshToken: null,
      isTokenExpired: () => true,
    };
    renderRoute();
    expect(screen.getByTestId('login')).toBeInTheDocument();
  });

  it('renders children when token expired but refresh token still available', () => {
    userStoreState = {
      isAuthenticated: true,
      accessToken: 'old',
      refreshToken: 'refresh',
      isTokenExpired: () => true,
    };
    renderRoute();
    // expired-but-refreshable still renders, http.ts handles silent refresh
    expect(screen.getByTestId('protected')).toBeInTheDocument();
  });

  it('redirects when no access token and no refresh token', () => {
    userStoreState = {
      isAuthenticated: true,
      accessToken: null,
      refreshToken: null,
      isTokenExpired: () => false,
    };
    renderRoute();
    expect(screen.getByTestId('login')).toBeInTheDocument();
  });
});
