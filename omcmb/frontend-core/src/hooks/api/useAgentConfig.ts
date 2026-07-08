import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { agentApi } from '../../services/api/agentApi';
import type { AgentAdminConfigUpdate } from '../../types/agentConfig';

const runtimeKey = ['agent', 'runtime-config'] as const;
const visibilityKey = ['agent', 'visibility-config'] as const;
const adminKey = ['agent', 'admin-config'] as const;

export function useAgentVisibilityConfig(enabled = true) {
  return useQuery({
    queryKey: visibilityKey,
    queryFn: () => agentApi.getVisibilityConfig(),
    enabled,
    staleTime: 30_000,
    retry: 1,
  });
}

export function useAgentRuntimeConfig(enabled = true) {
  return useQuery({
    queryKey: runtimeKey,
    queryFn: () => agentApi.getRuntimeConfig(),
    enabled,
    staleTime: 30_000,
    retry: 1,
  });
}

export function useAdminAgentConfig() {
  return useQuery({
    queryKey: adminKey,
    queryFn: () => agentApi.getAdminConfig(),
    staleTime: 30_000,
  });
}

export function useSaveAdminAgentConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: AgentAdminConfigUpdate) => agentApi.saveAdminConfig(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminKey });
      void queryClient.invalidateQueries({ queryKey: visibilityKey });
      void queryClient.invalidateQueries({ queryKey: runtimeKey });
    },
  });
}

export function useTestAdminAgentConfig() {
  return useMutation({
    mutationFn: (payload: AgentAdminConfigUpdate) => agentApi.testAdminConfig(payload),
  });
}

export function useSyncAdminAgentConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: AgentAdminConfigUpdate) => agentApi.syncAdminConfig(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminKey });
      void queryClient.invalidateQueries({ queryKey: visibilityKey });
      void queryClient.invalidateQueries({ queryKey: runtimeKey });
    },
  });
}
