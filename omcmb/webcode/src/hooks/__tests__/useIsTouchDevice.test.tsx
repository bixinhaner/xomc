import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';

beforeEach(() => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

describe('useIsTouchDevice', () => {
  it('returns isTouchPrimary + hasTouchSupport booleans', async () => {
    const { useIsTouchDevice } = await import('../useIsTouchDevice');
    const { result } = renderHook(() => useIsTouchDevice());
    expect(result.current).toHaveProperty('isTouchPrimary');
    expect(result.current).toHaveProperty('hasTouchSupport');
    expect(typeof result.current.isTouchPrimary).toBe('boolean');
    expect(typeof result.current.hasTouchSupport).toBe('boolean');
  });

  it('isTouchPrimary is false when matchMedia(pointer:coarse) is false', async () => {
    const { useIsTouchDevice } = await import('../useIsTouchDevice');
    const { result } = renderHook(() => useIsTouchDevice());
    expect(result.current.isTouchPrimary).toBe(false);
  });
});
