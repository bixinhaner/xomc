/**
 * T-0188 性能仪表盘 · 页签2 设备列表（独立即席查看，不依赖任何聚合任务）。
 *
 * 交互：
 *   - 制式 Segmented（默认 LTE；ENB/GNB/GSM）。
 *   - 设备多选（≤10，复用 KPIQuery/components/DevicePickerModal；超 10 拦截提示并截断）。
 *   - 指标可换（复用 KPIQuery/components/MetricPickerModal，initialDeviceType 随制式）；
 *     默认集 = 选中制式的内置任务指标集（usePmAdhocList isBuiltin，按 technology 找一个取 metricPaths）。
 *     用户未手动改过指标时，切制式默认集随之切换；手动改过则保留用户选择。
 *   - 粒度选择（默认 15min）。
 *   - 共用三级筛选（DashboardFilterBar）：大时间段（默认近 7 天）+ 星期多选 + 小时段多选 + 周期对比开关（T-0189）。
 *   - 出图：取数 useAggregatedMetricsByDevices → 星期/小时段前端筛 → buildDeviceMetricCharts → 每指标一张 ChartCard（每设备一条线）。
 *   - 周期对比开关打开：再拉上一周期窗口数据，套同口径星期/小时段，叠加虚线（T-0189）。
 *
 * 本任务到 SN 级，不下钻小区/PLMN。
 */

