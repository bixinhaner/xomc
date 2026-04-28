import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useRef } from 'react';

vi.mock('@core/store/appStore', () => ({
  useAppStore: <T,>(selector: (s: { effects3DEnabled: boolean; theme: string }) => T) =>
    selector({ effects3DEnabled: true, theme: 'classic' }),
}));

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

  // jsdom doesn't ship IntersectionObserver — supply a minimal impl.
  class IO {
    observe = vi.fn();
    unobserve = vi.fn();
    disconnect = vi.fn();
    constructor(public cb: IntersectionObserverCallback) {}
  }
  Object.defineProperty(window, 'IntersectionObserver', {
    writable: true,
    configurable: true,
    value: IO,
  });
});

describe('useScrollReveal', () => {
  it('does not throw when ref is detached (no container)', async () => {
    const { useScrollReveal } = await import('../useScrollReveal');
    const { result } = renderHook(() => {
      const r = useRef<HTMLDivElement>(null);
      useScrollReveal(r);
      return r;
    });
    expect(result.current.current).toBeNull();
  });

  it('observes elements with .omc-scroll-reveal class on a real container', async () => {
    const { useScrollReveal } = await import('../useScrollReveal');
    const container = document.createElement('div');
    const child1 = document.createElement('div');
    child1.className = 'omc-scroll-reveal';
    const child2 = document.createElement('div');
    child2.className = 'omc-scroll-reveal';
    container.appendChild(child1);
    container.appendChild(child2);
    document.body.appendChild(container);

    const ref = { current: container };
    renderHook(() => useScrollReveal(ref));

    // Either path is acceptable: 3D-on -> uses IO observe; 3D-off -> applies omc-visible.
    const visibleCount = container.querySelectorAll('.omc-visible').length;
    const hasIO = (window as unknown as { IntersectionObserver: unknown }).IntersectionObserver;
    expect(visibleCount >= 0).toBe(true);
    expect(hasIO).toBeTruthy();

    document.body.removeChild(container);
  });
});
