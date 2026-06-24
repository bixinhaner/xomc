/**
 * #595 粒度切换时自动联动时间范围。
 *
 * 映射关系：
 *   15min  → 近 3 小时
 *   hourly → 近 24 小时
 *   daily  → 近 7 天
 *   weekly → 近 30 天（≈1 月）
 *   monthly → 近 6 月
 */

import type { Granularity } from '../types/pmDashboard';
import type { TimeRangePreset } from '../types/pmQuery';

/** 粒度 → 推荐的时间范围预设（v1/v2 使用） */
export function getDefaultTimeRangeForGranularity(granularity: Granularity): TimeRangePreset {
  switch (granularity) {
    case '15min':
      return 'last_3h';
    case 'hourly':
      return 'last_24h';
    case 'daily':
      return 'last_7d';
    case 'weekly':
      return 'last_30d';
    case 'monthly':
      return 'last_6m';
    default:
      return 'last_24h';
  }
}

/** 粒度 → 推荐的时间范围小时数（v3 使用） */
export function getDefaultRangeHoursForGranularity(granularity: Granularity): number {
  switch (granularity) {
    case '15min':
      return 3;
    case 'hourly':
      return 24;
    case 'daily':
      return 24 * 7;
    default:
      return 24;
  }
}
