import { GlobalOutlined } from '@ant-design/icons';
import { useSystemClock } from '@core/hooks/useSystemClock';
import { useT } from '@/hooks/useT';
import styles from './Header.module.css';

/**
 * 顶部只读时钟（#459 子单 D）。
 *
 * 原为「本地/UTC」切换按钮（写 appStore.timezone 但无页面读取）→ 改为只读展示
 * 「当前系统时区 + 实时当前时间」（每秒按系统时区刷新），去掉切换交互。
 * 系统时区来自 sys_configs basic/timezoneCode（登录后由 useSystemTimezone 写入 appStore）。
 */
export default function TimezoneSelector() {
  const { timezoneLabel, dateTime } = useSystemClock();
  const t = useT();

  return (
    <div
      className={styles.timezoneClock}
      title={t('header.timezoneTitle', { tz: timezoneLabel })}
    >
      <GlobalOutlined style={{ fontSize: 12 }} />
      <span className={styles.timezoneLabel}>{timezoneLabel}</span>
      <span className={styles.timezoneClockTime}>{dateTime}</span>
    </div>
  );
}
