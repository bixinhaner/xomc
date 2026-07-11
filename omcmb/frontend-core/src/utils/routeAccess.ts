/** V1 动态菜单下的路由访问判定。 */
export interface RouteAccessCtx {
  role?: string
  isSuperAdmin?: boolean
  routePaths: ReadonlySet<string>
  dynamicEnabled: boolean
  menuLoaded: boolean
}

const ALWAYS_ALLOWED_PATHS = new Set(['/dashboard', '/403', '/404', '/login', '/notifications'])

export function isRouteAllowed(pathname: string, ctx: RouteAccessCtx): boolean {
  if (ctx.role === 'admin' || ctx.isSuperAdmin === true) return true
  if (!ctx.dynamicEnabled || !ctx.menuLoaded) return true
  if (ALWAYS_ALLOWED_PATHS.has(pathname)) return true

  for (const path of ctx.routePaths) {
    if (pathname === path || pathname.startsWith(`${path}/`)) return true
  }
  return false
}
