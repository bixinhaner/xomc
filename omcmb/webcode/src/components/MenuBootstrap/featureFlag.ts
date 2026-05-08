// 菜单动态加载灰度开关。独立文件以满足 react-refresh/only-export-components 规则。
//
// PRD docs/prd/system/menu-dynamic-loading.md §4.3.3：env 默认 false，UAT 通过后切 true。
const dynamicMenuEnabled =
  String(import.meta.env.VITE_DYNAMIC_MENU ?? 'false').toLowerCase() === 'true';

export function isDynamicMenuEnabled(): boolean {
  return dynamicMenuEnabled;
}
