import { useState, useMemo } from 'react';
import { Button, Tag, Space, Modal, Form, Input, Select, DatePicker, message, Popconfirm } from 'antd';
import { PlusOutlined, PlayCircleOutlined, DeleteOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useSoftwareVersions } from '@/hooks/api/useSoftware';
import { useT } from '@/hooks/useT';

type ActivationStatus = 'pending' | 'activating' | 'activated' | 'failed' | 'scheduled';
type ActivationMethod = 'immediate' | 'scheduled';

interface ActivationPlanRecord {
  id: string;
  planName: string;
  targetVersion: string;
  deviceRange: number;
  activationMethod: ActivationMethod;
  status: ActivationStatus;
  scheduledTime?: string;
  executedTime?: string;
  creator: string;
}

const mockActivationPlans: ActivationPlanRecord[] = [
  {
    id: 'act-001',
    planName: '北京eNB版本激活计划',
    targetVersion: 'V100R011C10SPC200',
    deviceRange: 25,
    activationMethod: 'immediate',
    status: 'activated',
    executedTime: '2024-06-01T10:00:00.000Z',
    creator: 'admin',
  },
  {
    id: 'act-002',
    planName: '上海gNB测试版激活',
    targetVersion: 'V200R001C10SPC100',
    deviceRange: 5,
    activationMethod: 'scheduled',
    status: 'scheduled',
    scheduledTime: '2024-06-15T02:00:00.000Z',
    creator: 'operator1',
  },
  {
    id: 'act-003',
    planName: '广州RRU固件激活',
    targetVersion: 'V100R011C10SPC100',
    deviceRange: 48,
    activationMethod: 'scheduled',
    status: 'pending',
    scheduledTime: '2024-06-20T03:00:00.000Z',
    creator: 'admin',
  },
];

const statusColorMap: Record<ActivationStatus, string> = {
  pending: 'default',
  activating: 'processing',
  activated: 'green',
  failed: 'red',
  scheduled: 'blue',
};

const statusLabelKeyMap: Record<ActivationStatus, string> = {
  pending: 'status.pending',
  activating: 'status.running',
  activated: 'status.success',
  failed: 'status.failed',
  scheduled: 'status.enabled',
};

