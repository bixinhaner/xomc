import React, { useCallback, useMemo, useState } from 'react';
import { Button, Card, Space, Tag, App, Drawer, Form, Input, Select } from 'antd';
import { ExportOutlined, PlusOutlined } from '@ant-design/icons';
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

// 告警库项类型（与 alarmApi.ts 中 AlarmLibraryItem 对齐）
interface AlarmLibrary {
  id: string;
  alarmIdentifier: string;
  alarmSource: string;
  eventType: string;
  severity: number; // 1=Critical, 2=Major, 3=Minor, 4=Warning
  enabled: boolean;
  probableCause: string;
  explanation?: string;
  additionalInfo?: Record<string, unknown>;
  carrier?: string;
  technology?: string;
  createdAt: string;
  updatedAt: string;
}

// 严重级别配置
const SEVERITY_CONFIG: Record<number, { color: string; bgColor: string }> = {
  1: { color: '#E53935', bgColor: '#FFEBEE' }, // Critical
  2: { color: '#FB8C00', bgColor: '#FFF3E0' }, // Major
  3: { color: '#FDD835', bgColor: '#FFFDE7' }, // Minor
  4: { color: '#42A5F5', bgColor: '#E3F2FD' }, // Warning
};

// 严重级别选项
const SEVERITY_OPTIONS = [
  { label: '1 - Critical', value: 1 },
  { label: '2 - Major', value: 2 },
  { label: '3 - Minor', value: 3 },
  { label: '4 - Warning', value: 4 },
];

// 事件类型选项
const EVENT_TYPE_OPTIONS = [
  { label: '通信告警', value: 'communicationsAlarm' },
  { label: '设备告警', value: 'equipmentAlarm' },
  { label: '处理失败告警', value: 'processingErrorAlarm' },
  { label: '服务质量告警', value: 'qualityOfServiceAlarm' },
  { label: '环境告警', value: 'environmentalAlarm' },
];

// 表单初始值
const INITIAL_FORM_VALUES = {
  alarmSource: '',
  probableCause: '',
  severity: 2,
  eventType: 'communicationsAlarm',
  explanation: '',
};

