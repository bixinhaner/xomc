import React, { useCallback, useMemo, useState } from 'react';
import { Button, Space, Tag, App, Drawer, Form, Input, Select, Switch, message } from 'antd';
import { ExportOutlined, PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import {
  useAlarmLibraries,
  useCreateAlarmLibrary,
  useUpdateAlarmLibrary,
  useDeleteAlarmLibrary,
} from '@/hooks/api/useAlarms';

// 告警源选项（对应后端 alarm_source）
const ALARM_SOURCE_OPTIONS = ['ENB', 'GNB', 'CPE', 'eGW', 'RRU', 'BBU', 'DU', 'CU'];

// 严重程度映射：后端 int 1-4 ↔ 前端显示
const SEVERITY_NUM_TO_LABEL: Record<number, string> = {
  1: 'Critical',
  2: 'Major',
  3: 'Minor',
  4: 'Warning',
};

const SEVERITY_LABEL_TO_NUM: Record<string, number> = {
  Critical: 1,
  Major: 2,
  Minor: 3,
  Warning: 4,
};

// 事件类型映射
const EVENT_TYPE_MAP: Record<string, string> = {
  '30000': 'communication',
  '30001': 'qualityOfService',
  '30002': 'processingError',
  '30003': 'device',
  '30004': 'environment',
  '30006': 'performance',
};

const EVENT_TYPE_OPTIONS = Object.entries(EVENT_TYPE_MAP).map(([value, key]) => ({ value, labelKey: `alarm.eventType.${key}` }));

// 严重程度配置
const SEVERITY_CONFIG: Record<string, { color: string; bgColor: string }> = {
  Critical: { color: '#E53935', bgColor: '#FFEBEE' },
  Major: { color: '#FB8C00', bgColor: '#FFF3E0' },
  Minor: { color: '#FDD835', bgColor: '#FFFDE7' },
  Warning: { color: '#42A5F5', bgColor: '#E3F2FD' },
};

type DrawerMode = 'add' | 'edit';

interface LibraryFormValues {
  alarmCode: string;
  alarmSource: string;
  probableCause: string;
  severity: string;
  eventType: string;
  explanation?: string;
  enabled?: boolean;
}

const INITIAL_FORM_VALUES: LibraryFormValues = {
  alarmCode: '',
  alarmSource: '',
  probableCause: '',
  severity: 'Major',
  eventType: '30000',
  explanation: '',
  enabled: true,
};

export default function AlarmSupportLibrary() {
  const t = useT();
  const { modal } = App.useApp();
  const [form] = Form.useForm();
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<DrawerMode>('add');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [exportLoading, setExportLoading] = useState(false);

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    Critical: t('alarm.severity.critical'),
    Major: t('alarm.severity.major'),
    Minor: t('alarm.severity.minor'),
    Warning: t('alarm.severity.warning'),
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    {
      name: 'keyword',
      label: t('alarm.search'),
      type: 'input',
      placeholder: t('alarm.librarySearchPlaceholder'),
    },
    {
      name: 'alarmSource',
      label: t('alarm.deviceTypeName'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        ...ALARM_SOURCE_OPTIONS.map((item) => ({ label: item, value: item })),
      ],
    },
    {
      name: 'severity',
      label: t('alarm.severity'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('alarm.severity.critical'), value: '1' },
        { label: t('alarm.severity.major'), value: '2' },
        { label: t('alarm.severity.minor'), value: '3' },
        { label: t('alarm.severity.warning'), value: '4' },
      ],
    },
    {
      name: 'eventType',
      label: t('alarm.eventType'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        ...EVENT_TYPE_OPTIONS.map((item) => ({ label: t(item.labelKey), value: item.value })),
      ],
    },
  ], [t]);

  // 构建查询参数
  const queryParams = useMemo(() => {
    const params: Record<string, unknown> = {
      page: currentPage,
      pageSize,
    };
    if (filterParams.keyword) params.keyword = filterParams.keyword;
    if (filterParams.alarmSource) params.alarmSource = filterParams.alarmSource;
    if (filterParams.severity) params.severity = String(filterParams.severity);
    if (filterParams.eventType) params.eventType = String(filterParams.eventType);
    return params;
  }, [filterParams, currentPage, pageSize]);

  const { data, isLoading } = useAlarmLibraries(queryParams);
  const createMutation = useCreateAlarmLibrary();
  const updateMutation = useUpdateAlarmLibrary();
  const deleteMutation = useDeleteAlarmLibrary();

  const libraries = data?.items ?? [];
  const total = data?.total ?? 0;

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const handleExport = useCallback(async () => {
    setExportLoading(true);
    try {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      message.success(t('common.exportSuccess'));
    } catch {
      message.error(t('common.exportFailed'));
    } finally {
      setExportLoading(false);
    }
  }, [t, message]);

  const handleAdd = useCallback(() => {
    setEditingId(null);
    setDrawerMode('add');
    form.resetFields();
    form.setFieldsValue(INITIAL_FORM_VALUES);
    setDrawerOpen(true);
  }, [form]);

  const handleEdit = useCallback((record: Record<string, unknown>) => {
    setEditingId(record.id as string);
    setDrawerMode('edit');
    form.setFieldsValue({
      alarmCode: record.alarmCode,
      alarmSource: record.alarmSource,
      probableCause: record.probableCause,
      severity: SEVERITY_NUM_TO_LABEL[record.severity as number] || 'Major',
      eventType: record.eventType,
      explanation: record.explanation || '',
      enabled: record.enabled ?? true,
    });
    setDrawerOpen(true);
  }, [form]);

  const handleDelete = useCallback((id: string) => {
    modal.confirm({
      title: t('common.confirmDelete'),
      content: t('alarm.library.deleteConfirm'),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteMutation.mutateAsync(id);
          message.success(t('common.deleteSuccess'));
        } catch {
          message.error(t('common.deleteFailed'));
        }
      },
    });
  }, [deleteMutation, modal, t, message]);

  const handleFormSubmit = useCallback(async () => {
    try {
      const values = await form.validateFields();
      const severityNum = SEVERITY_LABEL_TO_NUM[values.severity] ?? 2;

      if (drawerMode === 'add') {
        await createMutation.mutateAsync({
          alarmCode: values.alarmCode,
          alarmSource: values.alarmSource,
          eventType: values.eventType,
          severity: severityNum,
          probableCause: values.probableCause,
          explanation: values.explanation || undefined,
          enabled: values.enabled ?? true,
        } as Parameters<typeof createMutation.mutateAsync>[0]);
        message.success(t('common.addSuccess'));
      } else if (editingId) {
        await updateMutation.mutateAsync({
          id: editingId,
          payload: {
            severity: severityNum,
            enabled: values.enabled ?? true,
            probableCause: values.probableCause,
            explanation: values.explanation || undefined,
          },
        });
        message.success(t('common.updateSuccess'));
      }
      setDrawerOpen(false);
    } catch {
      // 表单验证失败
    }
  }, [drawerMode, editingId, form, createMutation, updateMutation, t, message]);

  const handleDrawerClose = useCallback(() => {
    setDrawerOpen(false);
    setEditingId(null);
    form.resetFields();
  }, [form]);

  const columns = useMemo(
    (): DataTableColumn<Record<string, unknown>>[] => [
      {
        key: 'action',
        title: t('table.operation'),
        width: 120,
        render: (_: unknown, record: Record<string, unknown>) => (
          <Space size="small">
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
              style={{ padding: 0 }}
            >
              {t('common.edit')}
            </Button>
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              style={{ padding: 0 }}
              onClick={() => handleDelete(record.id as string)}
            >
              {t('common.delete')}
            </Button>
          </Space>
        ),
      },
      {
        key: 'enabled',
        title: t('alarm.status'),
        dataIndex: 'enabled',
        width: 80,
        render: (value: boolean) => (
          <Tag color={value ? 'green' : 'default'}>
            {value ? t('common.enabled') : t('common.disabled')}
          </Tag>
        ),
      },
      {
        key: 'alarmSource',
        title: t('alarm.deviceTypeName'),
        dataIndex: 'alarmSource',
        width: 110,
      },
      {
        key: 'alarmCode',
        title: t('alarm.alarmIdentifier'),
        dataIndex: 'alarmCode',
        width: 130,
        mono: true,
      },
      {
        key: 'probableCause',
        title: t('alarm.possibleCause'),
        dataIndex: 'probableCause',
        width: 280,
        ellipsis: true,
      },
      {
        key: 'severity',
        title: t('alarm.severity'),
        dataIndex: 'severity',
        width: 100,
        render: (value: number) => {
          const label = SEVERITY_NUM_TO_LABEL[value] || 'Warning';
          const config = SEVERITY_CONFIG[label];
          return (
            <Tag style={{ color: config?.color, backgroundColor: config?.bgColor, border: 'none' }}>
              {SEVERITY_LABEL[label] || label}
            </Tag>
          );
        },
      },
      {
        key: 'eventType',
        title: t('alarm.eventType'),
        dataIndex: 'eventType',
        width: 130,
        render: (value: string) => {
          const typeKey = EVENT_TYPE_MAP[value];
          return typeKey ? t(`alarm.eventType.${typeKey}`) : value;
        },
      },
      {
        key: 'explanation',
        title: t('alarm.explanation'),
        dataIndex: 'explanation',
        width: 400,
        ellipsis: true,
      },
    ],
    [t, SEVERITY_LABEL, handleEdit, handleDelete]
  );

  return (
    <ListPageLayout
      title={t('nav.alarm.library')}
      extra={
        <Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={handleAdd}
          >
            {t('common.add')}
          </Button>
          <Button
            icon={<ExportOutlined />}
            onClick={handleExport}
            loading={exportLoading}
          >
            {t('common.export')}
          </Button>
        </Space>
      }
    >
      <FilterBar
        filterId="alarm-library"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      <DataTable<Record<string, unknown>>
        tableId="alarm-library-table"
        columns={columns}
        dataSource={libraries}
        loading={isLoading}
        rowKey="id"
        total={total}
        pageSize={pageSize}
        currentPage={currentPage}
        onPageChange={(page, size) => {
          setCurrentPage(page);
          setPageSize(size);
        }}
        defaultDensity="default"
        scroll={{ x: 1400, y: 'calc(100vh - 350px)' }}
        showRowNumber
        rowNumberTitle="序号"
      />

      <Drawer
        title={drawerMode === 'edit' ? t('alarm.library.edit') : t('alarm.library.add')}
        open={drawerOpen}
        onClose={handleDrawerClose}
        width={480}
        footer={
          <Space style={{ float: 'right' }}>
            <Button onClick={handleDrawerClose}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={handleFormSubmit} loading={createMutation.isPending || updateMutation.isPending}>
              {t('common.confirm')}
            </Button>
          </Space>
        }
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={INITIAL_FORM_VALUES}
        >
          <Form.Item
            name="alarmCode"
            label={t('alarm.alarmIdentifier')}
            rules={[{ required: true, message: t('common.required') }]}
          >
            <Input placeholder={t('common.pleaseInput')} disabled={drawerMode === 'edit'} />
          </Form.Item>
          <Form.Item
            name="alarmSource"
            label={t('alarm.library.alarmSource')}
            rules={[{ required: true, message: t('common.required') }]}
          >
            <Select
              placeholder={t('common.pleaseSelect')}
              disabled={drawerMode === 'edit'}
              options={ALARM_SOURCE_OPTIONS.map((item) => ({ label: item, value: item }))}
            />
          </Form.Item>
          <Form.Item
            name="probableCause"
            label={t('alarm.possibleCause')}
            rules={[{ required: true, message: t('common.required') }]}
          >
            <Input.TextArea rows={3} placeholder={t('common.pleaseInput')} />
          </Form.Item>
          <Form.Item
            name="severity"
            label={t('alarm.severity')}
            rules={[{ required: true, message: t('common.required') }]}
          >
            <Select
              placeholder={t('common.pleaseSelect')}
              options={Object.entries(SEVERITY_LABEL).map(([value, label]) => ({
                label,
                value,
              }))}
            />
          </Form.Item>
          <Form.Item
            name="eventType"
            label={t('alarm.eventType')}
            rules={[{ required: true, message: t('common.required') }]}
          >
            <Select
              placeholder={t('common.pleaseSelect')}
              disabled={drawerMode === 'edit'}
              options={EVENT_TYPE_OPTIONS.map((item) => ({
                label: t(item.labelKey),
                value: item.value,
              }))}
            />
          </Form.Item>
          <Form.Item
            name="explanation"
            label={t('alarm.explanation')}
          >
            <Input.TextArea rows={4} placeholder={t('common.pleaseInput')} />
          </Form.Item>
          <Form.Item
            name="enabled"
            label={t('alarm.status')}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
