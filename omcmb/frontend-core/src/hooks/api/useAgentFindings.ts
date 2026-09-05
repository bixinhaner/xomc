import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { agentFindingApi } from '../../services/api/agentFindingApi';
import { attentionKeys } from './useAttention';

export const agentFindingKeys = {
  all: ['agent', 'findings'] as const,
  detail: (id: string) => [...agentFindingKeys.all, 'detail', id] as const,
};

export function useAgentFinding(id: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: agentFindingKeys.detail(id ?? ''),
    queryFn: () => agentFindingApi.get(id ?? ''),
    enabled: enabled && Boolean(id),
    staleTime: 15_000,
    retry: 1,
  });
}

export function useMarkAgentFindingRead() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: agentFindingApi.markRead,
    onSuccess: (_data, id) => {
      client.setQueryData(agentFindingKeys.detail(id), (current: unknown) => (
        current && typeof current === 'object' ? { ...current, read: true } : current
      ));
      void client.invalidateQueries({ queryKey: attentionKeys.all });
    },
  });
}

export function useDismissAgentFinding() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: agentFindingApi.dismiss,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: attentionKeys.all });
      void client.invalidateQueries({ queryKey: agentFindingKeys.all });
    },
  });
}

export function useContinueAgentFinding() {
  return useMutation({ mutationFn: agentFindingApi.continueAgent });
}
