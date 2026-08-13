import { useQuery, useQueries, useMutation, useQueryClient } from '@tanstack/react-query';
import type { Device, DeviceFilter, DeviceListResponse, NameFilterItem } from '../../types/device';
import type { PageRequest } from '../../types/pagination';
import { deviceService } from '../../mock/services/deviceService';
import { deviceApi } from '../../services/api/deviceApi';
import { quicksettingsApi } from '../../services/api/quicksettingsApi';
import { createApiSwitchWithMock } from '../../services/apiSwitch';
import { notificationKeys } from './useNotificationCenter';

// 避免在多工作区/多 node_modules 情况下因 QueryClient 私有字段导致名义类型不兼容。
// 这里使用结构化接口，仅声明预取逻辑实际依赖的方法。
type QueryClientLike = {
  getQueriesData: <TData = unknown>(filters: { queryKey: unknown[] }) => Array<[unknown, TData | undefined]>;
  setQueryData: <TData = unknown>(queryKey: unknown[], updater: TData) => void;
  prefetchQuery: <TData = unknown>(options: {
    queryKey: unknown[];
    queryFn: () => Promise<TData>;
    staleTime?: number;
  }) => Promise<unknown>;
};

const api = createApiSwitchWithMock(deviceService, deviceApi);
const DEVICE_DETAIL_STALE_TIME_MS = 10 * 60 * 1000;

/**
 * 按当前前端 API 环境查询设备列表。
 *
 * 供需要在用户操作中即时校验设备的 UI 复用，保证真实 API 与 mock API
 * 使用同一套筛选语义。
 */
export function fetchDeviceList(params: DeviceFilter & PageRequest) {
  return api.getList(params);
}

function findDeviceInListCaches(queryClient: QueryClientLike, sn: string): Device | undefined {
  const cachedLists = queryClient.getQueriesData<DeviceListResponse>({
    queryKey: ['devices', 'list'],
  });

  for (const [, data] of cachedLists) {
    const device = data?.items.find((item) => item.sn === sn);
    if (device) return device;
  }

  return undefined;
}

function seedDeviceCaches(queryClient: QueryClientLike, device: Device) {
  queryClient.setQueryData(['devices', 'sn', device.sn], device);
  queryClient.setQueryData(['devices', 'detail', device.id], device);
}

export function prefetchDeviceDetailContext(queryClient: QueryClientLike, device: Device) {
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
  matching_mode?: 'deviceName' | 'lac' | 'tac' | 'serialNumber' | '';
  source_group_id?: string;
  name_rule_list?: NameFilterItem[];
  lac_list?: number[];
  tac_list?: number[];
  serial_number_list?: string[];
}

// 更新分组的请求类型
export interface UpdateGroupRequest {
  name?: string;
  parent_id?: string; // 修改父级分组（L1 转 L2 或 L2 转 L1）
  remark?: string;
  matching_mode?: 'deviceName' | 'lac' | 'tac' | 'serialNumber' | '';
  source_group_id?: string;
  name_rule_list?: NameFilterItem[];
  lac_list?: number[];
  tac_list?: number[];
  serial_number_list?: string[];
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
    queryFn: () => fetchDeviceList(params),
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

export function useDeviceControlActions(id: string, page = 1, pageSize = 20, enabled = true) {
  return useQuery({
    queryKey: ['devices', 'control-actions', id, page, pageSize],
    queryFn: () => api.getControlActions(id, page, pageSize),
    enabled: Boolean(id) && enabled,
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

type DeviceGroupCacheInvalidator = Pick<ReturnType<typeof useQueryClient>, 'invalidateQueries'>;

export async function invalidateDeviceGroupCaches(queryClient: DeviceGroupCacheInvalidator) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ['devices', 'groups'] }),
    queryClient.invalidateQueries({ queryKey: ['system', 'deviceGroups', 'all'] }),
  ]);
}

export function useCreateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateGroupRequest) => api.createGroup(data),
    onSuccess: () => invalidateDeviceGroupCaches(queryClient),
  });
}

export function useUpdateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateGroupRequest }) =>
      api.updateGroup(id, data),
    onSuccess: async () => {
      await Promise.all([
        invalidateDeviceGroupCaches(queryClient),
        // 分组「设备匹配规则」改动后后端会异步重算归属，必须同时失效设备列表
        // 缓存，否则用户在 DeviceGrouping 页面里看到的还是旧的归属结果。
        // （仅改名也 invalidate 一次代价可忽略——分组更新本身就是低频操作。）
        queryClient.invalidateQueries({ queryKey: ['devices', 'list'] }),
      ]);
    },
  });
}

export function useDeleteGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteGroup(id),
    onSuccess: () => invalidateDeviceGroupCaches(queryClient),
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
    mutationFn: ({ id, data, fallbackDevice }: { id: string; data: Parameters<typeof api.update>[1]; fallbackDevice?: Partial<Device> }) =>
      api.update(id, data, fallbackDevice),
    onSuccess: (result, { id, fallbackDevice }) => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'detail', id] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'detail-composite-v2', id] });
      const sn = result.sn || fallbackDevice?.sn;
      if (sn) {
        void queryClient.invalidateQueries({ queryKey: ['devices', 'sn', sn] });
      }
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

// 手动触发 durable paramsync 参数同步（reason="manual"）。
// sync-params 是兼容 URL；后端运行时通过 paramSyncStarter 提交 paramsync request/run。
export function useSyncDeviceParams() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceId, force, parameterPaths }: { deviceId: string; force?: boolean; parameterPaths?: string[] }) =>
      api.syncDeviceParams(deviceId, force !== undefined || parameterPaths !== undefined ? { force, parameterPaths } : undefined),
    onSuccess: (_data, variables) => {
      // 失效设备参数缓存，让 paramsync 完成后展示新值。
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
    refetchOnMount: 'always',
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
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'groups'] });
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

// 网管侧手动改基站名（即时下发）
// 成功后 invalidate 设备详情缓存，让列表/详情刷新新名称
export function useRenameDevice(deviceId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => deviceApi.renameDevice(deviceId, name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', deviceId] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    },
  });
}
