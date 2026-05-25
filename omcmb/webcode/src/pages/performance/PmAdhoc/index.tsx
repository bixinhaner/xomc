/**
 * T-0164-P7 / G7 自定义聚合任务管理页面。
 *
 * 左侧 TaskList 列表 + 右侧 ResultsViewer + 顶部 CreateTaskDrawer。
 *
 * T-0164 收尾：
 *   G7-Gap-1  创建抽屉抽到 CreateAdhocTaskDrawer 公共组件（DashboardEditorPane 复用）
 *   G7-Gap-2  结果查看用 AdhocResultPanel（G6 panel 风格，多指标多 series ECharts）
 *   G7-Gap-3  AdhocResultPanel 内置粒度 Tab（与 G6-Gap-6 一致）
 *   G6-Gap-12 联动：PanelConfigDrawer 跳转时携带 ?preset=panel&device_sns=...&metric_paths=...&granularities=...
 */

import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Button, Card, Drawer, Modal, Space, Table, Tag, Progress, message } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  usePmAdhocList,
  useCancelPmAdhoc,
} from '@core/hooks/api/usePmAdhoc';
import type { AdhocMode, AdhocStatus, AdhocTask } from '@core/types/pmAdhoc';
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

export default function PmAdhocPage() {
  const { data: tasks = [], isLoading } = usePmAdhocList({ refetchInterval: 5000 });
  const cancelMut = useCancelPmAdhoc();

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
      // 清除 query 避免刷新页面重复打开
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
    <Card
      title="自定义聚合任务"
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setCreatePreset(undefined);
            setCreateOpen(true);
          }}
        >
          新建任务
        </Button>
      }
    >
      <Table<AdhocTask>
        rowKey="id"
        size="small"
        loading={isLoading}
        dataSource={tasks}
        columns={[
          { title: '名称', dataIndex: 'name' },
          {
            title: '模式',
            dataIndex: 'mode',
            width: 90,
            render: (m: AdhocMode) => (m === 'continuous' ? <Tag color="purple">持续</Tag> : <Tag>单次</Tag>),
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 100,
            render: (s: AdhocStatus) => <Tag color={statusColor[s]}>{s}</Tag>,
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
          { title: '创建时间', dataIndex: 'createdAt', width: 180 },
          {
            title: '操作',
            width: 180,
            render: (_, r) => (
              <Space>
                <Button size="small" onClick={() => setSelectedTask(r)}>
                  查看结果
                </Button>
                {(r.status === 'pending' || r.status === 'running' || r.status === 'scheduled') && (
                  <Button size="small" danger icon={<DeleteOutlined />} onClick={() => handleCancel(r.id)}>
                    取消
                  </Button>
                )}
              </Space>
            ),
          },
        ]}
      />

      <CreateAdhocTaskDrawer
        open={createOpen}
        preset={createPreset}
        onClose={() => {
          setCreateOpen(false);
          setCreatePreset(undefined);
        }}
      />

      <Drawer
        title={selectedTask ? `结果：${selectedTask.name}` : '结果'}
        width={820}
        open={Boolean(selectedTask)}
        onClose={() => setSelectedTask(null)}
      >
        {selectedTask && <AdhocResultPanel taskId={selectedTask.id} />}
      </Drawer>
    </Card>
  );
}
