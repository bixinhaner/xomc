import type {
  AgentApprovedAction,
  AgentError,
  AgentRiskLevel,
} from './protocol';

export type AgentPanelMessageRole = 'user' | 'assistant';
export type AgentPanelMessageStatus = 'streaming' | 'done' | 'error';
export type AgentPanelActivityStatus =
  | 'calling'
  | 'preview'
  | 'running'
  | 'ok'
  | 'error'
  | 'cancelled';

export interface AgentPanelMessage {
  id: string;
  role: AgentPanelMessageRole;
  text: string;
  status: AgentPanelMessageStatus;
  createdAt: number;
}

export interface AgentPanelActivity {
  callId: string;
  toolName: string;
  title: string;
  status: AgentPanelActivityStatus;
  input?: unknown;
  request?: AgentApprovedAction;
  risk?: AgentRiskLevel;
  summary?: string;
  preview?: unknown;
  output?: unknown;
  error?: AgentError;
  updatedAt: number;
}

export interface AgentPendingAction {
  callId: string;
  message: string;
  title: string;
  summary: string;
  risk: AgentRiskLevel;
  request: AgentApprovedAction;
  preview: unknown;
}

export interface AgentDisplayRow {
  key: string;
  value: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function parseApprovedAction(input: unknown): AgentApprovedAction | null {
  if (!isRecord(input) || typeof input.actionId !== 'string' || input.actionId.trim() === '') {
    return null;
  }
  return {
    actionId: input.actionId,
    input: isRecord(input.input) ? input.input : {},
    dryRun: typeof input.dryRun === 'boolean' ? input.dryRun : undefined,
  };
}

export function formatAgentValue(value: unknown): string {
  if (value === null || value === undefined) return '-';
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  if (Array.isArray(value)) return `${value.length}`;
  if (isRecord(value)) return JSON.stringify(value);
  return String(value);
}

function unwrapResult(value: unknown): unknown {
  if (!isRecord(value)) return value;
  const nested = value.result;
  return nested === undefined ? value : nested;
}

export function extractAgentRows(value: unknown, maxRows = 6): AgentDisplayRow[] {
  const payload = unwrapResult(value);
  if (!isRecord(payload)) return [{ key: 'value', value: formatAgentValue(payload) }];

  const preferred = isRecord(payload.statistics) ? payload.statistics : payload;
  const rows: AgentDisplayRow[] = [];
  for (const [key, rowValue] of Object.entries(preferred)) {
    if (rows.length >= maxRows) break;
    if (rowValue === undefined || typeof rowValue === 'function') continue;
    rows.push({ key, value: formatAgentValue(rowValue) });
  }
  if (Array.isArray(payload.items) && rows.length < maxRows) {
    rows.push({ key: 'items', value: String(payload.items.length) });
  }
  return rows;
}

