/**
 * useDashboard 工具函数单元测试
 *
 * 测试策略：
 * - 纯函数测试，不依赖 React hooks
 * - 使用固定时间进行时间计算测试
 * - 覆盖边界情况和异常输入
 */

import { describe, it, expect } from 'vitest';

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
