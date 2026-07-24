/**
 * T-0188 性能仪表盘 · 页签2 设备列表（独立即席查看，不依赖任何聚合任务）。
 *
 * 交互：
 *   - 制式 Segmented（默认 LTE；ENB/GNB/GSM）。
 *   - 设备多选（最多 50，复用 KPIQuery/components/DevicePickerModal；超限拦截提示并截断）。
 *   - 指标可换（最多 50，复用共享件 components/MetricPickerModal，initialDeviceType 随制式）；
 *     默认集 = 选中制式的内置任务指标集（usePmAdhocList isBuiltin，按 technology 找一个取 metricPaths）。
 *     用户未手动改过指标时，切制式默认集随之切换；手动改过则保留用户选择。
 *   - 粒度选择（默认 15min）。
 *   - 共用三级筛选（DashboardFilterBar）：大时间段（默认近 7 天）+ 星期多选 + 小时段多选 + 周期对比开关（T-0189）。
 *   - 出图：取数 useAggregatedMetricsByDevices → 星期/小时段前端筛 → buildDeviceMetricCharts → 每指标一张 ChartCard（每设备一条线）。
 *   - 周期对比开关打开：再拉上一周期窗口数据，套同口径星期/小时段，叠加虚线（T-0189）。
 *
 * T-0193：选设备后可下钻勾选小区/PLMN（CellDrilldownSelector），即席纯前端过滤
 *   （在已取的聚合行里筛命中行，不落库）；出图按「设备+小区+PLMN」分线（deviceListUtils）。
 */

import { useEffect, useMemo, useState } from 'react';
import { useIntl } from 'react-intl';
import {
  Alert,
  App,
  Button,
  Card,
  Empty,
  Form,
  Input,
  Radio,
  Segmented,
  Space,
  Tag,
  Typography,
} from 'antd';
import { LoadingSpinner } from '@/components/LoadingSpinner';
import { ReloadOutlined, LineChartOutlined, ExportOutlined } from '@ant-design/icons';
import {
  useAggregatedMetricsByDevices,
  useMetricObjectsByDevices,
} from '@core/hooks/api/usePmQuery';
import { usePmAdhocList } from '@core/hooks/api/usePmAdhoc';
import { useCreateKpiExport } from '@core/hooks/api/useKpiExport';
import { useSystemTimezoneValue } from '@core/hooks/api/useSystemTimezone';
import type { DeviceType } from '@core/types/indicatorLibrary';
import type { CreateKpiExportInput } from '@core/types/kpiExport';
import type { Granularity } from '@core/types/pmDashboard';
import { PM_QUERY_SELECTION_LIMIT } from '@/constants/pmQueryLimits';
import DevicePickerModal from '../KPIQuery/components/DevicePickerModal';
import MetricPickerModal from '@/components/MetricPickerModal';
import ChartCard from './ChartCard';
import { buildDeviceMetricCharts, filterRowsByObjectLdns } from './deviceListUtils';
import CellDrilldownSelector from './CellDrilldownSelector';
import { getEffectiveLdns, type CellSelection } from './cellDrilldownUtils';
import DashboardFilterBar, { type DashboardFilterValue } from './DashboardFilterBar';
import {
  ALL_HOURS,
  ALL_WEEKDAYS,
  attachCompareSeries,
  previousWindow,
} from './dashboardFilterUtils';
import {
  actualRangeFromMeta,
  buildDeviceViewRequestTimeWindow,
  defaultRangeForGranularity,
  toDeviceViewRequestRFC3339,
} from './deviceListPaneTimeUtils';
import {
  buildDashboardExportParams,
  validateDashboardExportSelection,
  defaultExportTaskName,
  type DashboardExportSelection,
} from '@core/utils/kpiExportParams';

// 制式 ↔ 设备类型 ↔ 内置任务 technology 三者映射。
type Tech = 'lte' | 'nr' | 'gsm';

