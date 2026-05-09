// useMenus — 菜单动态加载 P1 React Query Hooks。
//
// 设计依据：docs/prd/system/menu-dynamic-loading.md §4.3.2。
//
// stale-while-revalidate：
//   - persist 已恢复 → menuStore 立即可用，UI 无白屏
//   - 后台 useUserMenus 静默更新；新数据回填时触发组件 re-render
//   - 切角色/退出后调用 invalidateUserMenus 强制重新拉取

import { useQuery, useQueryClient } from '@tanstack/react-query';

import type { Menu } from '../../types/menu';
import { fetchMenuTree, fetchUserMenus } from '../../services/api/menuApi';
import { useMenuStore } from '../../store/menuStore';
import { useUserStore } from '../../store/userStore';

/** Query key 常量，便于 invalidate 与单点维护。 */
export const userMenusQueryKey = ['userMenus'] as const;
export const menuTreeQueryKey = ['menus', 'tree'] as const;

/**
 * 当前用户当前角色的菜单树。拉取成功自动写入 menuStore。
 *
 * queryKey 必须带 userId：用户切换（admin → test）时 key 变化 → React Query 不会
 * 复用上一个用户的菜单缓存。修复"切换用户后侧边栏短暂显示前一个用户菜单，刷新才
 * 恢复"的 bug —— 旧实现固定 queryKey=['userMenus'] + staleTime=5min，logout 只
 * clearAuth/menuStore 不动 React Query 缓存，下个用户登录在 5 分钟内会命中前一个
 * 用户的缓存（fresh，不重新 fetch）。
 *
 * enabled 依赖 userId：未登录态（logout 后回到 /login）不发请求。
 */
export function useUserMenus() {
  const setMenus = useMenuStore((s) => s.setMenus);
  const userId = useUserStore((s) => s.currentUser?.id);
  return useQuery<Menu[]>({
    queryKey: [...userMenusQueryKey, userId],
    enabled: !!userId,
    queryFn: async () => {
      const data = await fetchUserMenus();
      setMenus(data);
      return data;
    },
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}

/** 全量菜单树（管理后台用）。 */
export function useMenuTree() {
  return useQuery<Menu[]>({
    queryKey: [...menuTreeQueryKey],
    queryFn: fetchMenuTree,
    staleTime: 60 * 1000,
  });
}

/** 切角色 / 退出后调用 → useUserMenus 下次访问立即重拉。 */
export function useInvalidateUserMenus(): () => Promise<void> {
  const qc = useQueryClient();
  return async () => {
    await qc.invalidateQueries({ queryKey: [...userMenusQueryKey] });
  };
}