export default function ActivationPlan() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createVisible, setCreateVisible] = useState(false);
  const [form] = Form.useForm();
  const [plans, setPlans] = useState<ActivationPlanRecord[]>(mockActivationPlans);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'planName', label: t('table.name'), type: 'input', placeholder: t('common.placeholder') },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.pending'), value: 'pending' },
        { label: t('status.running'), value: 'activating' },
        { label: t('status.success'), value: 'activated' },
        { label: t('status.failed'), value: 'failed' },
        { label: t('status.enabled'), value: 'scheduled' },
      ],
    },
    {
      name: 'activationMethod',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: t('common.execute'), value: 'immediate' },
        { label: t('common.deploy'), value: 'scheduled' },
      ],
    },
  ], [t]);

  const { data: versionsData } = useSoftwareVersions({ page: 1, pageSize: 100 });

  const filtered = plans.filter((p) => {
    if (filters.status && p.status !== filters.status) return false;
    if (filters.activationMethod && p.activationMethod !== filters.activationMethod) return false;
    if (filters.planName && !p.planName.includes(String(filters.planName))) return false;
    return true;
  });

  const handleCreate = () => {
    form.validateFields().then((vals) => {
      const newPlan: ActivationPlanRecord = {
        id: `act-${Date.now()}`,
        planName: vals.planName as string,
        targetVersion: vals.targetVersion as string,
        deviceRange: Number(vals.deviceRange ?? 0),
        activationMethod: vals.activationMethod as ActivationMethod,
        status: vals.activationMethod === 'scheduled' ? 'scheduled' : 'pending',
        scheduledTime: vals.scheduledTime ? String(vals.scheduledTime) : undefined,
        creator: 'admin',
      };
      setPlans((prev) => [newPlan, ...prev]);
      void message.success(t('common.save'));
      setCreateVisible(false);
      form.resetFields();
    });
  };

  const columns: DataTableColumn<ActivationPlanRecord & Record<string, unknown>>[] = useMemo(() => [
    { key: 'planName', title: t('table.name'), dataIndex: 'planName', width: 180, ellipsis: true },
    {
      key: 'targetVersion',
      title: t('table.version'),
      dataIndex: 'targetVersion',
      width: 200,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    {
      key: 'deviceRange',
      title: t('table.total'),
      dataIndex: 'deviceRange',
      width: 110,
      render: (val) => `${String(val)} 台`,
    },
    {
      key: 'activationMethod',
      title: t('table.type'),
      dataIndex: 'activationMethod',
      width: 110,
      render: (val) => (
        <Tag color={val === 'immediate' ? 'cyan' : 'geekblue'}>
          {val === 'immediate' ? t('common.execute') : t('common.deploy')}
        </Tag>
      ),
    },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 100,
      render: (val) => {
        const s = val as ActivationStatus;
        return <Tag color={statusColorMap[s]}>{t(statusLabelKeyMap[s])}</Tag>;
      },
    },
    {
      key: 'scheduledTime',
      title: '执行时间',
      dataIndex: 'scheduledTime',
      width: 160,
      render: (val, record) => {
        const r = record as ActivationPlanRecord;
        const time = r.executedTime ?? r.scheduledTime;
        return time ? new Date(time).toLocaleString('zh-CN') : '—';
      },
    },
    { key: 'creator', title: t('table.operator'), dataIndex: 'creator', width: 100 },
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 140,
      fixed: 'right',
      render: (_, record) => {
        const plan = record as ActivationPlanRecord;
        return (
          <Space size="small">
            {(plan.status === 'pending' || plan.status === 'scheduled') && (
              <Button
                type="link"
                size="small"
                icon={<PlayCircleOutlined />}
                onClick={() => {
                  setPlans((prev) =>
                    prev.map((p) => (p.id === plan.id ? { ...p, status: 'activating' } : p)),
                  );
                  void message.success(t('common.execute'));
                }}
              >
                {t('common.execute')}
              </Button>
            )}
            <Popconfirm
              title={t('common.confirmDelete')}
              onConfirm={() => {
                setPlans((prev) => prev.filter((p) => p.id !== plan.id));
                void message.success(t('common.deleteSuccess'));
              }}
            >
              <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ], [t]);

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  return (
    <ListPageLayout
      title={t('nav.software.activation')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
          {t('common.add')}
        </Button>
      }
    >
      <FilterBar
        filterId="activation-plan-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="activation-plan-list"
        columns={columns}
        dataSource={paginated as (ActivationPlanRecord & Record<string, unknown>)[]}
        loading={false}
        rowKey="id"
        total={filtered.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => setPlans([...mockActivationPlans])}
        scroll={{ x: 1100 }}
      />

      <Modal
        title={t('common.add')}
        open={createVisible}
        onOk={handleCreate}
        onCancel={() => { setCreateVisible(false); form.resetFields(); }}
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="planName" label={t('table.name')} rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="targetVersion" label={t('table.version')} rules={[{ required: true }]}>
            <Select
              placeholder={t('common.pleaseSelect')}
              options={(versionsData?.list ?? []).map((v) => ({
                label: `${v.versionCode} (${v.deviceType})`,
                value: v.versionCode,
              }))}
            />
          </Form.Item>
          <Form.Item name="deviceRange" label={t('table.total')} rules={[{ required: true }]}>
            <Input type="number" placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="activationMethod" label={t('table.type')} initialValue="immediate">
            <Select
              options={[
                { label: t('common.execute'), value: 'immediate' },
                { label: t('common.deploy'), value: 'scheduled' },
              ]}
            />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, cur) => prev.activationMethod !== cur.activationMethod}
          >
            {({ getFieldValue }) =>
              getFieldValue('activationMethod') === 'scheduled' ? (
                <Form.Item name="scheduledTime" label={t('table.time')} rules={[{ required: true }]}>
                  <DatePicker showTime style={{ width: '100%' }} />
                </Form.Item>
              ) : null
            }
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
