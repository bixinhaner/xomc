/**
 * React Query 缓存监控 Hook
 *
 * 用于监控 React Query 缓存性能，包括：
 * - 缓存命中率
 * - 过期状态
 * - 活跃/非活跃查询
 *
 * 使用场景：开发环境调试、性能监控面板
 *
 * @module hooks/useReactQueryMonitor
 */

import { useQueryClient } from '@tanstack/react-query';
import { useState, useEffect } from 'react';

/**
 * 缓存统计信息
 */
export interface CacheStats {
  /** 总查询数量 */
  total: number;
  /** 过期的查询数量 */
  stale: number;
  /** 非活跃查询数量（无 observers） */
  inactive: number;
  /** 正在获取的查询数量 */
  fetching: number;
  /** 缓存命中率 (%) */
  hitRate: number;
  /** 更新时间戳 */
  timestamp: number;
}

/**
 * React Query 缓存监控 Hook
 *
 * @example
 * ```tsx
 * const monitor = useReactQueryMonitor();
 * console.log(monitor.stats);
 * // { total: 42, stale: 5, inactive: 20, fetching: 1, hitRate: 88.1 }
 * ```
 */
export function useReactQueryMonitor(options?: {
  /** 更新间隔（毫秒），默认 5000ms */
  interval?: number;
  /** 是否启用，默认 true */
  enabled?: boolean;
}) {
  const { interval = 5000, enabled = true } = options ?? {};

  const queryClient = useQueryClient();
  // 使用函数式初始化避免在渲染期间调用不纯函数
  const [stats, setStats] = useState<CacheStats>(() => ({
    total: 0,
    stale: 0,
    inactive: 0,
    fetching: 0,
    hitRate: 0,
    timestamp: Date.now(),
  }));

  // 定时更新统计信息
  useEffect(() => {
    if (!enabled) return;

    const updateStats = () => {
      const cache = queryClient.getQueryCache();
      const queries = cache.getAll();

      const total = queries.length;
      const stale = queries.filter(q => q.isStale()).length;
      const inactive = queries.filter(q => q.getObserversCount() === 0).length;
      const fetching = queries.filter(q => q.state.fetchStatus === 'fetching').length;

      // 计算缓存命中率
      // 这里简化计算：非过期的查询占比作为命中率参考
      const hitRate = total > 0 ? ((total - stale) / total) * 100 : 0;

      setStats({
        total,
        stale,
        inactive,
        fetching,
        hitRate,
        timestamp: Date.now(),
      });
    };

    updateStats(); // 立即执行一次
    const timer = setInterval(updateStats, interval);

    return () => clearInterval(timer);
  }, [queryClient, interval, enabled]);

  /**
   * 获取特定查询的详细信息
   */
  const getQueryInfo = (queryKey: unknown[]) => {
    const cache = queryClient.getQueryCache();
    const query = cache.find({ queryKey });

    if (!query) {
      return null;
    }

    return {
      queryKey: query.queryKey,
      state: {
        data: query.state.data,
        isLoading: query.state.status === 'pending',
        isError: query.state.status === 'error',
        isStale: query.isStale(),
        observersCount: query.getObserversCount(),
        lastUpdated: query.state.dataUpdatedAt,
      },
    };
  };

  /**
   * 清除所有缓存
   */
  const clearAll = () => {
    queryClient.clear();
  };

  /**
   * 清除过期缓存
   */
  const clearStale = () => {
    queryClient.getQueryCache().getAll().forEach(query => {
      if (query.isStale()) {
        queryClient.removeQueries({ queryKey: query.queryKey });
      }
    });
  };

  return {
    /** 当前缓存统计 */
    stats,
    /** 获取特定查询的详细信息 */
    getQueryInfo,
    /** 清除所有缓存 */
    clearAll,
    /** 清除过期缓存 */
    clearStale,
    /** 底层 queryClient 实例 */
    queryClient,
  };
}
