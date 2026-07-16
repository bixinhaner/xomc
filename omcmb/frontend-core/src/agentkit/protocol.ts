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

export interface AgentAttachmentRef {
  attachmentId: string;
  filename: string;
  mimeType: string;
  sizeBytes: number;
  sha256?: string;
  createdAt?: string;
}

export interface AgentArtifactRef {
  artifactId: string;
  filename: string;
  mimeType?: string | null;
  sizeBytes?: number | null;
  previewStatus?: string | null;
  downloadStatus?: string | null;
  blockedReason?: string | null;
}

export interface AgentUiIntent {
  kind: 'navigate' | 'show_records' | 'refresh_record' | 'toast';
  route?: string;
  entity?: string;
  ids?: string[];
  query?: Record<string, string>;
  message?: string;
}

export type AgentThoughtStatus = 'streaming' | 'completed';
export type AgentProcessKind =
  | 'status'
  | 'thought'
  | 'tool_call'
  | 'action_preview'
  | 'tool_result'
  | 'artifact'
  | 'ui_intent'
  | 'reasoning'
  | 'source'
  | 'process'
  | 'done'
  | 'debug'
  | 'error';

export interface AgentPageContext {
  path: string;
  title?: string;
  query?: Record<string, string | string[]>;
  selected?: Array<Record<string, unknown>>;
  filters?: Record<string, unknown>;
  extra?: Record<string, unknown>;
}

export type AgentRuntimeMode = 'preview' | 'execute';

export interface AgentApprovedAction {
  actionId: string;
  input?: Record<string, unknown>;
  dryRun?: boolean;
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
  mode?: AgentRuntimeMode;
  approvedAction?: AgentApprovedAction;
  attachments?: Array<Pick<AgentAttachmentRef, 'attachmentId' | 'filename'>>;
  locale: string;
  timezone: string;
  context: AgentPageContext;
}

export type AgentStreamEvent =
  | { type: 'start'; runId: string; conversationId: string }
  | {
      type: 'tool_request';
      runId: string;
      toolCallId: string;
      tool: string;
      title: string;
      input: unknown;
    }
  | {
      type: 'thought';
      id?: string;
      text: string;
      append?: boolean;
      status?: AgentThoughtStatus;
      at?: string;
      lastEventAt?: number;
    }
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
  | {
      type: 'process';
      id?: string;
      kind: AgentProcessKind;
      title: string;
      detail?: unknown;
      at?: string;
    }
  | { type: 'artifact'; files: AgentArtifactRef[] }
  | { type: 'ui_intent'; intent: AgentUiIntent }
  | { type: 'done'; usage?: AgentUsage; durationMs?: number }
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

function isThoughtStatus(value: unknown): value is AgentThoughtStatus {
  return value === 'streaming' || value === 'completed';
}

function isProcessKind(value: unknown): value is AgentProcessKind {
  return (
    value === 'status' ||
    value === 'thought' ||
    value === 'tool_call' ||
    value === 'action_preview' ||
    value === 'tool_result' ||
    value === 'artifact' ||
    value === 'ui_intent' ||
    value === 'reasoning' ||
    value === 'source' ||
    value === 'process' ||
    value === 'done' ||
    value === 'debug' ||
    value === 'error'
  );
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
    case 'tool_request':
      return (
        isString(value.runId) &&
        isString(value.toolCallId) &&
        isString(value.tool) &&
        isString(value.title) &&
        'input' in value
      );
    case 'thought':
      return (
        isString(value.text) &&
        (value.id === undefined || isString(value.id)) &&
        (value.append === undefined || typeof value.append === 'boolean') &&
        (value.status === undefined || isThoughtStatus(value.status)) &&
        (value.at === undefined || isString(value.at)) &&
        (value.lastEventAt === undefined || typeof value.lastEventAt === 'number')
      );
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
    case 'process':
      return (
        isProcessKind(value.kind) &&
        isString(value.title) &&
        (value.id === undefined || isString(value.id)) &&
        (value.at === undefined || isString(value.at))
      );
    case 'artifact':
      return (
        Array.isArray(value.files) &&
        value.files.every(
          (file) =>
            isRecord(file) &&
            isString(file.artifactId) &&
            isString(file.filename)
        )
      );
    case 'ui_intent':
      return isRecord(value.intent) && isString(value.intent.kind);
    case 'done':
      return (
        (value.usage === undefined || isRecord(value.usage)) &&
        (value.durationMs === undefined || typeof value.durationMs === 'number')
      );
    case 'error':
      return isAgentError(value.error);
    default:
      return false;
  }
}
