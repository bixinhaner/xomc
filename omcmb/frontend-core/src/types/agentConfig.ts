export type AgentConfigStatus =
  | 'not_configured'
  | 'disabled'
  | 'connected'
  | 'error'
  | string;

export interface AgentAdminConfig {
  enabled: boolean;
  agentStudioBaseUrl: string;
  serviceTokenConfigured: boolean;
  omcPublicBaseUrl: string;
  connectorSlug: string;
  connectorId: string;
  runtimeStreamUrl: string;
  status: AgentConfigStatus;
  lastValidatedAt: string;
  lastError: string;
  healthPath: string;
  actionListPath: string;
  actionSearchPath: string;
  actionDescribePath: string;
  actionPreviewPath: string;
  actionExecutePath: string;
}

export interface AgentRuntimeServerConfig {
  enabled: boolean;
  endpoint: string;
  connectorId: string;
  status: AgentConfigStatus;
  lastValidatedAt: string;
  lastError: string;
  configuredSource: 'server' | string;
}

export interface AgentAdminConfigUpdate {
  enabled: boolean;
  agentStudioBaseUrl: string;
  agentStudioServiceToken?: string;
  omcPublicBaseUrl: string;
  connectorSlug?: string;
}

export interface AgentProvisionResult {
  connectorId: string;
  slug: string;
  status: AgentConfigStatus;
  runtimeStreamPath: string;
  runtimeStreamUrl: string;
  detail?: string;
}