const TECH_OPTIONS: { label: string; value: Tech }[] = [
  { label: 'LTE', value: 'lte' },
  { label: 'NR', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
];

const TECH_TO_DEVICE_TYPE: Record<Tech, DeviceType> = {
  lte: 'ENB',
  nr: 'GNB',
  gsm: 'GSM',
};

export function buildDeviceViewExportInput(
  selection: DashboardExportSelection,
  prefixLabel: string,
  sourceLabel: string,
  now: Date = new Date(),
): CreateKpiExportInput {
  return {
    sourceType: 'device_view',
    params: buildDashboardExportParams(selection),
    taskName: defaultExportTaskName('device_view', now, { prefixLabel, sourceLabel }),
  };
}

// 粒度选项语料键（label 走 i18n，value 不变）。
const GRANULARITY_MSG_IDS: { id: string; value: Granularity }[] = [
  { id: 'perf.dashboard.granular15min', value: '15min' },
  { id: 'perf.dashboard.granularHourly', value: 'hourly' },
  { id: 'perf.dashboard.granularDaily', value: 'daily' },
  { id: 'perf.dashboard.granularWeekly', value: 'weekly' },
  { id: 'perf.dashboard.granularMonthly', value: 'monthly' },
];

export function isDeviceViewDeviceSelectionOverLimit(deviceSns: string[]): boolean {
  return deviceSns.length > PM_QUERY_SELECTION_LIMIT;
}

export function isDeviceViewMetricSelectionOverLimit(metricPaths: string[]): boolean {
  return metricPaths.length > PM_QUERY_SELECTION_LIMIT;
}

export default function DeviceListPane() {
  const intl = useIntl();
  const { message } = App.useApp();
  const systemTimezone = useSystemTimezoneValue();

  const granularityOptions = useMemo(
    () =>
      GRANULARITY_MSG_IDS.map(({ id, value }) => ({
        label: intl.formatMessage({ id }),
        value,
      })),
    [intl],
  );

  // ── 选择条件 ───────────────────────────────────────────────────────
  const [tech, setTech] = useState<Tech>('lte');
  const [deviceSns, setDeviceSns] = useState<string[]>([]);
  // T-0193 下钻：每设备选中的小区/PLMN 子集（缺席=全选不过滤）。
  const [cellSel, setCellSel] = useState<CellSelection>({});
  const [metricPaths, setMetricPaths] = useState<string[]>([]);
  // 用户是否手动改过指标——改过则切制式不再覆盖默认集。
  const [metricsTouched, setMetricsTouched] = useState(false);
  const [granularity, setGranularity] = useState<Granularity>('15min');
  const [rangeTouched, setRangeTouched] = useState(false);
  // 共用三级筛选 + 周期对比开关（大时间段 + 星期 + 小时段 + 对比）。
  const [filter, setFilter] = useState<DashboardFilterValue>(() => ({
    range: defaultRangeForGranularity('15min', systemTimezone),
    weekdays: [...ALL_WEEKDAYS],
    hours: [...ALL_HOURS],
    compare: false,
  }));
  const [defaultRangeKey, setDefaultRangeKey] = useState<{
    granularity: Granularity;
    systemTimezone?: string;
  }>({ granularity: '15min', systemTimezone });
  const [devicePickerOpen, setDevicePickerOpen] = useState(false);
  const [metricPickerOpen, setMetricPickerOpen] = useState(false);

  useEffect(() => {
    if (
      rangeTouched ||
      (defaultRangeKey.granularity === granularity && defaultRangeKey.systemTimezone === systemTimezone)
    ) {
      return;
    }
    setDefaultRangeKey({ granularity, systemTimezone });
    setFilter((cur) => ({
      ...cur,
      range: defaultRangeForGranularity(granularity, systemTimezone),
    }));
  }, [defaultRangeKey.granularity, defaultRangeKey.systemTimezone, granularity, rangeTouched, systemTimezone]);

  // ── 默认指标集：选中制式的内置任务指标集（单一真相源）─────────────────
  const { data: builtinTasks = [] } = usePmAdhocList({ isBuiltin: true });
  const defaultMetricPaths = useMemo(() => {
    const t = builtinTasks.find((bt) => (bt.technology ?? '').toLowerCase() === tech);
    return t?.metricPaths ?? [];
  }, [builtinTasks, tech]);

  // 用户未手动改过指标时，默认集随制式切换。
  useEffect(() => {
    if (!metricsTouched) {
      setMetricPaths(defaultMetricPaths);
    }
  }, [defaultMetricPaths, metricsTouched]);

  // ── 取数（提交快照，避免每次条件变动即查询）──────────────────────────
  const [submitted, setSubmitted] = useState<{
    deviceSns: string[];
    metricPaths: string[];
    granularity: Granularity;
    startTime: string;
    endTime: string;
    weekdays: number[];
    hours: number[];
    compare: boolean;
    offsetMs: number;
    prevStartTime: string;
    prevEndTime: string;
    // T-0193：提交时定格的小区/PLMN 白名单（空=不过滤）。
    allowedLdns: string[];
  } | null>(null);

  // 下钻选择器与有效白名单计算共用的「按设备小区清单」（react-query 与选择器内部同 key 去重，无额外请求）。
  const { byDevice: objectsByDevice } = useMetricObjectsByDevices(deviceSns, tech);

  const baseParams = useMemo(() => {
    if (!submitted) return null;
    return {
      granularity: submitted.granularity,
      metricPaths: submitted.metricPaths,
      startTime: submitted.startTime,
      endTime: submitted.endTime,
      limit: 5000,
      fillEmpty: true,
      // #599：星期/小时段后端过滤（全选不传 = 不过滤，向后兼容）。
      weekdays: submitted.weekdays.length < 7 ? submitted.weekdays : undefined,
      hours: submitted.hours.length < 24 ? submitted.hours : undefined,
    };
  }, [submitted]);

  const {
    data: rawRows = [],
    meta: currentMeta,
    total: rawTotal,
    truncated,
    isLoading,
    isFetching,
    errors,
    refetch,
  } = useAggregatedMetricsByDevices(
    baseParams ?? { granularity: '15min', metricPaths: [], startTime: undefined, endTime: undefined },
    submitted?.deviceSns ?? [],
    Boolean(baseParams),
  );

  // 周期对比：上一周期窗口同样取数（同设备/指标/粒度，窗口换为 previousWindow）。
  const actualRange = useMemo(() => actualRangeFromMeta(currentMeta), [currentMeta]);
  const prevParams = useMemo(() => {
    if (!submitted || !submitted.compare) return null;
    const actualPrevRange = actualRange ? previousWindow(actualRange) : null;
    return {
      granularity: submitted.granularity,
      metricPaths: submitted.metricPaths,
      startTime: actualPrevRange
        ? toDeviceViewRequestRFC3339(actualPrevRange[0], systemTimezone)
        : submitted.prevStartTime,
      endTime: actualPrevRange
        ? toDeviceViewRequestRFC3339(actualPrevRange[1], systemTimezone)
        : submitted.prevEndTime,
      limit: 5000,
      fillEmpty: true,
      // #599：周期对比同口径传 weekdays/hours。
      weekdays: submitted.weekdays.length < 7 ? submitted.weekdays : undefined,
      hours: submitted.hours.length < 24 ? submitted.hours : undefined,
    };
  }, [actualRange, submitted, systemTimezone]);

  const {
    data: rawPrevRows = [],
    isFetching: prevFetching,
  } = useAggregatedMetricsByDevices(
    prevParams ?? { granularity: '15min', metricPaths: [], startTime: undefined, endTime: undefined },
    prevParams ? (submitted?.deviceSns ?? []) : [],
    Boolean(prevParams),
  );

  useEffect(() => {
    if (errors.length > 0) {
      const first = errors[0] as Error;
      message.error(
        intl.formatMessage(
          { id: 'perf.dashboard.queryFailed' },
          {
            msg:
              first?.message ??
              intl.formatMessage({ id: 'perf.dashboard.queryUnknownError' }),
          },
        ),
      );
    }
  }, [errors, message, intl]);

  // 星期/小时段已由后端过滤（#599），前端只需按小区/PLMN 白名单即席过滤 + 转置分线。
  const charts = useMemo(() => {
    if (!submitted) return [];
    // T-0193：按小区/PLMN 白名单即席过滤，再转置分线。
    const curRows = filterRowsByObjectLdns(rawRows, submitted.allowedLdns);
    // #88：设备性能查看不再由前端按请求范围生成桶轴；后端已按完整时间桶补齐，
    // 前端只使用后端返回的 startTime 集合画图。
    const cur = buildDeviceMetricCharts(curRows, submitted.granularity);
    if (!submitted.compare) return cur;
    const prevFilteredRows = filterRowsByObjectLdns(rawPrevRows, submitted.allowedLdns);
    const prev = buildDeviceMetricCharts(prevFilteredRows, submitted.granularity);
    const compareOffsetMs = actualRange
      ? actualRange[1].valueOf() - actualRange[0].valueOf()
      : submitted.offsetMs;
    return attachCompareSeries(cur, prev, compareOffsetMs, submitted.granularity);
  }, [actualRange, rawRows, rawPrevRows, submitted]);

  // ── 行为 ───────────────────────────────────────────────────────────
  const handleTechChange = (v: Tech) => {
    setTech(v);
    // 切制式清空已选设备（设备制式与图制式应一致），指标默认集由 effect 随制式切换。
    setDeviceSns([]);
    setCellSel({}); // 设备清空 → 下钻选择重置（全选）。
  };

  const handleGranularityChange = (next: Granularity) => {
    setGranularity(next);
  };

  const handleFilterChange = (next: DashboardFilterValue) => {
    if (
      !next.range[0].isSame(filter.range[0]) ||
      !next.range[1].isSame(filter.range[1])
    ) {
      setRangeTouched(true);
    }
    setFilter(next);
  };

  // ── 导出（T4 dashboard 来源）：带当前筛选 POST 建任务，不卡页面 ──────────
  const createExport = useCreateKpiExport();

  // 组装当前筛选快照（与 handleQuery 同口径：设备/指标/粒度/时间/小区下钻白名单/星期/小时段）。
  // A1：下钻定格的小区/PLMN 白名单一并带进导出（复用 handleQuery 的 getEffectiveLdns，空=不过滤）。
  const buildExportSelection = (): DashboardExportSelection => {
    const [start, end] = filter.range;
    const actualExportRange = submitted ? actualRange : null;
    return {
      technology: tech,
      deviceSns: submitted?.deviceSns ?? deviceSns,
      metricPaths: submitted?.metricPaths ?? metricPaths,
      granularity: submitted?.granularity ?? granularity,
      startTime: actualExportRange
        ? toDeviceViewRequestRFC3339(actualExportRange[0], systemTimezone)
        : submitted?.startTime ?? toDeviceViewRequestRFC3339(start, systemTimezone),
      endTime: actualExportRange
        ? toDeviceViewRequestRFC3339(actualExportRange[1], systemTimezone)
        : submitted?.endTime ?? toDeviceViewRequestRFC3339(end, systemTimezone),
      objectLdns: submitted?.allowedLdns ?? getEffectiveLdns(cellSel, objectsByDevice),
      // #599：导出与出图同口径。
      weekdays: submitted?.weekdays ?? filter.weekdays,
      hours: submitted?.hours ?? filter.hours,
    };
  };

  const handleExport = () => {
    const sel = buildExportSelection();
    const missing = validateDashboardExportSelection(sel);
    if (missing) {
      message.warning(intl.formatMessage({ id: missing }));
      return;
    }
    createExport.mutate(
      buildDeviceViewExportInput(
        sel,
        intl.formatMessage({ id: 'kpiExport.fileName.prefix' }),
        intl.formatMessage({ id: 'kpiExport.source.deviceView' }),
      ),
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

  const handleQuery = () => {
    if (deviceSns.length === 0) {
      message.warning(intl.formatMessage({ id: 'perf.dashboard.selectAtLeastOneDevice' }));
      return;
    }
    if (isDeviceViewDeviceSelectionOverLimit(deviceSns)) {
      message.warning(intl.formatMessage(
        { id: 'perf.picker.deviceLimitExceeded' },
        { max: PM_QUERY_SELECTION_LIMIT, count: deviceSns.length },
      ));
      return;
    }
    if (metricPaths.length === 0) {
      message.warning(intl.formatMessage({ id: 'perf.dashboard.selectAtLeastOneMetric' }));
      return;
    }
    if (isDeviceViewMetricSelectionOverLimit(metricPaths)) {
      message.warning(intl.formatMessage(
        { id: 'perf.picker.metricLimitExceeded' },
        { max: PM_QUERY_SELECTION_LIMIT, count: metricPaths.length },
      ));
      return;
    }
    const [start, end] = filter.range;
    const requestWindow = buildDeviceViewRequestTimeWindow(filter.range, systemTimezone);
    setSubmitted({
      deviceSns,
      metricPaths,
      granularity,
      startTime: requestWindow.startTime,
      endTime: requestWindow.endTime,
      weekdays: filter.weekdays,
      hours: filter.hours,
      compare: filter.compare,
      offsetMs: end.valueOf() - start.valueOf(),
      prevStartTime: requestWindow.prevStartTime,
      prevEndTime: requestWindow.prevEndTime,
      // 定格当前下钻白名单（空=全选不过滤）。
      allowedLdns: getEffectiveLdns(cellSel, objectsByDevice),
    });
  };

  return (
    <div style={{ height: 'calc(100vh - 190px)', overflow: 'auto' }}>
      <Card size="small" style={{ marginBottom: 12 }}>
        <Form layout="vertical" size="middle">
          <Space wrap size="middle" align="start">
            <Form.Item
              label={intl.formatMessage({ id: 'perf.dashboard.fieldTech' })}
              style={{ marginBottom: 0 }}
            >
              <Segmented
                value={tech}
                onChange={(v) => handleTechChange(v as Tech)}
                options={TECH_OPTIONS}
              />
            </Form.Item>

            <Form.Item
              label={intl.formatMessage({ id: 'perf.dashboard.fieldDevice' })}
              style={{ marginBottom: 0 }}
            >
              <Space.Compact style={{ width: 360 }}>
                <Input
                  readOnly
                  value={
                    deviceSns.length === 0
                      ? ''
                      : intl.formatMessage(
                          { id: 'perf.dashboard.deviceSummary' },
                          {
                            count: deviceSns.length,
                            preview: deviceSns.slice(0, 2).join(', '),
                            more: deviceSns.length > 2 ? ' ...' : '',
                          },
                        )
                  }
                  placeholder={intl.formatMessage({ id: 'perf.dashboard.devicePlaceholder' })}
                />
                <Button onClick={() => setDevicePickerOpen(true)}>
                  {intl.formatMessage({ id: 'perf.dashboard.pickFromList' })}
                </Button>
              </Space.Compact>
            </Form.Item>

            <Form.Item
              label={intl.formatMessage({ id: 'perf.dashboard.fieldMetric' })}
              style={{ marginBottom: 0 }}
            >
              <Space.Compact style={{ width: 360 }}>
                <Input
                  readOnly
                  value={
                    metricPaths.length === 0
                      ? ''
                      : intl.formatMessage(
                          {
                            id: metricsTouched
                              ? 'perf.dashboard.metricSummary'
                              : 'perf.dashboard.metricSummaryDefault',
                          },
                          { count: metricPaths.length },
                        )
                  }
                  placeholder={intl.formatMessage({ id: 'perf.dashboard.metricPlaceholder' })}
                />
                <Button onClick={() => setMetricPickerOpen(true)}>
                  {intl.formatMessage({ id: 'perf.dashboard.pickFromList' })}
                </Button>
              </Space.Compact>
            </Form.Item>

            <Form.Item
              label={intl.formatMessage({ id: 'perf.dashboard.fieldGranularity' })}
              style={{ marginBottom: 0 }}
            >
              <Radio.Group
                value={granularity}
                onChange={(e) => handleGranularityChange(e.target.value)}
                options={granularityOptions}
                optionType="button"
                buttonStyle="solid"
              />
            </Form.Item>

          </Space>

          {deviceSns.length > 0 && (
            <div style={{ marginTop: 16 }}>
              <Form.Item
                label={intl.formatMessage({ id: 'perf.drilldown.label' })}
                style={{ marginBottom: 0 }}
              >
                <CellDrilldownSelector
                  deviceSns={deviceSns}
                  technology={tech}
                  value={cellSel}
                  onChange={setCellSel}
                />
              </Form.Item>
            </div>
          )}

          <div style={{ marginTop: 16 }}>
            <DashboardFilterBar value={filter} onChange={handleFilterChange} />
          </div>

          <div style={{ marginTop: 16 }}>
            <Space>
              <Button
                type="primary"
                icon={<LineChartOutlined />}
                loading={isFetching}
                onClick={handleQuery}
              >
                {intl.formatMessage({ id: 'perf.dashboard.btnPlot' })}
              </Button>
              <Button
                icon={<ReloadOutlined />}
                onClick={() => void refetch()}
                disabled={!submitted}
              >
                {intl.formatMessage({ id: 'common.refresh' })}
              </Button>
              <Button
                icon={<ExportOutlined />}
                loading={createExport.isPending}
                onClick={handleExport}
                title={intl.formatMessage({ id: 'kpiExport.export.tooltip' })}
              >
                {intl.formatMessage({ id: 'kpiExport.export.button' })}
              </Button>
            </Space>
          </div>
        </Form>
      </Card>

      {!submitted ? (
        <Card>
          <Empty
            description={intl.formatMessage({ id: 'perf.dashboard.emptyPickConditions' })}
            style={{ marginTop: 40 }}
          />
        </Card>
      ) : isLoading || isFetching || prevFetching ? (
        <Card>
          <LoadingSpinner tip={intl.formatMessage({ id: 'common.loading' })} />
        </Card>
      ) : charts.length === 0 ? (
        <Card>
          <Empty description={intl.formatMessage({ id: 'perf.dashboard.emptyNoDataForCondition' })} />
        </Card>
      ) : (
        <>
          {truncated ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 12 }}
              message={intl.formatMessage(
                { id: 'perf.dashboard.truncatedTip' },
                { shown: rawRows.length, total: rawTotal },
              )}
            />
          ) : null}
          <Card size="small" style={{ marginBottom: 12 }}>
            <Space size={8} wrap>
              <Typography.Text strong>
                {intl.formatMessage({ id: 'perf.dashboard.deviceListSummary' })}
              </Typography.Text>
              <Tag color="geekblue">{tech.toUpperCase()}</Tag>
              <Tag color="blue">
                {intl.formatMessage(
                  { id: 'perf.dashboard.deviceUnit' },
                  { count: submitted.deviceSns.length },
                )}
              </Tag>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {intl.formatMessage(
                  { id: 'perf.dashboard.metricCount' },
                  { count: charts.length },
                )}
              </Typography.Text>
            </Space>
          </Card>
          {charts.map((c) => (
            <ChartCard key={c.metricPath} chart={c} />
          ))}
        </>
      )}

      <DevicePickerModal
        open={devicePickerOpen}
        onClose={() => setDevicePickerOpen(false)}
        onConfirm={(sns) => {
          setCellSel({}); // 设备变更 → 重置下钻选择为全选。
          if (isDeviceViewDeviceSelectionOverLimit(sns)) {
            message.warning(
              intl.formatMessage(
                { id: 'perf.picker.deviceLimitExceeded' },
                { max: PM_QUERY_SELECTION_LIMIT, count: sns.length },
              ),
            );
            setDeviceSns(sns.slice(0, PM_QUERY_SELECTION_LIMIT));
          } else {
            setDeviceSns(sns);
          }
        }}
        initialSelected={deviceSns}
        technology={tech}
        maxSelected={PM_QUERY_SELECTION_LIMIT}
      />

      <MetricPickerModal
        open={metricPickerOpen}
        onClose={() => setMetricPickerOpen(false)}
        onConfirm={(paths) => {
          setMetricPaths(paths);
          setMetricsTouched(true);
        }}
        initialSelected={metricPaths}
        initialDeviceType={TECH_TO_DEVICE_TYPE[tech]}
        lockDeviceType
        maxSelected={PM_QUERY_SELECTION_LIMIT}
        enableBatchInput
      />
    </div>
  );
}
