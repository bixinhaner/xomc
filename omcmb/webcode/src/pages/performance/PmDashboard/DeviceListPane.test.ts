import { afterEach, describe, expect, it, vi } from 'vitest';
import dayjs from 'dayjs';
import { PM_QUERY_SELECTION_LIMIT } from '@/constants/pmQueryLimits';
import type { DashboardExportSelection } from '@core/utils/kpiExportParams';
import {
  buildDeviceViewAggregatedParams,
  buildDeviceViewExportInput,
  buildDeviceViewRequestedObjectLdns,
  buildSubmittedDeviceViewExportSelection,
  isDeviceViewQueryTimeoutError,
  isDeviceViewDeviceSelectionOverLimit,
  isDeviceViewMetricSelectionOverLimit,
} from './DeviceListPane';
import {
  actualRangeFromMeta,
  buildDeviceViewRequestTimeWindow,
  defaultRangeForGranularity,
  toDeviceViewRequestRFC3339,
} from './deviceListPaneTimeUtils';
import {
  buildDeviceViewStateSnapshot,
  buildDeviceViewSubmittedQuery,
  restoredDeviceViewQueryDelayMs,
  restoreDeviceViewState,
} from './deviceViewState';

const selection: DashboardExportSelection = {
  technology: 'lte',
  deviceSns: ['ISSUE95-LTE-01'],
  metricPaths: ['K001'],
  granularity: '15min',
  startTime: '2026-07-16T07:00:00.000Z',
  endTime: '2026-07-16T08:00:00.000Z',
};

describe('buildDeviceViewExportInput', () => {
  it('设备性能查看导出使用独立 sourceType 和中文任务名前缀', () => {
    const input = buildDeviceViewExportInput(
      selection,
      'KPI导出',
      '设备性能查看',
      new Date(2026, 6, 16, 18, 40, 5),
    );

    expect(input.sourceType).toBe('device_view');
    expect(input.taskName).toBe('KPI导出_设备性能查看_20260716_184005');
    expect(input.taskName).not.toContain('仪表盘');
  });

  it('英文界面下使用英文设备性能查看任务名前缀', () => {
    const input = buildDeviceViewExportInput(
      selection,
      'KPI Export',
      'Device Performance View',
      new Date(2026, 6, 16, 18, 40, 5),
    );

    expect(input.sourceType).toBe('device_view');
    expect(input.taskName).toBe('KPI_Export_Device_Performance_View_20260716_184005');
  });
});

describe('buildSubmittedDeviceViewExportSelection', () => {
  it('没有已提交出图条件时不构造导出参数', () => {
    expect(buildSubmittedDeviceViewExportSelection(null, null, 'UTC')).toBeNull();
  });

  it('导出只使用已提交出图条件，并优先使用后端实际时间范围', () => {
    const submitted = buildDeviceViewSubmittedQuery({
      tech: 'nr',
      deviceSns: ['GNB00001'],
      metricPaths: ['K001'],
      granularity: 'hourly',
      filter: {
        range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
        weekdays: [1, 2],
        hours: [8, 9],
        compare: false,
      },
      allowedLdns: ['Cell=1'],
      systemTimezone: 'UTC',
    });

    const selectionFromSubmitted = buildSubmittedDeviceViewExportSelection(
      submitted,
      [dayjs('2026-07-01T01:00:00Z'), dayjs('2026-07-01T23:00:00Z')],
      'UTC',
    );
    const expectedStart = toDeviceViewRequestRFC3339(dayjs('2026-07-01T01:00:00Z'), 'UTC');
    const expectedEnd = toDeviceViewRequestRFC3339(dayjs('2026-07-01T23:00:00Z'), 'UTC');

    expect(selectionFromSubmitted).toMatchObject({
      technology: 'nr',
      deviceSns: ['GNB00001'],
      metricPaths: ['K001'],
      granularity: 'hourly',
      objectLdns: ['Cell=1'],
      weekdays: [1, 2],
      hours: [8, 9],
      startTime: expectedStart,
      endTime: expectedEnd,
    });
  });
});

