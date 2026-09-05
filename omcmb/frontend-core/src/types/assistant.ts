export interface AssistantCapability { operationId: string; title: string; description: string; path: string; deviceScoped: boolean }
export interface AssistantEvent { type: string; title: string; fields: string[] }
export interface AssistantCondition { field: string; op: 'eq' | 'ne' | 'in' | 'gte' | 'lte'; value: string | number | boolean | string[] }
export interface AssistantTrigger {
  kind: 'manual' | 'interval' | 'schedule' | 'event'; intervalMinutes?: number; time?: string;
  timezone?: string; weekdays?: number[]; eventType?: string; conditions: AssistantCondition[];
}
export interface AssistantDefinition {
  name: string; goal: string; scope: { kind: 'visible' | 'device'; deviceId?: string; label?: string };
  trigger: AssistantTrigger; operations: string[]; notify: 'always' | 'findings'; cooldownMinutes: number;
}
export interface Assistant {
  id: string; revision: number; definition: AssistantDefinition | null;
  messages: Array<{ role: 'user' | 'assistant'; text: string }>;
  readiness: 'ready' | 'needs_input' | 'unsupported'; questions: string[]; missingCapabilities: string[];
  state: 'draft' | 'active' | 'paused' | 'blocked'; publishedRevision: number | null;
  publishedDefinition?: AssistantDefinition; publishedAt?: string; nextRunAt: string | null;
  lastRunAt: string | null; lastError?: string; createdAt: string; updatedAt: string; locale: string; timezone: string;
}
export interface AssistantResult {
  outcome: 'finding' | 'no_change' | 'insufficient_data'; title: string; summary: string;
  facts: Array<{ text: string; evidenceRefs: string[] }>; hypotheses: string[]; nextSteps: string[];
}
export interface AssistantRun {
  id: string; assistantId: string; revision: number; kind: 'trial' | 'manual' | 'schedule' | 'event';
  status: 'QUEUED' | 'RUNNING' | 'CANCELLING' | 'CANCELLED' | 'COMPLETED' | 'FAILED';
  output: AssistantResult | null; errorCode?: string; errorMessage?: string;
  tools: Array<{ id: string; operationId: string; status: string; createdAt: string }>;
  notifyVisible?: boolean; attempts: number; createdAt: string; startedAt: string | null; completedAt: string | null; readAt: string | null;
}
export interface AssistantCatalog { capabilities: AssistantCapability[]; events: AssistantEvent[]; connected: boolean; connectionError?: string }
