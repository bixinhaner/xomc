export interface AgentRuntimeConfig {
  enabled: boolean;
  endpoint: string;
  connectorId: string;
}

export function disabledAgentRuntimeConfig(): AgentRuntimeConfig {
  return {
    enabled: false,
    endpoint: '',
    connectorId: '',
  };
}

export function resolveAgentRuntimeConfig(): AgentRuntimeConfig {
  return disabledAgentRuntimeConfig();
}
