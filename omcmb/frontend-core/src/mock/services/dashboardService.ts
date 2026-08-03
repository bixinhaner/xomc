import { mockDashboardSummary, mockDashboardChartData, mockDashboardWidgets } from '../data/dashboard';
import type { DashboardSummary, DashboardChartData, KPITimeSeriesPoint } from '../data/dashboard';
import type {
  KPIDefinitionsResponse,
  KPIDefinitionItem,
  KPITechDefinitions,
  KPILayout,
  KPILayoutPanel,
  DashboardKPIGranularity,
  DashboardKPITimeSeriesSnapshot,
} from '../../types/dashboard';
import { delay } from '../utils';
import { useAppStore } from '../../store/appStore';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';

dayjs.extend(utc);
dayjs.extend(timezone);

// issue #213 S2：首页 KPI 全局布局 Mock 数据。
// 与后端 seed（migrations/seed/000002_dashboard_kpi_layout_seed.sql）等价：
// LTE 6 图 / NR 2 图 / GSM 3 图，12 列网格，半宽 w=6 / 满宽 w=12 / 统一 h=8。
// dev:mock 下首页能按布局渲染、与真实接口同形状。
const MOCK_KPI_LAYOUTS: Record<string, KPILayoutPanel[]> = {
  lte: [
    { title: 'dashboard.panel.traffic', metrics: ['K900010015', 'K900010016', 'K900010040', 'K900010041'], x: 0, y: 0, w: 6, h: 8, chartType: 'line' },
    { title: 'dashboard.panel.availability', metrics: ['K900010076'], x: 6, y: 0, w: 6, h: 8, chartType: 'line' },
    { title: 'dashboard.panel.utilization', metrics: ['K900010014', 'K900010013'], x: 0, y: 8, w: 6, h: 8, chartType: 'line' },
    { title: 'dashboard.panel.accessibility', metrics: ['K900010006', 'K900010002', 'K900010005', 'K900010029'], x: 6, y: 8, w: 6, h: 8, chartType: 'line' },
    { title: 'dashboard.panel.retainability', metrics: ['K900010027'], x: 0, y: 16, w: 6, h: 8, chartType: 'line' },
    { title: 'dashboard.panel.mobility', metrics: ['K900010017', 'K900010022', 'K900010021', 'K900010026'], x: 6, y: 16, w: 6, h: 8, chartType: 'line' },
  ],
  nr: [
    { title: 'dashboard.panel.traffic', metrics: ['KGNB0511', 'KGNB0510', 'KGNB0517', 'KGNB0516'], x: 0, y: 0, w: 6, h: 8, chartType: 'line' },
    { title: 'dashboard.panel.utilization', metrics: ['KGNB0506', 'KGNB0505'], x: 6, y: 0, w: 6, h: 8, chartType: 'line' },
  ],
  gsm: [
    { title: 'dashboard.panel.accessibility', metrics: ['KGSM0102'], x: 0, y: 0, w: 6, h: 8, chartType: 'line' },
    { title: 'dashboard.panel.retainability', metrics: ['KGSM0103'], x: 6, y: 0, w: 6, h: 8, chartType: 'line' },
    { title: 'dashboard.panel.mobility', metrics: ['KGSM0101'], x: 0, y: 8, w: 12, h: 8, chartType: 'line' },
  ],
};

