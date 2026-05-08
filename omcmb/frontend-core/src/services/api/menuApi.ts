// menuApi.ts — 菜单动态加载 P1 起的统一 API 客户端。
//
// 与 adminApi.ts 中遗留的 getMenuTree / getUserMenuTree 区别：
//   - 类型干净：响应解码用 BackendMenu（snake_case） → mapBackendMenu → Menu（camelCase）
//   - 路径正确：当前用户菜单走 GET /auth/menus（含 currentRoleID 感知），
//     不再走已破的 /admin/menus/user
//   - 共享：menuStore / useUserMenus / useMenuTree / RolePermission 全部消费本文件
//
// 设计依据：docs/prd/system/menu-dynamic-loading.md §4.2.1 / §4.3.2。

import { http } from '../http';
import type { BackendMenu, Menu } from '../../types/menu';
import { mapBackendMenu } from '../../types/menu';

export interface SetRoleMenusPayload {
  menu_ids: string[];
}

/**
 * 当前用户当前激活角色的菜单树。
 *
 * 后端逻辑（auth_handler.go::GetUserMenusByRole）：
 *   - 优先 JWT claims.CurrentRoleID 的菜单
 *   - 否则回退用户全部角色菜单的并集
 *   - users.source='builtIn' 超管旁路：返回所有 active 菜单
 */
export async function fetchUserMenus(): Promise<Menu[]> {
  const { data } = await http.get<BackendMenu[]>('/auth/menus');
  return (data ?? []).map(mapBackendMenu);
}

/**
 * 全量菜单树（管理后台 RolePermission / MenuManagement 用）。
 *
 * 后端逻辑（menu_handler.go::GetMenuTree）：包一层 {data: [...]}，
 * 故响应类型用 `{ data: BackendMenu[] }` 而非裸数组。
 */
export async function fetchMenuTree(): Promise<Menu[]> {
  const { data } = await http.get<{ data: BackendMenu[] }>('/admin/menus/tree');
  return (data?.data ?? []).map(mapBackendMenu);
}

/** 角色已绑定菜单 ID 列表（RolePermission 编辑用）。 */
export async function fetchRoleMenuIds(roleId: string): Promise<string[]> {
  const { data } = await http.get<{ menu_ids?: string[] }>(`/admin/roles/${roleId}/menus`);
  return data?.menu_ids ?? [];
}

/** 设置角色 ↔ 菜单绑定。 */
export async function setRoleMenus(roleId: string, payload: SetRoleMenusPayload): Promise<void> {
  await http.put(`/admin/roles/${roleId}/menus`, payload);
}
