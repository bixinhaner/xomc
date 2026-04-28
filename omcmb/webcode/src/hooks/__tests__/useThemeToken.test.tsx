import { describe, it, expect, vi } from 'vitest';
import { renderHook } from '@testing-library/react';

let currentTheme = 'classic';

vi.mock('@core/store/appStore', () => ({
  useAppStore: <T,>(selector: (s: { theme: string }) => T) =>
    selector({ theme: currentTheme }),
}));

vi.mock('antd', () => ({
  theme: {
    useToken: () => ({ token: { colorPrimary: '#1677FF', colorText: '#000' } }),
  },
}));

describe('useIsDark', () => {
  it('returns true for "tech" theme', async () => {
    currentTheme = 'tech';
    const { useIsDark } = await import('../useThemeToken');
    const { result } = renderHook(() => useIsDark());
    expect(result.current).toBe(true);
  });

  it('returns true for "cyberpunk" theme', async () => {
    currentTheme = 'cyberpunk';
    const { useIsDark } = await import('../useThemeToken');
    const { result } = renderHook(() => useIsDark());
    expect(result.current).toBe(true);
  });

  it('returns false for "classic" theme', async () => {
    currentTheme = 'classic';
    const { useIsDark } = await import('../useThemeToken');
    const { result } = renderHook(() => useIsDark());
    expect(result.current).toBe(false);
  });

  it('returns false for "fresh" theme', async () => {
    currentTheme = 'fresh';
    const { useIsDark } = await import('../useThemeToken');
    const { result } = renderHook(() => useIsDark());
    expect(result.current).toBe(false);
  });
});

describe('useThemeToken', () => {
  it('exposes Ant Design token object', async () => {
    const { useThemeToken } = await import('../useThemeToken');
    const { result } = renderHook(() => useThemeToken());
    expect(result.current.colorPrimary).toBe('#1677FF');
  });
});
