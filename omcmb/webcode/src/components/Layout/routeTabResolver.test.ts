import { describe, expect, it } from 'vitest';
import type { Menu } from '@core/types/menu';
import { NAV_CONFIG } from './Sidebar/navConfig';
import { resolveRouteTab } from './routeTabResolver';

const dynamicKpiMenu: Menu = {
  id: 'kpi-config',
  name: '首页 KPI 配置',
  nameI18n: { 'zh-CN': '首页 KPI 配置', 'en-US': 'Dashboard KPI Config' },
  type: 'menu',
  permissionKey: 'system:kpi-config',
  parentId: 'system',
  sortOrder: 20,
  routePath: '/system/kpi-config',
  showStatus: 'show',
  status: 'normal',
};

const dynamicActiveIntelligenceMenu: Menu = {
  ...dynamicKpiMenu,
  id: 'active-intelligence',
  name: '主动智能',
  nameI18n: { 'zh-CN': '主动智能', 'en-US': 'Active Intelligence' },
  permissionKey: 'system:active-intelligence',
  routePath: '/system/active-intelligence',
};

describe('resolveRouteTab', () => {
  it('resolves a visible dynamic menu route for an address-bar deep link', () => {
    expect(resolveRouteTab({
      pathname: '/system/kpi-config',
      dynamicMenuEnabled: true,
      dynamicMenus: [dynamicKpiMenu],
      staticNav: NAV_CONFIG,
      locale: 'zh-CN',
      isAdmin: true,
      isSuperAdmin: false,
    })).toEqual({
      key: '/system/kpi-config',
      label: '首页 KPI 配置',
      path: '/system/kpi-config',
      closable: true,
      labelRaw: true,
    });
  });

  it('keeps the active-intelligence tab after a direct link or refresh', () => {
    expect(resolveRouteTab({
      pathname: '/system/active-intelligence',
      dynamicMenuEnabled: true,
      dynamicMenus: [dynamicActiveIntelligenceMenu],
      staticNav: NAV_CONFIG,
      locale: 'zh-CN',
      isAdmin: true,
      isSuperAdmin: false,
    })).toEqual({
      key: '/system/active-intelligence',
      label: '主动智能',
      path: '/system/active-intelligence',
      closable: true,
      labelRaw: true,
    });
  });

  it('does not create tabs for hidden or unknown routes', () => {
    expect(resolveRouteTab({
      pathname: '/system/kpi-config',
      dynamicMenuEnabled: true,
      dynamicMenus: [{ ...dynamicKpiMenu, showStatus: 'hide' }],
      staticNav: NAV_CONFIG,
      locale: 'zh-CN',
      isAdmin: true,
      isSuperAdmin: false,
    })).toBeNull();
    expect(resolveRouteTab({
      pathname: '/403',
      dynamicMenuEnabled: true,
      dynamicMenus: [dynamicKpiMenu],
      staticNav: NAV_CONFIG,
      locale: 'zh-CN',
      isAdmin: true,
      isSuperAdmin: false,
    })).toBeNull();
  });

  it('resolves the license recovery surface even though its menu is hidden', () => {
    expect(resolveRouteTab({
      pathname: '/license',
      dynamicMenuEnabled: true,
      dynamicMenus: [],
      staticNav: NAV_CONFIG,
      locale: 'zh-CN',
      isAdmin: true,
      isSuperAdmin: false,
    })).toEqual({
      key: '/license',
      label: 'nav.systemLicense',
      path: '/license',
      closable: true,
      labelRaw: false,
    });
    expect(resolveRouteTab({
      pathname: '/license/history',
      dynamicMenuEnabled: true,
      dynamicMenus: [],
      staticNav: NAV_CONFIG,
      locale: 'zh-CN',
      isAdmin: true,
      isSuperAdmin: false,
    })?.label).toBe('systemLicense.history.title');
  });

  it('keeps the static KPI route admin-only', () => {
    expect(resolveRouteTab({
      pathname: '/system/kpi-config',
      dynamicMenuEnabled: false,
      dynamicMenus: [],
      staticNav: NAV_CONFIG,
      locale: 'zh-CN',
      isAdmin: false,
      isSuperAdmin: false,
    })).toBeNull();
    expect(resolveRouteTab({
      pathname: '/system/kpi-config',
      dynamicMenuEnabled: false,
      dynamicMenus: [],
      staticNav: NAV_CONFIG,
      locale: 'zh-CN',
      isAdmin: true,
      isSuperAdmin: false,
    })?.key).toBe('sys-kpi-config');
  });
});
