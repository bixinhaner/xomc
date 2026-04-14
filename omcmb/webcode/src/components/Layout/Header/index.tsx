import { Badge } from 'antd';
import {
  BellOutlined,
  MenuOutlined,
  SunOutlined,
  MoonOutlined,
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

export default function Header() {
  const deviceType = useAppStore((s) => s.deviceType);
  const theme = useAppStore((s) => s.theme);
  const toggleTheme = useAppStore((s) => s.toggleTheme);
  const setMobileOverlayOpen = useAppStore((s) => s.setMobileOverlayOpen);
  const { isMobile } = useResponsive();
  const t = useT();

  const deviceLabel = DEVICE_TYPE_KEY[deviceType] ? t(DEVICE_TYPE_KEY[deviceType]) : deviceType;

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
        {!isMobile && <div className={styles.logoDivider} />}
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

        {/* Light/Dark toggle */}
        <button
          className={styles.headerAction}
          onClick={toggleTheme}
          title={theme === 'tech' ? t('header.switchToLight') : t('header.switchToDark')}
          type="button"
        >
          {theme === 'tech' ? <SunOutlined style={{ fontSize: 16 }} /> : <MoonOutlined style={{ fontSize: 16 }} />}
        </button>

        <div className={styles.divider} />

        {/* User dropdown */}
        <UserDropdown />
      </div>
    </header>
  );
}
