import { memo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAlarmStore } from '@core/store/alarmStore';
import { useShallow } from 'zustand/react/shallow';
import type { AlarmSeverity } from '@core/store/alarmStore';
import { useTabStore } from '@core/store/tabStore';
import { useT } from '@/hooks/useT';
import styles from './Header.module.css';

interface SeverityConfig {
  key: AlarmSeverity;
  labelKey: string;
  pillClass: string;
  icon: string;
}

const SEVERITY_CONFIG: SeverityConfig[] = [
  { key: 'critical', labelKey: 'alarm.severity.critical', pillClass: styles.alarmPillCritical, icon: '■' },
  { key: 'major',    labelKey: 'alarm.severity.major',    pillClass: styles.alarmPillMajor,    icon: '▲' },
  { key: 'minor',    labelKey: 'alarm.severity.minor',    pillClass: styles.alarmPillMinor,    icon: '●' },
  { key: 'warning',  labelKey: 'alarm.severity.warning',  pillClass: styles.alarmPillWarning,  icon: '◆' },
];

// 性能（#15）：本组件无 props，仅从 alarmStore 选 counts。常驻 Header 会因未读数/
// 主题/侧栏/响应式等无关原因频繁重渲染——memo 让它在 Header 重渲染时跳过，
// 只在自身 counts 选择值变化时才更新。
function AlarmBadges() {
  const counts = useAlarmStore(useShallow((s) => s.counts));
  const navigate = useNavigate();
  const openTab = useTabStore((s) => s.openTab);
  const t = useT();

  const handleClick = (severity: AlarmSeverity) => {
    const path = `/alarm/current?severity=${severity}`;
    openTab({
      key: 'alarm-current',
      label: 'nav.alarm.current',
      labelRaw: false,
      path,
      closable: true,
    });
    void navigate(path);
  };

  return (
    <div className={styles.alarmBadgesWrapper}>
      {SEVERITY_CONFIG.map(({ key, labelKey, pillClass, icon }) => (
        <button
          key={key}
          className={`${styles.alarmPill} ${pillClass}`}
          onClick={() => handleClick(key)}
          title={`${t(labelKey)}: ${counts[key]}`}
          type="button"
        >
          <span className={styles.alarmPillIcon}>{icon}</span>
          <span className={styles.alarmPillCount}>{counts[key]}</span>
        </button>
      ))}
    </div>
  );
}

export default memo(AlarmBadges);
