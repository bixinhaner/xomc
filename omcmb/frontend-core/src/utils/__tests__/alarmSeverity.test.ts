import { describe, expect, it } from 'vitest';
import {
  DEFAULT_ALARM_SEVERITY_LABELS_ZH,
  formatAlarmSeverityBadgeLabel,
  getAlarmSeverityBadgeVariant,
  hasAlarmSeverity,
  normalizeAlarmSeverity,
} from '../alarmSeverity';

describe('alarmSeverity utils', () => {
  it('normalizes legacy and dictionary-coded severities', () => {
    expect(normalizeAlarmSeverity('critical')).toBe('critical');
    expect(normalizeAlarmSeverity('31001')).toBe('critical');
    expect(normalizeAlarmSeverity('major')).toBe('major');
    expect(normalizeAlarmSeverity('31002')).toBe('major');
    expect(normalizeAlarmSeverity('minor')).toBe('minor');
    expect(normalizeAlarmSeverity('31003')).toBe('minor');
    expect(normalizeAlarmSeverity('warning')).toBe('warning');
    expect(normalizeAlarmSeverity('31004')).toBe('warning');
    expect(normalizeAlarmSeverity('none')).toBe('none');
    expect(normalizeAlarmSeverity(undefined)).toBe('none');
    expect(normalizeAlarmSeverity('bogus')).toBe('none');
  });

  it('returns badge variants and labels consistently', () => {
    expect(getAlarmSeverityBadgeVariant('critical')).toBe('destructive');
    expect(getAlarmSeverityBadgeVariant('minor')).toBe('warning');
    expect(getAlarmSeverityBadgeVariant('none')).toBe('muted');

    expect(formatAlarmSeverityBadgeLabel('major', 3, DEFAULT_ALARM_SEVERITY_LABELS_ZH)).toBe('重要 · 3');
    expect(formatAlarmSeverityBadgeLabel('none', 8, DEFAULT_ALARM_SEVERITY_LABELS_ZH)).toBe('无');
    expect(formatAlarmSeverityBadgeLabel('warning', 0, DEFAULT_ALARM_SEVERITY_LABELS_ZH)).toBe('警告');
  });

  it('detects active severities', () => {
    expect(hasAlarmSeverity('critical')).toBe(true);
    expect(hasAlarmSeverity('31004')).toBe(true);
    expect(hasAlarmSeverity('none')).toBe(false);
  });
});