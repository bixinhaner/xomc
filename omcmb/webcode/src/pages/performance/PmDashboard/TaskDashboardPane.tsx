/**
 * T-0187 任务仪表盘右侧出图面板。
 *
 * 选中任务 → 读其详情（维度/粒度/指标集）+ 结果行（limit=10000 取最近 N 行）→ 按 N 个指标自动出图：
 *   - 每指标一张 ECharts 折线（横轴时间、纵轴值，height~260），横向铺满、纵向单列堆叠。
 *   - 多组/多设备同图多条线（系列键派生见 taskDashboardUtils）。
 *   - 任务 granularities 多个时顶部 Segmented 选粒度，默认首个。
 *
 * 自动出图、固定布局，无手工拖拽。
 */

import { useMemo, useState } from 'react';
import { useIntl } from 'react-intl';
import { Alert, App, Button, Card, Empty, Segmented, Space, Spin, Tag, Typography } from 'antd';
import { ExportOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import {
  usePmAdhocDetail,
  usePmAdhocFilterOptions,
  usePmAdhocResults,
} from '@core/hooks/api/usePmAdhoc';
import { useCreateKpiExport } from '@core/hooks/api/useKpiExport';
import { buildAdhocExportParams, defaultExportTaskName } from '@core/utils/kpiExportParams';
import { buildMetricCharts, filterChartsByMetricPaths } from './taskDashboardUtils';
import ChartCard from './ChartCard';
import DashboardFilterBar, { type DashboardFilterValue } from './DashboardFilterBar';
import {
  ALL_HOURS,
  ALL_WEEKDAYS,
  attachCompareSeries,
  dimSelectionToParams,
  extendChartsAxis,
  filterRowsByWeekdayHour,
  previousWindow,
} from './dashboardFilterUtils';

interface Props {
  taskId: string;
}

const RESULTS_LIMIT = 10000;

// 粒度短标签语料键（与旧 GRAN_LABEL 一一对应）。
const GRAN_MSG_IDS: Record<string, string> = {
  '15min': 'perf.dashboard.gran15min',
  hourly: 'perf.dashboard.granHourly',
  daily: 'perf.dashboard.granDaily',
  weekly: 'perf.dashboard.granWeekly',
  monthly: 'perf.dashboard.granMonthly',
};

export default function TaskDashboardPane({ taskId }: Props) {
  const intl = useIntl();
  const { message } = App.useApp();
  // 粒度短标签：有对应键走语料，无键回退原值（等价旧 GRAN_LABEL[g] ?? g）。
  const granLabel = (g: string) =>
    GRAN_MSG_IDS[g] ? intl.formatMessage({ id: GRAN_MSG_IDS[g] }) : g;
  const taskQuery = usePmAdhocDetail(taskId);
  const dimension = taskQuery.data?.dimension;

  // ── 共用三级筛选 + 周期对比开关（本 Pane 持状态，驱动取数 + 二拉）──────
  const [filter, setFilter] = useState<DashboardFilterValue>({
    range: [dayjs().subtract(7, 'day'), dayjs()],
    weekdays: [...ALL_WEEKDAYS],
    hours: [...ALL_HOURS],
    compare: false,
  });

  // ── PM-DASH-DIMFILTER 维度子集筛选（按维度动态显示产品/设备组/频段多选框）──────
  // dimSelected 是筛选框上选中的"分组键原值"（product 维度=product_id；device_group=DeviceGroup=<uuid>；band=Band=<值>）。
  // 切任务（taskId 变）即清空选中，避免上个任务的子集串到新任务——用 React 官方"渲染期调整 state"
  // 模式（记录上次 taskId，变化时同步重置），不放 effect 里 setState。
  const [dimSelected, setDimSelected] = useState<string[]>([]);
  const [prevTaskId, setPrevTaskId] = useState(taskId);
  if (taskId !== prevTaskId) {
    setPrevTaskId(taskId);
    setDimSelected([]);
  }
  const { data: filterOpts } = usePmAdhocFilterOptions(taskId, dimension);
  // 维度→入参映射（纯函数，便于单测）：product→productIds；device_group/band→objectLdns；空选不过滤。
  const { productIds, objectLdns } = dimSelectionToParams(dimension, dimSelected);

  const [start, end] = filter.range;
  const startISO = start.toISOString();
  const endISO = end.toISOString();
  const offsetMs = end.valueOf() - start.valueOf();
  const [prevStart, prevEnd] = previousWindow(filter.range);

  // 大时间段驱动取数（后端按 time 窗口过滤）。仪表盘取较多结果行用于画线。
  const { data: rowsResp, isLoading: rowsLoading } = usePmAdhocResults(taskId, {
    limit: RESULTS_LIMIT,
    startTime: startISO,
    endTime: endISO,
    productIds,
    objectLdns,
  });
  const rawRows = rowsResp?.rows ?? [];
  // 周期对比开关打开时再拉一次上一周期（同任务、上一周期窗口）。
  const { data: prevResp, isLoading: prevLoading } = usePmAdhocResults(
    filter.compare ? taskId : undefined,
    {
      limit: RESULTS_LIMIT,
      startTime: prevStart.toISOString(),
      endTime: prevEnd.toISOString(),
      productIds,
      objectLdns,
    },
  );
  const rawPrevRows = prevResp?.rows ?? [];

  // 星期/小时段=纯前端在已取行里筛命中点（全选不过滤），当前与上一周期套同口径。
  const weekdaySet = useMemo(() => new Set(filter.weekdays), [filter.weekdays]);
  const hourSet = useMemo(() => new Set(filter.hours), [filter.hours]);
  const rows = useMemo(
    () => filterRowsByWeekdayHour(rawRows, weekdaySet, hourSet),
    [rawRows, weekdaySet, hourSet],
  );
  const prevRows = useMemo(
    () => filterRowsByWeekdayHour(rawPrevRows, weekdaySet, hourSet),
    [rawPrevRows, weekdaySet, hourSet],
  );

  // 触顶提示：结果接口 LIMIT 上限，触顶可能截断 → 给可见提示，不静默。
  const truncated = rawRows.length >= RESULTS_LIMIT;

  const granularities = useMemo(
    () => taskQuery.data?.granularities ?? [],
    [taskQuery.data?.granularities],
  );
  const [activeGran, setActiveGran] = useState<string | undefined>(undefined);
  const effectiveGran = activeGran && granularities.includes(activeGran) ? activeGran : granularities[0];

  const charts = useMemo(() => {
    if (!taskQuery.data || !effectiveGran) return [];
    // T-0194：按任务已选指标清单过滤出图（空清单则不过滤，兜底全画），让"指标数 X"与出图数一致。
    const metricPaths = taskQuery.data.metricPaths;
    // T-AXISFILL：转置出当前图集后立即扩轴（按范围+粒度连续铺刻度、套星期/小时筛选、并集真实桶），
    // 空刻度补 '-'，再挂周期对比（compare 按毫秒对齐到已扩展的 cur.buckets，prev 不单独扩轴）。
    const cur = extendChartsAxis(
      filterChartsByMetricPaths(
        buildMetricCharts(rows, taskQuery.data.dimension, effectiveGran),
        metricPaths,
      ),
      {
        rangeStartMs: start.valueOf(),
        rangeEndMs: end.valueOf(),
        weekdays: weekdaySet,
        hours: hourSet,
        granularity: effectiveGran,
      },
    );
    if (!filter.compare) return cur;
    const prev = filterChartsByMetricPaths(
      buildMetricCharts(prevRows, taskQuery.data.dimension, effectiveGran),
      metricPaths,
    );
    return attachCompareSeries(cur, prev, offsetMs, effectiveGran);
  }, [
    rows,
    prevRows,
    taskQuery.data,
    effectiveGran,
    filter.compare,
    offsetMs,
    start,
    end,
    weekdaySet,
    hourSet,
  ]);

  // ── 导出（T4 adhoc 来源）：带 task_id + 当前大时间段 POST 建任务 ──────────
  const createExport = useCreateKpiExport();
  const handleExport = () => {
    createExport.mutate(
      {
        sourceType: 'adhoc',
        params: buildAdhocExportParams({
          taskId,
          startTime: startISO,
          endTime: endISO,
        }),
        taskName: defaultExportTaskName('adhoc'),
      },
      {
        onSuccess: () => {
          message.success(intl.formatMessage({ id: 'kpiExport.export.submitted' }));
        },
        onError: (e) => {
          message.error(
            intl.formatMessage(
              { id: 'kpiExport.export.submitFailed' },
              { reason: (e as Error)?.message ?? '' },
            ),
          );
        },
      },
    );
  };

  if (taskQuery.isLoading) {
    return (
      <Card>
        <Spin tip={intl.formatMessage({ id: 'perf.dashboard.loadingTask' })} />
      </Card>
    );
  }
  if (taskQuery.isError || !taskQuery.data) {
    return (
      <Card>
        <Alert
          type="warning"
          showIcon
          message={intl.formatMessage({ id: 'perf.dashboard.taskUnavailable' })}
          description={intl.formatMessage({ id: 'perf.dashboard.taskUnavailableDesc' })}
        />
      </Card>
    );
  }

  const task = taskQuery.data;

  return (
    <div>
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space
          size={8}
          wrap
          style={{ width: '100%', justifyContent: 'space-between' }}
        >
          <Space size={8} wrap>
            <Typography.Text strong>{task.name}</Typography.Text>
            {task.technology && <Tag color="geekblue">{task.technology.toUpperCase()}</Tag>}
            <Tag color="purple">
              {task.mode === 'continuous'
                ? intl.formatMessage({ id: 'perf.dashboard.modeContinuous' })
                : intl.formatMessage({ id: 'perf.dashboard.modeOneshot' })}
            </Tag>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {intl.formatMessage(
                { id: 'perf.dashboard.metricCount' },
                { count: task.metricPaths.length },
              )}
            </Typography.Text>
            {granularities.length > 1 && (
              <Segmented
                size="small"
                value={effectiveGran}
                onChange={(v) => setActiveGran(v as string)}
                options={granularities.map((g) => ({ label: granLabel(g), value: g }))}
              />
            )}
          </Space>
          <Button
            size="small"
            icon={<ExportOutlined />}
            loading={createExport.isPending}
            onClick={handleExport}
            title={intl.formatMessage({ id: 'kpiExport.export.tooltip' })}
          >
            {intl.formatMessage({ id: 'kpiExport.export.button' })}
          </Button>
        </Space>
      </Card>

      <Card size="small" style={{ marginBottom: 12 }}>
        <DashboardFilterBar
          value={filter}
          onChange={setFilter}
          dimension={dimension}
          dimOptions={filterOpts?.options}
          dimSelected={dimSelected}
          onDimChange={setDimSelected}
        />
      </Card>

      {truncated && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          message={intl.formatMessage({ id: 'perf.dashboard.truncated' })}
          description={intl.formatMessage(
            { id: 'perf.dashboard.truncatedDesc' },
            { limit: RESULTS_LIMIT },
          )}
        />
      )}

      {rowsLoading || (filter.compare && prevLoading) ? (
        <Card>
          <Spin tip={intl.formatMessage({ id: 'perf.dashboard.loadingResult' })} />
        </Card>
      ) : charts.length === 0 ? (
        <Card>
          <Empty
            description={
              effectiveGran
                ? intl.formatMessage(
                    { id: 'perf.dashboard.emptyGranNoData' },
                    { gran: granLabel(effectiveGran) },
                  )
                : intl.formatMessage({ id: 'perf.dashboard.emptyTaskNoData' })
            }
          />
        </Card>
      ) : (
        charts.map((c) => <ChartCard key={c.metricPath} chart={c} />)
      )}
    </div>
  );
}
