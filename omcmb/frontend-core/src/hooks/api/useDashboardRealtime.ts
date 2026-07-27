import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useUserStore } from '../../store/userStore';

const getBaseURL = () => {
  return import.meta.env.VITE_API_PROXY_TARGET
    ? '/api/v1'
    : window.location.origin + '/api/v1';
};

/**
 * 订阅服务器针对 Dashboard 大盘级别的全局 SSE 事件。
 * 代替原本极其消耗性能的 refetchInterval 轮询。
 */
export function useDashboardRealtime() {
  const queryClient = useQueryClient();
  const accessToken = useUserStore((s) => s.accessToken);
  const timerRef = useRef<number | undefined>(undefined);
  const lastErrorAtRef = useRef(0);

  useEffect(() => {
    if (!accessToken) return;

    const baseURL = getBaseURL();
    const url = `${baseURL}/events/stream?token=${encodeURIComponent(
      accessToken
    )}`;

    let source: EventSource | null = null;
    try {
      source = new EventSource(url, { withCredentials: true });

      source.addEventListener('dashboard_update', () => {
        // 防抖：后端事件如果像瀑布一样涌来（比如批量解析结束或者大批量告警），
        // 控制 React Query 不要1秒内发几十次无意义的相同请求，锁定 2 秒内的只算作 1 次渲染。
        clearTimeout(timerRef.current);
        timerRef.current = window.setTimeout(() => {
          queryClient.invalidateQueries({ queryKey: ['dashboard'] });
        }, 2000);
      });

      source.onerror = () => {
        const now = Date.now();
        if (import.meta.env.DEV && now - lastErrorAtRef.current >= 30_000) {
          lastErrorAtRef.current = now;
          console.warn('[Dashboard SSE] 实时连接暂时不可用，将由浏览器自动重连');
        }
      };

    } catch (e) {
      if (import.meta.env.DEV) {
        console.warn('[Dashboard SSE] 建立实时连接失败', e);
      }
    }

    return () => {
      source?.close();
      clearTimeout(timerRef.current);
    };
  }, [accessToken, queryClient]);
}
