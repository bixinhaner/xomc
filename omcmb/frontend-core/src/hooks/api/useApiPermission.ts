import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiPermissionApi } from '../../services/api/apiPermissionApi';
import type { ApiEndpoint } from '../../types/system';

/**
 * API permission hooks — endpoint-level RBAC for roles.
 *
 * Wraps `apiPermissionApi`, which uses ID-based payloads
 * (`endpoint_ids: string[]`) — distinct from `adminApi.getRoleApiPermissions`
 * which returns full `ApiPermission` objects. Pages that drive a permission
 * tree by checkbox toggles should prefer this hook.
 *
 * Server-only (no mock equivalent); calls `apiPermissionApi.*` directly.
 */

export function useApiEndpoints() {
  return useQuery<ApiEndpoint[]>({
    queryKey: ['api-permissions', 'endpoints'],
    queryFn: () => apiPermissionApi.listEndpoints(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useRolePermissionIds(roleId: string) {
  return useQuery<string[]>({
    queryKey: ['api-permissions', 'role', roleId],
    queryFn: () => apiPermissionApi.getRolePermissions(roleId),
    enabled: Boolean(roleId),
  });
}

export function useSetRolePermissionIds() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ roleId, endpointIds }: { roleId: string; endpointIds: string[] }) =>
      apiPermissionApi.setRolePermissions(roleId, endpointIds),
    onSuccess: (_, { roleId }) => {
      void queryClient.invalidateQueries({
        queryKey: ['api-permissions', 'role', roleId],
      });
    },
  });
}
