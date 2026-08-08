import http from '../http';

export type StorageTargetType =
  | 'filesystem'
  | 'minio'
  | 'database'
  | 'redis'
  | 'nats'
  | 'monitoring'
  | 'application';

export type StorageWriteScope =
  | 'upload'
  | 'log'
  | 'backup'
  | 'report'
  | 'trace'
  | 'pm'
  | 'mr'
  | 'all';

export type StorageProtectionState = 'normal' | 'warning' | 'blocked' | 'unknown';
export type StorageUnknownBehavior = 'allow_with_alarm' | 'block_new_uploads';

export const UNIFIED_STORAGE_TARGET = {
  targetType: 'filesystem' as const,
  targetId: 'root',
  mountpoint: '/',
  writeScope: 'all' as const,
};

export interface StorageProtectionPolicy {
  id: string;
  targetType: StorageTargetType;
  targetId: string;
  writeScope: StorageWriteScope;
  enabled: boolean;
  warnUsedPercent: number;
  blockUsedPercent: number;
  recoverUsedPercent: number;
  checkIntervalSeconds: number;
  unknownBehavior: StorageUnknownBehavior;
  currentState: StorageProtectionState;
  stateObservations: number;
  lastObservedRatio?: number;
  lastObservedAt?: string;
  lastStateChangedAt?: string;
  updatedBy?: string;
  version?: number;
  createdAt?: string;
  updatedAt?: string;
}

export interface StorageProtectionPolicyPayload {
  targetType: StorageTargetType;
  targetId: string;
  writeScope: StorageWriteScope;
  enabled: boolean;
  warnUsedPercent: number;
  blockUsedPercent: number;
  recoverUsedPercent: number;
  checkIntervalSeconds: number;
  unknownBehavior: StorageUnknownBehavior;
  updatedBy?: string;
}

export interface StorageProtectionEvent {
  policyId: string;
  targetType: StorageTargetType;
  targetId: string;
  writeScope: StorageWriteScope;
  previousState?: StorageProtectionState;
  newState: StorageProtectionState;
  reason: string;
  observedRatio?: number;
  policyVersion: number;
  operatorId: string;
  createdAt: string;
}

export interface StorageProtectionTarget {
  targetType: StorageTargetType;
  targetId: string;
  label?: string;
  mountpoint?: string;
  mountPath?: string;
  components: string[];
  sourcePaths: string[];
  protectedPaths: string[];
  capacityBytes: number;
  usedBytes: number;
  usedRatio: number;
  available: boolean;
  reason?: string;
  observedAt?: string;
  currentState: StorageProtectionState;
  writeScopes: StorageWriteScope[];
}

interface BackendStorageProtectionPolicy {
  id: string;
  target_type: StorageTargetType;
  target_id: string;
  write_scope: StorageWriteScope;
  enabled: boolean;
  warn_used_percent: number;
  block_used_percent: number;
  recover_used_percent: number;
  check_interval_seconds: number;
  unknown_behavior: StorageUnknownBehavior;
  current_state: StorageProtectionState;
  state_observations: number;
  last_observed_ratio?: number;
  last_observed_at?: string;
  last_state_changed_at?: string;
  updated_by?: string;
  version?: number;
  created_at?: string;
  updated_at?: string;
}

interface BackendStorageProtectionEvent {
  policy_id: string;
  target_type: StorageTargetType;
  target_id: string;
  write_scope: StorageWriteScope;
  previous_state?: StorageProtectionState;
  new_state: StorageProtectionState;
  reason: string;
  observed_ratio?: number;
  policy_version: number;
  operator_id: string;
  created_at: string;
}

interface BackendStorageProtectionTarget {
  target_type: StorageTargetType;
  target_id: string;
  label?: string;
  mountpoint?: string;
  mount_path?: string;
  components?: string[];
  source_paths?: string[];
  protected_paths?: string[];
  capacity_bytes: number;
  used_bytes: number;
  used_ratio: number;
  available: boolean;
  reason?: string;
  observed_at?: string;
  current_state: StorageProtectionState;
  write_scopes: StorageWriteScope[];
}

