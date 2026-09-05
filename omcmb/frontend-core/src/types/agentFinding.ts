export type AgentFindingSeverity = 'info' | 'low' | 'medium' | 'high' | 'critical';

export interface AgentFindingResourceRef {
  type: string;
  id: string;
  role: string;
  label?: string;
}

export interface AgentFindingFact {
  id: string;
  text: string;
  evidenceRefs: string[];
}

export interface AgentFindingHypothesis extends AgentFindingFact {
  confidence: number;
}

export type AgentFindingActionType =
  | 'open-finding'
  | 'open-resource'
  | 'continue-agent'
  | 'dismiss'
  | 'copy-summary';

export interface AgentFindingAction {
  type: AgentFindingActionType;
  label: string;
  resourceRole?: string;
  promptKey?: string;
}

export interface AgentFinding {
  id: string;
  deliveryId: string;
  remoteFindingId: string;
  runId: string;
  scenarioKey: string;
  title: string;
  summary: string;
  severity: AgentFindingSeverity;
  confidence: number;
  facts: AgentFindingFact[];
  hypotheses: AgentFindingHypothesis[];
  details: Record<string, unknown>;
  suggestedActions: AgentFindingAction[];
  presentation: { surfaces?: string[]; sections?: string[]; icon?: string };
  resources: AgentFindingResourceRef[];
  read: boolean;
  dismissed: boolean;
  createdAt: string;
  expiresAt?: string;
}

export interface AgentFindingContinue {
  context: { agentFindingId: string };
  message: string;
}
