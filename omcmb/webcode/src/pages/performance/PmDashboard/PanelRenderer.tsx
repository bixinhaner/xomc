/**
 * 按 panel.panelType 路由到具体图表组件。
 *
 * v1：7 种 PanelType 全部走 ECharts 或 antd Statistic；数据由 usePmPanelData hook 提供
 * （hook 内部是 deterministic mock，后续 v2 替换为真 API 调用，组件无需改）。
 *
 * T-0164 收尾：
 *   G6-Gap-5  加 topn + big_number 两类型
 *   G6-Gap-6  接 activeGranularity prop（PanelGrid Tab 切换驱动）
 *   G6-Gap-7  compareMode 非空时拉双 series（mock 内置 previous_window / same_window_other_devices）
 *   G6-Gap-9  pct 单位 / 阈值线 / 缺采断线 / 时间轴切换
 */

import { Statistic, Table, Empty } from 'antd';
import ReactECharts from 'echarts-for-react';
import type { Granularity, Panel } from '@core/types/pmDashboard';
import { usePmPanelData, type PanelSeries } from '@core/hooks/api/usePmPanelData';

interface Props {
  panel: Panel;
  // G6-Gap-6: 当 panel.granularities.length > 1 时，由 PanelGrid 的 Tab 切换决定当前粒度。
  // 不传则用 panel.granularities[0]，再不行兜底 'hourly'。
  activeGranularity?: Granularity;
}

export function PanelRenderer({ panel, activeGranularity }: Props) {
  const gran: Granularity = activeGranularity ?? panel.granularities?.[0] ?? 'hourly';
  switch (panel.panelType) {
    case 'kpi_card':
      return <KpiCardRenderer panel={panel} gran={gran} />;
    case 'line_chart':
      return <LineChartRenderer panel={panel} gran={gran} />;
    case 'bar_chart':
      return <BarChartRenderer panel={panel} gran={gran} />;
    case 'table':
      return <TableRenderer panel={panel} gran={gran} />;
    case 'gauge':
      return <GaugeRenderer panel={panel} gran={gran} />;
    case 'topn':
      return <TopNRenderer panel={panel} gran={gran} />;
    case 'big_number':
      return <BigNumberRenderer panel={panel} gran={gran} />;
    default:
      return <Empty description={`未支持的 panel type: ${panel.panelType}`} />;
  }
}

interface RenderProps {
  panel: Panel;
  gran: Granularity;
}

// G6-Gap-9: 自动判断指标是否为百分比类（path 含 .Rate / .SuccRate / Pct / .Ratio 等关键词）。
function inferPctUnit(metricPath: string, configUnit?: string): string {
  if (configUnit) return configUnit;
  if (/(\.|^)(Rate|SuccRate|FailRate|DropRate|Pct|Percentage|Ratio)(\.|$)/i.test(metricPath)) {
    return '%';
  }
  return '';
}

// G6-Gap-9: markLine 阈值线（panel.config.threshold 已有字段；非空时画水平线）。
function buildThresholdMarkLine(threshold: number | undefined) {
  if (threshold === undefined || threshold === null) return undefined;
  return {
    silent: true,
    symbol: 'none',
    lineStyle: { color: '#ff7a45', type: 'dashed', width: 1.5 },
    data: [{ yAxis: threshold, label: { formatter: `阈值 ${threshold}` } }],
  };
}

// G6-Gap-9 (c) 缺采：把 null 转换为 echarts 的 '-' 占位符（断线）。
function toEchartsData(series: PanelSeries): Array<number | '-'> {
  return series.points.map((p) => (p.value === null ? '-' : p.value));
}

function bucketLabels(series: PanelSeries[]): string[] {
  return series[0]?.points.map((p) => p.label) ?? [];
}

