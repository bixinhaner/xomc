import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { SoftwareVersion, UpgradePlan, UpgradeTaskInfo, UpgradeSubTaskInfo } from '../../mock/data/software';
import type { PageRequest } from '../../types/pagination';
import { softwareService } from '../../mock/services/softwareService';
import { softwareApi } from '../../services/api/softwareApi';
import { createApiSwitch } from '../../services/apiSwitch';

const api = createApiSwitch(softwareService, softwareApi);

// ============================================================================
// Firmware hooks
// ============================================================================

export function useSoftwareVersions(
  params: { deviceType?: string; status?: string; vendor?: string; fileType?: number } & PageRequest
) {
  return useQuery({
    queryKey: ['software', 'versions', params],
    queryFn: () => api.getVersions(params),
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
        carrier: string;
        version: string;
        productClass?: string;
        releaseNotes?: string;
        fileType?: number;
        recommend?: boolean;
        uploader?: string;
        manufacturer?: string;
        description?: string;
      };
    }) => api.uploadFirmware(params.file, params.metadata),
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

// ============================================================================
// Upgrade Task hooks (main tasks)
// ============================================================================

export function useUpgradeTasks(
  params: { taskType?: number; status?: string; productClass?: string; createUser?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['software', 'upgrade-tasks', params],
    queryFn: () => api.getUpgradeTasks(params),
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
// Rollback hooks
// ============================================================================

export function useCreateRollback() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: {
      deviceIds: string[];
      taskName: string;
      operatorCode: string;
      createUser: string;
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
  });
}

export function useSubTaskById(id: string) {
  return useQuery({
    queryKey: ['software', 'sub-tasks', 'detail', id],
    queryFn: () => api.getSubTaskById(id),
    enabled: Boolean(id),
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
