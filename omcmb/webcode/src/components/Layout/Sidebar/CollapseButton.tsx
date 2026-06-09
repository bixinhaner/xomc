import { MenuFoldOutlined, MenuUnfoldOutlined } from '@ant-design/icons';
import { useAppStore } from '@core/store/appStore';
import { useT } from '@/hooks/useT';
import { APP_VERSION } from '@/version';
import styles from './Sidebar.module.css';

export default function CollapseButton() {
  const collapsed = useAppStore((s) => s.sidebarCollapsed);
  const sidebarPosition = useAppStore((s) => s.sidebarPosition);
  const toggleSidebar = useAppStore((s) => s.toggleSidebar);
  const t = useT();

  const isLeft = sidebarPosition === 'left';
  const CollapseIcon = collapsed
    ? (isLeft ? MenuUnfoldOutlined : MenuFoldOutlined)
    : (isLeft ? MenuFoldOutlined : MenuUnfoldOutlined);

  return (
    <div className={`${styles.collapseArea}${collapsed ? ` ${styles.collapsedCollapseArea}` : ''}`}>
      {!collapsed && (
        <span className={styles.version} title={APP_VERSION}>{APP_VERSION}</span>
      )}
      <button
        className={styles.collapseBtn}
        onClick={toggleSidebar}
        title={collapsed ? t('sidebar.expand') : t('sidebar.collapse')}
        type="button"
        aria-label={collapsed ? t('sidebar.expand') : t('sidebar.collapse')}
      >
        <CollapseIcon style={{ fontSize: 10 }} />
      </button>
    </div>
  );
}