// issue #213 Phase1：Dashboard KPI 动态定义 Mock 数据。
// 与对照表（symbolic key → K 编号 → 中文名 → 单位 → Panel）一致；none 项 available=false。
// 元组：[key, k_code, cn_name, unit, panel, needs_review, available]
type KpiDefTuple = [string, string, string, string, string, boolean, boolean];
const KPI_DEF_TUPLES: Record<string, KpiDefTuple[]> = {
  lte: [
    ['K900010015', 'K900010015', '下行数据业务流量', 'MByte', 'traffic', false, true],
    ['K900010016', 'K900010016', '上行数据业务流量', 'MByte', 'traffic', false, true],
    ['K900010040', 'K900010040', 'UE下行速率', 'Mbps', 'traffic', false, true],
    ['K900010041', 'K900010041', 'UE上行速率', 'Mbps', 'traffic', false, true],
    ['K900010076', 'K900010076', '小区可用率', '%', 'availability', false, true],
    ['K900010014', 'K900010014', '下行PRB平均占用率', '%', 'utilization', false, true],
    ['K900010013', 'K900010013', '上行PRB平均占用率', '%', 'utilization', false, true],
    ['K900010006', 'K900010006', '无线初始连接成功率', '%', 'accessibility', false, true],
    ['K900010002', 'K900010002', 'RRC连接建立成功率', '%', 'accessibility', false, true],
    ['K900010005', 'K900010005', 'E-RAB建立成功率', '%', 'accessibility', false, true],
    ['K900010029', 'K900010029', 'CSFB成功率', '%', 'accessibility', false, true],
    ['K900010027', 'K900010027', 'E-RAB掉线率', '%', 'retainability', false, true],
    ['K900010017', 'K900010017', '同频切换成功率-切出', '%', 'mobility', false, true],
    ['K900010022', 'K900010022', '同频切换成功率-切入', '%', 'mobility', false, true],
    ['K900010021', 'K900010021', 'eNB间切换成功率-切出', '%', 'mobility', false, true],
    ['K900010026', 'K900010026', 'eNB间切换成功率-切入', '%', 'mobility', false, true],
  ],
  nr: [
    ['KGNB0511', 'KGNB0511', 'PDCP下行业务字节数', 'MByte', 'traffic', false, true],
    ['KGNB0510', 'KGNB0510', 'PDCP上行业务字节数', 'MByte', 'traffic', false, true],
    ['KGNB0517', 'KGNB0517', '下行用户平均速率', 'Mbps', 'traffic', false, true],
    ['KGNB0516', 'KGNB0516', '上行用户平均速率', 'Mbps', 'traffic', false, true],
    ['KGNB0506', 'KGNB0506', '下行PRB平均利用率', '%', 'utilization', false, true],
    ['KGNB0505', 'KGNB0505', '上行PRB平均利用率', '%', 'utilization', false, true],
  ],
  gsm: [
    ['KGSM0102', 'KGSM0102', '电话成功率', '%', 'accessibility', false, true],
    ['KGSM0103', 'KGSM0103', '电话掉线率', '%', 'retainability', false, true],
    ['KGSM0101', 'KGSM0101', 'Handover切换成功率', '%', 'mobility', false, true],
  ],
};

const mockKPIDefinitions: KPIDefinitionsResponse = (() => {
  const technologies: KPITechDefinitions[] = Object.entries(KPI_DEF_TUPLES).map(
    ([tech, tuples]) => ({
      tech,
      items: tuples.map<KPIDefinitionItem>(
        ([key, kCode, cnName, unit, panel, needsReview, available]) => ({
          key,
          k_code: kCode,
          cn_name: cnName,
          unit,
          panel,
          needs_review: needsReview,
          available,
        }),
      ),
    }),
  );
  const total = technologies.reduce((sum, t) => sum + t.items.length, 0);
  return { technologies, total };
})();

function buildMockKPIBucketTimes(
  start: Date,
  end: Date,
  granularity: DashboardKPIGranularity,
): string[] {
  const systemTimezone = useAppStore.getState().systemTimezone || 'UTC';
  const result: string[] = [];

  if (granularity === 'daily' || granularity === 'weekly') {
    let cursor = dayjs(start).tz(systemTimezone);
    while (cursor.isBefore(end)) {
      result.push(cursor.format('YYYY-MM-DDTHH:mm:ssZ'));
      const nextWallClock = cursor
        .add(1, granularity === 'weekly' ? 'week' : 'day')
        .format('YYYY-MM-DD HH:mm:ss');
      cursor = dayjs.tz(nextWallClock, 'YYYY-MM-DD HH:mm:ss', systemTimezone);
    }
    return result;
  }

  let cursor = dayjs(start);
  while (cursor.isBefore(end)) {
    result.push(cursor.tz(systemTimezone).format('YYYY-MM-DDTHH:mm:ssZ'));
    cursor = cursor.add(1, 'hour');
  }
  return result;
}

