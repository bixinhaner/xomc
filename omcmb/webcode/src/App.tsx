import { RouterProvider } from 'react-router-dom';
import QueryProvider from './providers/QueryProvider';
import ThemeProvider from './providers/ThemeProvider';
import LocaleProvider from './providers/LocaleProvider';
import router from './router';

/**
 * App is the root composition component.
 *
 * Provider nesting order (outermost → innermost):
 *   QueryProvider       — React Query cache and client
 *   ThemeProvider       — Ant Design theme + data-theme attribute on <html>
 *   LocaleProvider      — react-intl + Ant Design locale
 *   RouterProvider      — React Router v6 with all routes
 */
export default function App() {
  return (
    <QueryProvider>
      <ThemeProvider>
        <LocaleProvider>
          <RouterProvider router={router} />
        </LocaleProvider>
      </ThemeProvider>
    </QueryProvider>
  );
}
