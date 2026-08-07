/**
 * T-0188 共享出图卡片：单指标一张 ECharts 折线（横轴时间、纵轴值，height~260）。
 *
 * 从 TaskDashboardPane 抽出，供页签1（任务仪表盘）+ 页签2（设备列表）复用同一渲染。
 *
 * #200 多设备折线断裂：多设备各自时钟/上报相位不同，同一 15min 窗口在并集轴上落成相邻
 * 但不同的桶，某设备系列在别设备的桶处取不到值 → 大量空洞。与首页 KPI 趋势统一为
 * connectNulls=true：跨空桶连接相邻有效点；缺失桶仍无数据点，tooltip 仍按桶显示。
 */

import { memo, useEffect, useMemo, useState } from 'react';
import { Button, Card, Typography } from 'antd';
import { CloseOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import { useT } from '@/hooks/useT';
import { formatSystemTime } from '@core/utils/systemTime';
import { formatPmMetricDisplayValue, formatPmMetricDisplayValueWithUnit } from '@core/utils/pmMetricValue';
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

interface ChartClickParam {
  dataIndex?: number;
}

const MANY_OBJECT_THRESHOLD = 12;
const TOOLTIP_WIDTH = 520;
const TOOLTIP_TEXT_COLOR = '#1f2937';
const TOOLTIP_MUTED_COLOR = '#344054';
const TOOLTIP_STRONG_COLOR = '#111827';
const TOOLTIP_BORDER_COLOR = '#d9d9d9';
const TOOLTIP_DIVIDER_COLOR = '#f0f0f0';
const TOOLTIP_ROW_DIVIDER_COLOR = '#fafafa';
const TOOLTIP_FONT_FAMILY = '-apple-system, system-ui, Segoe UI, Roboto, Helvetica Neue, Arial, Noto Sans, sans-serif';
const TOOLTIP_MONO_FONT_FAMILY = 'SFMono-Regular, Consolas, Liberation Mono, Menlo, monospace';
const TOOLTIP_SHADOW = '0 8px 24px rgba(0,0,0,0.16)';

function escapeTooltipHtml(value: number | string | undefined): string {
  return String(value ?? '-')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function formatMetricValueWithUnit(
  value: number | string | null | undefined,
  unit: string | undefined,
): string {
  return formatPmMetricDisplayValueWithUnit(value, unit);
}

function formatTooltipRowHtml(param: TooltipParam, unit: string | undefined): string {
  return [
    `<div style="display:grid;grid-template-columns:minmax(0,1fr) auto;column-gap:12px;align-items:start;padding:6px 0;border-bottom:1px solid ${TOOLTIP_ROW_DIVIDER_COLOR};">`,
    `<span style="min-width:0;font-size:13px;line-height:18px;color:${TOOLTIP_TEXT_COLOR};font-family:${TOOLTIP_MONO_FONT_FAMILY};overflow-wrap:anywhere;word-break:break-word;">`,
    param.marker ?? '',
    escapeTooltipHtml(param.seriesName),
    '</span>',
    `<strong style="font-size:13px;line-height:18px;color:${TOOLTIP_STRONG_COLOR};font-family:${TOOLTIP_FONT_FAMILY};">`,
    escapeTooltipHtml(formatMetricValueWithUnit(param.value, unit)),
    '</strong>',
    '</div>',
  ].join('');
}

function formatTooltipHeader(
  chart: MetricChart,
  index: number,
  t: ReturnType<typeof useT>,
): string {
  const start = chart.buckets[index] ?? '';
  const end = chart.bucketEnds[index] ?? '';
  let header = end
    ? `${t('pm.chart.tooltipStart')} ${fmtTime(start)}<br/>${t('pm.chart.tooltipEnd')} ${fmtTime(end)}`
    : fmtTime(start);

  // T-0194：周期对比开启时，补一行上一周期对应桶的真实「开始~结束」时间段（非照搬当前轴标签）。
  if (chart.compareSeries && chart.compareSeries.length > 0) {
    const pStart = chart.compareBuckets?.[index] ?? '';
    const pEnd = chart.compareBucketEnds?.[index] ?? '';
    if (pStart) {
      header += pEnd
        ? `<br/>${t('pm.chart.tooltipPrevPeriod')} ${fmtTime(pStart)} ~ ${fmtTime(pEnd)}`
        : `<br/>${t('pm.chart.tooltipPrevPeriod')} ${fmtTime(pStart)}`;
    }
  }

  return header;
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
    a.unit === b.unit &&
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
  const [lockedIndex, setLockedIndex] = useState<number | null>(null);
  const effectiveLockedIndex = lockedIndex != null && lockedIndex < chart.buckets.length ? lockedIndex : null;
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
        // 与首页 KPI 趋势一致：空桶不补值，但连接前后已有数据点。
        connectNulls: true,
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
        // 对比虚线同当前系列，空桶保留但跨空桶续连。
        connectNulls: true,
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
      const header = formatTooltipHeader(chart, idx, t);
      const lines = arr.map((param) => formatTooltipRowHtml(param, chart.unit)).join('');
      const pinHint = arr.length > MANY_OBJECT_THRESHOLD
        ? `<div style="padding:6px 0;border-bottom:1px solid ${TOOLTIP_DIVIDER_COLOR};color:#667085;font-size:12px;line-height:18px;">${t('pm.chart.tooltipPinHint')}</div>`
        : '';
      return [
        `<div style="font-size:13px;line-height:20px;color:${TOOLTIP_MUTED_COLOR};padding-bottom:6px;border-bottom:1px solid ${TOOLTIP_DIVIDER_COLOR};">`,
        header,
        '</div>',
        pinHint,
        `<div style="padding-top:6px;">${lines}</div>`,
      ].join('');
    };
    return {
      grid: { left: 56, right: 16, top: 36, bottom: 40 },
      xAxis: { type: 'category', data: xLabels, boundaryGap: false },
      yAxis: {
        type: 'value',
        scale: true,
        axisLabel: {
          formatter: (value: number) => formatPmMetricDisplayValue(value),
        },
      },
      series: [...currentSeries, ...compareSeries],
      // hover tooltip 只承担快速预览和提示；对象很多时，用户点击数据点后打开固定浮层再滚动查看。
      tooltip: {
        trigger: 'axis',
        confine: true,
        renderMode: 'html',
        enterable: true,
        extraCssText: [
          `width:${TOOLTIP_WIDTH}px`,
          'max-width:calc(100% - 32px)',
          'max-height:220px',
          'overflow-y:auto',
          'overflow-x:hidden',
          'white-space:normal',
          'background:#fff',
          `color:${TOOLTIP_STRONG_COLOR}`,
          `border:1px solid ${TOOLTIP_BORDER_COLOR}`,
          'border-radius:6px',
          `box-shadow:${TOOLTIP_SHADOW}`,
          'padding:10px',
        ].join(';'),
        formatter: tooltipFormatter,
      },
      legend: { type: 'scroll', top: 4 },
    };
  }, [chart, xLabels, currentSeries, compareSeries, t]);

  const onEvents = useMemo(
    () => ({
      click: (params: ChartClickParam) => {
        if (typeof params.dataIndex === 'number') {
          setLockedIndex(params.dataIndex);
        }
      },
    }),
    [],
  );

  useEffect(() => {
    if (effectiveLockedIndex == null) return undefined;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setLockedIndex(null);
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [effectiveLockedIndex]);

  const lockedRows = useMemo(() => {
    if (effectiveLockedIndex == null) return [];
    const current = chart.series.map((s) => ({
      key: s.key,
      name: s.name,
      value: s.values[effectiveLockedIndex] ?? '-',
      displayValue: formatMetricValueWithUnit(s.values[effectiveLockedIndex] ?? '-', chart.unit),
      dashed: false,
    }));
    const previous = (chart.compareSeries ?? []).map((s) => ({
      key: `compare-${s.key}`,
      name: s.name,
      value: s.values[effectiveLockedIndex] ?? '-',
      displayValue: formatMetricValueWithUnit(s.values[effectiveLockedIndex] ?? '-', chart.unit),
      dashed: true,
    }));
    return [...current, ...previous];
  }, [chart.series, chart.compareSeries, chart.unit, effectiveLockedIndex]);

  return (
    <Card size="small" title={chart.displayName} style={{ marginBottom: 12 }}>
      <div style={{ position: 'relative' }}>
        <ReactECharts option={option} style={{ height: 260, cursor: 'pointer' }} notMerge onEvents={onEvents} />
        {effectiveLockedIndex != null ? (
          <div
            role="dialog"
            aria-label={t('pm.chart.tooltipPinnedAria')}
            style={{
              position: 'absolute',
              top: 40,
              right: 16,
              zIndex: 5,
              width: TOOLTIP_WIDTH,
              maxWidth: 'calc(100% - 32px)',
              background: '#fff',
              color: TOOLTIP_STRONG_COLOR,
              border: `1px solid ${TOOLTIP_BORDER_COLOR}`,
              borderRadius: 6,
              boxShadow: TOOLTIP_SHADOW,
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: 12,
                padding: '8px 10px',
                borderBottom: `1px solid ${TOOLTIP_DIVIDER_COLOR}`,
              }}
            >
              <Typography.Text strong style={{ fontSize: 14, color: TOOLTIP_STRONG_COLOR }}>
                {t('pm.chart.tooltipPinnedTitle')}
              </Typography.Text>
              <Button
                aria-label={t('pm.chart.tooltipPinnedClose')}
                type="text"
                size="small"
                icon={<CloseOutlined />}
                style={{ color: TOOLTIP_MUTED_COLOR }}
                onClick={() => setLockedIndex(null)}
              />
            </div>
            <div
              style={{
                padding: '8px 10px',
                fontSize: 13,
                lineHeight: '20px',
                color: TOOLTIP_MUTED_COLOR,
                borderBottom: `1px solid ${TOOLTIP_DIVIDER_COLOR}`,
              }}
            >
              <span
                dangerouslySetInnerHTML={{
                  __html: formatTooltipHeader(chart, effectiveLockedIndex, t),
                }}
              />
            </div>
            <div style={{ maxHeight: 260, overflowY: 'auto', overflowX: 'hidden', padding: '6px 10px' }}>
              {lockedRows.map((row) => (
                <div
                  key={row.key}
                  style={{
                    display: 'grid',
                    gridTemplateColumns: 'minmax(0, 1fr) auto',
                    columnGap: 12,
                    alignItems: 'start',
                    padding: '6px 0',
                    borderBottom: `1px solid ${TOOLTIP_ROW_DIVIDER_COLOR}`,
                  }}
                >
                  <Typography.Text
                    style={{
                      fontSize: 13,
                      lineHeight: '18px',
                      color: TOOLTIP_TEXT_COLOR,
                      fontFamily: TOOLTIP_MONO_FONT_FAMILY,
                      overflowWrap: 'anywhere',
                      wordBreak: 'break-word',
                      fontStyle: row.dashed ? 'italic' : undefined,
                    }}
                  >
                    {row.name}
                  </Typography.Text>
                  <Typography.Text strong style={{ fontSize: 13, lineHeight: '18px', color: TOOLTIP_STRONG_COLOR }}>
                    {row.displayValue}
                  </Typography.Text>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </div>
    </Card>
  );
}

export default memo(ChartCard, chartContentEqual);
