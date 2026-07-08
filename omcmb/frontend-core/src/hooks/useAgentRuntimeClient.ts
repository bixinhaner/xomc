import { useMemo } from 'react';
import {
  createAgentRuntimeClient,
  type AgentRuntimeClient,
} from '../agentkit';
import { disabledAgentRuntimeConfig, type AgentRuntimeConfig } from '../config/agent';
import { useUserStore } from '../store/userStore';
import { useAgentRuntimeConfig } from './api/useAgentConfig';

export interface UseAgentRuntimeClientResult {
  enabled: boolean;
  config: AgentRuntimeConfig;
  client: AgentRuntimeClient | null;
}

export function useAgentRuntimeClient(configOverride?: AgentRuntimeConfig): UseAgentRuntimeClientResult {
  const runtimeConfigQuery = useAgentRuntimeConfig(configOverride === undefined);
  const serverConfig = runtimeConfigQuery.isSuccess && runtimeConfigQuery.data
    ? {
        enabled: runtimeConfigQuery.data.enabled && Boolean(runtimeConfigQuery.data.endpoint),
        endpoint: runtimeConfigQuery.data.endpoint || '',
        connectorId: runtimeConfigQuery.data.connectorId || '',
      }
    : null;
  const config = configOverride ?? serverConfig ?? disabledAgentRuntimeConfig();

  const client = useMemo(() => {
    if (!config.enabled || !config.endpoint) return null;
    return createAgentRuntimeClient({
      endpoint: config.endpoint,
      getAuthHeaders: (): Record<string, string> => {
        const token = useUserStore.getState().accessToken;
        if (!token) return {};
        return { Authorization: `Bearer ${token}` };
      },
    });
  }, [config.enabled, config.endpoint]);

  return {
    enabled: config.enabled,
    config,
    client,
  };
}
