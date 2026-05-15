import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { topologyService } from '../../mock/services/topologyService';
import { topologyApi } from '../../services/api/topologyApi';
import { useMock } from '../../services/apiSwitch';
import type { MapFilterParams, MapBounds, MapStats } from '../../types/map';
import type { SiteStatus, NodeType, NodeStatus, EdgeStatus } from '../../types/topology';

export function useDomains() {
  return useQuery({
    queryKey: ['topology', 'domains'],
    queryFn: () =>
      useMock ? topologyService.getDomains() : topologyApi.getDomains(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useDomainTree() {
  return useQuery({
    queryKey: ['topology', 'domain-tree'],
    queryFn: () =>
      useMock ? topologyService.getDomainTree() : topologyApi.getDomainTree(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useSites(params?: {
  domainId?: string;
  status?: SiteStatus;
  keyword?: string;
  page?: number;
  pageSize?: number;
}) {
  return useQuery({
    queryKey: ['topology', 'sites', params],
    queryFn: () =>
      useMock
        ? topologyService.getSites(params).then((items) => ({ items, total: items.length, page: 1, pageSize: items.length }))
        : topologyApi.getSites(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useSiteById(id: string) {
  return useQuery({
    queryKey: ['topology', 'sites', 'detail', id],
    queryFn: () =>
      useMock ? topologyService.getSiteById(id) : topologyApi.getSiteById(id),
    enabled: Boolean(id),
  });
}

export function useTopoNodes(params?: {
  domainId?: string;
  nodeType?: NodeType;
  status?: NodeStatus;
  page?: number;
  pageSize?: number;
}) {
  return useQuery({
    queryKey: ['topology', 'nodes', params],
    queryFn: () =>
      useMock
        ? topologyService.getTopoNodes(params).then((items) => ({ items, total: items.length, page: 1, pageSize: items.length }))
        : topologyApi.getTopoNodes(params),
    refetchInterval: 30000,
  });
}

export function useTopoEdges(params?: {
  status?: EdgeStatus;
  page?: number;
  pageSize?: number;
}) {
  return useQuery({
    queryKey: ['topology', 'edges', params],
    queryFn: () =>
      useMock
        ? topologyService.getTopoEdges().then((items) => ({ items, total: items.length, page: 1, pageSize: items.length }))
        : topologyApi.getTopoEdges(params),
    refetchInterval: 30000,
  });
}

export function useTopoGraph(params?: { domainId?: string; layoutType?: string; nodeType?: NodeType; status?: NodeStatus; limit?: number }) {
  return useQuery({
    queryKey: ['topology', 'graph', params],
    queryFn: () =>
      useMock
        ? topologyService.getTopoGraph()
        : topologyApi.getTopoGraph({ ...params, limit: params?.limit ?? 500 }),
    refetchInterval: 60000, // 降低到 60 秒，减少服务端压力
  });
}

export function useGeoData() {
  return useQuery({
    queryKey: ['topology', 'geo'],
    queryFn: () =>
      useMock ? topologyService.getGeoData() : topologyApi.getGeoData(),
    staleTime: 60 * 1000,
  });
}

// ── Group CRUD hooks ──

// Shape returned by topologyApi.createGroup / updateGroup (mirror of BackendDeviceGroup
// after http.ts camelCase transform). Kept local because the api file does not export it.
export interface DeviceGroupResult {
  id: string;
  name: string;
  parentId: string | null;
  carrier: string;
  description: string;
  sortOrder: number;
  children?: DeviceGroupResult[];
  createdAt: string;
  updatedAt: string;
}

function makeMockGroupResult(input: {
  name?: string;
  parent_id?: string;
  description?: string;
}): DeviceGroupResult {
  const now = new Date().toISOString();
  return {
    id: `mock-${Date.now()}`,
    name: input.name ?? '',
    parentId: input.parent_id ?? null,
    carrier: '',
    description: input.description ?? '',
    sortOrder: 0,
    createdAt: now,
    updatedAt: now,
  };
}

export function useCreateGroup() {
  const queryClient = useQueryClient();
  return useMutation<DeviceGroupResult, Error, { name: string; parent_id?: string; description?: string }>({
    mutationFn: (data) =>
      useMock
        ? topologyService.getDomains().then(() => makeMockGroupResult(data))
        : (topologyApi.createGroup(data) as unknown as Promise<DeviceGroupResult>),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domains'] });
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domain-tree'] });
    },
  });
}

export function useUpdateGroup() {
  const queryClient = useQueryClient();
  return useMutation<DeviceGroupResult, Error, { id: string; data: { name?: string; description?: string } }>({
    mutationFn: ({ id, data }) =>
      useMock
        ? topologyService.getDomains().then(() => makeMockGroupResult(data))
        : (topologyApi.updateGroup(id, data) as unknown as Promise<DeviceGroupResult>),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domains'] });
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domain-tree'] });
    },
  });
}

export function useDeleteGroup() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (id) =>
      useMock
        ? topologyService.getDomains().then(() => undefined)
        : topologyApi.deleteGroup(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domains'] });
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domain-tree'] });
    },
  });
}

export function useGroupDevices(groupId: string) {
  return useQuery({
    queryKey: ['topology', 'group-devices', groupId],
    queryFn: () => topologyApi.getGroupDevices(groupId),
    enabled: Boolean(groupId),
    staleTime: 5 * 60 * 1000,
  });
}

export function useAddDeviceToGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ groupId, deviceId }: { groupId: string; deviceId: string }) =>
      topologyApi.addDeviceToGroup(groupId, deviceId),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ['topology', 'group-devices', variables.groupId],
      });
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domains'] });
    },
  });
}

