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
  /** save 动作专用:记录被保存的实例号,task=completed 时据此清掉对应行的 rowEdits + draft,
      避免切 tab 走回后乐观显示的旧用户输入覆盖 schema 新值。 */
  savedInstId?: string;
  /** 批量 save 动作:一次保存多个新增实例时,在任务完成后一起清掉乐观编辑态。 */
  savedInstIds?: string[];
}

export type Feedback = CellFeedback | MultiFeedback;

export type QuickSettingsDraftValue =
  | string
  | number
  | boolean
  | null
  | QuickSettingsDraftValue[]
  | { [key: string]: QuickSettingsDraftValue };

export function feedbackKey(
  deviceId: string,
  groupId: string,
  fapInstance: number,
  cellInstance?: number,
): string {
  return cellInstance === undefined
    ? `${deviceId}::${groupId}::${fapInstance}`
    : `${deviceId}::${groupId}::${fapInstance}::${cellInstance}`;
}

interface FeedbackState {
  entries: Record<string, Feedback>;
  /**
   * 未保存的表单草稿，按 feedbackKey 索引。
   * 用于跨顶层 TabBar 切换时保留用户编辑（DeviceDetail 整树卸载，CellParameterForm 内部 form state 丢失）。
   * 保存成功后由调用方 clearDraft 清掉。
   */
  drafts: Record<string, Record<string, QuickSettingsDraftValue>>;
  /**
   * 强制 remount 计数器，按 deviceId 索引。
   * 头部"刷新"按钮 bump 后，QuickSettingsTab 把它拼进子组件 key，触发 CellParameterForm / MultiInstanceTable
   * 整体重挂载，让 form.touched / rowEdits 等组件内 state 全部归零，回到 schema 服务器值。
   */
  refreshTicks: Record<string, number>;

  setFeedback: (key: string, feedback: Feedback) => void;
  patchFeedback: (key: string, patch: Partial<Feedback>) => void;
  clearByDevice: (deviceId: string) => void;

  setDraftField: (key: string, name: string, value: QuickSettingsDraftValue) => void;
  clearDraft: (key: string) => void;
  /**
   * 删除 drafts[key] 下所有以 prefix 开头的字段，用于 MultiInstanceTable 单行 save/delete
   * 后只清理该行的草稿、保留其他行未保存编辑。
   */
  clearDraftPrefix: (key: string, prefix: string) => void;

  bumpRefreshTick: (deviceId: string) => void;
}

export const useQuickSettingsFeedbackStore = create<FeedbackState>()(
  persist(
    (set, get) => ({
      entries: {},
      drafts: {},
      refreshTicks: {},

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
        const nextDrafts: Record<string, Record<string, QuickSettingsDraftValue>> = {};
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

      clearDraftPrefix: (key, prefix) => {
        const cur = get().drafts[key];
        if (!cur) return;
        const filtered: Record<string, QuickSettingsDraftValue> = {};
        for (const [n, v] of Object.entries(cur)) {
          if (!n.startsWith(prefix)) filtered[n] = v;
        }
        const next = { ...get().drafts };
        if (Object.keys(filtered).length === 0) {
          delete next[key];
        } else {
          next[key] = filtered;
        }
        set({ drafts: next });
      },

      bumpRefreshTick: (deviceId) => {
        const cur = get().refreshTicks[deviceId] ?? 0;
        set({ refreshTicks: { ...get().refreshTicks, [deviceId]: cur + 1 } });
      },
    }),
    {
      name: 'omc-quicksettings-feedback',
      storage: createJSONStorage(() => sessionStorage),
    },
  ),
);
