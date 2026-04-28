import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';

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

describe('useMousePosition', () => {
  it('returns initial { x:0, y:0, nx:0, ny:0 }', async () => {
    const { useMousePosition } = await import('../useMousePosition');
    const { result } = renderHook(() => useMousePosition());
    expect(result.current).toEqual({ x: 0, y: 0, nx: 0, ny: 0 });
  });

  it('updates after a window mousemove with rAF flush', async () => {
    const { useMousePosition } = await import('../useMousePosition');
    Object.defineProperty(window, 'innerWidth', { value: 1000, writable: true, configurable: true });
    Object.defineProperty(window, 'innerHeight', { value: 500, writable: true, configurable: true });
    const { result } = renderHook(() => useMousePosition());

    await act(async () => {
      window.dispatchEvent(
        new MouseEvent('mousemove', { clientX: 750, clientY: 250 }),
      );
      // flush rAF — jsdom polyfills rAF as setTimeout(_, ~16ms)
      await new Promise((r) => setTimeout(r, 32));
    });

    // After moving to (750, 250) of a 1000x500 viewport, nx = 0.5, ny = 0
    expect(result.current.x).toBe(750);
    expect(result.current.y).toBe(250);
    expect(result.current.nx).toBeCloseTo(0.5, 2);
    expect(result.current.ny).toBeCloseTo(0, 2);
  });
});
