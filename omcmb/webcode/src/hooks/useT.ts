import { useIntl } from 'react-intl';

export function useT() {
  const intl = useIntl();
  return (id: string, values?: Record<string, string | number>) => {
    if (!id) return id as string;
    return intl.formatMessage({ id }, values);
  };
}
