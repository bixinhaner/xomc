import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { BackupTask, BackupSchedule, FTPConfig, BackupPolicy, RestoreTask, RestoreStatus } from '../../mock/data/backup';
import { DEFAULT_BACKUP_POLICY, mockRestoreTasks } from '../../mock/data/backup';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { backupService } from '../../mock/services/backupService';
import { backupApi } from '../../services/api/backupApi';
import { useMock } from '../../services/apiSwitch';

export function useBackupTasks(params: { status?: string; taskType?: string } & PageRequest) {
  return useQuery({
    queryKey: ['backup', 'tasks', params],
    queryFn: () =>
      useMock ? backupService.getTasks(params) : backupApi.getTasks(params),
  });
}

export function useBackupTaskById(id: string) {
  return useQuery({
    queryKey: ['backup', 'tasks', 'detail', id],
    queryFn: () =>
      useMock ? backupService.getTaskById(id) : backupApi.getTaskById(id),
    enabled: Boolean(id),
  });
}

export function useCreateBackupTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<BackupTask, 'id' | 'createdAt' | 'updatedAt' | 'status' | 'progress' | 'successCount' | 'failCount'>) =>
      useMock ? backupService.createTask(data) : backupApi.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'tasks'] });
    },
  });
}

export function useCancelBackupTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      useMock ? backupService.cancelTask(id) : backupApi.cancelTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'tasks'] });
    },
  });
}

export function useDeleteBackupTasks() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? backupService.deleteTasks(ids) : backupApi.deleteTasks(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'tasks'] });
    },
  });
}

export function useBackupSchedules(params: PageRequest) {
  return useQuery({
    queryKey: ['backup', 'schedules', params],
    queryFn: () =>
      useMock ? backupService.getSchedules(params) : backupApi.getSchedules(params),
  });
}

export function useCreateBackupSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<BackupSchedule, 'id' | 'createTime'>) =>
      useMock ? backupService.createSchedule(data) : backupApi.createSchedule(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'schedules'] });
    },
  });
}

export function useUpdateBackupSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<BackupSchedule> }) =>
      useMock ? backupService.updateSchedule(id, data) : backupApi.updateSchedule(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'schedules'] });
    },
  });
}

export function useDeleteBackupSchedules() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? backupService.deleteSchedules(ids) : backupApi.deleteSchedules(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'schedules'] });
    },
  });
}

// ---------------------------------------------------------------------------
// FTP Config hooks — with useMock switch for backend/mock toggle.
// ---------------------------------------------------------------------------

export function useFTPConfigs(params: PageRequest) {
  return useQuery({
    queryKey: ['backup', 'ftp', params],
    queryFn: () =>
      useMock ? backupService.getFTPConfigs(params) : backupApi.getFTPConfigs(params),
  });
}

export function useCreateFTPConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<FTPConfig, 'id' | 'createTime'>) =>
      useMock ? backupService.createFTPConfig(data) : backupApi.createFTPConfig(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'ftp'] });
    },
  });
}

export function useUpdateFTPConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<FTPConfig> }) =>
      useMock ? backupService.updateFTPConfig(id, data) : backupApi.updateFTPConfig(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'ftp'] });
    },
  });
}

export function useDeleteFTPConfigs() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? backupService.deleteFTPConfigs(ids) : backupApi.deleteFTPConfigs(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'ftp'] });
    },
  });
}

export function useTestFTPConnection() {
  return useMutation({
    mutationFn: (id: string) =>
      useMock ? backupService.testFTPConnection(id) : backupApi.testFTPConnection(id),
  });
}

// ---------------------------------------------------------------------------
// Backup Policy hooks (T-0071, singleton)
//
// Mock mode returns DEFAULT_BACKUP_POLICY synchronously (no backupService stub
// since policy isn't exercised in unit tests yet); real mode goes through
// backupApi which gracefully falls back to defaults on 404. Mutation invalidates
// the query so the form refreshes after a successful PUT.
// ---------------------------------------------------------------------------

export function useBackupPolicy() {
  return useQuery<BackupPolicy>({
    queryKey: ['backup', 'policy'],
    queryFn: () => (useMock ? Promise.resolve({ ...DEFAULT_BACKUP_POLICY }) : backupApi.getPolicy()),
  });
}

export function useUpdateBackupPolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: BackupPolicy) =>
      useMock ? Promise.resolve(data) : backupApi.updatePolicy(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'policy'] });
    },
  });
}

// ---------------------------------------------------------------------------
// T-0078 Restore: queries + mutation. The list query polls every 5s so
// pending/running tasks update in place while the user watches; once T-0079
// fans per-device progress back, this can refine to "poll only when
// in-flight" but the cost-benefit at MVP scale is negligible.
// ---------------------------------------------------------------------------

interface MockRestoreListParams extends PageRequest {
  status?: RestoreStatus;
}

function mockRestoreList(
  params: MockRestoreListParams
): Promise<PageResponse<RestoreTask>> {
  let items = [...mockRestoreTasks];
  if (params.status) {
    items = items.filter((t) => t.status === params.status);
  }
  const total = items.length;
  const page = params.page ?? 1;
  const pageSize = params.pageSize ?? 20;
  const start = (page - 1) * pageSize;
  const sliced = items.slice(start, start + pageSize);
  return Promise.resolve({
    items: sliced,
    total,
    page,
    pageSize,
    totalPages: Math.max(1, Math.ceil(total / pageSize)),
  });
}

export function useBackupRestoreTasks(
  params: PageRequest & { status?: RestoreStatus }
) {
  return useQuery<PageResponse<RestoreTask>>({
    queryKey: ['backup', 'restore-tasks', params],
    queryFn: () =>
      useMock ? mockRestoreList(params) : backupApi.listRestoreTasks(params),
    refetchInterval: 5000,
  });
}

export function useBackupRestoreTask(id: string | undefined) {
  return useQuery<RestoreTask>({
    queryKey: ['backup', 'restore-tasks', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.reject(new Error('id required'));
      if (useMock) {
        const t = mockRestoreTasks.find((x) => x.id === id);
        return t
          ? Promise.resolve({ ...t })
          : Promise.reject(new Error('not found'));
      }
      return backupApi.getRestoreTask(id);
    },
    enabled: Boolean(id),
  });
}

export function useCreateBackupRestore() {
  const queryClient = useQueryClient();
  return useMutation<
    RestoreTask,
    Error,
    { bucket: string; objectPath: string; targetDeviceSns: string[] }
  >({
    mutationFn: (req) => {
      if (useMock) {
        const now = new Date().toISOString();
        return Promise.resolve<RestoreTask>({
          id: `rt-mock-${Date.now()}`,
          sourceBucket: req.bucket,
          sourceObjectPath: req.objectPath,
          targetDeviceSns: req.targetDeviceSns,
          status: 'pending',
          progress: 0,
          createdAt: now,
          updatedAt: now,
        });
      }
      return backupApi.createRestore(req);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['backup', 'restore-tasks'],
      });
    },
  });
}
