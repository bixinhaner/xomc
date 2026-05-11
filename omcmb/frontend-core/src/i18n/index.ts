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
 * SUPPORTED_LOCALES：当前应用支持的所有语言代码。
 *
 * 单点真理：扩展新语言时，**只需在此数组追加** + 在 messages 注册对应包 +
 * 在 LOCALE_DISPLAY 加显示名，菜单管理表单 / 语言切换器 / 译文 fallback 链路
 * 自动跟进。
 *
 * 设计依据：T-0113 菜单多语言改造方案 §C，「后期要能够支持新语言」需求。
 */
export const SUPPORTED_LOCALES: readonly Locale[] = ['zh-CN', 'en-US'];

/**
 * LOCALE_DISPLAY：locale code → 用户可见的语言标签。
 *
 * 设计取舍：用语言自身书写的名字（中文、English、日本語），而非翻译过的名字
 * （中文→Chinese、English→英文），这样切语言菜单本身永远能让用户「看见自己的语言」。
 */
export const LOCALE_DISPLAY: Record<Locale, string> = {
  'zh-CN': '中文',
  'en-US': 'English',
};

/**
 * Get the messages for a given locale, falling back to zh-CN.
 */
export function getMessages(locale: Locale): Record<string, string> {
  return messages[locale] ?? messages[defaultLocale];
}
