import { useMemo } from 'react';
import {
  createAgentRuntimeClient,
  type AgentRuntimeClient,
} from '../agentkit';
import { resolveAgentRuntimeConfig, type AgentRuntimeConfig } from '../config/agent';
import { agentApi } from '../services/api/agentApi';

export interface UseAgentRuntimeClientResult {
  enabled: boolean;
  config: AgentRuntimeConfig;
  client: AgentRuntimeClient | null;
}

export function useAgentRuntimeClient(configOverride?: AgentRuntimeConfig): UseAgentRuntimeClientResult {
  const config = configOverride ?? resolveAgentRuntimeConfig();

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

