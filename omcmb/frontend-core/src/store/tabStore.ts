import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { TabItem } from '../types/common';

export type { TabItem };

const MAX_TABS = 10;

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
  key: 'dashboard',
  label: 'nav.dashboard',
  path: '/dashboard',
  closable: false,
};

export const useTabStore = create<TabState>()(
  persist(
    (set, get) => ({
      tabs: [DASHBOARD_TAB],
      activeTabKey: 'dashboard',

      openTab: (tab) => {
        const { tabs } = get();
        const existsIdx = tabs.findIndex((t) => t.key === tab.key);
        if (existsIdx !== -1) {
          // 命中同 key 时，同步更新 path/label/labelRaw/closable，
          // 让"复用 tab 显示不同记录详情"场景的 URL 与标题正确刷新。
          const merged = tabs.slice();
          merged[existsIdx] = { ...merged[existsIdx], ...tab, closable: tab.closable ?? merged[existsIdx].closable };
          set({ tabs: merged, activeTabKey: tab.key });
          return;
        }
        // Enforce max tabs — remove the oldest non-dashboard, non-active tab if at limit
        const newTabs = [...tabs, { ...tab, closable: tab.closable ?? true }];
        if (newTabs.length > MAX_TABS) {
          const removeIdx = newTabs.findIndex(
            (t) => t.key !== 'dashboard' && t.key !== get().activeTabKey
          );
          if (removeIdx !== -1) {
            newTabs.splice(removeIdx, 1);
          }
        }
        set({ tabs: newTabs, activeTabKey: tab.key });
      },

      closeTab: (key) => {
        const { tabs, activeTabKey } = get();
        if (key === 'dashboard') return;
        const index = tabs.findIndex((t) => t.key === key);
        const newTabs = tabs.filter((t) => t.key !== key);
        let newActiveKey = activeTabKey;
        if (activeTabKey === key) {
          const prev = newTabs[index - 1];
          const next = newTabs[index];
          newActiveKey = (next ?? prev)?.key ?? 'dashboard';
        }
        set({ tabs: newTabs, activeTabKey: newActiveKey });
      },

      closeOtherTabs: (key) => {
        const { tabs } = get();
        const newTabs = tabs.filter((t) => !t.closable || t.key === key);
        set({ tabs: newTabs, activeTabKey: key });
      },

      closeAllTabs: () => {
        set({ tabs: [DASHBOARD_TAB], activeTabKey: 'dashboard' });
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
    }
  )
);
