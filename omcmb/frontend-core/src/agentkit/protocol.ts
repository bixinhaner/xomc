export type AgentRiskLevel = 'read' | 'low' | 'high';

export interface AgentError {
  code: string;
  message: string;
  retryable?: boolean;
  requestId?: string;
  details?: unknown;
}

export interface AgentUsage {
  inputTokens?: number;
  outputTokens?: number;
  totalTokens?: number;
}

export interface AgentPageContext {
  path: string;
  title?: string;
  query?: Record<string, string | string[]>;
  selected?: Array<Record<string, unknown>>;
  filters?: Record<string, unknown>;
  extra?: Record<string, unknown>;
}

export interface AgentActionDescriptor {
  id: string;
  title: string;
  description: string;
  inputSchema: Record<string, unknown>;
  risk: AgentRiskLevel;
  scopes: string[];
}

export interface AgentRuntimeRequest {
  message: string;
  conversationId?: string;
  locale: string;
  timezone: string;
  context: AgentPageContext;
}

export type AgentStreamEvent =
  | { type: 'start'; runId: string; conversationId: string }
  | { type: 'delta'; text: string }
  | {
      type: 'tool_call';
      callId: string;
      toolName: string;
      title: string;
      input: unknown;
    }
  | {
      type: 'action_preview';
      callId: string;
      title: string;
      summary: string;
      risk: AgentRiskLevel;
      preview: unknown;
    }
  | {
      type: 'tool_result';
      callId: string;
      status: 'ok' | 'error';
      output?: unknown;
      error?: AgentError;
    }
  | { type: 'done'; usage?: AgentUsage }
  | { type: 'error'; error: AgentError };

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isString(value: unknown): value is string {
  return typeof value === 'string';
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every(isString);
}

function isRisk(value: unknown): value is AgentRiskLevel {
  return value === 'read' || value === 'low' || value === 'high';
}

function isToolResultStatus(value: unknown): value is 'ok' | 'error' {
  return value === 'ok' || value === 'error';
}

export function isAgentError(value: unknown): value is AgentError {
  return (
    isRecord(value) &&
    isString(value.code) &&
    isString(value.message) &&
    (value.retryable === undefined || typeof value.retryable === 'boolean') &&
    (value.requestId === undefined || isString(value.requestId))
  );
}

export function isAgentActionDescriptor(
  value: unknown
): value is AgentActionDescriptor {
  return (
    isRecord(value) &&
    isString(value.id) &&
    isString(value.title) &&
    isString(value.description) &&
    isRecord(value.inputSchema) &&
    isRisk(value.risk) &&
    isStringArray(value.scopes)
  );
}

export function isAgentStreamEvent(value: unknown): value is AgentStreamEvent {
  if (!isRecord(value) || !isString(value.type)) return false;

  switch (value.type) {
    case 'start':
      return isString(value.runId) && isString(value.conversationId);
    case 'delta':
      return isString(value.text);
    case 'tool_call':
      return (
        isString(value.callId) &&
        isString(value.toolName) &&
        isString(value.title) &&
        'input' in value
      );
    case 'action_preview':
      return (
        isString(value.callId) &&
        isString(value.title) &&
        isString(value.summary) &&
        isRisk(value.risk) &&
        'preview' in value
      );
    case 'tool_result':
      return (
        isString(value.callId) &&
        isToolResultStatus(value.status) &&
        (value.error === undefined || isAgentError(value.error))
      );
    case 'done':
      return value.usage === undefined || isRecord(value.usage);
    case 'error':
      return isAgentError(value.error);
    default:
      return false;
  }
}

