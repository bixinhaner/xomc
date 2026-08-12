import type { MapBounds } from './map';

export type GeofenceRuntimeMode = 'off' | 'observe' | 'enforce';
export type GeofenceCarrier = 'cmcc' | 'ctcc' | 'cucc';
export type GeofenceRuleType =
  | 'polygon_allow_zone'
  | 'baseline_radius';
export type GeofenceDefinitionStatus =
  | 'draft'
  | 'enabled'
  | 'disabled'
  | 'archived';
export type GeofenceLifecycleTarget =
  | 'enabled'
  | 'disabled'
  | 'archived';
export type GeofenceVersionStatus =
  | 'draft'
  | 'published'
  | 'superseded';
export type GeofenceBindingStatus =
  | 'pending'
  | 'active'
  | 'suspended'
  | 'removed';
export type GeofenceJobStatus =
  | 'pending'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'zombie'
  | 'canceled';
export type GeofenceBatchItemStatus =
  | 'pending'
  | 'succeeded'
  | 'skipped'
  | 'failed';
export type GeofenceBindingInputKind = 'device_id' | 'device_sn';
export type GeofenceBindingDecision = 'eligible' | 'move' | 'skipped';
export type GeofenceConfirmedState = 'unknown' | 'inside' | 'outside';
export type GeofenceCandidateState = 'exit' | 'reentry';
export type GeofenceEvaluationHealth = 'healthy' | 'stale' | 'failed';
export type GeofenceExitAction =
  | 'notify_only'
  | 'manual_review'
  | 'deactivate';

export interface GeofencePolygonGeometry {
  type: 'Polygon';
  coordinates: number[][][];
}

export interface GeofenceCircleGeometry {
  type: 'Circle';
  center: [number, number];
  radiusMeters: number;
  source?: string;
  sourceObservationVersion?: number;
}

export type GeofenceGeometry =
  | GeofencePolygonGeometry
  | GeofenceCircleGeometry;

export interface GeofencePolicy {
  exitAction: GeofenceExitAction;
  exitToleranceMeters?: number;
  reentryToleranceMeters?: number;
  exitConsecutiveSamples?: number;
  reentryConsecutiveSamples?: number;
  sampleMaxAgeSeconds?: number;
  maxJumpDistanceMeters?: number;
  maxImpliedSpeedMps?: number;
  maxDeviceClockSkewSeconds?: number;
  minimumStateDurationSeconds?: number;
}

export interface GeofenceBoundingBox {
  minLongitude: number;
  minLatitude: number;
  maxLongitude: number;
  maxLatitude: number;
}

export interface GeofenceCarrierSetting {
  carrier: GeofenceCarrier | string;
  mode: GeofenceRuntimeMode;
  effectiveMode: GeofenceRuntimeMode;
  defaultBaselineRadiusMeters: number;
  updatedBy?: string;
  updatedAt: string;
}

export interface GeofenceSettings {
  systemMode: GeofenceRuntimeMode;
  carriers: GeofenceCarrierSetting[];
}

export interface GeofenceAvailability {
  enabled: boolean;
}

export interface UpdateGeofenceSettingsInput {
  systemMode: GeofenceRuntimeMode;
  carriers: Array<{
    carrier: GeofenceCarrier | string;
    mode: GeofenceRuntimeMode;
    defaultBaselineRadiusMeters: number;
  }>;
}

export interface GeofenceSettingsPreview {
  current: GeofenceSettings;
  proposed: GeofenceSettings;
  enabledGeofences: number;
  activeBindings: number;
  newlyObservedDevices: number;
}

