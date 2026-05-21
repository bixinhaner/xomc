import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  deviceAbnormalRebootApi,
  type AbnormalRebootListParams,
} from '../../services/api/deviceAbnormalRebootApi';

// ============================================================================
// T-0158: React Query hooks for device-abnormal-reboots endpoints
// ============================================================================

export const abnormalRebootKeys = {
  all: ['device-abnormal-reboots'] as const,
  lists: () => [...abnormalRebootKeys.all, 'list'] as const,
  list: (params: AbnormalRebootListParams) =>
    [...abnormalRebootKeys.lists(), params] as const,
  detail: (id: string) => [...abnormalRebootKeys.all, 'detail', id] as const,
};

export function useAbnormalRebootList(
  params: AbnormalRebootListParams,
  enabled = true,
) {
  return useQuery({
    queryKey: abnormalRebootKeys.list(params),
    queryFn: () => deviceAbnormalRebootApi.list(params),
    enabled,
    staleTime: 15_000,
  });
}

export function useAbnormalReboot(id: string | undefined) {
  return useQuery({
    queryKey: id ? abnormalRebootKeys.detail(id) : abnormalRebootKeys.all,
    queryFn: () => deviceAbnormalRebootApi.getById(id as string),
    enabled: !!id,
  });
}

export function useDownloadAbnormalReboot() {
  return useMutation({
    mutationFn: (id: string) => deviceAbnormalRebootApi.getDownloadUrl(id),
    onSuccess: (url) => {
      window.open(url, '_blank');
    },
  });
}

export function useDeleteAbnormalReboot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deviceAbnormalRebootApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: abnormalRebootKeys.all });
    },
  });
}

// 批量删除：循环调用单删除，保持 API 简单（后端目前无 batch 端点）。
// 任一失败仍继续，最终统一刷新列表；调用方据返回值给用户反馈。
export function useBatchDeleteAbnormalReboot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (ids: string[]) => {
      const results = await Promise.allSettled(
        ids.map((id) => deviceAbnormalRebootApi.delete(id)),
      );
      return {
        success: results.filter((r) => r.status === 'fulfilled').length,
        failed: results.filter((r) => r.status === 'rejected').length,
      };
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: abnormalRebootKeys.all });
    },
  });
}
