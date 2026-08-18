import http from '../http';
import type { PageResponse } from '../../types/pagination';

export type AccessState =
  | 'review_required'
  | 'collecting'
  | 'accepted'
  | 'rejected'
  | 'revalidating'
  | 'revoked';
export type AccessListType = 'deny' | 'allow' | 'revoked';
export type PolicyDefaultAction = 'reject';
export type SerialScopeType = 'all' | 'list' | 'prefix' | 'range';
export type AccessConditionType = 'tac' | 'ecgi' | 'observed_ip' | 'gps';
export type AccessConditionOperator = 'equal' | 'in' | 'cidr' | 'within_radius';
export type ActionStatus =
  | 'pending_dispatch'
  | 'dispatching'
  | 'succeeded'
  | 'failed'
  | 'cancelled';
export interface DeviceAccessRuntimeSettings {
  carrier: string;
  enabled: boolean;
  updated_by?: string;
  updated_at?: string;
}

export interface AccessStateItem {
  carrier: string;
  serial_number: string;
  product_name: string;
  device_id?: string;
  candidate_id?: string;
  state: AccessState;
  effective_decision: string;
  reason_code: string;
  policy_version_id?: string;
  evidence_version: number;
  decision_version: number;
  normal_tasks_frozen: boolean;
  last_decided_at: string;
  updated_at: string;
}

export interface DecisionCheckItem {
  id: string;
  check_type: string;
  result: string;
  expected_summary?: string;
  observed_summary?: string;
  evidence_source?: string;
  observed_at?: string;
  reason_code?: string;
}

export interface DecisionItem {
  id: string;
  trigger_type: string;
  previous_state?: AccessState;
  new_state: AccessState;
  decision: string;
  reason_code: string;
  evidence_version: number;
  decision_version: number;
  occurred_at: string;
  checks: DecisionCheckItem[];
}

export interface EvidenceItem {
  id: string;
  evidence_version: number;
  evidence_type: string;
  evidence_status: string;
  normalized_value: unknown;
  source: string;
  task_id?: string;
  observed_at: string;
  expires_at?: string;
}

export interface AccessAction {
  id: string;
  device_id: string;
  decision_id: string;
  action_type: 'rf_off' | 'rf_on';
  direction: 'contain' | 'release';
  status: ActionStatus;
  device_task_id?: string;
  recovery_of_action_id?: string;
  owned_rf_change: boolean;
  attempts: number;
  error_message?: string;
  dispatched_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
  serial_number: string;
  product_name: string;
  carrier: string;
}

export interface AccessDetail {
  state: AccessStateItem;
  decisions: DecisionItem[];
  evidence: EvidenceItem[];
  actions: AccessAction[];
}

export interface PolicyVersionSummary {
  id: string;
  policy_set_id: string;
  name: string;
  carrier: string;
  version: number;
  status: 'draft' | 'published' | 'retired';
  default_action: PolicyDefaultAction;
  created_by?: string;
  published_by?: string;
  published_at?: string;
  created_at: string;
}

export interface CompiledAccessCondition {
  id: string;
  type: AccessConditionType;
  operator: AccessConditionOperator;
  expected?: string;
  expected_any?: string[];
  geo_fence?: {
    center: { latitude: number; longitude: number };
    radius_meters: number;
    allow_missing?: boolean;
  };
  required: boolean;
  evidence_ttl: number;
}

export interface CompiledAccessRule {
  id: string;
  name: string;
  enabled: boolean;
  serial_scope: { type: SerialScopeType; values?: string[]; prefix?: string; start?: string; end?: string };
  conditions: CompiledAccessCondition[];
}

export interface CompiledAccessPolicy {
  version_id?: string;
  default_action: PolicyDefaultAction;
  list_entries?: never[];
  rules: CompiledAccessRule[];
}

export interface PolicyVersion {
  id: string;
  carrier: string;
  name: string;
  version: number;
  status: 'draft' | 'published' | 'retired';
  policy: CompiledAccessPolicy;
  content_hash?: string;
  created_by?: string;
  published_by?: string;
  published_at?: string;
}

export interface AccessListItem {
  id: string;
  carrier: string;
  entry_type: AccessListType;
  identity_type: string;
  identity_value: string;
  product_name: string;
  reason_code: string;
  reason?: string;
  valid_from: string;
  valid_until?: string;
  status: 'active' | 'disabled';
  created_at: string;
  updated_at: string;
}

export interface CandidateItem {
  id: string;
  carrier: string;
  serial_number: string;
  oui: string;
  product_class?: string;
  software_version?: string;
  observed_remote_ip?: string;
  device_id?: string;
  first_seen_at: string;
  last_seen_at: string;
  inform_count: number;
  review_status: 'pending' | 'approved' | 'rejected' | 'expired';
  reviewed_at?: string;
  expires_at: string;
}

type ListParams = {
  operatorCode: string;
  page: number;
  pageSize: number;
  serialNumber?: string;
};

const headers = (operatorCode: string) => ({ 'X-Operator-Code': operatorCode });

function page<T, S = unknown>(data: { items: T[]; total: number; page: number; page_size: number; stats?: S }): PageResponse<T, S> {
  return {
    items: data.items ?? [],
    total: data.total ?? 0,
    page: data.page,
    pageSize: data.page_size,
    ...(data.stats === undefined ? {} : { stats: data.stats }),
  };
}

