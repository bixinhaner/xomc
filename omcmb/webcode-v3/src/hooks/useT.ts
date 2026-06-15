import { useIntl } from 'react-intl'

/** useT 返回的翻译函数签名。供需要把 t 当参数传递的辅助函数复用，保持契约一致。 */
export type TranslateFn = (id: string, values?: Record<string, string | number>) => string

/**
 * useT —— v3 (STARFORGE HUD) 皮肤的轻量翻译 hook（与 v1 webcode/src/hooks/useT.ts 契约一致）。
 * 消费 IntlProvider（providers/IntlProvider.tsx，读 appStore.locale + frontend-core getMessages）。
 * HUD 风格的纯英文装饰文案（· TECH/PLOT 等）不走 t()，仅中文实义文案国际化。
 */
export function useT(): TranslateFn {
  const intl = useIntl()
  return (id: string, values?: Record<string, string | number>) => {
    if (!id) return id
    return intl.formatMessage({ id }, values)
  }
}
