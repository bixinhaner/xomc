/**
 * T-0164-P7 / G7 自定义聚合任务管理页面。
 *
 * T-0186：
 *   - 列表分两区：内置区（is_builtin=true）+ 自建区（is_builtin=false），各一张 Table。
 *   - 任务详情 Drawer 加 Tabs：运行历史（pm_adhoc_task_runs）+ 结果（AdhocResultPanel）。
 *   - 详情顶部制式渲染为只读 Tag（建后不可改、无切换控件）。
 *
 * T-0164 收尾：
 *   G7-Gap-2  结果查看用 AdhocResultPanel（G6 panel 风格，多指标多 series ECharts）
 *   G6-Gap-12 联动：PanelConfigDrawer 跳转携带 ?preset=panel&device_sns=...&metric_paths=...&granularities=...
 */

import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  Button,
  Card,
  Drawer,
  Modal,
  Space,
  Table,
  Tabs,
  Tag,
  Progress,
  Descriptions,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  usePmAdhocList,
  usePmAdhocRuns,
  useCancelPmAdhoc,
} from '@core/hooks/api/usePmAdhoc';
import type {
  AdhocMode,
  AdhocStatus,
  AdhocTask,
  AdhocTaskRun,
} from '@core/types/pmAdhoc';
import { CreateAdhocTaskDrawer, type CreateAdhocPreset } from './CreateAdhocTaskDrawer';
import { AdhocResultPanel } from './AdhocResultPanel';

const statusColor: Record<AdhocStatus, string> = {
  pending: 'default',
  running: 'processing',
  scheduled: 'cyan',
  succeeded: 'success',
  failed: 'error',
  canceled: 'warning',
};

const statusLabel: Record<AdhocStatus, string> = {
  pending: '待执行',
  running: '执行中',
  scheduled: '已排期',
  succeeded: '成功',
  failed: '失败',
  canceled: '已取消',
};

const dimensionLabel: Record<string, string> = {
  device: '按设备',
  aggregate_group: '临时组',
  product: '按产品',
  band: '按频段',
  network: '全网',
  device_group: '设备组',
};

const technologyLabel: Record<string, string> = {
  lte: 'LTE (4G)',
  nr: '5G NR',
  gsm: 'GSM (2G)',
};

function fmtTime(v?: string): string {
  if (!v) return '—';
  const d = new Date(v);
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString();
}

/** 运行历史表（任务详情 Tab）。 */
function RunHistoryTab({ taskId }: { taskId: string }) {
  // 运行中任务会持续产生 run，轮询刷新
  const { data: runs = [], isLoading } = usePmAdhocRuns(taskId, { refetchInterval: 10000 });

  const columns: ColumnsType<AdhocTaskRun> = [
    { title: '编号', dataIndex: 'runSeq', width: 70 },
    {
      title: '粒度',
      dataIndex: 'granularity',
      width: 100,
      render: (g: string) => g || '—',
    },
    {
      title: '维度',
      dataIndex: 'dimension',
      width: 100,
      render: (d: string) => dimensionLabel[d] ?? d ?? '—',
    },
    {
      title: '时间窗',
      width: 240,
      render: (_, r) =>
        r.windowStart || r.windowEnd
          ? `${fmtTime(r.windowStart)} ~ ${fmtTime(r.windowEnd)}`
          : '滚动窗口',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (s: AdhocStatus) => <Tag color={statusColor[s]}>{statusLabel[s] ?? s}</Tag>,
    },
    { title: '入队时间', dataIndex: 'queuedAt', width: 180, render: (v: string) => fmtTime(v) },
    { title: '完成时间', dataIndex: 'finishedAt', width: 180, render: (v: string) => fmtTime(v) },
    { title: '结果行数', dataIndex: 'rowsTotal', width: 90 },
    {
      title: '失败原因',
      dataIndex: 'error',
      ellipsis: true,
      render: (e: string) => (e ? <Typography.Text type="danger">{e}</Typography.Text> : '—'),
    },
  ];

  return (
    <Table<AdhocTaskRun>
      rowKey="id"
      size="small"
      loading={isLoading}
      dataSource={runs}
      columns={columns}
      pagination={{ pageSize: 10, size: 'small' }}
      expandable={{
        // failed 行 error 全文可展开
        rowExpandable: (r) => Boolean(r.error),
        expandedRowRender: (r) => (
          <Typography.Paragraph
            type="danger"
            style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}
          >
            {r.error}
          </Typography.Paragraph>
        ),
      }}
    />
  );
}

