/**
 * Y轴智能刻度计算结果
 */
export interface SmartTicksResult {
  /** 刻度间隔 */
  interval: number;
  /** Y轴最大值 */
  max: number;
}

/**
 * 智能计算Y轴刻度间隔和最大值
 * @param seriesData 所有系列的数据数组（二维数组）
 * @param targetTickCount 目标刻度数量，默认5个
 * @returns 刻度间隔和最大值
 */
export function calculateSmartTicks(
  seriesData: number[][],
  targetTickCount = 5
): SmartTicksResult {
  // 找出所有数据中的最大值
  let maxVal = 0;
  seriesData.forEach((series) => {
    series.forEach((v) => {
      if (typeof v === 'number' && v > maxVal) maxVal = v;
    });
  });

  if (maxVal === 0) return { interval: 1, max: 10 };

  // 计算原始间隔
  const rawInterval = maxVal / targetTickCount;

  // 计算数量级
  const magnitude = Math.pow(10, Math.floor(Math.log10(rawInterval)));

  // 标准化间隔候选：1、2、5、10的倍数
  const normalizedInterval = rawInterval / magnitude;

  let interval: number;
  if (normalizedInterval <= 1.5) {
    interval = 1 * magnitude;
  } else if (normalizedInterval <= 3.5) {
    interval = 2 * magnitude;
  } else if (normalizedInterval <= 7.5) {
    interval = 5 * magnitude;
  } else {
    interval = 10 * magnitude;
  }

  // 将最大值向上取整到间隔的倍数
  const yMax = Math.ceil(maxVal / interval) * interval;

  return { interval, max: yMax };
}

/**
 * 从图表系列配置中提取数据数组
 * @param series 图表系列配置
 * @returns 二维数据数组
 */
export function extractSeriesData(series: Array<{ data: (number | null)[] }>): number[][] {
  return series.map((s) => s.data.filter((v): v is number => v !== null));
}