function mapPolicy(policy: BackendStorageProtectionPolicy): StorageProtectionPolicy {
  return {
    id: policy.id,
    targetType: policy.target_type,
    targetId: policy.target_id,
    writeScope: policy.write_scope,
    enabled: policy.enabled,
    warnUsedPercent: policy.warn_used_percent,
    blockUsedPercent: policy.block_used_percent,
    recoverUsedPercent: policy.recover_used_percent,
    checkIntervalSeconds: policy.check_interval_seconds,
    unknownBehavior: policy.unknown_behavior,
    currentState: policy.current_state,
    stateObservations: policy.state_observations,
    lastObservedRatio: policy.last_observed_ratio,
    lastObservedAt: policy.last_observed_at,
    lastStateChangedAt: policy.last_state_changed_at,
    updatedBy: policy.updated_by,
    version: policy.version,
    createdAt: policy.created_at,
    updatedAt: policy.updated_at,
  };
}

function mapPayload(payload: StorageProtectionPolicyPayload) {
  return {
    // Keep the API defensive even if an older page or cached form submits a
    // logical component target. Capacity policy is global to the one physical
    // filesystem in this deployment.
    target_type: UNIFIED_STORAGE_TARGET.targetType,
    target_id: UNIFIED_STORAGE_TARGET.targetId,
    write_scope: UNIFIED_STORAGE_TARGET.writeScope,
    enabled: payload.enabled,
    warn_used_percent: payload.warnUsedPercent,
    block_used_percent: payload.blockUsedPercent,
    recover_used_percent: payload.recoverUsedPercent,
    check_interval_seconds: payload.checkIntervalSeconds,
    unknown_behavior: payload.unknownBehavior,
    updated_by: payload.updatedBy,
  };
}

function mapEvent(event: BackendStorageProtectionEvent): StorageProtectionEvent {
  return {
    policyId: event.policy_id,
    targetType: event.target_type,
    targetId: event.target_id,
    writeScope: event.write_scope,
    previousState: event.previous_state,
    newState: event.new_state,
    reason: event.reason,
    observedRatio: event.observed_ratio,
    policyVersion: event.policy_version,
    operatorId: event.operator_id,
    createdAt: event.created_at,
  };
}

function mapTarget(target: BackendStorageProtectionTarget): StorageProtectionTarget {
  return {
    targetType: target.target_type,
    targetId: target.target_id,
    label: target.label,
    mountpoint: target.mountpoint,
    mountPath: target.mount_path,
    components: target.components ?? [],
    sourcePaths: target.source_paths ?? [],
    protectedPaths: target.protected_paths ?? [],
    capacityBytes: target.capacity_bytes,
    usedBytes: target.used_bytes,
    usedRatio: target.used_ratio,
    available: target.available,
    reason: target.reason,
    observedAt: target.observed_at,
    currentState: target.current_state,
    writeScopes: target.write_scopes ?? [],
  };
}

export const storageProtectionApi = {
  async getPolicies() {
    const { data } = await http.get<BackendStorageProtectionPolicy[]>('/admin/storage-protection/policies');
    return (data ?? []).map(mapPolicy);
  },

  async getTargets() {
    const { data } = await http.get<BackendStorageProtectionTarget[]>('/admin/storage-protection/targets');
    return (data ?? []).map(mapTarget);
  },

  async getEvents(limit = 5) {
    const { data } = await http.get<BackendStorageProtectionEvent[]>('/admin/storage-protection/events', {
      params: { limit },
    });
    return (data ?? []).map(mapEvent);
  },

  async savePolicy(payload: StorageProtectionPolicyPayload) {
    const { data } = await http.post<BackendStorageProtectionPolicy>(
      '/admin/storage-protection/policies',
      mapPayload(payload),
    );
    return mapPolicy(data);
  },

  async updatePolicy(id: string, payload: StorageProtectionPolicyPayload) {
    const { data } = await http.put<BackendStorageProtectionPolicy>(
      `/admin/storage-protection/policies/${encodeURIComponent(id)}`,
      mapPayload(payload),
    );
    return mapPolicy(data);
  },
};
