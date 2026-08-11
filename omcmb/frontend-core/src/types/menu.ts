// Backend Menu 类型对照（参 omcgo/internal/admin/model.go::Menu）。
//
// 后端 JSON 形态 snake_case；http.ts axios 拦截器仅拆 envelope（不做 key 转换）。
// 因此本文件定义两套：
//   - BackendMenu：与后端 JSON 字段名严格 1:1（snake_case）
//   - Menu：前端使用的清晰类型（camelCase），由 mapBackendMenu 转换得到
//
// 设计依据：docs/prd/system/menu-dynamic-loading.md §4.3.2 (P1 menuStore + Hook)。

export type MenuType = 'directory' | 'menu' | 'button';
// 与后端 admin.MenuStatus 严格一致；后端 DDL chk_status 仅允许 'normal'/'disabled'
// （migrations/000009_sys_admin.sql:38）。历史代码曾把 'active' 当 alias 但与 DB
// 实际值漂移导致动态菜单全空，故彻底统一到 DB 口径。
export type MenuStatus = 'normal' | 'disabled';
export type MenuShowStatus = 'show' | 'hide';

/**
 * 菜单多语言译文字典：键为 locale code（zh-CN/en-US/...），值为对应译文。
 * 由后端 menus.name_i18n JSONB 列承载，菜单管理 UI 可编辑。
 */
export type MenuNameI18n = Record<string, string>;

/** 后端 Menu JSON 形态。永远用于解码 HTTP 响应，不直接给业务层消费。 */
export interface BackendMenu {
  id: string;
  name: string;
  /** 多语言译文（migration 000083 引入），未配置时缺失。 */
  name_i18n?: MenuNameI18n | null;
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
  /** P7-A：菜单关联的 license feature code（多值 OR）；缺失/空=不受 license 控制。 */
  feature_code?: string[] | null;
  children?: BackendMenu[];
}

/** 前端业务层使用的 Menu 类型（camelCase）。 */
export interface Menu {
  id: string;
  name: string;
  nameI18n?: MenuNameI18n;
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
  /** P7-A：菜单关联的 license feature code（多值 OR）；后端已按 license 过滤，前端仅供诊断/扩展用。 */
  featureCode?: string[];
  children?: Menu[];
}

/** 后端 → 前端：递归转换菜单子树。 */
export function mapBackendMenu(b: BackendMenu): Menu {
  return {
    id: b.id,
    name: b.name,
    nameI18n: b.name_i18n ?? undefined,
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
    featureCode: b.feature_code ?? undefined,
    children: b.children?.map(mapBackendMenu),
  };
}

/**
 * 根据当前 locale 解析菜单显示名（3 级 fallback）：
 *   1. nameI18n[locale]   → 当前语言译文
 *   2. nameI18n['zh-CN']  → 中文兜底（覆盖率最广的语言）
 *   3. name               → 终极 fallback（DB 原始 name 字段）
 *
 * 设计依据：方案 C — DB JSONB 主路径，菜单管理 UI 编辑译文。
 * 单测见 menu.test.ts。
 */
export function resolveMenuLabel(
  menu: Pick<Menu, 'name' | 'nameI18n'>,
  locale: string,
): string {
  const i18n = menu.nameI18n;
  if (i18n) {
    if (i18n[locale]) return i18n[locale];
    if (i18n['zh-CN']) return i18n['zh-CN'];
  }
  return menu.name;
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
