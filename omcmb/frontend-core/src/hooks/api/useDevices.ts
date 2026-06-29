import { useQuery, useQueries, useMutation, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import type { Device, DeviceFilter, DeviceListResponse, NameFilterItem } from '../../types/device';
import type { PageRequest } from '../../types/pagination';
import { deviceService } from '../../mock/services/deviceService';
import { deviceApi } from '../../services/api/deviceApi';
import { quicksettingsApi } from '../../services/api/quicksettingsApi';
import { createApiSwitchWithMock } from '../../services/apiSwitch';

const api = createApiSwitchWithMock(deviceService, deviceApi);
const DEVICE_DETAIL_STALE_TIME_MS = 10 * 60 * 1000;

function findDeviceInListCaches(queryClient: QueryClient, sn: string): Device | undefined {
  const cachedLists = queryClient.getQueriesData<DeviceListResponse>({
    queryKey: ['devices', 'list'],
  });

  for (const [, data] of cachedLists) {
    const device = data?.items.find((item) => item.sn === sn);
    if (device) return device;
  }

  return undefined;
}

function seedDeviceCaches(queryClient: QueryClient, device: Device) {
  queryClient.setQueryData(['devices', 'sn', device.sn], device);
  queryClient.setQueryData(['devices', 'detail', device.id], device);
}

export function prefetchDeviceDetailContext(queryClient: QueryClient, device: Device) {
  seedDeviceCaches(queryClient, device);

  return Promise.allSettled([
    queryClient.prefetchQuery({
      queryKey: ['devices', 'sn', device.sn],
      queryFn: () => api.getBySn(device.sn),
      staleTime: DEVICE_DETAIL_STALE_TIME_MS,
    }),
    queryClient.prefetchQuery({
      queryKey: ['quicksettings', 'groups', device.id],
      queryFn: () => quicksettingsApi.getGroups(device.id),
      staleTime: DEVICE_DETAIL_STALE_TIME_MS,
    }),
  ]);
}

// 创建分组的请求类型
export interface CreateGroupRequest {
  name: string;
  parent_id?: string;
  remark?: string;
  matching_mode?: 'deviceName' | 'lac' | 'tac';
  name_rule_list?: NameFilterItem[];
  lac_list?: number[];
  tac_list?: number[];
}

// 更新分组的请求类型
export interface UpdateGroupRequest {
  name?: string;
  parent_id?: string; // 修改父级分组（L1 转 L2 或 L2 转 L1）
  remark?: string;
  matching_mode?: 'deviceName' | 'lac' | 'tac';
  name_rule_list?: NameFilterItem[];
  lac_list?: number[];
  tac_list?: number[];
}

export interface UseDeviceListOptions {
  /** 自动刷新间隔（毫秒），不传或 0 表示不自动刷新 */
  refetchInterval?: number;
  /** 是否启用 query；false 时不发请求（AutoComplete 异步搜索场景：用户未输入时跳过） */
  enabled?: boolean;
}

export function useDeviceList(
  params: DeviceFilter & PageRequest,
  options?: UseDeviceListOptions
) {
  return useQuery({
    queryKey: ['devices', 'list', params],
    queryFn: () => api.getList(params),
    refetchInterval: options?.refetchInterval || false,
    enabled: options?.enabled ?? true,
  });
}

export function useDeviceById(id: string) {
  return useQuery({
    queryKey: ['devices', 'detail', id],
    queryFn: () => api.getById(id),
    enabled: Boolean(id),
  });
}

export function useDevicesByIds(ids: string[]) {
  const uniqueIds = Array.from(new Set(ids.filter(Boolean)));

  return useQueries({
    queries: uniqueIds.map((id) => ({
      queryKey: ['devices', 'detail', id],
      queryFn: () => api.getById(id),
      enabled: Boolean(id),
    })),
  });
}

export function useDeviceBySn(sn: string) {
  const queryClient = useQueryClient();
  const cachedDevice = sn ? findDeviceInListCaches(queryClient, sn) : undefined;

  return useQuery({
    queryKey: ['devices', 'sn', sn],
    queryFn: () => api.getBySn(sn),
    enabled: Boolean(sn),
    initialData: cachedDevice,
    initialDataUpdatedAt: cachedDevice ? Date.now() : undefined,
    staleTime: DEVICE_DETAIL_STALE_TIME_MS,
  });
}

export function useDeviceGroups() {
  return useQuery({
    queryKey: ['devices', 'groups'],
    queryFn: () => api.getGroups(),
    staleTime: 5 * 60 * 1000,
  });
}

function invalidateDeviceGroupCaches(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: ['devices', 'groups'] });
  void queryClient.invalidateQueries({ queryKey: ['system', 'deviceGroups', 'all'] });
}

