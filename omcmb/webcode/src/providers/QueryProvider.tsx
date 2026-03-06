import { type ReactNode } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

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
 * QueryProvider wraps the application with React Query's QueryClientProvider.
 */
export default function QueryProvider({ children }: QueryProviderProps) {
  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

export { queryClient };
