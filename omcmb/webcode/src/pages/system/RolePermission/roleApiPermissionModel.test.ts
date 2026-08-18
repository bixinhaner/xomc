import { describe, expect, it } from 'vitest';
import type { Menu } from '@core/types/menu';
import type { ApiEndpoint } from '@core/types/system';
import {
  applyNewApiSuggestions,
  inferReadApiEndpointIds,
} from './roleApiPermissionModel';

const menus: Menu[] = [
  {
    id: 'menu-system-role',
    name: '角色管理',
    type: 'menu',
    permissionKey: 'system:role',
    parentId: null,
    sortOrder: 1,
    showStatus: 'show',
    status: 'normal',
    children: [
      {
        id: 'button-system-role-edit',
        name: '修改',
        type: 'button',
        permissionKey: 'system:role:edit',
        parentId: 'menu-system-role',
        sortOrder: 1,
        showStatus: 'show',
        status: 'normal',
      },
    ],
  },
  {
    id: 'menu-unmapped',
    name: '未映射页面',
    type: 'menu',
    permissionKey: 'custom:unmapped',
    parentId: null,
    sortOrder: 2,
    showStatus: 'show',
    status: 'normal',
  },
  {
    id: 'menu-dashboard',
    name: '仪表板',
    type: 'menu',
    permissionKey: 'dashboard:home',
    parentId: null,
    sortOrder: 3,
    showStatus: 'show',
    status: 'normal',
  },
  {
    id: 'menu-device-list',
    name: '设备列表',
    type: 'menu',
    permissionKey: 'device:list',
    parentId: null,
    sortOrder: 4,
    showStatus: 'show',
    status: 'normal',
  },
  {
    id: 'menu-device-access-control',
    name: '接入控制',
    type: 'menu',
    permissionKey: 'device:access-control',
    parentId: null,
    sortOrder: 5,
    showStatus: 'show',
    status: 'normal',
  },
];

const endpoints: ApiEndpoint[] = [
  {
    id: 'endpoint-get-roles',
    path: '/api/v1/admin/roles',
    method: 'GET',
    name: '角色列表',
    apiGroup: 'roles',
    module: 'admin',
    description: '列出角色',
  },
  {
    id: 'endpoint-get-menus',
    path: '/api/v1/admin/menus/tree',
    method: 'get',
    name: '菜单树',
    apiGroup: 'menus',
    module: 'admin',
    description: '读取菜单树',
  },
  {
    id: 'endpoint-put-role',
    path: '/api/v1/admin/roles/:id',
    method: 'PUT',
    name: '修改角色',
    apiGroup: 'roles',
    module: 'admin',
    description: '修改角色主体',
  },
  {
    id: 'endpoint-get-dashboard',
    path: '/api/v1/dashboard/summary',
    method: 'GET',
    name: '仪表板汇总',
    apiGroup: 'dashboard',
    module: 'dashboard',
    description: '读取仪表板汇总',
  },
  {
    id: 'endpoint-put-dashboard',
    path: '/api/v1/dashboard/widgets',
    method: 'PUT',
    name: '保存仪表板组件',
    apiGroup: 'dashboard',
    module: 'dashboard',
    description: '保存仪表板组件',
  },
  {
    id: 'endpoint-get-devices',
    path: '/api/v1/devices',
    method: 'GET',
    name: '设备列表',
    apiGroup: 'devices',
    module: 'device',
    description: '读取设备列表',
  },
  {
    id: 'endpoint-get-device-access-states',
    path: '/api/v1/device-access/states',
    method: 'GET',
    name: '接入状态',
    apiGroup: 'device-access',
    module: 'device-access',
    description: '读取接入状态',
  },
  {
    id: 'endpoint-post-device-access-review',
    path: '/api/v1/device-access/candidates/:candidateID/review',
    method: 'POST',
    name: '候选审核',
    apiGroup: 'device-access',
    module: 'device-access',
    description: '审核候选设备',
  },
];

describe('inferReadApiEndpointIds', () => {
  it('suggests only GET endpoints from explicitly mapped menu groups', () => {
    expect(inferReadApiEndpointIds(
      ['menu-system-role'],
      menus,
      endpoints,
    )).toEqual(['endpoint-get-roles', 'endpoint-get-menus']);
  });

  it('never suggests write endpoints', () => {
    expect(inferReadApiEndpointIds(
      ['menu-system-role'],
      menus,
      endpoints,
    )).not.toContain('endpoint-put-role');
  });

  it('returns no suggestions for an unmapped menu', () => {
    expect(inferReadApiEndpointIds(
      ['menu-unmapped'],
      menus,
      endpoints,
    )).toEqual([]);
  });

  it('suggests the dashboard and device-list read groups used by issue 225', () => {
    expect(inferReadApiEndpointIds(
      ['menu-dashboard', 'menu-device-list'],
      menus,
      endpoints,
    )).toEqual(['endpoint-get-dashboard', 'endpoint-get-devices']);
  });

  it('suggests only read APIs when access-control menu is granted', () => {
    expect(inferReadApiEndpointIds(
      ['menu-device-access-control'],
      menus,
      endpoints,
    )).toEqual(['endpoint-get-device-access-states']);
  });
});

describe('applyNewApiSuggestions', () => {
  it('keeps administrator selections before appending new suggestions', () => {
    expect(applyNewApiSuggestions(
      ['endpoint-put-role'],
      ['endpoint-get-roles'],
      [],
      [],
    )).toEqual({
      selectedIds: ['endpoint-put-role', 'endpoint-get-roles'],
      suggestedIds: ['endpoint-get-roles'],
      autoSelectedIds: ['endpoint-get-roles'],
    });
  });

  it('does not re-add a suggestion the administrator removed', () => {
    expect(applyNewApiSuggestions(
      [],
      ['endpoint-get-roles'],
      ['endpoint-get-roles'],
      [],
    )).toEqual({
      selectedIds: [],
      suggestedIds: ['endpoint-get-roles'],
      autoSelectedIds: [],
    });
  });

  it('deduplicates suggestions without reordering administrator selections', () => {
    expect(applyNewApiSuggestions(
      ['endpoint-put-role', 'endpoint-get-roles'],
      ['endpoint-get-roles', 'endpoint-get-menus', 'endpoint-get-menus'],
      [],
      [],
    )).toEqual({
      selectedIds: ['endpoint-put-role', 'endpoint-get-roles', 'endpoint-get-menus'],
      suggestedIds: ['endpoint-get-roles', 'endpoint-get-menus'],
      autoSelectedIds: ['endpoint-get-menus'],
    });
  });

  it('removes stale automatic grants when the corresponding menu is unchecked', () => {
    expect(applyNewApiSuggestions(
      ['endpoint-put-role', 'endpoint-get-dashboard'],
      [],
      ['endpoint-get-dashboard'],
      ['endpoint-get-dashboard'],
    )).toEqual({
      selectedIds: ['endpoint-put-role'],
      suggestedIds: ['endpoint-get-dashboard'],
      autoSelectedIds: [],
    });
  });

  it('keeps an explicitly selected endpoint when menu suggestions shrink', () => {
    expect(applyNewApiSuggestions(
      ['endpoint-get-dashboard'],
      [],
      ['endpoint-get-dashboard'],
      [],
    )).toEqual({
      selectedIds: ['endpoint-get-dashboard'],
      suggestedIds: ['endpoint-get-dashboard'],
      autoSelectedIds: [],
    });
  });
});
