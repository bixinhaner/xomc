/**
 * 格式化工具函数
 * 包含时间、数值、KPI等格式化功能
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

/**
 * 格式化KPI数值显示
 * 统一保留两位小数，单位紧跟数值无空格（参考老系统：76.36Mbps）
 * 非正常值（undefined/null/NaN）显示为 -
 *
 * @param value - 数值
 * @param unitKey - 单位i18n key
 * @param t - 翻译函数
 * @returns 格式化后的字符串
 *
 * @example
 * formatKPIValue(1024, 'unit.gb', t) // "1.00TB"
 * formatKPIValue(76.36, 'unit.mbps', t) // "76.36Mbps"
 * formatKPIValue(95, 'unit.percent', t) // "95.00%"
 * formatKPIValue(null, 'unit.gb', t) // "-"
 */
export function formatKPIValue(
  value: number | null | undefined,
  unitKey: string,
  t: (key: string) => string
): string {
  // 处理非正常值，统一显示 -
  if (value === null || value === undefined || Number.isNaN(value)) {
    return '-';
  }

  const unit = t(unitKey);

  // 统一保留两位小数
  const formattedValue = value.toFixed(2);

  // 处理百分比 - 直接显示数值+%
  if (unit === '%' || unitKey === 'unit.percent') {
    return `${formattedValue}%`;
  }

  // 处理GB - 超过1024自动转换为TB，保持2位小数
  if (unit === 'GB' || unitKey === 'unit.gb') {
    if (value >= 1024) {
      return `${(value / 1024).toFixed(2)}TB`;
    }
    return `${formattedValue}GB`;
  }

  // 处理MB - 超过1024自动转换为GB，保持2位小数
  if (unit === 'MB' || unitKey === 'unit.mb') {
    if (value >= 1024) {
      return `${(value / 1024).toFixed(2)}GB`;
    }
    return `${formattedValue}MB`;
  }

  // 处理Mbps - 超过1024自动转换为Gbps，保持2位小数
  if (unit === 'Mbps' || unitKey === 'unit.mbps') {
    if (value >= 1024) {
      return `${(value / 1024).toFixed(2)}Gbps`;
    }
    return `${formattedValue}Mbps`;
  }

  // 默认格式：数值+单位（无空格，参考76.36Mbps格式）
  return `${formattedValue}${unit}`;
}

/**
 * 生成Day视图的X轴时间标签（整点：0:00, 01:00, ..., 23:00）
 *
 * @returns 24小时整点时间标签数组
 *
 * @example
 * generateDayAxisLabels() // ["0:00", "1:00", "2:00", ..., "23:00"]
 */
export function generateDayAxisLabels(): string[] {
  const labels: string[] = [];
  for (let hour = 0; hour < 24; hour++) {
    labels.push(`${hour.toString().padStart(2, '0')}:00`);
  }
  return labels;
}

/**
 * 生成Day视图的完整X轴时间戳（用于tooltip显示）
 *
 * @param baseDate - 基准日期，默认为当前日期
 * @returns 24小时整点完整时间戳数组
 *
 * @example
 * generateDayAxisTimestamps(new Date('2024-06-04'))
 * // ["2024-06-04 00:00:00", "2024-06-04 01:00:00", ..., "2024-06-04 23:00:00"]
 */
export function generateDayAxisTimestamps(baseDate: Date = new Date()): string[] {
  const timestamps: string[] = [];
  const year = baseDate.getFullYear();
  const month = (baseDate.getMonth() + 1).toString().padStart(2, '0');
  const day = baseDate.getDate().toString().padStart(2, '0');

  for (let hour = 0; hour < 24; hour++) {
    const h = hour.toString().padStart(2, '0');
    timestamps.push(`${year}-${month}-${day} ${h}:00:00`);
  }
  return timestamps;
}
