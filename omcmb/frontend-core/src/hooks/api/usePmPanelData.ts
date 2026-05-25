/**
 * G6-Gap-7：Panel 数据加载 hook（含对比双模式语义）。
 *
 * v1 实现：返回 deterministic mock series（按 panelId + granularity 哈希），
 * 当 panel.compareMode 非空时返回 compareSeries（双 series 或多 series）。
 *
 * v2 计划：替换 mockSeries() 内部为调 /pm/aggregation/query 或 KPI 查询接口；
 * Hook 对外 API（PanelSeriesData / usePmPanelData 签名）保持不变。
 */

import { useMemo } from 'react';
import type { CompareMode, Granularity, Panel } from '../../types/pmDashboard';

export interface PanelSeriesPoint {
  // X 轴标签（时间桶或设备名）。
  label: string;
  // Y 轴值；null 表示缺采，前端按"缺采"渲染（不画 0）。
  value: number | null;
  // 入库延迟秒数；非空且超阈值时 PanelRenderer 画角标（G6-Gap-9）。
  ingestLagSeconds?: number;
}

export interface PanelSeries {
  name: string;
  // 对比模式下 kind=primary 是主 series；compare 是对比 series。
  kind: 'primary' | 'compare';
  points: PanelSeriesPoint[];
}

export interface PanelSeriesData {
  // 已展开为多 series（单 series + 对比 series；或多设备对比时多个 compare）。
  series: PanelSeries[];
  // 当前 panel 是否处于对比模式（非空 = 渲染时多 series 处理）。
  compareMode: CompareMode | undefined;
  isLoading: boolean;
}

// 基于 (panelId, granularity, metric) 生成 deterministic 数字 0..N-1
function hash(seed: string): number {
  let h = 2166136261;
  for (let i = 0; i < seed.length; i++) {
    h ^= seed.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return Math.abs(h >>> 0);
}

function granularityBuckets(g: Granularity): string[] {
  switch (g) {
    case '15min':
      return ['00:00', '00:15', '00:30', '00:45', '01:00', '01:15', '01:30', '01:45'];
    case 'hourly':
      return ['10:00', '11:00', '12:00', '13:00', '14:00', '15:00'];
    case 'daily':
      return ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
    case 'weekly':
      return ['W1', 'W2', 'W3', 'W4'];
    case 'monthly':
      return ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'];
  }
}

// 给一组 bucket 生成 deterministic 数列；可叠加偏移做对比 series。
function mockSeries(seed: string, buckets: string[], offset: number): PanelSeriesPoint[] {
  const base = hash(seed) % 20;
  return buckets.map((label, i) => {
    // 在固定模式下塞一个 null 模拟缺采（G6-Gap-9 (c)）
    const value =
      i === Math.floor(buckets.length / 2) && seed.endsWith(':primary:0')
        ? null
        : Math.round((90 + ((base + i * 3 + offset) % 11) + Math.sin(i + offset) * 1.5) * 100) / 100;
    return { label, value };
  });
}

/**
 * v1：纯 mock；后续 v2 替换为真实 API 调用。
 *
 * 行为：
 *   - 主 series：panel.metricPaths 每个 metric 生成一条 primary series。
 *   - compareMode='previous_window'：每个 metric 多一条 "前一周期" compare series（偏移 5）。
 *   - compareMode='same_window_other_devices'：每个 metric 多一条 "对比设备" compare series（偏移 -3）。
 *   - compareMode=undefined：只返主 series。
 */
export function usePmPanelData(panel: Panel, activeGranularity: Granularity): PanelSeriesData {
  return useMemo(() => {
    const buckets = granularityBuckets(activeGranularity);
    const series: PanelSeries[] = [];
    panel.metricPaths.forEach((metric, idx) => {
      series.push({
        name: metric,
        kind: 'primary',
        points: mockSeries(`${panel.id}:${activeGranularity}:${metric}:primary:${idx}`, buckets, 0),
      });
      if (panel.compareMode === 'previous_window') {
        series.push({
          name: `${metric} (上一周期)`,
          kind: 'compare',
          points: mockSeries(`${panel.id}:${activeGranularity}:${metric}:prev:${idx}`, buckets, 5),
        });
      } else if (panel.compareMode === 'same_window_other_devices') {
        series.push({
          name: `${metric} (对比设备)`,
          kind: 'compare',
          points: mockSeries(`${panel.id}:${activeGranularity}:${metric}:other:${idx}`, buckets, -3),
        });
      }
    });
    return { series, compareMode: panel.compareMode, isLoading: false };
  }, [panel, activeGranularity]);
}
