import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { storageProtectionApi, type StorageProtectionPolicyPayload } from '../../services/api/storageProtectionApi';
import { useMock } from '../../services/apiSwitch';

const emptyPolicies = () => Promise.resolve([]);

export function useStorageProtectionPolicies() {
  return useQuery({
    queryKey: ['system', 'storage-protection', 'policies'],
    queryFn: () => (useMock ? emptyPolicies() : storageProtectionApi.getPolicies()),
    refetchInterval: 30_000,
  });
}

export function useStorageProtectionTargets() {
  return useQuery({
    queryKey: ['system', 'storage-protection', 'targets'],
    queryFn: () => (useMock ? emptyPolicies() : storageProtectionApi.getTargets()),
    refetchInterval: 30_000,
  });
}

export function useStorageProtectionEvents(limit = 5) {
  return useQuery({
    queryKey: ['system', 'storage-protection', 'events', limit],
    queryFn: () => (useMock ? Promise.resolve([]) : storageProtectionApi.getEvents(limit)),
    refetchInterval: 30_000,
  });
}

export function useSaveStorageProtectionPolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: StorageProtectionPolicyPayload) => storageProtectionApi.savePolicy(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'storage-protection'] });
    },
  });
}

export function useUpdateStorageProtectionPolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: StorageProtectionPolicyPayload }) =>
      storageProtectionApi.updatePolicy(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'storage-protection'] });
    },
  });
}
