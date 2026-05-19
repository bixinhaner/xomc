import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

export interface CellFeedback {
  kind: 'cell';
  submitStatus: 'queued' | 'failed_to_queue';
  taskId?: string;
  count: number;
  at: number;
  errorMsg?: string;
  notifiedFailedTaskId?: string;
}

export interface MultiFeedback {
  kind: 'multi';
  action: 'save' | 'add' | 'delete';
  submitStatus: 'queued' | 'failed_to_queue';
  taskId?: string;
  detail: string;
  at: number;
  notifiedFailedTaskId?: string;
}

export type Feedback = CellFeedback | MultiFeedback;

export function feedbackKey(deviceId: string, groupId: string, fapInstance: number): string {
  return `${deviceId}::${groupId}::${fapInstance}`;
}

interface FeedbackState {
  entries: Record<string, Feedback>;
  setFeedback: (key: string, feedback: Feedback) => void;
  patchFeedback: (key: string, patch: Partial<Feedback>) => void;
  clearByDevice: (deviceId: string) => void;
}

export const useQuickSettingsFeedbackStore = create<FeedbackState>()(
  persist(
    (set, get) => ({
      entries: {},

      setFeedback: (key, feedback) => {
        set({ entries: { ...get().entries, [key]: feedback } });
      },

      patchFeedback: (key, patch) => {
        const cur = get().entries[key];
        if (!cur) return;
        set({ entries: { ...get().entries, [key]: { ...cur, ...patch } as Feedback } });
      },

      clearByDevice: (deviceId) => {
        const next: Record<string, Feedback> = {};
        const prefix = `${deviceId}::`;
        for (const [k, v] of Object.entries(get().entries)) {
          if (!k.startsWith(prefix)) next[k] = v;
        }
        set({ entries: next });
      },
    }),
    {
      name: 'omc-quicksettings-feedback',
      storage: createJSONStorage(() => sessionStorage),
    },
  ),
);
