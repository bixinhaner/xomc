import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useRef } from 'react';
import { use3DTilt } from '../use3DTilt';

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

describe('use3DTilt', () => {
  it('does not throw when ref is null', () => {
    const { unmount } = renderHook(() => {
      const r = useRef<HTMLDivElement>(null);
      use3DTilt(r);
    });
    unmount();
  });

  it('respects disabled flag — no listeners registered', () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    const addSpy = vi.spyOn(el, 'addEventListener');
    const ref = { current: el };

    const { unmount } = renderHook(() => use3DTilt(ref, { disabled: true }));
    expect(addSpy).not.toHaveBeenCalled();
    unmount();
    document.body.removeChild(el);
  });

  it('promotes element layer + binds enter/move/leave on enabled run', () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    const addSpy = vi.spyOn(el, 'addEventListener');
    const ref = { current: el };

    const { unmount } = renderHook(() => use3DTilt(ref));

    // Cleanup runs when unmount
    unmount();
    // mousemove + mouseenter + mouseleave were bound (3 calls)
    expect(addSpy.mock.calls.length).toBeGreaterThanOrEqual(3);
    document.body.removeChild(el);
  });
});
