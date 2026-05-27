import { createBrowserRouter } from 'react-router-dom';
import { routes } from './routes';

/**
 * Application router instance.
 * Use with <RouterProvider router={router} /> in App.tsx.
 */
const router = createBrowserRouter(routes, {
  future: {
    v7_normalizeFormMethod: true,
    // @ts-expect-error - v7_startTransition is supported in 6.30+ but types may lag
    v7_startTransition: true,
  },
});

export default router;
export { routes };
