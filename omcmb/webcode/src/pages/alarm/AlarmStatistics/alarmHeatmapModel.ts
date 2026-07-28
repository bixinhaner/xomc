import type {
  AlarmHeatmapBySeverity,
  HeatmapData,
} from '@core/types/dashboard';

export type HeatmapSeverity = 'all' | 'critical' | 'major' | 'minor' | 'warning';

const BLUE_HEATMAP_COLORS = ['#e6f4ff', '#bae0ff', '#69b1ff', '#1677ff', '#003eb3'];

export const HEATMAP_COLORS_BY_SEVERITY: Record<HeatmapSeverity, string[]> = {
  all: BLUE_HEATMAP_COLORS,
  critical: ['#fff1f0', '#ffccc7', '#ff7875', '#f5222d', '#820014'],
  major: ['#fff7e6', '#ffd591', '#ffa940', '#fa8c16', '#873800'],
  minor: ['#fffbe6', '#fff1b8', '#ffec3d', '#fadb14', '#876800'],
  warning: BLUE_HEATMAP_COLORS,
};

interface AlarmHeatmapAPI {
  getAlarmHeatmap(params: { days: number }): Promise<HeatmapData>;
  getAlarmHeatmapBySeverity(params: {
    days: number;
    severity?: string;
  }): Promise<AlarmHeatmapBySeverity>;
}

export async function loadAlarmHeatmap(
  api: AlarmHeatmapAPI,
  days: number,
  severity: HeatmapSeverity,
): Promise<HeatmapData> {
  if (severity === 'all') {
    return api.getAlarmHeatmap({ days });
  }
  const result = await api.getAlarmHeatmapBySeverity({ days, severity });
  return result.data;
}
