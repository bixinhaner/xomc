import { useCallback, useMemo, useState } from 'react';
import { Button, Dropdown, Form, Input, Modal, Select, Space, Table, Tabs, Tag, message } from 'antd';
import type { MenuProps } from 'antd';
import {
  CheckCircleOutlined,
  CloudSyncOutlined,
  DeleteOutlined,
  EditOutlined,
  MoreOutlined,
  PlusOutlined,
  StopOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import {
  useDataModelList,
  useDataModelStats,
  useCreateDataModel,
  useUpdateDataModel,
  useDeleteDataModel,
  useActivateDataModel,
  useDeprecateDataModel,
  useRefreshDataModelCache,
} from '@core/hooks/api/useDataModels';
import type { DataModel, CreateDataModelRequest, UpdateDataModelRequest } from '@core/services/api/datamodelApi';
import OUIPanel from './OUIPanel';

const STATUS_COLOR: Record<string, string> = {
  draft: 'default',
  active: 'success',
  deprecated: 'warning',
};

const SCOPE_LABEL: Record<string, string> = {
  product: 'Product',
  oui: 'OUI',
  carrier_default: 'Carrier Default',
};

const CARRIER_OPTIONS = [
  { label: 'CMCC', value: 'cmcc' },
  { label: 'CTCC', value: 'ctcc' },
  { label: 'CUCC', value: 'cucc' },
];

const TECH_OPTIONS = [
  { label: 'LTE', value: 'lte' },
  { label: 'NR', value: 'nr' },
];

const STATUS_OPTIONS = [
  { label: 'Draft', value: 'draft' },
  { label: 'Active', value: 'active' },
  { label: 'Deprecated', value: 'deprecated' },
];

const SCOPE_OPTIONS = [
  { label: 'Product', value: 'product' },
  { label: 'OUI', value: 'oui' },
  { label: 'Carrier Default', value: 'carrier_default' },
];

export default function DataModelManagement() {
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [modalOpen, setModalOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<DataModel | null>(null);
  const [form] = Form.useForm();

  const queryParams = useMemo(
    () => ({ ...filters, page: currentPage, pageSize }),
    [filters, currentPage, pageSize]
  );

  const { data, isLoading, refetch } = useDataModelList(queryParams);
  const { data: stats } = useDataModelStats();
  const createMutation = useCreateDataModel();
  const updateMutation = useUpdateDataModel();
  const deleteMutation = useDeleteDataModel();
  const activateMutation = useActivateDataModel();
  const deprecateMutation = useDeprecateDataModel();
  const refreshCacheMutation = useRefreshDataModelCache();

  const dataModels: DataModel[] = data?.items ?? [];
  const total = data?.total ?? 0;

  const handleCreate = useCallback(() => {
    setEditingRecord(null);
    form.resetFields();
    setModalOpen(true);
  }, [form]);

  const handleEdit = useCallback(
    (record: DataModel) => {
      setEditingRecord(record);
      form.setFieldsValue({
        carrier: record.carrier,
        technology: record.technology,
        version: record.version,
        oui: record.oui,
        productClass: record.productClass,
        rootObject: record.rootObject,
        source: record.source,
        specDocumentRef: record.specDocumentRef,
        description: record.description,
        parameterTree: JSON.stringify(record.parameterTree, null, 2),
      });
      setModalOpen(true);
    },
    [form]
  );

  const handleDelete = useCallback(
    (id: string) => {
      Modal.confirm({
        title: 'Confirm Delete',
        content: 'Only draft data models can be deleted. Continue?',
        okType: 'danger',
        onOk: async () => {
          await deleteMutation.mutateAsync(id);
          void message.success('Data model deleted');
        },
      });
    },
    [deleteMutation]
  );

  const handleActivate = useCallback(
    (id: string) => {
      Modal.confirm({
        title: 'Activate Data Model',
        content: 'This will activate this data model and deprecate any existing active model in the same category.',
        onOk: async () => {
          await activateMutation.mutateAsync(id);
          void message.success('Data model activated');
        },
      });
    },
    [activateMutation]
  );

  const handleDeprecate = useCallback(
    (id: string) => {
      Modal.confirm({
        title: 'Deprecate Data Model',
        content: 'This will deprecate the data model. It can no longer be used for new devices.',
        okType: 'danger',
        onOk: async () => {
          await deprecateMutation.mutateAsync(id);
          void message.success('Data model deprecated');
        },
      });
    },
    [deprecateMutation]
  );

  const handleRefreshCache = useCallback(async () => {
    await refreshCacheMutation.mutateAsync();
    void message.success('Cache refreshed');
  }, [refreshCacheMutation]);

  const handleModalOk = useCallback(async () => {
    try {
      const values = await form.validateFields();
      let paramTree: unknown;
      try {
        paramTree = JSON.parse(values.parameterTree as string);
      } catch {
        void message.error('Parameter tree must be valid JSON');
        return;
      }

      if (editingRecord) {
        const updateReq: UpdateDataModelRequest = {
          version: values.version,
          rootObject: values.rootObject,
          parameterTree: paramTree,
          source: values.source,
          specDocumentRef: values.specDocumentRef,
          description: values.description,
        };
        await updateMutation.mutateAsync({ id: editingRecord.id, data: updateReq });
        void message.success('Data model updated');
      } else {
        const createReq: CreateDataModelRequest = {
          carrier: values.carrier,
          technology: values.technology,
          version: values.version,
          oui: values.oui,
          productClass: values.productClass,
          rootObject: values.rootObject,
          parameterTree: paramTree,
          source: values.source,
          specDocumentRef: values.specDocumentRef,
          description: values.description,
        };
        await createMutation.mutateAsync(createReq);
        void message.success('Data model created');
      }
      setModalOpen(false);
    } catch {
      // validation failed
    }
  }, [form, editingRecord, createMutation, updateMutation]);

  const columns = useMemo(
    (): ColumnsType<DataModel> => [
      {
        title: 'Carrier',
        dataIndex: 'carrier',
        key: 'carrier',
        width: 80,
        render: (val: string) => <Tag>{val.toUpperCase()}</Tag>,
      },
      {
        title: 'Technology',
        dataIndex: 'technology',
        key: 'technology',
        width: 90,
        render: (val: string) => val.toUpperCase(),
      },
      {
        title: 'Version',
        dataIndex: 'version',
        key: 'version',
        width: 100,
      },
      {
        title: 'OUI',
        dataIndex: 'oui',
        key: 'oui',
        width: 100,
        render: (val: string) => val || '-',
      },
      {
        title: 'Product Class',
        dataIndex: 'productClass',
        key: 'productClass',
        width: 120,
        render: (val: string) => val || '-',
      },
      {
        title: 'Scope',
        dataIndex: 'scope',
        key: 'scope',
        width: 120,
        render: (val: string) => <Tag color="blue">{SCOPE_LABEL[val] ?? val}</Tag>,
      },
      {
        title: 'Status',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (val: string) => <Tag color={STATUS_COLOR[val] ?? 'default'}>{val}</Tag>,
      },
      {
        title: 'Source',
        dataIndex: 'source',
        key: 'source',
        width: 100,
        ellipsis: true,
      },
      {
        title: 'Description',
        dataIndex: 'description',
        key: 'description',
        width: 180,
        ellipsis: true,
      },
      {
        title: 'Updated',
        dataIndex: 'updatedAt',
        key: 'updatedAt',
        width: 160,
        render: (val: string) => (val ? new Date(val).toLocaleString('zh-CN') : '-'),
      },
      {
        title: 'Actions',
        key: 'actions',
        width: 100,
        fixed: 'right',
        render: (_: unknown, record: DataModel) => {
          const moreItems: MenuProps['items'] = [
            ...(record.status === 'draft'
              ? [{ key: 'activate', label: 'Activate', icon: <CheckCircleOutlined />, onClick: () => handleActivate(record.id) }]
              : []),
            ...(record.status === 'active'
              ? [{ key: 'deprecate', label: 'Deprecate', icon: <StopOutlined />, onClick: () => handleDeprecate(record.id) }]
              : []),
            ...(record.status === 'draft'
              ? [
                  { type: 'divider' as const },
                  { key: 'delete', label: 'Delete', icon: <DeleteOutlined />, danger: true, onClick: () => handleDelete(record.id) },
                ]
              : []),
          ];
          return (
            <Space size={4}>
              <Button type="link" size="small" onClick={() => handleEdit(record)}>
                Edit
              </Button>
              {moreItems.length > 0 && (
                <Dropdown
                  menu={{ items: moreItems }}
                  trigger={['click']}
                >
                  <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
                </Dropdown>
              )}
            </Space>
          );
        },
      },
    ],
    [handleEdit, handleActivate, handleDeprecate, handleDelete],
  );

  const statsBar = stats ? (
    <Space size={24} style={{ marginBottom: 16 }}>
      <span>Total: <strong>{stats.total}</strong></span>
      <span>Active: <Tag color="success">{stats.active}</Tag></span>
      <span>Draft: <Tag>{stats.draft}</Tag></span>
      <span>Deprecated: <Tag color="warning">{stats.deprecated}</Tag></span>
    </Space>
  ) : null;

  const filterBar = (
    <Space wrap style={{ marginBottom: 16 }}>
      <Select
        placeholder="Carrier"
        allowClear
        style={{ width: 120 }}
        options={CARRIER_OPTIONS}
        onChange={(val) => {
          setFilters((f) => ({ ...f, carrier: val ?? '' }));
          setCurrentPage(1);
        }}
      />
      <Select
        placeholder="Technology"
        allowClear
        style={{ width: 120 }}
        options={TECH_OPTIONS}
        onChange={(val) => {
          setFilters((f) => ({ ...f, technology: val ?? '' }));
          setCurrentPage(1);
        }}
      />
      <Select
        placeholder="Status"
        allowClear
        style={{ width: 120 }}
        options={STATUS_OPTIONS}
        onChange={(val) => {
          setFilters((f) => ({ ...f, status: val ?? '' }));
          setCurrentPage(1);
        }}
      />
      <Select
        placeholder="Scope"
        allowClear
        style={{ width: 140 }}
        options={SCOPE_OPTIONS}
        onChange={(val) => {
          setFilters((f) => ({ ...f, scope: val ?? '' }));
          setCurrentPage(1);
        }}
      />
      <Button icon={<CloudSyncOutlined />} onClick={() => void refetch()}>
        Refresh
      </Button>
    </Space>
  );

  return (
    <ListPageLayout
      title="Data Model Management"
      extra={
        <Space>
          <Button
            icon={<ThunderboltOutlined />}
            onClick={() => void handleRefreshCache()}
            loading={refreshCacheMutation.isPending}
          >
            Refresh Cache
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            New Data Model
          </Button>
        </Space>
      }
    >
      <Tabs
        defaultActiveKey="models"
        items={[
          {
            key: 'models',
            label: 'Data Models',
            children: (
              <>
                {statsBar}
                {filterBar}
                <Table<DataModel>
                  columns={columns}
                  dataSource={dataModels}
                  loading={isLoading}
                  rowKey="id"
                  size="small"
                  scroll={{ x: 1400 }}
                  pagination={{
                    current: currentPage,
                    pageSize,
                    total,
                    showSizeChanger: true,
                    showTotal: (t) => `Total ${t} items`,
                    onChange: (page, size) => {
                      setCurrentPage(page);
                      setPageSize(size);
                    },
                  }}
                />
              </>
            ),
          },
          {
            key: 'oui',
            label: 'OUI Registry',
            children: <OUIPanel />,
          },
        ]}
      />

      <Modal
        title={editingRecord ? 'Edit Data Model' : 'New Data Model'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => void handleModalOk()}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        width={640}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Space style={{ width: '100%' }} size={16}>
            <Form.Item
              name="carrier"
              label="Carrier"
              rules={[{ required: true }]}
              style={{ width: 200 }}
            >
              <Select options={CARRIER_OPTIONS} disabled={!!editingRecord} />
            </Form.Item>
            <Form.Item
              name="technology"
              label="Technology"
              rules={[{ required: true }]}
              style={{ width: 200 }}
            >
              <Select options={TECH_OPTIONS} disabled={!!editingRecord} />
            </Form.Item>
          </Space>
          <Form.Item name="version" label="Version" rules={[{ required: true }]}>
            <Input placeholder="e.g. TR-181:2.15" />
          </Form.Item>
          <Space style={{ width: '100%' }} size={16}>
            <Form.Item name="oui" label="OUI" style={{ width: 200 }}>
              <Input placeholder="e.g. 00259E" disabled={!!editingRecord} />
            </Form.Item>
            <Form.Item name="productClass" label="Product Class" style={{ width: 200 }}>
              <Input placeholder="e.g. LTE-PICO" disabled={!!editingRecord} />
            </Form.Item>
          </Space>
          <Form.Item name="rootObject" label="Root Object">
            <Input placeholder="Device." />
          </Form.Item>
          <Form.Item name="source" label="Source">
            <Input placeholder="e.g. BBF, carrier-custom" />
          </Form.Item>
          <Form.Item name="specDocumentRef" label="Spec Document Ref">
            <Input placeholder="e.g. TR-181 Issue 2 Amendment 15" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item
            name="parameterTree"
            label="Parameter Tree (JSON)"
            rules={[{ required: true }]}
          >
            <Input.TextArea rows={6} style={{ fontFamily: 'monospace', fontSize: 12 }} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
