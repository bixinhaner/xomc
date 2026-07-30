import { describe, expect, it } from 'vitest';
import { saveRolePermissionBindings } from './rolePermissionSave';

const payload = {
  roleId: 'role-225',
  menuIds: ['menu-dashboard', 'menu-device-list'],
  endpointIds: ['endpoint-dashboard-summary', 'endpoint-device-list'],
  deviceGroupIds: ['default-device-group'],
  networkTypes: ['lte', 'nr'],
};

describe('saveRolePermissionBindings', () => {
  it('persists exact API endpoint IDs between menu and data bindings', async () => {
    const events: Array<{ kind: string; value: unknown }> = [];

    await saveRolePermissionBindings({
      setMenus: async (input) => { events.push({ kind: 'menus', value: input }); },
      setApiPermissions: async (input) => { events.push({ kind: 'api', value: input }); },
      setDeviceGroups: async (input) => { events.push({ kind: 'groups', value: input }); },
    }, payload);

    expect(events).toEqual([
      {
        kind: 'menus',
        value: { roleId: 'role-225', menuIds: ['menu-dashboard', 'menu-device-list'] },
      },
      {
        kind: 'api',
        value: {
          roleId: 'role-225',
          endpointIds: ['endpoint-dashboard-summary', 'endpoint-device-list'],
        },
      },
      {
        kind: 'groups',
        value: {
          roleId: 'role-225',
          deviceGroupIds: ['default-device-group'],
          networkTypes: ['lte', 'nr'],
        },
      },
    ]);
  });

  it('stops before data bindings when API persistence fails', async () => {
    const events: string[] = [];

    await expect(saveRolePermissionBindings({
      setMenus: async () => { events.push('menus'); },
      setApiPermissions: async () => {
        events.push('api');
        throw new Error('API permission persistence failed');
      },
      setDeviceGroups: async () => { events.push('groups'); },
    }, payload)).rejects.toThrow('API permission persistence failed');

    expect(events).toEqual(['menus', 'api']);
  });
});
