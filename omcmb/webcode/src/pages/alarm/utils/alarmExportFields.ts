import type { Alarm, DealState, EventType } from '@core/types/alarm';
import type { TranslateFn } from '@/hooks/useT';
import { formatSystemTime } from '@core/utils/systemTime';
import { formatBaseStationTypeLabel } from './baseStationType';

export type AlarmExportFieldKey =
  | 'deviceSn'
  | 'alarmIdentifier'
  | 'severity'
  | 'possibleCause'
  | 'neType'
  | 'equipInfo'
  | 'eventType'
  | 'dealState'
  | 'eventTime'
  | 'updTime'
  | 'dealUser'
  | 'dealTime'
  | 'dealMemo'
  | 'clearUser'
  | 'clearTime'
  | 'clearMemo'
  | 'additionalText'
  | 'additionalInfo';

interface AlarmExportFieldDefinition {
  key: AlarmExportFieldKey;
  label: string;
  getValue: (alarm: Alarm) => string;
}

const SEVERITY_LABEL_KEYS: Record<string, string> = {
  critical: 'alarm.severity.critical',
  major: 'alarm.severity.major',
  minor: 'alarm.severity.minor',
  warning: 'alarm.severity.warning',
};

const EVENT_TYPE_LABEL_KEYS: Record<EventType, string> = {
  communication: 'alarm.eventType.communication',
  qualityOfService: 'alarm.eventType.qualityOfService',
  processingError: 'alarm.eventType.processingError',
  device: 'alarm.eventType.device',
  environment: 'alarm.eventType.environment',
  performance: 'alarm.eventType.performance',
};

const DEAL_STATE_LABEL_KEYS: Record<DealState, string> = {
  '0': 'alarm.dealState.unconfirmedUncleared',
  '1': 'alarm.dealState.confirmedUncleared',
  '2': 'alarm.dealState.unconfirmedCleared',
  '3': 'alarm.dealState.confirmedCleared',
};

function formatAdditionalInfo(additionalInfo?: Record<string, string>): string {
  if (!additionalInfo) {
    return '';
  }

  return Object.entries(additionalInfo)
    .map(([key, value]) => `${key}: ${value || '-'}`)
    .join('\n');
}

export function buildAlarmExportFieldDefinitions(
  t: TranslateFn,
): AlarmExportFieldDefinition[] {
  return [
    {
      key: 'deviceSn',
      label: t('alarm.deviceSn'),
      getValue: (alarm) => alarm.deviceSn || '',
    },
    {
      key: 'alarmIdentifier',
      label: t('alarm.alarmIdentifier'),
      getValue: (alarm) => alarm.alarmIdentifier || '',
    },
    {
      key: 'severity',
      label: t('alarm.severity'),
      getValue: (alarm) => t(SEVERITY_LABEL_KEYS[alarm.severity] || 'common.unknown'),
    },
    {
      key: 'possibleCause',
      label: t('alarm.possibleCause'),
      getValue: (alarm) => alarm.probableCause || '',
    },
    {
      key: 'neType',
      label: t('alarm.neTypeCol'),
      getValue: (alarm) => formatBaseStationTypeLabel(alarm.neType || ''),
    },
    {
      key: 'equipInfo',
      label: t('alarm.equipInfo'),
      getValue: (alarm) => alarm.equipInfo || '',
    },
    {
      key: 'eventType',
      label: t('alarm.eventType'),
      getValue: (alarm) => t(EVENT_TYPE_LABEL_KEYS[alarm.eventType] || 'common.unknown'),
    },
    {
      key: 'dealState',
      label: t('alarm.dealState'),
      getValue: (alarm) => t(DEAL_STATE_LABEL_KEYS[alarm.dealState] || 'common.unknown'),
    },
    {
      key: 'eventTime',
      label: t('alarm.eventTime'),
      getValue: (alarm) => (alarm.eventTime ? formatSystemTime(alarm.eventTime) : ''),
    },
    {
      key: 'updTime',
      label: t('alarm.updTime'),
      getValue: (alarm) => (alarm.updTime ? formatSystemTime(alarm.updTime) : ''),
    },
    {
      key: 'dealUser',
      label: t('alarm.dealUser'),
      getValue: (alarm) => alarm.dealUser || '',
    },
    {
      key: 'dealTime',
      label: t('alarm.dealTime'),
      getValue: (alarm) => {
        const raw = alarm.dealTime || alarm.acknowledgedAt;
        return raw ? formatSystemTime(raw) : '';
      },
    },
    {
      key: 'dealMemo',
      label: t('alarm.dealMemo'),
      getValue: (alarm) => alarm.dealMemo || '',
    },
    {
      key: 'clearUser',
      label: t('alarm.clearUser'),
      getValue: (alarm) => alarm.clearUser || '',
    },
    {
      key: 'clearTime',
      label: t('alarm.clearTime'),
      getValue: (alarm) => {
        const raw = alarm.clearTime || alarm.clearedAt;
        return raw ? formatSystemTime(raw) : '';
      },
    },
    {
      key: 'clearMemo',
      label: t('alarm.clearMemo'),
      getValue: (alarm) => alarm.clearMemo || '',
    },
    {
      key: 'additionalText',
      label: t('alarm.additionalText'),
      getValue: (alarm) => alarm.additionalText || '',
    },
    {
      key: 'additionalInfo',
      label: t('alarm.additionalInfo'),
      getValue: (alarm) => formatAdditionalInfo(alarm.additionalInfo),
    },
  ];
}
