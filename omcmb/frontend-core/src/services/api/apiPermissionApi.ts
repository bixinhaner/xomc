import http from '../http';
import type { ApiEndpoint } from '../../types/system';
import { adminApi } from './adminApi';

export const apiPermissionApi = {
  // 获取所有 API 端点列表（分页拼装）。
  //
  // 历史 bug：曾直接把 `data.items` 当 ApiEndpoint[] 用，跳过了
  // adminApi.mapBackendApiEndpoint 的 snake_case → camelCase 映射，导致
  // ep.apiGroup 永远 undefined → 角色面板 API 权限树全部归到「other」分组。
  // 现统一走 adminApi.getApiEndpoints（内部已 map），分页拼装到末页为止。
  listEndpoints: async (): Promise<ApiEndpoint[]> => {
    const all: ApiEndpoint[] = [];
    let page = 1;
    const pageSize = 200;
    for (;;) {
      const resp = await adminApi.getApiEndpoints({ page, pageSize });
      all.push(...resp.items);
      if (all.length >= resp.total || resp.items.length === 0) break;
      page++;
    }
    return all;
  },

  // 获取角色的API权限（返回 endpoint ID 列表）
  getRolePermissions: async (roleId: string): Promise<string[]> => {
    const { data } = await http.get<{ endpoint_ids: string[] }>(`/admin/roles/${roleId}/api-permissions`);
    return data.endpoint_ids || [];
  },

  // 设置角色的API权限（发送 endpoint ID 列表）
  setRolePermissions: async (roleId: string, endpointIds: string[]): Promise<void> => {
    await http.put(`/admin/roles/${roleId}/api-permissions`, { endpoint_ids: endpointIds });
  },
};
