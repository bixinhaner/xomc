import { theme } from 'antd';
import { useAppStore } from '@core/store/appStore';

/**
 * Returns whether the current theme is a dark variant (tech or cyberpunk style).
 * Useful for components that need boolean dark-mode awareness
 * (e.g. ECharts, third-party libs, inline style conditionals).
 */
export function useIsDark() {
  const t = useAppStore((s) => s.theme);
  return t === 'tech' || t === 'cyberpunk';
}

/**
 * Re-exports Ant Design's useToken for convenience.
 * Use `token.colorText`, `token.colorBgContainer`, etc.
 * These values automatically respond to the active Ant Design theme.
 */
export function useThemeToken() {
  const { token } = theme.useToken();
  return token;
}
