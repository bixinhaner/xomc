import { useState, useCallback, useRef } from 'react';
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
} from '@core/hooks/api/useSystem';
import type { ApiEndpoint, ApiEndpointPayload } from '@core/types/system';
import { useT } from '@/hooks/useT';

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];

const METHOD_COLORS: Record<string, string> = {
  GET: 'blue',
  POST: 'green',
  PUT: 'orange',
  DELETE: 'red',
  PATCH: 'purple',
};

// FilterBar sessionStorage key（与 <FilterBar filterId="api-management" /> 对齐）。
const FILTER_STORAGE_KEY = 'omc_filter_api-management';

export default function ApiManagement() {
  const t = useT();
  const { modal, message } = App.useApp();

  // 进入页面时把 sessionStorage 里残留的 apiGroup 清掉，让筛选区的"API 分组"
  // 恢复为空——其他字段（path / name / method）仍由 FilterBar 自身的恢复逻辑
  // 还原。必须在子组件 FilterBar 的 mount useEffect 之前完成清理，因此放在
  // 渲染函数顶部并用 ref 保证只跑一次。
  const filterCleanupDoneRef = useRef<true | null>(null);
  if (filterCleanupDoneRef.current == null) {
    filterCleanupDoneRef.current = true;
    try {
      const raw = sessionStorage.getItem(FILTER_STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as Record<string, unknown>;
        if (parsed.apiGroup !== undefined) {
          delete parsed.apiGroup;
          sessionStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify(parsed));
        }
      }
    } catch {
      // ignore — 仅影响首次加载时的"API 分组"默认空逻辑
    }
  }

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
      width: 240,
    },
    {
      name: 'name',
      label: t('api.name'),
      type: 'input',
      placeholder: t('api.namePlaceholder'),
      width: 240,
    },
    {
      name: 'apiGroup',
      label: t('api.group'),
      type: 'select',
      placeholder: t('common.pleaseSelect'),
      options: apiGroups.map((g) => ({ label: g, value: g })),
      width: 180,
    },
    {
      name: 'method',
      label: t('api.method'),
      type: 'select',
      placeholder: t('common.pleaseSelect'),
      options: HTTP_METHODS.map((m) => ({ label: m, value: m })),
      width: 180,
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
      width: 180,
      render: (val: unknown) => {
        const v = val as string;
        return v ? <Tag color="cyan">{v}</Tag> : <span style={{ color: '#999' }}>-</span>;
      },
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
      render: (val: unknown) => {
        const v = String(val ?? '');
        return <Tag color={METHOD_COLORS[v] ?? 'default'}>{v}</Tag>;
      },
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
      {/* 2026-06-03 用户决策:"新增API"放"同步API"之后;按钮文案精简为"新增" */}
      <Button
        type="primary"
        icon={<PlusOutlined />}
        onClick={handleAdd}
      >
        {t('common.add')}
      </Button>
    </Space>
  );

  return (
    <ListPageLayout>
      {/* 2026-06-03 用户决策:筛选与操作按钮(新增/批量删除/同步)同一行,筛选靠左、按钮靠右。 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 12 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
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
        </div>
        <div style={{ flexShrink: 0 }}>{toolbar}</div>
      </div>

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
          hideToolbar
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
