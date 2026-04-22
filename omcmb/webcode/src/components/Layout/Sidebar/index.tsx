import { useAppStore } from '@core/store/appStore';
import { SIDEBAR_WIDTH, SIDEBAR_COLLAPSED_WIDTH } from '@/theme/tokens';
import NavMenu from './NavMenu';
import { useT } from '@/hooks/useT';
import styles from './Sidebar.module.css';

export default function Sidebar() {
  const collapsed = useAppStore((s) => s.sidebarCollapsed);
  const sidebarPosition = useAppStore((s) => s.sidebarPosition);
  const isTop = sidebarPosition === 'top';
  const width = isTop ? '100%' : collapsed ? SIDEBAR_COLLAPSED_WIDTH : SIDEBAR_WIDTH;
  const t = useT();

  return (
    <aside
      className={`${styles.sidebar}${isTop ? ` ${styles.sidebarHorizontal}` : ''}`}
      style={{ width }}
      aria-label="导航菜单"
    >
      {/* Logo area — fixed at top, not scrollable */}
      {!isTop && (
        <div className={`${styles.logoArea}${collapsed ? ` ${styles.logoAreaCollapsed}` : ''}`}>
          <div className={styles.logoMark}>OMC</div>
          {!collapsed && <span className={styles.logoName}>{t('app.title')}</span>}
        </div>
      )}
      <div className={styles.menuWrapper}>
        <NavMenu collapsed={collapsed} position={sidebarPosition} />
      </div>
      {!isTop && (
        <div className={styles.versionArea}>
          {!collapsed && <span className={styles.version}>v1.0.0</span>}
        </div>
      )}
    </aside>
  );
}
