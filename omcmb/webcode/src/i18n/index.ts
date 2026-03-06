import type { Locale } from '../types/common';
import zhCN from './zh-CN';
import enUS from './en-US';

export { zhCN, enUS };

export const messages: Record<Locale, Record<string, string>> = {
  'zh-CN': zhCN,
  'en-US': enUS,
};

export const defaultLocale: Locale = 'zh-CN';

/**
 * Get the messages for a given locale, falling back to zh-CN.
 */
export function getMessages(locale: Locale): Record<string, string> {
  return messages[locale] ?? messages[defaultLocale];
}
