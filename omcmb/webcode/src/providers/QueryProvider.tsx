import { useEffect, useRef, type ReactNode } from 'react';
import { QueryClient, QueryClientProvider, useQueryClient } from '@tanstack/react-query';
import { useUserStore } from '@core/store/userStore';

interface QueryProviderProps {
  children: ReactNode;
}

/**
 * Shared QueryClient instance.
 * - staleTime: 30 000 ms — cached data is considered fresh for 30 seconds.
 * - retry: 2 — failed requests are retried up to 2 times before surfacing an error.
 * - refetchOnWindowFocus: false — avoid unexpected re-fetches on tab switch.
 */
const queryClient = new QueryClient({
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

/**
 * AuthQueryBridge — 监听用户登录态从 true → false 时炸掉所有 React Query 缓存。
 *
 * 与 userStore.clearAuth 配套：clearAuth 已经清 zustand 的 menuStore + localStorage；
 * 本组件再清 React Query 内存缓存（userMenus / devices / alarms / roles 等所有
 * 用户作用域查询），确保下一个用户登录看到的所有数据都是新拉的，不会复用前一个
 * 用户的查询结果。
 *
 * 设计选择：
 *   - frontend-core 的 userStore 不能直接 import webcode 的 queryClient（依赖方向
 *     错位）。改用观察者模式，在 webcode 这层订阅 userStore 变化、操作 queryClient。
 *   - 仅监听 true → false 的转变，避免初次挂载（false → true）误清初始查询。
 */
function AuthQueryBridge() {
  const qc = useQueryClient();
  const isAuthenticated = useUserStore((s) => s.isAuthenticated);
  const prevAuthRef = useRef(isAuthenticated);

  useEffect(() => {
    if (prevAuthRef.current && !isAuthenticated) {
      qc.clear();
    }
    prevAuthRef.current = isAuthenticated;
  }, [isAuthenticated, qc]);

  return null;
}

/**
 * QueryProvider wraps the application with React Query's QueryClientProvider.
 */
export default function QueryProvider({ children }: QueryProviderProps) {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthQueryBridge />
      {children}
    </QueryClientProvider>
  );
}

export { queryClient };
