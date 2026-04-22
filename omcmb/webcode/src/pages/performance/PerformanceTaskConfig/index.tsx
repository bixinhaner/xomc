import { useState, useMemo } from 'react';
import { Button, Card, Form, Input, Modal, Select, Space, Switch, Tag, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { usePerformanceTasks, useCreatePerformanceTask } from '@core/hooks/api/usePerformance';
import { useT } from '@/hooks/useT';

interface PerfTaskRow extends Record<string, unknown> {
  id: string;
  taskName: string;
  collectType: string;
  granularity: string;
  deviceRange: string;
  enabled: boolean;
  nextRunTime: string;
  status: 'active' | 'paused' | 'error';
}

const GRANULARITY_OPTIONS = [
  { label: '15min', value: '15min' },
  { label: '30min', value: '30min' },
  { label: '1h', value: '1h' },
  { label: '1d', value: '1d' },
];

const COLLECT_TYPE_OPTIONS = [
  { label: 'scheduled', value: 'scheduled' },
  { label: 'immediate', value: 'immediate' },
  { label: 'triggered', value: 'triggered' },
];

const mockData: PerfTaskRow[] = [
  { id: '1', taskName: '全网15分钟定时采集', collectType: '定时采集', granularity: '15min', deviceRange: '全部设备', enabled: true, nextRunTime: '2026-03-02 10:30:00', status: 'active' },
  { id: '2', taskName: '核心设备小时采集', collectType: '定时采集', granularity: '1h', deviceRange: '核心基站组', enabled: true, nextRunTime: '2026-03-02 11:00:00', status: 'active' },
  { id: '3', taskName: '5G基站日报采集', collectType: '定时采集', granularity: '1d', deviceRange: '5G基站组', enabled: true, nextRunTime: '2026-03-03 00:05:00', status: 'active' },
  { id: '4', taskName: '北京节点专项采集', collectType: '触发采集', granularity: '15min', deviceRange: '北京区域', enabled: false, nextRunTime: '-', status: 'paused' },
  { id: '5', taskName: '上海区域异常告警采集', collectType: '触发采集', granularity: '15min', deviceRange: '上海区域', enabled: true, nextRunTime: '告警触发时', status: 'error' },
];

export default function PerformanceTaskConfig() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRow, setEditingRow] = useState<PerfTaskRow | null>(null);
  const [form] = Form.useForm();

  const { data, isLoading, refetch } = usePerformanceTasks({ page, pageSize });
  const createTask = useCreatePerformanceTask();

  const tableSource = (data?.items ?? mockData) as unknown as PerfTaskRow[];

  const STATUS_MAP: Record<string, { color: string; text: string }> = useMemo(() => ({
    active: { color: 'success', text: t('status.running') },
    paused: { color: 'default', text: t('status.disabled') },
    error: { color: 'error', text: t('status.failed') },
  }), [t]);

  const openEdit = (record: PerfTaskRow) => {
    setEditingRow(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleSave = () => {
    form.validateFields().then((vals) => {
      if (editingRow) {
        void message.success(t('common.save'));
      } else {
        createTask.mutate({
          taskName: vals.taskName as string,
          taskType: 'extraction',
          deviceSns: [],
          kpiCodes: [],
          granularity: vals.granularity as '15min' | '30min' | '1h' | '1d',
          timeRange: ['', ''],
          creator: 'admin',
        }, {
          onSuccess: () => void message.success(t('common.save')),
        });
      }
      setModalVisible(false);
      form.resetFields();
    }).catch(() => undefined);
  };

  const columns: DataTableColumn<PerfTaskRow>[] = useMemo(() => [
    { key: 'taskName', title: t('table.name'), dataIndex: 'taskName', width: 220, ellipsis: true },
    { key: 'collectType', title: t('table.type'), dataIndex: 'collectType', width: 100 },
    {
      key: 'granularity',
      title: t('perf.granularity'),
      dataIndex: 'granularity',
      width: 90,
      render: (val) => <Tag color="blue">{val as string}</Tag>,
    },
    { key: 'deviceRange', title: t('table.region'), dataIndex: 'deviceRange', width: 150 },
    {
      key: 'enabled',
      title: t('table.status'),
      dataIndex: 'enabled',
      width: 90,
      render: (val, record) => (
        <Switch
          size="small"
          checked={val as boolean}
          checkedChildren={t('common.enable')}
          unCheckedChildren={t('common.disable')}
          onChange={() => void message.info(`${record.taskName as string} ${val ? t('common.disable') : t('common.enable')}`)}
        />
      ),
    },
    { key: 'nextRunTime', title: t('table.time'), dataIndex: 'nextRunTime', width: 160 },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 100,
      render: (val) => {
        const cfg = STATUS_MAP[val as string] ?? STATUS_MAP.paused;
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => void message.info(`${t('common.execute')}: ${record.taskName as string}`)}>{t('common.execute')}</Button>
          <Button type="link" size="small" onClick={() => openEdit(record)}>{t('common.edit')}</Button>
        </Space>
      ),
    },
  ], [t, STATUS_MAP]);

  return (
    <ListPageLayout
      title={t('nav.performance.taskConfig')}
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => { setEditingRow(null); form.resetFields(); setModalVisible(true); }}
        >
          {t('common.add')}
        </Button>
      }
    >
      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<PerfTaskRow>
          tableId="performance-task-config"
          columns={columns}
          dataSource={tableSource}
          loading={isLoading}
          rowKey="id"
          total={data?.total ?? tableSource.length}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          onRefresh={() => void refetch()}
          scroll={{ x: 1100 }}
        />
      </Card>

      <Modal
        title={editingRow ? t('common.edit') : t('common.add')}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        okText={t('common.save')}
        width={560}
        confirmLoading={createTask.isPending}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('table.name')} name="taskName" rules={[{ required: true, message: t('common.placeholder') }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.type')} name="collectType" rules={[{ required: true }]}>
            <Select options={COLLECT_TYPE_OPTIONS} placeholder={t('common.pleaseSelect')} />
          </Form.Item>
          <Form.Item label={t('perf.granularity')} name="granularity" rules={[{ required: true }]}>
            <Select options={GRANULARITY_OPTIONS} placeholder={t('common.pleaseSelect')} />
          </Form.Item>
          <Form.Item label={t('table.region')} name="deviceRange" rules={[{ required: true }]}>
            <Select
              placeholder={t('common.pleaseSelect')}
              options={[
                { label: '全部设备', value: '全部设备' },
                { label: '核心基站组', value: '核心基站组' },
                { label: '5G基站组', value: '5G基站组' },
                { label: '北京区域', value: '北京区域' },
                { label: '上海区域', value: '上海区域' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
