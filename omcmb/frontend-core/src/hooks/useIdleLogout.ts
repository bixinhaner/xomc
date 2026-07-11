import { useEffect, useRef } from 'react';
import { useUserStore } from '../store/userStore';
import { loginUrl } from '../utils/appBase';

/**
 * P2-⑦ 屏幕锁定：用户连续 idleMinutes 无操作后强制登出。
 *
 * 监听全局 mouse / keyboard / touch / scroll 事件刷新最近活动时间；setInterval
 * 每 30s 检查是否超时。idleMinutes <= 0 时禁用（idle 检查不运行）。
 *
 * 设计：
 *   - useRef 持有最近活动时间戳，避免闭包陈旧。
 *   - 监听 passive: true，对滚动性能 0 开销。
 *   - 触发后调用 useUserStore.logout 清 token + 跳 /login。
 *   - 多 Tab 间不同步 — 单 Tab 各自计时；登出后所有 tab 因 token 失效自然同步。
 *
 * 实例化建议：挂在主 layout（仅登录态可见），LoginPage 不挂避免误触发。
 */
export function useIdleLogout(idleMinutes: number): void {
  const logout = useUserStore((s) => s.logout);
  // null = 未初始化；effect 挂载时由 recordActivity 写入第一个时间戳。
  // 不在 useRef 初始值里调 Date.now() — React 19 purity 规则禁止 render 时调非纯函数。
  const lastActivityRef = useRef<number | null>(null);

  useEffect(() => {
    if (!idleMinutes || idleMinutes <= 0) {
      return; // 禁用
    }

    const idleMs = idleMinutes * 60 * 1000;
    lastActivityRef.current = Date.now();

    const recordActivity = (): void => {
      lastActivityRef.current = Date.now();
    };

    const events: Array<keyof WindowEventMap> = [
      'mousedown',
      'mousemove',
      'keydown',
      'touchstart',
      'scroll',
    ];
    for (const ev of events) {
      window.addEventListener(ev, recordActivity, { passive: true });
    }

    // 每 30s 检查一次：足够及时（远小于通常 idleMinutes ≥ 5）且开销低
    const checkInterval = window.setInterval(() => {
      const now = Date.now();
      const last = lastActivityRef.current ?? now; // effect 挂载时已写入，正常情况下非空
      if (now - last >= idleMs) {
        // 超时 — 登出 + 主动跳登录页（避免依赖 store 内的 navigate 副作用）
        logout();
        // 跳当前应用 base 下的登录页，兼容自定义子路径部署。
        const target = loginUrl();
        if (window.location.pathname !== target) {
          window.location.href = target;
        }
      }
    }, 30_000);

    return () => {
      for (const ev of events) {
        window.removeEventListener(ev, recordActivity);
      }
      window.clearInterval(checkInterval);
    };
  }, [idleMinutes, logout]);
}
