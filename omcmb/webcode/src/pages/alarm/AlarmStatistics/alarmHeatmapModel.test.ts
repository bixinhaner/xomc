import { describe, expect, it, vi } from 'vitest';
import {
  HEATMAP_COLORS_BY_SEVERITY,
  loadAlarmHeatmap,
} from './alarmHeatmapModel';

const heatmap = {
  days_of_week: Array.from({ length: 7 }, (_, day) => ({
    day,
    hours: Array.from({ length: 24 }, () => 0),
  })),
  max_count: 0,
};

describe('loadAlarmHeatmap', () => {
  it('uses the aggregate endpoint for all severities', async () => {
    const api = {
      getAlarmHeatmap: vi.fn().mockResolvedValue(heatmap),
      getAlarmHeatmapBySeverity: vi.fn(),
    };

    const result = await loadAlarmHeatmap(api, 7, 'all');

    expect(api.getAlarmHeatmap).toHaveBeenCalledWith({ days: 7 });
    expect(api.getAlarmHeatmapBySeverity).not.toHaveBeenCalled();
    expect(result).toBe(heatmap);
  });

  it('uses the severity endpoint and unwraps its data', async () => {
    const api = {
      getAlarmHeatmap: vi.fn(),
      getAlarmHeatmapBySeverity: vi.fn().mockResolvedValue({
        severity: 'critical',
        data: heatmap,
      }),
    };

    const result = await loadAlarmHeatmap(api, 30, 'critical');

    expect(api.getAlarmHeatmapBySeverity).toHaveBeenCalledWith({
      days: 30,
      severity: 'critical',
    });
    expect(result).toBe(heatmap);
  });
});

describe('HEATMAP_COLORS_BY_SEVERITY', () => {
  it('uses distinct severity color families', () => {
    expect(HEATMAP_COLORS_BY_SEVERITY.critical).not.toEqual(HEATMAP_COLORS_BY_SEVERITY.major);
    expect(HEATMAP_COLORS_BY_SEVERITY.major).not.toEqual(HEATMAP_COLORS_BY_SEVERITY.minor);
    expect(HEATMAP_COLORS_BY_SEVERITY.warning).not.toEqual(HEATMAP_COLORS_BY_SEVERITY.critical);
    expect(HEATMAP_COLORS_BY_SEVERITY.all).toEqual(HEATMAP_COLORS_BY_SEVERITY.warning);
  });
});
