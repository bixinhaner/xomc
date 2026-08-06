import http from '../http';
import {
  filenameFromContentDisposition,
  saveBlob,
} from '../../utils/saveBlob';
import type {
  GeofenceBatchItem,
  GeofenceBatchItemFilter,
  GeofenceBatchItemPage,
  GeofenceBatchJobAccepted,
  GeofenceAvailability,
  GeofenceBinding,
  GeofenceBindingDetail,
  GeofenceBindingFilter,
  GeofenceBindingInputs,
  GeofenceBindingPage,
  GeofenceCandidatePreview,
  GeofenceControlAction,
  GeofenceBoundingBox,
  CreateGeofenceDefinitionInput,
  CreateGeofenceDefinitionResult,
  CreateGeofenceManualBindJobInput,
  CreateGeofenceVersionInput,
  GeofenceDefinition,
  GeofenceDefinitionFilter,
  GeofenceGeometry,
  GeofenceLifecycleImpact,
  GeofenceLifecycleTarget,
  GeofenceManualBindJob,
  GeofenceManualBindingPreview,
  GeofenceMapDefinition,
  GeofenceMapDefinitionList,
  GeofenceMapFilter,
  GeofencePolicy,
  GeofenceSettings,
  GeofenceSettingsPreview,
  GeofenceTransitionInput,
  GeofenceVersion,
  GeofenceVersionList,
  UpdateGeofenceSettingsInput,
} from '../../types/geofence';

interface GeofenceCarrierSettingRaw {
  carrier: string;
  mode: GeofenceSettings['systemMode'];
  effective_mode: GeofenceSettings['systemMode'];
  default_baseline_radius_meters: number;
  updated_by?: string;
  updated_at: string;
}

interface GeofenceSettingsRaw {
  system_mode: GeofenceSettings['systemMode'];
  carriers?: GeofenceCarrierSettingRaw[];
}

interface GeofenceDefinitionRaw {
  id: string;
  name: string;
  carrier: string;
  rule_type: GeofenceDefinition['ruleType'];
  owner_device_id?: string;
  status: GeofenceDefinition['status'];
  current_version_id?: string;
  created_by: string;
  updated_by: string;
  created_at: string;
  updated_at: string;
}

interface GeofenceBoundingBoxRaw {
  min_longitude: number;
  min_latitude: number;
  max_longitude: number;
  max_latitude: number;
}

interface GeofencePolicyRaw {
  exit_action: GeofencePolicy['exitAction'];
  exit_tolerance_meters?: number;
  reentry_tolerance_meters?: number;
  exit_consecutive_samples?: number;
  reentry_consecutive_samples?: number;
  sample_max_age_seconds?: number;
  max_jump_distance_meters?: number;
  max_implied_speed_mps?: number;
  max_device_clock_skew_seconds?: number;
  minimum_state_duration_seconds?: number;
}

interface GeofenceVersionRaw {
  id: string;
  geofence_id: string;
  version: number;
  status: GeofenceVersion['status'];
  geometry: GeofenceGeometry;
  bounding_box: GeofenceBoundingBoxRaw;
  policy: GeofencePolicyRaw;
  created_by: string;
  published_by?: string;
  created_at: string;
  published_at?: string;
}

interface GeofenceMapDefinitionRaw {
  definition: GeofenceDefinitionRaw;
  current_version?: GeofenceVersionRaw | null;
}

interface GeofenceBindingRaw {
  id: string;
  device_id: string;
  geofence_id: string;
  rule_type: GeofenceBinding['ruleType'];
  status: GeofenceBinding['status'];
  bind_source: string;
  bound_by: string;
  bound_at: string;
  removed_by?: string;
  removed_at?: string;
  remove_reason?: string;
}

interface GeofenceBindingDetailRaw extends GeofenceBindingRaw {
  device_sn: string;
  device_name: string;
  device_carrier: string;
  device_group_id?: string;
  device_group_name?: string;
  has_location: boolean;
  evaluation: {
    confirmed_state: GeofenceBindingDetail['evaluation']['confirmedState'];
    candidate_state?: GeofenceBindingDetail['evaluation']['candidateState'];
    candidate_count: number;
    last_observation_version?: number;
    last_observed_at?: string;
    last_distance_to_boundary?: number;
    evaluation_health: GeofenceBindingDetail['evaluation']['evaluationHealth'];
    error_code?: string;
  };
}

