import { useMemo } from 'react';
import {
  createAgentRuntimeClient,
  type AgentRuntimeClient,
} from '../agentkit';
import { resolveAgentRuntimeConfig, type AgentRuntimeConfig } from '../config/agent';
import { agentApi } from '../services/api/agentApi';
import { useAgentRuntimeConfig } from './api/useAgentConfig';

export interface UseAgentRuntimeClientResult {
  enabled: boolean;
  config: AgentRuntimeConfig;
  client: AgentRuntimeClient | null;
}

export function useAgentRuntimeClient(configOverride?: AgentRuntimeConfig): UseAgentRuntimeClientResult {
  const runtimeConfigQuery = useAgentRuntimeConfig(configOverride === undefined);
  const envConfig = useMemo(() => resolveAgentRuntimeConfig(), []);
  const serverConfig = runtimeConfigQuery.isSuccess && runtimeConfigQuery.data
    ? {
        enabled: runtimeConfigQuery.data.enabled && Boolean(runtimeConfigQuery.data.endpoint),
        endpoint: runtimeConfigQuery.data.endpoint || '',
        connectorId: runtimeConfigQuery.data.connectorId || '',
      }
    : null;
  const config = configOverride ?? serverConfig ?? envConfig;

  const client = useMemo(() => {
    if (!config.enabled || !config.endpoint) return null;
    return createAgentRuntimeClient({
      endpoint: config.endpoint,
      getDelegationToken: async () => {
        const response = await agentApi.requestDelegationToken();
        return response.token;
      },
    });
  }, [config.enabled, config.endpoint]);

  return {
    enabled: config.enabled,
    config,
    client,
  };
}
