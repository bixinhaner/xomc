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

const fmtTime = (t: string) => (dayjs(t).isValid() ? dayjs(t).format('MM-DD HH:mm') : t);

/** ECharts axis-trigger tooltip 回调单项（只取本组件用到的字段）。 */
interface TooltipParam {
  dataIndex: number;
  seriesName?: string;
  marker?: string;
  value?: number | string;
}

export default function ChartCard({ chart }: { chart: MetricChart }) {
  const xLabels = useMemo(() => chart.buckets.map(fmtTime), [chart.buckets]);
  const currentSeries = chart.series.map((s) => ({
    name: s.name,
    type: 'line',
    smooth: true,
    showSymbol: chart.buckets.length <= 30,
    data: s.values,
    connectNulls: false,
  }));
  // T-0189 周期对比：上一周期系列画虚线（已在上游按 +L 对齐到当前轴）。
  const compareSeries = (chart.compareSeries ?? []).map((s) => ({
    name: s.name,
    type: 'line',
    smooth: true,
    showSymbol: chart.buckets.length <= 30,
    data: s.values,
    connectNulls: false,
    lineStyle: { type: 'dashed' as const },
  }));
  // tooltip 表头显示该桶的「开始~结束」时间段（每个点代表一个时间桶，非单时间点）。
  // 数据点起止时间来自后端返回的 startTime/endTime（已转置进 chart.buckets / chart.bucketEnds）。
  const tooltipFormatter = (params: TooltipParam | TooltipParam[]) => {
    const arr = Array.isArray(params) ? params : [params];
    if (arr.length === 0) return '';
    const idx = arr[0].dataIndex;
    const start = chart.buckets[idx] ?? '';
    const end = chart.bucketEnds[idx] ?? '';
    let header = end
      ? `开始 ${fmtTime(start)}<br/>结束 ${fmtTime(end)}`
      : fmtTime(start);
    // T-0194：周期对比开启时，补一行上一周期对应桶的真实「开始~结束」时间段（非照搬当前轴标签）。
    if (chart.compareSeries && chart.compareSeries.length > 0) {
      const pStart = chart.compareBuckets?.[idx] ?? '';
      const pEnd = chart.compareBucketEnds?.[idx] ?? '';
      if (pStart) {
        header += pEnd
          ? `<br/>上一周期 ${fmtTime(pStart)} ~ ${fmtTime(pEnd)}`
          : `<br/>上一周期 ${fmtTime(pStart)}`;
      }
    }
    const lines = arr
      .map((p) => `${p.marker ?? ''}${p.seriesName ?? ''}: ${p.value ?? '-'}`)
      .join('<br/>');
    return `${header}<hr style="margin:4px 0;border:none;border-top:1px solid #eee"/>${lines}`;
  };
  const option = {
    grid: { left: 56, right: 16, top: 36, bottom: 40 },
    xAxis: { type: 'category', data: xLabels, boundaryGap: false },
    yAxis: { type: 'value', scale: true },
    series: [...currentSeries, ...compareSeries],
    tooltip: { trigger: 'axis', formatter: tooltipFormatter },
    legend: { type: 'scroll', top: 4 },
  };
  return (
    <Card size="small" title={chart.displayName} style={{ marginBottom: 12 }}>
      <ReactECharts option={option} style={{ height: 260 }} notMerge />
    </Card>
  );
}
