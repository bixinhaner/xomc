// Backend Menu 类型对照（参 omcgo/internal/admin/model.go::Menu）。
//
// 后端 JSON 形态 snake_case；http.ts axios 拦截器仅拆 envelope（不做 key 转换）。
// 因此本文件定义两套：
//   - BackendMenu：与后端 JSON 字段名严格 1:1（snake_case）
//   - Menu：前端使用的清晰类型（camelCase），由 mapBackendMenu 转换得到
//
// 设计依据：docs/prd/system/menu-dynamic-loading.md §4.3.2 (P1 menuStore + Hook)。

export type MenuType = 'directory' | 'menu' | 'button';
export type MenuStatus = 'active' | 'disabled';
export type MenuShowStatus = 'show' | 'hide';

/** 后端 Menu JSON 形态。永远用于解码 HTTP 响应，不直接给业务层消费。 */
export interface BackendMenu {
  id: string;
  name: string;
  type: MenuType;
  permission_key: string;
  parent_id?: string | null;
  sort_order: number;
  route_path?: string;
  component_path?: string;
  icon?: string;
  show_status: MenuShowStatus;
  status: MenuStatus;
  created_at?: string;
  updated_at?: string;
  children?: BackendMenu[];
}

/** 前端业务层使用的 Menu 类型（camelCase）。 */
export interface Menu {
  id: string;
  name: string;
  type: MenuType;
  permissionKey: string;
  parentId: string | null;
  sortOrder: number;
  routePath?: string;
  componentPath?: string;
  icon?: string;
  showStatus: MenuShowStatus;
  status: MenuStatus;
  createdAt?: string;
  updatedAt?: string;
  children?: Menu[];
}

/** 后端 → 前端：递归转换菜单子树。 */
export function mapBackendMenu(b: BackendMenu): Menu {
  return {
    id: b.id,
    name: b.name,
    type: b.type,
    permissionKey: b.permission_key,
    parentId: b.parent_id ?? null,
    sortOrder: b.sort_order,
    routePath: b.route_path || undefined,
    componentPath: b.component_path || undefined,
    icon: b.icon || undefined,
    showStatus: b.show_status,
    status: b.status,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
    children: b.children?.map(mapBackendMenu),
  };
}

/** 把树扁平化为线性数组，供 Set 构造（permissionKeys / routePaths）使用。 */
export function flattenMenus(menus: Menu[]): Menu[] {
  const out: Menu[] = [];
  const walk = (list: Menu[]) => {
    for (const m of list) {
      out.push(m);
      if (m.children?.length) walk(m.children);
    }
  };
  walk(menus);
  return out;
}
