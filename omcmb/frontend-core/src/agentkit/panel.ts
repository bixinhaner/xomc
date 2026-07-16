import type {
  AgentApprovedAction,
  AgentError,
  AgentProcessKind,
  AgentRiskLevel,
  AgentThoughtStatus,
  AgentAttachmentRef,
  AgentArtifactRef,
  AgentUiIntent,
} from './protocol';

export type AgentPanelMessageRole = 'user' | 'assistant';
export type AgentPanelMessageStatus = 'streaming' | 'done' | 'error' | 'cancelled';
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
  runId?: string;
  conversationId?: string;
  thoughts?: AgentThoughtEntry[];
  process?: AgentProcessEntry[];
  attachments?: AgentAttachmentRef[];
  artifacts?: AgentArtifactRef[];
  uiIntents?: AgentUiIntent[];
  durationMs?: number;
}

export interface AgentThoughtEntry {
  id: string;
  text: string;
  lines: string[];
  status: AgentThoughtStatus;
  source?: 'thought' | 'status' | 'delta';
  at?: string;
}

export interface AgentProcessEntry {
  id: string;
  kind: AgentProcessKind;
  title: string;
  detail?: unknown;
  at?: string;
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

export interface AgentRestInput {
  method: string;
  path: string;
  operationId: string;
  query: Record<string, unknown>;
  body?: unknown;
  reason: string;
}

export interface AgentProcessSummary {
  total: number;
  searches: number;
  calls: number;
  readOnly: number;
  writes: number;
  errors: number;
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

export function formatAgentDetail(value: unknown): string {
  if (value === null || value === undefined || value === '') return '';
  if (typeof value === 'string') return value;
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export function extractAgentRestInput(input: unknown): AgentRestInput | null {
  if (!isRecord(input)) return null;
  const method = typeof input.method === 'string' ? input.method.trim().toUpperCase() : '';
  const path = typeof input.path === 'string' ? input.path.trim() : '';
  if (!method || !path) return null;
  return {
    method,
    path,
    operationId: typeof input.operationId === 'string' ? input.operationId.trim() : '',
    query: isRecord(input.query) ? input.query : {},
    body: input.body,
    reason: typeof input.reason === 'string' ? input.reason.trim() : '',
  };
}

export function summarizeAgentProcess(entries: AgentProcessEntry[] | undefined): AgentProcessSummary {
  const summary: AgentProcessSummary = {
    total: entries?.length ?? 0,
    searches: 0,
    calls: 0,
    readOnly: 0,
    writes: 0,
    errors: 0,
  };
  if (!entries?.length) return summary;

  for (const entry of entries) {
    if (entry.kind === 'error') summary.errors += 1;
    if (entry.kind !== 'tool_call') continue;

    const rest = extractAgentRestInput(entry.detail);
    if (!rest) continue;
    if (rest.path === '/api/v1/agent/catalog' || rest.path === '/api/v1/agent/catalog/describe') {
      summary.searches += 1;
    } else {
      summary.calls += 1;
    }
    if (rest.method === 'GET') {
      summary.readOnly += 1;
    } else {
      summary.writes += 1;
    }
  }

  return summary;
}

function thoughtLines(text: string): string[] {
  return text
    .split(/\n{2,}/)
    .map((line) => line.trim())
    .filter(Boolean);
}

function normalizeText(value: string): string {
  return value.replace(/\s+/g, ' ').trim();
}

export function mergeAgentThought(
  thoughts: AgentThoughtEntry[] | undefined,
  input: {
    id: string;
    text: string;
    append?: boolean;
    status?: AgentThoughtStatus;
    source?: AgentThoughtEntry['source'];
    at?: string;
  }
): AgentThoughtEntry[] {
  const normalized = input.append === true ? input.text : input.text.trim();
  if (!normalized) return thoughts ?? [];
  const next = [...(thoughts ?? [])];
  const matchedIndex = next.findIndex((thought) => thought.id === input.id);
  const duplicateIndex =
    matchedIndex >= 0
      ? -1
      : next.findIndex(
          (thought) =>
            thought.source === input.source &&
            normalizeText(thought.text) === normalizeText(normalized)
        );
  const last = next[next.length - 1];
  const targetIndex =
    matchedIndex >= 0
      ? matchedIndex
      : duplicateIndex >= 0
        ? duplicateIndex
        : input.append === true && last?.status === 'streaming' && last.source === input.source
          ? next.length - 1
          : -1;
  const status = input.status ?? 'streaming';

  if (targetIndex >= 0) {
    const current = next[targetIndex];
    if (!current) return next;
    const text = input.append === true ? `${current.text}${normalized}` : normalized;
    next[targetIndex] = {
      ...current,
      text,
      lines: input.append === true ? [text] : thoughtLines(text),
      status,
      source: input.source ?? current.source,
      at: current.at ?? input.at,
    };
    return next;
  }

  return [
    ...next,
    {
      id: input.id,
      text: normalized,
      lines: thoughtLines(normalized),
      status,
      source: input.source,
      at: input.at,
    },
  ];
}

export function completeAgentThoughts(
  thoughts: AgentThoughtEntry[] | undefined
): AgentThoughtEntry[] | undefined {
  if (!thoughts?.length) return thoughts;
  return thoughts.map((thought) => ({ ...thought, status: 'completed' }));
}

export function upsertAgentProcess(
  entries: AgentProcessEntry[] | undefined,
  nextEntry: AgentProcessEntry
): AgentProcessEntry[] {
  const current = entries ?? [];
  const index = current.findIndex((entry) => entry.id === nextEntry.id);
  if (index === -1) return [...current, nextEntry];
  const copy = current.slice();
  copy[index] = { ...copy[index], ...nextEntry };
  return copy;
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
