/**
 * T-0164-P6 / G6 PM 性能查看仪表盘编辑器状态。
 *
 * 关注点：当前编辑中的 dashboard + 未保存改动追踪 + layout 局部 mutation。
 * 持久化 API 状态走 React Query；此 store 只承载 UI 临时编辑态。
 */

import { create } from 'zustand';
import type { Dashboard, Panel, PanelGridItem } from '../types/pmDashboard';

// G6-Gap-2：共享筛选条 — 编辑器顶部全局筛选，panel.config.inherit_global=true 时消费。
export interface GlobalFilter {
  // 相对时间偏移（如 -1h / -24h / -7d）；优先级低于 absolute
  startOffset?: string;
  // 绝对时间窗（ISO RFC3339）；置后优先于 offset
  absoluteStart?: string;
  absoluteEnd?: string;
  // 选中的设备组 ID 列表（多选）
  deviceGroupIds: string[];
  // 选中的设备 SN 列表（多选）
  deviceSns: string[];
}

const DEFAULT_FILTER: GlobalFilter = {
  startOffset: '-24h',
  deviceGroupIds: [],
  deviceSns: [],
};

interface PmDashboardState {
  // 当前打开的 dashboard + panels（fetched 后由 hook 初始化）
  currentDashboard: Dashboard | null;
  currentPanels: Panel[];

  // 编辑模式开关 + dirty 跟踪
  editMode: boolean;
  unsavedChanges: boolean;

  // G6-Gap-2 共享筛选条（dashboard 级，不持久化到 pm_panels；保存到 user_preferences.shared_filters）
  globalFilter: GlobalFilter;
  setGlobalFilter: (f: Partial<GlobalFilter>) => void;

  // ── Actions ──────────────────────────────────────────────────────

  /** hook fetch 完后初始化。重置 dirty + 退出 edit 模式。 */
  loadDashboard: (dashboard: Dashboard, panels: Panel[]) => void;
  /** 切换编辑模式（toolbar 上的 toggle）。 */
  toggleEditMode: () => void;
  /** 显式退出编辑（保存后或取消后调）。 */
  exitEditMode: () => void;
  /** 标 dirty 但不写远端（用户拖拽 / 改 panel 配置时）。 */
  markDirty: () => void;
  /** 完成保存后清 dirty 标记。 */
  markClean: () => void;
  /** 更新单 panel 的 layout 位置（拖拽 react-grid-layout 触发）。 */
  updatePanelLayout: (panelId: string, layout: Partial<PanelGridItem>) => void;
  /** 加新 panel 到 layout（后端 createPanel 后调）。 */
  addPanel: (panel: Panel, layout?: PanelGridItem) => void;
  /** 删 panel + 同步 layout。 */
  removePanel: (panelId: string) => void;
  /** 替换整体 layout（拖拽 batch 后调）。 */
  replaceLayout: (panels: PanelGridItem[]) => void;
  /** 重置 store（路由离开时 / 切 dashboard 前调）。 */
  reset: () => void;
}

const initialState = {
  currentDashboard: null as Dashboard | null,
  currentPanels: [] as Panel[],
  editMode: false,
  unsavedChanges: false,
  globalFilter: DEFAULT_FILTER,
};

export const usePmDashboardStore = create<PmDashboardState>((set, get) => ({
  ...initialState,

  setGlobalFilter: (f) => set((s) => ({ globalFilter: { ...s.globalFilter, ...f } })),

  loadDashboard: (dashboard, panels) =>
    set({
      currentDashboard: dashboard,
      currentPanels: panels,
      editMode: false,
      unsavedChanges: false,
      // 切 dashboard 重置筛选（每个 dashboard 独立筛选语义；持久化到 user_preferences.shared_filters 由调用方负责）
      globalFilter: DEFAULT_FILTER,
    }),

  toggleEditMode: () => set((s) => ({ editMode: !s.editMode })),

  exitEditMode: () => set({ editMode: false, unsavedChanges: false }),

  markDirty: () => set({ unsavedChanges: true }),

  markClean: () => set({ unsavedChanges: false }),

  updatePanelLayout: (panelId, layout) => {
    const { currentDashboard } = get();
    if (!currentDashboard) return;
    const panels = currentDashboard.layout.panels.map((p) =>
      p.i === panelId ? { ...p, ...layout } : p,
    );
    set({
      currentDashboard: {
        ...currentDashboard,
        layout: { panels },
      },
      unsavedChanges: true,
    });
  },

  addPanel: (panel, layout) => {
    const { currentDashboard, currentPanels } = get();
    if (!currentDashboard) return;
    const item: PanelGridItem = layout ?? {
      i: panel.id,
      x: 0,
      y: Number.POSITIVE_INFINITY, // react-grid-layout 自动放到底部
      w: 6,
      h: 4,
    };
    set({
      currentPanels: [...currentPanels, panel],
      currentDashboard: {
        ...currentDashboard,
        layout: { panels: [...currentDashboard.layout.panels, item] },
      },
      unsavedChanges: true,
    });
  },

  removePanel: (panelId) => {
    const { currentDashboard, currentPanels } = get();
    if (!currentDashboard) return;
    set({
      currentPanels: currentPanels.filter((p) => p.id !== panelId),
      currentDashboard: {
        ...currentDashboard,
        layout: { panels: currentDashboard.layout.panels.filter((p) => p.i !== panelId) },
      },
      unsavedChanges: true,
    });
  },

  replaceLayout: (panels) => {
    const { currentDashboard } = get();
    if (!currentDashboard) return;
    set({
      currentDashboard: { ...currentDashboard, layout: { panels } },
      unsavedChanges: true,
    });
  },

  reset: () => set(initialState),
}));
