/**
 * 三皮肤统一的路由访问判定（动态菜单门禁）。
 *
 * 背景：v1 用 routePath 精确门禁；v2/v3 路由 slug 与菜单（v1 式 routePath）不一致，
 * 无法直接精确匹配。本工具提供**模块级**门禁 + **admin/超管 bypass**，让三皮肤门禁
 * 行为一致：
 *   - admin / 超管：整体放行（授权全量）——系统管理员应能进所有业务路由。
 *   - 其他角色：按"模块"门禁——皮肤路由首段映射到 v1 菜单前缀，用户菜单里只要授予
 *     了该模块下任一路由，即放行整模块；未登记的模块默认放行（不过度拦截）。
 *
 * 纯函数，便于单测；皮肤侧的 RouteGuard 组件读 store 后调用本函数。
 */

/** 皮肤路由首段 → v1 菜单 routePath 前缀（仅服务非 admin 角色的模块级门禁）。 */
const SEGMENT_TO_MENU_PREFIX: Record<string, string> = {
  device: '/device', devices: '/device', fleet: '/device',
  alarm: '/alarm', alarms: '/alarm',
  performance: '/performance',
  config: '/config',
  topology: '/topology',
  mml: '/mml',
  backup: '/backup',
  file: '/file', files: '/file',
  transfer: '/transfer',
  mr: '/mr',
  ops: '/ops',
  report: '/report', reports: '/report',
  software: '/software',
  system: '/system',
  product: '/product',
  license: '/license',
  log: '/log', logs: '/log',
  eventlog: '/eventlog',
}

/** 恒放行的首段（登录落点 / 跨域功能 / 错误页）。 */
const ALWAYS_ALLOWED_SEGMENTS = new Set(['', 'dashboard', 'bridge', '403', '404', 'login', 'notification', 'notifications'])

export interface RouteAccessCtx {
  role?: string
  isSuperAdmin?: boolean
  routePaths: ReadonlySet<string>
  dynamicEnabled: boolean
  menuLoaded: boolean
}

/** 去掉皮肤 base 前缀（/v2 /v3），取路径首段。 */
export function firstSegment(pathname: string): string {
  const stripped = pathname.replace(/^\/(v2|v3)(?=\/|$)/, '')
  return stripped.split('/').filter(Boolean)[0] || ''
}

/**
 * 判定某模块（按其代表路由）是否应出现在侧边栏——与 v1 菜单驱动侧栏对齐：
 * 仅当用户菜单（routePaths=可见菜单）含该模块前缀下任一路由时显示。
 *
 * 注意：本函数**不做 admin bypass**——v1 侧栏对 admin 也是菜单驱动（curated），
 * 故三皮肤 admin 侧栏一致（都只显示有可见菜单的模块）；admin 仍可经门禁 bypass 直达
 * 未在侧栏的页面。dashboard/bridge 等恒显示；未登记模块默认显示（不隐藏未知）。
 */
export function isModuleVisible(pathname: string, routePaths: ReadonlySet<string>): boolean {
  const seg = firstSegment(pathname);
  if (ALWAYS_ALLOWED_SEGMENTS.has(seg)) return true;
  const prefix = SEGMENT_TO_MENU_PREFIX[seg];
  if (!prefix) return true;
  for (const p of routePaths) {
    if (p === prefix || p.startsWith(prefix + '/')) return true;
  }
  return false;
}

/** 判定某路由对当前用户是否可访问。 */
export function isRouteAllowed(pathname: string, ctx: RouteAccessCtx): boolean {
  // admin / 超管整体放行（授权全量）
  if (ctx.role === 'admin' || ctx.isSuperAdmin === true) return true
  // 非动态模式 / 菜单未加载 → 放行，避免首屏未就绪时误拦
  if (!ctx.dynamicEnabled || !ctx.menuLoaded) return true
  const seg = firstSegment(pathname)
  if (ALWAYS_ALLOWED_SEGMENTS.has(seg)) return true
  const prefix = SEGMENT_TO_MENU_PREFIX[seg]
  if (!prefix) return true // 未登记模块 → 放行
  // 模块级：用户菜单含该模块前缀下任一路由即放行整模块
  for (const p of ctx.routePaths) {
    if (p === prefix || p.startsWith(prefix + '/')) return true
  }
  return false
}