/** 单张任务表（内置区 / 自建区共用）。 */
function TaskTable({
  tasks,
  loading,
  onView,
  onCancel,
}: {
  tasks: AdhocTask[];
  loading: boolean;
  onView: (t: AdhocTask) => void;
  onCancel: (id: string) => void;
}) {
  const columns: ColumnsType<AdhocTask> = [
    { title: '名称', dataIndex: 'name' },
    {
      title: '模式',
      dataIndex: 'mode',
      width: 90,
      render: (m: AdhocMode) =>
        m === 'continuous' ? <Tag color="purple">持续</Tag> : <Tag>单次</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (s: AdhocStatus) => <Tag color={statusColor[s]}>{statusLabel[s] ?? s}</Tag>,
    },
    {
      title: '进度',
      dataIndex: 'progress',
      width: 140,
      render: (p: number, r) =>
        r.status === 'running' || r.status === 'succeeded' ? (
          <Progress percent={p} size="small" />
        ) : (
          '—'
        ),
    },
    { title: '设备数', render: (_, r) => r.deviceSns.length, width: 80 },
    {
      title: '指标 × 粒度',
      render: (_, r) => `${r.metricPaths.length} × ${r.granularities.length}`,
      width: 110,
    },
    { title: '创建时间', dataIndex: 'createdAt', width: 180, render: (v: string) => fmtTime(v) },
    {
      title: '操作',
      width: 180,
      render: (_, r) => (
        <Space>
          <Button size="small" onClick={() => onView(r)}>
            查看详情
          </Button>
          {(r.status === 'pending' || r.status === 'running' || r.status === 'scheduled') && (
            <Button size="small" danger icon={<DeleteOutlined />} onClick={() => onCancel(r.id)}>
              取消
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <Table<AdhocTask>
      rowKey="id"
      size="small"
      loading={loading}
      dataSource={tasks}
      columns={columns}
      pagination={{ pageSize: 10, size: 'small' }}
    />
  );
}

export default function PmAdhocPage() {
  // T-0186：分两区，各发一次 list（内置 / 自建）。
  const { data: builtinTasks = [], isLoading: builtinLoading } = usePmAdhocList({
    refetchInterval: 5000,
    isBuiltin: true,
  });
  const { data: customTasks = [], isLoading: customLoading } = usePmAdhocList({
    refetchInterval: 5000,
    isBuiltin: false,
  });
  const cancelMut = useCancelPmAdhoc();
  const navigate = useNavigate();

  const [searchParams, setSearchParams] = useSearchParams();
  const [createOpen, setCreateOpen] = useState(false);
  const [createPreset, setCreatePreset] = useState<CreateAdhocPreset | undefined>(undefined);
  const [selectedTask, setSelectedTask] = useState<AdhocTask | null>(null);

  // G6-Gap-12 联动：URL preset 触发自动打开 Drawer
  useEffect(() => {
    if (searchParams.get('preset') === 'panel') {
      const deviceSns = (searchParams.get('device_sns') ?? '')
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      const metricPaths = (searchParams.get('metric_paths') ?? '')
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      const granularities = (searchParams.get('granularities') ?? '')
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      setCreatePreset({
        name: `从 panel 派生 (${deviceSns.length} 设备)`,
        deviceSns,
        metricPaths,
        granularities,
      });
      setCreateOpen(true);
      const next = new URLSearchParams(searchParams);
      next.delete('preset');
      next.delete('device_sns');
      next.delete('metric_paths');
      next.delete('granularities');
      setSearchParams(next, { replace: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleCancel = (id: string) => {
    Modal.confirm({
      title: '取消任务',
      content: '已在运行的任务取消后不会回滚已写入结果',
      onOk: async () => {
        await cancelMut.mutateAsync(id);
        message.success('已取消');
      },
    });
  };

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Card
        title="内置聚合任务"
        size="small"
        extra={<Tag color="blue">系统预置</Tag>}
      >
        <TaskTable
          tasks={builtinTasks}
          loading={builtinLoading}
          onView={setSelectedTask}
          onCancel={handleCancel}
        />
      </Card>

      <Card
        title="自建聚合任务"
        size="small"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate('/performance/pm-adhoc/new')}
          >
            新建任务
          </Button>
        }
      >
        <TaskTable
          tasks={customTasks}
          loading={customLoading}
          onView={setSelectedTask}
          onCancel={handleCancel}
        />
      </Card>

      <CreateAdhocTaskDrawer
        open={createOpen}
        preset={createPreset}
        onClose={() => {
          setCreateOpen(false);
          setCreatePreset(undefined);
        }}
      />

      <Drawer
        title={selectedTask ? `任务详情：${selectedTask.name}` : '任务详情'}
        width={920}
        open={Boolean(selectedTask)}
        onClose={() => setSelectedTask(null)}
        destroyOnClose
      >
        {selectedTask && (
          <>
            <Descriptions size="small" column={2} bordered style={{ marginBottom: 16 }}>
              <Descriptions.Item label="模式">
                {selectedTask.mode === 'continuous' ? '持续' : '单次'}
              </Descriptions.Item>
              <Descriptions.Item label="维度">
                {dimensionLabel[selectedTask.dimension] ?? selectedTask.dimension}
              </Descriptions.Item>
              <Descriptions.Item label="制式">
                {/* T-0186：制式只读，建后不可改，无切换控件 */}
                {selectedTask.technology ? (
                  <Tag color="geekblue">
                    {technologyLabel[selectedTask.technology] ?? selectedTask.technology}
                  </Tag>
                ) : (
                  <Tag>不限制式</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="当前状态">
                <Tag color={statusColor[selectedTask.status]}>
                  {statusLabel[selectedTask.status] ?? selectedTask.status}
                </Tag>
              </Descriptions.Item>
            </Descriptions>

            <Tabs
              defaultActiveKey="runs"
              items={[
                {
                  key: 'runs',
                  label: '运行历史',
                  children: <RunHistoryTab taskId={selectedTask.id} />,
                },
                {
                  key: 'results',
                  label: '结果',
                  children: <AdhocResultPanel taskId={selectedTask.id} />,
                },
              ]}
            />
          </>
        )}
      </Drawer>
    </Space>
  );
}