import { useEffect, useMemo, useState } from 'react';
import {
  App,
  Button,
  Card,
  Empty,
  Form,
  Input,
  Radio,
  Segmented,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import { ReloadOutlined, LineChartOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAggregatedMetricsByDevices } from '@core/hooks/api/usePmQuery';
import { usePmAdhocList } from '@core/hooks/api/usePmAdhoc';
import type { DeviceType } from '@core/types/indicatorLibrary';
import type { Granularity } from '@core/types/pmDashboard';
import DevicePickerModal from '../KPIQuery/components/DevicePickerModal';
import MetricPickerModal from '../KPIQuery/components/MetricPickerModal';
import ChartCard from './ChartCard';
import { buildDeviceMetricCharts } from './deviceListUtils';
import DashboardFilterBar, { type DashboardFilterValue } from './DashboardFilterBar';
import {
  ALL_HOURS,
  ALL_WEEKDAYS,
  attachCompareSeries,
  filterRowsByWeekdayHour,
  previousWindow,
} from './dashboardFilterUtils';

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

const GRANULARITY_OPTIONS: { label: string; value: Granularity }[] = [
  { label: '15 分钟', value: '15min' },
  { label: '小时', value: 'hourly' },
  { label: '日', value: 'daily' },
  { label: '周', value: 'weekly' },
  { label: '月', value: 'monthly' },
];

const MAX_DEVICES = 10;

export default function DeviceListPane() {
  const { message } = App.useApp();

  // ── 选择条件 ───────────────────────────────────────────────────────
  const [tech, setTech] = useState<Tech>('lte');
  const [deviceSns, setDeviceSns] = useState<string[]>([]);
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
  } | null>(null);

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
      message.error(`查询失败：${first?.message ?? '未知错误'}`);
    }
  }, [errors, message]);

  // 星期/小时段=纯前端在已取行里筛命中点（全选不过滤），当前与上一周期套同口径。
  const charts = useMemo(() => {
    if (!submitted) return [];
    const wd = new Set(submitted.weekdays);
    const hr = new Set(submitted.hours);
    const cur = buildDeviceMetricCharts(
      filterRowsByWeekdayHour(rawRows, wd, hr),
      submitted.granularity,
    );
    if (!submitted.compare) return cur;
    const prev = buildDeviceMetricCharts(
      filterRowsByWeekdayHour(rawPrevRows, wd, hr),
      submitted.granularity,
    );
    return attachCompareSeries(cur, prev, submitted.offsetMs);
  }, [rawRows, rawPrevRows, submitted]);

  // ── 行为 ───────────────────────────────────────────────────────────
  const handleTechChange = (v: Tech) => {
    setTech(v);
    // 切制式清空已选设备（设备制式与图制式应一致），指标默认集由 effect 随制式切换。
    setDeviceSns([]);
  };

  const handleQuery = () => {
    if (deviceSns.length === 0) {
      message.warning('请选择至少一个设备');
      return;
    }
    if (metricPaths.length === 0) {
      message.warning('请选择至少一个指标');
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
    });
  };

  return (
    <div style={{ height: 'calc(100vh - 190px)', overflow: 'auto' }}>
      <Card size="small" style={{ marginBottom: 12 }}>
        <Form layout="vertical" size="middle">
          <Space wrap size="middle" align="start">
            <Form.Item label="制式" style={{ marginBottom: 0 }}>
              <Segmented
                value={tech}
                onChange={(v) => handleTechChange(v as Tech)}
                options={TECH_OPTIONS}
              />
            </Form.Item>

            <Form.Item label="设备（≤10）" style={{ marginBottom: 0 }}>
              <Space.Compact style={{ width: 360 }}>
                <Input
                  readOnly
                  value={
                    deviceSns.length === 0
                      ? ''
                      : `已选 ${deviceSns.length} 个：${deviceSns.slice(0, 2).join(', ')}${deviceSns.length > 2 ? ' ...' : ''}`
                  }
                  placeholder="点击右侧按钮选择设备"
                />
                <Button onClick={() => setDevicePickerOpen(true)}>列表选</Button>
              </Space.Compact>
            </Form.Item>

            <Form.Item label="指标" style={{ marginBottom: 0 }}>
              <Space.Compact style={{ width: 360 }}>
                <Input
                  readOnly
                  value={
                    metricPaths.length === 0
                      ? ''
                      : `已选 ${metricPaths.length} 个${metricsTouched ? '' : '（默认集）'}`
                  }
                  placeholder="默认取该制式内置任务指标集"
                />
                <Button onClick={() => setMetricPickerOpen(true)}>列表选</Button>
              </Space.Compact>
            </Form.Item>

            <Form.Item label="粒度" style={{ marginBottom: 0 }}>
              <Radio.Group
                value={granularity}
                onChange={(e) => setGranularity(e.target.value)}
                options={GRANULARITY_OPTIONS}
                optionType="button"
                buttonStyle="solid"
              />
            </Form.Item>

          </Space>

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
                出图
              </Button>
              <Button
                icon={<ReloadOutlined />}
                onClick={() => void refetch()}
                disabled={!submitted}
              >
                刷新
              </Button>
            </Space>
          </div>
        </Form>
      </Card>

      {!submitted ? (
        <Card>
          <Empty description="选择制式 / 设备 / 指标后点「出图」" style={{ marginTop: 40 }} />
        </Card>
      ) : isLoading || isFetching || prevFetching ? (
        <Card>
          <Spin tip="加载中..." />
        </Card>
      ) : charts.length === 0 ? (
        <Card>
          <Empty description="所选条件下暂无数据" />
        </Card>
      ) : (
        <>
          <Card size="small" style={{ marginBottom: 12 }}>
            <Space size={8} wrap>
              <Typography.Text strong>设备列表出图</Typography.Text>
              <Tag color="geekblue">{tech.toUpperCase()}</Tag>
              <Tag color="blue">{submitted.deviceSns.length} 设备</Tag>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {charts.length} 指标
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
          if (sns.length > MAX_DEVICES) {
            message.warning(`最多选择 ${MAX_DEVICES} 个设备，已自动截取前 ${MAX_DEVICES} 个`);
            setDeviceSns(sns.slice(0, MAX_DEVICES));
          } else {
            setDeviceSns(sns);
          }
        }}
        initialSelected={deviceSns}
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
      />
    </div>
  );
}
