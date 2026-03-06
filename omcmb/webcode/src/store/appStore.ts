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
      theme: 'classic',
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
          const themes: Theme[] = ['classic', 'tech', 'fresh', 'cyberpunk', 'minions', 'tiffany', 'rmb'];
          const currentIndex = themes.indexOf(state.theme);
          const next = themes[(currentIndex + 1) % themes.length];
          return { theme: next };
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
      version: 1,
      migrate: (persisted: unknown, version: number) => {
        const state = persisted as Record<string, unknown>;
        if (version === 0 || !version) {
          // Migrate old theme values to new style names
          const oldTheme = state.theme as string;
          if (oldTheme && THEME_MIGRATION[oldTheme]) {
            state.theme = THEME_MIGRATION[oldTheme];
          }
        }
        return state as AppState;
      },
    }
  )
);
