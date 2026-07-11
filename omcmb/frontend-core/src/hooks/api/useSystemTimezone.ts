/**
 * 系统时区读取 hook（issue #459，子单 D / 母单 #455）。
 *
 * 系统时区唯一源 = sys_configs（category='basic', key='timezoneCode'，#456 子单 A）。
 * 该配置项**非公开**（is_public=false），故只能在**登录后**经鉴权通道
 * `/admin/sysConfig?category=basic` 读取（与 mrOMCName 走公开通道不同）。
 *
 * 拉到后写入 appStore.systemTimezone（persist），供全局：
 *   - 顶部只读时钟（useSystemClock）；
 *   - 时间范围筛选输入按系统时区附加偏移（toSystemTimezoneRFC3339）；
 *   - epoch/Date 类时间显示落系统钟面（formatSystemTime）。
 *
 * 失败安全：未登录 / 401 / 解析失败 → 不覆盖已 persist 的旧值，消费方回落 UTC。
 */
import { useEffect, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { adminApi } from '../../services/api/adminApi';
import type { SysConfigItem } from '../../types/system';
import { useAppStore } from '../../store/appStore';
import { useUserStore } from '../../store/userStore';

/** 系统时区配置 key（与后端 systimezone.Provider 的 Key 常量一致）。 */
export const TIMEZONE_CONFIG_KEY = 'timezoneCode';
/** 系统时区配置所在分类。 */
export const TIMEZONE_CONFIG_CATEGORY = 'basic';
/** 解析失败 / 未配置时的回落时区。 */
export const DEFAULT_SYSTEM_TIMEZONE = 'UTC';

/**
 * useSystemTimezone —— 登录后拉取系统时区并同步进 appStore。
 *
 * @returns systemTimezone：拉到的 IANA 名（如 'Asia/Tokyo'）；未拉到时返回 store 中
 *          的 persist 旧值或 undefined（消费方回落 UTC）。
 */
export function useSystemTimezone(): { systemTimezone: string | undefined } {
  const isAuthenticated = useUserStore((s) => s.isAuthenticated);
  const setSystemTimezone = useAppStore((s) => s.setSystemTimezone);
  const stored = useAppStore((s) => s.systemTimezone);

  const { data } = useQuery<SysConfigItem[] | undefined>({
    queryKey: ['sysConfig', 'basic', 'timezone'],
    queryFn: async () => {
      try {
        return await adminApi.getSysConfigsByCategory(TIMEZONE_CONFIG_CATEGORY);
      } catch {
        return undefined; // 失败安全 — 让消费方回落 UTC
      }
    },
    enabled: isAuthenticated,
    staleTime: 5 * 60_000,
    retry: false,
  });

  const fetched = useMemo<string | undefined>(() => {
    if (!data) return undefined;
    const item = data.find((c) => c.key === TIMEZONE_CONFIG_KEY);
    const v = item?.value?.trim();
    return v && v !== '' ? v : undefined;
  }, [data]);

  useEffect(() => {
    // 只在拉到明确值时同步；undefined（未配置/失败/未登录）不覆盖已 persist 的旧值。
    if (fetched) setSystemTimezone(fetched);
  }, [fetched, setSystemTimezone]);

  return { systemTimezone: fetched ?? stored };
}

/**
 * useSystemTimezoneValue —— 仅读取当前系统时区（不触发拉取），供格式化 / 筛选消费。
 * 返回 store 中的值，未拉到时为 undefined（消费方回落 UTC）。
 */
export function useSystemTimezoneValue(): string | undefined {
  return useAppStore((s) => s.systemTimezone);
}
