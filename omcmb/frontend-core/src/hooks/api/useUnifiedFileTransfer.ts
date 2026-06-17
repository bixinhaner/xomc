import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import type { PageRequest } from '../../types/pagination';
import type {
  CreateUnifiedFileTransferTaskInput,
  CreateUnifiedFileTransferTypeInput,
  UpdateUnifiedFileTransferTaskTypeInput,
} from '../../types/unifiedFileTransfer';
import { unifiedFileTransferService } from '../../mock/services/unifiedFileTransferService';
import { createApiSwitch } from '../../services/apiSwitch';
import { unifiedFileTransferApi } from '../../services/api/unifiedFileTransferApi';

type QueryMountOptions = {
  refetchOnMount?: boolean | 'always';
};

const api = createApiSwitch(unifiedFileTransferService, unifiedFileTransferApi);

export function useUnifiedFileTransferOverview() {
  return useQuery({
    queryKey: ['ufte', 'overview'],
    queryFn: () => api.getOverview(),
    staleTime: 60_000,
  });
}

export function useUnifiedFileTransferTaskTypes(options?: QueryMountOptions) {
  return useQuery({
    queryKey: ['ufte', 'task-types'],
    queryFn: () => api.getTaskTypes(),
    staleTime: 60_000,
    refetchOnMount: options?.refetchOnMount,
  });
}

export function useUnifiedFileTransferTasks(
  params: { status?: string; typeCode?: string; keyword?: string; category?: string } & PageRequest,
) {
  return useQuery({
    queryKey: ['ufte', 'tasks', params],
    queryFn: () => api.getTasks(params),
    refetchInterval: 10_000,
  });
}

export function useUnifiedFileTransferDevices(
  params: { status?: string; typeCode?: string; keyword?: string; category?: string; productType?: string } & PageRequest,
) {
  return useQuery({
    queryKey: ['ufte', 'devices', params],
    queryFn: () => api.getDevices(params),
    refetchInterval: 10_000,
  });
}

export function useUnifiedFileTransferDeviceCandidates(
  params: { keyword?: string; category?: string; typeCode?: string; productType?: string; productName?: string } & PageRequest,
) {
  return useQuery({
    queryKey: ['ufte', 'device-candidates', params],
    queryFn: () => api.getDeviceCandidates(params),
    refetchInterval: 10_000,
  });
}

export function useCreateUnifiedFileTransferTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateUnifiedFileTransferTaskInput) => api.createTask(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useCreateUnifiedFileTransferTaskType() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateUnifiedFileTransferTypeInput) => api.createTaskType(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useUpdateUnifiedFileTransferTaskType() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateUnifiedFileTransferTaskTypeInput) => api.updateTaskType(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useDeleteUnifiedFileTransferTaskType() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (typeCode: string) => api.deleteTaskType(typeCode),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useStartUfteTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.startTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useSuspendUfteTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.suspendTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useTerminateUfteTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.terminateTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useDeleteUfteTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useBatchDeleteUfteTasks() {
  const queryClient = useQueryClient();
  return useMutation<
    { succeeded: string[]; failed: Array<{ taskId: string; error: string }> },
    Error,
    string[]
  >({
    mutationFn: (ids: string[]) => api.batchDeleteTasks(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useRetryUfteTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.retryTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}