function KpiCardRenderer({ panel, gran }: RenderProps) {
  const { series } = usePmPanelData(panel, gran);
  const path0 = panel.metricPaths[0] ?? '';
  const unit = inferPctUnit(path0, panel.config?.unit as string | undefined);
  const threshold = panel.config?.threshold as number | undefined;
  // 取主 series 最后一个非缺采点
  const primary = series.find((s) => s.kind === 'primary');
  const last = primary?.points.slice().reverse().find((p) => p.value !== null);
  const value = (last?.value as number | undefined) ?? 0;
  const overThreshold = threshold !== undefined && value < threshold;

  // G6-Gap-7：对比 series 末值，作为副指标显示在卡片下方
  const compare = series.find((s) => s.kind === 'compare');
  const compareLast = compare?.points.slice().reverse().find((p) => p.value !== null);
  const compareValue = compareLast?.value as number | undefined;
  const delta = compareValue !== undefined ? Math.round((value - compareValue) * 100) / 100 : undefined;

  return (
    <div style={{ position: 'relative' }}>
      <Statistic
        title={panel.metricPaths.join(' / ')}
        value={value}
        suffix={unit}
        precision={2}
        valueStyle={overThreshold ? { color: '#ff4d4f' } : undefined}
      />
      {delta !== undefined && compare && (
        <div style={{ marginTop: 4, fontSize: 12, color: delta >= 0 ? '#3f8600' : '#cf1322' }}>
          {compare.name}: {compareValue}
          {unit} （Δ {delta >= 0 ? '+' : ''}
          {delta}）
        </div>
      )}
    </div>
  );
}

function LineChartRenderer({ panel, gran }: RenderProps) {
  const { series } = usePmPanelData(panel, gran);
  const threshold = panel.config?.threshold as number | undefined;
  const timeAxis = (panel.config?.time_axis as string) ?? 'end_time';
  const markLine = buildThresholdMarkLine(threshold);
  const buckets = bucketLabels(series);
  const option = {
    grid: { left: 40, right: 16, top: 24, bottom: 40 },
    xAxis: {
      type: 'category',
      data: buckets,
      name: timeAxis === 'ingest_time' ? '入库时间' : '采集时间',
      nameLocation: 'middle',
      nameGap: 24,
    },
    yAxis: { type: 'value' },
    series: series.map((s, i) => ({
      name: s.name,
      type: 'line',
      smooth: true,
      // G6-Gap-7 对比 series 用虚线区分
      lineStyle: s.kind === 'compare' ? { type: 'dashed' } : undefined,
      data: toEchartsData(s),
      connectNulls: false,
      markLine: i === 0 ? markLine : undefined,
    })),
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
  };
  return <ReactECharts option={option} style={{ height: 200 }} />;
}

function BarChartRenderer({ panel, gran }: RenderProps) {
  const { series } = usePmPanelData(panel, gran);
  const threshold = panel.config?.threshold as number | undefined;
  const markLine = buildThresholdMarkLine(threshold);
  const buckets = bucketLabels(series);
  const option = {
    grid: { left: 40, right: 16, top: 24, bottom: 32 },
    xAxis: { type: 'category', data: buckets },
    yAxis: { type: 'value' },
    series: series.map((s, i) => ({
      name: s.name,
      type: 'bar',
      // G6-Gap-7 对比 series 浅色 + 半透明 stack 偏移视觉
      itemStyle: s.kind === 'compare' ? { opacity: 0.55 } : undefined,
      data: toEchartsData(s),
      markLine: i === 0 ? markLine : undefined,
    })),
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
  };
  return <ReactECharts option={option} style={{ height: 200 }} />;
}

