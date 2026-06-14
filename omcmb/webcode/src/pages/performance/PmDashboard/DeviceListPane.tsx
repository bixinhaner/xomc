/**
 * T-0188 性能仪表盘 · 页签2 设备列表（独立即席查看，不依赖任何聚合任务）。
 *
 * 交互：
 *   - 制式 Segmented（默认 LTE；ENB/GNB/GSM）。
 *   - 设备多选（≤10，复用 KPIQuery/components/DevicePickerModal；超 10 拦截提示并截断）。
 *   - 指标可换（复用共享件 components/MetricPickerModal，initialDeviceType 随制式）；
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
import dayjs from 'dayjs';
import {
  useAggregatedMetricsByDevices,
  useMetricObjectsByDevices,
} from '@core/hooks/api/usePmQuery';
import { usePmAdhocList } from '@core/hooks/api/usePmAdhoc';
import { useCreateKpiExport } from '@core/hooks/api/useKpiExport';
import type { DeviceType } from '@core/types/indicatorLibrary';
import type { Granularity } from '@core/types/pmDashboard';
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
  extendChartsAxis,
  filterRowsByWeekdayHour,
  previousWindow,
} from './dashboardFilterUtils';
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

// 粒度选项语料键（label 走 i18n，value 不变）。
const GRANULARITY_MSG_IDS: { id: string; value: Granularity }[] = [
  { id: 'perf.dashboard.granular15min', value: '15min' },
  { id: 'perf.dashboard.granularHourly', value: 'hourly' },
  { id: 'perf.dashboard.granularDaily', value: 'daily' },
  { id: 'perf.dashboard.granularWeekly', value: 'weekly' },
  { id: 'perf.dashboard.granularMonthly', value: 'monthly' },
];

const MAX_DEVICES = 10;

export default function DeviceListPane() {
  const intl = useIntl();
  const { message } = App.useApp();

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
  // 共用三级筛选 + 周期对比开关（大时间段 + 星期 + 小时段 + 对比）。
  const [filter, setFilter] = useState<DashboardFilterValue>({
    range: [dayjs().subtract(7, 'day'), dayjs()],
    weekdays: [...ALL_WEEKDAYS],
    hours: [...ALL_HOURS],
    compare: false,
  });
  const [devicePickerOpen, setDevicePickerOpen] = useState(false);
  const [metricPickerOpen, setMetricPickerOpen] = useState(false);

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
    };
  }, [submitted]);

  const {
    data: rawRows = [],
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
  const prevParams = useMemo(() => {
    if (!submitted || !submitted.compare) return null;
    return {
      granularity: submitted.granularity,
      metricPaths: submitted.metricPaths,
      startTime: submitted.prevStartTime,
      endTime: submitted.prevEndTime,
      limit: 5000,
      fillEmpty: true,
    };
  }, [submitted]);

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

  // 星期/小时段=纯前端在已取行里筛命中点（全选不过滤），当前与上一周期套同口径。
  const charts = useMemo(() => {
    if (!submitted) return [];
    const wd = new Set(submitted.weekdays);
    const hr = new Set(submitted.hours);
    // T-0193：先按小区/PLMN 白名单即席过滤，再套星期/小时段，再转置分线。
    const curRows = filterRowsByWeekdayHour(
      filterRowsByObjectLdns(rawRows, submitted.allowedLdns),
      wd,
      hr,
    );
    // T-AXISFILL：转置出当前图集后立即扩轴（按 submitted 范围+粒度连续铺刻度、套星期/小时筛选、并集真实桶），
    // 空刻度补 '-'，再挂周期对比（compare 按毫秒对齐到已扩展的 cur.buckets，prev 不单独扩轴）。
    const cur = extendChartsAxis(buildDeviceMetricCharts(curRows, submitted.granularity), {
      rangeStartMs: dayjs(submitted.startTime).valueOf(),
      rangeEndMs: dayjs(submitted.endTime).valueOf(),
      weekdays: wd,
      hours: hr,
      granularity: submitted.granularity,
    });
    if (!submitted.compare) return cur;
    const prevRows = filterRowsByWeekdayHour(
      filterRowsByObjectLdns(rawPrevRows, submitted.allowedLdns),
      wd,
      hr,
    );
    const prev = buildDeviceMetricCharts(prevRows, submitted.granularity);
    return attachCompareSeries(cur, prev, submitted.offsetMs, submitted.granularity);
  }, [rawRows, rawPrevRows, submitted]);

  // ── 行为 ───────────────────────────────────────────────────────────
  const handleTechChange = (v: Tech) => {
    setTech(v);
    // 切制式清空已选设备（设备制式与图制式应一致），指标默认集由 effect 随制式切换。
    setDeviceSns([]);
    setCellSel({}); // 设备清空 → 下钻选择重置（全选）。
  };

  // ── 导出（T4 dashboard 来源）：带当前筛选 POST 建任务，不卡页面 ──────────
  const createExport = useCreateKpiExport();

  // 组装当前筛选快照（与 handleQuery 同口径：设备/指标/粒度/时间/小区下钻白名单）。
  // A1：下钻定格的小区/PLMN 白名单一并带进导出（复用 handleQuery 的 getEffectiveLdns，空=不过滤）。
  const buildExportSelection = (): DashboardExportSelection => {
    const [start, end] = filter.range;
    return {
      technology: tech,
      deviceSns,
      metricPaths,
      granularity,
      startTime: start.toISOString(),
      endTime: end.toISOString(),
      objectLdns: getEffectiveLdns(cellSel, objectsByDevice),
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
      {
        sourceType: 'dashboard',
        params: buildDashboardExportParams(sel),
        taskName: defaultExportTaskName('dashboard'),
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

  const handleQuery = () => {
    if (deviceSns.length === 0) {
      message.warning(intl.formatMessage({ id: 'perf.dashboard.selectAtLeastOneDevice' }));
      return;
    }
    if (metricPaths.length === 0) {
      message.warning(intl.formatMessage({ id: 'perf.dashboard.selectAtLeastOneMetric' }));
      return;
    }
    const [start, end] = filter.range;
    const [prevStart, prevEnd] = previousWindow(filter.range);
    setSubmitted({
      deviceSns,
      metricPaths,
      granularity,
      startTime: start.toISOString(),
      endTime: end.toISOString(),
      weekdays: filter.weekdays,
      hours: filter.hours,
      compare: filter.compare,
      offsetMs: end.valueOf() - start.valueOf(),
      prevStartTime: prevStart.toISOString(),
      prevEndTime: prevEnd.toISOString(),
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
                onChange={(e) => setGranularity(e.target.value)}
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
            <DashboardFilterBar value={filter} onChange={setFilter} />
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
          if (sns.length > MAX_DEVICES) {
            message.warning(
              intl.formatMessage(
                { id: 'perf.dashboard.maxDevicesTruncated' },
                { max: MAX_DEVICES },
              ),
            );
            setDeviceSns(sns.slice(0, MAX_DEVICES));
          } else {
            setDeviceSns(sns);
          }
        }}
        initialSelected={deviceSns}
        technology={tech}
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
      />
    </div>
  );
}
