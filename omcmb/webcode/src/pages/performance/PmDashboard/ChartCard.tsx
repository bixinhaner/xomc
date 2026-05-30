/**
 * T-0188 共享出图卡片：单指标一张 ECharts 折线（横轴时间、纵轴值，height~260）。
 *
 * 从 TaskDashboardPane 抽出，供页签1（任务仪表盘）+ 页签2（设备列表）复用同一渲染。
 * 渲染行为与抽取前保持一致：grid/series/connectNulls=false/showSymbol 阈值/legend 滚动 均不变。
 */

import { useMemo } from 'react';
import { Card } from 'antd';
import ReactECharts from 'echarts-for-react';
import dayjs from 'dayjs';
import type { MetricChart } from './taskDashboardUtils';

export default function ChartCard({ chart }: { chart: MetricChart }) {
  const xLabels = useMemo(
    () => chart.buckets.map((b) => (dayjs(b).isValid() ? dayjs(b).format('MM-DD HH:mm') : b)),
    [chart.buckets],
  );
  const option = {
    grid: { left: 56, right: 16, top: 36, bottom: 40 },
    xAxis: { type: 'category', data: xLabels, boundaryGap: false },
    yAxis: { type: 'value', scale: true },
    series: chart.series.map((s) => ({
      name: s.name,
      type: 'line',
      smooth: true,
      showSymbol: chart.buckets.length <= 30,
      data: s.values,
      connectNulls: false,
    })),
    tooltip: { trigger: 'axis' },
    legend: { type: 'scroll', top: 4 },
  };
  return (
    <Card size="small" title={chart.displayName} style={{ marginBottom: 12 }}>
      <ReactECharts option={option} style={{ height: 260 }} notMerge />
    </Card>
  );
}
