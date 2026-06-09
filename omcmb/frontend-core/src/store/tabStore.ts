import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { TabItem } from '../types/common';

export type { TabItem };

const MAX_TABS = 10;

function dedupeTabsByExactPath(tabs: TabItem[]) {
  const seenPathToKey = new Map<string, string>();
  const deduped: TabItem[] = [];
  const removedKeyToKeptKey = new Map<string, string>();

  for (const tab of tabs) {
    const existingKey = seenPathToKey.get(tab.path);
    if (!existingKey) {
      seenPathToKey.set(tab.path, tab.key);
      deduped.push(tab);
      continue;
    }
    removedKeyToKeptKey.set(tab.key, existingKey);
  }

  return { deduped, removedKeyToKeptKey };
}

// 仪表板 tab 的稳定标识。使用 'dashboard'（与 navConfig 中其他菜单项的 key 命名风格一致）。
// 动态菜单使用 routePath（'/dashboard'）作为 key，但通过 path 匹配兼容静态菜单的 'dashboard' key。
const DASHBOARD_TAB_KEY = 'dashboard';

interface TabState {
  tabs: TabItem[];
  activeTabKey: string;

  openTab: (tab: TabItem) => void;
  closeTab: (key: string) => void;
  closeOtherTabs: (key: string) => void;
  closeAllTabs: () => void;
  closeTabsToRight: (key: string) => void;
  setActiveTab: (key: string) => void;
  /**
   * 把当前完整 URL（pathname + search）同步进「激活 tab」的 path，使页面内二级（drill-down，
   * 走 URL search 参数）状态被记进标签页；切走再切回时 navigate(tab.path) 即可恢复二级状态。
   * 带同基础路由守卫：仅当 pathname 一致时才更新（避免把别的路由 URL 误写进当前 tab）。
   */
  syncActiveTabPath: (fullPath: string) => void;
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
        // 兼容动态菜单：动态菜单使用 routePath（'/dashboard'）作为 key，
        // 静态菜单使用 'dashboard' 作为 key。通过 path 匹配确保两者指向同一个 tab。
        const existsIdx = tabs.findIndex(
          (t) =>
            t.key === tab.key ||
            t.path === tab.path ||
            (tab.path === '/dashboard' && t.path === '/dashboard'),
        );
        if (existsIdx !== -1) {
          // 命中同 key（或同 path 的 /dashboard）时，同步 path/label/labelRaw/closable，
          // 让静态/动态菜单别名和"复用 tab 显示不同记录详情"场景的 URL 与标题正确刷新。
          // 注意：key 始终保留原 tab 的 key —— 防止静态/动态菜单两种 key 形态
          // （如 'device-list' vs '/device/list'）互相覆盖导致 activeTabKey 漂移。
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

      syncActiveTabPath: (fullPath) => {
        const { tabs, activeTabKey } = get();
        const idx = tabs.findIndex((t) => t.key === activeTabKey);
        if (idx === -1) return;
        const cur = tabs[idx];
        // 仅在同一基础路由（pathname 相同）内同步 search/二级状态，避免把别的路由 URL 写进当前 tab。
        if (cur.path.split('?')[0] !== fullPath.split('?')[0]) return;
        if (cur.path === fullPath) return;
        const next = tabs.slice();
        next[idx] = { ...cur, path: fullPath };
        set({ tabs: next });
      },

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
      version: 4,
      migrate: (persisted: unknown, version: number) => {
        if (!persisted || typeof persisted !== 'object') return persisted as TabState;
        const p = persisted as { tabs?: TabItem[]; activeTabKey?: string };
        // 确保 tabs 始终有值，避免后续代码中的类型错误
        p.tabs = p.tabs ?? [];

        // v3 → v4：按完全相同 path 去重，收敛历史上静态/动态菜单别名造成的重复 tab。
        if (version < 4) {
          const { deduped, removedKeyToKeptKey } = dedupeTabsByExactPath(p.tabs);
          p.tabs = deduped;
          if (p.activeTabKey) {
            p.activeTabKey = removedKeyToKeptKey.get(p.activeTabKey) ?? p.activeTabKey;
          }
        }

        // v2 → v3：DASHBOARD_TAB_KEY 从 '/dashboard' 改回 'dashboard'（与 navConfig 命名风格一致）。
        if (version < 3) {
          const migratedTabs = p.tabs.map((t) =>
            t.key === '/dashboard' && t.path === '/dashboard'
              ? { ...t, key: DASHBOARD_TAB_KEY, closable: false }
              : t,
          );
          const migratedActiveKey = p.activeTabKey === '/dashboard' ? DASHBOARD_TAB_KEY : p.activeTabKey;
          p.tabs = migratedTabs;
          p.activeTabKey = migratedActiveKey ?? DASHBOARD_TAB_KEY;
        }

        // v1 → v2：DASHBOARD_TAB_KEY 从 'dashboard' 改 '/dashboard'（已废弃，保留以防旧数据）。
        if (version < 2) {
          const v2Tabs = p.tabs.map((t) =>
            t.key === 'dashboard' && t.path === '/dashboard'
              ? { ...t, key: DASHBOARD_TAB_KEY, closable: false }
              : t,
          );
          p.tabs = v2Tabs;
          if (p.activeTabKey === 'dashboard') p.activeTabKey = DASHBOARD_TAB_KEY;
        }

        // 兜底：确保始终有 dashboard tab
        const hasDashboard = p.tabs.some((t) => t.key === DASHBOARD_TAB_KEY);
        if (!hasDashboard) {
          p.tabs = [DASHBOARD_TAB, ...p.tabs];
          p.activeTabKey = p.activeTabKey ?? DASHBOARD_TAB_KEY;
        }

        return p as TabState;
      },
    }
  )
);