export function useCreateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateGroupRequest) => api.createGroup(data),
    onSuccess: () => {
      invalidateDeviceGroupCaches(queryClient);
    },
  });
}

export function useUpdateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateGroupRequest }) =>
      api.updateGroup(id, data),
    onSuccess: () => {
      invalidateDeviceGroupCaches(queryClient);
      // 分组「设备匹配规则」改动后后端会异步重算归属，必须同时失效设备列表
      // 缓存，否则用户在 DeviceGrouping 页面里看到的还是旧的归属结果。
      // （仅改名也 invalidate 一次代价可忽略——分组更新本身就是低频操作。）
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

export function useDeleteGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteGroup(id),
    onSuccess: () => {
      invalidateDeviceGroupCaches(queryClient);
    },
  });
}

export function useMoveDevices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { device_ids: string[]; target_group_id: string }) =>
      api.moveDevices(params),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'groups'] });
    },
  });
}

export function useAddDevicesToGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ groupId, deviceIds }: { groupId: string; deviceIds: string[] }) =>
      api.addDevicesToGroup(groupId, deviceIds),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'groups'] });
    },
  });
}

export function useCreateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof api.create>[0]) =>
      api.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices'] });
    },
  });
}

export function useUpdateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof api.update>[1] }) =>
      api.update(id, data),
    onSuccess: (_result, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'detail', id] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

export function useDeleteDevices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.delete(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'groups'] });
    },
  });
}

export function useNEList(params: { keyword?: string } & PageRequest) {
  return useQuery({
    queryKey: ['devices', 'ne-list', params],
    queryFn: () => api.getNEList(params),
  });
}

export function useNEBySn(sn: string) {
  return useQuery({
    queryKey: ['devices', 'ne-sn', sn],
    queryFn: () => api.getNEBySn(sn),
    enabled: Boolean(sn),
  });
}

export function useRebootDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.reboot(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

// T-0126: 手动触发 Path B 全量参数同步（reason="manual"）。
// 后端走 Path B 完整链路：reason 通道 + 差异日志 + last_param_sync_at 回写 + Translator 翻译。
// 替代旧 useSyncParameters（Path A 已下线）。
export function useSyncDeviceParams() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceId, force, parameterPaths }: { deviceId: string; force?: boolean; parameterPaths?: string[] }) =>
      api.syncDeviceParams(deviceId, force !== undefined || parameterPaths !== undefined ? { force, parameterPaths } : undefined),
    onSuccess: (_data, variables) => {
      // 失效设备参数缓存让前端在 Path B 完成后展示新值
      void queryClient.invalidateQueries({ queryKey: ['device-parameters', variables.deviceId] });
    },
  });
}

export function useBatchRebootDevices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (ids: string[]) => {
      const results = await Promise.allSettled(ids.map((id) => api.reboot(id)));
      const failed = results.filter((r) => r.status === 'rejected');
      if (failed.length > 0) {
        throw new Error(`${failed.length}/${ids.length} devices failed to reboot`);
      }
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

// ========== Recycle Bin Hooks ==========

export interface RecycleBinFilter {
  search?: string;
  carrier?: string;
  technology?: string;
  group_id?: string;
  deleted_by?: string;
  page?: number;
  pageSize?: number;
  sortField?: string;
  sortOrder?: string;
}

export function useRecycleBinList(params: RecycleBinFilter) {
  return useQuery({
    queryKey: ['devices', 'recycle-bin', params],
    queryFn: () => deviceApi.listRecycleBin(params),
  });
}

export function useRestoreDevices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => deviceApi.restoreDevices(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'recycle-bin'] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

export function usePermanentDeleteDevices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => deviceApi.permanentDeleteDevices(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'recycle-bin'] });
    },
  });
}

export function useProductClasses() {
  return useQuery({
    queryKey: ['devices', 'product-classes'],
    queryFn: () => deviceApi.getProductClasses(),
    staleTime: 5 * 60 * 1000,
  });
}
