import { useCallback, useMemo, useState } from 'react';
import { Button, Form, Input, Modal, Progress, Select, Space, Table, Tag, message } from 'antd';
import {
  PlusOutlined,
  RedoOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import {
  useProvisioningTasks,
  useCreateProvisioningTask,
  useRetryProvisioningTask,
} from '@/hooks/api/useProvisioning';
import type { ProvisioningTask } from '@/services/api/provisionApi';

const STATUS_COLOR: Record<string, string> = {
  discovered: 'default',
  identifying: 'processing',
  matching: 'processing',
  configuring: 'processing',
  verifying: 'processing',
  completed: 'success',
  failed: 'error',
};

const STATUS_OPTIONS = [
  { label: 'Discovered', value: 'discovered' },
  { label: 'Identifying', value: 'identifying' },
  { label: 'Matching', value: 'matching' },
  { label: 'Configuring', value: 'configuring' },
  { label: 'Verifying', value: 'verifying' },
  { label: 'Completed', value: 'completed' },
  { label: 'Failed', value: 'failed' },
];

export default function AutoProvisioning() {
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [deviceIdFilter, setDeviceIdFilter] = useState<string>('');
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [selectedTask, setSelectedTask] = useState<ProvisioningTask | null>(null);
  const [form] = Form.useForm();

  const queryParams = useMemo(
    () => ({
      page: currentPage,
      pageSize,
      status: statusFilter || undefined,
      deviceId: deviceIdFilter || undefined,
    }),
    [currentPage, pageSize, statusFilter, deviceIdFilter]
  );

  const { data, isLoading, refetch } = useProvisioningTasks(queryParams);
  const createMutation = useCreateProvisioningTask();
  const retryMutation = useRetryProvisioningTask();

  const tasks: ProvisioningTask[] = data?.items ?? [];
  const total = data?.total ?? 0;

  const handleCreate = useCallback(async () => {
    try {
      const values = await form.validateFields();
      await createMutation.mutateAsync({ deviceId: values.deviceId });
      void message.success('Provisioning task created');
      setCreateModalOpen(false);
      form.resetFields();
    } catch {
      // validation failed
    }
  }, [form, createMutation]);

  const handleRetry = useCallback(
    (id: string) => {
      Modal.confirm({
        title: 'Retry Provisioning',
        content: 'This will create a new provisioning task for the same device. Continue?',
        onOk: async () => {
          await retryMutation.mutateAsync(id);
          void message.success('Retry task created');
        },
      });
    },
    [retryMutation]
  );

  const handleViewDetail = useCallback((task: ProvisioningTask) => {
    setSelectedTask(task);
    setDetailModalOpen(true);
  }, []);

  const columns = useMemo(
    (): ColumnsType<ProvisioningTask> => [
      {
        title: 'Task ID',
        dataIndex: 'id',
        key: 'id',
        width: 280,
        render: (val: string) => (
          <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{val}</span>
        ),
      },
      {
        title: 'Device ID',
        dataIndex: 'deviceId',
        key: 'deviceId',
        width: 280,
        render: (val: string) => (
          <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{val}</span>
        ),
      },
      {
        title: 'Status',
        dataIndex: 'status',
        key: 'status',
        width: 120,
        render: (val: string) => <Tag color={STATUS_COLOR[val] ?? 'default'}>{val}</Tag>,
      },
      {
        title: 'Progress',
        key: 'progress',
        width: 160,
        render: (_: unknown, record: ProvisioningTask) => {
          const pct = record.totalSteps > 0
            ? Math.round((record.currentStep / record.totalSteps) * 100)
            : 0;
          return (
            <Progress
              percent={pct}
              size="small"
              status={record.status === 'failed' ? 'exception' : record.status === 'completed' ? 'success' : 'active'}
              format={() => `${record.currentStep}/${record.totalSteps}`}
            />
          );
        },
      },
      {
        title: 'Retries',
        key: 'retries',
        width: 90,
        render: (_: unknown, record: ProvisioningTask) =>
          `${record.retryCount}/${record.maxRetries}`,
      },
      {
        title: 'Created',
        dataIndex: 'createdAt',
        key: 'createdAt',
        width: 160,
        render: (val: string) => (val ? new Date(val).toLocaleString('zh-CN') : '-'),
      },
      {
        title: 'Completed',
        dataIndex: 'completedAt',
        key: 'completedAt',
        width: 160,
        render: (val: string | null) => (val ? new Date(val).toLocaleString('zh-CN') : '-'),
      },
      {
        title: 'Actions',
        key: 'actions',
        width: 160,
        fixed: 'right',
        render: (_: unknown, record: ProvisioningTask) => (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => handleViewDetail(record)}>
              Detail
            </Button>
            {record.status === 'failed' && (
              <Button
                type="link"
                size="small"
                icon={<RedoOutlined />}
                onClick={() => handleRetry(record.id)}
              >
                Retry
              </Button>
            )}
          </Space>
        ),
      },
    ],
    [handleViewDetail, handleRetry]
  );

  return (
    <ListPageLayout
      title="Auto-Provisioning"
      subtitle="Manage automatic device provisioning tasks (F09)"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
          New Task
        </Button>
      }
    >
      <Space wrap style={{ marginBottom: 16 }}>
        <Select
          placeholder="Status"
          allowClear
          style={{ width: 140 }}
          options={STATUS_OPTIONS}
          onChange={(val) => {
            setStatusFilter(val ?? '');
            setCurrentPage(1);
          }}
        />
        <Input.Search
          placeholder="Device ID"
          allowClear
          style={{ width: 300 }}
          onSearch={(val) => {
            setDeviceIdFilter(val);
            setCurrentPage(1);
          }}
        />
        <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
          Refresh
        </Button>
      </Space>

      <Table<ProvisioningTask>
        columns={columns}
        dataSource={tasks}
        loading={isLoading}
        rowKey="id"
        size="small"
        scroll={{ x: 1400 }}
        pagination={{
          current: currentPage,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (t) => `Total ${t} tasks`,
          onChange: (page, size) => {
            setCurrentPage(page);
            setPageSize(size);
          },
        }}
      />

      {/* Create Task Modal */}
      <Modal
        title="Create Provisioning Task"
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
        onOk={() => void handleCreate()}
        confirmLoading={createMutation.isPending}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="deviceId"
            label="Device ID (UUID)"
            rules={[{ required: true, message: 'Device ID is required' }]}
          >
            <Input placeholder="Enter device UUID" style={{ fontFamily: 'monospace' }} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Detail Modal */}
      <Modal
        title="Provisioning Task Detail"
        open={detailModalOpen}
        onCancel={() => setDetailModalOpen(false)}
        footer={null}
        width={560}
      >
        {selectedTask && (
          <div style={{ lineHeight: 2 }}>
            <div><strong>Task ID:</strong> <code>{selectedTask.id}</code></div>
            <div><strong>Device ID:</strong> <code>{selectedTask.deviceId}</code></div>
            <div><strong>Template ID:</strong> {selectedTask.templateId ?? '-'}</div>
            <div>
              <strong>Status:</strong>{' '}
              <Tag color={STATUS_COLOR[selectedTask.status] ?? 'default'}>
                {selectedTask.status}
              </Tag>
            </div>
            <div>
              <strong>Progress:</strong> Step {selectedTask.currentStep} / {selectedTask.totalSteps}
            </div>
            <div>
              <strong>Retries:</strong> {selectedTask.retryCount} / {selectedTask.maxRetries}
            </div>
            {selectedTask.errorMessage && (
              <div>
                <strong>Error:</strong>{' '}
                <Tag color="error">{selectedTask.errorMessage}</Tag>
              </div>
            )}
            <div>
              <strong>Started:</strong>{' '}
              {selectedTask.startedAt ? new Date(selectedTask.startedAt).toLocaleString('zh-CN') : '-'}
            </div>
            <div>
              <strong>Completed:</strong>{' '}
              {selectedTask.completedAt ? new Date(selectedTask.completedAt).toLocaleString('zh-CN') : '-'}
            </div>
            <div>
              <strong>Created:</strong> {new Date(selectedTask.createdAt).toLocaleString('zh-CN')}
            </div>
            <div>
              <strong>Updated:</strong> {new Date(selectedTask.updatedAt).toLocaleString('zh-CN')}
            </div>
          </div>
        )}
      </Modal>
    </ListPageLayout>
  );
}