interface GeofenceBindingPageRaw {
  items?: GeofenceBindingDetailRaw[];
  total: number;
  page: number;
  page_size: number;
}

interface GeofenceSettingsPreviewRaw {
  current: GeofenceSettingsRaw;
  proposed: GeofenceSettingsRaw;
  enabled_geofences: number;
  active_bindings: number;
  newly_observed_devices: number;
}

interface GeofenceLifecycleImpactRaw {
  geofence_id: string;
  current_version_id?: string;
  current_status: GeofenceLifecycleImpact['currentStatus'];
  target_status: GeofenceLifecycleImpact['targetStatus'];
  binding_count: number;
  device_count: number;
  active_batch_job_count: number;
  preview_fingerprint: string;
}

interface GeofenceBindingPreviewItemRaw {
  input_key: string;
  input: string;
  device_id?: string;
  device_sn?: string;
  decision: GeofenceManualBindingPreview['items'][number]['decision'];
  reason_code?: string;
  source_binding_id?: string;
  source_geofence_id?: string;
}

interface GeofenceManualBindingPreviewRaw {
  geofence_id: string;
  geofence_version_id: string;
  rule_type: GeofenceManualBindingPreview['ruleType'];
  input_count: number;
  eligible_count: number;
  move_count: number;
  skipped_count: number;
  items?: GeofenceBindingPreviewItemRaw[];
  preview_fingerprint: string;
}

interface GeofenceCandidateDeviceRaw {
  id: string;
  serial_number: string;
  name?: string;
  latitude?: number;
  longitude?: number;
}

interface GeofenceCandidatePreviewRaw {
  geofence_id: string;
  version_id: string;
  inside?: GeofenceCandidateDeviceRaw[];
  outside?: GeofenceCandidateDeviceRaw[];
  no_location?: GeofenceCandidateDeviceRaw[];
}

interface GeofenceControlActionRaw {
  id: string;
  device_sn: string;
  action_key: string;
  action_type: GeofenceControlAction['actionType'];
  status: GeofenceControlAction['status'];
  before_state?: Array<{ path: string; value: string }>;
  requested_state?: Array<{ path: string; value: string }>;
  verified_state?: Array<{ path: string; value: string }>;
  last_error?: string;
  created_at: string;
  completed_at?: string;
}

interface GeofenceManualBindJobRaw {
  id: string;
  job_type: string;
  status: GeofenceManualBindJob['status'];
  geofence_id: string;
  requested_by: string;
  reason: string;
  attempt: number;
  max_attempts: number;
  error_message?: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  progress: GeofenceManualBindJob['progress'];
}

interface GeofenceBatchItemRaw {
  id: string;
  job_id: string;
  geofence_id: string;
  input_key: string;
  input_kind: GeofenceBatchItem['inputKind'];
  input_value: string;
  device_id?: string;
  device_sn?: string;
  status: GeofenceBatchItem['status'];
  reason_code?: string;
  error_message?: string;
  binding_id?: string;
  attempt: number;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
}

interface GeofenceBatchItemPageRaw {
  items?: GeofenceBatchItemRaw[];
  total: number;
  page: number;
  page_size: number;
}

function mapSettings(raw: GeofenceSettingsRaw): GeofenceSettings {
  return {
    systemMode: raw.system_mode,
    carriers: (raw.carriers ?? []).map((setting) => ({
      carrier: setting.carrier,
      mode: setting.mode,
      effectiveMode: setting.effective_mode,
      defaultBaselineRadiusMeters:
        setting.default_baseline_radius_meters,
      updatedBy: setting.updated_by,
      updatedAt: setting.updated_at,
    })),
  };
}

function mapDefinition(
  raw: GeofenceDefinitionRaw,
): GeofenceDefinition {
  return {
    id: raw.id,
    name: raw.name,
    carrier: raw.carrier,
    ruleType: raw.rule_type,
    ownerDeviceId: raw.owner_device_id,
    status: raw.status,
    currentVersionId: raw.current_version_id,
    createdBy: raw.created_by,
    updatedBy: raw.updated_by,
    createdAt: raw.created_at,
    updatedAt: raw.updated_at,
  };
}

