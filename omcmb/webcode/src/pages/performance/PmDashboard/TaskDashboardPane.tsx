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
import { usePmAdhocDetail, usePmAdhocResults } from '@core/hooks/api/usePmAdhoc';
import { buildMetricCharts } from './taskDashboardUtils';
import ChartCard from './ChartCard';

interface Props {
  taskId: string;
}

const GRAN_LABEL: Record<string, string> = {
  '15min': '15 分',
  hourly: '小时',
  daily: '日',
  weekly: '周',
  monthly: '月',
};

export default function TaskDashboardPane({ taskId }: Props) {
  const taskQuery = usePmAdhocDetail(taskId);
  // 仪表盘取较多结果行用于画线（最近 N 行）。
  const { data: rows = [], isLoading: rowsLoading } = usePmAdhocResults(taskId, { limit: 10000 });

  const granularities = useMemo(
    () => taskQuery.data?.granularities ?? [],
    [taskQuery.data?.granularities],
  );
  const [activeGran, setActiveGran] = useState<string | undefined>(undefined);
  const effectiveGran = activeGran && granularities.includes(activeGran) ? activeGran : granularities[0];

  const charts = useMemo(() => {
    if (!taskQuery.data || !effectiveGran) return [];
    return buildMetricCharts(rows, taskQuery.data.dimension, effectiveGran);
  }, [rows, taskQuery.data, effectiveGran]);

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

      {rowsLoading ? (
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
