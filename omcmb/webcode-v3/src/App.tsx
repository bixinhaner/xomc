import { useEffect } from 'react'
import { RouterProvider } from 'react-router-dom'

import { QueryProvider } from '@/providers/QueryProvider'
import { IntlProvider } from '@/providers/IntlProvider'
import { router } from '@/router'

export default function App() {
  // STARFORGE 默认强制深色模式（仅 v3 生效）
  useEffect(() => {
    document.documentElement.classList.add('dark', 'starforge')
    return () => {
      document.documentElement.classList.remove('starforge')
    }
  }, [])

  return (
    <QueryProvider>
      <IntlProvider>
        <RouterProvider router={router} />
      </IntlProvider>
    </QueryProvider>
  )
}
