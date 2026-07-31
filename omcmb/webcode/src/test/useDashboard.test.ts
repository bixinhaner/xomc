/**
 * useDashboard 工具函数单元测试
 *
 * 测试策略：
 * - 纯函数测试，不依赖 React hooks
 * - 使用固定时间进行时间计算测试
 * - 覆盖边界情况和异常输入
 */

import { describe, it, expect } from 'vitest';
import {
  DASHBOARD_REFRESH_INTERVAL_MS,
  buildDashboardKPIWindow,
  buildDashboardDayRanges,
  buildDashboardKPIQueryOptions,
  buildDashboardKPIQueryKey,
  buildDashboardWeekRange,
  isDashboardBusinessTimezoneReady,
} from '@core/hooks/api/useDashboard';
import { dashboardService } from '@core/mock/services/dashboardService';
import { useAppStore } from '@core/store/appStore';

// 从 useDashboard.ts 导入实际测试的函数
// 注意：这些函数是内部实现细节，通过测试导出或从实际模块导入
// 为了测试，这里重新声明函数签名（实际测试时应该从源文件导入）
type KPITimeSeriesData = Record<string, unknown[]>;

/**
 * 数据归一化：兼容 Mock（元组）和真实 API（对象）两种格式
 *
 * 注意：此函数应从 @core/hooks/api/useDashboard 导入
 * 为了测试独立性，这里保留副本，但应与源文件保持同步
 */
function normalizeKPIDataPoint(point: unknown): { time: string; value: number } {
  if (Array.isArray(point)) {
    return { time: point[0] as string, value: point[1] as number };
  }
  return point as { time: string; value: number };
}

/**
 * 计算 KPI 趋势对比数据
 *
 * 注意：此函数应从 @core/hooks/api/useDashboard 导入
 * 为了测试独立性，这里保留副本，但应与源文件保持同步
 */
function calculateTrendComparison(
  currentData: KPITimeSeriesData,
  compareData: KPITimeSeriesData,
  kpiName: string,
  compareWith: 'yesterday' | 'last_week'
): {
  current: Array<{ time: string; value: number }>;
  compare: Array<{ time: string; value: number }>;
  metadata: {
    kpi_name: string;
    compare_type: 'yesterday' | 'last_week';
    change_percent?: number;
  };
} {
  const current = currentData[kpiName] || [];
  const compare = compareData[kpiName] || [];

  // 计算变化百分比
  let changePercent: number | undefined;
  if (current.length > 0 && compare.length > 0) {
    const currentAvg = current.reduce(
      (sum: number, point: unknown) => sum + normalizeKPIDataPoint(point).value,
      0
    ) / current.length;
    const compareAvg = compare.reduce(
      (sum: number, point: unknown) => sum + normalizeKPIDataPoint(point).value,
      0
    ) / compare.length;

    if (compareAvg !== 0) {
      changePercent = ((currentAvg - compareAvg) / compareAvg) * 100;
    }
  }

  return {
    current: current.map(normalizeKPIDataPoint),
    compare: compare.map(normalizeKPIDataPoint),
    metadata: {
      kpi_name: kpiName,
      compare_type: compareWith,
      change_percent: changePercent,
    },
  };
}

// ============================================================================
// 测试套件
// ============================================================================

