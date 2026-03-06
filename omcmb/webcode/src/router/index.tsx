import { createBrowserRouter } from 'react-router-dom';
import { routes } from './routes';

/**
 * Application router instance.
 * Use with <RouterProvider router={router} /> in App.tsx.
 */
const router = createBrowserRouter(routes, {
  future: {
    v7_normalizeFormMethod: true,
  },
});

export default router;
export { routes };
