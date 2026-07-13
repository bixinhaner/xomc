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
import dayjs from 'dayjs';
import { toSystemTimezoneRFC3339 } from './systemTime';

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

export interface AlignedPresetRangeOptions {
  granularity: Granularity;
  preset: TimeRangePreset;
  systemTimezone?: string | null;
  now?: Date;
}

export interface AlignedPresetRange {
  start: string;
  end: string;
}

function presetStart(end: dayjs.Dayjs, preset: TimeRangePreset): dayjs.Dayjs | null {
  switch (preset) {
    case 'last_1h':
      return end.subtract(1, 'hour');
    case 'last_3h':
      return end.subtract(3, 'hour');
    case 'last_24h':
      return end.subtract(24, 'hour');
    case 'last_7d':
      return end.subtract(7, 'day');
    case 'last_30d':
      return end.subtract(30, 'day');
    case 'last_6m':
      return end.subtract(6, 'month');
    case 'custom':
      return null;
  }
}

function alignWeeklyStart(d: dayjs.Dayjs): dayjs.Dayjs {
  const weekday = d.day();
  const daysSinceMonday = (weekday + 6) % 7;
  return d.subtract(daysSinceMonday, 'day').startOf('day');
}

function alignBucketStart(d: dayjs.Dayjs, granularity: Granularity): dayjs.Dayjs {
  switch (granularity) {
    case '15min': {
      const minute = Math.floor(d.minute() / 15) * 15;
      return d.minute(minute).second(0).millisecond(0);
    }
    case 'hourly':
      return d.startOf('hour');
    case 'daily':
      return d.startOf('day');
    case 'weekly':
      return alignWeeklyStart(d);
    case 'monthly':
      return d.startOf('month');
    default:
      return d;
  }
}

export function buildAlignedPresetRange({
  granularity,
  preset,
  systemTimezone,
  now = new Date(),
}: AlignedPresetRangeOptions): AlignedPresetRange | null {
  if (preset === 'custom') return null;

  const tz = systemTimezone?.trim() || 'UTC';
  const base = dayjs(now).tz(tz);
  if (!base.isValid()) return null;

  const end = alignBucketStart(base, granularity);
  const rawStart = presetStart(end, preset);
  if (!rawStart) return null;
  const start = alignBucketStart(rawStart, granularity);

  const startIso = toSystemTimezoneRFC3339(start, tz);
  const endIso = toSystemTimezoneRFC3339(end, tz);
  if (!startIso || !endIso) return null;

  return { start: startIso, end: endIso };
}
