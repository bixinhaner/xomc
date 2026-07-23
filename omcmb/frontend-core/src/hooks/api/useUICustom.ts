import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { adminApi } from '../../services/api/adminApi';
import type { SysConfigItem } from '../../types/system';

/**
 * UI 定制化运行时值：登录背景图、Logo。
 *
 * 与后端 sys_configs (category='ui_custom') 的 key 一一对应；
 * seed 默认值（migrations/seed/000065_seed_ui_custom.sql）保证字段始终非空。
 *
 * 产品名称（OMC名称）由「系统配置 > 基本设置」统一维护，不在此处。
 * 主题色不在此处管理 — 全局只用 Header 右上角两主题切换（tech ↔ fresh）。
 */
export interface UICustomSettings {
  loginBackground: string;
  menuLogoUp: string;
  menuLogoDown: string;
}

// 默认值（与 omcmb/webcode/src/pages/system/SystemConfig/uiCustomConstants.ts
// 的 UI_CUSTOM_DEFAULTS 保持一致；后端 seed 失败 / 网络异常时兜底使用）。
export const UI_CUSTOM_DEFAULTS: UICustomSettings = {
  loginBackground: './images/login/login_bg.png',
  menuLogoUp: './images/login/nav_logo_collapse.png',
  menuLogoDown: './images/login/logo_big.png',
};

/**
 * usePublicUICustom 走 `/admin/public/configs?category=ui_custom`（is_public=true，
 * 无需鉴权）拉取登录页 / 侧栏 / 主题用得到的品牌化资产，登录前 / 登录后都可用。
 *
 * 失败安全：网络 / 401 / 解析失败由 React Query 记录为 error，data 保持 undefined，
 * 返回 UI_CUSTOM_DEFAULTS，让消费方走默认样式而非空白。
 *
 * 缓存：5 分钟 staleTime — 管理员保存"UI 定制化"页后通过
 * `queryClient.invalidateQueries(['public','sysConfig','ui_custom'])` 主动刷新
 * （详见 useBatchUpdateSysConfigs onSuccess）。
 */
export function usePublicUICustom(enabled = true): { settings: UICustomSettings } {
  const { data } = useQuery<SysConfigItem[]>({
    queryKey: ['public', 'sysConfig', 'ui_custom'],
    queryFn: () => adminApi.getPublicSysConfigsByCategory('ui_custom'),
    enabled,
    staleTime: 5 * 60_000,
    retry: false,
  });

  const settings = useMemo<UICustomSettings>(() => {
    if (!data) return UI_CUSTOM_DEFAULTS;
    const raw = new Map<string, string>();
    for (const item of data) raw.set(item.key, item.value);
    const nonEmpty = (k: string, fallback: string): string => {
      const v = raw.get(k);
      return v && v.trim() !== '' ? v : fallback;
    };
    return {
      loginBackground: nonEmpty('ui_login_background', UI_CUSTOM_DEFAULTS.loginBackground),
      menuLogoUp: nonEmpty('ui_menu_logo_up', UI_CUSTOM_DEFAULTS.menuLogoUp),
      menuLogoDown: nonEmpty('ui_menu_logo_down', UI_CUSTOM_DEFAULTS.menuLogoDown),
    };
  }, [data]);

  return { settings };
}
