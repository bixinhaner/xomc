import type { AlarmSeverity } from '../types/common';

export type AlarmSeverityLabelMap = Record<AlarmSeverity | 'none', string>;
export type AlarmSeverityBadgeVariant = 'destructive' | 'warning' | 'default' | 'muted';

const ALARM_SEVERITY_ALIAS_MAP: Record<string, AlarmSeverity | 'none'> = {
  critical: 'critical',
  '31001': 'critical',
  major: 'major',
  '31002': 'major',
  minor: 'minor',
  '31003': 'minor',
  warning: 'warning',
  '31004': 'warning',
  none: 'none',
};

export const DEFAULT_ALARM_SEVERITY_LABELS_ZH: AlarmSeverityLabelMap = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
  none: '无',
};

export const DEFAULT_ALARM_SEVERITY_BADGE_VARIANTS: Record<AlarmSeverity | 'none', AlarmSeverityBadgeVariant> = {
  critical: 'destructive',
  major: 'destructive',
  minor: 'warning',
  warning: 'warning',
  none: 'muted',
};

export function normalizeAlarmSeverity(raw?: string | null): AlarmSeverity | 'none' {
  const normalized = (raw ?? '').toLowerCase().trim();
  return ALARM_SEVERITY_ALIAS_MAP[normalized] ?? 'none';
}

export function getAlarmSeverityBadgeVariant(level: AlarmSeverity | 'none'): AlarmSeverityBadgeVariant {
  return DEFAULT_ALARM_SEVERITY_BADGE_VARIANTS[level] ?? 'default';
}

export function hasAlarmSeverity(level: AlarmSeverity | 'none' | string | null | undefined): boolean {
  return normalizeAlarmSeverity(level) !== 'none';
}

export function formatAlarmSeverityBadgeLabel(
  level: AlarmSeverity | 'none',
  count: number | null | undefined,
  labels: AlarmSeverityLabelMap = DEFAULT_ALARM_SEVERITY_LABELS_ZH,
): string {
  const label = labels[level] ?? level;
  if (level !== 'none' && (count ?? 0) > 0) {
    return `${label} · ${count}`;
  }
  return label;
}