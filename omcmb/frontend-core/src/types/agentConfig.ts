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
  connectorSlug: string;
  connectorId: string;
  runtimeStreamUrl: string;
  status: AgentConfigStatus;
  lastValidatedAt: string;
  lastError: string;
  policy: AgentRuntimePolicy;
}

export interface AgentRuntimePolicy {
  allowedMethods: string[];
  blockedPathPrefixes: string[];
  toolTimeoutSeconds: number;
  maxResponseBytes: number;
}

export interface AgentRuntimeServerConfig {
  visible: boolean;
  enabled: boolean;
  endpoint: string;
  connectorId: string;
  status: AgentConfigStatus;
  lastValidatedAt: string;
  lastError: string;
  configuredSource: 'server' | string;
  policy: AgentRuntimePolicy;
}

export interface AgentVisibilityConfig {
  visible: boolean;
  enabled: boolean;
  status: AgentConfigStatus;
  lastValidatedAt: string;
  lastError: string;
}

export interface AgentConversation {
  conversationId: string;
}

export interface AgentAdminConfigUpdate {
  enabled?: boolean;
  agentStudioBaseUrl?: string;
  agentStudioServiceToken?: string;
  allowedMethods?: string[];
  blockedPathPrefixes?: string[];
  toolTimeoutSeconds?: number;
  maxResponseBytes?: number;
}

export interface AgentProvisionResult {
  connectorId: string;
  slug: string;
  status: AgentConfigStatus;
  runtimeStreamPath: string;
  runtimeStreamUrl: string;
  detail?: string;
}
