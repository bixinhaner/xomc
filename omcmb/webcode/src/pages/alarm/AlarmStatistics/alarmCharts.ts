import type { TopAlarmDevice } from '@core/types/dashboard';

export const SEVERITY_COLORS = {
  critical: '#F5222D',
  major: '#FA8C16',
  minor: '#FADB14',
  warning: '#1677FF',
} as const;

interface SeverityLabels {
  critical: string;
  major: string;
  minor: string;
  warning: string;
}

export function buildTopAlarmDeviceChart(
  devices: TopAlarmDevice[] | undefined,
  labels: SeverityLabels,
) {
  const chartDevices = (devices ?? []).slice(0, 10).reverse();
  return {
    devices: chartDevices,
    labels: chartDevices.map((device) => device.deviceSN),
    series: [
      {
        name: labels.critical,
        data: chartDevices.map((device) => device.critical),
        color: SEVERITY_COLORS.critical,
        stack: 'alarm-total',
      },
      {
        name: labels.major,
        data: chartDevices.map((device) => device.major),
        color: SEVERITY_COLORS.major,
        stack: 'alarm-total',
      },
      {
        name: labels.minor,
        data: chartDevices.map((device) => device.minor),
        color: SEVERITY_COLORS.minor,
        stack: 'alarm-total',
      },
      {
        name: labels.warning,
        data: chartDevices.map((device) => device.warning),
        color: SEVERITY_COLORS.warning,
        stack: 'alarm-total',
      },
    ],
  };
}

export function escapeChartTooltipText(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}
