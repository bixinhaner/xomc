import { useEffect, useCallback } from 'react';
import { Outlet } from 'react-router-dom';
import { useAppStore } from '@core/store/appStore';
import { useSecuritySettings } from '@core/hooks/api/useSecuritySettings';
import { useIdleLogout } from '@core/hooks/useIdleLogout';
import { SIDEBAR_WIDTH, SIDEBAR_COLLAPSED_WIDTH } from '@/theme/tokens';
import { useResponsive } from '@/hooks/useResponsive';
import { useIsTouchDevice } from '@/hooks/useIsTouchDevice';
import Header from './Header';
import Sidebar from './Sidebar';
import TabBar from './TabBar';
import TaskPanel from './TaskPanel';
import ParticleCanvas from '@/components/Effects/ParticleCanvas';
import DynamicLightSource from '@/components/Effects/DynamicLightSource';
import styles from './AppShell.module.css';

export default function AppShell() {
  const sidebarCollapsed = useAppStore((s) => s.sidebarCollapsed);
  const sidebarPosition = useAppStore((s) => s.sidebarPosition);
  const tabBarPosition = useAppStore((s) => s.tabBarPosition);
  const effects3D = useAppStore((s) => s.effects3DEnabled);
  const isMobileOverlayOpen = useAppStore((s) => s.isMobileOverlayOpen);
  const setMobileOverlayOpen = useAppStore((s) => s.setMobileOverlayOpen);

  const { isMobile, isTablet } = useResponsive();
  const { isTouchPrimary } = useIsTouchDevice();

  // P2-⑦ 屏幕锁定：监听 sys_configs.security.userSessionExpirationMin。
  // 0 = 禁用；非 0 表示 N 分钟无操作后强制登出。AppShell 仅在登录态渲染，
  // 因此 hook 在此挂载是正确时机（LoginPage 上不会跑）。
  const { settings: securitySettings } = useSecuritySettings();
  useIdleLogout(securitySettings?.idleLockMinutes ?? 0);

  // 隐藏任务面板
  const hideTaskPanel = true;

  // Auto-collapse sidebar on tablet
  useEffect(() => {
    if (isTablet && !sidebarCollapsed) {
      useAppStore.getState().setSidebarCollapsed(true);
    }
  }, [isTablet, sidebarCollapsed]);

  // Auto-disable 3D effects on mobile touch devices
  useEffect(() => {
    if (isMobile && isTouchPrimary && effects3D) {
      useAppStore.getState().setEffects3DEnabled(false);
    }
  }, [isMobile, isTouchPrimary, effects3D]);

  // Close mobile overlay on Escape key
  useEffect(() => {
    if (!isMobileOverlayOpen) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setMobileOverlayOpen(false);
    };
    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [isMobileOverlayOpen, setMobileOverlayOpen]);

  const handleOverlayClick = useCallback(() => {
    setMobileOverlayOpen(false);
  }, [setMobileOverlayOpen]);

  const isTop = sidebarPosition === 'top';
  const sidebarWidth = sidebarCollapsed ? SIDEBAR_COLLAPSED_WIDTH : SIDEBAR_WIDTH;

  return (
    <>
      <div
        className={styles.shell}
        data-sidebar={sidebarPosition}
        data-tabbar={tabBarPosition}
        style={
          isTop ? undefined : { '--current-sidebar-width': `${sidebarWidth}px` } as React.CSSProperties
        }
      >
        <div className={styles.header}>
          <Header />
        </div>
        <div className={styles.sidebar}>
          <Sidebar />
        </div>
        <div className={styles.tabBar}>
          <TabBar />
        </div>
        <main className={styles.content} style={{ position: 'relative' }}>
          {effects3D && <ParticleCanvas />}
          {effects3D && <DynamicLightSource />}
          <div style={{ position: 'relative', zIndex: 1, height: '100%' }}>
            <Outlet />
          </div>
        </main>
        {!hideTaskPanel && (
          <div className={styles.taskPanel}>
            <TaskPanel />
          </div>
        )}
      </div>

      {/* Mobile sidebar drawer overlay */}
      {isMobile && isMobileOverlayOpen && (
        <div className={styles.mobileOverlay} onClick={handleOverlayClick}>
          <aside
            className={styles.mobileDrawer}
            onClick={(e) => e.stopPropagation()}
          >
            <Sidebar />
          </aside>
        </div>
      )}
    </>
  );
}