// KPI 配置（用于动态生成数据）
const KPI_CONFIG: Record<string, { base: number; variance: number }> = {
  // LTE Traffic
  K900010015: { base: 450, variance: 80 },
  K900010016: { base: 120, variance: 30 },
  K900010040: { base: 76, variance: 15 },
  K900010041: { base: 18, variance: 5 },
  // LTE Availability
  K900010076: { base: 99.5, variance: 0.3 },
  // LTE Utilization
  K900010014: { base: 65, variance: 15 },
  K900010013: { base: 45, variance: 12 },
  // LTE Accessibility
  K900010006: { base: 98.5, variance: 1.5 },
  K900010002: { base: 99.2, variance: 0.8 },
  K900010005: { base: 98.8, variance: 1.2 },
  K900010029: { base: 97.5, variance: 2.5 },
  // LTE Retainability
  K900010027: { base: 0.05, variance: 0.02 },
  // LTE Mobility
  K900010017: { base: 98.2, variance: 1.8 },
  K900010022: { base: 98.5, variance: 1.5 },
  K900010021: { base: 97.8, variance: 2.2 },
  K900010026: { base: 98.0, variance: 2.0 },
  // NR Traffic
  KGNB0511: { base: 650, variance: 120 },
  KGNB0510: { base: 180, variance: 45 },
  KGNB0517: { base: 120, variance: 25 },
  KGNB0516: { base: 35, variance: 10 },
  // NR Utilization
  KGNB0506: { base: 55, variance: 18 },
  KGNB0505: { base: 38, variance: 15 },
  // GSM
  KGSM0102: { base: 97, variance: 2.5 },
  KGSM0103: { base: 0.08, variance: 0.03 },
  KGSM0101: { base: 96.5, variance: 3.0 },
};

