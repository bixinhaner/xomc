/**
 * i18n 数据多语言取值工具。
 *
 * 项目里两种 i18n JSONB 形态并存:
 *   - 长 key (推荐):  { 'zh-CN': '...', 'en-US': '...' }   menus / device_groups / sys_dictionaries
 *   - 短 key (历史):  { zh: '...', en: '...' }              mml_command_groups / mml_commands
 *
 * getI18nText 把两种形态都吃进来,按 locale 命中即返,缺失依次回退到:
 *   1. 同 locale 长 key
 *   2. 同 locale 短 key
 *   3. 'zh-CN' / 'zh' (中文兜底)
 *   4. legacy 单语言列值
 *   5. 空串
 */

export type Locale = 'zh-CN' | 'en-US';

export type I18nMap = Record<string, string | null | undefined> | null | undefined;

const SHORT_KEY: Record<Locale, string> = {
  'zh-CN': 'zh',
  'en-US': 'en',
};

/**
 * 从 i18n map 里按 locale 取文案,缺失自动回退。
 */
export function getI18nText(
  i18n: I18nMap,
  locale: Locale,
  legacy?: string | null,
): string {
  if (i18n) {
    const longVal = i18n[locale];
    if (longVal) return longVal;
    const shortKey = SHORT_KEY[locale];
    const shortVal = shortKey ? i18n[shortKey] : undefined;
    if (shortVal) return shortVal;
    const zh = i18n['zh-CN'] ?? i18n['zh'];
    if (zh) return zh;
  }
  return legacy ?? '';
}

/**
 * 从 record 上同时按 `${fieldBase}_i18n` 取多语言,以 `record[fieldBase]` 作 legacy 兜底。
 * 例: getRecordI18n(group, 'name', 'en-US') →
 *   group.name_i18n['en-US'] ?? group.name_i18n.en ?? group.name_i18n['zh-CN'] ?? group.name_i18n.zh ?? group.name
 */
/** snake_case 转 camelCase 后追加 I18n。'name' → 'nameI18n', 'name_suffix' → 'nameSuffixI18n'。 */
function toCamelI18nKey(fieldBase: string): string {
  const camel = fieldBase.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase());
  return `${camel}I18n`;
}

export function getRecordI18n<T extends Record<string, unknown>>(
  record: T | null | undefined,
  fieldBase: string,
  locale: Locale,
): string {
  if (!record) return '';
  // 后端原样 (snake_case `name_i18n`) 与 Axios camelCase 转换后 (`nameI18n`) 两种 key 都吃。
  const i18n =
    (record[`${fieldBase}_i18n`] as I18nMap) ??
    (record[toCamelI18nKey(fieldBase)] as I18nMap);
  const legacy = record[fieldBase] as string | null | undefined;
  return getI18nText(i18n, locale, legacy);
}