function mapBoundingBox(
  raw: GeofenceBoundingBoxRaw,
): GeofenceBoundingBox {
  return {
    minLongitude: raw.min_longitude,
    minLatitude: raw.min_latitude,
    maxLongitude: raw.max_longitude,
    maxLatitude: raw.max_latitude,
  };
}

function mapPolicy(raw: GeofencePolicyRaw): GeofencePolicy {
  return {
    exitAction: raw.exit_action,
    exitToleranceMeters: raw.exit_tolerance_meters,
    reentryToleranceMeters: raw.reentry_tolerance_meters,
    exitConsecutiveSamples: raw.exit_consecutive_samples,
    reentryConsecutiveSamples: raw.reentry_consecutive_samples,
    sampleMaxAgeSeconds: raw.sample_max_age_seconds,
    maxJumpDistanceMeters: raw.max_jump_distance_meters,
    maxImpliedSpeedMps: raw.max_implied_speed_mps,
    maxDeviceClockSkewSeconds: raw.max_device_clock_skew_seconds,
    minimumStateDurationSeconds:
      raw.minimum_state_duration_seconds,
  };
}

function mapVersion(raw: GeofenceVersionRaw): GeofenceVersion {
  return {
    id: raw.id,
    geofenceId: raw.geofence_id,
    version: raw.version,
    status: raw.status,
    geometry: raw.geometry,
    boundingBox: mapBoundingBox(raw.bounding_box),
    policy: mapPolicy(raw.policy),
    createdBy: raw.created_by,
    publishedBy: raw.published_by,
    createdAt: raw.created_at,
    publishedAt: raw.published_at,
  };
}

function mapMapDefinition(
  raw: GeofenceMapDefinitionRaw,
): GeofenceMapDefinition {
  return {
    definition: mapDefinition(raw.definition),
    currentVersion: raw.current_version
      ? mapVersion(raw.current_version)
      : null,
  };
}

function mapBindingDetail(
  raw: GeofenceBindingDetailRaw,
): GeofenceBindingDetail {
  return {
    ...mapBinding(raw),
    deviceSN: raw.device_sn,
    deviceName: raw.device_name,
    deviceCarrier: raw.device_carrier,
    deviceGroupId: raw.device_group_id,
    deviceGroupName: raw.device_group_name,
    hasLocation: raw.has_location,
    evaluation: {
      confirmedState: raw.evaluation.confirmed_state,
      candidateState: raw.evaluation.candidate_state,
      candidateCount: raw.evaluation.candidate_count,
      lastObservationVersion:
        raw.evaluation.last_observation_version,
      lastObservedAt: raw.evaluation.last_observed_at,
      lastDistanceToBoundary:
        raw.evaluation.last_distance_to_boundary,
      evaluationHealth: raw.evaluation.evaluation_health,
      errorCode: raw.evaluation.error_code,
    },
  };
}

function mapBinding(raw: GeofenceBindingRaw): GeofenceBinding {
  return {
    id: raw.id,
    deviceId: raw.device_id,
    geofenceId: raw.geofence_id,
    ruleType: raw.rule_type,
    status: raw.status,
    bindSource: raw.bind_source,
    boundBy: raw.bound_by,
    boundAt: raw.bound_at,
    removedBy: raw.removed_by,
    removedAt: raw.removed_at,
    removeReason: raw.remove_reason,
  };
}

function mapManualBindJob(
  raw: GeofenceManualBindJobRaw,
): GeofenceManualBindJob {
  return {
    id: raw.id,
    jobType: raw.job_type,
    status: raw.status,
    geofenceId: raw.geofence_id,
    requestedBy: raw.requested_by,
    reason: raw.reason,
    attempt: raw.attempt,
    maxAttempts: raw.max_attempts,
    errorMessage: raw.error_message,
    createdAt: raw.created_at,
    startedAt: raw.started_at,
    finishedAt: raw.finished_at,
    progress: raw.progress,
  };
}

function mapBatchItem(raw: GeofenceBatchItemRaw): GeofenceBatchItem {
  return {
    id: raw.id,
    jobId: raw.job_id,
    geofenceId: raw.geofence_id,
    inputKey: raw.input_key,
    inputKind: raw.input_kind,
    inputValue: raw.input_value,
    deviceId: raw.device_id,
    deviceSN: raw.device_sn,
    status: raw.status,
    reasonCode: raw.reason_code,
    errorMessage: raw.error_message,
    bindingId: raw.binding_id,
    attempt: raw.attempt,
    startedAt: raw.started_at,
    finishedAt: raw.finished_at,
    createdAt: raw.created_at,
    updatedAt: raw.updated_at,
  };
}

