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
export type PolicyDefaultAction = 'reject' | 'review';
export type AccessFailureMode = 'fail_closed' | 'review_hold';
export type SerialScopeType = 'all' | 'list' | 'prefix' | 'range';
export type AccessConditionType = 'tac' | 'ecgi' | 'observed_ip' | 'gps';
export type AccessConditionOperator = 'equal' | 'in' | 'cidr' | 'ip_range' | 'within_radius' | 'within_bounds';
export type ActionStatus =
  | 'pending_dispatch'
  | 'dispatching'
  | 'verifying'
  | 'retry_wait'
  | 'succeeded'
  | 'failed'
  | 'dead'
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
  decision_expires_at?: string;
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
  policy_version_id?: string;
  matched_list_entry_id?: string;
  matched_rule_id?: string;
  occurred_at: string;
  checks: DecisionCheckItem[];
  archive?: {
    id: string;
    decision_id: string;
    archived_by: string;
    reason: string;
    archived_at: string;
  };
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
  device_id?: string;
  candidate_id?: string;
  decision_id: string;
  action_type: 'rf_off' | 'rf_on';
  direction: 'contain' | 'release';
  status: ActionStatus;
  device_task_id?: string;
  recovery_of_action_id?: string;
  owned_rf_change: boolean;
  attempts: number;
  max_attempts: number;
  next_attempt_at?: string;
  last_failure_code?: string;
  dead_at?: string;
  manual_repair_required: boolean;
  error_message?: string;
  dispatched_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
  serial_number: string;
  product_name: string;
  carrier: string;
}

export interface AccessActionAttempt {
  id: string;
  action_id: string;
  attempt_no: number;
  phase: 'baseline_gpv' | 'spv' | 'readback_gpv';
  device_task_id?: string;
  command_key?: string;
  status: 'queued' | 'sent' | 'succeeded' | 'failed' | 'timeout' | 'cancelled';
  fault_code?: string;
  failure_code?: string;
  error_message?: string;
  started_at: string;
  completed_at?: string;
}

export interface AccessDetail {
  state: AccessStateItem;
  decisions: DecisionItem[];
  evidence: EvidenceItem[];
  actions: AccessAction[];
}

export interface AccessStateSummary {
  accepted: number;
  rejected: number;
  review_required: number;
  revoked: number;
  total: number;
}

export interface AccessIdentitySnapshot {
  id: string;
  request_id: string;
  decision_id?: string;
  device_id?: string;
  candidate_id?: string;
  carrier: string;
  serial_number: string;
  device_code?: string;
  cloud_key?: string;
  oui?: string;
  product_class?: string;
  raw_remote_ip?: string;
  observed_remote_ip?: string;
  inform_event?: string;
  inform_time: string;
  identity_source: string;
  identity_status: 'resolved' | 'unresolved' | 'conflict' | 'ambiguous';
  identity_reason_code?: string;
}

export interface AccessManualOperationItem {
  id: string;
  user_id?: string;
  username: string;
  operation: string;
  resource_id: string;
  details: Record<string, unknown>;
  created_at: string;
}

export interface AccessNotificationItem {
  id: string;
  channel: string;
  recipients: string[];
  subject: string;
  status: 'pending' | 'sent' | 'failed' | 'dead_letter' | 'not_configured';
  error_message?: string;
  source_type: string;
  source_id?: string;
  event_id?: string;
  correlation_id?: string;
  retry_count: number;
  sent_at?: string;
  created_at: string;
}

export type AccessAuditFilters = {
  decision?: string;
  reasonCode?: string;
  actionStatus?: ActionStatus;
  dimension?: RuleDimension;
  policyVersionId?: string;
  matchedRuleId?: string;
  startedAt?: string;
  endedAt?: string;
  archiveStatus?: 'visible' | 'archived' | 'all';
};

export interface PolicyVersionSummary {
  id: string;
  policy_set_id: string;
  name: string;
  carrier: string;
  version: number;
  status: 'draft' | 'published' | 'retired';
  default_action: PolicyDefaultAction;
  failure_mode?: AccessFailureMode;
  collection_timeout_seconds?: number;
  bypass_profile_count?: number;
  created_by?: string;
  published_by?: string;
  published_at?: string;
  created_at: string;
}

