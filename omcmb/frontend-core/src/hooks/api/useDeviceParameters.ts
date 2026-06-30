import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { ParameterFilter, ParameterUpdateRequest } from '../../types/deviceParameter';
import type { PageRequest } from '../../types/pagination';
import { deviceParameterService } from '../../mock/services/deviceParameterService';
import { deviceParameterApi } from '../../services/api/deviceParameterApi';
import { createApiSwitchWithMock } from '../../services/apiSwitch';

type ParamApi = typeof deviceParameterApi & { syncConfigFile?: (deviceId: string) => Promise<unknown> };
const api = createApiSwitchWithMock<ParamApi>(deviceParameterService, deviceParameterApi);

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
      deviceParameterApi.invalidateParameterSchemaCache(deviceId);
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
    // 设备详情的参数树 / 快速设置会频繁在内部 tab 间切换；这里给 schema 查询一个短期缓存，
    // 避免每次切 tab 都把同一批 path_prefix 重新打满。真正需要最新值的场景仍通过显式 invalidate 触发。
    staleTime: 5 * 60 * 1000,
  });
}

// 把 objectPath 归一到 useParameterSchema 用的"父路径"前缀:
//  - AddObject 传入的本身就是父路径,例如 "DeviceGSM.Bts.1.Trx." —— 保持不变。
//  - DeleteObject 传入的是带实例号的路径,例如 "DeviceGSM.Bts.1.Trx.6." —— 剥掉末尾数字段还原为父路径。
// 这样 invalidateQueries 的 queryKey 能精准命中调用方表格的 ['devices','parameter-schema', deviceId, '<父路径>'],
// 不再波及同页其它表(BSC 邻区、Si2quater 等)。
function schemaParentPrefix(objectPath: string): string {
  const trimmed = objectPath.endsWith('.') ? objectPath.slice(0, -1) : objectPath;
  const lastDot = trimmed.lastIndexOf('.');
  if (lastDot < 0) return objectPath;
  const lastSeg = trimmed.slice(lastDot + 1);
  if (/^\d+$/.test(lastSeg)) {
    return `${trimmed.slice(0, lastDot)}.`;
  }
  return objectPath.endsWith('.') ? objectPath : `${objectPath}.`;
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
    onSuccess: (_result, { deviceId, objectPath }) => {
      deviceParameterApi.invalidateParameterSchemaCache(deviceId, schemaParentPrefix(objectPath));
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-tree', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameters', deviceId],
      });
      // 仅作废本次新增涉及的父路径 schema(例如 TRX),避免连带刷新整页所有表。
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-schema', deviceId, schemaParentPrefix(objectPath)],
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
    onSuccess: (_result, { deviceId, objectPath }) => {
      deviceParameterApi.invalidateParameterSchemaCache(deviceId, schemaParentPrefix(objectPath));
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-tree', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameters', deviceId],
      });
      // 仅作废本次删除涉及的父路径 schema(例如 TRX),避免连带刷新整页所有表。
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-schema', deviceId, schemaParentPrefix(objectPath)],
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
