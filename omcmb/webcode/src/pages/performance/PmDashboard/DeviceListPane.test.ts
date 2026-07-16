import { describe, expect, it } from 'vitest';
import type { DashboardExportSelection } from '@core/utils/kpiExportParams';
import { buildDeviceViewExportInput } from './DeviceListPane';

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
      'zh-CN',
      new Date(2026, 6, 16, 18, 40, 5),
    );

    expect(input.sourceType).toBe('device_view');
    expect(input.taskName).toBe('KPI导出_设备性能查看_20260716_184005');
    expect(input.taskName).not.toContain('仪表盘');
  });

  it('英文界面下使用英文设备性能查看任务名前缀', () => {
    const input = buildDeviceViewExportInput(
      selection,
      'en-US',
      new Date(2026, 6, 16, 18, 40, 5),
    );

    expect(input.sourceType).toBe('device_view');
    expect(input.taskName).toBe('KPI_Export_Device_Performance_View_20260716_184005');
  });
});