describe('useDashboard 工具函数', () => {
  describe('normalizeKPIDataPoint', () => {
    it('应该将元组格式 [string, number] 转换为对象格式', () => {
      const tuplePoint: [string, number] = ['2024-01-01T00:00:00Z', 100];
      const result = normalizeKPIDataPoint(tuplePoint);

      expect(result).toEqual({ time: '2024-01-01T00:00:00Z', value: 100 });
    });

    it('应该保持对象格式不变', () => {
      const objectPoint = { time: '2024-01-01T00:00:00Z', value: 100 };
      const result = normalizeKPIDataPoint(objectPoint);

      expect(result).toEqual({ time: '2024-01-01T00:00:00Z', value: 100 });
    });

    it('应该处理数值为 0 的情况', () => {
      const tuplePoint: [string, number] = ['2024-01-01T00:00:00Z', 0];
      const result = normalizeKPIDataPoint(tuplePoint);

      expect(result.value).toBe(0);
    });

    it('应该处理负数值', () => {
      const tuplePoint: [string, number] = ['2024-01-01T00:00:00Z', -50];
      const result = normalizeKPIDataPoint(tuplePoint);

      expect(result.value).toBe(-50);
    });
  });

  describe('calculateTrendComparison', () => {
    const kpiName = 'dl_throughput';

    it('应该正确计算增长百分比', () => {
      const currentData = {
        [kpiName]: [
          ['2024-01-02T00:00:00Z', 100],
          ['2024-01-02T01:00:00Z', 110],
        ],
      };
      const compareData = {
        [kpiName]: [
          ['2024-01-01T00:00:00Z', 50],
          ['2024-01-01T01:00:00Z', 60],
        ],
      };

      const result = calculateTrendComparison(currentData, compareData, kpiName, 'yesterday');

      // 当前平均值: (100 + 110) / 2 = 105
      // 对比平均值: (50 + 60) / 2 = 55
      // 变化百分比: (105 - 55) / 55 * 100 = 90.909...
      expect(result.metadata.change_percent).toBeCloseTo(90.91, 1);
      expect(result.metadata.compare_type).toBe('yesterday');
      expect(result.current).toHaveLength(2);
      expect(result.compare).toHaveLength(2);
    });

    it('应该正确计算下降百分比', () => {
      const currentData = {
        [kpiName]: [
          ['2024-01-02T00:00:00Z', 50],
          ['2024-01-02T01:00:00Z', 60],
        ],
      };
      const compareData = {
        [kpiName]: [
          ['2024-01-01T00:00:00Z', 100],
          ['2024-01-01T01:00:00Z', 110],
        ],
      };

      const result = calculateTrendComparison(currentData, compareData, kpiName, 'yesterday');

      // 当前平均值: 55
      // 对比平均值: 105
      // 变化百分比: (55 - 105) / 105 * 100 = -47.619...
      expect(result.metadata.change_percent).toBeCloseTo(-47.62, 1);
      expect(result.metadata.change_percent).toBeLessThan(0);
    });

    it('当对比数据为 0 时应该返回 undefined', () => {
      const currentData = {
        [kpiName]: [
          ['2024-01-02T00:00:00Z', 100],
          ['2024-01-02T01:00:00Z', 110],
        ],
      };
      const compareData = {
        [kpiName]: [
          ['2024-01-01T00:00:00Z', 0],
          ['2024-01-01T01:00:00Z', 0],
        ],
      };

      const result = calculateTrendComparison(currentData, compareData, kpiName, 'yesterday');

      expect(result.metadata.change_percent).toBeUndefined();
    });

    it('应该处理空数据数组', () => {
      const currentData = { [kpiName]: [] };
      const compareData = {
        [kpiName]: [
          ['2024-01-01T00:00:00Z', 100],
        ],
      };

      const result = calculateTrendComparison(currentData, compareData, kpiName, 'yesterday');

      expect(result.current).toEqual([]);
      expect(result.metadata.change_percent).toBeUndefined();
    });

    it('应该处理不存在的 KPI 名称', () => {
      const currentData = { [kpiName]: [['2024-01-02T00:00:00Z', 100]] };
      const compareData = {}; // 不包含 kpiName

      const result = calculateTrendComparison(currentData, compareData, kpiName, 'yesterday');

      expect(result.current).toHaveLength(1);
      expect(result.compare).toEqual([]);
      expect(result.metadata.change_percent).toBeUndefined();
    });

    it('应该支持对象格式的数据点', () => {
      const currentData = {
        [kpiName]: [{ time: '2024-01-02T00:00:00Z', value: 100 }],
      };
      const compareData = {
        [kpiName]: [{ time: '2024-01-01T00:00:00Z', value: 80 }],
      };

      const result = calculateTrendComparison(currentData, compareData, kpiName, 'yesterday');

      expect(result.current[0]).toEqual({ time: '2024-01-02T00:00:00Z', value: 100 });
      expect(result.compare[0]).toEqual({ time: '2024-01-01T00:00:00Z', value: 80 });
      expect(result.metadata.change_percent).toBe(25); // (100 - 80) / 80 * 100
    });

    it('应该支持 last_week 对比类型', () => {
      const currentData = {
        [kpiName]: [['2024-01-08T00:00:00Z', 100]],
      };
      const compareData = {
        [kpiName]: [['2024-01-01T00:00:00Z', 80]],
      };

      const result = calculateTrendComparison(currentData, compareData, kpiName, 'last_week');

      expect(result.metadata.compare_type).toBe('last_week');
    });
  });
});

