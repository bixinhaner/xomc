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
  /** add_rollback:add 阶段 AddObject 成功但 SPV 部分参数被拒,顶层 useEffect 触发的自动 DeleteObject。 */
  action: 'save' | 'add' | 'delete' | 'add_rollback';
  submitStatus: 'queued' | 'failed_to_queue';
  taskId?: string;
  detail: string;
  at: number;
  notifiedFailedTaskId?: string;
  /** 终态 task → 查询失效仅做一次的去重标记(避免 useEffect 反复触发)。 */
  invalidatedForTaskId?: string;
  /** 终态 task → 相关多实例参数已回读并刷新列表后才允许展示成功态。 */
  syncedForTaskId?: string;
  /** save 动作专用:记录被保存的实例号,task=completed 时据此清掉对应行的 rowEdits + draft,
      避免切 tab 走回后乐观显示的旧用户输入覆盖 schema 新值。 */
  savedInstId?: string;
  /** 批量 save 动作:一次保存多个新增实例时,在任务完成后一起清掉乐观编辑态。 */
  savedInstIds?: string[];
  /** save 动作来源:区分新增后的 SPV 与编辑已有实例的 SPV,避免把编辑失败误当新增失败回滚。 */
  saveMode?: 'add' | 'edit';
  /** BSC add 动作:AddObject 创建出来的实例号,用于 SPV 失败时自动回滚 DeleteObject。 */
  instanceNumber?: number;
  /** add_rollback 动作:从被回滚的原 SPV 失败任务中提取的简短被拒原因,用于 Tag 文案。 */
  originFaultBrief?: string;
}

export type Feedback = CellFeedback | MultiFeedback;

export interface QuickSettingsSyncMonitor {
  sourceId?: string;
  lastParamSyncAt?: string;
  lastParamSyncFailedAt?: string;
  targetCount: number;
  gpvTaskCount: number;
  startedAt: number;
  congestionHinted?: boolean;
}

export interface QuickSettingsScopedSyncResult {
  targetCount: number;
  gpvTaskCount: number;
  completedAt?: string;
  wallClockSeconds?: number;
}

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
  quickSettingsSyncs: Record<string, QuickSettingsSyncMonitor>;
  lastScopedSyncs: Record<string, QuickSettingsScopedSyncResult>;

  setFeedback: (key: string, feedback: Feedback) => void;
  patchFeedback: (key: string, patch: Partial<Feedback>) => void;
  clearFeedback: (key: string) => void;
  clearByDevice: (deviceId: string) => void;
  clearDraftsByDevice: (deviceId: string) => void;

  setDraftField: (key: string, name: string, value: QuickSettingsDraftValue) => void;
  clearDraft: (key: string) => void;
  /**
   * 删除 drafts[key] 下所有以 prefix 开头的字段，用于 MultiInstanceTable 单行 save/delete
   * 后只清理该行的草稿、保留其他行未保存编辑。
   */
  clearDraftPrefix: (key: string, prefix: string) => void;

  bumpRefreshTick: (deviceId: string) => void;
  startQuickSettingsSync: (deviceId: string, sync: QuickSettingsSyncMonitor) => void;
  patchQuickSettingsSync: (deviceId: string, patch: Partial<QuickSettingsSyncMonitor>) => void;
  finishQuickSettingsSync: (deviceId: string, result?: QuickSettingsScopedSyncResult) => void;
  clearLastScopedSync: (deviceId: string) => void;
}

export const useQuickSettingsFeedbackStore = create<FeedbackState>()(
  persist(
    (set, get) => ({
      entries: {},
      drafts: {},
      refreshTicks: {},
      quickSettingsSyncs: {},
      lastScopedSyncs: {},

      setFeedback: (key, feedback) => {
        set({ entries: { ...get().entries, [key]: feedback } });
      },

      patchFeedback: (key, patch) => {
        const cur = get().entries[key];
        if (!cur) return;
        set({ entries: { ...get().entries, [key]: { ...cur, ...patch } as Feedback } });
      },

      clearFeedback: (key) => {
        const next = { ...get().entries };
        delete next[key];
        set({ entries: next });
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

      clearDraftsByDevice: (deviceId) => {
        const nextDrafts: Record<string, Record<string, QuickSettingsDraftValue>> = {};
        const prefix = `${deviceId}::`;
        for (const [k, v] of Object.entries(get().drafts)) {
          if (!k.startsWith(prefix)) nextDrafts[k] = v;
        }
        set({ drafts: nextDrafts });
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

      startQuickSettingsSync: (deviceId, sync) => {
        set({
          quickSettingsSyncs: { ...get().quickSettingsSyncs, [deviceId]: sync },
        });
      },

      patchQuickSettingsSync: (deviceId, patch) => {
        const cur = get().quickSettingsSyncs[deviceId];
        if (!cur) return;
        set({ quickSettingsSyncs: { ...get().quickSettingsSyncs, [deviceId]: { ...cur, ...patch } } });
      },

      finishQuickSettingsSync: (deviceId, result) => {
        const nextSyncs = { ...get().quickSettingsSyncs };
        delete nextSyncs[deviceId];
        const nextLast = { ...get().lastScopedSyncs };
        if (result?.targetCount) {
          nextLast[deviceId] = result;
        }
        set({ quickSettingsSyncs: nextSyncs, lastScopedSyncs: nextLast });
      },

      clearLastScopedSync: (deviceId) => {
        const next = { ...get().lastScopedSyncs };
        delete next[deviceId];
        set({ lastScopedSyncs: next });
      },
    }),
    {
      name: 'omc-quicksettings-feedback',
      storage: createJSONStorage(() => sessionStorage),
      // quickSettingsSyncs 是"正在运行的任务监控"，刷新页面后这些 monitor 已失效；
      // 让 QuickSettingsSyncWatcher 重水合时拿到 startedAt 远早于实际终态时间戳，
      // 会按时钟容差误判一次 success → 弹无关 toast。partialize 显式排除该字段，
      // 让它只在内存中存在；其它字段（entries/drafts/refreshTicks/lastScopedSyncs）继续持久化。
      partialize: (state) => ({
        entries: state.entries,
        drafts: state.drafts,
        refreshTicks: state.refreshTicks,
        lastScopedSyncs: state.lastScopedSyncs,
      }),
    },
  ),
);
