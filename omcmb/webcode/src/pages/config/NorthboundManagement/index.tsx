import { useCallback, useMemo, useState } from 'react';
import {
  Button,
  Card,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  message,
} from 'antd';
import {
  ApiOutlined,
  PlusOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import {
  usePushTargets,
  useAddPushTarget,
  useRemovePushTarget,
  useFullSync,
  useIncrementalSync,
} from '@/hooks/api/useNorthbound';
import type { PushTarget, AddPushTargetRequest } from '@/services/api/northboundApi';

const AUTH_TYPE_OPTIONS = [
  { label: 'None', value: 'none' },
  { label: 'Bearer Token', value: 'bearer' },
  { label: 'Basic Auth', value: 'basic' },
];

const FORMAT_OPTIONS = [
  { label: 'JSON', value: 'json' },
  { label: 'XML', value: 'xml' },
];

const DATA_TYPE_OPTIONS = [
  { label: 'Alarm', value: 'alarm' },
  { label: 'PM', value: 'pm' },
  { label: 'Config', value: 'config' },
  { label: 'Device', value: 'device' },
];

const SYNC_TYPE_OPTIONS = [
  { label: 'Device', value: 'device' },
  { label: 'Alarm', value: 'alarm' },
  { label: 'PM', value: 'pm' },
  { label: 'Config', value: 'config' },
];

export default function NorthboundManagement() {
  const [modalOpen, setModalOpen] = useState(false);
  const [syncModalOpen, setSyncModalOpen] = useState(false);
  const [syncType, setSyncType] = useState<'full' | 'incremental'>('full');
  const [form] = Form.useForm();
  const [syncForm] = Form.useForm();

  const { data: targets, isLoading, refetch } = usePushTargets();
  const addMutation = useAddPushTarget();
  const removeMutation = useRemovePushTarget();
  const fullSyncMutation = useFullSync();
  const incrementalSyncMutation = useIncrementalSync();

  const pushTargets: PushTarget[] = targets ?? [];

  const handleAddTarget = useCallback(async () => {
    try {
      const values = await form.validateFields();
      const req: AddPushTargetRequest = {
        id: values.id,
        url: values.url,
        authType: values.authType,
        authToken: values.authToken,
        dataTypes: values.dataTypes,
        format: values.format,
        batchSize: values.batchSize,
        retryCount: values.retryCount,
        enabled: values.enabled ?? true,
      };
      await addMutation.mutateAsync(req);
      void message.success('Push target added');
      setModalOpen(false);
      form.resetFields();
    } catch {
      // validation failed
    }
  }, [form, addMutation]);

  const handleRemoveTarget = useCallback(
    (id: string) => {
      Modal.confirm({
        title: 'Remove Push Target',
        content: `Are you sure you want to remove push target "${id}"?`,
        okType: 'danger',
        onOk: async () => {
          await removeMutation.mutateAsync(id);
          void message.success('Push target removed');
        },
      });
    },
    [removeMutation]
  );

  const handleSync = useCallback(async () => {
    try {
      const values = await syncForm.validateFields();
      if (syncType === 'full') {
        const result = await fullSyncMutation.mutateAsync(values.dataType);
        void message.success(`Full sync completed: ${result.total} items synced`);
      } else {
        const since = values.since?.toISOString();
        if (!since) {
          void message.error('Please select a "since" timestamp');
          return;
        }
        const result = await incrementalSyncMutation.mutateAsync({
          dataType: values.dataType,
          since,
        });
        void message.success(`Incremental sync completed: ${result.total} items synced`);
      }
      setSyncModalOpen(false);
      syncForm.resetFields();
    } catch {
      // validation failed
    }
  }, [syncForm, syncType, fullSyncMutation, incrementalSyncMutation]);

  const columns = useMemo(
    (): ColumnsType<PushTarget> => [
      {
        title: 'ID',
        dataIndex: 'id',
        key: 'id',
        width: 120,
        render: (val: string) => <span style={{ fontFamily: 'monospace' }}>{val}</span>,
      },
      {
        title: 'URL',
        dataIndex: 'url',
        key: 'url',
        width: 280,
        ellipsis: true,
        render: (val: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{val}</span>,
      },
      {
        title: 'Auth Type',
        dataIndex: 'authType',
        key: 'authType',
        width: 100,
        render: (val: string) => <Tag>{val}</Tag>,
      },
      {
        title: 'Data Types',
        dataIndex: 'dataTypes',
        key: 'dataTypes',
        width: 200,
        render: (vals: string[]) => (
          <Space size={4} wrap>
            {(vals || []).map((t) => (
              <Tag key={t} color="blue">{t}</Tag>
            ))}
          </Space>
        ),
      },
      {
        title: 'Format',
        dataIndex: 'format',
        key: 'format',
        width: 80,
        render: (val: string) => val.toUpperCase(),
      },
      {
        title: 'Batch Size',
        dataIndex: 'batchSize',
        key: 'batchSize',
        width: 100,
      },
      {
        title: 'Retries',
        dataIndex: 'retryCount',
        key: 'retryCount',
        width: 80,
      },
      {
        title: 'Enabled',
        dataIndex: 'enabled',
        key: 'enabled',
        width: 80,
        render: (val: boolean) => (
          <Tag color={val ? 'success' : 'default'}>{val ? 'Yes' : 'No'}</Tag>
        ),
      },
      {
        title: 'Actions',
        key: 'actions',
        width: 80,
        fixed: 'right',
        render: (_: unknown, record: PushTarget) => (
          <Button
            type="link"
            size="small"
            danger
            onClick={() => handleRemoveTarget(record.id)}
          >
            Remove
          </Button>
        ),
      },
    ],
    [handleRemoveTarget]
  );

  const pushTargetsTab = (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<SyncOutlined />} onClick={() => void refetch()}>
          Refresh
        </Button>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
          Add Push Target
        </Button>
      </Space>

      <Table<PushTarget>
        columns={columns}
        dataSource={pushTargets}
        loading={isLoading}
        rowKey="id"
        size="small"
        scroll={{ x: 1200 }}
        pagination={false}
      />
    </>
  );

  const syncTab = (
    <Card>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <div>
          Trigger full or incremental data synchronization to northbound/OSS systems.
          Full sync exports all data of the selected type. Incremental sync exports
          only data changed since the specified timestamp.
        </div>
        <Space>
          <Button
            type="primary"
            icon={<SyncOutlined />}
            onClick={() => {
              setSyncType('full');
              setSyncModalOpen(true);
            }}
          >
            Full Sync
          </Button>
          <Button
            icon={<SyncOutlined />}
            onClick={() => {
              setSyncType('incremental');
              setSyncModalOpen(true);
            }}
          >
            Incremental Sync
          </Button>
        </Space>
      </Space>
    </Card>
  );

  return (
    <ListPageLayout
      title="Northbound/OSS Management"
      subtitle="Manage northbound push targets and data synchronization"
      extra={
        <Tag icon={<ApiOutlined />} color="processing">
          F08
        </Tag>
      }
    >
      <Tabs
        defaultActiveKey="targets"
        items={[
          { key: 'targets', label: 'Push Targets', children: pushTargetsTab },
          { key: 'sync', label: 'Data Sync', children: syncTab },
        ]}
      />

      {/* Add Push Target Modal */}
      <Modal
        title="Add Push Target"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => void handleAddTarget()}
        confirmLoading={addMutation.isPending}
        width={560}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }} initialValues={{ authType: 'none', format: 'json', batchSize: 100, retryCount: 3, enabled: true }}>
          <Form.Item name="id" label="Target ID" rules={[{ required: true }]}>
            <Input placeholder="e.g. oss-primary" />
          </Form.Item>
          <Form.Item name="url" label="URL" rules={[{ required: true, type: 'url' }]}>
            <Input placeholder="https://oss.example.com/api/v1/push" />
          </Form.Item>
          <Space style={{ width: '100%' }} size={16}>
            <Form.Item name="authType" label="Auth Type" style={{ width: 200 }}>
              <Select options={AUTH_TYPE_OPTIONS} />
            </Form.Item>
            <Form.Item name="authToken" label="Auth Token" style={{ width: 300 }}>
              <Input.Password placeholder="Token or credentials" />
            </Form.Item>
          </Space>
          <Form.Item name="dataTypes" label="Data Types" rules={[{ required: true }]}>
            <Select mode="multiple" options={DATA_TYPE_OPTIONS} placeholder="Select data types" />
          </Form.Item>
          <Space style={{ width: '100%' }} size={16}>
            <Form.Item name="format" label="Format" style={{ width: 120 }}>
              <Select options={FORMAT_OPTIONS} />
            </Form.Item>
            <Form.Item name="batchSize" label="Batch Size" style={{ width: 120 }}>
              <InputNumber min={1} max={10000} />
            </Form.Item>
            <Form.Item name="retryCount" label="Retry Count" style={{ width: 120 }}>
              <InputNumber min={0} max={10} />
            </Form.Item>
          </Space>
          <Form.Item name="enabled" label="Enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* Sync Modal */}
      <Modal
        title={syncType === 'full' ? 'Full Sync' : 'Incremental Sync'}
        open={syncModalOpen}
        onCancel={() => setSyncModalOpen(false)}
        onOk={() => void handleSync()}
        confirmLoading={fullSyncMutation.isPending || incrementalSyncMutation.isPending}
        destroyOnClose
      >
        <Form form={syncForm} layout="vertical" style={{ marginTop: 16 }} initialValues={{ dataType: 'device' }}>
          <Form.Item name="dataType" label="Data Type" rules={[{ required: true }]}>
            <Select options={SYNC_TYPE_OPTIONS} />
          </Form.Item>
          {syncType === 'incremental' && (
            <Form.Item
              name="since"
              label="Since (RFC3339)"
              rules={[{ required: true, message: 'Please select a timestamp' }]}
            >
              <DatePicker showTime style={{ width: '100%' }} />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
