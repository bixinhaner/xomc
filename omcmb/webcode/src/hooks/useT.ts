import { useIntl } from 'react-intl';

export function useT() {
  const intl = useIntl();
  return (id: string, values?: Record<string, string | number>) =>
    intl.formatMessage({ id }, values);
}
