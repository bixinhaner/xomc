import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { ManagedFile, FileType, FileStatus } from '@/mock/data/fileManagement';
import type { PageRequest } from '@/types/pagination';
import { fileService } from '@/mock/services/fileService';
import { fileApi } from '@/services/api/fileApi';
import { useMock } from '@/services/apiSwitch';

export function useFileList(
  params: { fileType?: FileType; status?: FileStatus; keyword?: string; deviceSn?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['files', 'list', params],
    queryFn: () =>
      useMock ? fileService.getList(params) : fileApi.getList(params),
  });
}

export function useFileById(id: string) {
  return useQuery({
    queryKey: ['files', 'detail', id],
    queryFn: () =>
      useMock ? fileService.getById(id) : fileApi.getById(id),
    enabled: Boolean(id),
  });
}

export function useStorageStats() {
  return useQuery({
    queryKey: ['files', 'storage-stats'],
    queryFn: () =>
      useMock ? fileService.getStorageStats() : fileApi.getStorageStats(),
    staleTime: 60000,
  });
}

export function useUploadFile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<ManagedFile, 'id' | 'uploadTime'>) =>
      useMock ? fileService.upload(data) : fileApi.upload(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['files'] });
    },
  });
}

export function useDeleteFiles() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? fileService.delete(ids) : fileApi.delete(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['files'] });
    },
  });
}

export function useDownloadFile() {
  return useMutation({
    mutationFn: (id: string) =>
      useMock ? fileService.download(id) : fileApi.download(id),
  });
}

export function useDistributeFile() {
  return useMutation({
    mutationFn: ({ fileId, deviceSns }: { fileId: string; deviceSns: string[] }) =>
      useMock ? fileService.distribute(fileId, deviceSns) : fileApi.distribute(fileId, deviceSns),
  });
}
