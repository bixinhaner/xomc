import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';

// matchMedia must be installed BEFORE importing the hook because the hook
// caches a module-level `current` snapshot at first call.
type MqInstance = {
  matches: boolean;
  media: string;
  onchange: null;
  addEventListener: ReturnType<typeof vi.fn>;
  removeEventListener: ReturnType<typeof vi.fn>;
  addListener: ReturnType<typeof vi.fn>;
  removeListener: ReturnType<typeof vi.fn>;
  dispatchEvent: ReturnType<typeof vi.fn>;
};

let activeWidth = 1280; // desktop default

function createMatchMedia(query: string): MqInstance {
  const minWidthMatch = /min-width:\s*(\d+)px/.exec(query);
  const min = minWidthMatch ? parseInt(minWidthMatch[1], 10) : 0;
  return {
    matches: activeWidth >= min,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  };
}

beforeEach(() => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: createMatchMedia,
  });
});

describe('useResponsive', () => {
  it('returns a stable shape with breakpoint + boolean flags', async () => {
    const { useResponsive } = await import('../useResponsive');
    const { result } = renderHook(() => useResponsive());
    expect(result.current).toHaveProperty('breakpoint');
    expect(result.current).toHaveProperty('isMobile');
    expect(result.current).toHaveProperty('isTablet');
    expect(result.current).toHaveProperty('isDesktop');
    expect(typeof result.current.isMobile).toBe('boolean');
    expect(typeof result.current.isTablet).toBe('boolean');
    expect(typeof result.current.isDesktop).toBe('boolean');
  });

  it('reports a valid breakpoint enum', async () => {
    const { useResponsive } = await import('../useResponsive');
    const { result } = renderHook(() => useResponsive());
    expect(['xs', 'sm', 'md', 'lg', 'xl']).toContain(result.current.breakpoint);
  });
});
