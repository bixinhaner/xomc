/**
 * Dashboard 组件专用类型定义
 */

import type { ReactNode } from 'react';

/**
 * KPI 状态类型
 */
export type KPIStatus = 'normal' | 'warning' | 'critical';

/**
 * KPI 趋势方向
 */
export type TrendDirection = 'up' | 'down' | 'stable';

/**
 * 时间范围类型
 */
export type TimeRange = 'day' | 'week';

/**
 * KPI 趋势数据点
 */
export interface KPITrendDataPoint {
  time: string;    // ISO8601 格式时间
  value: number;   // KPI 值
}

/**
 * KPI 趋势响应 - 今日vs昨日对比
 */
export interface KPITrendComparison {
  current: KPITrendDataPoint[];      // 今日数据
  compare: KPITrendDataPoint[];      // 昨日数据
  metadata: {
    kpi_name: string;
    compare_type: 'yesterday' | 'last_week';
    change_percent?: number;         // 变化百分比
  };
}

/**
 * KPI 卡片属性
 */
export interface KPICardProps {
  title: string;           // KPI 名称
  value: number;          // 当前值
  unit?: string;           // 单位
  trend?: number;         // 趋势百分比
  status?: KPIStatus;     // 状态
  icon?: ReactNode;        // 图标
  loading?: boolean;       // 加载状态
  timestamp?: string;      // 数据时间戳
}

/**
 * KPI 趋势图配置
 */
export interface KPITrendConfig {
  kpiCode: string;        // KPI 代码（如 'prb_util_dl'）
  label: string;          // 显示标签
  unit: string;           // 单位
  statusConfig?: {
    normal?: { min?: number; max?: number };
    warning?: { min?: number; max?: number };
    critical?: { min?: number; max?: number };
  };
}

/**
 * KPI 趋势图属性
 */
export interface KPITrendChartProps {
  title: string;          // 图表标题
  kpiCode: string;        // KPI 代码
  kpiLabel: string;       // KPI 显示标签
  unit: string;           // 单位
  value?: number;         // 当前值（顶部显示）
  status?: KPIStatus;     // 状态
  timeRange?: TimeRange;  // 时间范围
  height?: number;        // 图表高度
  loading?: boolean;      // 加载状态
  className?: string;     // 自定义类名
  onTimeRangeChange?: (range: TimeRange) => void;  // 时间范围切换回调
  trendData?: KPITrendComparison;  // 趋势数据（今日vs昨日）
}

/**
 * 图表时间范围选项
 */
export interface TimeRangeOption {
  label: string;
  value: TimeRange;
  hours: number;         // 对应小时数
}
