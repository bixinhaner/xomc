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
import { Alert, Card, Empty, Segmented, Space, Spin, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { usePmAdhocDetail, usePmAdhocResults } from '@core/hooks/api/usePmAdhoc';
import { buildMetricCharts } from './taskDashboardUtils';
import ChartCard from './ChartCard';
import DashboardFilterBar, { type DashboardFilterValue } from './DashboardFilterBar';
import {
  ALL_HOURS,
  ALL_WEEKDAYS,
  attachCompareSeries,
  filterRowsByWeekdayHour,
  previousWindow,
} from './dashboardFilterUtils';

interface Props {
  taskId: string;
}

const RESULTS_LIMIT = 10000;

const GRAN_LABEL: Record<string, string> = {
  '15min': '15 分',
  hourly: '小时',
  daily: '日',
  weekly: '周',
  monthly: '月',
};

export default function TaskDashboardPane({ taskId }: Props) {
  const taskQuery = usePmAdhocDetail(taskId);

  // ── 共用三级筛选 + 周期对比开关（本 Pane 持状态，驱动取数 + 二拉）──────
  const [filter, setFilter] = useState<DashboardFilterValue>({
    range: [dayjs().subtract(7, 'day'), dayjs()],
    weekdays: [...ALL_WEEKDAYS],
    hours: [...ALL_HOURS],
    compare: false,
  });
  const [start, end] = filter.range;
  const startISO = start.toISOString();
  const endISO = end.toISOString();
  const offsetMs = end.valueOf() - start.valueOf();
  const [prevStart, prevEnd] = previousWindow(filter.range);

  // 大时间段驱动取数（后端按 time 窗口过滤）。仪表盘取较多结果行用于画线。
  const { data: rawRows = [], isLoading: rowsLoading } = usePmAdhocResults(taskId, {
    limit: RESULTS_LIMIT,
    startTime: startISO,
    endTime: endISO,
  });
  // 周期对比开关打开时再拉一次上一周期（同任务、上一周期窗口）。
  const { data: rawPrevRows = [], isLoading: prevLoading } = usePmAdhocResults(
    filter.compare ? taskId : undefined,
    {
      limit: RESULTS_LIMIT,
      startTime: prevStart.toISOString(),
      endTime: prevEnd.toISOString(),
    },
  );

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
    const cur = buildMetricCharts(rows, taskQuery.data.dimension, effectiveGran);
    if (!filter.compare) return cur;
    const prev = buildMetricCharts(prevRows, taskQuery.data.dimension, effectiveGran);
    return attachCompareSeries(cur, prev, offsetMs);
  }, [rows, prevRows, taskQuery.data, effectiveGran, filter.compare, offsetMs]);

  if (taskQuery.isLoading) {
    return (
      <Card>
        <Spin tip="加载任务..." />
      </Card>
    );
  }
  if (taskQuery.isError || !taskQuery.data) {
    return (
      <Card>
        <Alert
          type="warning"
          showIcon
          message="任务不可用"
          description="任务已被删除或结果已超出保留期。"
        />
      </Card>
    );
  }

  const task = taskQuery.data;

  return (
    <div>
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space size={8} wrap>
          <Typography.Text strong>{task.name}</Typography.Text>
          {task.technology && <Tag color="geekblue">{task.technology.toUpperCase()}</Tag>}
          <Tag color="purple">{task.mode === 'continuous' ? '持续' : '单次'}</Tag>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {task.metricPaths.length} 指标
          </Typography.Text>
          {granularities.length > 1 && (
            <Segmented
              size="small"
              value={effectiveGran}
              onChange={(v) => setActiveGran(v as string)}
              options={granularities.map((g) => ({ label: GRAN_LABEL[g] ?? g, value: g }))}
            />
          )}
        </Space>
      </Card>

      <Card size="small" style={{ marginBottom: 12 }}>
        <DashboardFilterBar value={filter} onChange={setFilter} />
      </Card>

      {truncated && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          message="结果可能不全"
          description={`本次时间段命中的结果行已达上限（${RESULTS_LIMIT} 行），图中可能未包含全部数据。请缩小时间段或减少指标/系列。`}
        />
      )}

      {rowsLoading || (filter.compare && prevLoading) ? (
        <Card>
          <Spin tip="加载结果..." />
        </Card>
      ) : charts.length === 0 ? (
        <Card>
          <Empty description={effectiveGran ? `${GRAN_LABEL[effectiveGran] ?? effectiveGran} 粒度暂无数据` : '任务暂无数据'} />
        </Card>
      ) : (
        charts.map((c) => <ChartCard key={c.metricPath} chart={c} />)
      )}
    </div>
  );
}
