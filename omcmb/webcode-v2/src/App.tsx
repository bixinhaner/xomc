import { RouterProvider } from 'react-router-dom'

import { QueryProvider } from '@/providers/QueryProvider'
import { IntlProvider } from '@/providers/IntlProvider'
import { router } from '@/router'
import QuickSettingsSyncWatcherCore from '@core/components/QuickSettingsSyncWatcherCore'

export default function App() {
  return (
    <QueryProvider>
      <IntlProvider>
        <QuickSettingsSyncWatcherCore />
        <RouterProvider router={router} />
      </IntlProvider>
    </QueryProvider>
  )
}
