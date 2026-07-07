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
        try {
          // 防抖：后端事件如果像瀑布一样涌来（比如批量解析结束或者大批量告警），
          // 控制 React Query 不要1秒内发几十次无意义的相同请求，锁定 2 秒内的只算作 1 次渲染。
          clearTimeout(timerRef.current);
          timerRef.current = window.setTimeout(() => {
            queryClient.invalidateQueries({ queryKey: ['dashboard'] });
          }, 2000);

        } catch (error) {
          console.error('[Dashboard SSE] 解析 update 事件失败', error);
        }
      });

      source.onerror = (e) => {
        console.error('[Dashboard SSE] 连接错误', e);
      };

    } catch (e) {
      console.error('[Dashboard SSE] 建立连接异常', e);
    }

    return () => {
      source?.close();
      clearTimeout(timerRef.current);
    };
  }, [accessToken, queryClient]);
}
