import { useIntl } from 'react-intl';
import { useCallback } from 'react';

/** useT 返回的翻译函数签名。供需要把 t 当参数传递的辅助函数复用，保持契约一致。 */
export type TranslateFn = (id: string, values?: Record<string, string | number>) => string;

export function useT(): TranslateFn {
  const intl = useIntl();
  return useCallback((id: string, values?: Record<string, string | number>) => {
    if (!id) return id as string;
    return intl.formatMessage({ id }, values);
  }, [intl]);
}
