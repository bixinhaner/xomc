/**
 * G7-Gap-2 + G7-Gap-3：把 adhoc 任务结果包装成 G6 panel 风格的可视化。
 *
 * - 行为类似 PanelRenderer.LineChartRenderer：按粒度 Tab 切换 + 多 series（每个 metric 一条）
 * - 不写 pm_panels 表，运行时构造（与 plan §G7-Gap-2 一致）
 * - 粒度 Tab 完全由结果数据驱动 — task.granularities 多个时 Tab 显示
 *
 * 与 G6 panel 一致的视觉：ECharts Line + 缺采 '-' 断线
 */

import { useMemo, useState, type ReactNode } from 'react';
import { useIntl, type IntlShape } from 'react-intl';
import { Alert, Button, Card, Empty, Space, Spin, Table, Tabs, Tag, Tooltip, Typography } from 'antd';
import { FileExcelOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import dayjs from 'dayjs';
import { usePmAdhocDetail, usePmAdhocResults } from '@core/hooks/api/usePmAdhoc';
import type { AdhocResultRow } from '@core/types/pmAdhoc';
import { exportWorkbook } from '@core/utils/excelExport';

interface Props {
  taskId: string;
  /**
   * 嵌入仪表盘 Panel 时为 true：去掉外层 Card（Panel 自身已是卡片，避免卡中卡 + 标题重复），
   * 只渲染「摘要 + 导出」头行 + 粒度 Tab。默认 false（独立结果弹窗用，带完整 Card）。
   */
  embedded?: boolean;
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
  // 按 metricPath 分组；系列名用友好名（displayName，KPI 不露 K 编号），回退 metricPath。
  const byMetric = new Map<string, { name: string; points: Map<string, number> }>();
  filtered.forEach((r) => {
    let m = byMetric.get(r.metricPath);
    if (!m) {
      m = { name: r.displayName || r.metricPath, points: new Map() };
      byMetric.set(r.metricPath, m);
    }
    m.points.set(r.startTime, r.metricValue);
  });
  const out: MetricSeries[] = [];
  byMetric.forEach((m) => {
    const values: Array<number | '-'> = buckets.map((b) => {
      const v = m.points.get(b);
      return v === undefined ? '-' : v;
    });
    out.push({ name: m.name, buckets, values });
  });
  return out;
}

// 结果表的列定义：页面表格与导出 Excel 共用同一份，保证列集合 / 列名 / 格式不漂移。
// toText 给出导出用纯文本（含时间格式化、聚合组文案）；renderCell 仅页面展示需要富渲染时提供。
interface AdhocCol {
  header: string;
  width?: number;
  toText: (r: AdhocResultRow) => string | number;
  renderCell?: (r: AdhocResultRow) => ReactNode;
}

function buildAdhocColumns(intl: IntlShape, taskDeviceSns: string[]): AdhocCol[] {
  const deviceText = (r: AdhocResultRow) =>
    r.deviceSn === 'AGGREGATED'
      ? intl.formatMessage({ id: 'perf.adhoc.aggregateGroupUnit' }, { count: taskDeviceSns.length })
      : r.deviceOui
        ? `${r.deviceOui}/${r.deviceSn}`
        : r.deviceSn;
  return [
    {
      header: intl.formatMessage({ id: 'perf.adhoc.colDevice' }),
      width: 200,
      toText: deviceText,
      renderCell: (r) =>
        r.deviceSn === 'AGGREGATED' ? (
          <Tooltip title={<pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{taskDeviceSns.join('\n')}</pre>}>
            <Tag color="purple" style={{ cursor: 'help' }}>
              {deviceText(r)}
            </Tag>
          </Tooltip>
        ) : (
          deviceText(r)
        ),
    },
    { header: intl.formatMessage({ id: 'perf.adhoc.colMetric' }), toText: (r) => r.displayName || r.metricPath },
    { header: intl.formatMessage({ id: 'perf.adhoc.colValue' }), width: 120, toText: (r) => r.metricValue },
    {
      header: intl.formatMessage({ id: 'perf.adhoc.colStartTime' }),
      width: 160,
      toText: (r) => (r.startTime ? dayjs(r.startTime).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      header: intl.formatMessage({ id: 'perf.adhoc.colEndTime' }),
      width: 160,
      toText: (r) => (r.endTime ? dayjs(r.endTime).format('YYYY-MM-DD HH:mm') : '-'),
    },
  ];
}

export function AdhocResultPanel({ taskId, embedded = false }: Props) {
  const intl = useIntl();
  const taskQuery = usePmAdhocDetail(taskId);
  const { data: resultsResp, isLoading: rowsLoading } = usePmAdhocResults(taskId);
  const rows = resultsResp?.rows ?? [];
  const totalRows = resultsResp?.total ?? rows.length;
  // T-0194：真实总数 > 返回行数 = 被 limit 截断，给诚实提示。
  const truncated = totalRows > rows.length;

  const granularities = useMemo(() => taskQuery.data?.granularities ?? [], [taskQuery.data?.granularities]);
  const [activeGran, setActiveGran] = useState<string | undefined>(undefined);
  const effectiveGran = activeGran ?? granularities[0];

  if (taskQuery.isLoading) return <Spin tip={intl.formatMessage({ id: 'perf.dashboard.loadingTask' })} />;
  if (taskQuery.isError || !taskQuery.data) {
    return (
      <Alert
        type="warning"
        showIcon
        message={intl.formatMessage({ id: 'perf.dashboard.taskUnavailable' })}
        description={intl.formatMessage({ id: 'perf.dashboard.taskUnavailableDesc' })}
      />
    );
  }

  const task = taskQuery.data;

  // G7-Gap-4：导出 adhoc 结果（多 sheet，按粒度分）
  // 列集合 / 列名 / 时间格式与页面表格 (GranularityView) 共用 buildAdhocColumns，保持一致
  const handleExportExcel = () => {
    const cols = buildAdhocColumns(intl, task.deviceSns);
    const sheets = granularities.map((g) => ({
      name: g,
      rows: rows
        .filter((r) => r.granularity === g)
        .map((r) => {
          const row: Record<string, string | number> = {};
          cols.forEach((c) => {
            row[c.header] = c.toText(r);
          });
          return row;
        }),
    }));
    exportWorkbook(`adhoc_${task.name}_${task.id.slice(0, 8)}`, sheets);
  };

  // 摘要 + 导出，Card 模式放 extra，嵌入模式单起一行
  const summary = (
    <Space size={6}>
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        {intl.formatMessage(
          { id: 'perf.adhoc.resultSummary' },
          {
            deviceCount: task.deviceSns.length,
            metricCount: task.metricPaths.length,
            granCount: granularities.length,
          },
        )}
      </Typography.Text>
      <Button
        size="small"
        icon={<FileExcelOutlined />}
        onClick={handleExportExcel}
        disabled={rows.length === 0}
      >
        {intl.formatMessage({ id: 'perf.adhoc.exportExcel' })}
      </Button>
    </Space>
  );

  const body =
    granularities.length === 0 ? (
      <Empty description={intl.formatMessage({ id: 'perf.adhoc.noGranInfo' })} />
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
              taskDeviceSns={task.deviceSns}
            />
          ),
        }))}
      />
    );

  // T-0194：截断诚实提示——返回行数 < 真实总数时显示，提示缩小范围。
  const truncationAlert = truncated ? (
    <Alert
      type="warning"
      showIcon
      style={{ marginBottom: 8 }}
      message={intl.formatMessage(
        { id: 'perf.adhoc.truncatedTip' },
        { shown: rows.length, total: totalRows },
      )}
    />
  ) : null;

  // 嵌入仪表盘 Panel：外层卡片由 Panel 提供，这里不再套 Card
  if (embedded) {
    return (
      <div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>{summary}</div>
        {truncationAlert}
        {body}
      </div>
    );
  }

  return (
    <Card
      size="small"
      title={
        <span>
          {task.name}
          <Tag color="purple" style={{ marginLeft: 8 }}>
            {task.mode === 'continuous'
              ? intl.formatMessage({ id: 'perf.adhoc.modeContinuous' })
              : intl.formatMessage({ id: 'perf.adhoc.modeOneshot' })}
          </Tag>
          <Tag color={task.status === 'succeeded' ? 'success' : task.status === 'failed' ? 'error' : 'processing'}>
            {task.status}
          </Tag>
        </span>
      }
      extra={summary}
    >
      {truncationAlert}
      {body}
    </Card>
  );
}

function GranularityView({
  rows,
  granularity,
  loading,
  taskDeviceSns,
}: {
  rows: AdhocResultRow[];
  granularity: string;
  loading: boolean;
  taskDeviceSns: string[];
}) {
  const intl = useIntl();
  const series = useMemo(() => buildSeriesByMetric(rows, granularity), [rows, granularity]);
  const cols = useMemo(() => buildAdhocColumns(intl, taskDeviceSns), [intl, taskDeviceSns]);
  if (loading) return <Spin />;
  if (series.length === 0) {
    return <Empty description={intl.formatMessage({ id: 'perf.adhoc.emptyGranNoData' }, { gran: granularity })} />;
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
        columns={cols.map((c) => ({
          title: c.header,
          key: c.header,
          width: c.width,
          render: (_: unknown, r: AdhocResultRow) => (c.renderCell ? c.renderCell(r) : c.toText(r)),
        }))}
      />
    </>
  );
}
