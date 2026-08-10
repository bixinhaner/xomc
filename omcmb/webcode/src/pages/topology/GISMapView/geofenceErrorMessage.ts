import type { IntlShape } from 'react-intl';
import { getI18nKeyByBizCode } from '@core/i18n/bizCodeMessages';

export function geofenceErrorMessage(
  intl: IntlShape,
  error: unknown,
  fallbackId = 'geofence.message.operationFailed',
): string {
  const bizCode = (error as { bizCode?: unknown } | null)?.bizCode;
  const messageId =
    typeof bizCode === 'number'
      ? getI18nKeyByBizCode(bizCode)
      : undefined;
  return intl.formatMessage({ id: messageId ?? fallbackId });
}
