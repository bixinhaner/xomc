/**
 * dashboardApi 契约测试（#22 关键 API + Dashboard 热力图集成）：
 *   - getSummary：BackendDashboardSummary → DashboardSummary 映射（device/alarm 计数、
 *     kpi_overview 动态键挑选 + 缺键兜 0、kpi_deltas 字符串字面量断言）。
 *   - 告警热力图集成：getAlarmHeatmap / getAlarmHeatmapBySeverity 打对端点、带 days/severity
 *     query，回 7×24 热力矩阵 + max_count；以及效率指标 getAlarmEfficiency。
 *   - getDeviceStatusPie / getRegionStats / getAlarmTypePie 映射。
 *   - 错误码 429/500 原样抛（不吞错，hook 走 React Query 错误态）。
 *
 * 用 vi.mock 替换 http 客户端（参照 alarmApi.test.ts 既有模式）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type {
  HeatmapData,
  AlarmHeatmapBySeverity,
  EfficiencyMetrics,
} from '../../../types/dashboard';

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }));
vi.mock('../../http', () => ({
  default: { get: getMock, post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));

import { dashboardApi } from '../dashboardApi';

beforeEach(() => {
  getMock.mockReset();
});

// 构造一个标准 7×24 热力矩阵（周一..周日，每天 24 小时）。
function buildHeatmap(maxCount: number): HeatmapData {
  return {
    days_of_week: Array.from({ length: 7 }, (_, day) => ({
      day,
      hours: Array.from({ length: 24 }, (_, h) => (h === 9 ? maxCount : 0)),
    })),
    max_count: maxCount,
  };
}

describe('dashboardApi.getSummary — Backend → DashboardSummary 映射', () => {
  it('完整映射：设备/告警计数 + kpi_overview 动态键挑选 + kpi_deltas 字面量断言', async () => {
    getMock.mockResolvedValue({
      data: {
        device_stats: { total: 100, online: 80, offline: 15, alarm: 5 },
        alarm_stats: { critical: 1, major: 2, minor: 3, warning: 4, total: 10 },
        kpi_overview: {
          RRC_CONN_SETUP_SR: 99.5,
          ERAB_SETUP_SR: 98.2,
          NR_SA_HO_SR: 97.1,
          NR_PDCP_RATE_DL: 1234.5,
          // 故意不给 CALL_DROP_RATE / NR_PRB_UTIL_DL，验证缺键兜 0
        },
        kpi_deltas: {
          RRC_CONN_SETUP_SR: {
            current_value: 99.5,
            previous_value: 99.0,
            change_percent: 0.5,
            trend: 'up',
            compare_type: 'yesterday',
          },
        },
        recent_alarms: [],
        timestamp: '2026-06-10T00:00:00Z',
      },
    });
    const s = await dashboardApi.getSummary();
    expect(getMock.mock.calls[0][0]).toBe('/dashboard/summary');
    expect(s.deviceCounts).toEqual({ total: 100, online: 80, offline: 15, alarm: 5 });
    expect(s.alarmCounts.total).toBe(10);
    expect(s.kpiSummary.rrcSuccRate).toBe(99.5);
    expect(s.kpiSummary.dlThroughput).toBe(1234.5);
    // 缺键兜 0（不是 undefined）
    expect(s.kpiSummary.radioDrop).toBe(0);
    expect(s.kpiSummary.prbUtil).toBe(0);
    // 后端 snake_case 被 mapBackendSummary 转为前端 camelCase 契约（Issue D 修复）
    expect(s.kpiDeltas.RRC_CONN_SETUP_SR.trend).toBe('up');
    expect(s.kpiDeltas.RRC_CONN_SETUP_SR.compareType).toBe('yesterday');
    expect(s.kpiDeltas.RRC_CONN_SETUP_SR.changePercent).toBe(0.5);
    expect(s.kpiDeltas.RRC_CONN_SETUP_SR.currentValue).toBe(99.5);
    expect(s.kpiDeltas.RRC_CONN_SETUP_SR.previousValue).toBe(99.0);
  });

  it('kpi_deltas 缺省（null）时不崩，返空对象', async () => {
    getMock.mockResolvedValue({
      data: {
        device_stats: { total: 0, online: 0, offline: 0, alarm: 0 },
        alarm_stats: { critical: 0, major: 0, minor: 0, warning: 0, total: 0 },
        kpi_overview: {},
        kpi_deltas: null,
        recent_alarms: [],
        timestamp: '2026-06-10T00:00:00Z',
      },
    });
    const s = await dashboardApi.getSummary();
    expect(s.kpiDeltas).toEqual({});
    // kpi_overview 全缺时所有 KPI 兜 0
    expect(s.kpiSummary.rrcSuccRate).toBe(0);
  });

  it('500 错误原样抛（不吞错）', async () => {
    getMock.mockRejectedValue({ response: { status: 500 } });
    await expect(dashboardApi.getSummary()).rejects.toEqual({ response: { status: 500 } });
  });
});

describe('dashboardApi — 告警热力图集成（Dashboard heatmap）', () => {
  it('getAlarmHeatmap 打 /dashboard/alarm-heatmap 带 days，回 7×24 矩阵 + max_count', async () => {
    const heatmap = buildHeatmap(42);
    getMock.mockResolvedValue({ data: heatmap });
    const out = await dashboardApi.getAlarmHeatmap({ days: 30 });
    const [url, opts] = getMock.mock.calls[0];
    expect(url).toBe('/dashboard/alarm-heatmap');
    expect(opts.params.days).toBe(30);
    // 矩阵结构完整：7 天 × 每天 24 小时
    expect(out.days_of_week).toHaveLength(7);
    expect(out.days_of_week[0].hours).toHaveLength(24);
    // max_count 用于热力色阶上界
    expect(out.max_count).toBe(42);
    // 周一 9 点峰值落点正确
    expect(out.days_of_week[0].hours[9]).toBe(42);
    expect(out.days_of_week[0].hours[0]).toBe(0);
    // 周日（day=6）也在矩阵内
    expect(out.days_of_week[6].day).toBe(6);
  });

  it('getAlarmHeatmap 兼容残留统一信封，避免热度图误判为空', async () => {
    const heatmap = buildHeatmap(42);
    getMock.mockResolvedValue({ data: { ret: 1, msg: 'ok', data: heatmap } });
    const out = await dashboardApi.getAlarmHeatmap({ days: 30 });
    expect(out.days_of_week).toHaveLength(7);
    expect(out.max_count).toBe(42);
    expect(out.days_of_week[0].hours[9]).toBe(42);
  });

  it('getAlarmHeatmapBySeverity 打 by-severity 端点 + 带 severity query', async () => {
    const payload: AlarmHeatmapBySeverity = { severity: 'critical', data: buildHeatmap(7) };
    getMock.mockResolvedValue({ data: payload });
    const out = await dashboardApi.getAlarmHeatmapBySeverity({ days: 30, severity: 'critical' });
    const [url, opts] = getMock.mock.calls[0];
    expect(url).toBe('/dashboard/alarm-heatmap-by-severity');
    expect(opts.params.days).toBe(30);
    expect(opts.params.severity).toBe('critical');
    expect(out.severity).toBe('critical');
    expect(out.data.days_of_week).toHaveLength(7);
    expect(out.data.max_count).toBe(7);
  });

  it('getAlarmHeatmapBySeverity 兼容残留统一信封', async () => {
    const payload: AlarmHeatmapBySeverity = { severity: 'critical', data: buildHeatmap(7) };
    getMock.mockResolvedValue({ data: { ret: 1, msg: 'ok', data: payload } });
    const out = await dashboardApi.getAlarmHeatmapBySeverity({ days: 30, severity: 'critical' });
    expect(out.severity).toBe('critical');
    expect(out.data.max_count).toBe(7);
  });

  it('全零热力图（无告警）仍是合法 7×24 矩阵、max_count=0', async () => {
    getMock.mockResolvedValue({ data: buildHeatmap(0) });
    const out = await dashboardApi.getAlarmHeatmap({ days: 7 });
    expect(out.max_count).toBe(0);
    const totalAlarms = out.days_of_week.reduce(
      (sum, d) => sum + d.hours.reduce((a, b) => a + b, 0),
      0,
    );
    expect(totalAlarms).toBe(0);
  });

  it('告警效率指标 getAlarmEfficiency 打 /dashboard/alarm-efficiency，null 指标透传', async () => {
    const eff: EfficiencyMetrics = {
      severity: 'all',
      acknowledged_count: 5,
      cleared_count: 3,
      total_count: 8,
      avg_acknowledge_minutes: 12.5,
      avg_resolve_minutes: null, // 无清除数据 → null（前端区分"0"与"无数据"）
      acknowledge_rate: 62.5,
      clear_rate: null,
      daily_trend: [
        { date: '2026-06-09', avg_acknowledge_minutes: 10, avg_resolve_minutes: null },
      ],
    };
    getMock.mockResolvedValue({ data: eff });
    const out = await dashboardApi.getAlarmEfficiency();
    expect(getMock.mock.calls[0][0]).toBe('/dashboard/alarm-efficiency');
    expect(out.avg_acknowledge_minutes).toBe(12.5);
    expect(out.avg_resolve_minutes).toBeNull();
    expect(out.daily_trend[0].avg_resolve_minutes).toBeNull();
  });

  it('热力图端点 429（限流）原样抛', async () => {
    getMock.mockRejectedValue({ response: { status: 429 } });
    await expect(dashboardApi.getAlarmHeatmap({ days: 30 })).rejects.toEqual({
      response: { status: 429 },
    });
  });
});

describe('dashboardApi — 图表数据映射', () => {
  it('getDeviceStatusPie：后端 status→count map 转 {name,value}[]', async () => {
    getMock.mockResolvedValue({ data: { 在线: 80, 离线: 20 } });
    const out = await dashboardApi.getDeviceStatusPie();
    expect(getMock.mock.calls[0][0]).toBe('/dashboard/device-status');
    expect(out).toContainEqual({ name: '在线', value: 80 });
    expect(out).toContainEqual({ name: '离线', value: 20 });
  });

  it('getRegionStats：device_count/online_count → total/online，offline 由差值派生', async () => {
    getMock.mockResolvedValue({
      data: [{ region: '华东', device_count: 100, online_count: 70, alarm_count: 5 }],
    });
    const out = await dashboardApi.getRegionStats();
    expect(getMock.mock.calls[0][0]).toBe('/dashboard/region-stats');
    expect(out[0]).toEqual({ region: '华东', total: 100, online: 70, offline: 30 });
  });

  it('getTopAlarmDevices：从 summary.recent_alarms 提取，缺省（null）兜空数组', async () => {
    getMock.mockResolvedValue({ data: { recent_alarms: null } });
    const out = await dashboardApi.getTopAlarmDevices();
    expect(out).toEqual([]);
  });
});
