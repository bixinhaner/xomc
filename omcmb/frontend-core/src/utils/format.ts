/**
 * 时间格式化工具函数
 */

/**
 * 格式化时间差为友好的文案
 * @param time - 要比较的时间
 * @param t - 国际化翻译函数
 * @returns 友好的时间差文案，如 "刚刚"、"5 分钟前"、"2 小时前"
 *
 * @example
 * formatTimeAgo(new Date(), t) // "刚刚"
 * formatTimeAgo(new Date(Date.now() - 5 * 60 * 1000), t) // "5 分钟前"
 */
export function formatTimeAgo(
  time: Date,
  t: (key: string, params?: Record<string, number | string>) => string
): string {
  const now = new Date();
  const diff = now.getTime() - time.getTime();
  const seconds = Math.floor(diff / 1000);

  if (seconds < 60) {
    return t('dashboard.justNow');
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return t('dashboard.minutesAgo', { count: minutes });
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return t('dashboard.hoursAgo', { count: hours });
  }
  const days = Math.floor(hours / 24);
  return t('dashboard.daysAgo', { count: days });
}
