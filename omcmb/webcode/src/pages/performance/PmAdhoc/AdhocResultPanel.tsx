/**
 * G7-Gap-2 + G7-Gap-3：把 adhoc 任务结果包装成 G6 panel 风格的可视化。
 *
 * - 行为类似 PanelRenderer.LineChartRenderer：按粒度 Tab 切换 + 多 series（每个 metric 一条）
 * - 不写 pm_panels 表，运行时构造（与 plan §G7-Gap-2 一致）
 * - 粒度 Tab 完全由结果数据驱动 — task.granularities 多个时 Tab 显示
 *
 * 与首页 KPI 趋势一致的视觉：ECharts Line + 缺采 '-' 保留、跨空桶续连
 */

import { useEffect, useMemo, useState } from 'react';
import { useIntl, type IntlShape } from 'react-intl';
import { Alert, Button, Card, DatePicker, Empty, Space, Spin, Table, Tabs, Tag, Tooltip, Typography, message } from 'antd';
import { LoadingSpinner } from '@/components/LoadingSpinner';
import { ExportOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import type { Dayjs } from 'dayjs';
import { usePmAdhocDetail, usePmAdhocResults } from '@core/hooks/api/usePmAdhoc';
import { useCreateKpiExport } from '@core/hooks/api/useKpiExport';
import { useSystemTimezoneValue } from '@core/hooks/api/useSystemTimezone';
import type { AdhocResultRow, AdhocDimension } from '@core/types/pmAdhoc';
import { formatPmMetricDisplayValue, isFinitePmMetricValue, normalizePmMetricValue } from '@core/utils/pmMetricValue';
import { buildAdhocExportParams, defaultExportTaskName } from '@core/utils/kpiExportParams';
import { adhocIncludesCell, adhocObjectHeaderKey, adhocObjectName, adhocTechnology, objectKeyOf } from './adhocObjectColumn';
import { formatSystemTime, nowInSystemTimezone, toSystemTimezoneRFC3339 } from '@core/utils/systemTime';
import { buildAdhocChartOption, type AdhocMetricSeries } from './adhocChartOption';
import { useTechnologyDictionary } from '@core/hooks/api/useTechnologyDictionary';
import { displayAdhocTaskName } from '../adhocTaskDisplay';

// 按粒度算默认时窗：覆盖最近 7 天，但粒度粗于"天"时至少 7 个周期。
// 15min / hourly / daily → 7 天；weekly → 7 周；monthly → 7 月。end 取系统时区当前时刻。
function defaultWindowByGranularity(granularity: string | undefined, systemTimezone?: string): [Dayjs, Dayjs] {
  const end = nowInSystemTimezone(systemTimezone);
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

function buildSeriesByMetric(rows: AdhocResultRow[], granularity: string): AdhocMetricSeries[] {
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
    if (isFinitePmMetricValue(r.metricValue)) {
      m.points.set(r.startTime, r.metricValue);
    }
  });
  const out: AdhocMetricSeries[] = [];
  byMetric.forEach((m) => {
    const values: Array<number | '-'> = buckets.map((b) => {
      const v = m.points.get(b);
      return v === undefined ? '-' : v;
    });
    out.push({ name: m.name, buckets, values });
  });
  return out;
}

function formatResultSummary(
  intl: IntlShape,
  task: { dimension: AdhocDimension; deviceSns: string[]; metricPaths: string[] },
  granularities: string[],
): string {
  const showsSelectedDeviceCount = task.dimension === 'device' || task.dimension === 'aggregate_group';

  if (!showsSelectedDeviceCount) {
    return intl.formatMessage(
      { id: 'perf.adhoc.resultSummaryWithoutDeviceCount' },
      {
        metricCount: task.metricPaths.length,
        granCount: granularities.length,
      },
    );
  }

  return intl.formatMessage(
    { id: 'perf.adhoc.resultSummary' },
    {
      deviceCount: task.deviceSns.length,
      metricCount: task.metricPaths.length,
      granCount: granularities.length,
    },
  );
}

// 结果表横表透视：把长表（每行一个数据点）摊成横表——同一 (设备 × 小区/PLMN × 时间) 行键凑一行，
// 每个指标占一列，列名「编号(名·类型)」（与导出 CSV 横表同口径，列名自带中文，顺带解决"指标列显编号"）。
interface WideMetricCol {
  metricPath: string;
  title: string; // 编号(名·类型)
}
interface WideResultRow {
  rowKey: string;
  deviceSn: string; // 用于 AGGREGATED 富渲染判断
  deviceLabel: string;
  cellPlmn: string;
  technology: string; // device_group 维度从 objectLdn 解析出的制式（lte/nr/gsm 大写）；其它维度空
  time: string;
  endTime: string;
  values: Record<string, number | null>;
}

