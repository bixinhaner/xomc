/**
 * 按 panel.panelType 路由到具体图表组件。
 *
 * v1：5 种 PanelType 全部走 ECharts 或 antd Statistic 占位实现；接 G5/G7 实际数据走
 * usePanelData hook（见底部）— 拉对应粒度 + 维度的聚合数据。
 *
 * 此 v1 用 mock 数据 + 简化展示，等 G6 任务 13 集成时可换真实拉取。
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
    default:
      return <Empty description={`未支持的 panel type: ${panel.panelType}`} />;
  }
}

function KpiCardRenderer({ panel }: Props) {
  // v1: 占位数据；G6 task 13 集成时改为 usePanelData
  const unit = (panel.config?.unit as string) ?? '';
  return (
    <Statistic
      title={panel.metricPaths.join(' / ')}
      value={94.5}
      suffix={unit}
      precision={2}
    />
  );
}

function LineChartRenderer({ panel }: Props) {
  const option = {
    grid: { left: 40, right: 16, top: 24, bottom: 32 },
    xAxis: {
      type: 'category',
      data: ['10:00', '11:00', '12:00', '13:00', '14:00', '15:00'],
    },
    yAxis: { type: 'value' },
    series: panel.metricPaths.map((path, i) => ({
      name: path,
      type: 'line',
      smooth: true,
      data: [95, 96, 94, 97, 96, 98].map((v) => v + i),
    })),
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
  };
  return <ReactECharts option={option} style={{ height: 200 }} />;
}

function BarChartRenderer({ panel }: Props) {
  const option = {
    grid: { left: 40, right: 16, top: 24, bottom: 32 },
    xAxis: { type: 'category', data: ['SN-001', 'SN-002', 'SN-003', 'SN-004'] },
    yAxis: { type: 'value' },
    series: panel.metricPaths.map((path) => ({
      name: path,
      type: 'bar',
      data: [120, 200, 150, 80],
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
