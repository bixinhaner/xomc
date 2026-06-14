import type { ReactNode } from 'react'
import { IntlProvider as ReactIntlProvider } from 'react-intl'

import { useAppStore } from '@core/store/appStore'
import { getMessages } from '@core/i18n'

export function IntlProvider({ children }: { children: ReactNode }) {
  const locale = useAppStore((s) => s.locale)
  const messages = getMessages(locale)

  return (
    <ReactIntlProvider
      locale={locale === 'zh-CN' ? 'zh-CN' : 'en-US'}
      messages={messages}
      defaultLocale="zh-CN"
    >
      {children}
    </ReactIntlProvider>
  )
}
