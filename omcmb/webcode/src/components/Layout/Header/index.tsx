import { Badge } from 'antd';
import {
  BellOutlined,
  MenuOutlined,
  SkinOutlined,
  ExperimentOutlined,
  CoffeeOutlined,
  ThunderboltOutlined,
  SmileOutlined,
  StarOutlined,
  DollarOutlined,
} from '@ant-design/icons';
import { useAppStore } from '@/store/appStore';
import { useResponsive } from '@/hooks/useResponsive';
import { useT } from '@/hooks/useT';
import AlarmBadges from './AlarmBadges';
import TimezoneSelector from './TimezoneSelector';
import UserDropdown from './UserDropdown';
import styles from './Header.module.css';

const DEVICE_TYPE_KEY: Record<string, string> = {
  all: 'device.type.all',
  eNB: 'device.type.eNB',
  gNB: 'device.type.gNB',
  CPE: 'device.type.CPE',
  eGW: 'device.type.eGW',
};

// Theme icon and tooltip mapping
const THEME_CONFIG: Record<string, { icon: React.ReactNode; nextTooltip: string }> = {
  classic: { icon: <SkinOutlined style={{ fontSize: 16 }} />, nextTooltip: 'header.switchToTech' },
  tech: { icon: <ExperimentOutlined style={{ fontSize: 16 }} />, nextTooltip: 'header.switchToFresh' },
  fresh: { icon: <CoffeeOutlined style={{ fontSize: 16 }} />, nextTooltip: 'header.switchToCyberpunk' },
  cyberpunk: { icon: <ThunderboltOutlined style={{ fontSize: 16 }} />, nextTooltip: 'header.switchToMinions' },
  minions: { icon: <SmileOutlined style={{ fontSize: 16 }} />, nextTooltip: 'header.switchToTiffany' },
  tiffany: { icon: <StarOutlined style={{ fontSize: 16 }} />, nextTooltip: 'header.switchToRmb' },
  rmb: { icon: <DollarOutlined style={{ fontSize: 16 }} />, nextTooltip: 'header.switchToClassic' },
};

export default function Header() {
  const deviceType = useAppStore((s) => s.deviceType);
  const theme = useAppStore((s) => s.theme);
  const toggleTheme = useAppStore((s) => s.toggleTheme);
  const setMobileOverlayOpen = useAppStore((s) => s.setMobileOverlayOpen);
  const { isMobile } = useResponsive();
  const t = useT();

  const deviceLabel = DEVICE_TYPE_KEY[deviceType] ? t(DEVICE_TYPE_KEY[deviceType]) : deviceType;
  const themeConfig = THEME_CONFIG[theme] || THEME_CONFIG.classic;

  return (
    <header className={styles.header}>
      {/* Left zone — Hamburger (mobile) + Logo + System name */}
      <div className={styles.left}>
        {isMobile && (
          <button
            className={styles.hamburger}
            onClick={() => setMobileOverlayOpen(true)}
            aria-label="Open navigation"
            type="button"
          >
            <MenuOutlined />
          </button>
        )}
        <div className={styles.logoMark}>OMC</div>
        {!isMobile && <span className={styles.systemName}>{t('app.title')}</span>}
      </div>

      {/* Center zone — Current device type indicator */}
      <div className={styles.center}>
        <div className={styles.deviceIndicator}>
          <span className={styles.deviceIndicatorLabel}>{t('header.currentView')}</span>
          <span className={styles.deviceIndicatorValue}>{deviceLabel}</span>
        </div>
      </div>

      {/* Right zone — Actions */}
      <div className={styles.right}>
        {/* Alarm severity badges */}
        <AlarmBadges />

        <div className={styles.divider} />

        {/* Notification bell */}
        <button className={styles.headerAction} title={t('header.notification')} type="button">
          <Badge count={0} size="small">
            <BellOutlined style={{ fontSize: 16 }} />
          </Badge>
        </button>

        <div className={styles.divider} />

        {/* Timezone selector */}
        <TimezoneSelector />

        <div className={styles.divider} />

        {/* Style toggle */}
        <button
          className={styles.headerAction}
          onClick={toggleTheme}
          title={t(themeConfig.nextTooltip)}
          type="button"
        >
          {themeConfig.icon}
        </button>

        <div className={styles.divider} />

        {/* User dropdown */}
        <UserDropdown />
      </div>
    </header>
  );
}
