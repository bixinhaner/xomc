import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { PageRequest } from '../../types/pagination';
import {
  provisionApi,
  type CreateProvisioningTaskRequest,
} from '../../services/api/provisionApi';

export function useProvisioningTasks(
  params: { status?: string; deviceId?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['provisioning', 'tasks', params],
    queryFn: () => provisionApi.getTasks(params),
    refetchInterval: 10 * 1000, // auto-refresh every 10s for live status
  });
}

export function useProvisioningTask(id: string) {
  return useQuery({
    queryKey: ['provisioning', 'task', id],
    queryFn: () => provisionApi.getTask(id),
    enabled: Boolean(id),
    refetchInterval: 5 * 1000,
  });
}

export function useCreateProvisioningTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateProvisioningTaskRequest) => provisionApi.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['provisioning'] });
    },
  });
}

export function useRetryProvisioningTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => provisionApi.retryTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['provisioning'] });
    },
  });
}
