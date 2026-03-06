import { useLayoutEffect, useMemo, type ReactNode } from 'react';
import type { ThemeConfig } from 'antd';
import { ConfigProvider } from 'antd';
import antdZhCN from 'antd/locale/zh_CN';
import antdEnUS from 'antd/locale/en_US';
import { useAppStore } from '@/store/appStore';
import { antdClassicTheme } from '@/theme/classicTheme';
import { antdTechTheme } from '@/theme/techTheme';
import { antdFreshTheme } from '@/theme/freshTheme';
import { antdCyberpunkTheme } from '@/theme/cyberpunkTheme';
import { antdMinionsTheme } from '@/theme/minionsTheme';
import { antdTiffanyTheme } from '@/theme/tiffanyTheme';
import { antdRmbTheme } from '@/theme/rmbTheme';
import type { Theme, Locale } from '@/types/common';
import type { Locale as AntdLocale } from 'antd/es/locale';

interface ThemeProviderProps {
  children: ReactNode;
}

const themeConfigMap: Record<Theme, ThemeConfig> = {
  classic: antdClassicTheme,
  tech: antdTechTheme,
  fresh: antdFreshTheme,
  cyberpunk: antdCyberpunkTheme,
  minions: antdMinionsTheme,
  tiffany: antdTiffanyTheme,
  rmb: antdRmbTheme,
};

const antdLocaleMap: Record<Locale, AntdLocale> = {
  'zh-CN': antdZhCN,
  'en-US': antdEnUS,
};

/**
 * ThemeProvider reads the current theme + locale from appStore and:
 * 1. Applies a `data-theme` attribute to <html> via useLayoutEffect (before paint)
 *    so CSS custom properties are ready when Ant Design tokens take effect.
 * 2. Wraps children with a SINGLE Ant Design ConfigProvider carrying both
 *    theme and locale config, avoiding the dual-ConfigProvider cascade that
 *    caused blank pages on theme switch.
 */
export default function ThemeProvider({ children }: ThemeProviderProps) {
  const theme = useAppStore((s) => s.theme);
  const locale = useAppStore((s) => s.locale);

  // useLayoutEffect: set data-theme BEFORE browser paint so CSS variables
  // are in sync with Ant Design tokens on the same frame.
  useLayoutEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    const isDark = theme === 'tech' || theme === 'cyberpunk';
    document.documentElement.style.colorScheme = isDark ? 'dark' : 'light';
  }, [theme]);

  const antdLocale = antdLocaleMap[locale] ?? antdZhCN;

  // Memoize to keep a stable object reference — prevents unnecessary
  // ConfigProvider context re-creation when neither theme nor locale changed.
  const themeConfig = useMemo(() => themeConfigMap[theme], [theme]);

  return (
    <ConfigProvider theme={themeConfig} locale={antdLocale}>
      {children}
    </ConfigProvider>
  );
}
