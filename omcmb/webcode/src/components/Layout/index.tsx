import { useEffect, useCallback, useState } from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import { useAppStore } from '@core/store/appStore';
import { useMenuStore } from '@core/store/menuStore';
import { useTabStore } from '@core/store/tabStore';
import { useUserStore } from '@core/store/userStore';
import { useSecuritySettings } from '@core/hooks/api/useSecuritySettings';
import { useSystemTimezone } from '@core/hooks/api/useSystemTimezone';
import { useIdleLogout } from '@core/hooks/useIdleLogout';
import { useAgentVisibilityConfig } from '@core/hooks/api/useAgentConfig';
import { SIDEBAR_WIDTH, SIDEBAR_COLLAPSED_WIDTH } from '@/theme/tokens';
import { useResponsive } from '@/hooks/useResponsive';
import { useIsTouchDevice } from '@/hooks/useIsTouchDevice';
import Header from './Header';
import Sidebar from './Sidebar';
import TabBar from './TabBar';
import TaskPanel from './TaskPanel';
import QuickSettingsSyncWatcher from './QuickSettingsSyncWatcher';
import { isDynamicMenuEnabled } from '../MenuBootstrap/featureFlag';
import { NAV_CONFIG } from './Sidebar/navConfig';
import { resolveRouteTab } from './routeTabResolver';
import { AgentPanel } from '@/components/AgentPanel/AgentPanel';
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
  const locale = useAppStore((s) => s.locale);
  const dynamicMenus = useMenuStore((s) => s.flatMenus);
  const currentUser = useUserStore((s) => s.currentUser);
  const [agentOpen, setAgentOpen] = useState(false);
  const agentVisibility = useAgentVisibilityConfig();
  const agentVisible = agentVisibility.data?.visible === true;

  const { isMobile, isTablet } = useResponsive();
  const { isTouchPrimary } = useIsTouchDevice();

  // 全站标签页二级状态保持：URL（pathname + search）变化时同步进激活 tab 的 path，
  // 使「列表→二级（走 URL search）」状态被记入标签页，切走再切回不丢（系统级修复）。
  // BUG-11 修复：同时根据 pathname 激活对应的 tab，解决侧边栏导航时 tab 高亮不跟随的问题。
  const location = useLocation();
  const syncActiveTabPath = useTabStore((s) => s.syncActiveTabPath);
  const activateByPath = useTabStore((s) => s.activateByPath);
  const openTabForDirectEntry = useTabStore((s) => s.openTabForDirectEntry);
  useEffect(() => {
    const fullPath = location.pathname + location.search;
    // 已有 tab：激活并同步 query；地址栏直达且 tab 不存在时，按当前可见菜单补建。
    if (activateByPath(location.pathname)) {
      syncActiveTabPath(fullPath);
      return;
    }

    const directTab = resolveRouteTab({
      pathname: location.pathname,
      dynamicMenuEnabled: isDynamicMenuEnabled(),
      dynamicMenus,
      staticNav: NAV_CONFIG,
      locale,
      isAdmin: currentUser?.role === 'admin',
      isSuperAdmin: currentUser?.isSuperAdmin === true,
    });
    if (directTab) {
      openTabForDirectEntry({ ...directTab, path: fullPath });
    }
  }, [
    location.pathname,
    location.search,
    dynamicMenus,
    locale,
    currentUser?.role,
    currentUser?.isSuperAdmin,
    syncActiveTabPath,
    activateByPath,
    openTabForDirectEntry,
  ]);

  // P2-⑦ 屏幕锁定：监听 sys_configs.security.userSessionExpirationMin。
  // 0 = 禁用；非 0 表示 N 分钟无操作后强制登出。AppShell 仅在登录态渲染，
  // 因此 hook 在此挂载是正确时机（LoginPage 上不会跑）。
  const { settings: securitySettings } = useSecuritySettings();
  useIdleLogout(securitySettings?.idleLockMinutes ?? 0);

  // 系统时区（#459 子单 D）：登录后拉取 sys_configs basic/timezoneCode 写入 appStore，
  // 供顶部只读时钟、时间筛选输入按系统时区构造、epoch/Date 时间显示落系统钟面。
  useSystemTimezone();

  useEffect(() => {
    if (!agentVisible && agentOpen) {
      setAgentOpen(false);
    }
  }, [agentVisible, agentOpen]);

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
        <QuickSettingsSyncWatcher />
        <div className={styles.header}>
          <Header
            agentVisible={agentVisible}
            agentOpen={agentOpen}
            onAgentToggle={() => {
              if (agentVisible) setAgentOpen((open) => !open);
            }}
          />
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
          {/*
            key={locale}：切换语言时强制重挂载路由内容子树。
            页面里大量表格列（DataTableColumn.title）通过 useMemo 缓存翻译文案，
            部分页面的依赖数组未纳入 t/locale，导致切语言后当前活跃 tab 的表头
            不刷新、要重新点一次才更新。以 locale 作为 key 让活跃页在切语言时
            整体重建，所有 i18n 依赖的 memo 一并重算，单点根治、避免逐页补依赖。
            （非活跃 tab 本就未挂载，点开即按新语言渲染；切语言为低频操作，
             当前页重渲染的代价可接受。）
          */}
          <div key={locale} style={{ position: 'relative', zIndex: 1, height: '100%' }}>
            <Outlet />
          </div>
        </main>
        {!hideTaskPanel && (
          <div className={styles.taskPanel}>
            <TaskPanel />
          </div>
        )}
        {agentVisible && (
          <AgentPanel open={agentOpen} onClose={() => setAgentOpen(false)} />
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
