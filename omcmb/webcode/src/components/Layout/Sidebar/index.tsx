import { useAppStore } from '@/store/appStore';
import { SIDEBAR_WIDTH, SIDEBAR_COLLAPSED_WIDTH } from '@/theme/tokens';
import DeviceTypeSelector from './DeviceTypeSelector';
import NavMenu from './NavMenu';
import CollapseButton from './CollapseButton';
import styles from './Sidebar.module.css';

export default function Sidebar() {
  const collapsed = useAppStore((s) => s.sidebarCollapsed);
  const sidebarPosition = useAppStore((s) => s.sidebarPosition);
  const isTop = sidebarPosition === 'top';
  const width = isTop ? '100%' : collapsed ? SIDEBAR_COLLAPSED_WIDTH : SIDEBAR_WIDTH;

  return (
    <aside
      className={`${styles.sidebar}${isTop ? ` ${styles.sidebarHorizontal}` : ''}`}
      style={{ width }}
      aria-label="导航菜单"
    >
      {!isTop && <DeviceTypeSelector collapsed={collapsed} />}
      <div className={styles.menuWrapper}>
        <NavMenu collapsed={collapsed} position={sidebarPosition} />
      </div>
      {!isTop && <CollapseButton />}
    </aside>
  );
}
