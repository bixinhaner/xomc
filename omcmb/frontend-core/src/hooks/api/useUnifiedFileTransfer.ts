import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import type { PageRequest } from '../../types/pagination';
import type {
  CreateUnifiedFileTransferTaskInput,
  CreateUnifiedFileTransferTypeInput,
  UpdateUnifiedFileTransferTaskTypeInput,
} from '../../types/unifiedFileTransfer';
import { unifiedFileTransferApi } from '../../services/api/unifiedFileTransferApi';

export function useUnifiedFileTransferOverview() {
  return useQuery({
    queryKey: ['ufte', 'overview'],
    queryFn: () => unifiedFileTransferApi.getOverview(),
    staleTime: 60_000,
  });
}

export function useUnifiedFileTransferTaskTypes() {
  return useQuery({
    queryKey: ['ufte', 'task-types'],
    queryFn: () => unifiedFileTransferApi.getTaskTypes(),
    staleTime: 60_000,
  });
}

export function useUnifiedFileTransferTasks(
  params: { status?: string; typeCode?: string; keyword?: string; category?: string } & PageRequest,
) {
  return useQuery({
    queryKey: ['ufte', 'tasks', params],
    queryFn: () => unifiedFileTransferApi.getTasks(params),
    refetchInterval: 10_000,
  });
}

export function useUnifiedFileTransferDevices(
  params: { status?: string; typeCode?: string; keyword?: string; category?: string; productType?: string } & PageRequest,
) {
  return useQuery({
    queryKey: ['ufte', 'devices', params],
    queryFn: () => unifiedFileTransferApi.getDevices(params),
    refetchInterval: 10_000,
  });
}

export function useUnifiedFileTransferDeviceCandidates(
  params: { keyword?: string; category?: string; productType?: string } & PageRequest,
) {
  return useQuery({
    queryKey: ['ufte', 'device-candidates', params],
    queryFn: () => unifiedFileTransferApi.getDeviceCandidates(params),
    refetchInterval: 10_000,
  });
}

export function useCreateUnifiedFileTransferTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateUnifiedFileTransferTaskInput) => unifiedFileTransferApi.createTask(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useCreateUnifiedFileTransferTaskType() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateUnifiedFileTransferTypeInput) => unifiedFileTransferApi.createTaskType(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}

export function useUpdateUnifiedFileTransferTaskType() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateUnifiedFileTransferTaskTypeInput) => unifiedFileTransferApi.updateTaskType(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['ufte'] });
    },
  });
}