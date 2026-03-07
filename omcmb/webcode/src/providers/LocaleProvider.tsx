import { type ReactNode } from 'react';
import { IntlProvider } from 'react-intl';
import { useAppStore } from '@/store/appStore';
import { getMessages } from '@/i18n';

interface LocaleProviderProps {
  children: ReactNode;
}

/**
 * LocaleProvider reads the current locale from appStore and wraps children
 * with react-intl IntlProvider for i18n message formatting.
 *
 * NOTE: Ant Design locale is now handled by ThemeProvider's single
 * ConfigProvider to avoid the dual-ConfigProvider cascade issue.
 */
export default function LocaleProvider({ children }: LocaleProviderProps) {
  const locale = useAppStore((s) => s.locale);
  const messages = getMessages(locale);

  return (
    <IntlProvider locale={locale} messages={messages} defaultLocale="zh-CN">
      {children}
    </IntlProvider>
  );
}