export interface GeofenceDefinition {
  id: string;
  name: string;
  carrier: GeofenceCarrier | string;
  ruleType: GeofenceRuleType;
  ownerDeviceId?: string;
  status: GeofenceDefinitionStatus;
  currentVersionId?: string;
  createdBy: string;
  updatedBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface GeofenceVersion {
  id: string;
  geofenceId: string;
  version: number;
  status: GeofenceVersionStatus;
  geometry: GeofenceGeometry;
  boundingBox: GeofenceBoundingBox;
  policy: GeofencePolicy;
  createdBy: string;
  publishedBy?: string;
  createdAt: string;
  publishedAt?: string;
}

export interface GeofenceMapDefinition {
  definition: GeofenceDefinition;
  currentVersion: GeofenceVersion | null;
}

export interface GeofenceMapDefinitionList {
  items: GeofenceMapDefinition[];
}

export interface GeofenceDefinitionFilter {
  carrier?: string;
  status?: GeofenceDefinitionStatus;
  name?: string;
}

export interface GeofenceMapFilter extends GeofenceDefinitionFilter {
  bounds?: MapBounds;
}

export interface GeofenceVersionList {
  items: GeofenceVersion[];
}

export interface GeofenceLifecycleImpact {
  geofenceId: string;
  currentVersionId?: string;
  currentStatus: GeofenceDefinitionStatus;
  targetStatus: GeofenceDefinitionStatus;
  bindingCount: number;
  deviceCount: number;
  activeBatchJobCount: number;
  previewFingerprint: string;
}

export interface GeofenceBinding {
  id: string;
  deviceId: string;
  geofenceId: string;
  ruleType: GeofenceRuleType;
  status: GeofenceBindingStatus;
  bindSource: string;
  boundBy: string;
  boundAt: string;
  removedBy?: string;
  removedAt?: string;
  removeReason?: string;
}

export interface GeofenceBindingDetail extends GeofenceBinding {
  deviceSN: string;
  deviceName: string;
  deviceCarrier: string;
  deviceGroupId?: string;
  deviceGroupName?: string;
  hasLocation: boolean;
  evaluation: GeofenceBindingEvaluation;
}

export interface GeofenceBindingEvaluation {
  confirmedState: GeofenceConfirmedState;
  candidateState?: GeofenceCandidateState;
  candidateCount: number;
  lastObservationVersion?: number;
  lastObservedAt?: string;
  lastDistanceToBoundary?: number;
  evaluationHealth: GeofenceEvaluationHealth;
  errorCode?: string;
}

export interface GeofenceBindingFilter {
  status?: Exclude<GeofenceBindingStatus, 'pending'> | 'current';
  keyword?: string;
  page?: number;
  pageSize?: number;
}

export interface GeofenceBindingPage {
  items: GeofenceBindingDetail[];
  total: number;
  page: number;
  pageSize: number;
}

export interface GeofenceBindingInputs {
  deviceIds?: string[];
  deviceSNs?: string[];
}

export interface GeofenceBindingPreviewItem {
  inputKey: string;
  input: string;
  deviceId?: string;
  deviceSN?: string;
  decision: GeofenceBindingDecision;
  reasonCode?: string;
  sourceBindingId?: string;
  sourceGeofenceId?: string;
}

export interface GeofenceManualBindingPreview {
  geofenceId: string;
  geofenceVersionId: string;
  ruleType: GeofenceRuleType;
  inputCount: number;
  eligibleCount: number;
  moveCount: number;
  skippedCount: number;
  items: GeofenceBindingPreviewItem[];
  previewFingerprint: string;
}

export interface GeofenceCandidateDevice {
  id: string;
  serialNumber: string;
  name?: string;
  latitude?: number;
  longitude?: number;
}

export interface GeofenceCandidatePreview {
  geofenceId: string;
  versionId: string;
  inside: GeofenceCandidateDevice[];
  outside: GeofenceCandidateDevice[];
  noLocation: GeofenceCandidateDevice[];
}

export interface GeofenceControlAction {
  id: string;
  deviceSN: string;
  commandKey: string;
  actionType: 'deactivate' | 'activate';
  status:
    | 'pending'
    | 'executing'
    | 'verifying'
    | 'verified'
    | 'partial_failed'
    | 'failed';
  contractVersion: number;
  beforeState: GeofenceControlParameterState[];
  requestedState: GeofenceControlParameterState[];
  terminalState: GeofenceControlParameterState[];
  verifiedState: GeofenceControlParameterState[];
  verificationAttempt: number;
  lastError?: string;
  createdAt: string;
  completedAt?: string;
}

export interface GeofenceControlParameterState {
  path: string;
  value: string;
  role?: 'admin' | 'admin_rf' | 'rf' | 'ipsec' | 'op_state';
}

export interface GeofenceBatchProgress {
  total: number;
  pending: number;
  succeeded: number;
  skipped: number;
  failed: number;
}

export interface GeofenceManualBindJob {
  id: string;
  jobType: string;
  status: GeofenceJobStatus;
  geofenceId: string;
  requestedBy: string;
  reason: string;
  attempt: number;
  maxAttempts: number;
  errorMessage?: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
  progress: GeofenceBatchProgress;
}

export interface GeofenceBatchJobAccepted {
  jobId: string;
}

export interface GeofenceBatchItem {
  id: string;
  jobId: string;
  geofenceId: string;
  inputKey: string;
  inputKind: GeofenceBindingInputKind;
  inputValue: string;
  deviceId?: string;
  deviceSN?: string;
  status: GeofenceBatchItemStatus;
  reasonCode?: string;
  errorMessage?: string;
  bindingId?: string;
  attempt: number;
  startedAt?: string;
  finishedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface GeofenceBatchItemFilter {
  status?: GeofenceBatchItemStatus;
  page?: number;
  pageSize?: number;
}

export interface GeofenceBatchItemPage {
  items: GeofenceBatchItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface CreateGeofenceDefinitionInput {
  name: string;
  carrier: string;
  ruleType: GeofenceRuleType;
  ownerDeviceId?: string;
  geometry: GeofenceGeometry;
  policy: GeofencePolicy;
}

export interface CreateGeofenceDefinitionResult {
  definition: GeofenceDefinition;
  draftVersion: GeofenceVersion;
}

export interface CreateGeofenceVersionInput {
  geometry: GeofenceGeometry;
  policy: GeofencePolicy;
}

export interface GeofenceTransitionInput {
  reason: string;
  previewFingerprint: string;
}

export interface CreateGeofenceManualBindJobInput
  extends GeofenceBindingInputs {
  previewFingerprint: string;
  reason: string;
  scheduledAt?: string;
}
