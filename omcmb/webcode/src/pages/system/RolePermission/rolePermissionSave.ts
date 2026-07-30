export interface RolePermissionBindingPayload {
  roleId: string;
  menuIds: string[];
  endpointIds: string[];
  deviceGroupIds: string[];
  networkTypes: string[];
}

export interface RolePermissionBindingWriters {
  setMenus(input: { roleId: string; menuIds: string[] }): Promise<unknown>;
  setApiPermissions(input: { roleId: string; endpointIds: string[] }): Promise<unknown>;
  setDeviceGroups(input: {
    roleId: string;
    deviceGroupIds: string[];
    networkTypes: string[];
  }): Promise<unknown>;
}

export async function saveRolePermissionBindings(
  writers: RolePermissionBindingWriters,
  payload: RolePermissionBindingPayload,
): Promise<void> {
  await writers.setMenus({
    roleId: payload.roleId,
    menuIds: payload.menuIds,
  });
  await writers.setApiPermissions({
    roleId: payload.roleId,
    endpointIds: payload.endpointIds,
  });
  await writers.setDeviceGroups({
    roleId: payload.roleId,
    deviceGroupIds: payload.deviceGroupIds,
    networkTypes: payload.networkTypes,
  });
}
