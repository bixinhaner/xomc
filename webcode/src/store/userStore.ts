import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '../types/system';

export type { User };
// Legacy alias
export type UserInfo = User & { permissions: string[]; avatar?: string; lastLogin?: string };
export type UserRole = 'admin' | 'operator' | 'viewer' | 'auditor';

interface UserState {
  currentUser: User | null;
  token: string | null;
  isAuthenticated: boolean;
  permissions: string[];
  loading: boolean;

  login: (user: User) => void;
  logout: () => void;
  setPermissions: (permissions: string[]) => void;
  setToken: (token: string) => void;
  setLoading: (loading: boolean) => void;
  hasPermission: (permission: string) => boolean;
  // Legacy alias
  setUser: (user: User) => void;
}

export const useUserStore = create<UserState>()(
  persist(
    (set, get) => ({
      currentUser: null,
      token: null,
      isAuthenticated: false,
      permissions: [],
      loading: false,

      login: (user) => set({ currentUser: user, isAuthenticated: true }),
      setUser: (user) => set({ currentUser: user, isAuthenticated: true }),

      logout: () => {
        set({ currentUser: null, token: null, isAuthenticated: false, permissions: [] });
        localStorage.removeItem('omc-user-store');
      },

      setPermissions: (permissions) => set({ permissions }),

      setToken: (token) => set({ token }),

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
      partialize: (state) => ({
        currentUser: state.currentUser,
        token: state.token,
        isAuthenticated: state.isAuthenticated,
        permissions: state.permissions,
      }),
    }
  )
);
