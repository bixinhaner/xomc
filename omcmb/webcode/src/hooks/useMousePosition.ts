import { useEffect, useSyncExternalStore } from 'react';

interface MousePosition {
  x: number;
  y: number;
  /** Normalized X: -1 (left) to 1 (right) relative to viewport center */
  nx: number;
  /** Normalized Y: -1 (top) to 1 (bottom) relative to viewport center */
  ny: number;
}

const DEFAULT: MousePosition = { x: 0, y: 0, nx: 0, ny: 0 };

let current: MousePosition = DEFAULT;
const listeners = new Set<() => void>();

function subscribe(cb: () => void) {
  listeners.add(cb);
  return () => { listeners.delete(cb); };
}

function getSnapshot() {
  return current;
}

// Single global listener — initialised lazily on first subscribe
let initialized = false;

function ensureListener() {
  if (initialized) return;
  initialized = true;

  // Check reduced motion preference
  const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
  if (mq.matches) return;

  let rafId = 0;
  let latestPos: { clientX: number; clientY: number } | null = null;

  const flush = () => {
    rafId = 0;
    if (!latestPos) return;
    const { clientX, clientY } = latestPos;
    const nx = (clientX / window.innerWidth) * 2 - 1;
    const ny = (clientY / window.innerHeight) * 2 - 1;
    current = { x: clientX, y: clientY, nx, ny };
    listeners.forEach((cb) => cb());
  };

  window.addEventListener(
    'mousemove',
    (e: MouseEvent) => {
      latestPos = e;
      if (!rafId) rafId = requestAnimationFrame(flush);
    },
    { passive: true },
  );

  // Touch support — same rAF throttle
  window.addEventListener(
    'touchmove',
    (e: TouchEvent) => {
      const touch = e.touches[0];
      if (!touch) return;
      latestPos = { clientX: touch.clientX, clientY: touch.clientY };
      if (!rafId) rafId = requestAnimationFrame(flush);
    },
    { passive: true },
  );
}

/**
 * Global mouse position hook.
 * Uses a single shared mousemove listener with rAF throttling.
 * Returns { x, y, nx, ny } where nx/ny are normalised to [-1, 1].
 */
export function useMousePosition(): MousePosition {
  // 初始化监听器（只执行一次）
  useEffect(() => {
    ensureListener();
  }, []);

  return useSyncExternalStore(subscribe, getSnapshot, () => DEFAULT);
}
