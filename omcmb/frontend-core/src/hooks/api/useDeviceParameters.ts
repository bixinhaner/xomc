import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { ParameterFilter, ParameterUpdateRequest } from '../../types/deviceParameter';
import type { PageRequest } from '../../types/pagination';
import { deviceParameterService } from '../../mock/services/deviceParameterService';
import { deviceParameterApi } from '../../services/api/deviceParameterApi';
import { useMock } from '../../services/apiSwitch';

type ParamApi = typeof deviceParameterApi & { syncConfigFile?: (deviceId: string) => Promise<unknown> };
const api: ParamApi = useMock
  ? (deviceParameterService as unknown as ParamApi)
  : (deviceParameterApi as ParamApi);

export function useDeviceParameters(
  deviceId: string,
  params?: ParameterFilter & PageRequest
) {
  return useQuery({
    queryKey: ['devices', 'parameters', deviceId, params],
    queryFn: () => api.getParameters(deviceId, params),
    enabled: Boolean(deviceId),
  });
}

export function useSearchParameters(deviceId: string, query: string, limit = 100, enabled = true) {
  return useQuery({
    queryKey: ['devices', 'parameters', 'search', deviceId, query, limit],
    queryFn: async () => {
      if (typeof api.searchParameters === 'function') {
        return api.searchParameters(deviceId, query, limit);
      }
      const result = await api.getParameters(deviceId, { search: query, page: 1, pageSize: limit });
      return result.items;
    },
    enabled: Boolean(deviceId) && Boolean(query) && enabled,
  });
}

export function useDeviceParameterTree(deviceId: string) {
  return useQuery({
    queryKey: ['devices', 'parameter-tree', deviceId],
    queryFn: () => api.getParameterTree(deviceId),
    enabled: Boolean(deviceId),
  });
}

export function useUpdateParameters() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceId,
      parameters,
    }: {
      deviceId: string;
      parameters: ParameterUpdateRequest[];
    }) => api.updateParameters(deviceId, parameters),
    onSuccess: (_result, { deviceId }) => {
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameters', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameters', 'search', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-tree', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'children', deviceId],
      });
    },
  });
}

// T-0126: useSyncParameters 已下线（Path A）。
// 切换到 useDevices.useSyncDeviceParams（Path B + reason="manual" 完整接入 F09 触发链）。

export function useDiscoverParameters() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (deviceId: string) => api.discoverParameters(deviceId),
    onSuccess: (_result, deviceId) => {
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'sync-status', deviceId],
      });
    },
  });
}

// useSyncStatus mount 时永远拉一次后端 sync-status;refetchInterval 由响应中的
// status 自适应:syncing 时每 3s 持续轮询,idle 时停止(用户切 tab 回来重新 mount
// 时自动再拉一次最新状态)。staleTime:0 避免 React Query cache 残留旧 syncing 值。
export function useSyncStatus(deviceId: string) {
  return useQuery({
    queryKey: ['devices', 'sync-status', deviceId],
    queryFn: () => api.getSyncStatus(deviceId),
    enabled: Boolean(deviceId),
    refetchInterval: (q) => (q.state.data?.status === 'syncing' ? 3000 : false),
    staleTime: 0,
  });
}

export function useObjectTree(deviceId: string) {
  return useQuery({
    queryKey: ['devices', 'object-tree', deviceId],
    queryFn: () => api.getObjectTree(deviceId),
    enabled: Boolean(deviceId),
  });
}

export function useDirectChildren(deviceId: string, pathPrefix: string, page: number, pageSize: number) {
  return useQuery({
    queryKey: ['devices', 'children', deviceId, pathPrefix, page, pageSize],
    queryFn: () => api.getDirectChildren(deviceId, pathPrefix, { page, pageSize }),
    enabled: Boolean(deviceId) && Boolean(pathPrefix),
  });
}

export function useParameterSchema(deviceId: string, pathPrefix?: string, enabled = true) {
  return useQuery({
    queryKey: ['devices', 'parameter-schema', deviceId, pathPrefix],
    queryFn: () => api.getParameterSchema(deviceId, pathPrefix),
    enabled: Boolean(deviceId) && enabled,
  });
}

export function useAddObject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceId,
      objectPath,
    }: {
      deviceId: string;
      objectPath: string;
    }) => api.addObject(deviceId, objectPath),
    onSuccess: (_result, { deviceId }) => {
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-tree', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameters', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-schema', deviceId],
      });
    },
  });
}

export function useDeleteObject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceId,
      objectPath,
    }: {
      deviceId: string;
      objectPath: string;
    }) => api.deleteObject(deviceId, objectPath),
    onSuccess: (_result, { deviceId }) => {
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-tree', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameters', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-schema', deviceId],
      });
    },
  });
}

export function useSyncConfigFile() {
  return useMutation({
    mutationFn: (deviceId: string) =>
      api.syncConfigFile ? api.syncConfigFile(deviceId) : Promise.resolve(undefined),
  });
}
