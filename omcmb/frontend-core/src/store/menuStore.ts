// menuStore — 菜单动态加载 P1 全局态。
//
// 与 userStore 同源（zustand + persist + localStorage），但 Set/Map 不能直接 JSON 序列化:
// 用 partialize 把 Set 转为 array 落盘，merge 时再回 Set（参 PRD §4.3.2 警告）。
//
// 切换角色 / 退出登录后由 useUserMenus refetch 自动刷新；本文件不订阅 userStore，
// 调用方在登出 / 切角色 hook 中手动调 useMenuStore.getState().clear()。

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

import type { Menu } from '../types/menu';
import { flattenMenus } from '../types/menu';

interface MenuState {
  /** 树形菜单（直接用于 Sidebar 渲染）。 */
  menus: Menu[];
  /** 扁平菜单（type='button' 在内）。 */
  flatMenus: Menu[];
  /** 全部 button 类型节点的 permission_key 集合（按钮级权限判断用）。 */
  permissionKeys: Set<string>;
  /** 全部 routePath 集合（路由守卫用）。 */
  routePaths: Set<string>;
  /** 是否已经至少完成过一次拉取。首屏 Bootstrap 用此判断要不要阻塞。 */
  loaded: boolean;

  setMenus: (menus: Menu[]) => void;
  clear: () => void;
}

const EMPTY_KEY_SET: Set<string> = new Set();

export const useMenuStore = create<MenuState>()(
  persist(
    (set) => ({
      menus: [],
      flatMenus: [],
      permissionKeys: EMPTY_KEY_SET,
      routePaths: EMPTY_KEY_SET,
      loaded: false,

      setMenus: (menus) => {
        const flat = flattenMenus(menus);
        const permissionKeys = new Set<string>();
        const routePaths = new Set<string>();
        for (const m of flat) {
          if (m.type === 'button' && m.permissionKey) {
            permissionKeys.add(m.permissionKey);
          }
          if (m.routePath) {
            routePaths.add(m.routePath);
          }
        }
        set({
          menus,
          flatMenus: flat,
          permissionKeys,
          routePaths,
          loaded: true,
        });
      },

      clear: () =>
        set({
          menus: [],
          flatMenus: [],
          permissionKeys: new Set<string>(),
          routePaths: new Set<string>(),
          loaded: false,
        }),
    }),
    {
      name: 'omc-menu-store',
      storage: createJSONStorage(() => localStorage),

      // Set 不能 JSON 序列化 → 落盘前转 array。
      partialize: (state) => ({
        menus: state.menus,
        flatMenus: state.flatMenus,
        permissionKeys: Array.from(state.permissionKeys),
        routePaths: Array.from(state.routePaths),
        loaded: state.loaded,
      }),

      // 还原：array → Set。注意类型：persisted 视为 unknown，安全断言后转换。
      merge: (persisted, current) => {
        const p = (persisted ?? {}) as Partial<{
          menus: Menu[];
          flatMenus: Menu[];
          permissionKeys: string[];
          routePaths: string[];
          loaded: boolean;
        }>;
        return {
          ...current,
          menus: p.menus ?? [],
          flatMenus: p.flatMenus ?? [],
          permissionKeys: new Set(p.permissionKeys ?? []),
          routePaths: new Set(p.routePaths ?? []),
          loaded: Boolean(p.loaded),
        };
      },
    },
  ),
);

/** 同步版本：用于事件回调内（无 React render 上下文）。 */
export function hasMenuPermission(key: string): boolean {
  return useMenuStore.getState().permissionKeys.has(key);
}