function buildWideTable(
  rows: AdhocResultRow[],
  granularity: string,
  intl: IntlShape,
  taskDeviceSns: string[],
  dimension: AdhocDimension,
): { columns: WideMetricCol[]; data: WideResultRow[] } {
  const filtered = rows.filter((r) => r.granularity === granularity);
  // 列集：distinct metricPath（按编号升序），列名「编号(名·类型)」。
  const colMap = new Map<string, WideMetricCol>();
  filtered.forEach((r) => {
    if (!colMap.has(r.metricPath)) {
      const name = r.displayName || r.metricPath;
      colMap.set(r.metricPath, { metricPath: r.metricPath, title: `${r.metricPath}(${name}·${r.metricType})` });
    }
  });
  const columns = Array.from(colMap.values()).sort((a, b) => a.metricPath.localeCompare(b.metricPath));
  // 行键：对象（按维度取分组键）× 小区/PLMN（仅 device 维度有意义）× 时间。
  // 聚合维度无小区后，行键退化为「对象 × 时间」，避免不同产品/组撞同 deviceSn 而碰撞。
  const rowMap = new Map<string, WideResultRow>();
  filtered.forEach((r) => {
    const cellPart = dimension === 'device' ? (r.objectLdn ?? '') : '';
    const rowKey = `${objectKeyOf(r, dimension)}|${cellPart}|${r.startTime}`;
    let wr = rowMap.get(rowKey);
    if (!wr) {
      wr = {
        rowKey,
        deviceSn: r.deviceSn,
        deviceLabel: adhocObjectName(r, dimension, taskDeviceSns, intl),
        cellPlmn: r.objectLdn || '-',
        technology: dimension === 'device_group' ? adhocTechnology(r) : '',
        time: r.startTime ? formatSystemTime(r.startTime, { format: 'YYYY-MM-DD HH:mm', placeholder: '-' }) : '-',
        endTime: r.endTime ? formatSystemTime(r.endTime, { format: 'YYYY-MM-DD HH:mm', placeholder: '-' }) : '-',
        values: {},
      };
      rowMap.set(rowKey, wr);
    }
    wr.values[r.metricPath] = normalizePmMetricValue(r.metricValue);
  });
  const data = Array.from(rowMap.values()).sort((a, b) => {
    if (a.deviceLabel !== b.deviceLabel) return a.deviceLabel.localeCompare(b.deviceLabel);
    if (a.cellPlmn !== b.cellPlmn) return a.cellPlmn.localeCompare(b.cellPlmn);
    return a.time.localeCompare(b.time);
  });
  return { columns, data };
}

