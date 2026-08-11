import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';

interface UserStoreShape {
  isAuthenticated: boolean;
  accessToken: string | null;
  refreshToken: string | null;
  isTokenExpired: () => boolean;
  currentUser?: {
    role: string;
    isSuperAdmin?: boolean;
  };
}

let userStoreState: UserStoreShape;
let systemLicenseState: {
  data: { id: string } | undefined;
  isLoading: boolean;
  error: unknown;
};

vi.mock('@core/store/userStore', () => ({
  useUserStore: () => userStoreState,
}));
vi.mock('@core/hooks/api/useSystemLicense', () => ({
  useSystemLicense: () => systemLicenseState,
}));

import PrivateRoute from '../PrivateRoute';

function renderRoute(initialPath = '/protected', requireSuperAdmin = false) {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route
          path="/protected"
          element={
            <PrivateRoute requireSuperAdmin={requireSuperAdmin}>
              <div data-testid="protected">secret</div>
            </PrivateRoute>
          }
        />
        <Route path="/login" element={<div data-testid="login">login page</div>} />
        <Route path="/403" element={<div data-testid="forbidden">forbidden</div>} />
        <Route
          path="/license"
          element={
            <PrivateRoute>
              <div data-testid="license">license page</div>
            </PrivateRoute>
          }
        />
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
  systemLicenseState = {
    data: { id: 'license-1' },
    isLoading: false,
    error: null,
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

  it('redirects a normal admin away from a super-admin-only route', () => {
    userStoreState = {
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      isTokenExpired: () => false,
      currentUser: { role: 'admin', isSuperAdmin: false },
    };

    renderRoute('/protected', true);

    expect(screen.getByTestId('forbidden')).toBeInTheDocument();
    expect(screen.queryByTestId('protected')).not.toBeInTheDocument();
  });

  it('renders a super-admin-only route for a super admin', () => {
    userStoreState = {
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      isTokenExpired: () => false,
      currentUser: { role: 'admin', isSuperAdmin: true },
    };

    renderRoute('/protected', true);

    expect(screen.getByTestId('protected')).toBeInTheDocument();
    expect(screen.queryByTestId('forbidden')).not.toBeInTheDocument();
  });

  it('redirects non-license routes to license when no license is configured', () => {
    userStoreState = {
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      isTokenExpired: () => false,
      currentUser: { role: 'admin', isSuperAdmin: true },
    };
    systemLicenseState = {
      data: undefined,
      isLoading: false,
      error: { bizCode: 12113 },
    };

    renderRoute();

    expect(screen.getByTestId('license')).toBeInTheDocument();
    expect(screen.queryByTestId('protected')).not.toBeInTheDocument();
  });

  it('keeps the license route available when no license is configured', () => {
    userStoreState = {
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      isTokenExpired: () => false,
    };
    systemLicenseState = {
      data: undefined,
      isLoading: false,
      error: { bizCode: 12113 },
    };

    renderRoute('/license');

    expect(screen.getByTestId('license')).toBeInTheDocument();
  });
});