export interface PolicyDifference {
  base_version_id?: string;
  base_version?: number;
  target_version_id: string;
  target_version: number;
  default_action_changed: boolean;
  failure_mode_changed: boolean;
  collection_timeout_changed: boolean;
  bypass_profiles_added: number;
  bypass_profiles_removed: number;
  bypass_profiles_changed: number;
  rules_added: number;
  rules_removed: number;
  rules_changed: number;
  conditions_added: number;
  conditions_removed: number;
  no_changes: boolean;
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
  ip_range?: {
    start: string;
    end: string;
  };
  ip_ranges?: Array<{ start: string; end: string }>;
  geo_bounds?: {
    min_latitude: number;
    max_latitude: number;
    min_longitude: number;
    max_longitude: number;
    allow_missing?: boolean;
  };
  geo_bounds_any?: Array<{
    min_latitude: number;
    max_latitude: number;
    min_longitude: number;
    max_longitude: number;
    allow_missing?: boolean;
  }>;
  required: boolean;
  evidence_ttl: number;
}

export interface CompiledAccessRule {
  id: string;
  name: string;
  enabled: boolean;
  priority: number;
  serial_scope: { type: SerialScopeType; values?: string[]; prefix?: string; start?: string; end?: string };
  conditions: CompiledAccessCondition[];
}

export interface CompiledAccessPolicy {
  version_id?: string;
  default_action: PolicyDefaultAction;
  failure_mode?: AccessFailureMode;
  collection_timeout?: number;
  bypass_profiles?: AccessBypassProfile[];
  list_entries?: never[];
  rules: CompiledAccessRule[];
}

