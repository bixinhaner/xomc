/**
 * T-0188 共享出图卡片：单指标一张 ECharts 折线（横轴时间、纵轴值，height~260）。
 *
 * 从 TaskDashboardPane 抽出，供页签1（任务仪表盘）+ 页签2（设备列表）复用同一渲染。
 *
 * #200 多设备折线断裂：多设备各自时钟/上报相位不同，同一 15min 窗口在并集轴上落成相邻
 * 但不同的桶，某设备系列在别设备的桶处取不到值 → 大量空洞 → connectNulls=false 时相邻点
 * 连不成线、整图贴底锯齿。后端已把 KPI 行 start_time 对齐到完整 15min 窗口（copy_ingest），
 * 前端再开 connectNulls=true 兜底：跨空洞桶连线，让稀疏多设备曲线连续（不改"真缺采样"语义——
 * 仅视觉连线，缺失桶仍无数据点 / tooltip 仍按桶显示）。
 */

import { useMemo } from 'react';
import { Card } from 'antd';
import ReactECharts from 'echarts-for-react';
import dayjs from 'dayjs';
import { useT } from '@/hooks/useT';
import type { MetricChart } from './taskDashboardUtils';
import { computeTooltipPosition } from '@/components/Charts/LineChart';

const fmtTime = (t: string) => (dayjs(t).isValid() ? dayjs(t).format('MM-DD HH:mm') : t);

/** ECharts axis-trigger tooltip 回调单项（只取本组件用到的字段）。 */
interface TooltipParam {
  dataIndex: number;
  seriesName?: string;
  marker?: string;
  value?: number | string;
}

export default function ChartCard({ chart }: { chart: MetricChart }) {
  const t = useT();
  const xLabels = useMemo(() => chart.buckets.map(fmtTime), [chart.buckets]);
  const currentSeries = chart.series.map((s) => ({
    name: s.name,
    type: 'line',
    smooth: true,
    showSymbol: chart.buckets.length <= 30,
    data: s.values,
    // issue #429：规整网格下断档槽位置 '-'/null，connectNulls=false 让缺口处线断开
    // （断档在网管有运维含义，不再连线抹平）；#200 的"假空洞"在规整网格下已不存在。
    connectNulls: false,
  }));
  // T-0189 周期对比：上一周期系列画虚线（已在上游按 +L 对齐到当前轴）。
  const compareSeries = (chart.compareSeries ?? []).map((s) => ({
    name: s.name,
    type: 'line',
    smooth: true,
    showSymbol: chart.buckets.length <= 30,
    data: s.values,
    // issue #429：对比虚线同当前系列，断档处断开线（与上文同口径）。
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
      ? `${t('pm.chart.tooltipStart')} ${fmtTime(start)}<br/>${t('pm.chart.tooltipEnd')} ${fmtTime(end)}`
      : fmtTime(start);
    // T-0194：周期对比开启时，补一行上一周期对应桶的真实「开始~结束」时间段（非照搬当前轴标签）。
    if (chart.compareSeries && chart.compareSeries.length > 0) {
      const pStart = chart.compareBuckets?.[idx] ?? '';
      const pEnd = chart.compareBucketEnds?.[idx] ?? '';
      if (pStart) {
        header += pEnd
          ? `<br/>${t('pm.chart.tooltipPrevPeriod')} ${fmtTime(pStart)} ~ ${fmtTime(pEnd)}`
          : `<br/>${t('pm.chart.tooltipPrevPeriod')} ${fmtTime(pStart)}`;
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
    // 智能避让：多设备 tooltip 行多、易盖住曲线/图例，按光标位置放到对角（issue #202）。
    tooltip: {
      trigger: 'axis',
      confine: true,
      position: (
        point: [number, number],
        _params: unknown,
        _dom: unknown,
        _rect: unknown,
        size: { contentSize: [number, number]; viewSize: [number, number] },
      ) => computeTooltipPosition(point, size),
      formatter: tooltipFormatter,
    },
    legend: { type: 'scroll', top: 4 },
  };
  return (
    <Card size="small" title={chart.displayName} style={{ marginBottom: 12 }}>
      <ReactECharts option={option} style={{ height: 260 }} notMerge />
    </Card>
  );
}
