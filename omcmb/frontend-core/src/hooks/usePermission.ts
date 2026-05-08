// usePermission — 按钮级权限 hook。
//
// 设计依据：docs/prd/system/menu-dynamic-loading.md §4.3.6。
// 消费约定（CRITICAL）：无权限按钮 disabled + Tooltip 提示，**不要隐藏**。

import { useMenuStore, hasMenuPermission } from '../store/menuStore';

/** 判断当前用户是否拥有指定 permission_key（来自菜单 button 类型节点）。 */
export function usePermission(key: string): boolean {
  return useMenuStore((s) => s.permissionKeys.has(key));
}

/** 同步版本：用于事件回调内。 */
export const hasPermission = hasMenuPermission;
