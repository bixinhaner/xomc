import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { TabItem } from '../types/common';

export type { TabItem };

const MAX_TABS = 10;

// 仪表板 tab 的稳定标识。与动态菜单 buildDynamicKeyToLeaf 产出的 key
// （= 菜单 routePath）保持一致，否则点击菜单"仪表板"会因 key 不匹配
// 创建出第二个同名 tab（NavMenu.tsx 走 leaf.routePath='/dashboard'）。
const DASHBOARD_TAB_KEY = '/dashboard';

interface TabState {
  tabs: TabItem[];
  activeTabKey: string;

  openTab: (tab: TabItem) => void;
  closeTab: (key: string) => void;
  closeOtherTabs: (key: string) => void;
  closeAllTabs: () => void;
  closeTabsToRight: (key: string) => void;
  setActiveTab: (key: string) => void;
  moveTab: (fromIndex: number, toIndex: number) => void;
  // Legacy alias
  setActiveKey: (key: string) => void;
  reorderTabs: (fromIndex: number, toIndex: number) => void;
}

const DASHBOARD_TAB: TabItem = {
  key: DASHBOARD_TAB_KEY,
  label: 'nav.dashboard',
  path: '/dashboard',
  closable: false,
};

export const useTabStore = create<TabState>()(
  persist(
    (set, get) => ({
      tabs: [DASHBOARD_TAB],
      activeTabKey: DASHBOARD_TAB_KEY,

      openTab: (tab) => {
        const { tabs } = get();
        // 兼容旧 sessionStorage：历史版本用 'dashboard' 作为 key，
        // 新版用 '/dashboard'。同 path 视为同 tab，merge 时把 key 收敛到新值。
        const existsIdx = tabs.findIndex(
          (t) => t.key === tab.key || (tab.path === '/dashboard' && t.path === '/dashboard'),
        );
        if (existsIdx !== -1) {
          // 命中同 key（或同 path 的 /dashboard）时，同步 path/label/labelRaw/closable，
          // 让"复用 tab 显示不同记录详情"场景的 URL 与标题正确刷新。
          // 注意：key 始终保留原 tab 的 key —— 防止静态/动态菜单两种 key 形态
          // （'dashboard' vs '/dashboard'）互相覆盖导致 activeTabKey 漂移。
          const merged = tabs.slice();
          const existing = merged[existsIdx];
          merged[existsIdx] = {
            ...existing,
            ...tab,
            key: existing.key,
            // 仪表板永不可关闭：忽略调用方传入的 closable
            closable: tab.path === '/dashboard' ? false : (tab.closable ?? existing.closable),
          };
          set({ tabs: merged, activeTabKey: existing.key });
          return;
        }
        // Enforce max tabs — remove the oldest non-dashboard, non-active tab if at limit
        const newTabs = [...tabs, { ...tab, closable: tab.closable ?? true }];
        if (newTabs.length > MAX_TABS) {
          const removeIdx = newTabs.findIndex(
            (t) => t.key !== DASHBOARD_TAB_KEY && t.key !== get().activeTabKey
          );
          if (removeIdx !== -1) {
            newTabs.splice(removeIdx, 1);
          }
        }
        set({ tabs: newTabs, activeTabKey: tab.key });
      },

      closeTab: (key) => {
        const { tabs, activeTabKey } = get();
        if (key === DASHBOARD_TAB_KEY) return;
        const index = tabs.findIndex((t) => t.key === key);
        const newTabs = tabs.filter((t) => t.key !== key);
        let newActiveKey = activeTabKey;
        if (activeTabKey === key) {
          const prev = newTabs[index - 1];
          const next = newTabs[index];
          newActiveKey = (next ?? prev)?.key ?? DASHBOARD_TAB_KEY;
        }
        set({ tabs: newTabs, activeTabKey: newActiveKey });
      },

      closeOtherTabs: (key) => {
        const { tabs } = get();
        const newTabs = tabs.filter((t) => !t.closable || t.key === key);
        set({ tabs: newTabs, activeTabKey: key });
      },

      closeAllTabs: () => {
        set({ tabs: [DASHBOARD_TAB], activeTabKey: DASHBOARD_TAB_KEY });
      },

      closeTabsToRight: (key) => {
        const { tabs, activeTabKey } = get();
        const index = tabs.findIndex((t) => t.key === key);
        const newTabs = tabs.slice(0, index + 1);
        const stillActive = newTabs.find((t) => t.key === activeTabKey);
        set({ tabs: newTabs, activeTabKey: stillActive ? activeTabKey : key });
      },

      setActiveTab: (key) => set({ activeTabKey: key }),
      setActiveKey: (key) => set({ activeTabKey: key }),

      moveTab: (fromIndex, toIndex) => {
        const { tabs } = get();
        if (fromIndex === 0 || toIndex === 0) return;
        const newTabs = [...tabs];
        const [moved] = newTabs.splice(fromIndex, 1);
        newTabs.splice(toIndex, 0, moved);
        set({ tabs: newTabs });
      },

      reorderTabs: (fromIndex, toIndex) => {
        get().moveTab(fromIndex, toIndex);
      },
    }),
    {
      name: 'omc-tab-store',
      storage: createJSONStorage(() => sessionStorage),
      version: 2,
      migrate: (persisted: unknown, version: number) => {
        // v1 → v2：DASHBOARD_TAB_KEY 从 'dashboard' 改 '/dashboard'。
        // 把残留的旧 dashboard tab key 改写，activeTabKey 同步迁移。
        if (version < 2 && persisted && typeof persisted === 'object') {
          const p = persisted as { tabs?: TabItem[]; activeTabKey?: string };
          const tabs = (p.tabs ?? []).map((t) =>
            t.key === 'dashboard' && t.path === '/dashboard'
              ? { ...t, key: DASHBOARD_TAB_KEY, closable: false }
              : t,
          );
          // 兜底：迁移完若没有 dashboard tab（理论不会发生），补回。
          const hasDashboard = tabs.some((t) => t.key === DASHBOARD_TAB_KEY);
          return {
            ...p,
            tabs: hasDashboard ? tabs : [DASHBOARD_TAB, ...tabs],
            activeTabKey:
              p.activeTabKey === 'dashboard' ? DASHBOARD_TAB_KEY : (p.activeTabKey ?? DASHBOARD_TAB_KEY),
          };
        }
        return persisted as TabState;
      },
    }
  )
);
