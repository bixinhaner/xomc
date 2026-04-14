import http from '../http';
import type { ApiEndpoint } from '@/types/system';

export const apiPermissionApi = {
  // 获取所有API端点列表（大分页一次拉取）
  listEndpoints: async (): Promise<ApiEndpoint[]> => {
    const { data } = await http.get<{ items: ApiEndpoint[]; total: number }>('/admin/api-endpoints', {
      params: { page: 1, page_size: 5000 },
    });
    return data.items || [];
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
