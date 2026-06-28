import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { adminApi } from '../../services/api/adminApi';
import type { SysConfigItem } from '../../types/system';
import { useSysConfigsByCategory } from './useSystem';

/**
 * 解码后的安全设置（FE 友好类型）— 与后端 SecurityPolicy
 * (internal/admin/security_policy.go) 一致。
 *
 * FE 消费方：
 *   - LoginPage：preventBrowserAutofill 切 `autocomplete` 属性
 *   - 主 Layout：idleLockMinutes 启动空闲计时器
 *   - LoginPage 成功后：loginNotifyMsg 弹 Modal
 */
export interface SecuritySettings {
  // ⑦ 屏幕锁定
  idleLockMinutes: number; // 0 = 禁用
  // ⑧ 浏览器记密
  preventBrowserAutofill: boolean;
  // ⑪ 登录提示（仅 FE 知道是否启用，文案由 Login 响应附带）
  loginNotifyEnabled: boolean;
  loginNotifyMsg: string;
  // 密码长度策略（动态从 sys_configs 获取）
  pwdMinLength: number;
  pwdMaxLength: number;
  // 其余字段以 raw 暴露，便于设置页消费
  raw: Map<string, string>;
}

/**
 * useSecuritySettings 拉 sys_configs?category=security 并解码常用字段。
 * 已用 useSysConfigsByCategory 的 30s staleTime 做缓存；多个组件调用复用同一份。
 *
 * enabled=false 时不发请求 — 给登录页（用户未登录态可能拉不到）一个旁路。
 *
 * Issue #649：暴露 refetch，让重置密码弹窗 onClick 时强制拿最新 defaultPasswd
 * （绕开缓存避免管理员刚清空后弹错弹窗的 UX bug）。
 */
export function useSecuritySettings(enabled = true): {
  settings: SecuritySettings | undefined;
  isFetching: boolean;
  refetch: () => Promise<unknown>;
} {
  const { data, isFetching, refetch } = useSysConfigsByCategory('security', enabled);

  const settings = useMemo<SecuritySettings | undefined>(() => {
    if (!data) return undefined;
    const raw = new Map<string, string>();
    for (const item of data) {
      raw.set(item.key, item.value);
    }
    return {
      idleLockMinutes: parseIntSafe(raw.get('userSessionExpirationMin'), 0),
      preventBrowserAutofill: parseBoolSafe(raw.get('isBrowserAutoRecordPass'), false),
      loginNotifyEnabled: parseBoolSafe(raw.get('enabledFlag'), false),
      loginNotifyMsg: raw.get('msg') ?? '',
      pwdMinLength: parseIntSafe(raw.get('pwdMinLength'), 8),
      pwdMaxLength: parseIntSafe(raw.get('pwdMaxLength'), 32),
      raw,
    };
  }, [data]);

  return { settings, isFetching, refetch };
}

function parseIntSafe(raw: string | undefined, fallback: number): number {
  if (!raw) return fallback;
  const n = parseInt(raw, 10);
  return Number.isFinite(n) ? n : fallback;
}

function parseBoolSafe(raw: string | undefined, fallback: boolean): boolean {
  if (raw === undefined || raw === '') return fallback;
  return raw === 'true' || raw === '1';
}

/**
 * usePublicSecuritySettings 走 `/admin/public/configs?category=security` 无需鉴权，
 * 仅能拉 is_public=true 的 4 个字段：
 *   userSessionExpirationMin / isBrowserAutoRecordPass / enabledFlag / msg
 *
 * 主要消费方：登录页（未登录态拉不到 useSysConfigsByCategory）。
 *
 * 失败安全：401/500 一律返 undefined，让 LoginPage 走 default 行为。
 */
export function usePublicSecuritySettings(enabled = true): {
  settings: SecuritySettings | undefined;
} {
  const { data } = useQuery<SysConfigItem[] | undefined>({
    queryKey: ['public', 'sysConfig', 'security'],
    queryFn: async () => {
      try {
        return await adminApi.getPublicSysConfigsByCategory('security');
      } catch {
        return undefined; // 失败安全 — 不阻塞登录
      }
    },
    enabled,
    staleTime: 60_000,
    retry: false,
  });

  const settings = useMemo<SecuritySettings | undefined>(() => {
    if (!data) return undefined;
    const raw = new Map<string, string>();
    for (const item of data) {
      raw.set(item.key, item.value);
    }
    return {
      idleLockMinutes: parseIntSafe(raw.get('userSessionExpirationMin'), 0),
      preventBrowserAutofill: parseBoolSafe(raw.get('isBrowserAutoRecordPass'), false),
      loginNotifyEnabled: parseBoolSafe(raw.get('enabledFlag'), false),
      loginNotifyMsg: raw.get('msg') ?? '',
      // 公开 API 不返回密码策略，用默认值
      pwdMinLength: parseIntSafe(raw.get('pwdMinLength'), 8),
      pwdMaxLength: parseIntSafe(raw.get('pwdMaxLength'), 32),
      raw,
    };
  }, [data]);

  return { settings };
}
