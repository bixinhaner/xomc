import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { stationLogApi, type StationLogListParams } from '../../services/api/stationLogApi';

// ============================================================================
// Query keys
// ============================================================================

export const stationLogKeys = {
  all: ['station-logs'] as const,
  lists: () => [...stationLogKeys.all, 'list'] as const,
  list: (params: StationLogListParams) => [...stationLogKeys.lists(), params] as const,
};

// ============================================================================
// Hooks
// ============================================================================

/**
 * 查询日志文件列表
 */
export function useStationLogList(params: StationLogListParams, enabled = true) {
  return useQuery({
    queryKey: stationLogKeys.list(params),
    queryFn: () => stationLogApi.list(params),
    enabled,
    staleTime: 30_000,
  });
}

/**
 * 获取指定设备最新的运行日志（page=1, size=1）
 * enabled 应在有 deviceId 时才为 true
 */
export function useLatestRunningLog(deviceId: string | undefined) {
  return useQuery({
    queryKey: stationLogKeys.list({ deviceId, logType: 'running', page: 1, pageSize: 1 }),
    queryFn: () => stationLogApi.list({ deviceId, logType: 'running', page: 1, pageSize: 1 }),
    enabled: !!deviceId,
    staleTime: 60_000,
    select: (data) => data.items[0] ?? null,
  });
}

/**
 * 获取下载 URL 并在新标签页打开
 */
export function useDownloadStationLog() {
  return useMutation({
    mutationFn: (id: string) => stationLogApi.getDownloadUrl(id),
    onSuccess: (url) => {
      window.open(url, '_blank');
    },
    onError: () => {
      console.error('获取下载链接失败');
    },
  });
}

/**
 * 删除日志文件
 */
export function useDeleteStationLog() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => stationLogApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: stationLogKeys.all });
    },
    onError: () => {
      console.error('删除日志文件失败');
    },
  });
}