export function AdhocResultPanel({ taskId, embedded = false }: Props) {
  const intl = useIntl();
  const { labelForTechnology } = useTechnologyDictionary();
  const taskQuery = usePmAdhocDetail(taskId);
  // #563：筛选器按系统时区展示和序列化，与图表 X 轴统一参照系。
  const systemTimezone = useSystemTimezoneValue();

  // 二次时窗筛选：同时驱动「页面展示重查」与「导出取数」。默认按粒度算（见 defaultWindowByGranularity）。
  // windowTouched=用户手动改过后不再被粒度联动覆盖。
  const [windowRange, setWindowRange] = useState<[Dayjs, Dayjs]>(() => defaultWindowByGranularity(undefined, systemTimezone));
  const [windowTouched, setWindowTouched] = useState(false);
  const startISO = toSystemTimezoneRFC3339(windowRange[0], systemTimezone) ?? windowRange[0].toISOString();
  const endISO = toSystemTimezoneRFC3339(windowRange[1], systemTimezone) ?? windowRange[1].toISOString();

  const { data: resultsResp, isLoading: rowsLoading } = usePmAdhocResults(taskId, {
    startTime: startISO,
    endTime: endISO,
  });
  const rows = resultsResp?.rows ?? [];
  const totalRows = resultsResp?.total ?? rows.length;
  const incompleteRows = rows.filter((row) => row.complete === false);
  const incompleteWindowCount = new Set(
    incompleteRows.map((row) => `${row.taskVersionId}|${row.granularity}|${row.startTime}|${row.endTime}`),
  ).size;
  const missingSlotCount = incompleteRows.reduce((total, row) => total + (row.missingSlots ?? 0), 0);
  // T-0194：真实总数 > 返回行数 = 被 limit 截断，给诚实提示。
  const truncated = totalRows > rows.length;

  const granularities = useMemo(() => taskQuery.data?.granularities ?? [], [taskQuery.data?.granularities]);
  const [activeGran, setActiveGran] = useState<string | undefined>(undefined);
  const effectiveGran = activeGran ?? granularities[0];

  // 粒度联动默认时窗：粒度就绪/切换时，若用户未手动改过则按当前粒度重设默认时窗。
  useEffect(() => {
    if (windowTouched) return;
    setWindowRange(defaultWindowByGranularity(effectiveGran, systemTimezone));
  }, [effectiveGran, windowTouched, systemTimezone]);

  // 导出（KPI-EXPORT 自定义聚合任务结果来源）：建后端异步任务 → 文件传输菜单下载，带当前二次时窗。
  const createExport = useCreateKpiExport();
  const handleExport = () => {
    createExport.mutate(
      {
        sourceType: 'adhoc_result',
        params: buildAdhocExportParams({ taskId, startTime: startISO, endTime: endISO }),
        taskName: defaultExportTaskName('adhoc_result', new Date(), {
          prefixLabel: intl.formatMessage({ id: 'kpiExport.fileName.prefix' }),
          sourceLabel: intl.formatMessage({ id: 'kpiExport.source.adhocResult' }),
          subjectName: taskQuery.data?.name,
        }),
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

  if (taskQuery.isLoading) return (
    <LoadingSpinner tip={intl.formatMessage({ id: 'perf.dashboard.loadingTask' })} />
  );
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
        {formatResultSummary(intl, task, granularities)}
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
              dimension={task.dimension}
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
  const incompleteAlert = incompleteRows.length > 0 ? (
    <Alert
      type="warning"
      showIcon
      style={{ marginBottom: 8 }}
      message={intl.formatMessage(
        { id: 'perf.adhoc.incompleteWindowTip' },
        { windows: incompleteWindowCount, slots: missingSlotCount },
      )}
    />
  ) : null;

  // 嵌入仪表盘 Panel：外层卡片由 Panel 提供，这里不再套 Card
  if (embedded) {
    return (
      <div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>{summary}</div>
        {truncationAlert}
        {incompleteAlert}
        {body}
      </div>
    );
  }

  return (
    <Card
      size="small"
      title={
        <span>
          {displayAdhocTaskName(task, labelForTechnology)}
          <Tag color={task.status === 'succeeded' ? 'success' : task.status === 'failed' ? 'error' : 'processing'} style={{ marginLeft: 8 }}>
            {task.status}
          </Tag>
        </span>
      }
      extra={summary}
    >
      {truncationAlert}
      {incompleteAlert}
      {body}
    </Card>
  );
}

function GranularityView({
  rows,
  granularity,
  loading,
  taskDeviceSns,
  dimension,
}: {
  rows: AdhocResultRow[];
  granularity: string;
  loading: boolean;
  taskDeviceSns: string[];
  dimension: AdhocDimension;
}) {
  const intl = useIntl();
  const { labelForTechnology } = useTechnologyDictionary();
  const series = useMemo(() => buildSeriesByMetric(rows, granularity), [rows, granularity]);
  const { columns: metricCols, data: wideData } = useMemo(
    () => buildWideTable(rows, granularity, intl, taskDeviceSns, dimension),
    [rows, granularity, intl, taskDeviceSns, dimension],
  );
  if (loading) return <Spin />;
  if (series.length === 0) {
    return <Empty description={intl.formatMessage({ id: 'perf.adhoc.emptyGranNoData' }, { gran: granularity })} />;
  }
  const buckets = series[0].buckets;
  const option = buildAdhocChartOption(series, buckets);
  return (
    <>
      <ReactECharts option={option} style={{ height: 240 }} />
      <Table
        size="small"
        rowKey={(r) => r.rowKey}
        dataSource={wideData}
        pagination={{ pageSize: 10, size: 'small' }}
        style={{ marginTop: 12 }}
        scroll={{ x: 'max-content' }}
        columns={[
          {
            title: intl.formatMessage({ id: adhocObjectHeaderKey(dimension) }),
            key: '__object',
            width: 200,
            fixed: 'left' as const,
            render: (_: unknown, r: WideResultRow) =>
              dimension === 'aggregate_group' && r.deviceSn === 'AGGREGATED' ? (
                <Tooltip title={<pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{taskDeviceSns.join('\n')}</pre>}>
                  <Tag color="purple" style={{ cursor: 'help' }}>
                    {r.deviceLabel}
                  </Tag>
                </Tooltip>
              ) : (
                r.deviceLabel
              ),
          },
          ...(dimension === 'device_group'
            ? [
                {
                  title: intl.formatMessage({ id: 'perf.adhoc.colTechnology' }),
                  key: '__tech',
                  width: 140,
                  render: (_: unknown, r: WideResultRow) =>
                    r.technology ? <Tag>{labelForTechnology(r.technology.toLowerCase())}</Tag> : '-',
                },
              ]
            : []),
          ...(adhocIncludesCell(dimension)
            ? [
                {
                  title: intl.formatMessage({ id: 'perf.adhoc.colCellPlmn' }),
                  key: '__cell',
                  width: 160,
                  render: (_: unknown, r: WideResultRow) => r.cellPlmn,
                },
              ]
            : []),
          {
            title: intl.formatMessage({ id: 'perf.adhoc.colStartTime' }),
            key: '__time',
            width: 160,
            render: (_: unknown, r: WideResultRow) => r.time,
          },
          {
            title: intl.formatMessage({ id: 'perf.adhoc.colEndTime' }),
            key: '__endTime',
            width: 160,
            render: (_: unknown, r: WideResultRow) => r.endTime,
          },
          ...metricCols.map((c) => ({
            title: c.title,
            key: c.metricPath,
            width: 200,
            render: (_: unknown, r: WideResultRow) => {
              const v = r.values[c.metricPath];
              return formatPmMetricDisplayValue(v);
            },
          })),
        ]}
      />
    </>
  );
}
