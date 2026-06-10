import { createBrowserRouter } from 'react-router-dom';
import { routes } from './routes';

/**
 * Application router instance.
 * Use with <RouterProvider router={router} /> in App.tsx.
 */
// react-router v7:原 v6 的 v7_* future flags 已成为默认行为,future 块随升级移除。
const router = createBrowserRouter(routes);

export default router;
export { routes };
