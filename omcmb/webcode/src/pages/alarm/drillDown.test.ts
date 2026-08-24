import { describe, it, expect } from 'vitest';
import dayjs from 'dayjs';
import { buildDrillDownSearch, parseAlarmId, parseDrillDownParams, withoutAlarmId } from './drillDown';

describe('buildDrillDownSearch (#236 统计页钻取建链)', () => {
  const now = dayjs('2026-06-12T10:30:00.000Z');

  it('serializes severity + deviceSN', () => {
    const p = buildDrillDownSearch({ severity: 'major', deviceSN: 'SN-1', timeRange: '7days', now });
    expect(p.get('severity')).toBe('major');
    expect(p.get('deviceSN')).toBe('SN-1');
  });

  it('7days spans 7 days ending now, start at start-of-day', () => {
    const p = buildDrillDownSearch({ timeRange: '7days', now });
    expect(p.get('endTime')).toBe(now.toISOString());
    expect(p.get('startTime')).toBe(now.subtract(6, 'day').startOf('day').toISOString());
  });

  it('30days spans 30 days ending now', () => {
    const p = buildDrillDownSearch({ timeRange: '30days', now });
    expect(p.get('endTime')).toBe(now.toISOString());
    expect(p.get('startTime')).toBe(now.subtract(29, 'day').startOf('day').toISOString());
  });

  it('custom uses the provided custom dates', () => {
    const s = dayjs('2026-05-01T00:00:00.000Z');
    const e = dayjs('2026-05-10T00:00:00.000Z');
    const p = buildDrillDownSearch({ timeRange: 'custom', customStartDate: s, customEndDate: e, now });
    expect(p.get('startTime')).toBe(s.toISOString());
    expect(p.get('endTime')).toBe(e.toISOString());
  });

  it('custom WITHOUT both dates emits no time params (failure/guard path)', () => {
    const p = buildDrillDownSearch({ timeRange: 'custom', customStartDate: null, customEndDate: null, now });
    expect(p.get('startTime')).toBeNull();
    expect(p.get('endTime')).toBeNull();
  });

  it('omits severity/deviceSN when not provided', () => {
    const p = buildDrillDownSearch({ timeRange: '7days', now });
    expect(p.get('severity')).toBeNull();
    expect(p.get('deviceSN')).toBeNull();
  });
});

describe('parseDrillDownParams (#236 列表侧解析)', () => {
  const parse = (qs: string) => parseDrillDownParams(new URLSearchParams(qs));

  it('maps severity + deviceSN into filter and formValues', () => {
    const { filter, formValues } = parse('severity=major&deviceSN=ABC123');
    expect(filter.severity).toEqual(['major']);
    expect(filter.deviceSn).toBe('ABC123');
    expect(formValues.severity).toEqual(['major']);
    expect(formValues.deviceSn).toBe('ABC123');
  });

  it('drops an invalid severity (allowlist guard)', () => {
    const { filter, formValues } = parse('severity=foo');
    expect(filter.severity).toBeUndefined();
    expect(formValues.severity).toBeUndefined();
  });

  it('honors the lowercase deviceSn alias', () => {
    const { filter } = parse('deviceSn=lower-sn');
    expect(filter.deviceSn).toBe('lower-sn');
  });

  it('requires BOTH startTime and endTime for a timeRange', () => {
    expect(parse('startTime=2026-05-01T00:00:00.000Z').filter.timeRange).toBeUndefined();
    const both = parse('startTime=2026-05-01T00:00:00.000Z&endTime=2026-05-10T00:00:00.000Z');
    expect(both.filter.timeRange).toEqual([
      '2026-05-01T00:00:00.000Z',
      '2026-05-10T00:00:00.000Z',
    ]);
  });

  it('passes eventType through when present', () => {
    expect(parse('eventType=communication').filter.eventType).toBe('communication');
  });

  it('returns empty filter for empty query', () => {
    expect(parse('')).toEqual({ filter: {}, formValues: {} });
  });
});

describe('build → parse round-trip', () => {
  it('a built severity+device+30days URL parses back to consistent filter', () => {
    const now = dayjs('2026-06-12T10:30:00.000Z');
    const search = buildDrillDownSearch({ severity: 'critical', deviceSN: 'SN-9', timeRange: '30days', now });
    const { filter } = parseDrillDownParams(search);
    expect(filter.severity).toEqual(['critical']);
    expect(filter.deviceSn).toBe('SN-9');
    expect(filter.timeRange).toEqual([
      now.subtract(29, 'day').startOf('day').toISOString(),
      now.toISOString(),
    ]);
  });
});

describe('alarm detail deep links', () => {
  it('accepts a UUID alarmId and rejects malformed values', () => {
    expect(parseAlarmId(new URLSearchParams('alarmId=892d12c0-ec1a-4fd1-8070-902d3aaf84e9'))).toBe(
      '892d12c0-ec1a-4fd1-8070-902d3aaf84e9',
    );
    expect(parseAlarmId(new URLSearchParams('alarmId=not-a-uuid'))).toBeUndefined();
  });

  it('removes alarmId while preserving drill-down filters', () => {
    expect(withoutAlarmId(new URLSearchParams(
      'alarmId=892d12c0-ec1a-4fd1-8070-902d3aaf84e9&severity=critical&from=dashboard',
    )).toString()).toBe('severity=critical&from=dashboard');
  });
});
