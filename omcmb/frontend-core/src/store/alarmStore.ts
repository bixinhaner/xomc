import { create } from 'zustand';
import type { AlarmSeverity } from '../types/common';
import type { AlarmCount } from '../types/alarm';

export type { AlarmSeverity };

// Legacy alias
export type AlarmCounts = AlarmCount;

interface AlarmState {
  counts: AlarmCount;
  totalActive: number;
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
  totalActive: 0,
  loading: false,

  setCounts: (counts) =>
    set((state) => {
      const next = { ...state.counts, ...counts };
      return {
        counts: next,
        totalActive: next.critical + next.major + next.minor + next.warning,
      };
    }),

  incrementSeverity: (severity) =>
    set((state) => {
      const next = { ...state.counts, [severity]: state.counts[severity] + 1 };
      return { counts: next, totalActive: state.totalActive + 1 };
    }),

  decrementSeverity: (severity) =>
    set((state) => {
      const next = {
        ...state.counts,
        [severity]: Math.max(0, state.counts[severity] - 1),
      };
      return {
        counts: next,
        totalActive: Math.max(0, state.totalActive - 1),
      };
    }),

  incrementCount: (severity) => get().incrementSeverity(severity),
  decrementCount: (severity) => get().decrementSeverity(severity),

  setLoading: (loading) => set({ loading }),
}));
