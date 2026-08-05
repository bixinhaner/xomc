import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { SoftwareVersion, UpgradePlan } from '../../mock/data/software';
import type { PageRequest } from '../../types/pagination';
import { softwareService } from '../../mock/services/softwareService';
import { softwareApi } from '../../services/api/softwareApi';
import { createApiSwitch } from '../../services/apiSwitch';

const api = createApiSwitch(softwareService, softwareApi);

// ============================================================================
// Firmware hooks
// ============================================================================

export function useSoftwareVersions(
  params: { deviceType?: string; productId?: string; status?: string; vendor?: string; fileType?: number } & PageRequest,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: ['software', 'versions', params],
    queryFn: () => api.getVersions(params),
    enabled: options?.enabled ?? true,
  });
}

export function useSoftwareVersionById(id: string) {
  return useQuery({
    queryKey: ['software', 'versions', 'detail', id],
    queryFn: () => api.getVersionById(id),
    enabled: Boolean(id),
  });
}

export function useUploadSoftwareVersion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<SoftwareVersion, 'id' | 'releaseDate'>) =>
      api.uploadVersion(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] });
    },
  });
}

export function useUploadFirmware() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: {
      file: File;
      metadata: {
        version: string;
        productId?: string;
        // #638：多产品复选。传了它后端会同步主产品 product_id = ids[0]。
        productIds?: string[];
        productClass?: string;
        releaseNotes?: string;
        fileType?: number;
        recommend?: boolean;
        uploader?: string;
        manufacturer?: string;
        description?: string;
      };
      // #623：透传上传进度回调给调用方（驱动 Progress UI）。
      onProgress?: (percent: number) => void;
    }) => api.uploadFirmware(params.file, params.metadata, params.onProgress),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] });
    },
  });
}

export function useDeleteSoftwareVersions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.deleteVersions(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] });
    },
  });
}

export function useToggleRecommend() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.toggleRecommend(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] });
    },
  });
}

export function useDownloadFirmware() {
  return useMutation({
    mutationFn: (params: { id: string; fileName: string }) =>
      api.downloadFirmware(params.id, params.fileName),
  });
}

export function useUpdateFirmware() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: {
      id: string;
      metadata: {
        productId?: string;
        // #638：产品归属改为多选。传了 productIds 后端会重写主产品 product_id = ids[0]；未传（undefined）代表不动，传空数组代表清空。
        productIds?: string[];
        productClass?: string;
        version?: string;
        recommend?: boolean;
        description?: string;
        releaseNotes?: string;
      };
    }) => api.updateFirmware(params.id, params.metadata),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] });
    },
  });
}

// ============================================================================
// Upgrade Task hooks (main tasks)
// ============================================================================

const SUB_TASK_TERMINAL = new Set(['completed', 'failed', 'terminated']);

export function useUpgradeTasks(
  params: { taskType?: number; status?: string; productClass?: string; productId?: string; createUser?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['software', 'upgrade-tasks', params],
    queryFn: () => api.getUpgradeTasks(params),
    refetchInterval: (query) => {
      const items = query.state.data?.items;
      if (!items || items.length === 0) return false;
      const hasActive = items.some((t) => t.status !== 'ended');
      return hasActive ? 5000 : false;
    },
  });
}

export function useUpgradeTaskById(id: string) {
  return useQuery({
    queryKey: ['software', 'upgrade-tasks', 'detail', id],
    queryFn: () => api.getUpgradeTaskById(id),
    enabled: Boolean(id),
  });
}

export function useCreateUpgradeTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: {
      deviceIds: string[];
      firmwareId: string;
      taskName: string;
      taskType?: number;
      isKeepConfig?: boolean;
      concurrency?: number;
      createSuspended?: boolean;
    }) => api.createUpgradeTask(req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

export function useSuspendTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.suspendTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

export function useResumeTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.resumeTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

export function useTerminateTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.terminateTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

export function useDeleteTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

export function useRetryTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.retryTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

// ============================================================================
// Canary stage transitions (T-0019, mirrors backend T-0018)
// ============================================================================

export function useAdvanceCanary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.advanceCanary(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

export function usePauseCanary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.pauseCanary(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

export function useResumeCanary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.resumeCanary(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

export function useAbortCanary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.abortCanary(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

// ============================================================================
// Rollback hooks
// ============================================================================

export function useCreateRollback() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: {
      deviceIds: string[];
      taskName: string;
      createUser: string;
      createSuspended?: boolean;
    }) => api.createRollback(req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'upgrade-tasks'] });
    },
  });
}

// ============================================================================
// Sub-task hooks
// ============================================================================

export function useSubTasks(
  taskId: string,
  params: { status?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['software', 'upgrade-tasks', taskId, 'sub-tasks', params],
    queryFn: () => api.getSubTasks(taskId, params),
    enabled: Boolean(taskId),
    refetchInterval: (query) => {
      const items = query.state.data?.items;
      if (!items || items.length === 0) return false;
      const hasActive = items.some((t) => !SUB_TASK_TERMINAL.has(t.status));
      return hasActive ? 3000 : false;
    },
  });
}

export function useSubTaskById(id: string) {
  return useQuery({
    queryKey: ['software', 'sub-tasks', 'detail', id],
    queryFn: () => api.getSubTaskById(id),
    enabled: Boolean(id),
  });
}

export function useAllSubTasks(
  params: { taskName?: string; deviceSn?: string; status?: string; taskType?: number } & PageRequest
) {
  return useQuery({
    queryKey: ['software', 'all-sub-tasks', params],
    queryFn: () => api.getAllSubTasks(params),
    refetchInterval: (query) => {
      const items = query.state.data?.items;
      if (!items || items.length === 0) return false;
      const hasActive = items.some((t) => !SUB_TASK_TERMINAL.has(t.status));
      return hasActive ? 5000 : false;
    },
  });
}

// ============================================================================
// Legacy hooks (for backwards compatibility)
// ============================================================================

export function useUpgradePlans(params: { status?: string } & PageRequest) {
  return useQuery({
    queryKey: ['software', 'plans', params],
    queryFn: () => api.getUpgradePlans(params),
  });
}

export function useUpgradePlanById(id: string) {
  return useQuery({
    queryKey: ['software', 'plans', 'detail', id],
    queryFn: () => api.getUpgradePlanById(id),
    enabled: Boolean(id),
  });
}

export function useCreateUpgradePlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<UpgradePlan, 'id' | 'status' | 'progress' | 'successCount' | 'failCount' | 'createdAt' | 'updatedAt'>) =>
      api.createUpgradePlan(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'plans'] });
    },
  });
}

export function useCancelUpgradePlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.cancelUpgradePlan(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'plans'] });
    },
  });
}

export function useUpgradePrecheck() {
  return useMutation({
    mutationFn: ({ deviceSns, versionId }: { deviceSns: string[]; versionId: string }) =>
      api.precheck(deviceSns, versionId),
  });
}
