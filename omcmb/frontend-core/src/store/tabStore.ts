import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { TabItem } from '../types/common';
import { performancePageKeyFromPath, usePmPageStateStore } from './pmPageStateStore';

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

function clearClosedPerformanceTabStates(closedTabs: TabItem[]) {
  const pageKeys = closedTabs
    .map((tab) => performancePageKeyFromPath(tab.path))
    .filter((pageKey): pageKey is string => Boolean(pageKey));
  if (pageKeys.length === 0) return;
  usePmPageStateStore.getState().clearPageStates(pageKeys);
}

function basePathOf(path: string): string {
  return path.split('?')[0].replace(/\/+$/, '') || path;
}

function tabRouteGroup(path: string): string {
  const basePath = basePathOf(path);
  if (basePath === '/performance/pm-adhoc/new' || /^\/performance\/pm-adhoc\/[^/]+\/edit$/.test(basePath)) {
    return '/performance/pm-adhoc';
  }
  return basePath;
}

// 仪表板 tab 的稳定标识。使用 'dashboard'（与 navConfig 中其他菜单项的 key 命名风格一致）。
// 动态菜单使用 routePath（'/dashboard'）作为 key，但通过 path 匹配兼容静态菜单的 'dashboard' key。
const DASHBOARD_TAB_KEY = 'dashboard';

interface TabState {
  tabs: TabItem[];
  activeTabKey: string;

  openTab: (tab: TabItem) => void;
  /** 新窗口/地址栏首次直达时，用当前业务页替换尚未实际访问的默认仪表板占位。 */
  openTabForDirectEntry: (tab: TabItem) => void;
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
  /**
   * 路由变化时根据 pathname 激活对应的 tab（若存在）。
   * 修复 BUG-11：通过侧边栏导航切换页面时 tab 高亮不跟随 URL 的问题。
   * @returns 是否成功激活了一个已存在的 tab
   */
  activateByPath: (pathname: string) => boolean;
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
        // 注意：syncActiveTabPath 会把 search params 写进 tab.path（如 /alarm/current?severity=2），
        // 因此这里用 basename（去掉 ?...）做匹配，避免同一路由因 search params 不同而重复建 tab。
        const tabBasePath = tabRouteGroup(tab.path);
        const existsIdx = tabs.findIndex(
          (t) =>
            t.key === tab.key ||
            tabRouteGroup(t.path) === tabBasePath,
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
        const normalizedTab = {
          ...tab,
          closable: tab.path === '/dashboard' ? false : (tab.closable ?? true),
        };
        // 直达业务页的会话可能暂时没有仪表板；用户随后进入首页时仍把固定页签放回首位。
        const newTabs = tab.path === '/dashboard'
          ? [normalizedTab, ...tabs]
          : [...tabs, normalizedTab];
        if (newTabs.length > MAX_TABS) {
          const removeIdx = newTabs.findIndex(
            (t) => t.key !== DASHBOARD_TAB_KEY && t.key !== get().activeTabKey
          );
          if (removeIdx !== -1) {
            const [closedTab] = newTabs.splice(removeIdx, 1);
            if (closedTab) {
              clearClosedPerformanceTabStates([closedTab]);
            }
          }
        }
        set({ tabs: newTabs, activeTabKey: tab.key });
      },

      openTabForDirectEntry: (tab) => {
        const { tabs, activeTabKey } = get();
        const hasOnlyDefaultDashboard =
          tabs.length === 1 &&
          tabs[0]?.key === DASHBOARD_TAB_KEY &&
          activeTabKey === DASHBOARD_TAB_KEY;
        if (!hasOnlyDefaultDashboard || tab.path === '/dashboard') {
          get().openTab(tab);
          return;
        }
        set({
          tabs: [{ ...tab, closable: tab.closable ?? true }],
          activeTabKey: tab.key,
        });
      },

      closeTab: (key) => {
        const { tabs, activeTabKey } = get();
        if (key === DASHBOARD_TAB_KEY) return;
        const index = tabs.findIndex((t) => t.key === key);
        const closedTab = tabs[index];
        const remainingTabs = tabs.filter((t) => t.key !== key);
        const newTabs = remainingTabs.length > 0 ? remainingTabs : [DASHBOARD_TAB];
        let newActiveKey = activeTabKey;
        if (activeTabKey === key) {
          const prev = newTabs[index - 1];
          const next = newTabs[index];
          newActiveKey = (next ?? prev)?.key ?? DASHBOARD_TAB_KEY;
        }
        if (closedTab) {
          clearClosedPerformanceTabStates([closedTab]);
        }
        set({ tabs: newTabs, activeTabKey: newActiveKey });
      },

      closeOtherTabs: (key) => {
        const { tabs } = get();
        const newTabs = tabs.filter((t) => !t.closable || t.key === key);
        const closedTabs = tabs.filter((tab) => !newTabs.includes(tab));
        clearClosedPerformanceTabStates(closedTabs);
        set({ tabs: newTabs, activeTabKey: key });
      },

      closeAllTabs: () => {
        const { tabs } = get();
        clearClosedPerformanceTabStates(tabs.filter((tab) => tab.closable));
        set({ tabs: [DASHBOARD_TAB], activeTabKey: DASHBOARD_TAB_KEY });
      },

      closeTabsToRight: (key) => {
        const { tabs, activeTabKey } = get();
        const index = tabs.findIndex((t) => t.key === key);
        if (index === -1) return;
        const newTabs = tabs.slice(0, index + 1);
        const closedTabs = tabs.slice(index + 1);
        const stillActive = newTabs.find((t) => t.key === activeTabKey);
        clearClosedPerformanceTabStates(closedTabs);
        set({ tabs: newTabs, activeTabKey: stillActive ? activeTabKey : key });
      },

      setActiveTab: (key) => set({ activeTabKey: key }),
      setActiveKey: (key) => set({ activeTabKey: key }),

      syncActiveTabPath: (fullPath) => {
        const { tabs, activeTabKey } = get();
        const idx = tabs.findIndex((t) => t.key === activeTabKey);
        if (idx === -1) return;
        const cur = tabs[idx];
        // 仅在同一 tab 路由组内同步 search/二级状态，避免把别的路由 URL 误写进当前 tab。
        if (tabRouteGroup(cur.path) !== tabRouteGroup(fullPath)) return;
        if (cur.path === fullPath) return;
        const next = tabs.slice();
        next[idx] = { ...cur, path: fullPath };
        set({ tabs: next });
      },

      activateByPath: (pathname) => {
        const { tabs, activeTabKey } = get();
        // 查找与传入 pathname 同一 tab 路由组的 tab。
        const matchedTab = tabs.find((t) => tabRouteGroup(t.path) === tabRouteGroup(pathname));
        if (!matchedTab) return false;
        // 如果已经是激活状态，不需要更新
        if (matchedTab.key === activeTabKey) return true;
        set({ activeTabKey: matchedTab.key });
        return true;
      },

      moveTab: (fromIndex, toIndex) => {
        const { tabs } = get();
        if (
          tabs[fromIndex]?.key === DASHBOARD_TAB_KEY ||
          tabs[toIndex]?.key === DASHBOARD_TAB_KEY
        ) return;
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
