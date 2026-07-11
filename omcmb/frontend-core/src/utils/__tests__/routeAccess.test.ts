import { describe, expect, it } from 'vitest'
import { isRouteAllowed } from '../routeAccess'

const base = {
  routePaths: new Set(['/device/list']),
  dynamicEnabled: true,
  menuLoaded: true,
}

describe('isRouteAllowed', () => {
  it('按 V1 菜单路径精确放行及其子路径', () => {
    expect(isRouteAllowed('/device/list', base)).toBe(true)
    expect(isRouteAllowed('/device/list/detail', base)).toBe(true)
    expect(isRouteAllowed('/device/stats', base)).toBe(false)
  })

  it('管理员、静态菜单和固定公共路径放行', () => {
    expect(isRouteAllowed('/system/users', { ...base, role: 'admin' })).toBe(true)
    expect(isRouteAllowed('/system/users', { ...base, isSuperAdmin: true })).toBe(true)
    expect(isRouteAllowed('/system/users', { ...base, dynamicEnabled: false })).toBe(true)
    expect(isRouteAllowed('/system/users', { ...base, menuLoaded: false })).toBe(true)
    expect(isRouteAllowed('/dashboard', base)).toBe(true)
    expect(isRouteAllowed('/404', base)).toBe(true)
  })
})
