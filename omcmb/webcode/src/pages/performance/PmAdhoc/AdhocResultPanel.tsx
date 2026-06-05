/**
 * G7-Gap-2 + G7-Gap-3：把 adhoc 任务结果包装成 G6 panel 风格的可视化。
 *
 * - 行为类似 PanelRenderer.LineChartRenderer：按粒度 Tab 切换 + 多 series（每个 metric 一条）
 * - 不写 pm_panels 表，运行时构造（与 plan §G7-Gap-2 一致）
 * - 粒度 Tab 完全由结果数据驱动 — task.granularities 多个时 Tab 显示
 *
 * 与 G6 panel 一致的视觉：ECharts Line + 缺采 '-' 断线
 */

import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { useIntl, type IntlShape } from 'react-intl';
import { Alert, Button, Card, DatePicker, Empty, Space, Spin, Table, Tabs, Tag, Tooltip, Typography, message } from 'antd';
import { ExportOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import dayjs, { type Dayjs } from 'dayjs';
import { usePmAdhocDetail, usePmAdhocResults } from '@core/hooks/api/usePmAdhoc';
import { useCreateKpiExport } from '@core/hooks/api/useKpiExport';
import type { AdhocResultRow } from '@core/types/pmAdhoc';
import { buildAdhocExportParams, defaultExportTaskName } from '../PmDashboard/kpiExportParams';

// 按粒度算默认时窗：覆盖最近 7 天，但粒度粗于"天"时至少 7 个周期。
// 15min / hourly / daily → 7 天；weekly → 7 周；monthly → 7 月。end 取当前时刻。
function defaultWindowByGranularity(granularity: string | undefined): [Dayjs, Dayjs] {
  const end = dayjs();
  switch (granularity) {
    case 'weekly':
      return [end.subtract(7, 'week'), end];
    case 'monthly':
      return [end.subtract(7, 'month'), end];
    default:
      // 15min / hourly / daily 以及未知粒度都按 7 天
      return [end.subtract(7, 'day'), end];
  }
}

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

  // 二次时窗筛选：同时驱动「页面展示重查」与「导出取数」。默认按粒度算（见 defaultWindowByGranularity）。
  // windowTouched=用户手动改过后不再被粒度联动覆盖。
  const [windowRange, setWindowRange] = useState<[Dayjs, Dayjs]>(() => defaultWindowByGranularity(undefined));
  const [windowTouched, setWindowTouched] = useState(false);
  const startISO = windowRange[0].toISOString();
  const endISO = windowRange[1].toISOString();

  const { data: resultsResp, isLoading: rowsLoading } = usePmAdhocResults(taskId, {
    startTime: startISO,
    endTime: endISO,
  });
  const rows = resultsResp?.rows ?? [];
  const totalRows = resultsResp?.total ?? rows.length;
  // T-0194：真实总数 > 返回行数 = 被 limit 截断，给诚实提示。
  const truncated = totalRows > rows.length;

  const granularities = useMemo(() => taskQuery.data?.granularities ?? [], [taskQuery.data?.granularities]);
  const [activeGran, setActiveGran] = useState<string | undefined>(undefined);
  const effectiveGran = activeGran ?? granularities[0];

  // 粒度联动默认时窗：粒度就绪/切换时，若用户未手动改过则按当前粒度重设默认时窗。
  useEffect(() => {
    if (windowTouched) return;
    setWindowRange(defaultWindowByGranularity(effectiveGran));
  }, [effectiveGran, windowTouched]);

  // 导出（KPI-EXPORT adhoc 来源）：建后端异步任务 → 文件传输菜单下载，带当前二次时窗。
  const createExport = useCreateKpiExport();
  const handleExport = () => {
    createExport.mutate(
      {
        sourceType: 'adhoc',
        params: buildAdhocExportParams({ taskId, startTime: startISO, endTime: endISO }),
        taskName: defaultExportTaskName('adhoc'),
      },
      {
        onSuccess: () => message.success(intl.formatMessage({ id: 'kpiExport.export.submitted' })),
        onError: (e) =>
          message.error(
            intl.formatMessage({ id: 'kpiExport.export.submitFailed' }, { reason: (e as Error)?.message ?? '' }),
          ),
      },
    );
  };

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

  // 摘要 + 时窗筛选 + 导出，Card 模式放 extra，嵌入模式单起一行
  const summary = (
    <Space size={6} wrap>
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
      <DatePicker.RangePicker
        size="small"
        showTime={{ format: 'HH:mm' }}
        format="YYYY-MM-DD HH:mm"
        allowClear={false}
        value={windowRange}
        onChange={(v) => {
          if (v && v[0] && v[1]) {
            setWindowRange([v[0], v[1]]);
            setWindowTouched(true);
          }
        }}
      />
      <Button
        size="small"
        icon={<ExportOutlined />}
        loading={createExport.isPending}
        onClick={handleExport}
        disabled={rows.length === 0}
        title={intl.formatMessage({ id: 'kpiExport.export.tooltip' })}
      >
        {intl.formatMessage({ id: 'kpiExport.export.button' })}
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