function TableRenderer({ panel, gran }: RenderProps) {
  const { series } = usePmPanelData(panel, gran);
  const buckets = bucketLabels(series);
  // 每行 = 一个 bucket；列 = metric × {primary | compare}
  const rows = buckets.map((label, i) => {
    const row: Record<string, string | number | null> = { key: String(i), name: label };
    series.forEach((s) => {
      row[s.name] = s.points[i]?.value ?? null;
    });
    return row;
  });
  return (
    <Table
      size="small"
      pagination={false}
      columns={[
        { title: '时间', dataIndex: 'name', width: 100 },
        ...series.map((s) => ({
          title: s.name,
          dataIndex: s.name,
          render: (v: number | null) => (v === null ? <span style={{ color: '#bfbfbf' }}>缺采</span> : v),
        })),
      ]}
      dataSource={rows}
    />
  );
}

function GaugeRenderer({ panel, gran }: RenderProps) {
  const { series } = usePmPanelData(panel, gran);
  const primary = series.find((s) => s.kind === 'primary');
  const last = primary?.points.slice().reverse().find((p) => p.value !== null);
  const value = (last?.value as number | undefined) ?? 0;
  const option = {
    series: [
      {
        type: 'gauge',
        progress: { show: true, width: 14 },
        axisLine: { lineStyle: { width: 14 } },
        detail: { formatter: '{value}%', fontSize: 18 },
        data: [{ value, name: panel.metricPaths[0] ?? '' }],
        min: 0,
        max: 100,
      },
    ],
  };
  return <ReactECharts option={option} style={{ height: 200 }} />;
}

// G6-Gap-5: TopN 排行榜
// config.n (number, 默认 10)；config.sort_desc (bool, 默认 true 表示降序)
function TopNRenderer({ panel, gran }: RenderProps) {
  void gran;
  const n = (panel.config?.n as number) ?? 10;
  const sortDesc = (panel.config?.sort_desc as boolean) ?? true;
  // mock 数据：N 条 (device, value) 行
  const mock = Array.from({ length: n }, (_, i) => ({
    key: String(i + 1),
    rank: i + 1,
    device: `SN-${String(i + 1).padStart(3, '0')}`,
    value: Math.round((100 - i * 4 + Math.random() * 2) * 100) / 100,
  }));
  const sorted = sortDesc ? mock : mock.slice().reverse();
  return (
    <Table
      size="small"
      pagination={false}
      columns={[
        { title: '#', dataIndex: 'rank', width: 50 },
        { title: '设备', dataIndex: 'device' },
        { title: panel.metricPaths[0] ?? '值', dataIndex: 'value' },
      ]}
      dataSource={sorted}
    />
  );
}

// G6-Gap-5: BigNumber 数值大屏
// config.font_size (number, 默认 56)；config.warning_threshold (number, 低于此值红色)
function BigNumberRenderer({ panel, gran }: RenderProps) {
  const { series } = usePmPanelData(panel, gran);
  const fontSize = (panel.config?.font_size as number) ?? 56;
  const warningThreshold = panel.config?.warning_threshold as number | undefined;
  const unit = (panel.config?.unit as string) ?? '';
  const primary = series.find((s) => s.kind === 'primary');
  const last = primary?.points.slice().reverse().find((p) => p.value !== null);
  const value = (last?.value as number | undefined) ?? 0;
  const color =
    warningThreshold !== undefined && value < warningThreshold ? '#ff4d4f' : '#1677ff';
  // G6-Gap-7：对比 series 末值显示在大数字下方
  const compare = series.find((s) => s.kind === 'compare');
  const compareLast = compare?.points.slice().reverse().find((p) => p.value !== null);
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', padding: 16 }}>
      <div style={{ fontSize, lineHeight: 1.2, color, fontWeight: 600 }}>
        {value}
        <span style={{ fontSize: fontSize * 0.4, marginLeft: 4 }}>{unit}</span>
      </div>
      <div style={{ marginTop: 8, color: '#888', fontSize: 12 }}>
        {panel.metricPaths.join(' / ')}
      </div>
      {compare && compareLast && (
        <div style={{ marginTop: 4, color: '#888', fontSize: 12 }}>
          {compare.name}: {compareLast.value}
          {unit}
        </div>
      )}
    </div>
  );
}
