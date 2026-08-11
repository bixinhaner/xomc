import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { PageRequest } from '../../types/pagination';
import {
  provisionApi,
  type CreateProvisioningTaskRequest,
  type SavePlugAndPlayPolicyRequest,
} from '../../services/api/provisionApi';

export function usePlugAndPlayPolicies(params: PageRequest & { productClass?: string; search?: string }) {
  return useQuery({
    queryKey: ['provisioning', 'policies', params],
    queryFn: () => provisionApi.getPolicies(params),
  });
}

export function usePlugAndPlayPolicy(id: string) {
  return useQuery({
    queryKey: ['provisioning', 'policy', id],
    queryFn: () => provisionApi.getPolicy(id),
    enabled: Boolean(id),
  });
}

export function useSavePlugAndPlayPolicy(id?: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: SavePlugAndPlayPolicyRequest) =>
      id ? provisionApi.updatePolicy(id, data) : provisionApi.createPolicy(data),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['provisioning', 'policies'] }),
  });
}

export function useSetPlugAndPlayPolicyEnabled() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => provisionApi.setPolicyEnabled(id, enabled),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['provisioning', 'policies'] }),
  });
}

export function useDeletePlugAndPlayPolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => provisionApi.deletePolicy(id),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['provisioning', 'policies'] }),
  });
}

export function useProvisioningTasks(
  params: { status?: string; deviceId?: string; policyOnly?: boolean } & PageRequest
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

export function useRetryPlugAndPlayTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { policyId: string; deviceId: string }) =>
      provisionApi.retryPolicyTask(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['provisioning'] });
    },
  });
}
