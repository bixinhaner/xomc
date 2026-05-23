/**
 * T-0164-P7 / G7 自定义聚合任务管理页面。
 *
 * 左侧 TaskList 列表 + 右侧 ResultsViewer + 顶部 CreateTaskDrawer。
 */

import { useState } from 'react';
import { Button, Card, Drawer, Form, Input, Modal, Select, Space, Table, Tag, message, Progress, DatePicker } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import {
  usePmAdhocList,
  useCreatePmAdhoc,
  useCancelPmAdhoc,
  usePmAdhocResults,
} from '@core/hooks/api/usePmAdhoc';
import type { AdhocMode, AdhocStatus, AdhocTask } from '@core/types/pmAdhoc';

const MODE_OPTIONS: { label: string; value: AdhocMode }[] = [
  { label: '单次执行', value: 'oneshot' },
  { label: '持续执行', value: 'continuous' },
];

const GRANULARITY_OPTIONS = ['hourly', 'daily', 'weekly', 'monthly'].map((g) => ({
  label: g,
  value: g,
}));

const statusColor: Record<AdhocStatus, string> = {
  pending: 'default',
  running: 'processing',
  scheduled: 'cyan',
  succeeded: 'success',
  failed: 'error',
  canceled: 'warning',
};

interface CreateForm {
  name: string;
  mode: AdhocMode;
  cronExpr?: string;
  deviceSns: string; // csv
  metricPaths: string; // csv
  granularities: string[];
  window: [dayjs.Dayjs, dayjs.Dayjs];
}

export default function PmAdhocPage() {
  const { data: tasks = [], isLoading } = usePmAdhocList({ refetchInterval: 5000 });
  const createMut = useCreatePmAdhoc();
  const cancelMut = useCancelPmAdhoc();

  const [createOpen, setCreateOpen] = useState(false);
  const [selectedTask, setSelectedTask] = useState<AdhocTask | null>(null);
  const [form] = Form.useForm<CreateForm>();

  const { data: results = [] } = usePmAdhocResults(selectedTask?.id);

  const handleCreate = async () => {
    const v = await form.validateFields();
    await createMut.mutateAsync({
      name: v.name,
      mode: v.mode,
      cronExpr: v.cronExpr,
      deviceSns: v.deviceSns
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      metricPaths: v.metricPaths
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      granularities: v.granularities,
      windowStart: v.window[0].toISOString(),
      windowEnd: v.window[1].toISOString(),
    });
    message.success('任务已创建，worker 将开始执行');
    setCreateOpen(false);
    form.resetFields();
  };

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
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
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

      {/* 创建抽屉 */}
      <Drawer
        title="新建自定义聚合任务"
        width={600}
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        extra={
          <Button type="primary" loading={createMut.isPending} onClick={handleCreate}>
            创建
          </Button>
        }
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            mode: 'oneshot' as AdhocMode,
            granularities: ['hourly'],
            window: [dayjs().subtract(1, 'day'), dayjs()],
          }}
        >
          <Form.Item label="任务名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="模式" name="mode" rules={[{ required: true }]}>
            <Select options={MODE_OPTIONS} />
          </Form.Item>
          <Form.Item shouldUpdate={(p, c) => p.mode !== c.mode}>
            {() =>
              form.getFieldValue('mode') === 'continuous' ? (
                <Form.Item
                  label="Cron 表达式（5 字段：m h dom mon dow）"
                  name="cronExpr"
                  rules={[{ required: true, message: 'continuous 必须填 cron_expr' }]}
                >
                  <Input placeholder="0 * * * *（每小时整点）" />
                </Form.Item>
              ) : null
            }
          </Form.Item>
          <Form.Item label="设备 SN（逗号分隔）" name="deviceSns" rules={[{ required: true }]}>
            <Input placeholder="BLQ-001, BLQ-002" />
          </Form.Item>
          <Form.Item label="指标路径（逗号分隔）" name="metricPaths" rules={[{ required: true }]}>
            <Input placeholder="L.RRC.SuccRate, L.ERAB.SuccRate" />
          </Form.Item>
          <Form.Item label="粒度（多选）" name="granularities" rules={[{ required: true }]}>
            <Select mode="multiple" options={GRANULARITY_OPTIONS} />
          </Form.Item>
          <Form.Item label="时间窗" name="window" rules={[{ required: true }]}>
            <DatePicker.RangePicker showTime style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Drawer>

      {/* 结果查看 */}
      <Drawer
        title={selectedTask ? `结果：${selectedTask.name}` : '结果'}
        width={720}
        open={Boolean(selectedTask)}
        onClose={() => setSelectedTask(null)}
      >
        <Table
          rowKey="id"
          size="small"
          pagination={{ pageSize: 20 }}
          dataSource={results}
          columns={[
            { title: '设备', render: (_, r) => `${r.deviceOui}/${r.deviceSn}` },
            { title: '指标', dataIndex: 'metricPath' },
            { title: '粒度', dataIndex: 'granularity', width: 80 },
            { title: '值', dataIndex: 'metricValue', width: 100 },
            { title: '时间', dataIndex: 'time', width: 180 },
          ]}
        />
      </Drawer>
    </Card>
  );
}
