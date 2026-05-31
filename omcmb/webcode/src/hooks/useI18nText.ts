import { useCallback } from 'react';
import { useAppStore } from '@core/store/appStore';
import {
  getI18nText,
  getRecordI18n,
  type I18nMap,
  type Locale,
} from '@core/utils/i18nText';

/**
 * React Hook 包装 getI18nText / getRecordI18n,从 appStore 自动读 locale。
 *
 * 用法:
 *   const { t, fromRecord } = useI18nText();
 *   t(group.name_i18n, group.name)               // → 当前 locale 文案
 *   fromRecord(group, 'name')                    // → 自动从 group.name_i18n + group.name 取
 */
export function useI18nText(): {
  locale: Locale;
  t: (i18n: I18nMap, legacy?: string | null) => string;
  fromRecord: <T extends Record<string, unknown>>(
    record: T | null | undefined,
    fieldBase: string,
  ) => string;
} {
  const locale = useAppStore((s) => s.locale);
  const t = useCallback(
    (i18n: I18nMap, legacy?: string | null) => getI18nText(i18n, locale, legacy),
    [locale],
  );
  const fromRecord = useCallback(
    <T extends Record<string, unknown>>(record: T | null | undefined, fieldBase: string) =>
      getRecordI18n(record, fieldBase, locale),
    [locale],
  );
  return { locale, t, fromRecord };
}
