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

import { useEffect, useMemo, useState } from 'react';
import { useIntl } from 'react-intl';
import { Alert, App, Button, Card, Empty, Segmented, Space, Tag, Typography } from 'antd';
import { LoadingSpinner } from '@/components/LoadingSpinner';
import { ExportOutlined, LineChartOutlined } from '@ant-design/icons';
import {
  usePmAdhocDetail,
  usePmAdhocFilterOptions,
  usePmAdhocResults,
} from '@core/hooks/api/usePmAdhoc';
import { useCreateKpiExport } from '@core/hooks/api/useKpiExport';
import { useSystemTimezoneValue } from '@core/hooks/api/useSystemTimezone';
import { nowInSystemTimezone, toSystemTimezoneRFC3339 } from '@core/utils/systemTime';
import { buildAdhocExportParams, defaultExportTaskName } from '@core/utils/kpiExportParams';
import { useTechnologyDictionary } from '@core/hooks/api/useTechnologyDictionary';
import { displayAdhocTaskName } from '../adhocTaskDisplay';
import { buildMetricCharts } from './taskDashboardUtils';
import ChartCard from './ChartCard';
import DashboardFilterBar, { type DashboardFilterValue } from './DashboardFilterBar';
import {
  ALL_HOURS,
  ALL_WEEKDAYS,
  attachCompareSeries,
  dimSelectionToParams,
  extendChartsAxis,
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
  const { labelForTechnology } = useTechnologyDictionary();
  // #563：筛选器按系统时区展示和序列化，与图表 X 轴统一参照系。
  const systemTimezone = useSystemTimezoneValue();
  // 粒度短标签：有对应键走语料，无键回退原值（等价旧 GRAN_LABEL[g] ?? g）。
  const granLabel = (g: string) =>
    GRAN_MSG_IDS[g] ? intl.formatMessage({ id: GRAN_MSG_IDS[g] }) : g;
  const taskQuery = usePmAdhocDetail(taskId);
  const dimension = taskQuery.data?.dimension;

  // ── 共用三级筛选 + 周期对比开关（本 Pane 持状态，驱动取数 + 二拉）──────
  // #563：默认范围按系统时区「当前时刻」，与图表 X 轴同一参照系。
  const [filter, setFilter] = useState<DashboardFilterValue>(() => {
    const now = nowInSystemTimezone(systemTimezone);
    return {
      range: [now.subtract(7, 'day'), now],
      weekdays: [...ALL_WEEKDAYS],
      hours: [...ALL_HOURS],
      compare: false,
    };
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

  // #599：改为「点出图才查」模式——所有条件变化只更新本地暂存 state，
  // 点「出图」按钮时把暂存条件一次性提交（提交快照驱动 usePmAdhocResults）。
  const [submitted, setSubmitted] = useState<{
    startISO: string;
    endISO: string;
    productIds?: string[];
    objectLdns?: string[];
    weekdays: number[];
    hours: number[];
    compare: boolean;
    offsetMs: number;
    prevStartISO: string;
    prevEndISO: string;
    rangeStartMs: number;
    rangeEndMs: number;
  } | null>(null);

  const handleQuery = () => {
    const [s, e] = filter.range;
    const sISO = toSystemTimezoneRFC3339(s, systemTimezone) ?? s.toISOString();
    const eISO = toSystemTimezoneRFC3339(e, systemTimezone) ?? e.toISOString();
    const [ps, pe] = previousWindow(filter.range);
    setSubmitted({
      startISO: sISO,
      endISO: eISO,
      productIds,
      objectLdns,
      weekdays: filter.weekdays,
      hours: filter.hours,
      compare: filter.compare,
      offsetMs: e.valueOf() - s.valueOf(),
      prevStartISO: toSystemTimezoneRFC3339(ps, systemTimezone) ?? ps.toISOString(),
      prevEndISO: toSystemTimezoneRFC3339(pe, systemTimezone) ?? pe.toISOString(),
      rangeStartMs: s.valueOf(),
      rangeEndMs: e.valueOf(),
    });
  };

  // #599：选中任务后自动触发一次出图（用当前默认筛选条件查一次）。
  // taskId 变化时（含首次加载）自动提交，用户不用手动点「出图」就能看到图。
  useEffect(() => {
    handleQuery();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [taskId]);

  // 大时间段驱动取数（后端按 time 窗口 + weekdays/hours 过滤）。仪表盘取较多结果行用于画线。
  const { data: rowsResp, isLoading: rowsLoading } = usePmAdhocResults(
    submitted ? taskId : undefined,
    {
      limit: RESULTS_LIMIT,
      startTime: submitted?.startISO,
      endTime: submitted?.endISO,
      productIds: submitted?.productIds,
      objectLdns: submitted?.objectLdns,
      weekdays: submitted?.weekdays,
      hours: submitted?.hours,
    },
  );
  const rawRows = rowsResp?.rows ?? [];
  // 周期对比开关打开时再拉一次上一周期（同任务、上一周期窗口、同 weekdays/hours）。
  const { data: prevResp, isLoading: prevLoading } = usePmAdhocResults(
    submitted?.compare ? taskId : undefined,
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

  // 触顶提示：结果接口 LIMIT 上限，触顶可能截断 → 给可见提示，不静默。
  const truncated = rawRows.length >= RESULTS_LIMIT;

  const granularities = useMemo(
    () => taskQuery.data?.granularities ?? [],
    [taskQuery.data?.granularities],
  );
  const [activeGran, setActiveGran] = useState<string | undefined>(undefined);
  const effectiveGran = activeGran && granularities.includes(activeGran) ? activeGran : granularities[0];
  const chartLocale = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';

  const charts = useMemo(() => {
    if (!taskQuery.data || !effectiveGran || !submitted) return [];
    // #185：任务配置指标集不再等同于最终输出指标集；后端结果已按任务指标 ∪ 启用指标收口。
    // 这里直接画结果里真实出现的指标，避免把启用指标再次按旧任务配置裁掉。
    // #599：后端已按 weekdays/hours 过滤，前端只需扩轴（轴刻度仍按完整范围铺、再套星期/小时剔除空桶）。
    const weekdaySet = new Set(submitted.weekdays);
    const hourSet = new Set(submitted.hours);
    const cur = extendChartsAxis(
      buildMetricCharts(rawRows, taskQuery.data.dimension, effectiveGran, chartLocale),
      {
        rangeStartMs: submitted.rangeStartMs,
        rangeEndMs: submitted.rangeEndMs,
        weekdays: weekdaySet,
        hours: hourSet,
        granularity: effectiveGran,
      },
    );
    if (!submitted.compare) return cur;
    const prev = buildMetricCharts(rawPrevRows, taskQuery.data.dimension, effectiveGran, chartLocale);
    return attachCompareSeries(cur, prev, submitted.offsetMs, effectiveGran);
  }, [
    rawRows,
    rawPrevRows,
    taskQuery.data,
    effectiveGran,
    submitted,
    chartLocale,
  ]);

  // ── 导出：性能仪表盘任务图表来源，复用 adhoc 结果参数和后端 CSV 链路 ──────────
  const createExport = useCreateKpiExport();
  const handleExport = () => {
    // #599：导出与出图同口径——用提交态的筛选快照（未出图时用当前 filter）。
    const [s, e] = filter.range;
    const exportStart = submitted?.startISO ?? (toSystemTimezoneRFC3339(s, systemTimezone) ?? s.toISOString());
    const exportEnd = submitted?.endISO ?? (toSystemTimezoneRFC3339(e, systemTimezone) ?? e.toISOString());
    const exportParams = buildAdhocExportParams({
      taskId,
      startTime: exportStart,
      endTime: exportEnd,
    });
    // #599：weekdays/hours 传入导出 params（后端导出时按同口径过滤）。
    const wd = submitted?.weekdays ?? filter.weekdays;
    const hr = submitted?.hours ?? filter.hours;
    if (wd.length > 0 && wd.length < 7) exportParams.weekdays = wd;
    if (hr.length > 0 && hr.length < 24) exportParams.hours = hr;
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
            {task.technology && <Tag color="geekblue">{labelForTechnology(task.technology)}</Tag>}
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
        <Button
          type="primary"
          icon={<LineChartOutlined />}
          onClick={handleQuery}
          loading={rowsLoading}
          style={{ marginTop: 8 }}
        >
          {intl.formatMessage({ id: 'perf.dashboard.btnPlot' })}
        </Button>
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
