import { RouterProvider } from 'react-router-dom'

import { QueryProvider } from '@/providers/QueryProvider'
import { IntlProvider } from '@/providers/IntlProvider'
import { router } from '@/router'

export default function App() {
  return (
    <QueryProvider>
      <IntlProvider>
        <RouterProvider router={router} />
      </IntlProvider>
    </QueryProvider>
  )
}
