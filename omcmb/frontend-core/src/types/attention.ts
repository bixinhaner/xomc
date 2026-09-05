export type AttentionSectionKey = 'abnormalities' | 'todos';
export type AttentionSectionStatus = 'ok' | 'partial' | 'error';
export type AttentionKind =
  | 'active_alarm'
  | 'device_access_review'
  | 'agent_finding' | 'assistant_result';
export type AttentionAction =
  | 'view_alarm'
  | 'review_device_candidate'
  | 'view_agent_finding' | 'view_assistant_result';

export interface AttentionTarget {
  type: string;
  id?: string;
  name?: string;
  serialNumber?: string;
}

export interface AttentionItem {
  id: string;
  kind: AttentionKind;
  source: string;
  sourceId: string;
  title: string;
  summary: string;
  severity?: string;
  priority: string;
  risk?: string;
  target?: AttentionTarget;
  occurredAt?: string;
  createdAt?: string;
  updatedAt?: string;
  detailRoute: string;
  allowedActions: AttentionAction[];
}

export interface AttentionSection {
  status: AttentionSectionStatus;
  total: number;
  items: AttentionItem[];
}

export interface AttentionSummary {
  abnormalities: AttentionSection;
  todos: AttentionSection;
  generatedAt: string;
}

export interface AttentionPage extends AttentionSection {
  page: number;
  pageSize: number;
}
