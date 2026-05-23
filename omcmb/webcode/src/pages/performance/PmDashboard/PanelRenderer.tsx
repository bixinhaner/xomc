/**
 * 按 panel.panelType 路由到具体图表组件。
 *
 * v1：7 种 PanelType 全部走 ECharts 或 antd Statistic 占位实现；接 G5/G7 实际数据走
 * usePanelData hook（见底部）— 拉对应粒度 + 维度的聚合数据。
 *
 * T-0164 收尾 G6-Gap-5 加 topn + big_number 两类型；mock 数据 + 简化展示。
 */

import { Statistic, Table, Empty } from 'antd';
import ReactECharts from 'echarts-for-react';
import type { Panel } from '@core/types/pmDashboard';

interface Props {
  panel: Panel;
}

export function PanelRenderer({ panel }: Props) {
  switch (panel.panelType) {
    case 'kpi_card':
      return <KpiCardRenderer panel={panel} />;
    case 'line_chart':
      return <LineChartRenderer panel={panel} />;
    case 'bar_chart':
      return <BarChartRenderer panel={panel} />;
    case 'table':
      return <TableRenderer panel={panel} />;
    case 'gauge':
      return <GaugeRenderer panel={panel} />;
    case 'topn':
      return <TopNRenderer panel={panel} />;
    case 'big_number':
      return <BigNumberRenderer panel={panel} />;
    default:
      return <Empty description={`未支持的 panel type: ${panel.panelType}`} />;
  }
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

function KpiCardRenderer({ panel }: Props) {
  // v1: 占位数据；G6 task 13 集成时改为 usePanelData
  // G6-Gap-9 (a): unit 自动判断 + (d) 延迟角标占位
  const path0 = panel.metricPaths[0] ?? '';
  const unit = inferPctUnit(path0, panel.config?.unit as string | undefined);
  const threshold = panel.config?.threshold as number | undefined;
  const value = 94.5;
  const overThreshold = threshold !== undefined && value < threshold;
  return (
    <div style={{ position: 'relative' }}>
      <Statistic
        title={panel.metricPaths.join(' / ')}
        value={value}
        suffix={unit}
        precision={2}
        valueStyle={overThreshold ? { color: '#ff4d4f' } : undefined}
      />
      {/* G6-Gap-9 (d) 延迟角标：mock 数据无延迟信息时占位；接真数据后由后端 ingest_lag_s 决定显示 */}
      {/* eslint-disable-next-line @typescript-eslint/no-unused-expressions */}
    </div>
  );
}

function LineChartRenderer({ panel }: Props) {
  // G6-Gap-9 (b) 阈值线 + (e) end_time vs ingest_time 取决于 panel.config.time_axis
  const threshold = panel.config?.threshold as number | undefined;
  const timeAxis = (panel.config?.time_axis as string) ?? 'end_time'; // end_time | ingest_time
  const markLine = buildThresholdMarkLine(threshold);
  const option = {
    grid: { left: 40, right: 16, top: 24, bottom: 40 },
    xAxis: {
      type: 'category',
      data: ['10:00', '11:00', '12:00', '13:00', '14:00', '15:00'],
      name: timeAxis === 'ingest_time' ? '入库时间' : '采集时间',
      nameLocation: 'middle',
      nameGap: 24,
    },
    yAxis: { type: 'value' },
    series: panel.metricPaths.map((path, i) => ({
      name: path,
      type: 'line',
      smooth: true,
      // G6-Gap-9 (c) 缺采：用 '-' 标记，echarts 自动断线（不画 0）
      data: [95, 96, '-', 97, 96, 98].map((v, j) => (typeof v === 'number' ? v + i : v)) as Array<
        number | '-'
      >,
      connectNulls: false,
      markLine: i === 0 ? markLine : undefined,
    })),
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
  };
  return <ReactECharts option={option} style={{ height: 200 }} />;
}

function BarChartRenderer({ panel }: Props) {
  const threshold = panel.config?.threshold as number | undefined;
  const markLine = buildThresholdMarkLine(threshold);
  const option = {
    grid: { left: 40, right: 16, top: 24, bottom: 32 },
    xAxis: { type: 'category', data: ['SN-001', 'SN-002', 'SN-003', 'SN-004'] },
    yAxis: { type: 'value' },
    series: panel.metricPaths.map((path, i) => ({
      name: path,
      type: 'bar',
      data: [120, 200, 150, 80],
      markLine: i === 0 ? markLine : undefined,
    })),
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
  };
  return <ReactECharts option={option} style={{ height: 200 }} />;
}

function TableRenderer({ panel }: Props) {
  return (
    <Table
      size="small"
      pagination={false}
      columns={[
        { title: '设备/组', dataIndex: 'name' },
        ...panel.metricPaths.map((p) => ({ title: p, dataIndex: p })),
      ]}
      dataSource={[
        { key: '1', name: '组A', ...Object.fromEntries(panel.metricPaths.map((p) => [p, 95.2])) },
        { key: '2', name: '组B', ...Object.fromEntries(panel.metricPaths.map((p) => [p, 92.7])) },
      ]}
    />
  );
}

function GaugeRenderer({ panel }: Props) {
  const value = 92;
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
function TopNRenderer({ panel }: Props) {
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
function BigNumberRenderer({ panel }: Props) {
  const fontSize = (panel.config?.font_size as number) ?? 56;
  const warningThreshold = panel.config?.warning_threshold as number | undefined;
  const unit = (panel.config?.unit as string) ?? '';
  const value = 92.4;
  const color =
    warningThreshold !== undefined && value < warningThreshold ? '#ff4d4f' : '#1677ff';
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', padding: 16 }}>
      <div style={{ fontSize, lineHeight: 1.2, color, fontWeight: 600 }}>
        {value}
        <span style={{ fontSize: fontSize * 0.4, marginLeft: 4 }}>{unit}</span>
      </div>
      <div style={{ marginTop: 8, color: '#888', fontSize: 12 }}>
        {panel.metricPaths.join(' / ')}
      </div>
    </div>
  );
}
