import { GlobalOutlined } from '@ant-design/icons';
import { useAppStore } from '@/store/appStore';
import { useT } from '@/hooks/useT';
import styles from './Header.module.css';

export default function TimezoneSelector() {
  const timezone = useAppStore((s) => s.timezone);
  const toggleTimezone = useAppStore((s) => s.toggleTimezone);
  const t = useT();

  const label = timezone === 'UTC' ? 'UTC' : t('header.localTimezone');

  return (
    <button
      className={styles.timezoneBtn}
      onClick={toggleTimezone}
      title={t('header.timezoneTitle', { tz: label })}
      type="button"
    >
      <GlobalOutlined style={{ fontSize: 12 }} />
      <span className={styles.timezoneLabel}>{t('header.timezone')}</span>
      <span>{label}</span>
    </button>
  );
}
