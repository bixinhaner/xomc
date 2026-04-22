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
      setMobileOverlayOpen: (open) => set({ isMobileOverlayOpen: open }),
    }),
    {
      name: 'omc-app-store',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
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
        return state as AppState;
      },
    }
  )
);
