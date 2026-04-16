import { useState, useCallback } from 'react';
import {
  App,
  Button,
  Card,
  Tag,
  Form,
  Input,
  Select,
  Drawer,
  Space,
  Alert,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useApiEndpoints,
  useApiGroups,
  useCreateApiEndpoint,
  useUpdateApiEndpoint,
  useDeleteApiEndpoint,
  useBatchDeleteApiEndpoints,
  useSyncApiEndpoints,
} from '@/hooks/api/useSystem';
import type { ApiEndpoint, ApiEndpointPayload } from '@/types/system';
import { useT } from '@/hooks/useT';

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];

const METHOD_COLORS: Record<string, string> = {
  GET: 'blue',
  POST: 'green',
  PUT: 'orange',
  DELETE: 'red',
  PATCH: 'purple',
};

export default function ApiManagement() {
  const t = useT();
  const { modal, message } = App.useApp();

  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  // Drawer state
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [editingEndpoint, setEditingEndpoint] = useState<ApiEndpoint | null>(null);
  const [form] = Form.useForm();

  // Queries
  const { data, isLoading } = useApiEndpoints({
    page,
    pageSize,
    path: filters.path as string | undefined,
    method: filters.method as string | undefined,
    apiGroup: filters.apiGroup as string | undefined,
    name: filters.name as string | undefined,
  });

  const { data: apiGroups = [] } = useApiGroups();

  // Mutations
  const createEndpoint = useCreateApiEndpoint();
  const updateEndpoint = useUpdateApiEndpoint();
  const deleteEndpoint = useDeleteApiEndpoint();
  const batchDelete = useBatchDeleteApiEndpoints();
  const syncApi = useSyncApiEndpoints();

  // Filter fields
  const filterFields: FilterField[] = [
    {
      name: 'path',
      label: t('api.path'),
      type: 'input',
      placeholder: t('api.pathPlaceholder'),
    },
    {
      name: 'name',
      label: t('api.name'),
      type: 'input',
      placeholder: t('api.namePlaceholder'),
    },
    {
      name: 'apiGroup',
      label: t('api.group'),
      type: 'select',
      placeholder: t('common.pleaseSelect'),
      options: apiGroups.map((g) => ({ label: g, value: g })),
    },
    {
      name: 'method',
      label: t('api.method'),
      type: 'select',
      placeholder: t('common.pleaseSelect'),
      options: HTTP_METHODS.map((m) => ({ label: m, value: m })),
    },
  ];

  // Table columns
  const columns: DataTableColumn<ApiEndpoint>[] = [
    {
      title: t('api.path'),
      dataIndex: 'path',
      key: 'path',
      ellipsis: true,
      width: 280,
    },
    {
      title: t('api.group'),
      dataIndex: 'apiGroup',
      key: 'apiGroup',
      width: 120,
      render: (val: string) =>
        val ? <Tag color="cyan">{val}</Tag> : <span style={{ color: '#999' }}>-</span>,
    },
    {
      title: t('api.name'),
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: t('api.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('api.method'),
      dataIndex: 'method',
      key: 'method',
      width: 100,
      render: (val: string) => (
        <Tag color={METHOD_COLORS[val] ?? 'default'}>{val}</Tag>
      ),
    },
    {
      title: t('common.operation'),
      key: 'action',
      width: 120,
      fixed: 'right',
      render: (_: unknown, record: ApiEndpoint) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            {t('common.edit')}
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record)}
          >
            {t('common.delete')}
          </Button>
        </Space>
      ),
    },
  ];

  const handleEdit = useCallback((record: ApiEndpoint) => {
    setEditingEndpoint(record);
    form.setFieldsValue({
      path: record.path,
      method: record.method,
      apiGroup: record.apiGroup,
      name: record.name,
      description: record.description,
    });
    setDrawerVisible(true);
  }, [form]);

  const handleAdd = useCallback(() => {
    setEditingEndpoint(null);
    form.resetFields();
    setDrawerVisible(true);
  }, [form]);

  const handleDelete = useCallback((record: ApiEndpoint) => {
    modal.confirm({
      title: t('api.deleteConfirm'),
      onOk: () => {
        deleteEndpoint.mutate(record.id, {
          onSuccess: () => message.success(t('common.deleteSuccess')),
          onError: () => message.error(t('common.deleteFailed')),
        });
      },
    });
  }, [deleteEndpoint, modal, message, t]);

  const handleBatchDelete = useCallback(() => {
    const ids = selectedKeys as string[];
    if (ids.length === 0) {
      message.warning(t('common.selectAtLeastOne'));
      return;
    }
    modal.confirm({
      title: t('api.batchDeleteConfirm', { count: ids.length }),
      onOk: () => {
        batchDelete.mutate(ids, {
          onSuccess: () => {
            message.success(t('common.deleteSuccess'));
            setSelectedKeys([]);
          },
          onError: () => message.error(t('common.deleteFailed')),
        });
      },
    });
  }, [selectedKeys, batchDelete, modal, message, t]);

  const handleSync = useCallback(() => {
    modal.confirm({
      title: t('api.syncConfirm'),
      content: t('api.syncConfirmContent'),
      onOk: () => {
        syncApi.mutate(undefined, {
          onSuccess: (result) => {
            message.success(
              t('api.syncSuccess', {
                created: result?.created ?? 0,
                updated: result?.updated ?? 0,
                total: result?.total ?? 0,
              })
            );
          },
          onError: () => message.error(t('common.operationFailed')),
        });
      },
    });
  }, [syncApi, modal, message, t]);

  const handleDrawerSubmit = useCallback(async () => {
    try {
      const values = await form.validateFields();
      const payload: ApiEndpointPayload = {
        path: values.path,
        method: values.method,
        apiGroup: values.apiGroup,
        name: values.name,
        description: values.description,
      };

      if (editingEndpoint) {
        updateEndpoint.mutate(
          { id: editingEndpoint.id, payload },
          {
            onSuccess: () => {
              message.success(t('common.saveSuccess'));
              setDrawerVisible(false);
            },
            onError: () => message.error(t('common.saveFailed')),
          }
        );
      } else {
        createEndpoint.mutate(payload, {
          onSuccess: () => {
            message.success(t('common.createSuccess'));
            setDrawerVisible(false);
          },
          onError: () => message.error(t('common.saveFailed')),
        });
      }
    } catch {
      // validation error — do nothing
    }
  }, [form, editingEndpoint, createEndpoint, updateEndpoint, message, t]);

  const isSubmitting = createEndpoint.isPending || updateEndpoint.isPending;

  const toolbar = (
    <Space>
      <Button
        type="primary"
        icon={<PlusOutlined />}
        onClick={handleAdd}
      >
        {t('api.addEndpoint')}
      </Button>
      <Button
        danger
        icon={<DeleteOutlined />}
        disabled={selectedKeys.length === 0}
        onClick={handleBatchDelete}
        loading={batchDelete.isPending}
      >
        {t('api.batchDelete')}
      </Button>
      <Button
        icon={<SyncOutlined />}
        onClick={handleSync}
        loading={syncApi.isPending}
      >
        {t('api.sync')}
      </Button>
    </Space>
  );

  return (
    <ListPageLayout>
      <FilterBar
        filterId="api-management"
        fields={filterFields}
        onSearch={(vals) => {
          setFilters(vals);
          setPage(1);
        }}
        onReset={() => {
          setFilters({});
          setPage(1);
        }}
      />

      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<ApiEndpoint>
          tableId="api-management"
          columns={columns}
          dataSource={data?.items ?? []}
          loading={isLoading}
          rowKey="id"
          selectable
          extraToolbarLeft={toolbar}
          total={data?.total ?? 0}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, ps) => {
            setPage(p);
            setPageSize(ps);
          }}
          selectedRowKeys={selectedKeys}
          onSelectionChange={(keys) => setSelectedKeys(keys)}
          scroll={{ x: 900 }}
        />
      </Card>

      <Drawer
        title={editingEndpoint ? t('api.editEndpoint') : t('api.addEndpoint')}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
        width={520}
        footer={
          <Space style={{ justifyContent: 'flex-end', display: 'flex' }}>
            <Button onClick={() => setDrawerVisible(false)}>{t('common.cancel')}</Button>
            <Button type="primary" loading={isSubmitting} onClick={handleDrawerSubmit}>
              {t('common.confirm')}
            </Button>
          </Space>
        }
      >
        <Alert
          type="warning"
          showIcon
          message={t('api.addTip')}
          style={{ marginBottom: 20 }}
        />
        <Form form={form} layout="vertical">
          <Form.Item
            label={t('api.path')}
            name="path"
            rules={[{ required: true, message: t('api.pathRequired') }]}
          >
            <Input placeholder={t('api.pathPlaceholder')} />
          </Form.Item>

          <Form.Item
            label={t('api.method')}
            name="method"
            rules={[{ required: true, message: t('api.methodRequired') }]}
          >
            <Select placeholder={t('common.pleaseSelect')}>
              {HTTP_METHODS.map((m) => (
                <Select.Option key={m} value={m}>
                  <Tag color={METHOD_COLORS[m] ?? 'default'} style={{ marginRight: 0 }}>
                    {m}
                  </Tag>
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            label={t('api.group')}
            name="apiGroup"
            rules={[{ required: true, message: t('api.groupRequired') }]}
          >
            <Select
              placeholder={t('api.groupPlaceholder')}
              showSearch
              allowClear
              mode={undefined}
              options={apiGroups.map((g) => ({ label: g, value: g }))}
            />
          </Form.Item>

          <Form.Item
            label={t('api.name')}
            name="name"
            rules={[{ required: true, message: t('api.nameRequired') }]}
          >
            <Input placeholder={t('api.namePlaceholder')} />
          </Form.Item>

          <Form.Item label={t('api.description')} name="description">
            <Input.TextArea rows={3} placeholder={t('api.descriptionPlaceholder')} />
          </Form.Item>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
