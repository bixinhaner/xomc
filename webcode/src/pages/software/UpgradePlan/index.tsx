import { useState, useMemo } from 'react';
import {
  Button,
  Tag,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Progress,
  Popconfirm,
  message,
  DatePicker,
  InputNumber,
  Divider,
} from 'antd';
import { PlusOutlined, PlayCircleOutlined, PauseCircleOutlined, DeleteOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useUpgradePlans, useCreateUpgradePlan, useCancelUpgradePlan, useSoftwareVersions } from '@/hooks/api/useSoftware';
import type { UpgradePlan, UpgradePlanStatus } from '@/mock/data/software';
import { useT } from '@/hooks/useT';

const statusColorMap: Record<UpgradePlanStatus, string> = {
  pending: 'default',
  running: 'processing',
  success: 'green',
  failed: 'red',
  cancelled: 'warning',
  scheduled: 'blue',
};

const statusLabelKeyMap: Record<UpgradePlanStatus, string> = {
  pending: 'status.pending',
  running: 'status.running',
  success: 'status.success',
  failed: 'status.failed',
  cancelled: 'status.cancelled',
  scheduled: 'status.enabled',
};

export default function UpgradePlan() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createVisible, setCreateVisible] = useState(false);
  const [form] = Form.useForm();

  const filterFields: FilterField[] = useMemo(() => [
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.pending'), value: 'pending' },
        { label: t('status.running'), value: 'running' },
        { label: t('status.success'), value: 'success' },
        { label: t('status.failed'), value: 'failed' },
        { label: t('status.cancelled'), value: 'cancelled' },
        { label: t('status.enabled'), value: 'scheduled' },
      ],
    },
  ], [t]);

  const { data, isLoading, refetch } = useUpgradePlans({ ...filters, page, pageSize });
  const { data: versionsData } = useSoftwareVersions({ page: 1, pageSize: 100 });
  const createPlan = useCreateUpgradePlan();
  const cancelPlan = useCancelUpgradePlan();

  const handleCreate = () => {
    form.validateFields().then((vals) => {
      createPlan.mutate(
        {
          planName: vals.planName as string,
          targetVersionId: vals.targetVersionId as string,
          targetVersionCode: vals.targetVersionCode as string,
          deviceSns: (vals.deviceSns as string).split('\n').map((s: string) => s.trim()).filter(Boolean),
          scheduledTime: vals.scheduledTime ? String(vals.scheduledTime) : undefined,
          creator: 'admin',
          preCheckRequired: true,
          rollbackEnabled: true,
          totalCount: 0,
        },
        {
          onSuccess: () => {
            void message.success(t('common.save'));
            setCreateVisible(false);
            form.resetFields();
          },
        },
      );
    });
  };

  const columns: DataTableColumn<UpgradePlan & Record<string, unknown>>[] = useMemo(() => [
    { key: 'planName', title: t('table.name'), dataIndex: 'planName', width: 180, ellipsis: true },
    {
      key: 'targetVersionCode',
      title: '目标��本',
      dataIndex: 'targetVersionCode',
      width: 200,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    {
      key: 'totalCount',
      title: t('table.total'),
      dataIndex: 'totalCount',
      width: 130,
      render: (val) => `${String(val)} 台`,
    },
    {
      key: 'scheduledTime',
      title: t('table.time'),
      dataIndex: 'scheduledTime',
      width: 160,
      render: (val) => val ? new Date(String(val)).toLocaleString('zh-CN') : t('common.execute'),
    },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 100,
      render: (val) => {
        const s = val as UpgradePlanStatus;
        return <Tag color={statusColorMap[s]}>{t(statusLabelKeyMap[s])}</Tag>;
      },
    },
    {
      key: 'progress',
      title: t('table.result'),
      dataIndex: 'progress',
      width: 150,
      render: (val, record) => {
        const plan = record as UpgradePlan;
        return (
          <Space direction="vertical" size={0} style={{ width: '100%' }}>
            <Progress percent={Number(val)} size="small" />
            <span style={{ fontSize: 11, color: '#999' }}>
              {t('table.success')} {plan.successCount}/{plan.totalCount}, {t('table.failed')} {plan.failCount}
            </span>
          </Space>
        );
      },
    },
    { key: 'creator', title: t('table.operator'), dataIndex: 'creator', width: 100 },
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 150,
      fixed: 'right',
      render: (_, record) => {
        const plan = record as UpgradePlan;
        return (
          <Space size="small">
            {(plan.status === 'pending' || plan.status === 'scheduled') && (
              <Button type="link" size="small" icon={<PlayCircleOutlined />}>
                {t('common.execute')}
              </Button>
            )}
            {plan.status === 'running' && (
              <Button type="link" size="small" icon={<PauseCircleOutlined />}>
                {t('common.disable')}
              </Button>
            )}
            <Popconfirm
              title={t('common.confirmDelete')}
              onConfirm={() => cancelPlan.mutate(plan.id, { onSuccess: () => void message.success(t('common.deleteSuccess')) })}
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

  return (
    <ListPageLayout
      title={t('nav.software.upgradePlan')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
          {t('common.add')}
        </Button>
}
    >
      <FilterBar
        filterId="upgrade-plan-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="upgrade-plan-list"
        columns={columns}
        dataSource={(data?.list ?? []) as (UpgradePlan & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1200 }}
      />

      <Modal
        title={t('common.add')}
        open={createVisible}
        onOk={handleCreate}
        onCancel={() => { setCreateVisible(false); form.resetFields(); }}
        confirmLoading={createPlan.isPending}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="planName" label={t('table.name')} rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="targetVersionId" label={t('table.version')} rules={[{ required: true }]}>
            <Select
              placeholder={t('common.pleaseSelect')}
              options={(versionsData?.list ?? []).map((v) => ({
                label: `${v.versionCode} (${v.deviceType})`,
                value: v.id,
              }))}
              onChange={(val) => {
                const ver = (versionsData?.list ?? []).find((v) => v.id === val);
                if (ver) form.setFieldValue('targetVersionCode', ver.versionCode);
              }}
            />
          </Form.Item>
          <Form.Item name="targetVersionCode" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="deviceSns" label={t('device.sn')} rules={[{ required: true }]}>
            <Input.TextArea rows={5} placeholder={t('common.placeholder')} />
          </Form.Item>
          <Divider orientation="left" plain>{t('table.time')}</Divider>
          <Form.Item name="scheduleType" label={t('table.type')} initialValue="immediate">
            <Select
              options={[
                { label: t('common.execute'), value: 'immediate' },
                { label: t('common.deploy'), value: 'scheduled' },
              ]}
            />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, cur) => prev.scheduleType !== cur.scheduleType}
          >
            {({ getFieldValue }) =>
              getFieldValue('scheduleType') === 'scheduled' ? (
                <Form.Item name="scheduledTime" label="执行时间" rules={[{ required: true }]}>
                  <DatePicker showTime style={{ width: '100%' }} />
                </Form.Item>
              ) : null
            }
          </Form.Item>
          <Form.Item name="maxParallel" label={t('table.total')} initialValue={10}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
