import { useSyncExternalStore } from 'react';
import { SCREEN_SM, SCREEN_MD, SCREEN_LG, SCREEN_XL } from '@/theme/tokens';

export type Breakpoint = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

export interface ResponsiveState {
  breakpoint: Breakpoint;
  /** Viewport < 768px */
  isMobile: boolean;
  /** 768px <= Viewport < 992px */
  isTablet: boolean;
  /** Viewport >= 992px */
  isDesktop: boolean;
}

const DESKTOP_DEFAULT: ResponsiveState = {
  breakpoint: 'lg',
  isMobile: false,
  isTablet: false,
  isDesktop: true,
};

// Ordered largest-first so first match wins
const QUERIES = [
  { bp: 'xl' as const, mq: `(min-width: ${SCREEN_XL}px)` },
  { bp: 'lg' as const, mq: `(min-width: ${SCREEN_LG}px)` },
  { bp: 'md' as const, mq: `(min-width: ${SCREEN_MD}px)` },
  { bp: 'sm' as const, mq: `(min-width: ${SCREEN_SM}px)` },
];

function calc(): ResponsiveState {
  let bp: Breakpoint = 'xs';
  for (const q of QUERIES) {
    if (window.matchMedia(q.mq).matches) {
      bp = q.bp;
      break;
    }
  }
  return {
    breakpoint: bp,
    isMobile: bp === 'xs' || bp === 'sm',
    isTablet: bp === 'md',
    isDesktop: bp === 'lg' || bp === 'xl',
  };
}

let current: ResponsiveState = DESKTOP_DEFAULT;
const listeners = new Set<() => void>();

function subscribe(cb: () => void) {
  listeners.add(cb);
  return () => { listeners.delete(cb); };
}

function getSnapshot() {
  return current;
}

function getServerSnapshot() {
  return DESKTOP_DEFAULT;
}

let initialized = false;

function ensureListener() {
  if (initialized) return;
  initialized = true;

  current = calc();

  // Listen to each breakpoint boundary change
  for (const q of QUERIES) {
    const mql = window.matchMedia(q.mq);
    mql.addEventListener('change', () => {
      const next = calc();
      if (next.breakpoint !== current.breakpoint) {
        current = next;
        listeners.forEach((cb) => cb());
      }
    });
  }
}

/**
 * Reactive breakpoint hook.
 * Uses matchMedia change events (not resize) for efficiency.
 * Returns { breakpoint, isMobile, isTablet, isDesktop }.
 */
export function useResponsive(): ResponsiveState {
  if (!initialized) ensureListener();
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
