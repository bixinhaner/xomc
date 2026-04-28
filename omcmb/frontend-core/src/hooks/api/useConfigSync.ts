import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { configSyncApi } from '../../services/api/configSyncApi';

/**
 * Config sync hooks — TR-069 SetParameterValues / GetParameterValues bridge.
 *
 * `useConfig.useUpdateConfigParam` already calls `configSyncApi.pushConfig`
 * inline for single-parameter updates. This file exposes the full surface
 * (push / pull / status) for pages that need to dispatch a multi-parameter
 * sync command and watch its pending count.
 *
 * Server-only (no mock equivalent); calls `configSyncApi.*` directly.
 */

interface ParameterValue {
  name: string;
  value: string;
  type?: string;
}

export function useSyncStatus(deviceId: string, options?: { refetchInterval?: number }) {
  return useQuery({
    queryKey: ['config-sync', 'status', deviceId],
    queryFn: () => configSyncApi.getSyncStatus(deviceId),
    enabled: Boolean(deviceId),
    refetchInterval: options?.refetchInterval ?? false,
  });
}

export function usePushConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceId, parameters }: { deviceId: string; parameters: ParameterValue[] }) =>
      configSyncApi.pushConfig(deviceId, parameters),
    onSuccess: (_, { deviceId }) => {
      void queryClient.invalidateQueries({ queryKey: ['config-sync', 'status', deviceId] });
    },
  });
}

export function usePullConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceId, parameterNames }: { deviceId: string; parameterNames: string[] }) =>
      configSyncApi.pullConfig(deviceId, parameterNames),
    onSuccess: (_, { deviceId }) => {
      void queryClient.invalidateQueries({ queryKey: ['config-sync', 'status', deviceId] });
    },
  });
}
