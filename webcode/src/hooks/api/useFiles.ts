import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { ManagedFile, FileType, FileStatus } from '@/mock/data/fileManagement';
import type { PageRequest } from '@/types/pagination';
import { fileService } from '@/mock/services/fileService';

export function useFileList(
  params: { fileType?: FileType; status?: FileStatus; keyword?: string; deviceSn?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['files', 'list', params],
    queryFn: () => fileService.getList(params),
  });
}

export function useFileById(id: string) {
  return useQuery({
    queryKey: ['files', 'detail', id],
    queryFn: () => fileService.getById(id),
    enabled: Boolean(id),
  });
}

export function useStorageStats() {
  return useQuery({
    queryKey: ['files', 'storage-stats'],
    queryFn: () => fileService.getStorageStats(),
    staleTime: 60000,
  });
}

export function useUploadFile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<ManagedFile, 'id' | 'uploadTime'>) =>
      fileService.upload(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['files'] });
    },
  });
}

export function useDeleteFiles() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => fileService.delete(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['files'] });
    },
  });
}

export function useDownloadFile() {
  return useMutation({
    mutationFn: (id: string) => fileService.download(id),
  });
}

export function useDistributeFile() {
  return useMutation({
    mutationFn: ({ fileId, deviceSns }: { fileId: string; deviceSns: string[] }) =>
      fileService.distribute(fileId, deviceSns),
  });
}
