import { useNavigate } from 'react-router-dom';
import { useAlarmStore } from '@core/store/alarmStore';
import type { AlarmSeverity } from '@core/store/alarmStore';
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

export default function AlarmBadges() {
  const counts = useAlarmStore((s) => s.counts);
  const navigate = useNavigate();
  const t = useT();

  const handleClick = (severity: AlarmSeverity) => {
    void navigate(`/alarm/current?severity=${severity}`);
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