export const dashboardService = {
  async getDashboardData(): Promise<{
    summary: DashboardSummary;
    chartData: DashboardChartData;
    widgets: typeof mockDashboardWidgets;
  }> {
    await delay(200, 400);
    return {
      summary: { ...mockDashboardSummary },
      chartData: { ...mockDashboardChartData },
      widgets: { ...mockDashboardWidgets },
    };
  },

  async getSummary(): Promise<DashboardSummary> {
    await delay(100, 200);
    return { ...mockDashboardSummary };
  },

  async getChartData(): Promise<DashboardChartData> {
    await delay(150, 300);
    return { ...mockDashboardChartData };
  },

  async getAlarmTrend(
    days = 7,
    _metric: 'raised' | 'active' = 'raised',
  ): Promise<DashboardChartData['alarmTrend']> {
    await delay(100, 200);
    return mockDashboardChartData.alarmTrend.slice(-days);
  },

  async getDeviceStatusPie(): Promise<DashboardChartData['deviceStatusPie']> {
    await delay(80, 150);
    return mockDashboardChartData.deviceStatusPie;
  },

  async getTopAlarmDevices(): Promise<DashboardChartData['topAlarmDevices']> {
    await delay(80, 150);
    return mockDashboardChartData.topAlarmDevices;
  },

  async getKPITrend(kpiCode: string): Promise<Array<[string, number]>> {
    await delay(100, 200);
    return mockDashboardChartData.kpiTimeSeries[kpiCode] ?? [];
  },

  async getRegionStats(): Promise<DashboardChartData['deviceByRegion']> {
    await delay(80, 150);
    return mockDashboardChartData.deviceByRegion;
  },

  async getWidgets(): Promise<{ id: string; user_id: string; layout: unknown; created_at: string; updated_at: string }> {
    await delay(80, 150);
    return {
      id: '',
      user_id: '',
      layout: mockDashboardWidgets,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
  },

  async saveWidgets(layout: unknown): Promise<{ id: string; user_id: string; layout: unknown; created_at: string; updated_at: string }> {
    await delay(100, 200);
    return {
      id: '',
      user_id: '',
      layout,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
  },

  async getAlarmTypePie(): Promise<DashboardChartData['alarmTypePie']> {
    await delay(80, 150);
    return mockDashboardChartData.alarmTypePie;
  },

  async getKPITimeSeries(
    kpiNames?: string[],
    startTime?: string,
    endTime?: string,
    granularity: DashboardKPIGranularity = 'hourly',
    _technology?: string,
  ): Promise<DashboardChartData['kpiTimeSeries']> {
    await delay(100, 200);

    const result: DashboardChartData['kpiTimeSeries'] = {};

    // 解析时间范围
    const start = startTime ? new Date(startTime) : new Date(Date.now() - 24 * 60 * 60 * 1000);
    const end = endTime ? new Date(endTime) : new Date();

    const bucketTimes = buildMockKPIBucketTimes(start, end, granularity);

    // 对每个请求的KPI动态生成数据
    if (kpiNames && kpiNames.length > 0) {
      for (const kpiName of kpiNames) {
        const kpiConfig = KPI_CONFIG[kpiName];
        const base = kpiConfig?.base ?? 100;
        const variance = kpiConfig?.variance ?? 20;

        // 生成覆盖完整时间范围的数据
        const timeSeries: KPITimeSeriesPoint[] = [];
        for (const timeStr of bucketTimes) {
          // 真实成功响应会返回系统时区钟面及其 RFC3339 offset；Mock 必须保持同一契约，
          // 否则首页会把 00:00+08:00 错画到 16:00 槽。
          const value = Math.max(0, base + (Math.random() - 0.5) * 2 * variance);
          timeSeries.push([timeStr, parseFloat(value.toFixed(2))]);
        }

        result[kpiName] = timeSeries;
      }
    }

    return result;
  },

  async getKPITimeSeriesWithProgress(
    kpiNames: string[],
    startTime: string,
    endTime: string,
    granularity: DashboardKPIGranularity,
    technology?: string,
  ): Promise<DashboardKPITimeSeriesSnapshot> {
    const series = await this.getKPITimeSeries(
      kpiNames, startTime, endTime, granularity, technology,
    );
    return {
      series,
      periodProgress: [],
      progressState: granularity === 'hourly' ? 'not_applicable' : 'available',
    };
  },

  // issue #213 Phase1：Dashboard KPI 动态定义（Mock）。
  // 真实数据源是后端别名表 + indicator 库；Mock 给一份与对照表一致的最小定义集，
  // 让 dev:mock 下首页能动态加载指标列表。snake_case 与真实 API / 后端 JSON 对齐。
  async getKPIDefinitions(): Promise<KPIDefinitionsResponse> {
    await delay(60, 120);
    return mockKPIDefinitions;
  },

  // issue #213 S2：首页 KPI 全局布局（Mock）。按制式返回 seed 等价布局。
  async getKPILayout(tech: string): Promise<KPILayout> {
    await delay(60, 120);
    return {
      tech,
      panels: MOCK_KPI_LAYOUTS[tech] ?? [],
      updatedAt: new Date().toISOString(),
    };
  },

  // issue #213 S3：保存首页 KPI 全局布局（Mock）。
  // dev:mock 下写回进程内 MOCK_KPI_LAYOUTS，后续 getKPILayout 即读到新布局，
  // 模拟「最后写入生效 + 存完对所有用户生效」。
  async saveKPILayout(tech: string, panels: KPILayoutPanel[]): Promise<KPILayout> {
    await delay(80, 160);
    MOCK_KPI_LAYOUTS[tech] = panels.map((p) => ({ ...p, metrics: [...p.metrics] }));
    return {
      tech,
      panels: MOCK_KPI_LAYOUTS[tech],
      updatedAt: new Date().toISOString(),
    };
  },

  async getDeviceStatusByType(): Promise<Record<string, { online: number; offline: number; alarm: number }>> {
    await delay(80, 150);
    return {
      lte: { online: 120, offline: 15, alarm: 8 },
      nr: { online: 45, offline: 5, alarm: 3 },
      gsm: { online: 10, offline: 10, alarm: 0 },
    };
  },

  async getKPITrendComparison(kpiName: string, compareWith: 'yesterday' | 'last_week' = 'yesterday'): Promise<{
    current: Array<{ time: string; value: number }>;
    compare: Array<{ time: string; value: number }>;
    metadata: { kpi_name: string; compare_type: string; change_percent?: number };
  }> {
    await delay(100, 200);
    const baseData = mockDashboardChartData.kpiTimeSeries[kpiName] || [];
    const current = baseData.slice(-24).map(([time, value]) => ({ time, value }));
    const compare = baseData.slice(-48, -24).map(([time, value]) => ({ time, value }));
    return {
      current,
      compare,
      metadata: {
        kpi_name: kpiName,
        compare_type: compareWith,
        change_percent: Math.random() * 10 - 5,
      },
    };
  },
};