export function useRemoveDeviceFromGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ groupId, deviceId }: { groupId: string; deviceId: string }) =>
      topologyApi.removeDeviceFromGroup(groupId, deviceId),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ['topology', 'group-devices', variables.groupId],
      });
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domains'] });
    },
  });
}

// ============ Map Hooks ============

/**
 * 获取设备地理数据（支持筛选）
 */
export function useMapDevicesGeo(params: MapFilterParams) {
  return useQuery({
    queryKey: ['topology', 'map', 'geo', params],
    queryFn: () =>
      useMock
        ? Promise.resolve({ items: [], total: 0 }) // Mock 实现
        : topologyApi.getDevicesGeo(params),
    staleTime: 5 * 60 * 1000,
    enabled: params.enabled !== false,
  });
}

/**
 * 获取聚合数据（大范围视图）
 */
export function useMapAggregation(params: {
  bounds: MapBounds;
  zoom: number;
  filters?: MapFilterParams;
}) {
  return useQuery({
    queryKey: ['topology', 'map', 'aggregation', params],
    queryFn: () =>
      useMock
        ? Promise.resolve({ clusters: [] }) // Mock 实现
        : topologyApi.getAggregation(params),
    staleTime: 2 * 60 * 1000,
    enabled: params.zoom < 12, // 仅在缩放级别较小时请求
  });
}

/**
 * 获取地图统计数据
 */
export function useMapStats(params?: { groupIds?: string[]; bounds?: string }) {
  return useQuery<MapStats>({
    queryKey: ['topology', 'map', 'stats', params],
    queryFn: () =>
      useMock
        ? Promise.resolve<MapStats>({ total: 0, statusCount: { onlineActive: 0, onlineInactive: 0, offline: 0 }, alarmCount: 0 }) // Mock 实现
        : topologyApi.getMapStats(params),
    staleTime: 5 * 60 * 1000,
    refetchInterval: 60 * 1000, // 每分钟刷新
  });
}

/**
 * 搜索设备（节点查找）
 */
export function useMapDeviceSearch(keyword: string) {
  return useQuery({
    queryKey: ['topology', 'map', 'search', keyword],
    queryFn: () =>
      useMock
        ? Promise.resolve([]) // Mock 实现
        : topologyApi.searchDevices(keyword),
    staleTime: 30 * 1000,
    enabled: keyword.length >= 2,
  });
}