function listQuery(params: Omit<ListParams, 'operatorCode'>) {
  return {
    page: params.page,
    page_size: params.pageSize,
    serial_number: params.serialNumber,
  };
}

export const deviceAccessApi = {
  async getRuntimeSettings(operatorCode: string): Promise<DeviceAccessRuntimeSettings> {
    const { data } = await http.get<DeviceAccessRuntimeSettings>('/device-access/settings', {
      headers: headers(operatorCode),
    });
    return data;
  },
  async updateRuntimeSettings(operatorCode: string, enabled: boolean): Promise<DeviceAccessRuntimeSettings> {
    const { data } = await http.put<DeviceAccessRuntimeSettings>('/device-access/settings', { enabled }, {
      headers: headers(operatorCode),
    });
    return data;
  },
  async listStates(params: ListParams & { state?: AccessState }): Promise<PageResponse<AccessStateItem>> {
    const { operatorCode, state, ...query } = params;
    const { data } = await http.get<{ items: AccessStateItem[]; total: number; page: number; page_size: number }>(
      '/device-access/states', { params: { ...listQuery(query), state }, headers: headers(operatorCode) },
    );
    return page(data);
  },
  async getDetail(operatorCode: string, serialNumber: string): Promise<AccessDetail> {
    const { data } = await http.get<AccessDetail>(`/device-access/states/${encodeURIComponent(serialNumber)}`, { headers: headers(operatorCode) });
    return data;
  },
  async reevaluateDevice(operatorCode: string, serialNumber: string): Promise<void> {
    await http.post(`/device-access/devices/${encodeURIComponent(serialNumber)}/reevaluate`, undefined, {
      headers: headers(operatorCode),
    });
  },
  async listPolicies(params: ListParams): Promise<PageResponse<PolicyVersionSummary>> {
    const { operatorCode, ...query } = params;
    const { data } = await http.get<{ items: PolicyVersionSummary[]; total: number; page: number; page_size: number }>(
      '/device-access/policies', { params: listQuery(query), headers: headers(operatorCode) },
    );
    return page(data);
  },
  async getPolicy(operatorCode: string, versionId: string): Promise<PolicyVersion> {
    const { data } = await http.get<PolicyVersion>(`/device-access/policies/${versionId}`, { headers: headers(operatorCode) });
    return data;
  },
  async createPolicyDraft(input: { operatorCode: string; name: string; policy: CompiledAccessPolicy }): Promise<PolicyVersion> {
    const { data } = await http.post<PolicyVersion>('/device-access/policies/drafts', {
      name: input.name,
      policy: input.policy,
    }, { headers: headers(input.operatorCode) });
    return data;
  },
  async publishPolicy(operatorCode: string, versionId: string): Promise<void> {
    await http.post(`/device-access/policies/${versionId}/publish`, undefined, { headers: headers(operatorCode) });
  },
  async deletePolicyDraft(operatorCode: string, versionId: string): Promise<void> {
    await http.delete(`/device-access/policies/${versionId}`, { headers: headers(operatorCode) });
  },
  async listEntries(params: ListParams & { entryType?: AccessListType }): Promise<PageResponse<AccessListItem>> {
    const { operatorCode, entryType, ...query } = params;
    const { data } = await http.get<{ items: AccessListItem[]; total: number; page: number; page_size: number }>(
      '/device-access/access-list', { params: { ...listQuery(query), entry_type: entryType }, headers: headers(operatorCode) },
    );
    return page(data);
  },
  async upsertEntry(input: { operatorCode: string; entryType: AccessListType; serialNumber: string; reason?: string; status?: 'active' | 'disabled' }): Promise<void> {
    await http.post('/device-access/access-list', {
      entry: {
        type: input.entryType,
        identity_type: 'serial_number',
        identity_value: input.serialNumber,
        status: input.status ?? 'active',
        reason: input.reason,
      },
    }, { headers: headers(input.operatorCode) });
  },
  async listCandidates(params: ListParams & { reviewStatus?: string }): Promise<PageResponse<CandidateItem>> {
    const { operatorCode, reviewStatus, ...query } = params;
    const { data } = await http.get<{ items: CandidateItem[]; total: number; page: number; page_size: number }>(
      '/device-access/candidates', { params: { ...listQuery(query), review_status: reviewStatus }, headers: headers(operatorCode) },
    );
    return page(data);
  },
  async reviewCandidate(input: { operatorCode: string; candidateId: string; outcome: 'allow' | 'deny'; reason: string }): Promise<void> {
    await http.post(`/device-access/candidates/${input.candidateId}/review`, { outcome: input.outcome, reason: input.reason }, { headers: headers(input.operatorCode) });
  },
  async listActions(params: ListParams & { status?: ActionStatus }): Promise<PageResponse<AccessAction>> {
    const { operatorCode, status, ...query } = params;
    const { data } = await http.get<{ items: AccessAction[]; total: number; page: number; page_size: number }>(
      '/device-access/actions', { params: { ...listQuery(query), status }, headers: headers(operatorCode) },
    );
    return page(data);
  },
  async retryAction(operatorCode: string, actionId: string): Promise<void> {
    await http.post(`/device-access/actions/${actionId}/retry`, undefined, { headers: headers(operatorCode) });
  },
};