export default function AlarmSupportLibrary() {
  const t = useT();
  const { message, modal } = App.useApp();
  const [form] = Form.useForm();
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  // 查询参数：将前端 severity 字符串转为后端数字
  const queryParams = useMemo(() => ({
    page: currentPage,
    pageSize,
    alarmIdentifier: filterParams.alarmIdentifier as string | undefined,
    alarmSource: filterParams.alarmSource as string | undefined,
    severity: filterParams.severity as number | undefined,
    eventType: filterParams.eventType as string | undefined,
    keyword: filterParams.keyword as string | undefined,
  }), [currentPage, pageSize, filterParams]);

  const { data, isLoading, refetch } = useAlarmLibraries(queryParams);
  const createMutation = useCreateAlarmLibrary();
  const updateMutation = useUpdateAlarmLibrary();
  const deleteMutation = useDeleteAlarmLibrary();

  const SEVERITY_LABEL: Record<number, string> = useMemo(() => ({
    1: t('alarm.severity.critical'),
    2: t('alarm.severity.major'),
    3: t('alarm.severity.minor'),
    4: t('alarm.severity.warning'),
  }), [t]);

  // 告警源选项：从已有数据中动态获取
  const alarmSourceOptions = useMemo(() => {
    const sources = new Set<string>();
    (data?.items || []).forEach((item) => {
      if (item.alarmSource) sources.add(item.alarmSource);
    });
    return Array.from(sources).sort();
  }, [data]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    {
      name: 'keyword',
      label: t('alarm.search'),
      type: 'input',
      placeholder: t('alarm.librarySearchPlaceholder'),
    },
    {
      name: 'alarmSource',
      label: t('alarm.alarmSource'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        ...alarmSourceOptions.map((s) => ({ label: s, value: s })),
      ],
    },
    {
      name: 'severity',
      label: t('alarm.severity'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        ...SEVERITY_OPTIONS.map((item) => ({ label: item.label, value: item.value })),
      ],
    },
    {
      name: 'eventType',
      label: t('alarm.eventType'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        ...EVENT_TYPE_OPTIONS.map((item) => ({ label: item.label, value: item.value })),
      ],
    },
  ], [t, alarmSourceOptions]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const handleExport = useCallback(async () => {
    message.success(t('common.exportSuccess'));
  }, [t, message]);

  const handleAdd = useCallback(() => {
    setEditingId(null);
    form.resetFields();
    form.setFieldsValue(INITIAL_FORM_VALUES);
    setDrawerOpen(true);
  }, [form]);

  const handleEdit = useCallback((record: AlarmLibrary) => {
    setEditingId(record.id);
    form.setFieldsValue({
      alarmSource: record.alarmSource,
      probableCause: record.probableCause,
      severity: record.severity,
      eventType: record.eventType,
      explanation: record.explanation || '',
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
      onOk: () => deleteMutation.mutateAsync(id),
    });
  }, [t, message, deleteMutation]);

  const handleFormSubmit = useCallback(async () => {
    try {
      const values = await form.validateFields();
      if (editingId) {
        await updateMutation.mutateAsync({
          id: editingId,
          payload: {
            alarm_source: values.alarmSource,
            probable_cause: values.probableCause,
            severity: values.severity,
            event_type: values.eventType,
            explanation: values.explanation,
          },
        });
        message.success(t('common.updateSuccess'));
      } else {
        await createMutation.mutateAsync({
          alarm_identifier: '',
          alarm_source: values.alarmSource,
          probable_cause: values.probableCause,
          severity: values.severity,
          event_type: values.eventType,
          explanation: values.explanation,
        });
        message.success(t('common.addSuccess'));
      }
      setDrawerOpen(false);
    } catch {
      // 表单验证失败
    }
  }, [editingId, form, t, message, createMutation, updateMutation]);

  const handleDrawerClose = useCallback(() => {
    setDrawerOpen(false);
    setEditingId(null);
    form.resetFields();
  }, [form]);

  const columns = useMemo(
    (): DataTableColumn<AlarmLibrary>[] => [
      {
        key: 'action',
        title: t('table.operation'),
        width: 120,
        fixed: 'right' as const,
        render: (_: unknown, record: AlarmLibrary) => (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              onClick={() => handleEdit(record)}
            >
              {t('common.edit')}
            </Button>
            <Button
              type="link"
              size="small"
              danger
              onClick={() => handleDelete(record.id)}
            >
              {t('common.delete')}
            </Button>
          </Space>
        ),
      },
      {
        key: 'alarmSource',
        title: t('alarm.alarmSource'),
        dataIndex: 'alarmSource',
        width: 110,
      },
      {
        key: 'alarmIdentifier',
        title: t('alarm.alarmIdentifier'),
        dataIndex: 'alarmIdentifier',
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
          const config = SEVERITY_CONFIG[value] || SEVERITY_CONFIG[3];
          return (
            <Tag style={{ color: config?.color, backgroundColor: config?.bgColor, border: 'none' }}>
              {SEVERITY_LABEL[value] || value}
            </Tag>
          );
        },
      },
      {
        key: 'eventType',
        title: t('alarm.eventType'),
        dataIndex: 'eventType',
        width: 150,
      },
      {
        key: 'explanation',
        title: t('alarm.explanation'),
        dataIndex: 'explanation',
        width: 400,
        ellipsis: true,
      },
    ],
    [t, SEVERITY_LABEL, handleEdit, handleDelete],
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

      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<AlarmLibrary>
          tableId="alarm-library-table"
          columns={columns}
          dataSource={data?.items || []}
          loading={isLoading}
          rowKey="id"
          total={data?.total || 0}
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
      </Card>

      <Drawer
        title={editingId ? t('alarm.library.edit') : t('alarm.library.add')}
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
            name="alarmSource"
            label={t('alarm.alarmSource')}
            rules={[{ required: true, message: t('common.required') }]}
          >
            <Select
              placeholder={t('common.pleaseSelect')}
              showSearch
              options={alarmSourceOptions.map((s) => ({ label: s, value: s }))}
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
              options={SEVERITY_OPTIONS.map((item) => ({
                label: item.label,
                value: item.value,
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
              options={EVENT_TYPE_OPTIONS.map((item) => ({
                label: item.label,
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
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
