import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import type { PmPageStateSnapshot } from '@core/store/pmPageStateStore';
import {
  buildKpiQueryStateSnapshot,
  kpiQuerySignature,
  PM_KPI_QUERY_PAGE_KEY,
  restoredKpiQueryDelayMs,
  restoreKpiQueryState,
} from './kpiQueryState';

function snapshotOf(
  input: Parameters<typeof buildKpiQueryStateSnapshot>[0],
): PmPageStateSnapshot {
  return {
    ...buildKpiQueryStateSnapshot(input),
    savedAt: '2026-07-29T00:00:00.000Z',
  };
}

describe('kpiQueryState', () => {
  it('restores form fields, submitted query and pagination', () => {
    const submitted = {
      payload: {
        deviceType: 'GNB' as const,
        deviceSns: ['GNB00001'],
        metricPaths: ['K001'],
        granularity: 'hourly' as const,
        timeRangePreset: 'custom' as const,
      },
      range: { start: '2026-07-01T00:00:00+08:00', end: '2026-07-02T00:00:00+08:00' },
      cellSel: { GNB00001: ['Cell=1'] },
    };

    const restored = restoreKpiQueryState(snapshotOf({
      payload: {
        deviceType: 'GNB',
        deviceSns: ['GNB00002'],
        metricPaths: ['K002'],
        granularity: 'daily',
        timeRangePreset: 'custom',
      },
      customRange: [dayjs('2026-07-03T00:00:00Z'), dayjs('2026-07-04T00:00:00Z')],
      timeRangeDirty: true,
      cellSel: { GNB00002: ['Cell=9'] },
      submitted,
      pivotPage: 3,
      pivotPageSize: 100,
      templateTab: 'private',
      activeTemplateId: 'tpl-1',
      sidebarCollapsed: true,
    }));

    expect(restored.payload.deviceSns).toEqual(['GNB00002']);
    expect(restored.cellSel).toEqual({ GNB00002: ['Cell=9'] });
    expect(restored.submitted?.payload.deviceSns).toEqual(['GNB00001']);
    expect(restored.submitted?.cellSel).toEqual({ GNB00001: ['Cell=1'] });
    expect(restored.pivotPage).toBe(3);
    expect(restored.pivotPageSize).toBe(100);
    expect(restored.templateTab).toBe('private');
    expect(restored.activeTemplateId).toBe('tpl-1');
    expect(restored.sidebarCollapsed).toBe(true);
    expect(restored.shouldRestoreQuery).toBe(true);
  });

  it('does not restore a query when conditions were saved without a submitted snapshot', () => {
    const restored = restoreKpiQueryState(snapshotOf({
      payload: {
        deviceType: 'ENB',
        deviceSns: ['ENB00001'],
        metricPaths: ['K001'],
        granularity: '15min',
        timeRangePreset: 'last_1h',
      },
      customRange: null,
      timeRangeDirty: true,
      cellSel: {},
      submitted: null,
      pivotPage: 2,
      pivotPageSize: 50,
      templateTab: 'public',
      sidebarCollapsed: false,
    }));

    expect(restored.submitted).toBeNull();
    expect(restored.shouldRestoreQuery).toBe(false);
  });

  it('stores only serializable query state under the KPI query page key', () => {
    const state = buildKpiQueryStateSnapshot({
      payload: {
        deviceType: 'ENB',
        deviceSns: ['ENB00001'],
        metricPaths: ['K001'],
        granularity: '15min',
        timeRangePreset: 'last_1h',
      },
      customRange: null,
      timeRangeDirty: false,
      cellSel: {},
      submitted: null,
      pivotPage: 1,
      pivotPageSize: 50,
      templateTab: 'public',
      sidebarCollapsed: false,
    });

    expect(PM_KPI_QUERY_PAGE_KEY).toBe('/performance/query');
    expect(JSON.stringify(state)).not.toContain('rows');
    expect(JSON.stringify(state)).not.toContain('exportData');
    expect(kpiQuerySignature(null)).toBeNull();
  });

  it('calculates restored query throttle delay', () => {
    expect(restoredKpiQueryDelayMs('2026-07-30T11:59:58.000Z', Date.parse('2026-07-30T12:00:00.000Z'))).toBe(8_000);
    expect(restoredKpiQueryDelayMs('2026-07-30T11:59:40.000Z', Date.parse('2026-07-30T12:00:00.000Z'))).toBe(0);
  });
});
