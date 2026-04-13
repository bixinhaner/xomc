import http from '../http';
import type { ApiEndpoint, ApiPermission } from '@/types/system';

export const apiPermissionApi = {
  // 获取所有API端点列表
  listEndpoints: () => {
    return http.get<{ data: ApiEndpoint[] }>('/admin/api-endpoints')
      .then(res => res.data.data);
  },

  // 获取角色的API权限
  getRolePermissions: (roleId: string) => {
    return http.get<{ data: ApiPermission[] }>(`/admin/roles/${roleId}/api-permissions`)
      .then(res => res.data.data);
  },

  // 设置角色的API权限
  setRolePermissions: (roleId: string, permissions: ApiPermission[]) => {
    return http.put(`/admin/roles/${roleId}/api-permissions`, { permissions });
  },
};
