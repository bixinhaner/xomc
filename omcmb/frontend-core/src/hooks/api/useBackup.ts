import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { BackupTask, BackupSchedule, FTPConfig } from '../../mock/data/backup';
import type { PageRequest } from '../../types/pagination';
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
