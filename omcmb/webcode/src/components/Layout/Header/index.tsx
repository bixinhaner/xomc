import { useState, useEffect } from 'react';
import { Badge, Popover } from 'antd';
import {
  BellOutlined,
  MenuOutlined,
  SunOutlined,
  MoonOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons';
import { useAppStore } from '@core/store/appStore';
import { useAlarmStore } from '@core/store/alarmStore';
import { useNotificationUnreadCount, useSyncStaleNotifications } from '@core/hooks/api/useNotificationCenter';
import { useAlarmCount } from '@core/hooks/api/useAlarms';
import { useResponsive } from '@/hooks/useResponsive';
import { useT } from '@/hooks/useT';
import NotificationCenter from '@/components/NotificationCenter';
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
  const sidebarCollapsed = useAppStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useAppStore((s) => s.toggleSidebar);
  const { isMobile } = useResponsive();
  const t = useT();
  // T-0157 C9: 消息中心 — 铃铛 Popover 受控开关 + 未读数 10s 轮询徽标
  const [notifOpen, setNotifOpen] = useState(false);
  const unreadQuery = useNotificationUnreadCount();
  const unreadCount = unreadQuery.data ?? 0;
  // T-0157 stale sync: 打开 Popover 时反查 task 状态修正卡死的 queued/sent 消息
  const syncStale = useSyncStaleNotifications();

  // 获取告警统计数据并同步到 store
  const { data: alarmCount } = useAlarmCount();
  const setAlarmCounts = useAlarmStore((s) => s.setCounts);

  useEffect(() => {
    if (alarmCount) {
      setAlarmCounts(alarmCount);
    }
  }, [alarmCount, setAlarmCounts]);

  const deviceLabel = DEVICE_TYPE_KEY[deviceType] ? t(DEVICE_TYPE_KEY[deviceType]) : deviceType;

  return (
    <header className={styles.header}>
      {/* Left zone */}
      <div className={styles.left}>
        {isMobile ? (
          <>
            <button
              className={styles.hamburger}
              onClick={() => setMobileOverlayOpen(true)}
              aria-label="Open navigation"
              type="button"
            >
              <MenuOutlined />
            </button>
            <div className={styles.logoMark}>OMC</div>
            <span className={styles.systemName}>{t('app.title')}</span>
          </>
        ) : (
          <button
            className={styles.collapseBtn}
            onClick={toggleSidebar}
            title={sidebarCollapsed ? t('sidebar.expand') : t('sidebar.collapse')}
            type="button"
            aria-label={sidebarCollapsed ? t('sidebar.expand') : t('sidebar.collapse')}
          >
            {sidebarCollapsed ? <MenuUnfoldOutlined style={{ fontSize: 16 }} /> : <MenuFoldOutlined style={{ fontSize: 16 }} />}
          </button>
        )}
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

        {/* Notification bell (T-0157 C9 — 消息中心 Popover) */}
        <Popover
          trigger="click"
          placement="bottomRight"
          open={notifOpen}
          onOpenChange={(open) => {
            setNotifOpen(open);
            // 打开时触发 stale sync — 失败不阻塞列表加载（hook 内部 onSuccess 自动 invalidate）
            if (open) {
              syncStale.mutate();
            }
          }}
          content={<NotificationCenter onClose={() => setNotifOpen(false)} />}
          styles={{ content: { padding: 0 } }}
        >
          <button className={styles.headerAction} title={t('header.notification')} type="button">
            <Badge count={unreadCount} size="small" overflowCount={99}>
              <BellOutlined style={{ fontSize: 16 }} />
            </Badge>
          </button>
        </Popover>

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
