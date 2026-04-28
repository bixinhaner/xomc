import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adminApi } from '../../services/api/adminApi';
import type { ApiPermission, MenuItem } from '../../types/system';

/**
 * Admin hooks — high-level administrative operations not covered by useSystem.
 *
 * Scope:
 *   - Current user password change (`/auth/change-password`)
 *   - Role-level API permission assignment
 *   - Role-level device group / network type binding
 *   - Menu tree retrieval (full tree + per-user tree)
 *
 * For user / role / group CRUD see `useSystem`.
 *
 * NOTE: These endpoints are server-only (no mock equivalents); we call
 * `adminApi.*` directly without the `useMock` switch.
 */

// ---------------- Current user ----------------

export function useChangePassword() {
  return useMutation({
    mutationFn: (payload: { old_password: string; new_password: string }) =>
      adminApi.changePassword(payload),
  });
}

// ---------------- Role API permissions ----------------

export function useApiEndpointsList() {
  return useQuery<unknown[]>({
    queryKey: ['admin', 'api-endpoints', 'list'],
    queryFn: () => adminApi.listApiEndpoints(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useRoleApiPermissions(roleId: string) {
  return useQuery<ApiPermission[]>({
    queryKey: ['admin', 'roles', roleId, 'api-permissions'],
    queryFn: () => adminApi.getRoleApiPermissions(roleId),
    enabled: Boolean(roleId),
  });
}

export function useSetRoleApiPermissions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ roleId, permissions }: { roleId: string; permissions: ApiPermission[] }) =>
      adminApi.setRoleApiPermissions(roleId, permissions),
    onSuccess: (_, { roleId }) => {
      void queryClient.invalidateQueries({
        queryKey: ['admin', 'roles', roleId, 'api-permissions'],
      });
    },
  });
}

// ---------------- Role device groups ----------------

export function useRoleDeviceGroups(roleId: string) {
  return useQuery<{ deviceGroupIds: string[]; networkTypes: string[] }>({
    queryKey: ['admin', 'roles', roleId, 'device-groups'],
    queryFn: () => adminApi.getRoleDeviceGroups(roleId),
    enabled: Boolean(roleId),
  });
}

export function useSetRoleDeviceGroups() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      roleId,
      payload,
    }: {
      roleId: string;
      payload: { deviceGroupIds: string[]; networkTypes: string[] };
    }) => adminApi.setRoleDeviceGroups(roleId, payload),
    onSuccess: (_, { roleId }) => {
      void queryClient.invalidateQueries({
        queryKey: ['admin', 'roles', roleId, 'device-groups'],
      });
    },
  });
}

// ---------------- Menu tree ----------------

export function useMenuTree() {
  return useQuery<MenuItem[]>({
    queryKey: ['admin', 'menus', 'tree'],
    queryFn: () => adminApi.getMenuTree(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useUserMenuTree() {
  return useQuery<MenuItem[]>({
    queryKey: ['admin', 'menus', 'user-tree'],
    queryFn: () => adminApi.getUserMenuTree(),
    staleTime: 5 * 60 * 1000,
  });
}
