import { describe, it, expect } from 'vitest';
import { firstSegment, isRouteAllowed, isModuleVisible, type RouteAccessCtx } from '../routeAccess';

const base: RouteAccessCtx = {
  role: 'operator',
  isSuperAdmin: false,
  routePaths: new Set<string>(['/device/list', '/alarm/current']),
  dynamicEnabled: true,
  menuLoaded: true,
};

describe('firstSegment', () => {
  it('剥掉 v2/v3 base 前缀取首段', () => {
    expect(firstSegment('/v2/devices')).toBe('devices');
    expect(firstSegment('/v3/fleet/detail/abc')).toBe('fleet');
    expect(firstSegment('/device/list')).toBe('device');
    expect(firstSegment('/v2')).toBe('');
    expect(firstSegment('/')).toBe('');
  });
});

describe('isRouteAllowed — admin/超管 bypass', () => {
  it('admin 角色恒放行（即便菜单未含该模块）', () => {
    expect(isRouteAllowed('/v2/system/users', { ...base, role: 'admin' })).toBe(true);
  });
  it('超管恒放行', () => {
    expect(isRouteAllowed('/v3/backup/policy', { ...base, role: 'operator', isSuperAdmin: true })).toBe(true);
  });
});

describe('isRouteAllowed — 非 admin 模块级门禁', () => {
  it('门禁未开 → 放行', () => {
    expect(isRouteAllowed('/v2/system/users', { ...base, dynamicEnabled: false })).toBe(true);
  });
  it('菜单未加载 → 放行（不误拦首屏）', () => {
    expect(isRouteAllowed('/v2/system/users', { ...base, menuLoaded: false })).toBe(true);
  });
  it('授予了该模块（device）→ v2 /devices 放行', () => {
    expect(isRouteAllowed('/v2/devices', base)).toBe(true);
  });
  it('授予了该模块（device）→ v3 /fleet 放行（slug 不同但同模块）', () => {
    expect(isRouteAllowed('/v3/fleet/group', base)).toBe(true);
  });
  it('未授予的模块（system）→ 拦截 403', () => {
    expect(isRouteAllowed('/v2/system/users', base)).toBe(false);
  });
  it('未授予的模块（backup）→ 拦截', () => {
    expect(isRouteAllowed('/v3/backup/policy', base)).toBe(false);
  });
  it('恒放行段（dashboard/bridge）即便菜单不含也放行', () => {
    expect(isRouteAllowed('/v2/dashboard', base)).toBe(true);
    expect(isRouteAllowed('/v3/bridge', base)).toBe(true);
  });
  it('未登记的模块 → 放行（不过度拦截）', () => {
    expect(isRouteAllowed('/v2/some-unknown-module', base)).toBe(true);
  });
});

describe('isModuleVisible — 侧栏菜单驱动（无 admin bypass，三皮肤一致）', () => {
  const rp = new Set<string>(['/device/list', '/alarm/current', '/performance/device-view']);
  it('有可见菜单的模块 → 显示（v2 /devices、v3 /fleet 同模块）', () => {
    expect(isModuleVisible('/v2/devices', rp)).toBe(true);
    expect(isModuleVisible('/v3/fleet', rp)).toBe(true);
    expect(isModuleVisible('/v2/performance/device-view', rp)).toBe(true);
  });
  it('无可见菜单的模块（backup/software/mr）→ 隐藏', () => {
    expect(isModuleVisible('/v2/backup', rp)).toBe(false);
    expect(isModuleVisible('/v3/software/version', rp)).toBe(false);
    expect(isModuleVisible('/v2/mr/files', rp)).toBe(false);
  });
  it('dashboard/bridge 恒显示', () => {
    expect(isModuleVisible('/v2/dashboard', rp)).toBe(true);
    expect(isModuleVisible('/v3/bridge', rp)).toBe(true);
  });
});
