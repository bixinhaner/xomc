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

import { useEffect, useMemo, useRef, useState } from 'react';
import { useIntl } from 'react-intl';
import { Alert, App, Button, Card, Empty, Segmented, Space, Typography } from 'antd';
import { LoadingSpinner } from '@/components/LoadingSpinner';
import { ExportOutlined, LineChartOutlined, ReloadOutlined } from '@ant-design/icons';
import {
  usePmAdhocDetail,
  usePmAdhocFilterOptions,
  usePmAdhocResults,
} from '@core/hooks/api/usePmAdhoc';
import { useIndicatorCandidates } from '@core/hooks/api/usePerformance';
import { useCreateKpiExport } from '@core/hooks/api/useKpiExport';
import { useSystemTimezoneValue } from '@core/hooks/api/useSystemTimezone';
import { buildAdhocExportParams, defaultExportTaskName } from '@core/utils/kpiExportParams';
import { usePmPageStateStore } from '@core/store/pmPageStateStore';
import type { Granularity } from '@core/types/pmDashboard';
import {
  isKnownTechnology,
  technologyToDeviceType,
  useTechnologyDictionary,
} from '@core/hooks/api/useTechnologyDictionary';
import { displayAdhocTaskName } from '../adhocTaskDisplay';
import {
  buildMetricCharts,
  buildTrustedSeriesIdentities,
  ensureConfiguredMetricCharts,
  filterChartsByMetricPaths,
  filterRowsByMetricPaths,
} from './taskDashboardUtils';
import ChartCard from './ChartCard';
import DashboardFilterBar, { type DashboardFilterValue } from './DashboardFilterBar';
import {
  attachCompareSeries,
  dimSelectionToParams,
  extendChartsAxis,
} from './dashboardFilterUtils';
import {
  buildDefaultTaskDashboardFilter,
  defaultTaskDashboardRangeMode,
  buildSubmittedTaskDashboardQuery,
  buildTaskDashboardStateSnapshot,
  buildTaskDashboardTaskSwitchReset,
  PM_DASHBOARD_PAGE_KEY,
  restoredQueryDelayMs,
  restoreTaskDashboardState,
  type DashboardRangeMode,
  type TaskDashboardSubmittedQuery,
} from './taskDashboardState';

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
  const { labelForTechnology } = useTechnologyDictionary();
  // #563：筛选器按系统时区展示和序列化，与图表 X 轴统一参照系。
  const systemTimezone = useSystemTimezoneValue();
  // 粒度短标签：有对应键走语料，无键回退原值（等价旧 GRAN_LABEL[g] ?? g）。
  const granLabel = (g: string) =>
    GRAN_MSG_IDS[g] ? intl.formatMessage({ id: GRAN_MSG_IDS[g] }) : g;
  const taskQuery = usePmAdhocDetail(taskId);
  const dimension = taskQuery.data?.dimension;
  const taskDeviceType = isKnownTechnology(taskQuery.data?.technology)
    ? technologyToDeviceType(taskQuery.data.technology)
    : undefined;
  const { data: metricCandidates } = useIndicatorCandidates(taskDeviceType, {
    includeCounters: true,
    enabledOnly: false,
  });

  // ── 共用三级筛选 + 周期对比开关（本 Pane 持状态，驱动取数 + 二拉）──────
  // #563：默认范围按系统时区「当前时刻」，与图表 X 轴同一参照系。
  const restoredState = useMemo(
    () => restoreTaskDashboardState(usePmPageStateStore.getState().getPageState(PM_DASHBOARD_PAGE_KEY), systemTimezone),
    // 初次挂载恢复一次即可；后续由本组件继续托管和保存状态。
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );
  const skipNextSaveRef = useRef(false);
  const restoredAppliesToTask = restoredState.taskId === taskId;
  const [filter, setFilter] = useState<DashboardFilterValue>(() =>
    restoredAppliesToTask
      ? restoredState.filter
      : buildDefaultTaskDashboardFilter(systemTimezone),
  );
  const [rangeMode, setRangeMode] = useState<DashboardRangeMode>(() =>
    restoredAppliesToTask
      ? restoredState.rangeMode
      : { kind: 'relative', durationMs: 7 * 24 * 60 * 60 * 1000 },
  );
  const [activeGran, setActiveGran] = useState<string | undefined>(() =>
    restoredAppliesToTask ? restoredState.activeGran : undefined,
  );

  // ── PM-DASH-DIMFILTER 维度子集筛选（按维度动态显示产品/设备组/频段多选框）──────
  // dimSelected 是筛选框上选中的"分组键原值"（product 维度=product_id；device_group=DeviceGroup=<uuid>；band=Band=<值>）。
  // 切任务（taskId 变）即清空选中，避免上个任务的子集串到新任务——用 React 官方"渲染期调整 state"
  // 模式（记录上次 taskId，变化时同步重置），不放 effect 里 setState。
  const [dimSelected, setDimSelected] = useState<string[]>(() =>
    restoredAppliesToTask ? restoredState.dimSelected : [],
  );
  const [submitted, setSubmitted] = useState<TaskDashboardSubmittedQuery | null>(() =>
    restoredAppliesToTask ? restoredState.submitted : null,
  );
  const restoredSubmittedNeedsEffectiveGranularity =
    restoredAppliesToTask &&
    Boolean(restoredState.submitted) &&
    !restoredState.submitted?.granularity &&
    !restoredState.activeGran;
  const initialRestoredQueryDelayMs = restoredAppliesToTask && restoredState.submitted
    ? restoredQueryDelayMs(restoredState.savedAt, Date.now())
    : 0;
  const [restoredQueryDelayElapsed, setRestoredQueryDelayElapsed] = useState(initialRestoredQueryDelayMs === 0);
  const [resultsQueryReady, setResultsQueryReady] = useState(
    initialRestoredQueryDelayMs === 0 && !restoredSubmittedNeedsEffectiveGranularity,
  );
  useEffect(() => {
    if (initialRestoredQueryDelayMs <= 0) return undefined;
    const timer = window.setTimeout(() => setRestoredQueryDelayElapsed(true), initialRestoredQueryDelayMs);
    return () => window.clearTimeout(timer);
  }, [initialRestoredQueryDelayMs]);
  useEffect(() => {
    if (!restoredAppliesToTask || !restoredState.submitted) return;
    if (!restoredQueryDelayElapsed) return;
    if (restoredSubmittedNeedsEffectiveGranularity && !submitted?.granularity) return;
    setResultsQueryReady(true);
  }, [
    restoredAppliesToTask,
    restoredQueryDelayElapsed,
    restoredState.submitted,
    restoredSubmittedNeedsEffectiveGranularity,
    submitted?.granularity,
  ]);
  const [prevTaskId, setPrevTaskId] = useState(taskId);
  if (taskId !== prevTaskId) {
    const reset = buildTaskDashboardTaskSwitchReset(systemTimezone);
    setPrevTaskId(taskId);
    setDimSelected(reset.dimSelected);
    setFilter(reset.filter);
    setRangeMode(reset.rangeMode);
    setActiveGran(reset.activeGran);
    setSubmitted(reset.submitted);
    setResultsQueryReady(true);
  }
  const { data: filterOpts } = usePmAdhocFilterOptions(taskId, dimension);
  // 维度→入参映射（纯函数，便于单测）：product→productIds；device_group/band→objectLdns；空选不过滤。
  const { productIds, objectLdns } = dimSelectionToParams(dimension, dimSelected);
  const granularities = useMemo(
    () => taskQuery.data?.granularities ?? [],
    [taskQuery.data?.granularities],
  );
  const effectiveGran = activeGran && granularities.includes(activeGran) ? activeGran : granularities[0];

  // #599：改为「点出图才查」模式——所有条件变化只更新本地暂存 state，
  // 点「出图」按钮时把暂存条件一次性提交（提交快照驱动 usePmAdhocResults）。
  const handleQuery = () => {
    setResultsQueryReady(true);
    setSubmitted(buildSubmittedTaskDashboardQuery(filter, {
      productIds,
      objectLdns,
      systemTimezone,
      granularity: effectiveGran,
      rangeMode,
    }));
  };

  useEffect(() => {
    if (skipNextSaveRef.current) {
      skipNextSaveRef.current = false;
      return;
    }
    usePmPageStateStore.getState().savePageState(
      PM_DASHBOARD_PAGE_KEY,
      buildTaskDashboardStateSnapshot({
        taskId,
        filter,
        dimSelected,
        activeGran,
        effectiveGran,
        rangeMode,
        submitted,
      }),
    );
  }, [activeGran, dimSelected, filter, rangeMode, submitted, taskId]);

  const handleFilterChange = (next: DashboardFilterValue) => {
    const rangeChanged =
      next.range[0].valueOf() !== filter.range[0].valueOf() ||
      next.range[1].valueOf() !== filter.range[1].valueOf();
    if (rangeChanged) {
      setRangeMode({ kind: 'absolute' });
    }
    setFilter(next);
    setSubmitted(null);
    setResultsQueryReady(true);
  };

  const handleDimChange = (next: string[]) => {
    setDimSelected(next);
    setSubmitted(null);
    setResultsQueryReady(true);
  };

  const handleReset = () => {
    const resetGranularity = effectiveGran as Granularity | undefined;
    const nextFilter = resetGranularity
      ? buildDefaultTaskDashboardFilter(systemTimezone, resetGranularity)
      : buildDefaultTaskDashboardFilter(systemTimezone);
    setFilter(nextFilter);
    setRangeMode(resetGranularity
      ? defaultTaskDashboardRangeMode(systemTimezone, resetGranularity)
      : { kind: 'relative', durationMs: 7 * 24 * 60 * 60 * 1000 });
    setDimSelected([]);
    setActiveGran(undefined);
    setSubmitted(null);
    setResultsQueryReady(true);
    skipNextSaveRef.current = true;
    usePmPageStateStore.getState().clearPageState(PM_DASHBOARD_PAGE_KEY);
  };

  // 大时间段驱动取数（后端按 time 窗口 + weekdays/hours 过滤）。仪表盘取较多结果行用于画线。
  const { data: rowsResp, isLoading: rowsLoading } = usePmAdhocResults(
    submitted && resultsQueryReady ? taskId : undefined,
    {
      limit: RESULTS_LIMIT,
      startTime: submitted?.startISO,
      endTime: submitted?.endISO,
      productIds: submitted?.productIds,
      objectLdns: submitted?.objectLdns,
      weekdays: submitted?.weekdays,
      hours: submitted?.hours,
      includePartial: true,
    },
  );
  const rawRows = [...(rowsResp?.rows ?? []), ...(rowsResp?.progressRows ?? [])];
  // 周期对比开关打开时再拉一次上一周期（同任务、上一周期窗口、同 weekdays/hours）。
  const { data: prevResp, isLoading: prevLoading } = usePmAdhocResults(
    submitted?.compare && resultsQueryReady ? taskId : undefined,
    {
      limit: RESULTS_LIMIT,
      startTime: submitted?.prevStartISO,
      endTime: submitted?.prevEndISO,
      productIds: submitted?.productIds,
      objectLdns: submitted?.objectLdns,
      weekdays: submitted?.weekdays,
      hours: submitted?.hours,
    },
  );
  const rawPrevRows = prevResp?.rows ?? [];

  // 触顶提示：后端用 limit+1 判断是否截断，前端直接信任 truncated 标记。
  const truncated = rowsResp?.truncated ?? false;

  useEffect(() => {
    if (!effectiveGran) return;
    const granularity = effectiveGran as Granularity;
    const nextRangeMode = rangeMode.kind === 'relative'
      ? defaultTaskDashboardRangeMode(systemTimezone, granularity)
      : rangeMode;
    if (rangeMode.kind === 'relative') {
      const nextRelativeRangeMode = defaultTaskDashboardRangeMode(systemTimezone, granularity);
      const nextRange = buildDefaultTaskDashboardFilter(systemTimezone, granularity).range;
      setFilter((cur) => (
        cur.range[0].valueOf() === nextRange[0].valueOf() && cur.range[1].valueOf() === nextRange[1].valueOf()
          ? cur
          : { ...cur, range: nextRange }
      ));
      setRangeMode((cur) => (
        cur.kind === 'relative' && cur.durationMs === nextRelativeRangeMode.durationMs
          ? cur
          : nextRelativeRangeMode
      ));
    }
    if (submitted) {
      const rebuiltSubmitted = buildSubmittedTaskDashboardQuery(filter, {
        productIds: submitted.productIds,
        objectLdns: submitted.objectLdns,
        systemTimezone,
        granularity,
        rangeMode: nextRangeMode,
      });
      if (
        submitted.granularity !== rebuiltSubmitted.granularity ||
        submitted.startISO !== rebuiltSubmitted.startISO ||
        submitted.endISO !== rebuiltSubmitted.endISO ||
        submitted.prevStartISO !== rebuiltSubmitted.prevStartISO ||
        submitted.prevEndISO !== rebuiltSubmitted.prevEndISO ||
        submitted.rangeStartMs !== rebuiltSubmitted.rangeStartMs ||
        submitted.rangeEndMs !== rebuiltSubmitted.rangeEndMs
      ) {
        setSubmitted(rebuiltSubmitted);
      }
    }
  }, [effectiveGran, filter, rangeMode, submitted, systemTimezone]);
  const activeProgress = useMemo(() => {
    if (effectiveGran !== 'daily' && effectiveGran !== 'weekly') return undefined;
    return (rowsResp?.periodProgress ?? [])
      .filter((progress) => progress.granularity === effectiveGran)
      .sort((left, right) => right.windowStart.localeCompare(left.windowStart))[0];
  }, [effectiveGran, rowsResp?.periodProgress]);
  const activeCoverage = activeProgress && (activeProgress.expectedSlots ?? 0) > 0
    ? Math.min(
        100,
        Math.round(
          ((activeProgress.receivedSlots ?? 0) / (activeProgress.expectedSlots ?? 1)) * 100,
        ),
      )
    : 0;
  const chartLocale = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';
  const metricDisplayNames = useMemo(() => {
    const map = new Map<string, string>();
    for (const metric of metricCandidates ?? []) {
      const name =
        chartLocale === 'en-US'
          ? metric.enName || metric.cnName || metric.name
          : metric.cnName || metric.enName || metric.name;
      map.set(metric.id, name);
    }
    return map;
  }, [chartLocale, metricCandidates]);
  const seriesLabels = useMemo(
    () => ({
      network: intl.formatMessage({ id: 'perf.adhoc.colObject.network' }),
      band: intl.formatMessage({ id: 'perf.adhoc.colObject.band' }),
      deviceGroup: intl.formatMessage({ id: 'perf.adhoc.colObject.deviceGroup' }),
      product: intl.formatMessage({ id: 'perf.adhoc.colObject.product' }),
      aggregateGroup: intl.formatMessage({ id: 'perf.adhoc.colObject.aggregateGroup' }),
    }),
    [intl],
  );

  const charts = useMemo(() => {
    if (!taskQuery.data || !effectiveGran || !submitted) return [];
    // #192：右侧图表严格按任务 metric_paths 展示。即使后端历史行或异常返回带出额外指标，
    // 也不能生成配置外图表；先滤原始行，避免额外指标污染固定 legend 全集。
    const taskMetricPaths = taskQuery.data.metricPaths;
    const visibleRows = filterRowsByMetricPaths(rawRows, taskMetricPaths);
    const visiblePrevRows = filterRowsByMetricPaths(rawPrevRows, taskMetricPaths);
    const selectedSeriesKeys =
      taskQuery.data.dimension === 'product'
        ? submitted.productIds
        : taskQuery.data.dimension === 'device_group' || taskQuery.data.dimension === 'band'
          ? submitted.objectLdns
          : undefined;
    const trustedSeries = buildTrustedSeriesIdentities(
      taskQuery.data.dimension,
      {
        deviceSns: taskQuery.data.deviceSns,
        objectLdns: taskQuery.data.objectLdns,
        filterOptions: filterOpts?.options,
        selectedKeys: selectedSeriesKeys,
      },
      seriesLabels,
    );
    // #599：后端已按 weekdays/hours 过滤，前端只需扩轴（轴刻度仍按完整范围铺、再套星期/小时剔除空桶）。
    const weekdaySet = new Set(submitted.weekdays);
    const hourSet = new Set(submitted.hours);
    const cur = extendChartsAxis(
      ensureConfiguredMetricCharts(
        filterChartsByMetricPaths(
          buildMetricCharts(
            visibleRows,
            taskQuery.data.dimension,
            effectiveGran,
            chartLocale,
            metricDisplayNames,
          ),
          taskMetricPaths,
        ),
        taskMetricPaths,
        trustedSeries,
        metricDisplayNames,
      ),
      {
        rangeStartMs: submitted.rangeStartMs,
        rangeEndMs: submitted.rangeEndMs,
        weekdays: weekdaySet,
        hours: hourSet,
        granularity: effectiveGran,
        systemTimezone,
      },
    );
    if (!submitted.compare) return cur;
    const prev = ensureConfiguredMetricCharts(
      filterChartsByMetricPaths(
        buildMetricCharts(
          visiblePrevRows,
          taskQuery.data.dimension,
          effectiveGran,
          chartLocale,
          metricDisplayNames,
        ),
        taskMetricPaths,
      ),
      taskMetricPaths,
      trustedSeries,
      metricDisplayNames,
    );
    return attachCompareSeries(cur, prev, submitted.offsetMs, effectiveGran);
  }, [
    rawRows,
    rawPrevRows,
    taskQuery.data,
    effectiveGran,
    submitted,
    chartLocale,
    metricDisplayNames,
    filterOpts?.options,
    seriesLabels,
    systemTimezone,
  ]);

  // ── 导出：性能仪表盘任务图表来源，复用 adhoc 结果参数和后端 CSV 链路 ──────────
  const createExport = useCreateKpiExport();
  const handleExport = () => {
    // #599：导出与出图同口径——用提交态的筛选快照（未出图时用当前 filter）。
    const exportSubmitted = submitted ?? buildSubmittedTaskDashboardQuery(filter, {
      productIds,
      objectLdns,
      systemTimezone,
      granularity: effectiveGran,
      rangeMode,
    });
    const exportParams = buildAdhocExportParams({
      taskId,
      startTime: exportSubmitted.startISO,
      endTime: exportSubmitted.endISO,
      productIds: exportSubmitted.productIds,
      objectLdns: exportSubmitted.objectLdns,
      weekdays: exportSubmitted.weekdays,
      hours: exportSubmitted.hours,
    });
    createExport.mutate(
      {
        sourceType: 'pm_dashboard',
        params: exportParams,
        taskName: defaultExportTaskName('pm_dashboard', new Date(), {
          prefixLabel: intl.formatMessage({ id: 'kpiExport.fileName.prefix' }),
          sourceLabel: intl.formatMessage({ id: 'kpiExport.source.pmDashboard' }),
          subjectName: taskQuery.data?.name,
        }),
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
        <LoadingSpinner tip={intl.formatMessage({ id: 'perf.dashboard.loadingTask' })} />
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
  const handleGranularityChange = (value: string | number) => {
    const next = String(value);
    const granularity = next as Granularity;
    setActiveGran(next);
    if (rangeMode.kind === 'relative') {
      setFilter((cur) => ({
        ...cur,
        range: buildDefaultTaskDashboardFilter(systemTimezone, granularity).range,
      }));
      setRangeMode(defaultTaskDashboardRangeMode(systemTimezone, granularity));
    }
    setSubmitted(null);
    setResultsQueryReady(true);
  };

  return (
    <div>
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space
          size={8}
          wrap
          style={{ width: '100%', justifyContent: 'space-between' }}
        >
          <Space size={8} wrap>
            <Typography.Text strong>{displayAdhocTaskName(task, labelForTechnology)}</Typography.Text>
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
                onChange={handleGranularityChange}
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
          onChange={handleFilterChange}
          dimension={dimension}
          dimOptions={filterOpts?.options}
          dimSelected={dimSelected}
          onDimChange={handleDimChange}
        />
        <Space style={{ marginTop: 8 }}>
          <Button
            type="primary"
            icon={<LineChartOutlined />}
            onClick={handleQuery}
            loading={rowsLoading}
          >
            {intl.formatMessage({ id: 'perf.dashboard.btnPlot' })}
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={handleReset}
          >
            {intl.formatMessage({ id: 'common.reset' })}
          </Button>
        </Space>
      </Card>

      {activeProgress && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          message={intl.formatMessage({ id: 'perf.dashboard.partialPeriod' })}
          description={intl.formatMessage(
            { id: 'perf.dashboard.partialPeriodDesc' },
            {
              received: activeProgress.receivedSlots ?? 0,
              expected: activeProgress.expectedSlots ?? 0,
              coverage: activeCoverage,
              revision: activeProgress.revision ?? 1,
              from: activeProgress.versionEffectiveFrom ?? '-',
              to: activeProgress.versionEffectiveTo ?? '-',
            },
          )}
        />
      )}

      {rowsResp?.progressState === 'unavailable' && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          message={intl.formatMessage({ id: 'perf.dashboard.progressUnavailable' })}
          description={intl.formatMessage({ id: 'perf.dashboard.progressUnavailableDesc' })}
        />
      )}

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

      {rowsLoading || (submitted?.compare && prevLoading) ? (
        <Card>
          <LoadingSpinner tip={intl.formatMessage({ id: 'perf.dashboard.loadingResult' })} />
        </Card>
      ) : !submitted ? (
        <Card>
          <Empty
            description={intl.formatMessage({ id: 'perf.dashboard.clickQueryToStart' })}
          />
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
