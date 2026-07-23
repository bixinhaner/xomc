import { describe, expect, it } from 'vitest';
import dayjs from 'dayjs';
import { PM_QUERY_SELECTION_LIMIT } from '@/constants/pmQueryLimits';
import type { DashboardExportSelection } from '@core/utils/kpiExportParams';
import {
  buildDeviceViewExportInput,
  isDeviceViewDeviceSelectionOverLimit,
  isDeviceViewMetricSelectionOverLimit,
  toDeviceViewRequestISOString,
} from './DeviceListPane';

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

describe('toDeviceViewRequestISOString', () => {
  it('设备性能查看提交给后端的时间不携带 DatePicker 隐藏毫秒', () => {
    expect(toDeviceViewRequestISOString(dayjs('2026-07-21T00:00:00.300+08:00')))
      .toBe('2026-07-20T16:00:00.000Z');
    expect(toDeviceViewRequestISOString(dayjs('2026-07-21T01:00:00.300+08:00')))
      .toBe('2026-07-20T17:00:00.000Z');
  });
});
