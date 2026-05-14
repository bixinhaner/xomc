import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import type { PageRequest } from '../../types/pagination';
import type {
  CreateUnifiedFileTransferTaskInput,
  CreateUnifiedFileTransferTypeInput,
  UpdateUnifiedFileTransferTaskTypeInput,
} from '../../types/unifiedFileTransfer';
import { useMock } from '../../services/apiSwitch';
import { unifiedFileTransferApi } from '../../services/api/unifiedFileTransferApi';
import { unifiedFileTransferService } from '../../mock/services/unifiedFileTransferService';

async function requestWithPreviewFallback<T>(
  realRequest: () => Promise<T>,
  mockRequest: () => Promise<T>,
): Promise<T> {
  if (useMock) {
    return mockRequest();
  }
  try {
    return await realRequest();
  } catch {
    return mockRequest();
  }
}

export function useUnifiedFileTransferOverview() {
  return useQuery({
    queryKey: ['ufte', 'overview'],
    queryFn: () =>
      requestWithPreviewFallback(
        () => unifiedFileTransferApi.getOverview(),
        () => unifiedFileTransferService.getOverview(),
      ),
    staleTime: 60_000,
  });
}

export function useUnifiedFileTransferTaskTypes() {
  return useQuery({
    queryKey: ['ufte', 'task-types'],
    queryFn: () =>
      requestWithPreviewFallback(
        () => unifiedFileTransferApi.getTaskTypes(),
        () => unifiedFileTransferService.getTaskTypes(),
      ),
    staleTime: 60_000,
  });
}

export function useUnifiedFileTransferTasks(
  params: { status?: string; typeCode?: string; keyword?: string; category?: string } & PageRequest,
) {
  return useQuery({
    queryKey: ['ufte', 'tasks', params],
    queryFn: () =>
      requestWithPreviewFallback(
        () => unifiedFileTransferApi.getTasks(params),
        () => unifiedFileTransferService.getTasks(params),
      ),
    refetchInterval: 10_000,
  });
}

export function useUnifiedFileTransferDevices(
  params: { status?: string; typeCode?: string; keyword?: string; category?: string } & PageRequest,
) {
  return useQuery({
    queryKey: ['ufte', 'devices', params],
    queryFn: () =>
      requestWithPreviewFallback(
        () => unifiedFileTransferApi.getDevices(params),
        () => unifiedFileTransferService.getDevices(params),
      ),
    refetchInterval: 10_000,
  });
}

export function useCreateUnifiedFileTransferTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateUnifiedFileTransferTaskInput) =>
      requestWithPreviewFallback(
        () => unifiedFileTransferApi.createTask(input),
        () => unifiedFileTransferService.createTask(input),
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useCreateUnifiedFileTransferTaskType() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateUnifiedFileTransferTypeInput) =>
      requestWithPreviewFallback(
        () => unifiedFileTransferApi.createTaskType(input),
        () => unifiedFileTransferService.createTaskType(input),
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useUpdateUnifiedFileTransferTaskType() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateUnifiedFileTransferTaskTypeInput) =>
      requestWithPreviewFallback(
        () => unifiedFileTransferApi.updateTaskType(input),
        () => unifiedFileTransferService.updateTaskType(input),
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}