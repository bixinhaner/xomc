/**
 * useKPIPanelData - KPIPanel数据请求Hook
 *
 * 根据选中的指标、时间范围选择、视图模式获取KPI时序数据
 */

import { useMemo } from 'react';
import { useQueries } from '@tanstack/react-query';
import {
  DASHBOARD_FRESH_TIME_MS,
  DASHBOARD_REFRESH_INTERVAL_MS,
} from '@core/hooks/api/useDashboard';
import { dashboardApi } from '@core/services/api/dashboardApi';
import { dashboardService } from '@core/mock/services/dashboardService';
import { useMock } from '@core/services/apiSwitch';
import type { TimeRangeOption, ViewMode } from '@/pages/dashboard/kpi-config';

const api = useMock ? dashboardService : dashboardApi;

export interface KPIDataPoint {
  time: string;
  value: number;
}

export interface KPIPanelDataResult {
  today?: KPIDataPoint[];
  yesterday?: KPIDataPoint[];
}

/**
 * 计算Today时间范围参数
 */
function calculateTodayParams(viewMode: ViewMode) {
  const now = new Date();
  let startTime: Date;

  if (viewMode === 'day') {
    // Day模式：今天00:00到现在
    startTime = new Date(now);
    startTime.setHours(0, 0, 0, 0);
  } else {
    // Week模式：本周一到现在
    const weekday = now.getDay() || 7; // 周日 = 7
    startTime = new Date(now);
    startTime.setDate(now.getDate() - weekday + 1);
    startTime.setHours(0, 0, 0, 0);
  }

  return {
    start_time: startTime.toISOString(),
    end_time: now.toISOString(),
  };
}

/**
 * 计算Yesterday时间范围参数
 */
function calculateYesterdayParams(viewMode: ViewMode) {
  const now = new Date();
  let startTime: Date;
  let endTime: Date;

  if (viewMode === 'day') {
    // Day模式：昨天00:00到23:59:59
    startTime = new Date(now);
    startTime.setDate(startTime.getDate() - 1);
    startTime.setHours(0, 0, 0, 0);

    endTime = new Date(startTime);
    endTime.setHours(23, 59, 59, 999);
  } else {
    // Week模式：上周一到上周日
    const weekday = now.getDay() || 7;
    const thisMonday = new Date(now);
    thisMonday.setDate(now.getDate() - weekday + 1);
    thisMonday.setHours(0, 0, 0, 0);

    startTime = new Date(thisMonday);
    startTime.setDate(startTime.getDate() - 7);
    startTime.setHours(0, 0, 0, 0);

    endTime = new Date(startTime);
    endTime.setDate(endTime.getDate() + 6);
    endTime.setHours(23, 59, 59, 999);
  }

  return {
    start_time: startTime.toISOString(),
    end_time: endTime.toISOString(),
  };
}

/**
 * 数据归一化：兼容 Mock（元组）和真实 API（对象）两种格式
 */
function normalizeKPIDataPoint(point: unknown): KPIDataPoint {
  if (Array.isArray(point)) {
    return { time: point[0] as string, value: point[1] as number };
  }
  return point as KPIDataPoint;
}

/**
 * KPIPanel数据请求Hook
 *
 * @param kpiName - KPI指标名称
 * @param timeRangeSelection - 时间范围选择（today/yesterday多选）
 * @param viewMode - 视图模式（day/week）
 * @param enabled - 是否启用查询
 * @returns 包含today/yesterday数据和加载状态的结果对象
 *
 * @example
 * ```ts
 * const { data, isLoading } = useKPIPanelData(
 *   'K900010015',
 *   ['today', 'yesterday'],
 *   'day',
 *   true
 * );
 * ```
 */
export function useKPIPanelData(
  kpiName: string,
  timeRangeSelection: TimeRangeOption[],
  viewMode: ViewMode,
  enabled = true
): { data: KPIPanelDataResult; isLoading: boolean } {
  // 计算查询参数
  const queries = useMemo(() => {
    if (!enabled || !kpiName) {
      return [];
    }

    const result = [];

    // Today数据查询
    if (timeRangeSelection.includes('today')) {
      const params = calculateTodayParams(viewMode);
      result.push({
        key: 'today',
        queryKey: ['dashboard', 'kpi-time-series', kpiName, 'today', viewMode],
        params: { kpi_names: [kpiName], ...params },
      });
    }

    // Yesterday数据查询
    if (timeRangeSelection.includes('yesterday')) {
      const params = calculateYesterdayParams(viewMode);
      result.push({
        key: 'yesterday',
        queryKey: ['dashboard', 'kpi-time-series', kpiName, 'yesterday', viewMode],
        params: { kpi_names: [kpiName], ...params },
      });
    }

    return result;
  }, [kpiName, timeRangeSelection, viewMode, enabled]);

  // 使用useQueries并行请求
  const results = useQueries({
    queries: queries.map(q => ({
      queryKey: q.queryKey,
      queryFn: () => api.getKPITimeSeries(q.params.kpi_names, q.params.start_time, q.params.end_time),
      enabled: enabled,
      retry: false,
      refetchInterval: DASHBOARD_REFRESH_INTERVAL_MS,
      refetchIntervalInBackground: false,
      refetchOnWindowFocus: false,
      refetchOnReconnect: false,
      staleTime: DASHBOARD_FRESH_TIME_MS,
    })),
  });

  // 组合结果
  const data = useMemo(() => {
    const result: KPIPanelDataResult = {};

    let hasToday = false;
    let hasYesterday = false;

    results.forEach((r, index) => {
      const query = queries[index];
      if (!query) {
        return;
      }

      const rawData = r.data;

      if (rawData && rawData[kpiName] && rawData[kpiName].length > 0) {
        const normalized = rawData[kpiName].map(normalizeKPIDataPoint);
        if (query.key === 'today') {
          result.today = normalized;
          hasToday = true;
        } else if (query.key === 'yesterday') {
          result.yesterday = normalized;
          hasYesterday = true;
        }
      }
    });

    // 如果两个查询都没有数据，但查询成功，则返回空数组
    if (!hasToday && timeRangeSelection.includes('today') && results.length > 0) {
      result.today = [];
    }
    if (!hasYesterday && timeRangeSelection.includes('yesterday') && results.length > 1) {
      result.yesterday = [];
    }

    return result;
  }, [results, queries, kpiName, timeRangeSelection]);

  const isLoading = results.some(r => r.isLoading);

  return { data, isLoading };
}
