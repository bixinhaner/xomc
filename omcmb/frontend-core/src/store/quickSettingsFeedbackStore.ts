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
  /** 本次 SPV 实际下发的 path/value；需要设备回读核实时用于判断何时可以清理草稿。 */
  expectedReadback?: Record<string, string>;
  /** 提交瞬间的草稿修订号；回读时若已变化，说明用户又进行了编辑。 */
  submittedDraftRevision?: number;
  /** 终态任务的回读结果已安全回填，避免组件重挂载后重复消费。 */
  syncedForTaskId?: string;
}

export interface MultiPendingAddRow {
  tempId: string;
  values: Record<string, string>;
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
  /** SPV 实际提交的 path/value；成功任务必须观察到这些值后才可清除行编辑和草稿。 */
  expectedReadback?: Record<string, string>;
  /** 提交时的草稿修订号；回读期间若用户继续编辑，则保留新草稿。 */
  submittedDraftRevision?: number;
  /** save 动作专用:记录被保存的实例号,task=completed 时据此清掉对应行的 rowEdits + draft,
      避免切 tab 走回后乐观显示的旧用户输入覆盖 schema 新值。 */
  savedInstId?: string;
  /** 批量 save 动作:一次保存多个新增实例时,在任务完成后一起清掉乐观编辑态。 */
  savedInstIds?: string[];
  /** 批量提交动作拆分:新增成功后应出现在回读列表中的实例号。 */
  addedInstIds?: string[];
  /** 批量提交动作拆分:删除成功后应从回读列表消失的实例号。 */
  deletedInstIds?: string[];
  /** Add/DeleteObject 终态后需要核对实例集合的对象路径。 */
  readbackObjectPath?: string;
  /** AddObject 提交前的实例集合，用于等待新实例真正出现。 */
  readbackInstancesBefore?: string[];
  /** 批量提交动作拆分:编辑成功后需要清掉本地编辑态的实例号。 */
  editedInstIds?: string[];
  /** 批量新增已提交但还未完成基站回读时,继续在表格中展示的本地行快照。 */
  pendingAddRows?: MultiPendingAddRow[];
  /** IPSec 统一提交尚未确认完成的删除实例，供重挂载后恢复待处理状态。 */
  pendingDeleteInstIds?: string[];
  /** save 动作来源:区分新增后的 SPV 与编辑已有实例的 SPV,避免把编辑失败误当新增失败回滚。 */
  saveMode?: 'add' | 'edit';
  /** BSC add 动作:AddObject 创建出来的实例号,用于 SPV 失败时自动回滚 DeleteObject。 */
  instanceNumber?: number;
  /** add_rollback 动作:从被回滚的原 SPV 失败任务中提取的简短被拒原因,用于 Tag 文案。 */
  originFaultBrief?: string;
  /** IPSec 统一提交可能拆成多个设备任务，按实际创建顺序持久化全部 task id。 */
  ipsecTaskIds?: string[];
  /** IPSec 统一提交成功后期望回读到的总开关状态。 */
  ipsecTargetEnabled?: boolean;
  /** IPSec 已完成的有序阶段，用于切换页签后恢复进度说明。 */
  ipsecCompletedPhases?: Array<'enable-global' | 'apply-tunnels' | 'disable-global'>;
  /** IPSec 统一提交失败时终止所在阶段。 */
  ipsecFailedPhase?: 'enable-global' | 'apply-tunnels' | 'disable-global';
  /** IPSec 统一操作状态；running/awaiting-readback 会在组件重挂载后继续阻止重复提交。 */
  ipsecOperationStatus?: 'running' | 'awaiting-readback' | 'completed' | 'failed';
  /** IPSec 当前正在执行的阶段。 */
  ipsecActivePhase?: 'enable-global' | 'apply-tunnels' | 'disable-global';
}

export type Feedback = CellFeedback | MultiFeedback;

export interface QuickSettingsSyncMonitor {
  scope?: 'quickSettings' | 'license';
  sourceId?: string;
  requestId?: string;
  runId?: string;
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
  /** 每次用户草稿写入都递增；独立于 drafts 保留，以识别提交后的新编辑。 */
  draftRevisions: Record<string, number>;
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

function withoutTransientIpsecOperations(
  entries: Record<string, Feedback>,
): Record<string, Feedback> {
  return Object.fromEntries(
    Object.entries(entries).filter(([, feedback]) => !(
      feedback.kind === 'multi'
      && (
        feedback.ipsecOperationStatus === 'running'
        || feedback.ipsecOperationStatus === 'awaiting-readback'
      )
    )),
  );
}

export const useQuickSettingsFeedbackStore = create<FeedbackState>()(
  persist(
    (set, get) => ({
      entries: {},
      drafts: {},
      draftRevisions: {},
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
        const nextDraftRevisions: Record<string, number> = {};
        const prefix = `${deviceId}::`;
        for (const [k, v] of Object.entries(get().entries)) {
          if (!k.startsWith(prefix)) next[k] = v;
        }
        for (const [k, v] of Object.entries(get().drafts)) {
          if (!k.startsWith(prefix)) nextDrafts[k] = v;
        }
        for (const [k, v] of Object.entries(get().draftRevisions)) {
          if (!k.startsWith(prefix)) nextDraftRevisions[k] = v;
        }
        set({ entries: next, drafts: nextDrafts, draftRevisions: nextDraftRevisions });
      },

      clearDraftsByDevice: (deviceId) => {
        const nextDrafts: Record<string, Record<string, QuickSettingsDraftValue>> = {};
        const nextDraftRevisions = { ...get().draftRevisions };
        const prefix = `${deviceId}::`;
        for (const [k, v] of Object.entries(get().drafts)) {
          if (k.startsWith(prefix)) {
            nextDraftRevisions[k] = (nextDraftRevisions[k] ?? 0) + 1;
          } else {
            nextDrafts[k] = v;
          }
        }
        set({ drafts: nextDrafts, draftRevisions: nextDraftRevisions });
      },

      setDraftField: (key, name, value) => {
        const cur = get().drafts[key] ?? {};
        set({
          drafts: { ...get().drafts, [key]: { ...cur, [name]: value } },
          draftRevisions: {
            ...get().draftRevisions,
            [key]: (get().draftRevisions[key] ?? 0) + 1,
          },
        });
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
      // 让它只在内存中存在；其它字段（entries/drafts/draftRevisions/refreshTicks/lastScopedSyncs）继续持久化。
      partialize: (state) => ({
        // 运行中的 IPSec 编排由当前页面闭包继续驱动，不能跨整页刷新恢复。
        // 不把临时锁写入 sessionStorage；页内切 Tab 仍共享 Zustand 内存状态，
        // 整页刷新则保留草稿但释放无法恢复的运行锁，避免永久禁用提交/清空。
        entries: withoutTransientIpsecOperations(state.entries),
        drafts: state.drafts,
        draftRevisions: state.draftRevisions,
        refreshTicks: state.refreshTicks,
        lastScopedSyncs: state.lastScopedSyncs,
      }),
      merge: (persisted, current) => {
        const saved = persisted as Partial<FeedbackState>;
        return {
          ...current,
          ...saved,
          entries: withoutTransientIpsecOperations(saved.entries ?? {}),
        };
      },
    },
  ),
);
