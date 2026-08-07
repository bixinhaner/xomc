import http from '../http';
import type {
  AlarmEmailDefaults,
  AlarmEmailSetting,
  AlarmEmailSettingPayload,
  NotificationChannel,
  NotificationChannelConfig,
  NotificationChannelHealth,
  NotificationDelivery,
  NotificationDeliveryAttempt,
  NotificationDeliveryListParams,
  StatusSummaryConfig,
  StatusSummaryConfigPayload,
} from '../../types/notification';

type BackendRecord = Record<string, unknown>;

function asRecord(value: unknown): BackendRecord {
  return value && typeof value === 'object' ? (value as BackendRecord) : {};
}

function asStringArray(value: unknown): string[] {
  return Array.isArray(value) ? value.map(String) : [];
}

function asNumberArray(value: unknown): number[] {
  return Array.isArray(value) ? value.map(Number).filter(Number.isFinite) : [];
}

function revisionFromHeaders(headers: unknown, fallback: number): number {
  const raw = asRecord(headers).etag;
  if (typeof raw !== 'string') return fallback;
  const parsed = Number(raw.replace(/^W\//, '').replaceAll('"', ''));
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback;
}

function ifMatch(revision: number) {
  return { headers: { 'If-Match': `"${revision}"` } };
}

export function isNotificationRevisionConflict(error: unknown): boolean {
  return asRecord(asRecord(error).response).status === 412;
}

function mapAlarmEmailSetting(value: unknown, headers?: unknown): AlarmEmailSetting {
  const item = asRecord(value);
  return {
    id: String(item.id ?? ''),
    name: String(item.name ?? ''),
    revision: revisionFromHeaders(headers, Number(item.revision ?? 1)),
    enabled: Boolean(item.enabled),
    alarmIdentifiers: asStringArray(item.alarm_identifiers),
    severities: asNumberArray(item.severities),
    deviceIds: asStringArray(item.device_ids),
    deviceGroupIds: asStringArray(item.device_group_ids),
    technologies: asStringArray(item.technologies),
    intervalMinutes: Number(item.interval_minutes ?? 0) as AlarmEmailSetting['intervalMinutes'],
    toleranceDurationMinutes: Number(item.tolerance_duration_minutes ?? 0) as AlarmEmailSetting['toleranceDurationMinutes'],
    recipients: asStringArray(item.recipients),
    includeDefaultRecipients: Boolean(item.include_default_recipients),
    updatedAt: String(item.updated_at ?? ''),
  };
}

function backendAlarmEmailSetting(payload: AlarmEmailSettingPayload): BackendRecord {
  return {
    name: payload.name,
    enabled: payload.enabled,
    alarm_identifiers: payload.alarmIdentifiers,
    severities: payload.severities,
    device_ids: payload.deviceIds,
    device_group_ids: payload.deviceGroupIds,
    technologies: payload.technologies,
    interval_minutes: payload.intervalMinutes,
    tolerance_duration_minutes: payload.toleranceDurationMinutes,
    recipients: payload.recipients,
    include_default_recipients: payload.includeDefaultRecipients,
  };
}

function mapAlarmEmailDefaults(value: unknown, headers?: unknown): AlarmEmailDefaults {
  const item = asRecord(value);
  return {
    recipients: asStringArray(item.recipients),
    revision: revisionFromHeaders(headers, Number(item.revision ?? 1)),
    updatedAt: String(item.updated_at ?? ''),
  };
}

export const alarmEmailSettingsApi = {
  async list(): Promise<AlarmEmailSetting[]> {
    const { data } = await http.get<unknown[]>('/alarm-email-settings');
    return (data ?? []).map((item) => mapAlarmEmailSetting(item));
  },
  async get(id: string): Promise<AlarmEmailSetting> {
    const response = await http.get<unknown>(`/alarm-email-settings/${id}`);
    return mapAlarmEmailSetting(response.data, response.headers);
  },
  async create(payload: AlarmEmailSettingPayload): Promise<AlarmEmailSetting> {
    const response = await http.post<unknown>('/alarm-email-settings', backendAlarmEmailSetting(payload));
    return mapAlarmEmailSetting(response.data, response.headers);
  },
  async update(id: string, revision: number, payload: AlarmEmailSettingPayload): Promise<AlarmEmailSetting> {
    const response = await http.patch<unknown>(
      `/alarm-email-settings/${id}`,
      backendAlarmEmailSetting(payload),
      ifMatch(revision),
    );
    return mapAlarmEmailSetting(response.data, response.headers);
  },
  async archive(id: string, revision: number): Promise<void> {
    await http.delete(`/alarm-email-settings/${id}`, ifMatch(revision));
  },
  async getDefaults(): Promise<AlarmEmailDefaults> {
    const response = await http.get<unknown>('/alarm-email-settings/default-recipients');
    return mapAlarmEmailDefaults(response.data, response.headers);
  },
  async updateDefaults(revision: number, recipients: string[]): Promise<AlarmEmailDefaults> {
    const response = await http.patch<unknown>(
      '/alarm-email-settings/default-recipients',
      { recipients },
      ifMatch(revision),
    );
    return mapAlarmEmailDefaults(response.data, response.headers);
  },
};

function mapStatusSummaryConfig(value: unknown, headers?: unknown): StatusSummaryConfig {
  const item = asRecord(value);
  return {
    id: String(item.id ?? ''),
    enabled: Boolean(item.enabled),
    sendTime: String(item.send_time ?? '00:00'),
    timeZone: String(item.time_zone ?? 'Asia/Shanghai'),
    recipients: asStringArray(item.recipients),
    revision: revisionFromHeaders(headers, Number(item.revision ?? 1)),
    updatedAt: String(item.updated_at ?? ''),
  };
}

export const statusSummaryConfigApi = {
  async get(): Promise<StatusSummaryConfig> {
    const response = await http.get<unknown>('/notification/status-summary-settings');
    return mapStatusSummaryConfig(response.data, response.headers);
  },
  async update(revision: number, payload: StatusSummaryConfigPayload): Promise<StatusSummaryConfig> {
    const response = await http.patch<unknown>('/notification/status-summary-settings', {
      enabled: payload.enabled,
      send_time: payload.sendTime,
      time_zone: payload.timeZone,
      recipients: payload.recipients,
    }, ifMatch(revision));
    return mapStatusSummaryConfig(response.data, response.headers);
  },
};

function mapChannel(value: unknown, headers?: unknown): NotificationChannelConfig {
  const item = asRecord(value);
  return {
    id: String(item.id ?? ''),
    channel: String(item.channel ?? 'email') as NotificationChannel,
    name: String(item.name ?? ''),
    enabled: Boolean(item.enabled),
    parameters: asRecord(item.parameters),
    secretConfigured: Boolean(item.secret_configured),
    revision: revisionFromHeaders(headers, Number(item.revision ?? 1)),
    createdBy: String(item.created_by ?? ''),
    createdAt: String(item.created_at ?? ''),
    updatedAt: String(item.updated_at ?? ''),
  };
}

export const notificationChannelApi = {
  async list(): Promise<NotificationChannelConfig[]> {
    const { data } = await http.get<unknown[]>('/notification-channels');
    return (data ?? []).map((item) => mapChannel(item));
  },
  async verify(id: string): Promise<void> {
    await http.post(`/notification-channels/${id}/verify`);
  },
  async health(id: string): Promise<NotificationChannelHealth> {
    const { data } = await http.get<BackendRecord>(`/notification-channels/${id}/health`);
    return {
      channelConfigId: String(data.channel_config_id ?? ''),
      circuitState: String(data.circuit_state ?? 'closed') as NotificationChannelHealth['circuitState'],
      consecutiveSuccesses: Number(data.consecutive_successes ?? 0),
      consecutiveFailures: Number(data.consecutive_failures ?? 0),
      lastSuccessAt: data.last_success_at ? String(data.last_success_at) : undefined,
      lastFailureAt: data.last_failure_at ? String(data.last_failure_at) : undefined,
      lastVerifiedAt: data.last_verified_at ? String(data.last_verified_at) : undefined,
      lastErrorCategory: data.last_error_category ? String(data.last_error_category) : undefined,
      lastErrorSummary: data.last_error_summary ? String(data.last_error_summary) : undefined,
      circuitOpenedAt: data.circuit_opened_at ? String(data.circuit_opened_at) : undefined,
      nextProbeAt: data.next_probe_at ? String(data.next_probe_at) : undefined,
      updatedAt: String(data.updated_at ?? ''),
    };
  },
};

function mapDelivery(value: unknown): NotificationDelivery {
  const item = asRecord(value);
  return {
    id: String(item.id ?? ''),
    eventId: String(item.event_id ?? ''),
    occurrenceId: String(item.occurrence_id ?? ''),
    deviceId: String(item.device_id ?? ''),
    deviceSn: String(item.device_sn ?? ''),
    technology: String(item.technology ?? ''),
    severity: Number(item.severity ?? 0),
    alarmStatus: String(item.alarm_status ?? ''),
    ruleVersionId: String(item.rule_version_id ?? ''),
    templateVersionId: String(item.template_version_id ?? ''),
    channel: String(item.channel ?? 'email') as NotificationChannel,
    dispatchKind: String(item.dispatch_kind ?? ''),
    sequenceNo: Number(item.sequence_no ?? 0),
    recipientType: String(item.recipient_type ?? ''),
    maskedAddress: String(item.masked_address ?? ''),
    flowState: String(item.flow_state ?? 'queued') as NotificationDelivery['flowState'],
    deliveryResult: String(item.delivery_result ?? 'none') as NotificationDelivery['deliveryResult'],
    suppressionReason: item.suppression_reason ? String(item.suppression_reason) : undefined,
    maintenanceWindowId: item.maintenance_window_id ? String(item.maintenance_window_id) : undefined,
    failureReason: item.failure_reason ? String(item.failure_reason) : undefined,
    originDeliveryId: item.origin_delivery_id ? String(item.origin_delivery_id) : undefined,
    createdAt: String(item.created_at ?? ''),
    updatedAt: String(item.updated_at ?? ''),
  };
}

export const notificationDeliveryApi = {
  async list(params: NotificationDeliveryListParams = {}): Promise<NotificationDelivery[]> {
    const { data } = await http.get<unknown[]>('/notification-deliveries', {
      params: {
        occurrence_id: params.occurrenceId,
        channel: params.channel,
        flow_state: params.flowState,
        limit: params.limit,
        offset: params.offset,
      },
    });
    return (data ?? []).map(mapDelivery);
  },
  async get(id: string): Promise<NotificationDelivery> {
    const { data } = await http.get<unknown>(`/notification-deliveries/${id}`);
    return mapDelivery(data);
  },
  async attempts(id: string): Promise<NotificationDeliveryAttempt[]> {
    const { data } = await http.get<BackendRecord[]>(`/notification-deliveries/${id}/attempts`);
    return (data ?? []).map((item) => ({
      id: String(item.id ?? ''),
      deliveryId: String(item.delivery_id ?? ''),
      attemptNo: Number(item.attempt_no ?? 0),
      startedAt: String(item.started_at ?? ''),
      finishedAt: item.finished_at ? String(item.finished_at) : undefined,
      result: String(item.result ?? ''),
      errorCategory: item.error_category ? String(item.error_category) : undefined,
      statusSummary: item.status_summary ? String(item.status_summary) : undefined,
      providerRequestId: item.provider_request_id ? String(item.provider_request_id) : undefined,
      nextRetryAt: item.next_retry_at ? String(item.next_retry_at) : undefined,
    }));
  },
  async retry(id: string, reason: string): Promise<number> {
    const { data } = await http.post<{ retried: number }>(`/notification-deliveries/${id}/retry`, { reason });
    return data.retried;
  },
};
