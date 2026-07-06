export interface AgentRuntimeConfig {
  enabled: boolean;
  endpoint: string;
  connectorId: string;
}

type AgentEnv = {
  VITE_AGENT_RUNTIME_STREAM_URL?: string;
  VITE_AGENT_RUNTIME_BASE_URL?: string;
  VITE_AGENT_ACTION_CONNECTOR_ID?: string;
};

function trim(value: string | undefined): string {
  return typeof value === 'string' ? value.trim() : '';
}

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '');
}

export function resolveAgentRuntimeConfig(env: AgentEnv = import.meta.env): AgentRuntimeConfig {
  const directEndpoint = trim(env.VITE_AGENT_RUNTIME_STREAM_URL);
  const baseUrl = trim(env.VITE_AGENT_RUNTIME_BASE_URL);
  const connectorId = trim(env.VITE_AGENT_ACTION_CONNECTOR_ID);

  if (directEndpoint) {
    return {
      enabled: true,
      endpoint: directEndpoint,
      connectorId,
    };
  }

  if (!baseUrl || !connectorId) {
    return {
      enabled: false,
      endpoint: '',
      connectorId,
    };
  }

  return {
    enabled: true,
    endpoint: `${trimTrailingSlash(baseUrl)}/api/action-connectors/${encodeURIComponent(connectorId)}/chat/stream`,
    connectorId,
  };
}

