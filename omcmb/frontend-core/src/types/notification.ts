export type NotificationChannel = 'email' | 'sms_kafka' | 'sms_direct';

export interface AlarmEmailSetting {
  id: string;
  name: string;
  revision: number;
  enabled: boolean;
  alarmIdentifiers: string[];
  severities: number[];
  deviceIds: string[];
  deviceGroupIds: string[];
  technologies: string[];
  intervalMinutes: 0 | 10 | 30 | 60;
  toleranceDurationMinutes: 0 | 10 | 30 | 60;
  recipients: string[];
  includeDefaultRecipients: boolean;
  updatedAt: string;
}

export type AlarmEmailSettingPayload = Omit<AlarmEmailSetting, 'id' | 'revision' | 'updatedAt'>;

export interface AlarmEmailDefaults {
  recipients: string[];
  revision: number;
  updatedAt: string;
}

export interface StatusSummaryConfig {
  id: string;
  enabled: boolean;
  sendTime: string;
  timeZone: string;
  recipients: string[];
  revision: number;
  updatedAt: string;
}

export interface StatusSummaryConfigPayload {
  enabled: boolean;
  sendTime: string;
  timeZone: string;
  recipients: string[];
}

export interface NotificationChannelConfig {
  id: string;
  channel: NotificationChannel;
  name: string;
  enabled: boolean;
  parameters: Record<string, unknown>;
  secretConfigured: boolean;
  revision: number;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface NotificationChannelHealth {
  channelConfigId: string;
  circuitState: 'closed' | 'open' | 'half_open';
  consecutiveSuccesses: number;
  consecutiveFailures: number;
  lastSuccessAt?: string;
  lastFailureAt?: string;
  lastVerifiedAt?: string;
  lastErrorCategory?: string;
  lastErrorSummary?: string;
  circuitOpenedAt?: string;
  nextProbeAt?: string;
  updatedAt: string;
}

export type NotificationDeliveryFlowState =
  | 'queued'
  | 'sending'
  | 'retry_wait'
  | 'awaiting_receipt'
  | 'completed'
  | 'dead_letter'
  | 'suppressed'
  | 'cancelled';

export type NotificationDeliveryResult =
  | 'none'
  | 'accepted'
  | 'delivered'
  | 'failed'
  | 'unknown'
  | 'handoff_only';

export interface NotificationDelivery {
  id: string;
  eventId: string;
  occurrenceId: string;
  deviceId: string;
  deviceSn: string;
  technology: string;
  severity: number;
  alarmStatus: string;
  ruleVersionId: string;
  templateVersionId: string;
  channel: NotificationChannel;
  dispatchKind: string;
  sequenceNo: number;
  recipientType: string;
  maskedAddress: string;
  flowState: NotificationDeliveryFlowState;
  deliveryResult: NotificationDeliveryResult;
  suppressionReason?: string;
  maintenanceWindowId?: string;
  failureReason?: string;
  originDeliveryId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface NotificationDeliveryAttempt {
  id: string;
  deliveryId: string;
  attemptNo: number;
  startedAt: string;
  finishedAt?: string;
  result: string;
  errorCategory?: string;
  statusSummary?: string;
  providerRequestId?: string;
  nextRetryAt?: string;
}

export interface NotificationDeliveryListParams {
  occurrenceId?: string;
  channel?: NotificationChannel;
  flowState?: NotificationDeliveryFlowState;
  limit?: number;
  offset?: number;
}
