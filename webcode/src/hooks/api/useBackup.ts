import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { BackupTask, BackupSchedule, FTPConfig } from '@/mock/data/backup';
import type { PageRequest } from '@/types/pagination';
import { backupService } from '@/mock/services/backupService';

export function useBackupTasks(params: { status?: string; taskType?: string } & PageRequest) {
  return useQuery({
    queryKey: ['backup', 'tasks', params],
    queryFn: () => backupService.getTasks(params),
  });
}

export function useBackupTaskById(id: string) {
  return useQuery({
    queryKey: ['backup', 'tasks', 'detail', id],
    queryFn: () => backupService.getTaskById(id),
    enabled: Boolean(id),
  });
}

export function useCreateBackupTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<BackupTask, 'id' | 'createdAt' | 'updatedAt' | 'status' | 'progress' | 'successCount' | 'failCount'>) =>
      backupService.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'tasks'] });
    },
  });
}

export function useCancelBackupTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => backupService.cancelTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'tasks'] });
    },
  });
}

export function useDeleteBackupTasks() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => backupService.deleteTasks(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'tasks'] });
    },
  });
}

export function useBackupSchedules(params: PageRequest) {
  return useQuery({
    queryKey: ['backup', 'schedules', params],
    queryFn: () => backupService.getSchedules(params),
  });
}

export function useCreateBackupSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<BackupSchedule, 'id' | 'createTime'>) =>
      backupService.createSchedule(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'schedules'] });
    },
  });
}

export function useUpdateBackupSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<BackupSchedule> }) =>
      backupService.updateSchedule(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'schedules'] });
    },
  });
}

export function useDeleteBackupSchedules() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => backupService.deleteSchedules(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'schedules'] });
    },
  });
}

export function useFTPConfigs(params: PageRequest) {
  return useQuery({
    queryKey: ['backup', 'ftp', params],
    queryFn: () => backupService.getFTPConfigs(params),
  });
}

export function useCreateFTPConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<FTPConfig, 'id' | 'createTime'>) =>
      backupService.createFTPConfig(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'ftp'] });
    },
  });
}

export function useUpdateFTPConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<FTPConfig> }) =>
      backupService.updateFTPConfig(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'ftp'] });
    },
  });
}

export function useDeleteFTPConfigs() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => backupService.deleteFTPConfigs(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', 'ftp'] });
    },
  });
}

export function useTestFTPConnection() {
  return useMutation({
    mutationFn: (id: string) => backupService.testFTPConnection(id),
  });
}
