/**
 * 顶部只读时钟 hook（issue #459，子单 D / 母单 #455）。
 *
 * 每秒按**系统时区**刷新「当前时间」，供 v1 TimezoneSelector / v2 AppShell /
 * v3 HUDStatusBar 顶栏只读展示「系统时区 + 实时当前时间」。
 *
 * 系统时区来自 appStore.systemTimezone（由 useSystemTimezone 登录后写入），
 * 未拉到时回落 UTC。不依赖浏览器 OS 时区。
 */
import { useEffect, useMemo, useState } from 'react';
import {
  nowInSystemTimezone,
  systemTimezoneLabel,
  DEFAULT_TIME_FORMAT,
  TIME_FORMAT,
  DATE_FORMAT,
} from '../utils/systemTime';
import { useSystemTimezoneValue } from './api/useSystemTimezone';

export interface SystemClock {
  /** 系统时区标注文案（如 'Asia/Tokyo' / 'UTC'）。 */
  timezoneLabel: string;
  /** 当前时间完整钟面（YYYY-MM-DD HH:mm:ss）。 */
  dateTime: string;
  /** 当前时间（HH:mm:ss）。 */
  time: string;
  /** 当前日期（YYYY-MM-DD）。 */
  date: string;
}

/**
 * useSystemClock —— 每秒刷新的系统时区只读时钟。
 *
 * @param tickMs 刷新间隔，默认 1000ms（每秒）
 */
export function useSystemClock(tickMs = 1000): SystemClock {
  const systemTimezone = useSystemTimezoneValue();
  const [now, setNow] = useState<number>(() => Date.now());

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), tickMs);
    return () => clearInterval(id);
  }, [tickMs]);

  return useMemo<SystemClock>(() => {
    // 用 now 触发重算（依赖项），实际取系统时区当前钟面
    void now;
    const m = nowInSystemTimezone(systemTimezone);
    return {
      timezoneLabel: systemTimezoneLabel(systemTimezone),
      dateTime: m.format(DEFAULT_TIME_FORMAT),
      time: m.format(TIME_FORMAT),
      date: m.format(DATE_FORMAT),
    };
  }, [now, systemTimezone]);
}
