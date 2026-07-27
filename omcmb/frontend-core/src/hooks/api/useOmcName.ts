import { useEffect, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { adminApi } from '../../services/api/adminApi';
import type { SysConfigItem } from '../../types/system';
import { useAppStore } from '../../store/appStore';

/**
 * OMC 名称（系统品牌标题）配置 key —— 与「系统配置 > 基本设置」表单字段
 * `mrOMCName` 一一对应，存 sys_configs (category='basic', is_public=true)。
 *
 * 种子迁移保证该行始终存在且 is_public=true，故公开通道（免登录）可读。
 */
export const OMC_NAME_CONFIG_KEY = 'mrOMCName';

/**
 * usePublicOmcName 走 `/admin/public/configs?category=basic`（is_public=true，
 * 无需鉴权）拉取用户配置的「OMC 名称」，登录前 / 登录后均可用。
 *
 * 返回 `omcName`：
 *   - 配置非空 → 返回该字符串（已 trim）。
 *   - 配置为空 / 未配置 / 拉取失败 → 返回 undefined，由各消费方自行回退到默认名
 *     （v1 i18n app.title / v2 'OMC · v2' / v3 'STARFORGE'），避免空白标题。
 *
 * 失败安全：网络 / 401 / 解析失败由 React Query 记录为 error，data 保持 undefined
 * （retry:false，不阻塞登录页渲染）。
 *
 * 副作用：拉到值后同步写入 appStore.omcName（Zustand persist），让下次冷启动 /
 * 登录后能在请求 resolve 前先用 localStorage 缓存名即时渲染，避免标题闪烁。
 */
export function usePublicOmcName(enabled = true): { omcName: string | undefined } {
  const setOmcName = useAppStore((s) => s.setOmcName);

  const { data } = useQuery<SysConfigItem[]>({
    queryKey: ['public', 'sysConfig', 'basic'],
    queryFn: () => adminApi.getPublicSysConfigsByCategory('basic'),
    enabled,
    staleTime: 5 * 60_000,
    retry: false,
  });

  const omcName = useMemo<string | undefined>(() => {
    if (!data) return undefined;
    const item = data.find((c) => c.key === OMC_NAME_CONFIG_KEY);
    const v = item?.value?.trim();
    return v && v !== '' ? v : undefined;
  }, [data]);

  useEffect(() => {
    // 只在拉到明确值时同步；undefined（未配置/失败）不覆盖已 persist 的旧值，
    // 也不写空——清空配置时由消费方回退默认名，store 留 undefined 即可。
    setOmcName(omcName);
  }, [omcName, setOmcName]);

  return { omcName };
}

/**
 * resolveOmcName —— 统一回退解析：配置值优先，空则回退到传入默认名。
 * 消费方写法：`resolveOmcName(omcName, t('app.title'))`。
 */
export function resolveOmcName(configured: string | undefined, fallback: string): string {
  const v = configured?.trim();
  return v && v !== '' ? v : fallback;
}