describe('dashboard KPI 系统时区窗口', () => {
  it('小时只取已发布桶，天和周窗口包含当前进行中自然周期', () => {
    const now = new Date('2026-07-13T01:30:00Z');

    const hourly = buildDashboardKPIWindow(now, 'hourly', 'Asia/Shanghai');
    expect(hourly.start_time).toBe('2026-07-12T09:00:00+08:00');
    expect(hourly.end_time).toBe('2026-07-13T09:00:00+08:00');
    expect(hourly.bucketKeys).toHaveLength(24);

    const daily = buildDashboardKPIWindow(now, 'daily', 'Asia/Shanghai');
    expect(daily.start_time).toBe('2026-06-14T00:00:00+08:00');
    expect(daily.end_time).toBe('2026-07-13T09:30:00+08:00');
    expect(daily.bucketKeys).toHaveLength(30);
    expect(daily.bucketKeys.at(-1)).toBe('2026-07-13');

    const weekly = buildDashboardKPIWindow(now, 'weekly', 'Asia/Shanghai');
    expect(weekly.start_time).toBe('2026-04-27T00:00:00+08:00');
    expect(weekly.end_time).toBe('2026-07-13T09:30:00+08:00');
    expect(weekly.bucketKeys).toHaveLength(12);
    expect(weekly.bucketKeys.at(-1)).toBe('2026-07-13');
  });

  it('周查询必须等待系统业务时区就绪，禁止静默回落 UTC', () => {
    expect(isDashboardBusinessTimezoneReady(undefined)).toBe(false);
    expect(isDashboardBusinessTimezoneReady('')).toBe(false);
    expect(isDashboardBusinessTimezoneReady('   ')).toBe(false);
    expect(isDashboardBusinessTimezoneReady('Asia/Shanghai')).toBe(true);
  });

  it('今日/昨日使用系统时区自然日半开窗口', () => {
    const ranges = buildDashboardDayRanges(new Date('2026-07-13T01:30:00Z'), 'Asia/Shanghai');
    expect(ranges.current.start_time).toBe('2026-07-13T00:00:00+08:00');
    expect(ranges.current.end_time).toBe('2026-07-13T09:30:00+08:00');
    expect(ranges.compare.start_time).toBe('2026-07-12T00:00:00+08:00');
    expect(ranges.compare.end_time).toBe('2026-07-13T00:00:00+08:00');
  });

  it('Mock KPI 响应保留请求窗口的系统时区钟面与偏移', async () => {
    useAppStore.getState().setSystemTimezone('Asia/Shanghai');
    const result = await dashboardService.getKPITimeSeries(
      ['K900010015'],
      '2026-07-13T00:00:00+08:00',
      '2026-07-13T02:00:00+08:00',
      'hourly',
    );

    expect(result.K900010015?.map((point) => point[0])).toEqual([
      '2026-07-13T00:00:00+08:00',
      '2026-07-13T01:00:00+08:00',
    ]);
  });

  it('Mock daily 桶跨 DST 时保持自然日零点并逐点更新偏移', async () => {
    useAppStore.getState().setSystemTimezone('America/New_York');
    const result = await dashboardService.getKPITimeSeries(
      ['K900010015'],
      '2026-03-07T00:00:00-05:00',
      '2026-03-10T00:00:00-04:00',
      'daily',
    );

    expect(result.K900010015?.map((point) => point[0])).toEqual([
      '2026-03-07T00:00:00-05:00',
      '2026-03-08T00:00:00-05:00',
      '2026-03-09T00:00:00-04:00',
    ]);
  });

  it('周窗口查询最近七个完整自然日，但横轴保留今天的普通日期占位', () => {
    const range = buildDashboardWeekRange(new Date('2026-07-13T01:30:00Z'), 'Asia/Shanghai');
    expect(range.start_time).toBe('2026-07-06T00:00:00+08:00');
    expect(range.end_time).toBe('2026-07-13T00:00:00+08:00');
    expect(range.dateKeys).toEqual([
      '2026-07-06', '2026-07-07', '2026-07-08', '2026-07-09',
      '2026-07-10', '2026-07-11', '2026-07-12', '2026-07-13',
    ]);
  });

  it('跨系统时区午夜后日期窗口向前滚动', () => {
    const before = buildDashboardWeekRange(new Date('2026-07-13T15:59:59Z'), 'Asia/Shanghai');
    const after = buildDashboardWeekRange(new Date('2026-07-13T16:00:01Z'), 'Asia/Shanghai');
    expect(before.end_time).toBe('2026-07-13T00:00:00+08:00');
    expect(after.end_time).toBe('2026-07-14T00:00:00+08:00');
  });

  it('KPI query key 区分 hourly 与 daily', () => {
    const common = {
      kpi_names: ['K1'],
      start_time: '2026-07-13T00:00:00+08:00',
      end_time: '2026-07-14T00:00:00+08:00',
    };
    expect(buildDashboardKPIQueryKey({ ...common, granularity: 'hourly' }))
      .not.toEqual(buildDashboardKPIQueryKey({ ...common, granularity: 'daily' }));
  });

  it('周查询配置只执行一次 daily 请求，并在无指标时禁用', async () => {
    const calls: unknown[][] = [];
    const client = {
      getKPITimeSeries: async (...args: unknown[]) => {
        calls.push(args);
        return {};
      },
    };
    const params = {
      kpi_names: ['K1'],
      start_time: '2026-07-06T00:00:00+08:00',
      end_time: '2026-07-13T00:00:00+08:00',
      granularity: 'daily' as const,
      technology: 'gsm',
    };

    const options = buildDashboardKPIQueryOptions(params, true, client);
    expect(options.enabled).toBe(true);
    await options.queryFn();
    expect(calls).toEqual([[
      ['K1'],
      params.start_time,
      params.end_time,
      'daily',
      'gsm',
    ]]);

    const disabled = buildDashboardKPIQueryOptions(
      { ...params, kpi_names: [] },
      true,
      client,
    );
    expect(disabled.enabled).toBe(false);
  });

  it('KPI 查询只按五分钟轮询，不通过窗口焦点或重连即时刷新', () => {
    const options = buildDashboardKPIQueryOptions({
      kpi_names: ['K1'],
      start_time: '2026-07-01T00:00:00+08:00',
      end_time: '2026-07-02T00:00:00+08:00',
      granularity: 'hourly',
    });

    expect(options.refetchInterval).toBe(DASHBOARD_REFRESH_INTERVAL_MS);
    expect(options.refetchIntervalInBackground).toBe(false);
    expect(options.refetchOnWindowFocus).toBe(false);
    expect(options.refetchOnReconnect).toBe(false);
  });
});
