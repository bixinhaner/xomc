import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { DeviceType, Theme, Locale, Timezone, SidebarPosition, TabBarPosition } from '../types/common';

// Legacy type aliases for backward compatibility
export type { DeviceType };
export type TimezoneMode = Timezone;
export type ThemeMode = Theme;
export type LocaleCode = Locale;

// Migration: map old theme values to new style names
const THEME_MIGRATION: Record<string, Theme> = {
  light: 'classic',
  dim: 'tech',
  dark: 'tech',
};

interface AppState {
  sidebarCollapsed: boolean;
  sidebarPosition: SidebarPosition;
  tabBarPosition: TabBarPosition;
  deviceType: DeviceType;
  timezone: Timezone;
  theme: Theme;
  locale: Locale;
  effects3DEnabled: boolean;
  /**
   * 动态菜单是否显示图标。来源于 sys_configs.system.show_menu_icon，
   * 启动期从 /admin/public/configs 读取后写入；用户在「菜单管理」页改 Switch
   * 后写回 DB 并同步本地。默认 true（与改造前行为一致）。
   *
   * NavMenu 据此决定是否传 icon 给 antd Menu item — 关掉后 directory/menu
   * 全部以纯文字渲染。
   */
  showMenuIcon: boolean;
  /**
   * OMC 名称（系统品牌标题），来源 sys_configs.basic.mrOMCName（is_public=true）。
   * 启动 / 登录后由 usePublicOmcName 从公开配置通道拉取后写入；persist 到 localStorage，
   * 让冷启动 / 登录页在请求 resolve 前先用缓存名即时渲染，避免标题闪烁。
   *
   * undefined = 未配置 / 未拉到 — 各皮肤标题处自行回退默认名（v1 app.title /
   * v2 'OMC · v2' / v3 'STARFORGE'），不渲染空白。
   */
  omcName: string | undefined;
  /** Mobile sidebar drawer visibility (not persisted) */
  isMobileOverlayOpen: boolean;

  setSidebarCollapsed: (collapsed: boolean) => void;
  toggleSidebar: () => void;
  setSidebarPosition: (pos: SidebarPosition) => void;
  setTabBarPosition: (pos: TabBarPosition) => void;
  setDeviceType: (type: DeviceType) => void;
  setTimezone: (tz: Timezone) => void;
  toggleTimezone: () => void;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
  setLocale: (locale: Locale) => void;
  toggleLocale: () => void;
  setEffects3DEnabled: (enabled: boolean) => void;
  setShowMenuIcon: (show: boolean) => void;
  setOmcName: (name: string | undefined) => void;
  setMobileOverlayOpen: (open: boolean) => void;
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      sidebarPosition: 'left',
      tabBarPosition: 'top',
      deviceType: 'all',
      timezone: 'local',
      theme: 'tech',
      locale: 'zh-CN',
      effects3DEnabled: false,
      showMenuIcon: true,
      omcName: undefined,
      isMobileOverlayOpen: false,

      setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
      toggleSidebar: () =>
        set((state) => {
          // When sidebar is at top, always keep collapsed
          if (state.sidebarPosition === 'top') return {};
          return { sidebarCollapsed: !state.sidebarCollapsed };
        }),
      setSidebarPosition: (pos) =>
        set(pos === 'top' ? { sidebarPosition: pos, sidebarCollapsed: true } : { sidebarPosition: pos }),
      setTabBarPosition: (pos) => set({ tabBarPosition: pos }),
      setDeviceType: (type) => set({ deviceType: type }),
      setTimezone: (tz) => set({ timezone: tz }),
      toggleTimezone: () =>
        set((state) => ({ timezone: state.timezone === 'UTC' ? 'local' : 'UTC' })),
      setTheme: (theme) => set({ theme }),
      toggleTheme: () =>
        set((state) => {
          // Simple light/dark toggle: tech (dark) ↔ fresh (light)
          return { theme: state.theme === 'tech' ? 'fresh' : 'tech' };
        }),
      setLocale: (locale) => set({ locale }),
      toggleLocale: () =>
        set((state) => ({ locale: state.locale === 'zh-CN' ? 'en-US' : 'zh-CN' })),
      setEffects3DEnabled: (enabled) => set({ effects3DEnabled: enabled }),
      setShowMenuIcon: (show) => set({ showMenuIcon: show }),
      setOmcName: (name) => set({ omcName: name }),
      setMobileOverlayOpen: (open) => set({ isMobileOverlayOpen: open }),
    }),
    {
      name: 'omc-app-store',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => {
         
        const { isMobileOverlayOpen, ...rest } = state;
        return rest;
      },
      version: 2,
      migrate: (persisted: unknown, version: number) => {
        const state = persisted as Record<string, unknown>;
        if (version === 0 || !version) {
          const oldTheme = state.theme as string;
          if (oldTheme && THEME_MIGRATION[oldTheme]) {
            state.theme = THEME_MIGRATION[oldTheme];
          }
        }
        // v1→v2: force theme to tech (Linear design)
        if (version < 2) {
          state.theme = 'tech';
        }
        return state as unknown as AppState;
      },
    }
  )
);