function definitionParams(filter: GeofenceDefinitionFilter) {
  return {
    carrier: filter.carrier,
    status: filter.status,
    name: filter.name,
  };
}

function settingsBody(settings: UpdateGeofenceSettingsInput) {
  return {
    system_mode: settings.systemMode,
    carriers: settings.carriers.map((setting) => ({
      carrier: setting.carrier,
      mode: setting.mode,
      default_baseline_radius_meters:
        setting.defaultBaselineRadiusMeters,
    })),
  };
}

function policyBody(policy: GeofencePolicy): GeofencePolicyRaw {
  return {
    exit_action: policy.exitAction,
    exit_tolerance_meters: policy.exitToleranceMeters,
    reentry_tolerance_meters: policy.reentryToleranceMeters,
    exit_consecutive_samples: policy.exitConsecutiveSamples,
    reentry_consecutive_samples:
      policy.reentryConsecutiveSamples,
    sample_max_age_seconds: policy.sampleMaxAgeSeconds,
    max_jump_distance_meters: policy.maxJumpDistanceMeters,
    max_implied_speed_mps: policy.maxImpliedSpeedMps,
    max_device_clock_skew_seconds:
      policy.maxDeviceClockSkewSeconds,
    minimum_state_duration_seconds:
      policy.minimumStateDurationSeconds,
  };
}

function mapLifecycleImpact(
  raw: GeofenceLifecycleImpactRaw,
): GeofenceLifecycleImpact {
  return {
    geofenceId: raw.geofence_id,
    currentVersionId: raw.current_version_id,
    currentStatus: raw.current_status,
    targetStatus: raw.target_status,
    bindingCount: raw.binding_count,
    deviceCount: raw.device_count,
    activeBatchJobCount: raw.active_batch_job_count,
    previewFingerprint: raw.preview_fingerprint,
  };
}

function bindingInputsBody(inputs: GeofenceBindingInputs) {
  return {
    device_ids: inputs.deviceIds ?? [],
    device_sns: inputs.deviceSNs ?? [],
  };
}

function mapManualBindingPreview(
  raw: GeofenceManualBindingPreviewRaw,
): GeofenceManualBindingPreview {
  return {
    geofenceId: raw.geofence_id,
    geofenceVersionId: raw.geofence_version_id,
    ruleType: raw.rule_type,
    inputCount: raw.input_count,
    eligibleCount: raw.eligible_count,
    moveCount: raw.move_count,
    skippedCount: raw.skipped_count,
    items: (raw.items ?? []).map((item) => ({
      inputKey: item.input_key,
      input: item.input,
      deviceId: item.device_id,
      deviceSN: item.device_sn,
      decision: item.decision,
      reasonCode: item.reason_code,
      sourceBindingId: item.source_binding_id,
      sourceGeofenceId: item.source_geofence_id,
    })),
    previewFingerprint: raw.preview_fingerprint,
  };
}

function mapCandidatePreview(
  raw: GeofenceCandidatePreviewRaw,
): GeofenceCandidatePreview {
  const mapDevice = (device: GeofenceCandidateDeviceRaw) => ({
    id: device.id,
    serialNumber: device.serial_number,
    name: device.name,
    latitude: device.latitude,
    longitude: device.longitude,
  });
  return {
    geofenceId: raw.geofence_id,
    versionId: raw.version_id,
    inside: (raw.inside ?? []).map(mapDevice),
    outside: (raw.outside ?? []).map(mapDevice),
    noLocation: (raw.no_location ?? []).map(mapDevice),
  };
}

const lifecyclePath: Record<GeofenceLifecycleTarget, string> = {
  enabled: 'enable',
  disabled: 'disable',
  archived: 'archive',
};

