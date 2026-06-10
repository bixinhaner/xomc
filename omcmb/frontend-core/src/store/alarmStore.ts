import { create } from 'zustand';
import type { AlarmSeverity } from '../types/common';
import type { AlarmCount } from '../types/alarm';

export type { AlarmSeverity };

// Legacy alias
export type AlarmCounts = AlarmCount;

interface AlarmState {
  counts: AlarmCount;
  loading: boolean;

  setCounts: (counts: Partial<AlarmCount>) => void;
  incrementSeverity: (severity: AlarmSeverity) => void;
  decrementSeverity: (severity: AlarmSeverity) => void;
  // Legacy aliases
  incrementCount: (severity: AlarmSeverity) => void;
  decrementCount: (severity: AlarmSeverity) => void;
  setLoading: (loading: boolean) => void;
}

export const useAlarmStore = create<AlarmState>()((set, get) => ({
  counts: {
    total_active: 0,
    unacknowledged: 0,
    unread: 0,
    critical: 0,
    major: 0,
    minor: 0,
    warning: 0,
  },
  loading: false,

  setCounts: (counts) =>
    set((state) => ({ counts: { ...state.counts, ...counts } })),

  incrementSeverity: (severity) =>
    set((state) => ({
      counts: { ...state.counts, [severity]: state.counts[severity] + 1 },
    })),

  decrementSeverity: (severity) =>
    set((state) => ({
      counts: {
        ...state.counts,
        [severity]: Math.max(0, state.counts[severity] - 1),
      },
    })),

  incrementCount: (severity) => get().incrementSeverity(severity),
  decrementCount: (severity) => get().decrementSeverity(severity),

  setLoading: (loading) => set({ loading }),
}));

/**
 * Derived count of active alarms by severity bucket.
 *
 * Replaces the previously hand-synced `totalActive` store field (issue #26): a
 * derived value computed from `counts` so it can never drift out of sync with
 * the source of truth. Use as a memoised selector:
 *   const totalActive = useAlarmStore(selectTotalActive);
 */
export const selectTotalActive = (state: AlarmState): number =>
  state.counts.critical + state.counts.major + state.counts.minor + state.counts.warning;
