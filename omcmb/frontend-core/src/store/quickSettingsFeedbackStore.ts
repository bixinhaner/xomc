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
  /**
   * 未保存的表单草稿，按 feedbackKey 索引。
   * 用于跨顶层 TabBar 切换时保留用户编辑（DeviceDetail 整树卸载，CellParameterForm 内部 form state 丢失）。
   * 保存成功后由调用方 clearDraft 清掉。
   */
  drafts: Record<string, Record<string, string>>;

  setFeedback: (key: string, feedback: Feedback) => void;
  patchFeedback: (key: string, patch: Partial<Feedback>) => void;
  clearByDevice: (deviceId: string) => void;

  setDraftField: (key: string, name: string, value: string) => void;
  clearDraft: (key: string) => void;
}

export const useQuickSettingsFeedbackStore = create<FeedbackState>()(
  persist(
    (set, get) => ({
      entries: {},
      drafts: {},

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
        const nextDrafts: Record<string, Record<string, string>> = {};
        const prefix = `${deviceId}::`;
        for (const [k, v] of Object.entries(get().entries)) {
          if (!k.startsWith(prefix)) next[k] = v;
        }
        for (const [k, v] of Object.entries(get().drafts)) {
          if (!k.startsWith(prefix)) nextDrafts[k] = v;
        }
        set({ entries: next, drafts: nextDrafts });
      },

      setDraftField: (key, name, value) => {
        const cur = get().drafts[key] ?? {};
        set({ drafts: { ...get().drafts, [key]: { ...cur, [name]: value } } });
      },

      clearDraft: (key) => {
        const next = { ...get().drafts };
        delete next[key];
        set({ drafts: next });
      },
    }),
    {
      name: 'omc-quicksettings-feedback',
      storage: createJSONStorage(() => sessionStorage),
    },
  ),
);
