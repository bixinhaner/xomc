import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { User } from '../types/system';
import { useMenuStore } from './menuStore';

export type { User };
// Legacy alias
export type UserInfo = User & { permissions: string[]; avatar?: string; lastLogin?: string };
export type UserRole = 'admin' | 'operator' | 'viewer' | 'auditor';

export interface TokenPairResponse {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  token_type?: string;
}

interface UserState {
  currentUser: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  tokenExpiresAt: number | null; // Unix timestamp in ms
  isAuthenticated: boolean;
  permissions: string[];
  loading: boolean;

  // JWT token pair methods
  setTokenPair: (pair: TokenPairResponse) => void;
  clearAuth: () => void;
  isTokenExpired: () => boolean;

  // Existing methods
  login: (user: User) => void;
  logout: () => void;
  setPermissions: (permissions: string[]) => void;
  setToken: (token: string) => void; // Legacy compat — sets accessToken
  setLoading: (loading: boolean) => void;
  hasPermission: (permission: string) => boolean;
  // Legacy alias
  setUser: (user: User) => void;
}

export const useUserStore = create<UserState>()(
  persist(
    (set, get) => ({
      currentUser: null,
      accessToken: null,
      refreshToken: null,
      tokenExpiresAt: null,
      isAuthenticated: false,
      permissions: [],
      loading: false,

      login: (user) => set({ currentUser: user, isAuthenticated: true }),
      setUser: (user) => set({ currentUser: user, isAuthenticated: true }),

      setTokenPair: (pair: TokenPairResponse) => {
        if (!pair?.access_token || !pair?.refresh_token || !pair?.expires_at) {
          throw new Error('setTokenPair: missing fields in token pair');
        }
        const expiresAt = new Date(pair.expires_at).getTime();
        if (Number.isNaN(expiresAt)) {
          throw new Error('setTokenPair: invalid expires_at format');
        }
        set({
          accessToken: pair.access_token,
          refreshToken: pair.refresh_token,
          tokenExpiresAt: expiresAt,
          isAuthenticated: true,
        });
      },

      clearAuth: () => {
        set({
          currentUser: null,
          accessToken: null,
          refreshToken: null,
          tokenExpiresAt: null,
          isAuthenticated: false,
          permissions: [],
        });
        localStorage.removeItem('omc-user-store');
        // 退出 / Token 失效时同步清空菜单缓存，防止下一个用户登录时
        // persist 残留指向上一个用户的角色菜单（PRD §6 风险表 / §3.3 #7）。
        useMenuStore.getState().clear();
        localStorage.removeItem('omc-menu-store');
      },

      isTokenExpired: () => {
        const { tokenExpiresAt } = get();
        if (!tokenExpiresAt) return true;
        // Consider expired 30 seconds early to allow refresh
        return Date.now() > tokenExpiresAt - 30_000;
      },

      logout: () => {
        get().clearAuth();
      },

      setPermissions: (permissions) => set({ permissions }),

      // Legacy compat — sets accessToken
      setToken: (token) => set({ accessToken: token }),

      setLoading: (loading) => set({ loading }),

      hasPermission: (permission) => {
        const { currentUser, permissions } = get();
        if (!currentUser) return false;
        if (currentUser.role === 'admin') return true;
        return permissions.includes(permission);
      },
    }),
    {
      name: 'omc-user-store',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        currentUser: state.currentUser,
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        tokenExpiresAt: state.tokenExpiresAt,
        isAuthenticated: state.isAuthenticated,
        permissions: state.permissions,
      }),
    }
  )
);
