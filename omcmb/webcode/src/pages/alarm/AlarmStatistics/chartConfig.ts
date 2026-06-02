import type { EChartsOption } from 'echarts';

/**
 * 告警趋势图的 ECharts 配置
 * @param xData X轴数据（日期数组）
 * @param series 系列数据（各严重程度的告警数量）
 * @param days 查询天数（影响柱状图宽度）
 * @param t 国际化函数
 * @returns ECharts 配置对象
 */
export function createAlarmTrendChartOption(
  xData: string[],
  series: Array<{ name: string; data: number[]; color: string }>,
  days: number,
  t: (key: string) => string
): EChartsOption {
  return {
    tooltip: {
      trigger: 'axis',
      confine: true,
      backgroundColor: 'rgba(0, 0, 0, 0.85)',
      borderColor: '#333',
      textStyle: { color: '#fff', fontSize: 12 },
      formatter: (params: unknown) => {
        const items = params as Array<{
          marker: string;
          seriesName: string;
          value: number;
          axisValue: string;
          color: string;
        }>;
        if (!Array.isArray(items) || items.length === 0) return '';
        const total = items.reduce((sum, item) => sum + item.value, 0);
        return `<div style="line-height: 1.8; padding: 6px;">
          <div style="font-weight: 600; margin-bottom: 8px; font-size: 13px; border-bottom: 1px solid #444; padding-bottom: 6px;">
            ${items[0].axisValue}
          </div>
          <div style="margin-bottom: 6px;">
            <span style="color: #bbb;">总计:</span>
            <span style="color: #fff; font-weight: 600; font-size: 14px; margin-left: 8px;">${total}</span>
          </div>
          ${items.map(item =>
            `<div style="margin: 2px 0;">
              ${item.marker} <span style="color: ${item.color};">${item.seriesName}</span>
              <span style="color: #fff; float: right; font-weight: 600;">${item.value}</span>
            </div>`
          ).join('')}
        </div>`;
      },
    },
    legend: {
      show: true,
      top: 4,
      left: 'center',
      itemWidth: 16,
      itemHeight: 10,
      itemGap: 24,
      textStyle: { fontSize: 12, color: '#595959' },
      data: [
        { name: t('alarm.severity.critical'), icon: 'rect' },
        { name: t('alarm.severity.major'), icon: 'rect' },
        { name: t('alarm.severity.minor'), icon: 'rect' },
        { name: t('alarm.severity.warning'), icon: 'rect' },
      ],
    },
    grid: {
      top: 48,
      left: 45,
      right: 20,
      bottom: 32,
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: xData,
      boundaryGap: true,
      axisLabel: {
        fontSize: 11,
        color: '#8c8c8c',
      },
      axisLine: { lineStyle: { color: '#e8e8e8' } },
      axisTick: { alignWithLabel: true, show: true },
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      axisLabel: {
        fontSize: 11,
        color: '#8c8c8c',
        formatter: (value: number) => Number.isInteger(value) ? value : '',
      },
      splitLine: {
        lineStyle: { type: 'dashed', color: '#f0f0f0' },
      },
    },
    series: series.map((s) => ({
      name: s.name,
      type: 'bar',
      data: s.data,
      stack: 'alarm',
      barWidth: days > 15 ? '60%' : '40%',
      itemStyle: {
        color: s.color,
        borderRadius: [2, 2, 0, 0],
      },
      emphasis: {
        focus: 'series',
        itemStyle: {
          shadowBlur: 10,
          shadowColor: 'rgba(0, 0, 0, 0.2)',
        },
      },
    })),
  };
}

/**
 * 告警分布饼图的 ECharts 配置
 * TODO: 待接入饼图组件（告警类型分布）
 * @param data 饼图数据
 * @param t 国际化函数
 * @returns ECharts 配置对象
 */
export function createAlarmPieChartOption(
  data: Array<{ name: string; value: number; color?: string }>,
  t: (key: string) => string
): EChartsOption {
  return {
    tooltip: {
      trigger: 'item',
      confine: true,
      formatter: '{b}: {c} ({d}%)',
    },
    legend: {
      show: true,
      top: 8,
      left: 'center',
      itemWidth: 12,
      itemHeight: 12,
      itemGap: 16,
      textStyle: { fontSize: 12, color: '#595959' },
      data: data.map(item => item.name),
    },
    series: [
      {
        name: t('alarm.stats.distribution'),
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['50%', '55%'],
        avoidLabelOverlap: true,
        itemStyle: {
          borderRadius: 4,
          borderColor: '#fff',
          borderWidth: 2,
        },
        label: {
          show: false,
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 14,
            fontWeight: 'bold',
          },
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.3)',
          },
        },
        labelLine: {
          show: false,
        },
        data: data.map(item => ({
          ...item,
          itemStyle: { color: item.color },
        })),
      },
    ],
  };
}
