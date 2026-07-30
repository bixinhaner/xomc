import { describe, expect, it } from 'vitest';
import { buildTopAlarmDeviceChart, escapeChartTooltipText } from './alarmCharts';

describe('buildTopAlarmDeviceChart', () => {
  it('keeps full SN and builds four stacked severity series', () => {
    const chart = buildTopAlarmDeviceChart([
      {
        deviceSN: '460001234567890',
        technology: 'lte',
        alarmCount: 10,
        critical: 4,
        major: 3,
        minor: 2,
        warning: 1,
      },
    ], {
      critical: 'Critical',
      major: 'Major',
      minor: 'Minor',
      warning: 'Warning',
    });

    expect(chart.labels).toEqual(['460001234567890']);
    expect(chart.series).toHaveLength(4);
    expect(chart.series.every((series) => series.stack === 'alarm-total')).toBe(true);
    expect(chart.series.map((series) => series.data[0])).toEqual([4, 3, 2, 1]);
  });

  it('uses backend ranking order and limits the chart to ten devices', () => {
    const devices = Array.from({ length: 12 }, (_, index) => ({
      deviceSN: `SN-${String(index).padStart(2, '0')}`,
      technology: 'lte',
      alarmCount: 12 - index,
      critical: 12 - index,
      major: 0,
      minor: 0,
      warning: 0,
    }));

    const chart = buildTopAlarmDeviceChart(devices, {
      critical: 'Critical',
      major: 'Major',
      minor: 'Minor',
      warning: 'Warning',
    });

    expect(chart.devices).toHaveLength(10);
    expect(chart.devices.at(-1)?.deviceSN).toBe('SN-00');
    expect(chart.devices[0]?.deviceSN).toBe('SN-09');
  });

  it('escapes device identifiers before placing them in HTML tooltips', () => {
    expect(escapeChartTooltipText('<SN&1>')).toBe('&lt;SN&amp;1&gt;');
  });
});
