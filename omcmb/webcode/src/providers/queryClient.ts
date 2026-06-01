import { QueryClient } from '@tanstack/react-query';

/**
 * Shared QueryClient instance.
 * - staleTime: 30 000 ms — cached data is considered fresh for 30 seconds.
 * - retry: 2 — failed requests are retried up to 2 times before surfacing an error.
 * - refetchOnWindowFocus: false — avoid unexpected re-fetches on tab switch.
 *
 * 从 QueryProvider.tsx 拆出（react-refresh/only-export-components：
 * 组件文件只导出组件，共享单例放本文件）。
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 2,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
});
