import { useMemo } from 'react';
import { useIntl } from 'react-intl';
import { useMenuStore } from '@core/store/menuStore';
import { resolveMenuLabel } from '@core/types/menu';
import type { TabItem } from '@core/store/tabStore';
import { useT } from '@/hooks/useT';

/**
 * 把一个 tab 解析成「当前语言」下的标题。
 *
 * tab.label 有两种来源：
 *  - labelRaw=false：label 是前端 i18n key（静态菜单）→ t(key) 实时翻译。
 *  - labelRaw=true ：label 是「打开 tab 那一刻」按当时 locale 解析好的字符串：
 *      · 动态菜单(DB)叶子 —— 见 NavMenu.handleMenuClick；
 *      · 设备详情等 drill-down —— label 是设备名等业务数据（不可翻译）。
 *    旧实现直接显示这个冻结串，导致切换语言后已打开的动态菜单 tab 标题不刷新
 *    （侧边栏重解析了、tab 没有），出现「中英混排」。
 *
 * 这里对 labelRaw 的 tab 按 tab.path 反查 menuStore 的 nameI18n，用当前 locale
 * 重新解析，使已打开 tab 的标题也随切换语言刷新；查不到对应菜单（菜单被删 /
 * 未加载 / 设备名这类 drill-down 路径不在菜单里）才回退到冻结串——设备名因此
 * 不会被误翻译。
 */
export function useTabLabelResolver(): (tab: TabItem) => string {
  const t = useT();
  const { locale } = useIntl();
  const flatMenus = useMenuStore((s) => s.flatMenus);

  // routePath → { name, nameI18n }，供按 path 反查菜单译文。
  const pathToMenu = useMemo(() => {
    const map = new Map<string, { name: string; nameI18n?: Record<string, string> }>();
    for (const m of flatMenus) {
      if (m.routePath) map.set(m.routePath, { name: m.name, nameI18n: m.nameI18n });
    }
    return map;
  }, [flatMenus]);

  return (tab: TabItem): string => {
    if (!tab.labelRaw) return t(tab.label);
    // 去掉 URL 上的二级 query（syncActiveTabPath 会把 search 写进 path）。
    const basePath = tab.path.split('?')[0];
    const menu = pathToMenu.get(basePath);
    if (menu) return resolveMenuLabel(menu, locale);
    return tab.label;
  };
}
