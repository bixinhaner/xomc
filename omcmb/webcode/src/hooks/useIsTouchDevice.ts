import { useSyncExternalStore } from 'react';

interface TouchState {
  /** Primary pointer is coarse (finger) rather than fine (mouse) */
  isTouchPrimary: boolean;
  /** Device has any touch capability (includes hybrid laptop+touchscreen) */
  hasTouchSupport: boolean;
}

const DEFAULT: TouchState = { isTouchPrimary: false, hasTouchSupport: false };

let current: TouchState = DEFAULT;
const listeners = new Set<() => void>();

function subscribe(cb: () => void) {
  listeners.add(cb);
  return () => { listeners.delete(cb); };
}

function getSnapshot() {
  return current;
}

function getServerSnapshot() {
  return DEFAULT;
}

let initialized = false;

function ensureListener() {
  if (initialized) return;
  initialized = true;

  const coarseMql = window.matchMedia('(pointer: coarse)');
  const hasTouchSupport = 'ontouchstart' in window || navigator.maxTouchPoints > 0;

  current = { isTouchPrimary: coarseMql.matches, hasTouchSupport };

  coarseMql.addEventListener('change', (e) => {
    current = { isTouchPrimary: e.matches, hasTouchSupport };
    listeners.forEach((cb) => cb());
  });
}

/**
 * Detects touch input capability.
 * `isTouchPrimary` — the main pointer is a finger (phone/tablet).
 * `hasTouchSupport` — device has any touch support (includes hybrid laptops).
 */
export function useIsTouchDevice(): TouchState {
  if (!initialized) ensureListener();
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