export interface AccessBypassProfile {
  id: string;
  name: string;
  enabled: boolean;
  priority: number;
  serial_scope?: { type: SerialScopeType; values?: string[]; prefix?: string; start?: string; end?: string };
  ouis?: string[];
  product_classes?: string[];
  reason: string;
  valid_from?: string;
  valid_until?: string;
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

export type ImportMode = 'append' | 'replace';
export type ImportFailurePolicy = 'strict' | 'valid_only';
export type ImportBatchStatus = 'uploaded' | 'validated' | 'committing' | 'committed' | 'failed' | 'rolled_back';
export type ImportRowStatus = 'valid' | 'invalid' | 'duplicate' | 'no_change';
export type RuleDimension = 'sn' | 'tac' | 'ecgi' | 'ip' | 'gps';

export interface AccessListImportBatch {
  id: string;
  carrier: string;
  import_type: 'access_list' | 'rule_dimension';
  entry_type?: AccessListType;
  target_policy_version_id?: string;
  target_rule_id?: string;
  dimension?: RuleDimension;
  mode: ImportMode;
  failure_policy: ImportFailurePolicy;
  status: ImportBatchStatus;
  source_filename: string;
  content_sha256: string;
  total_count: number;
  valid_count: number;
  invalid_count: number;
  changed_count: number;
  created_by?: string;
  committed_by?: string;
  rolled_back_by?: string;
  reversal_of_batch_id?: string;
  created_at: string;
  updated_at: string;
  committed_at?: string;
  rolled_back_at?: string;
}

export interface AccessListImportRow {
  id?: string;
  batch_id: string;
  row_number: number;
  raw_value?: Record<string, unknown>;
  normalized_value?: Record<string, unknown>;
  validation_status: ImportRowStatus;
  error_code?: string;
  error_message?: string;
  target_id?: string;
  created_at: string;
}

export interface AccessListImportPreview {
  batch: AccessListImportBatch;
  rows: AccessListImportRow[];
  entries: unknown[];
  disable_count: number;
}

export interface RuleDimensionValue {
  dimension: RuleDimension;
  serial_number?: string;
  value?: string;
  ip_range?: { start: string; end: string };
  geo_bounds?: {
    min_latitude: number;
    max_latitude: number;
    min_longitude: number;
    max_longitude: number;
    allow_missing?: boolean;
  };
}

export interface RuleDimensionImportPreview {
  batch: AccessListImportBatch;
  rows: AccessListImportRow[];
  values: RuleDimensionValue[];
  disable_count: number;
}

export interface AccessListImportDetail {
  batch: AccessListImportBatch;
  rows: AccessListImportRow[];
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

function auditQuery(filters: AccessAuditFilters) {
  return Object.fromEntries(Object.entries({
    decision: filters.decision,
    reason_code: filters.reasonCode,
    action_status: filters.actionStatus,
    dimension: filters.dimension,
    policy_version_id: filters.policyVersionId,
    matched_rule_id: filters.matchedRuleId,
    started_at: filters.startedAt,
    ended_at: filters.endedAt,
    archive_status: filters.archiveStatus,
  }).filter(([, value]) => value !== undefined && value !== ''));
}

async function listStateSection<T>(section: string, params: ListParams, extra: Record<string, unknown> = {}): Promise<PageResponse<T>> {
  const { operatorCode, serialNumber = '', page: currentPage, pageSize } = params;
  const { data } = await http.get<{ items: T[]; total: number; page: number; page_size: number }>(
    `/device-access/states/${encodeURIComponent(serialNumber)}/${section}`,
    { params: { page: currentPage, page_size: pageSize, ...extra }, headers: headers(operatorCode) },
  );
  return page(data);
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
  async listStates(params: ListParams & AccessAuditFilters & { state?: AccessState }): Promise<PageResponse<AccessStateItem>> {
    const { operatorCode, state, page: currentPage, pageSize, serialNumber, ...filters } = params;
    const { data } = await http.get<{ items: AccessStateItem[]; total: number; page: number; page_size: number }>(
      '/device-access/states', { params: { ...listQuery({ page: currentPage, pageSize, serialNumber }), ...auditQuery(filters), state }, headers: headers(operatorCode) },
    );
    return page(data);
  },
  async summarizeStates(params: Omit<ListParams, 'page' | 'pageSize'> & AccessAuditFilters & { state?: AccessState }): Promise<AccessStateSummary> {
    const { operatorCode, serialNumber, state, ...filters } = params;
    const { data } = await http.get<AccessStateSummary>('/device-access/states/summary', {
      params: { serial_number: serialNumber, state, ...auditQuery(filters) }, headers: headers(operatorCode),
    });
    return data;
  },
  async getDetail(operatorCode: string, serialNumber: string): Promise<AccessDetail> {
    const { data } = await http.get<AccessDetail>(`/device-access/states/${encodeURIComponent(serialNumber)}`, { headers: headers(operatorCode) });
    return data;
  },
  async listDecisions(params: ListParams & AccessAuditFilters): Promise<PageResponse<DecisionItem>> {
    const { operatorCode, serialNumber = '', page: currentPage, pageSize, ...filters } = params;
    const { data } = await http.get<{ items: DecisionItem[]; total: number; page: number; page_size: number }>(
      `/device-access/states/${encodeURIComponent(serialNumber)}/decisions`,
      { params: { page: currentPage, page_size: pageSize, ...auditQuery(filters) }, headers: headers(operatorCode) },
    );
    return page(data);
  },
  async listIdentitySnapshots(params: ListParams): Promise<PageResponse<AccessIdentitySnapshot>> {
    return listStateSection<AccessIdentitySnapshot>('identity-snapshots', params);
  },
  async listEvidence(params: ListParams): Promise<PageResponse<EvidenceItem>> {
    return listStateSection<EvidenceItem>('evidence', params);
  },
  async listNotifications(params: ListParams): Promise<PageResponse<AccessNotificationItem>> {
    return listStateSection<AccessNotificationItem>('notifications', params);
  },
  async listManualOperations(params: ListParams): Promise<PageResponse<AccessManualOperationItem>> {
    return listStateSection<AccessManualOperationItem>('manual-operations', params);
  },
  async archiveDecision(operatorCode: string, decisionId: string, reason: string): Promise<void> {
    await http.post(`/device-access/decisions/${encodeURIComponent(decisionId)}/archive`, { reason }, { headers: headers(operatorCode) });
  },
  async restoreDecision(operatorCode: string, decisionId: string): Promise<void> {
    await http.post(`/device-access/decisions/${encodeURIComponent(decisionId)}/restore`, undefined, { headers: headers(operatorCode) });
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
  async updatePolicyDraft(input: { operatorCode: string; versionId: string; policy: CompiledAccessPolicy }): Promise<PolicyVersion> {
    const { data } = await http.put<PolicyVersion>(`/device-access/policies/${encodeURIComponent(input.versionId)}`, {
      policy: input.policy,
    }, { headers: headers(input.operatorCode) });
    return data;
  },
  async publishPolicy(operatorCode: string, versionId: string): Promise<void> {
	await http.post(`/device-access/policies/${versionId}/publish`, undefined, { headers: headers(operatorCode) });
  },
  async getPolicyDifference(operatorCode: string, versionId: string, baseVersionId?: string): Promise<PolicyDifference> {
    const { data } = await http.get<PolicyDifference>(`/device-access/policies/${encodeURIComponent(versionId)}/difference`, {
      params: baseVersionId ? { base_version_id: baseVersionId } : {}, headers: headers(operatorCode),
    });
    return data;
  },
  async rollbackPolicy(operatorCode: string, versionId: string): Promise<PolicyVersion> {
    const { data } = await http.post<PolicyVersion>(`/device-access/policies/${encodeURIComponent(versionId)}/rollback`, undefined, {
      headers: headers(operatorCode),
    });
    return data;
  },
  async deletePolicyDraft(operatorCode: string, versionId: string): Promise<void> {
    await http.delete(`/device-access/policies/${versionId}`, { headers: headers(operatorCode) });
  },
  async listEntries(params: ListParams & { entryType?: AccessListType; productName?: string; status?: 'active' | 'disabled' }): Promise<PageResponse<AccessListItem>> {
    const { operatorCode, entryType, productName, status, ...query } = params;
    const { data } = await http.get<{ items: AccessListItem[]; total: number; page: number; page_size: number }>(
      '/device-access/access-list', { params: { ...listQuery(query), entry_type: entryType, product_name: productName, status }, headers: headers(operatorCode) },
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
  async upsertEntries(input: { operatorCode: string; entryType: AccessListType; serialNumbers: string[]; reason: string }): Promise<void> {
    await http.post('/device-access/access-list', {
      entries: input.serialNumbers.map((serialNumber) => ({
        type: input.entryType,
        identity_type: 'serial_number',
        identity_value: serialNumber,
        status: 'active',
        reason: input.reason,
      })),
    }, { headers: headers(input.operatorCode) });
  },
  async disableEntries(input: { operatorCode: string; entryType: AccessListType; serialNumbers: string[]; reason: string }): Promise<void> {
    await http.post('/device-access/access-list/batch-disable', {
      entry_type: input.entryType,
      serial_numbers: input.serialNumbers,
      reason: input.reason,
    }, { headers: headers(input.operatorCode) });
  },
  async downloadAccessListTemplate(operatorCode: string): Promise<Blob> {
    const { data } = await http.get<Blob>('/device-access/access-list/template', {
      headers: headers(operatorCode),
      responseType: 'blob',
    });
    return data;
  },
  async downloadRuleDimensionTemplate(operatorCode: string, dimension: RuleDimension): Promise<Blob> {
    const { data } = await http.get<Blob>('/device-access/import-templates/rule_dimension', {
      params: { dimension }, headers: headers(operatorCode), responseType: 'blob',
    });
    return data;
  },
  async previewAccessListImport(input: {
    operatorCode: string;
    entryType: AccessListType;
    mode: ImportMode;
    failurePolicy: ImportFailurePolicy;
    file: File;
    idempotencyKey: string;
  }): Promise<AccessListImportPreview> {
    const form = new FormData();
    form.append('file', input.file);
    form.append('import_type', 'access_list');
    form.append('entry_type', input.entryType);
    form.append('mode', input.mode);
    form.append('failure_policy', input.failurePolicy);
    const { data } = await http.post<AccessListImportPreview>('/device-access/imports/preview', form, {
      headers: {
        ...headers(input.operatorCode),
        'Content-Type': 'multipart/form-data',
        'Idempotency-Key': input.idempotencyKey,
      },
    });
    return data;
  },
  async previewRuleDimensionImport(input: {
    operatorCode: string;
    policyVersionId: string;
    ruleId: string;
    dimension: RuleDimension;
    mode: ImportMode;
    failurePolicy: ImportFailurePolicy;
    file: File;
    idempotencyKey: string;
  }): Promise<RuleDimensionImportPreview> {
    const form = new FormData();
    form.append('file', input.file);
    form.append('import_type', 'rule_dimension');
    form.append('target_policy_version_id', input.policyVersionId);
    form.append('target_rule_id', input.ruleId);
    form.append('dimension', input.dimension);
    form.append('mode', input.mode);
    form.append('failure_policy', input.failurePolicy);
    const { data } = await http.post<RuleDimensionImportPreview>('/device-access/imports/preview', form, {
      headers: {
        ...headers(input.operatorCode),
        'Content-Type': 'multipart/form-data',
        'Idempotency-Key': input.idempotencyKey,
      },
    });
    return data;
  },
  async exportRuleDimension(input: { operatorCode: string; policyVersionId: string; ruleId: string; dimension: RuleDimension }): Promise<Blob> {
    const { data } = await http.get<Blob>(
      `/device-access/policies/${encodeURIComponent(input.policyVersionId)}/rules/${encodeURIComponent(input.ruleId)}/dimensions/${input.dimension}/export`,
      { headers: headers(input.operatorCode), responseType: 'blob' },
    );
    return data;
  },
  async clearRuleDimension(input: { operatorCode: string; policyVersionId: string; ruleId: string; dimension: RuleDimension }): Promise<AccessListImportBatch> {
    const { data } = await http.delete<AccessListImportBatch>(
      `/device-access/policies/${encodeURIComponent(input.policyVersionId)}/rules/${encodeURIComponent(input.ruleId)}/dimensions/${input.dimension}`,
      { headers: headers(input.operatorCode) },
    );
    return data;
  },
  async listAccessListImports(params: { operatorCode: string; page: number; pageSize: number }): Promise<PageResponse<AccessListImportBatch>> {
    const { operatorCode, ...query } = params;
    const { data } = await http.get<{ items: AccessListImportBatch[]; total: number; page: number; page_size: number }>(
      '/device-access/imports', { params: { page: query.page, page_size: query.pageSize, import_type: 'access_list' }, headers: headers(operatorCode) },
    );
    return page(data);
  },
  async listRuleDimensionImports(params: { operatorCode: string; policyVersionId: string; ruleId: string; dimension: RuleDimension; page: number; pageSize: number }): Promise<PageResponse<AccessListImportBatch>> {
    const { operatorCode, ...query } = params;
    const { data } = await http.get<{ items: AccessListImportBatch[]; total: number; page: number; page_size: number }>(
      '/device-access/imports', { params: {
        page: query.page, page_size: query.pageSize, import_type: 'rule_dimension',
        target_policy_version_id: query.policyVersionId, target_rule_id: query.ruleId, dimension: query.dimension,
      }, headers: headers(operatorCode) },
    );
    return page(data);
  },
  async getAccessListImport(operatorCode: string, batchId: string): Promise<AccessListImportDetail> {
    const { data } = await http.get<AccessListImportDetail>(`/device-access/imports/${encodeURIComponent(batchId)}`, { headers: headers(operatorCode) });
    return data;
  },
  async listAccessListImportErrors(operatorCode: string, batchId: string): Promise<AccessListImportRow[]> {
    const { data } = await http.get<{ rows: AccessListImportRow[] }>(`/device-access/imports/${encodeURIComponent(batchId)}/errors`, { headers: headers(operatorCode) });
    return data.rows ?? [];
  },
  async commitAccessListImport(input: { operatorCode: string; batchId: string; confirmReplaceWithInvalid?: boolean }): Promise<AccessListImportBatch> {
    const { data } = await http.post<AccessListImportBatch>(`/device-access/imports/${encodeURIComponent(input.batchId)}/commit`, {
      confirm_replace_with_invalid: Boolean(input.confirmReplaceWithInvalid),
    }, { headers: headers(input.operatorCode) });
    return data;
  },
  async rollbackAccessListImport(input: { operatorCode: string; batchId: string }): Promise<AccessListImportBatch> {
    const { data } = await http.post<AccessListImportBatch>(`/device-access/imports/${encodeURIComponent(input.batchId)}/rollback`, undefined, { headers: headers(input.operatorCode) });
    return data;
  },
  async listCandidates(params: ListParams & { reviewStatus?: string; candidateId?: string }): Promise<PageResponse<CandidateItem>> {
    const { operatorCode, reviewStatus, candidateId, ...query } = params;
    const { data } = await http.get<{ items: CandidateItem[]; total: number; page: number; page_size: number }>(
      '/device-access/candidates', { params: { ...listQuery(query), review_status: reviewStatus, candidate_id: candidateId }, headers: headers(operatorCode) },
    );
    return page(data);
  },
  async reviewCandidate(input: { operatorCode: string; candidateId: string; outcome: 'allow' | 'deny'; reason: string }): Promise<void> {
    await http.post(`/device-access/candidates/${input.candidateId}/review`, { outcome: input.outcome, reason: input.reason }, { headers: headers(input.operatorCode) });
  },
  async listActions(params: ListParams & AccessAuditFilters & { status?: ActionStatus }): Promise<PageResponse<AccessAction>> {
    const { operatorCode, status, page: currentPage, pageSize, serialNumber, ...filters } = params;
    const { data } = await http.get<{ items: AccessAction[]; total: number; page: number; page_size: number }>(
      '/device-access/actions', { params: { ...listQuery({ page: currentPage, pageSize, serialNumber }), ...auditQuery(filters), status }, headers: headers(operatorCode) },
    );
    return page(data);
  },
  async listActionAttempts(operatorCode: string, actionId: string): Promise<AccessActionAttempt[]> {
    const { data } = await http.get<{ items: AccessActionAttempt[] }>(
      `/device-access/actions/${actionId}/attempts`, { headers: headers(operatorCode) },
    );
    return data.items ?? [];
  },
  async retryAction(operatorCode: string, actionId: string, reason: string): Promise<void> {
    await http.post(`/device-access/actions/${actionId}/retry`, { reason }, { headers: headers(operatorCode) });
  },
};
