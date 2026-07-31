import { useQuery } from '@tanstack/react-query';
import {
  rebootRecordApi,
  type RebootRecordListParams,
  type RebootRecordStatParams,
} from '../../services/api/rebootRecordApi';

export const rebootRecordKeys = {
  all: ['reboot-records'] as const,
  lists: () => [...rebootRecordKeys.all, 'list'] as const,
  list: (params: RebootRecordListParams) => [...rebootRecordKeys.lists(), params] as const,
  stats: (params: RebootRecordStatParams) => [...rebootRecordKeys.all, 'stats', params] as const,
};

export interface UseRebootRecordListOptions {
  enabled?: boolean;
  refetchInterval?: number | false;
}

export function useRebootRecordList(
  params: RebootRecordListParams,
  options: boolean | UseRebootRecordListOptions = true,
) {
  const resolvedOptions =
    typeof options === 'boolean' ? { enabled: options } : { enabled: true, ...options };

  return useQuery({
    queryKey: rebootRecordKeys.list(params),
    queryFn: () => rebootRecordApi.list(params),
    enabled: resolvedOptions.enabled,
    refetchInterval: resolvedOptions.refetchInterval,
    staleTime: 15_000,
  });
}

// 按设备聚合统计（总次数/异常次数，跟随筛选；通常仅统计弹窗打开时 enabled）
export function useRebootRecordStatByDevice(params: RebootRecordStatParams, enabled = true) {
  return useQuery({
    queryKey: rebootRecordKeys.stats(params),
    queryFn: () => rebootRecordApi.statByDevice(params),
    enabled,
    staleTime: 15_000,
  });
}
