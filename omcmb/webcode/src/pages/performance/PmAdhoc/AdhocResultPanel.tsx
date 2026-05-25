/**
 * G7-Gap-2 + G7-Gap-3：把 adhoc 任务结果包装成 G6 panel 风格的可视化。
 *
 * - 行为类似 PanelRenderer.LineChartRenderer：按粒度 Tab 切换 + 多 series（每个 metric 一条）
 * - 不写 pm_panels 表，运行时构造（与 plan §G7-Gap-2 一致）
 * - 粒度 Tab 完全由结果数据驱动 — task.granularities 多个时 Tab 显示
 *
 * 与 G6 panel 一致的视觉：ECharts Line + 缺采 '-' 断线
 */

import { useMemo, useState } from 'react';
import { Alert, Button, Card, Empty, Space, Spin, Table, Tabs, Tag, Typography } from 'antd';
import { FileExcelOutlined, PrinterOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import { usePmAdhocDetail, usePmAdhocResults } from '@core/hooks/api/usePmAdhoc';
import type { AdhocResultRow } from '@core/types/pmAdhoc';
import { exportWorkbook, printAsPDF } from '@core/utils/excelExport';

interface Props {
  taskId: string;
}

interface MetricSeries {
  name: string;
  buckets: string[];
  values: Array<number | '-'>;
}

function buildSeriesByMetric(rows: AdhocResultRow[], granularity: string): MetricSeries[] {
  const filtered = rows.filter((r) => r.granularity === granularity);
  if (filtered.length === 0) return [];
  // 取所有 bucket（按 startTime 去重升序）
  const bucketSet = new Set<string>();
  filtered.forEach((r) => bucketSet.add(r.startTime));
  const buckets = Array.from(bucketSet).sort();
  // 按 metricPath 分组
  const byMetric = new Map<string, Map<string, number>>();
  filtered.forEach((r) => {
    let m = byMetric.get(r.metricPath);
    if (!m) {
      m = new Map();
      byMetric.set(r.metricPath, m);
    }
    m.set(r.startTime, r.metricValue);
  });
  const out: MetricSeries[] = [];
  byMetric.forEach((m, name) => {
    const values: Array<number | '-'> = buckets.map((b) => {
      const v = m.get(b);
      return v === undefined ? '-' : v;
    });
    out.push({ name, buckets, values });
  });
  return out;
}

export function AdhocResultPanel({ taskId }: Props) {
  const taskQuery = usePmAdhocDetail(taskId);
  const { data: rows = [], isLoading: rowsLoading } = usePmAdhocResults(taskId);

  const granularities = useMemo(() => taskQuery.data?.granularities ?? [], [taskQuery.data?.granularities]);
  const [activeGran, setActiveGran] = useState<string | undefined>(undefined);
  const effectiveGran = activeGran ?? granularities[0];

  if (taskQuery.isLoading) return <Spin tip="加载任务..." />;
  if (taskQuery.isError || !taskQuery.data) {
    return (
      <Alert
        type="warning"
        showIcon
        message="任务不可用"
        description="任务已被删除或结果已超出保留期。"
      />
    );
  }

  const task = taskQuery.data;

  // G7-Gap-4：导出 adhoc 结果（多 sheet，按粒度分）
  const handleExportExcel = () => {
    const sheets = granularities.map((g) => ({
      name: g,
      rows: rows
        .filter((r) => r.granularity === g)
        .map((r) => ({
          device_oui: r.deviceOui,
          device_sn: r.deviceSn,
          metric_path: r.metricPath,
          metric_type: r.metricType,
          metric_value: r.metricValue,
          statis_type: r.statisType ?? '',
          time: r.time,
          start_time: r.startTime,
          end_time: r.endTime,
        })),
    }));
    exportWorkbook(`adhoc_${task.name}_${task.id.slice(0, 8)}`, sheets);
  };

  return (
    <Card
      size="small"
      title={
        <span>
          {task.name}
          <Tag color="purple" style={{ marginLeft: 8 }}>
            {task.mode === 'continuous' ? '持续' : '单次'}
          </Tag>
          <Tag color={task.status === 'succeeded' ? 'success' : task.status === 'failed' ? 'error' : 'processing'}>
            {task.status}
          </Tag>
        </span>
      }
      extra={
        <Space size={6}>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {task.deviceSns.length} 设备 × {task.metricPaths.length} 指标 × {granularities.length} 粒度
          </Typography.Text>
          <Button
            size="small"
            icon={<FileExcelOutlined />}
            onClick={handleExportExcel}
            disabled={rows.length === 0}
          >
            导出 Excel
          </Button>
          <Button
            size="small"
            icon={<PrinterOutlined />}
            onClick={() => printAsPDF(`adhoc_${task.name}`)}
          >
            导出 PDF
          </Button>
        </Space>
      }
    >
      {granularities.length === 0 ? (
        <Empty description="任务无粒度信息" />
      ) : (
        <Tabs
          size="small"
          activeKey={effectiveGran}
          onChange={setActiveGran}
          items={granularities.map((g) => ({
            key: g,
            label: g,
            children: (
              <GranularityView
                rows={rows}
                granularity={g}
                loading={rowsLoading}
              />
            ),
          }))}
        />
      )}
    </Card>
  );
}

function GranularityView({
  rows,
  granularity,
  loading,
}: {
  rows: AdhocResultRow[];
  granularity: string;
  loading: boolean;
}) {
  const series = useMemo(() => buildSeriesByMetric(rows, granularity), [rows, granularity]);
  if (loading) return <Spin />;
  if (series.length === 0) {
    return <Empty description={`${granularity} 粒度暂无数据`} />;
  }
  const buckets = series[0].buckets;
  const option = {
    grid: { left: 50, right: 16, top: 30, bottom: 40 },
    xAxis: { type: 'category', data: buckets, name: 'start_time', nameLocation: 'middle', nameGap: 24 },
    yAxis: { type: 'value' },
    series: series.map((s) => ({
      name: s.name,
      type: 'line',
      smooth: true,
      data: s.values,
      connectNulls: false,
    })),
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
  };
  return (
    <>
      <ReactECharts option={option} style={{ height: 240 }} />
      <Table
        size="small"
        rowKey={(r) => `${r.metricPath}:${r.deviceSn}:${r.startTime}`}
        dataSource={rows.filter((r) => r.granularity === granularity)}
        pagination={{ pageSize: 10, size: 'small' }}
        style={{ marginTop: 12 }}
        columns={[
          { title: '设备', render: (_, r) => `${r.deviceOui}/${r.deviceSn}`, width: 200 },
          { title: '指标', dataIndex: 'metricPath' },
          { title: '值', dataIndex: 'metricValue', width: 120 },
          { title: '时间', dataIndex: 'startTime', width: 200 },
        ]}
      />
    </>
  );
}
