import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';

let appState: { theme: string; locale: string };

vi.mock('@core/store/appStore', () => ({
  useAppStore: <T,>(selector: (s: { theme: string; locale: string }) => T) =>
    selector(appState),
}));

beforeEach(() => {
  appState = { theme: 'classic', locale: 'zh-CN' };
  document.documentElement.removeAttribute('data-theme');
  document.documentElement.style.colorScheme = '';
});

describe('ThemeProvider', () => {
  it('sets data-theme attribute on <html> matching the active theme', async () => {
    const ThemeProvider = (await import('../ThemeProvider')).default;
    render(
      <ThemeProvider>
        <span data-testid="child">x</span>
      </ThemeProvider>,
    );
    expect(document.documentElement.getAttribute('data-theme')).toBe('classic');
    expect(screen.getByTestId('child').textContent).toBe('x');
  });

  it('uses dark colorScheme for tech theme', async () => {
    appState = { theme: 'tech', locale: 'zh-CN' };
    const ThemeProvider = (await import('../ThemeProvider')).default;
    render(
      <ThemeProvider>
        <span data-testid="child">x</span>
      </ThemeProvider>,
    );
    expect(document.documentElement.getAttribute('data-theme')).toBe('tech');
    expect(document.documentElement.style.colorScheme).toBe('dark');
  });

  it('uses light colorScheme for non-dark themes', async () => {
    appState = { theme: 'fresh', locale: 'en-US' };
    const ThemeProvider = (await import('../ThemeProvider')).default;
    render(
      <ThemeProvider>
        <span data-testid="child">x</span>
      </ThemeProvider>,
    );
    expect(document.documentElement.style.colorScheme).toBe('light');
  });

  it('falls back to zh-CN antd locale for unknown locale codes', async () => {
    // Force-cast a locale that's not in the map: provider should not throw.
    appState = { theme: 'classic', locale: 'fr-FR' as unknown as string };
    const ThemeProvider = (await import('../ThemeProvider')).default;
    expect(() => {
      render(
        <ThemeProvider>
          <span>ok</span>
        </ThemeProvider>,
      );
    }).not.toThrow();
  });
});
