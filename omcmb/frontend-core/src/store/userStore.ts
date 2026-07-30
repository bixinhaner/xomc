import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { User } from '../types/system';
import { useMenuStore } from './menuStore';
import { usePmPageStateStore } from './pmPageStateStore';
import { useTabStore } from './tabStore';

export type { User };
// Legacy alias
export type UserInfo = User & { permissions: string[]; avatar?: string; lastLogin?: string };
export type UserRole = 'admin' | 'operator' | 'viewer' | 'auditor';

const LOCKED_SESSION_KEY = 'omc-locked-session';

function safeRemoveStorage(storage: Storage, key: string): void {
  try {
    storage.removeItem(key);
  } catch {
    // Storage can be disabled by browser policy. Auth cleanup must still continue.
  }
}

export interface LockedSession {
  userId: string;
  username: string;
  displayName: string;
  returnPath: string;
}

export function getLockedSession(): LockedSession | null {
  try {
    const raw = sessionStorage.getItem(LOCKED_SESSION_KEY);
    if (!raw) return null;
    const value = JSON.parse(raw) as Partial<LockedSession>;
    if (
      typeof value.userId !== 'string' ||
      typeof value.username !== 'string' ||
      typeof value.displayName !== 'string' ||
      typeof value.returnPath !== 'string' ||
      !value.returnPath.startsWith('/') ||
      value.returnPath.startsWith('//')
    ) {
      safeRemoveStorage(sessionStorage, LOCKED_SESSION_KEY);
      return null;
    }
    return value as LockedSession;
  } catch {
    safeRemoveStorage(sessionStorage, LOCKED_SESSION_KEY);
    return null;
  }
}

function clearPersistedTabs(): void {
  useTabStore.getState().closeAllTabs();
  usePmPageStateStore.getState().clearAllPageStates();
  safeRemoveStorage(sessionStorage, 'omc-pm-page-state-store');
  safeRemoveStorage(sessionStorage, 'omc-tab-store');
}

function prepareUserSessionFor(user: User, currentUser: User | null): void {
  if (currentUser && currentUser.id !== user.id) {
    clearPersistedTabs();
  }
  const lockedSession = getLockedSession();
  if (lockedSession && lockedSession.userId !== user.id) {
    clearPersistedTabs();
  }
  safeRemoveStorage(sessionStorage, LOCKED_SESSION_KEY);
}

export interface TokenPairResponse {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  token_type?: string;
  // P1 密码策略派生字段（仅 Login 响应；refresh 不带）
  must_change_password?: boolean;
  password_expires_in_days?: number;
  // P2-⑪ 登录提示文案（管理员配置；空 / 未启用时不下发）
  login_notify_msg?: string;
}

interface UserState {
  currentUser: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  tokenExpiresAt: number | null; // Unix timestamp in ms
  isAuthenticated: boolean;
  permissions: string[];
  loading: boolean;
  // Issue #649：必须修改密码标记（后端 login 响应 must_change_password=true 时置位）。
  // UI 层读取该标记决定是否弹阐塞改密 Modal。
  mustChangePassword: boolean;

  // JWT token pair methods
  setTokenPair: (pair: TokenPairResponse) => void;
  clearAuth: () => void;
  lock: (returnPath?: string) => void;
  isTokenExpired: () => boolean;

  // Existing methods
  login: (user: User) => void;
  logout: () => void;
  setPermissions: (permissions: string[]) => void;
  setToken: (token: string) => void; // Legacy compat — sets accessToken
  setLoading: (loading: boolean) => void;
  hasPermission: (permission: string) => boolean;
  // Issue #649：设置 mustChangePassword 标记。
  setMustChangePassword: (v: boolean) => void;
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
      mustChangePassword: false,

      login: (user) => {
        prepareUserSessionFor(user, get().currentUser);
        set({ currentUser: user, isAuthenticated: true });
      },
      setUser: (user) => {
        prepareUserSessionFor(user, get().currentUser);
        set({ currentUser: user, isAuthenticated: true });
      },

      setMustChangePassword: (v: boolean) => set({ mustChangePassword: v }),

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

      lock: (returnPath = '/dashboard') => {
        const lockedUser = get().currentUser;
        let snapshotSaved = false;
        if (lockedUser) {
          const lockedSession: LockedSession = {
            userId: lockedUser.id,
            username: lockedUser.username,
            displayName: lockedUser.displayName,
            returnPath,
          };
          try {
            sessionStorage.setItem(LOCKED_SESSION_KEY, JSON.stringify(lockedSession));
            snapshotSaved = true;
          } catch {
            // 锁屏快照保存失败时降级为普通退出，但绝不能阻止 Token 清理。
          }
        }
        set({
          currentUser: null,
          accessToken: null,
          refreshToken: null,
          tokenExpiresAt: null,
          isAuthenticated: false,
          permissions: [],
          mustChangePassword: false,
        });
        safeRemoveStorage(localStorage, 'omc-user-store');
        // 退出 / Token 失效时同步清空菜单缓存，防止下一个用户登录时
        // persist 残留指向上一个用户的角色菜单（PRD §6 风险表 / §3.3 #7）。
        useMenuStore.getState().clear();
        safeRemoveStorage(localStorage, 'omc-menu-store');
        if (lockedUser && !snapshotSaved) {
          clearPersistedTabs();
          safeRemoveStorage(sessionStorage, LOCKED_SESSION_KEY);
        }
      },

      clearAuth: () => {
        get().lock();
        // 同步清空标签页（sessionStorage 存），避免下一个用户进来后
        // 右侧仍残留上一个用户打开过的页面。
        clearPersistedTabs();
        safeRemoveStorage(sessionStorage, LOCKED_SESSION_KEY);
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
        // Issue #649：持久化 mustChangePassword，防止用户刷新页面或关闭浏览器
        // 后未改密就继续使用系统（token 还在有效期内会被静默放行）。
        mustChangePassword: state.mustChangePassword,
      }),
    }
  )
);