export const geofenceApi = {
  async getAvailability(): Promise<GeofenceAvailability> {
    const { data } = await http.get<GeofenceAvailability>(
      '/geofences/availability',
    );
    return data;
  },

  async getSettings(): Promise<GeofenceSettings> {
    const { data } = await http.get<GeofenceSettingsRaw>(
      '/geofences/settings',
    );
    return mapSettings(data);
  },

  async previewSettings(
    settings: UpdateGeofenceSettingsInput,
  ): Promise<GeofenceSettingsPreview> {
    const { data } = await http.post<GeofenceSettingsPreviewRaw>(
      '/geofences/settings/preview',
      settingsBody(settings),
    );
    return {
      current: mapSettings(data.current),
      proposed: mapSettings(data.proposed),
      enabledGeofences: data.enabled_geofences,
      activeBindings: data.active_bindings,
      newlyObservedDevices: data.newly_observed_devices,
    };
  },

  async updateSettings(
    settings: UpdateGeofenceSettingsInput,
  ): Promise<GeofenceSettings> {
    const { data } = await http.put<GeofenceSettingsRaw>(
      '/geofences/settings',
      settingsBody(settings),
    );
    return mapSettings(data);
  },

  async listMapDefinitions(
    filter: GeofenceMapFilter = {},
  ): Promise<GeofenceMapDefinitionList> {
    const { data } = await http.get<{
      items?: GeofenceMapDefinitionRaw[];
    }>('/geofences/map', {
      params: {
        bounds: filter.bounds
          ? [
              filter.bounds.minLng,
              filter.bounds.maxLng,
              filter.bounds.minLat,
              filter.bounds.maxLat,
            ].join(',')
          : undefined,
        ...definitionParams(filter),
      },
    });
    return {
      items: (data.items ?? []).map(mapMapDefinition),
    };
  },

  async listDefinitions(
    filter: GeofenceDefinitionFilter = {},
  ): Promise<GeofenceDefinition[]> {
    const { data } = await http.get<GeofenceDefinitionRaw[]>(
      '/geofences',
      { params: definitionParams(filter) },
    );
    return (data ?? []).map(mapDefinition);
  },

  async getDefinition(id: string): Promise<GeofenceDefinition> {
    const { data } = await http.get<GeofenceDefinitionRaw>(
      `/geofences/${id}`,
    );
    return mapDefinition(data);
  },

  async renameDefinition(id: string, name: string): Promise<void> {
    await http.patch(`/geofences/${id}`, { name });
  },

  async createDefinition(
    input: CreateGeofenceDefinitionInput,
  ): Promise<CreateGeofenceDefinitionResult> {
    const { data } = await http.post<{
      definition: GeofenceDefinitionRaw;
      draft_version: GeofenceVersionRaw;
    }>('/geofences', {
      name: input.name,
      carrier: input.carrier,
      rule_type: input.ruleType,
      owner_device_id: input.ownerDeviceId,
      geometry: input.geometry,
      policy: policyBody(input.policy),
    });
    return {
      definition: mapDefinition(data.definition),
      draftVersion: mapVersion(data.draft_version),
    };
  },

  async listVersions(id: string): Promise<GeofenceVersionList> {
    const { data } = await http.get<{
      items?: GeofenceVersionRaw[];
    }>(`/geofences/${id}/versions`);
    return {
      items: (data.items ?? []).map(mapVersion),
    };
  },

  async createDraftVersion(
    id: string,
    input: CreateGeofenceVersionInput,
  ): Promise<GeofenceVersion> {
    const { data } = await http.post<GeofenceVersionRaw>(
      `/geofences/${id}/versions`,
      {
        geometry: input.geometry,
        policy: policyBody(input.policy),
      },
    );
    return mapVersion(data);
  },

  async publishDraft(
    id: string,
    versionId: string,
  ): Promise<{ status: 'executed' }> {
    const { data } = await http.post<{ status: 'executed' }>(
      `/geofences/${id}/publish`,
      { version_id: versionId },
    );
    return data;
  },

  async previewLifecycleTransition(
    id: string,
    target: GeofenceLifecycleTarget,
  ): Promise<GeofenceLifecycleImpact> {
    const { data } = await http.post<GeofenceLifecycleImpactRaw>(
      `/geofences/${id}/${lifecyclePath[target]}-preview`,
    );
    return mapLifecycleImpact(data);
  },

  async transitionLifecycle(
    id: string,
    target: GeofenceLifecycleTarget,
    input: GeofenceTransitionInput,
  ): Promise<{ status: 'executed' }> {
    const { data } = await http.post<{ status: 'executed' }>(
      `/geofences/${id}/${lifecyclePath[target]}`,
      {
        reason: input.reason,
        preview_fingerprint: input.previewFingerprint,
      },
    );
    return data;
  },

  async listBindings(
    id: string,
    filter: GeofenceBindingFilter = {},
  ): Promise<GeofenceBindingPage> {
    const { data } = await http.get<GeofenceBindingPageRaw>(
      `/geofences/${id}/bindings`,
      {
        params: {
          status: filter.status,
          keyword: filter.keyword,
          page: filter.page,
          page_size: filter.pageSize,
        },
      },
    );
    return {
      items: (data.items ?? []).map(mapBindingDetail),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async exportBindings(
    id: string,
    filter: Pick<GeofenceBindingFilter, 'status' | 'keyword'> = {},
  ): Promise<void> {
    const response = await http.get<Blob>(
      `/geofences/${id}/bindings/export`,
      {
        params: {
          status: filter.status,
          keyword: filter.keyword,
        },
        responseType: 'blob',
      },
    );
    const filename = filenameFromContentDisposition(
      response.headers['content-disposition'] as string | undefined,
      'geofence-bindings.csv',
    );
    saveBlob(
      response.data,
      filename,
      'text/csv;charset=utf-8',
    );
  },

  async previewManualBindings(
    id: string,
    inputs: GeofenceBindingInputs,
  ): Promise<GeofenceManualBindingPreview> {
    const { data } =
      await http.post<GeofenceManualBindingPreviewRaw>(
        `/geofences/${id}/binding-preview`,
        bindingInputsBody(inputs),
      );
    return mapManualBindingPreview(data);
  },

  async previewCandidates(id: string): Promise<GeofenceCandidatePreview> {
    const { data } = await http.post<GeofenceCandidatePreviewRaw>(
      `/geofences/${id}/candidate-preview`,
    );
    return mapCandidatePreview(data);
  },

  async listControlActions(id: string): Promise<GeofenceControlAction[]> {
    const { data } = await http.get<GeofenceControlActionRaw[]>(
      `/geofences/${id}/control-actions`,
    );
    return (data ?? []).map((item) => ({
      id: item.id,
      deviceSN: item.device_sn,
      commandKey: item.action_key,
      actionType: item.action_type,
      status: item.status,
      beforeState: item.before_state ?? [],
      requestedState: item.requested_state ?? [],
      verifiedState: item.verified_state ?? [],
      lastError: item.last_error,
      createdAt: item.created_at,
      completedAt: item.completed_at,
    }));
  },

  async createManualBindJob(
    id: string,
    input: CreateGeofenceManualBindJobInput,
  ): Promise<GeofenceBatchJobAccepted> {
    const { data } = await http.post<{ job_id: string }>(
      `/geofences/${id}/bindings`,
      {
        ...bindingInputsBody(input),
        preview_fingerprint: input.previewFingerprint,
        reason: input.reason,
        scheduled_at: input.scheduledAt,
      },
    );
    return { jobId: data.job_id };
  },

  async suspendBinding(
    id: string,
    reason: string,
  ): Promise<GeofenceBinding> {
    const { data } = await http.post<GeofenceBindingRaw>(
      `/geofence-bindings/${id}/suspend`,
      { reason },
    );
    return mapBinding(data);
  },

  async resumeBinding(
    id: string,
    reason: string,
  ): Promise<GeofenceBinding> {
    const { data } = await http.post<GeofenceBindingRaw>(
      `/geofence-bindings/${id}/resume`,
      { reason },
    );
    return mapBinding(data);
  },

  async removeBinding(
    id: string,
    reason: string,
  ): Promise<GeofenceBinding> {
    const { data } = await http.delete<GeofenceBindingRaw>(
      `/geofence-bindings/${id}`,
      { data: { reason } },
    );
    return mapBinding(data);
  },

  async getManualBindJob(
    id: string,
  ): Promise<GeofenceManualBindJob> {
    const { data } = await http.get<GeofenceManualBindJobRaw>(
      `/geofence-jobs/${id}`,
    );
    return mapManualBindJob(data);
  },

  async listManualBindItems(
    id: string,
    filter: GeofenceBatchItemFilter = {},
  ): Promise<GeofenceBatchItemPage> {
    const { data } = await http.get<GeofenceBatchItemPageRaw>(
      `/geofence-jobs/${id}/items`,
      {
        params: {
          status: filter.status,
          page: filter.page,
          page_size: filter.pageSize,
        },
      },
    );
    return {
      items: (data.items ?? []).map(mapBatchItem),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },
};