describe('isDeviceViewMetricSelectionOverLimit', () => {
  it('设备性能查看沿用 PM 查询设备数量上限', () => {
    expect(isDeviceViewDeviceSelectionOverLimit(
      Array.from({ length: PM_QUERY_SELECTION_LIMIT }, (_, index) => `SN-${index + 1}`),
    )).toBe(false);
    expect(isDeviceViewDeviceSelectionOverLimit(
      Array.from({ length: PM_QUERY_SELECTION_LIMIT + 1 }, (_, index) => `SN-${index + 1}`),
    )).toBe(true);
  });

  it('设备性能查看沿用 PM 查询指标数量上限', () => {
    expect(isDeviceViewMetricSelectionOverLimit(
      Array.from({ length: PM_QUERY_SELECTION_LIMIT }, (_, index) => `K${index + 1}`),
    )).toBe(false);
    expect(isDeviceViewMetricSelectionOverLimit(
      Array.from({ length: PM_QUERY_SELECTION_LIMIT + 1 }, (_, index) => `K${index + 1}`),
    )).toBe(true);
  });
});

describe('buildDeviceViewAggregatedParams', () => {
  it('设备性能查看使用透视行分页保护大窗口查询', () => {
    const submitted = buildDeviceViewSubmittedQuery({
      tech: 'lte',
      deviceSns: ['ENB00001'],
      metricPaths: ['K001', 'K002'],
      granularity: '15min',
      filter: {
        range: [dayjs('2026-08-22T00:00:00Z'), dayjs('2026-08-23T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
        weekdays: [0, 1, 2, 3, 4, 5, 6],
        hours: [0, 1, 2],
        compare: false,
      },
      allowedLdns: [],
      systemTimezone: 'UTC',
    });

    const params = buildDeviceViewAggregatedParams(submitted);

    expect(params).toMatchObject({
      granularity: '15min',
      technology: 'lte',
      metricPaths: ['K001', 'K002'],
      limit: 5000,
      countMode: 'n_plus_one',
      pageBy: 'pivot_row',
      fillEmpty: true,
      hours: [0, 1, 2],
    });
    expect(params?.objectLdns).toBeUndefined();
    expect(params?.weekdays).toBeUndefined();
  });

  it('周期对比复用同一查询口径，只替换时间窗口', () => {
    const submitted = buildDeviceViewSubmittedQuery({
      tech: 'nr',
      deviceSns: ['GNB00001'],
      metricPaths: ['K001'],
      granularity: 'hourly',
      filter: {
        range: [dayjs('2026-08-22T00:00:00Z'), dayjs('2026-08-23T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
        weekdays: [1, 2],
        hours: Array.from({ length: 24 }, (_, hour) => hour),
        compare: true,
      },
      allowedLdns: ['Cellid=1,PLMN=46000'],
      systemTimezone: 'UTC',
    });

    const params = buildDeviceViewAggregatedParams(submitted, {
      startTime: '2026-08-21T00:00:00Z',
      endTime: '2026-08-22T00:00:00Z',
    });

    expect(params).toMatchObject({
      startTime: '2026-08-21T00:00:00Z',
      endTime: '2026-08-22T00:00:00Z',
      pageBy: 'pivot_row',
      fillEmpty: true,
      objectLdns: ['Cellid=1,PLMN=46000'],
      weekdays: [1, 2],
    });
    expect(params?.hours).toBeUndefined();
  });
});

describe('buildDeviceViewRequestedObjectLdns', () => {
  it('15min 全选对象也显式传递已发现 LDN，避免后端重复发现对象', () => {
    const ldns = buildDeviceViewRequestedObjectLdns({}, {
      ENB00001: [{ objectLdn: 'Cellid=1' }, { objectLdn: 'Cellid=2' }],
    }, '15min');

    expect(ldns).toEqual(['Cellid=1', 'Cellid=2']);
  });

  it.each(['hourly', 'daily', 'weekly', 'monthly'] as const)(
    '%s 全选对象保持不过滤，避免改变旧聚合结果口径',
    (granularity) => {
      const ldns = buildDeviceViewRequestedObjectLdns({}, {
        ENB00001: [{ objectLdn: 'Cellid=1' }, { objectLdn: 'Cellid=2' }],
      }, granularity);

      expect(ldns).toEqual([]);
    },
  );

  it('聚合粒度保持 5G 推荐对象默认下推的旧行为', () => {
    const ldns = buildDeviceViewRequestedObjectLdns({}, {
      GNB00001: [
        { objectLdn: 'Type=gNB,gNBID=1' },
        { objectLdn: 'Type=Cell,CellID=1,PLMNID=46000' },
        { objectLdn: 'Type=Slice,SNSSAI=1' },
      ],
    }, 'hourly');

    expect(ldns).toEqual(['Type=gNB,gNBID=1', 'Type=Cell,CellID=1,PLMNID=46000']);
  });

  it('手动选择子集时只传递子集，跨设备去重', () => {
    const ldns = buildDeviceViewRequestedObjectLdns({
      ENB00001: ['Cellid=1'],
      ENB00002: ['Cellid=1', 'Cellid=3'],
    }, {
      ENB00001: [{ objectLdn: 'Cellid=1' }, { objectLdn: 'Cellid=2' }],
      ENB00002: [{ objectLdn: 'Cellid=1' }, { objectLdn: 'Cellid=3' }],
    });

    expect(ldns).toEqual(['Cellid=1', 'Cellid=3']);
  });
});

describe('isDeviceViewQueryTimeoutError', () => {
  it('识别 Axios 超时和普通 timeout 文本', () => {
    expect(isDeviceViewQueryTimeoutError({ code: 'ECONNABORTED', message: 'timeout of 30000ms exceeded' })).toBe(true);
    expect(isDeviceViewQueryTimeoutError(new Error('request timed out'))).toBe(true);
    expect(isDeviceViewQueryTimeoutError(new Error('server returned 500'))).toBe(false);
  });
});

describe('device view page state snapshot', () => {
  it('恢复设备、指标、粒度、时间范围、星期、小时段、小区对象、周期对比和已提交条件', () => {
    const filter = {
      range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [1, 2, 3],
      hours: [8, 9],
      compare: true,
    };
    const snapshot = {
      ...buildDeviceViewStateSnapshot({
        tech: 'nr',
        deviceSns: ['GNB00001'],
        cellSel: { GNB00001: ['Cell=1'] },
        metricPaths: ['K001'],
        metricsTouched: true,
        granularity: 'hourly',
        rangeTouched: true,
        filter,
        submitted: {
          tech: 'nr',
          deviceSns: ['GNB00001'],
          metricPaths: ['K001'],
          granularity: 'hourly',
          startTime: '2026-07-01T00:00:00Z',
          endTime: '2026-07-02T00:00:00Z',
          weekdays: [1, 2, 3],
          hours: [8, 9],
          compare: true,
          offsetMs: 86_400_000,
          prevStartTime: '2026-06-30T00:00:00Z',
          prevEndTime: '2026-07-01T00:00:00Z',
          allowedLdns: ['Cell=1'],
        },
        refreshed: true,
      }),
      savedAt: '2026-07-02T00:00:00.000Z',
    };

    const restored = restoreDeviceViewState(snapshot, 'UTC');

    expect(restored.tech).toBe('nr');
    expect(restored.deviceSns).toEqual(['GNB00001']);
    expect(restored.metricPaths).toEqual(['K001']);
    expect(restored.granularity).toBe('hourly');
    expect(restored.filter.range.map((d) => d.toISOString())).toEqual([
      '2026-07-01T00:00:00.000Z',
      '2026-07-02T00:00:00.000Z',
    ]);
    expect(restored.filter.weekdays).toEqual([1, 2, 3]);
    expect(restored.filter.hours).toEqual([8, 9]);
    expect(restored.filter.compare).toBe(true);
    expect(restored.cellSel).toEqual({ GNB00001: ['Cell=1'] });
    expect(restored.submitted?.allowedLdns).toEqual(['Cell=1']);
    expect(restored.shouldRestoreQuery).toBe(true);
  });

  it('未提交出图时不恢复 submitted，也不标记自动请求', () => {
    const filter = {
      range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [1],
      hours: [8],
      compare: false,
    };
    const snapshot = {
      ...buildDeviceViewStateSnapshot({
        tech: 'lte',
        deviceSns: ['ENB00001'],
        cellSel: {},
        metricPaths: ['K001'],
        metricsTouched: true,
        granularity: '15min',
        rangeTouched: true,
        filter,
        submitted: null,
        refreshed: false,
      }),
      savedAt: '2026-07-02T00:00:00.000Z',
    };

    const restored = restoreDeviceViewState(snapshot, 'UTC');

    expect(restored.deviceSns).toEqual(['ENB00001']);
    expect(restored.metricPaths).toEqual(['K001']);
    expect(restored.submitted).toBeNull();
    expect(restored.shouldRestoreQuery).toBe(false);
  });

  it('旧版本保存的空对象白名单不自动恢复查询，避免进页面触发慢查询', () => {
    const filter = {
      range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [0, 1, 2, 3, 4, 5, 6],
      hours: Array.from({ length: 24 }, (_, i) => i),
      compare: false,
    };
    const snapshot = {
      ...buildDeviceViewStateSnapshot({
        tech: 'lte',
        deviceSns: ['ENB00001'],
        cellSel: {},
        metricPaths: ['K001'],
        metricsTouched: true,
        granularity: '15min',
        rangeTouched: true,
        filter,
        submitted: {
          tech: 'lte',
          deviceSns: ['ENB00001'],
          metricPaths: ['K001'],
          granularity: '15min',
          startTime: '2026-07-01T00:00:00Z',
          endTime: '2026-07-02T00:00:00Z',
          weekdays: [0, 1, 2, 3, 4, 5, 6],
          hours: Array.from({ length: 24 }, (_, i) => i),
          compare: false,
          offsetMs: 86_400_000,
          allowedLdns: [],
        },
        refreshed: false,
      }),
      savedAt: '2026-07-02T00:00:00.000Z',
    };

    const restored = restoreDeviceViewState(snapshot, 'UTC');

    expect(restored.deviceSns).toEqual(['ENB00001']);
    expect(restored.metricPaths).toEqual(['K001']);
    expect(restored.submitted).toBeNull();
    expect(restored.shouldRestoreQuery).toBe(false);
  });

  it('按保存时间计算恢复请求的轻量节流时间', () => {
    expect(restoredDeviceViewQueryDelayMs(
      '2026-07-30T11:59:59.000Z',
      Date.parse('2026-07-30T12:00:00.000Z'),
    )).toBe(1_000);
    expect(restoredDeviceViewQueryDelayMs(
      '2026-07-30T11:59:40.000Z',
      Date.parse('2026-07-30T12:00:00.000Z'),
    )).toBe(0);
  });
});

describe('defaultRangeForGranularity', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('系统时区为 UTC 时默认结束时间使用 UTC 当前钟面', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-24T02:41:30.456Z'));

    const [, end] = defaultRangeForGranularity('15min', 'UTC');

    expect(end.format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-07-24 02:41:30.000');
  });

  it('非法系统时区按 UTC 回退，不使用浏览器本地偏移', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-24T02:41:30.456Z'));

    const [, end] = defaultRangeForGranularity('15min', 'Bad/Zone');

    expect(end.format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-07-24 02:41:30.000');
  });

  it('按粒度生成默认窗口长度', () => {
    const now = dayjs('2026-07-24 02:41:30.456');

    expect(defaultRangeForGranularity('15min', 'UTC', now).map((d) => d.format('YYYY-MM-DD HH:mm:ss')))
      .toEqual(['2026-07-23 02:41:30', '2026-07-24 02:41:30']);
    expect(defaultRangeForGranularity('hourly', 'UTC', now).map((d) => d.format('YYYY-MM-DD HH:mm:ss')))
      .toEqual(['2026-07-23 02:41:30', '2026-07-24 02:41:30']);
    expect(defaultRangeForGranularity('daily', 'UTC', now).map((d) => d.format('YYYY-MM-DD HH:mm:ss')))
      .toEqual(['2026-07-17 02:41:30', '2026-07-24 02:41:30']);
    expect(defaultRangeForGranularity('weekly', 'UTC', now).map((d) => d.format('YYYY-MM-DD HH:mm:ss')))
      .toEqual(['2026-06-24 02:41:30', '2026-07-24 02:41:30']);
    expect(defaultRangeForGranularity('monthly', 'UTC', now).map((d) => d.format('YYYY-MM-DD HH:mm:ss')))
      .toEqual(['2026-01-24 02:41:30', '2026-07-24 02:41:30']);
  });
});

describe('toDeviceViewRequestRFC3339', () => {
  it('设备性能查看提交给后端的时间按系统时区附加偏移并去掉隐藏毫秒', () => {
    expect(toDeviceViewRequestRFC3339(dayjs('2026-07-21 00:00:00.300'), 'UTC'))
      .toBe('2026-07-21T00:00:00Z');
    expect(toDeviceViewRequestRFC3339(dayjs('2026-07-21 01:00:00.300'), 'Asia/Shanghai'))
      .toBe('2026-07-21T01:00:00+08:00');
  });

  it('非法系统时区提交时间按 UTC 钟面兜底', () => {
    expect(toDeviceViewRequestRFC3339(dayjs('2026-07-21 01:00:00.300'), 'Bad/Zone'))
      .toBe('2026-07-21T01:00:00Z');
  });

  it('上一周期窗口与当前窗口使用同一系统时区提交口径', () => {
    const window = buildDeviceViewRequestTimeWindow(
      [dayjs('2026-07-24 00:00:00.300'), dayjs('2026-07-24 03:00:00.300')],
      'UTC',
    );

    expect(window).toEqual({
      startTime: '2026-07-24T00:00:00Z',
      endTime: '2026-07-24T03:00:00Z',
      prevStartTime: '2026-07-23T21:00:00Z',
      prevEndTime: '2026-07-24T00:00:00Z',
    });
  });

  it('非 UTC 系统时区下查询和上一周期窗口使用相同 RFC3339 偏移', () => {
    const window = buildDeviceViewRequestTimeWindow(
      [dayjs('2026-07-24 10:00:00.300'), dayjs('2026-07-24 13:00:00.300')],
      'Asia/Shanghai',
    );

    expect(window).toEqual({
      startTime: '2026-07-24T10:00:00+08:00',
      endTime: '2026-07-24T13:00:00+08:00',
      prevStartTime: '2026-07-24T07:00:00+08:00',
      prevEndTime: '2026-07-24T10:00:00+08:00',
    });
  });

  it('接口 actual range 保留系统钟面后再用于上一周期计算', () => {
    const actualRange = actualRangeFromMeta({
      actualStartTime: '2026-07-24T02:00:00Z',
      actualEndTime: '2026-07-24T05:00:00Z',
    });

    expect(actualRange?.map((d) => d.format('YYYY-MM-DD HH:mm:ss'))).toEqual([
      '2026-07-24 02:00:00',
      '2026-07-24 05:00:00',
    ]);
    expect(buildDeviceViewRequestTimeWindow(actualRange!, 'UTC')).toEqual({
      startTime: '2026-07-24T02:00:00Z',
      endTime: '2026-07-24T05:00:00Z',
      prevStartTime: '2026-07-23T23:00:00Z',
      prevEndTime: '2026-07-24T02:00:00Z',
    });
  });

  it('接口 actual range 带非 UTC 偏移时保留后端系统钟面', () => {
    const actualRange = actualRangeFromMeta({
      actualStartTime: '2026-07-24T10:00:00+08:00',
      actualEndTime: '2026-07-24T13:00:00+08:00',
    });

    expect(actualRange?.map((d) => d.format('YYYY-MM-DD HH:mm:ss'))).toEqual([
      '2026-07-24 10:00:00',
      '2026-07-24 13:00:00',
    ]);
    expect(buildDeviceViewRequestTimeWindow(actualRange!, 'Asia/Shanghai')).toEqual({
      startTime: '2026-07-24T10:00:00+08:00',
      endTime: '2026-07-24T13:00:00+08:00',
      prevStartTime: '2026-07-24T07:00:00+08:00',
      prevEndTime: '2026-07-24T10:00:00+08:00',
    });
  });
});
