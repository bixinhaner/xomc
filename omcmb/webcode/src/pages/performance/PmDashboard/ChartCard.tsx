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

import { memo, useMemo } from 'react';
import { Card } from 'antd';
import ReactECharts from 'echarts-for-react';
import { useT } from '@/hooks/useT';
import { formatSystemTime } from '@core/utils/systemTime';
import type { MetricChart, MetricSeries } from './taskDashboardUtils';

// #459 子单 D：图表横轴时间标签按系统时区显示（与列表一致），不按浏览器本地转换。
const fmtTime = (t: string) => formatSystemTime(t, { format: 'MM-DD HH:mm', placeholder: t });

/** ECharts axis-trigger tooltip 回调单项（只取本组件用到的字段）。 */
interface TooltipParam {
  dataIndex: number;
  seriesName?: string;
  marker?: string;
  value?: number | string;
}

function computeScrollableTooltipPosition(
  point: [number, number],
  size: { contentSize: [number, number]; viewSize: [number, number] },
): [number, number] {
  const [pointerX, pointerY] = point;
  const [boxW, boxH] = size.contentSize;
  const [viewW, viewH] = size.viewSize;
  const margin = 12;
  const offset = 12;
  const x = pointerX + boxW + offset <= viewW - margin
    ? pointerX + offset
    : pointerX - boxW - offset;
  const y = pointerY + boxH + offset <= viewH - margin
    ? pointerY + offset
    : pointerY - boxH - offset;

  return [
    Math.max(margin, Math.min(x, Math.max(margin, viewW - boxW - margin))),
    Math.max(margin, Math.min(y, Math.max(margin, viewH - boxH - margin))),
  ];
}

// #444 渲染隔离：包 React.memo + **内容级** areEqual 比较器（chartContentEqual）。
// 关键背景：勾选小时段/星期只 setFilter（不动 submitted）；但 DeviceListPane 的 charts
// useMemo 依赖 rawRows，而上游 useAggregatedMetricsByDevices 每次 render 都 flatMap 出
// **新数组引用**（内容不变），导致 charts 重算、chart prop **引用每次都变**——纯引用浅比较
// 的 React.memo 因此放行重渲染，<ReactECharts notMerge /> 把数据没变的图全量重绘 → 卡顿
// （运行栈实测：chartPropRefChanged=true 但首系列内容未变）。
// 故这里改用内容级比较：chart 的可视字段（buckets/series 等）逐项相等就跳过整卡重渲染，
// 哪怕对象引用变了。点「出图」时 submitted 变 → 数据真变 → 内容不等 → 正常重画一次。
const seriesEqual = (a?: MetricSeries[], b?: MetricSeries[]): boolean => {
  if (a === b) return true;
  if (!a || !b || a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    const x = a[i];
    const y = b[i];
    if (x.key !== y.key || x.name !== y.name || x.values.length !== y.values.length) return false;
    for (let j = 0; j < x.values.length; j++) {
      if (x.values[j] !== y.values[j]) return false;
    }
  }
  return true;
};

const strArrEqual = (a?: string[], b?: string[]): boolean => {
  if (a === b) return true;
  if (!a || !b || a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
};

/** 内容级相等：chart 引用变但可视内容（横轴桶 + 系列值 + 对比系列）全相同 → 视为无需重绘。 */
function chartContentEqual(prev: { chart: MetricChart }, next: { chart: MetricChart }): boolean {
  const a = prev.chart;
  const b = next.chart;
  if (a === b) return true;
  return (
    a.metricPath === b.metricPath &&
    a.displayName === b.displayName &&
    strArrEqual(a.buckets, b.buckets) &&
    strArrEqual(a.bucketEnds, b.bucketEnds) &&
    seriesEqual(a.series, b.series) &&
    seriesEqual(a.compareSeries, b.compareSeries) &&
    strArrEqual(a.compareBuckets, b.compareBuckets) &&
    strArrEqual(a.compareBucketEnds, b.compareBucketEnds)
  );
}

function ChartCard({ chart }: { chart: MetricChart }) {
  const t = useT();
  const xLabels = useMemo(() => chart.buckets.map(fmtTime), [chart.buckets]);
  const currentSeries = useMemo(
    () =>
      chart.series.map((s) => ({
        name: s.name,
        type: 'line',
        smooth: true,
        // issue #514：点标记始终可见（原 buckets.length<=30 条件下，多桶时隐藏点标记，
        // 单个/极稀疏孤立点连不成线段仍会隐身）。改为始终画点，避免误判"暂无数据"。
        showSymbol: true,
        symbolSize: 4,
        data: s.values,
        // issue #429：规整网格下断档槽位置 '-'/null，connectNulls=false 让缺口处线断开
        // （断档在网管有运维含义，不再连线抹平）；#200 的"假空洞"在规整网格下已不存在。
        connectNulls: false,
      })),
    [chart.series],
  );
  // T-0189 周期对比：上一周期系列画虚线（已在上游按 +L 对齐到当前轴）。
  const compareSeries = useMemo(
    () =>
      (chart.compareSeries ?? []).map((s) => ({
        name: s.name,
        type: 'line',
        smooth: true,
        // issue #514：点标记始终可见（同当前系列口径）。
        showSymbol: true,
        symbolSize: 4,
        data: s.values,
        // issue #429：对比虚线同当前系列，断档处断开线（与上文同口径）。
        connectNulls: false,
        lineStyle: { type: 'dashed' as const },
      })),
    [chart.compareSeries],
  );
  const option = useMemo(() => {
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
    return {
      grid: { left: 56, right: 16, top: 36, bottom: 40 },
      xAxis: { type: 'category', data: xLabels, boundaryGap: false },
      yAxis: { type: 'value', scale: true },
      series: [...currentSeries, ...compareSeries],
      // 多对象 tooltip 需要可进入后滚动：贴近光标显示，边界不足时自动换侧。
      // 若放到远端对角，用户移动鼠标去 tooltip 的途中会不断刷新 axis tooltip，实际无法滚动。
      tooltip: {
        trigger: 'axis',
        confine: true,
        renderMode: 'html',
        // 允许鼠标进入 tooltip 后滚动；否则 ECharts 默认会忽略其鼠标事件，滚动条不可操作。
        enterable: true,
        // 多设备时允许 tooltip 自身滚动，避免超出图表后底部设备被裁掉（#24）。
        extraCssText: 'max-height:220px;overflow-y:auto;overflow-x:hidden;',
        position: (
          point: [number, number],
          _params: unknown,
          _dom: unknown,
          _rect: unknown,
          size: { contentSize: [number, number]; viewSize: [number, number] },
        ) => computeScrollableTooltipPosition(point, size),
        formatter: tooltipFormatter,
      },
      legend: { type: 'scroll', top: 4 },
    };
  }, [chart, xLabels, currentSeries, compareSeries, t]);
  return (
    <Card size="small" title={chart.displayName} style={{ marginBottom: 12 }}>
      <ReactECharts option={option} style={{ height: 260 }} notMerge />
    </Card>
  );
}

export default memo(ChartCard, chartContentEqual);
