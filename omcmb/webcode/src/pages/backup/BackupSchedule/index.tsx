import { useState, useMemo } from 'react';
import { Button, Form, Input, Modal, Popconfirm, Select, Space, Switch, Tag, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import SchedulePicker from '@/components/SchedulePicker';
import type { ScheduleValue } from '@/components/SchedulePicker';
import { useBackupSchedules, useCreateBackupSchedule, useUpdateBackupSchedule, useDeleteBackupSchedules } from '@/hooks/api/useBackup';
import type { BackupSchedule } from '@/mock/data/backup';
import { useT } from '@/hooks/useT';

interface ScheduleRow extends Record<string, unknown> {
  id: string;
  scheduleName: string;
  backupType: 'full' | 'incremental' | 'config-only';
  scheduleType: string;
  cronDescription: string;
  nextRunTime: string;
  lastRunTime: string;
  enabled: boolean;
  retentionDays: number;
  creator: string;
}

const BACKUP_TYPE_OPTIONS = [
  { label: '全量备份', value: 'full' },
  { label: '增量备份', value: 'incremental' },
  { label: '配置备份', value: 'config-only' },
];

const BACKUP_TYPE_COLORS: Record<string, string> = {
  full: 'blue',
  incremental: 'purple',
  'config-only': 'green',
};

const mockData: ScheduleRow[] = [
  { id: '1', scheduleName: '全网每日全量备份', backupType: 'full', scheduleType: '每天', cronDescription: '每天 02:00', nextRunTime: '2026-03-03 02:00:00', lastRunTime: '2026-03-02 02:45:32', enabled: true, retentionDays: 30, creator: 'admin' },
  { id: '2', scheduleName: '5G基站每周增量', backupType: 'incremental', scheduleType: '每周', cronDescription: '每周一 04:00', nextRunTime: '2026-03-09 04:00:00', lastRunTime: '2026-03-02 04:12:05', enabled: true, retentionDays: 90, creator: 'admin' },
  { id: '3', scheduleName: '核心设备配置备份', backupType: 'config-only', scheduleType: '每天', cronDescription: '每天 06:00', nextRunTime: '2026-03-03 06:00:00', lastRunTime: '2026-03-02 06:08:21', enabled: true, retentionDays: 60, creator: 'operator1' },
  { id: '4', scheduleName: '月度全量归档', backupType: 'full', scheduleType: '每月', cronDescription: '每月1日 00:30', nextRunTime: '2026-04-01 00:30:00', lastRunTime: '2026-03-01 01:22:15', enabled: false, retentionDays: 365, creator: 'admin' },
];

export default function BackupSchedule() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRow, setEditingRow] = useState<ScheduleRow | null>(null);
  const [scheduleValue, setScheduleValue] = useState<ScheduleValue | null>(null);
  const [form] = Form.useForm();

  const { data, isLoading, refetch } = useBackupSchedules({ page, pageSize });
  const createSchedule = useCreateBackupSchedule();
  const updateSchedule = useUpdateBackupSchedule();
  const deleteSchedules = useDeleteBackupSchedules();

  const tableSource = (data?.items ?? mockData) as unknown as ScheduleRow[];

  const openEdit = (record: ScheduleRow) => {
    setEditingRow(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleSave = () => {
    form.validateFields().then((vals: Partial<BackupSchedule>) => {
      const payload = { ...vals, cronExpression: '0 2 * * *', cronDescription: scheduleValue ? `${scheduleValue.mode}执行` : '每天执行' };
      if (editingRow) {
        updateSchedule.mutate({ id: editingRow.id, data: payload }, {
          onSuccess: () => { void message.success(t('common.save')); setModalVisible(false); },
        });
      } else {
        createSchedule.mutate(payload as Omit<BackupSchedule, 'id' | 'createTime'>, {
          onSuccess: () => { void message.success(t('common.save')); setModalVisible(false); },
        });
      }
      form.resetFields();
    }).catch(() => undefined);
  };

  const handleToggle = (record: ScheduleRow) => {
    updateSchedule.mutate(
      { id: record.id, data: { enabled: !record.enabled } },
      { onSuccess: () => void message.success(record.enabled ? t('common.disable') : t('common.enable')) },
    );
  };

  const columns: DataTableColumn<ScheduleRow>[] = useMemo(() => [
    { key: 'scheduleName', title: t('table.name'), dataIndex: 'scheduleName', width: 220, ellipsis: true },
    {
      key: 'backupType',
      title: t('table.type'),
      dataIndex: 'backupType',
      width: 110,
      render: (val) => {
        const opt = BACKUP_TYPE_OPTIONS.find((o) => o.value === val);
        return <Tag color={BACKUP_TYPE_COLORS[val as string]}>{opt?.label ?? (val as string)}</Tag>;
      },
    },
    { key: 'scheduleType', title: t('table.type'), dataIndex: 'scheduleType', width: 90 },
    { key: 'cronDescription', title: t('table.description'), dataIndex: 'cronDescription', width: 140 },
    { key: 'nextRunTime', title: t('table.time'), dataIndex: 'nextRunTime', width: 160 },
    { key: 'lastRunTime', title: t('table.time'), dataIndex: 'lastRunTime', width: 160 },
    { key: 'retentionDays', title: t('table.description'), dataIndex: 'retentionDays', width: 90 },
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
          onChange={() => handleToggle(record)}
        />
      ),
    },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>{t('common.edit')}</Button>
          <Popconfirm title={t('common.confirmDelete')} onConfirm={() => deleteSchedules.mutate([record.id])}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>{t('common.delete')}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.backup.schedule')}
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => { setEditingRow(null); form.resetFields(); setScheduleValue(null); setModalVisible(true); }}
        >
          {t('common.add')}
        </Button>
      }
    >
      <DataTable<ScheduleRow>
        tableId="backup-schedule"
        columns={columns}
        dataSource={tableSource}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? tableSource.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1400 }}
      />

      <Modal
        title={editingRow ? t('common.edit') : t('common.add')}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        okText={t('common.save')}
        width={600}
        confirmLoading={createSchedule.isPending || updateSchedule.isPending}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('table.name')} name="scheduleName" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.type')} name="backupType" rules={[{ required: true }]}>
            <Select options={BACKUP_TYPE_OPTIONS} placeholder={t('common.pleaseSelect')} />
          </Form.Item>
          <Form.Item label={t('table.description')} name="deviceGroups" rules={[{ required: true }]}>
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              options={[
                { label: '全部设备', value: 'all' },
                { label: '北京设备组', value: 'bj' },
                { label: '上海设备组', value: 'sh' },
                { label: '5G基站组', value: '5g' },
                { label: '核心设备组', value: 'core' },
              ]}
            />
          </Form.Item>
          <Form.Item label={t('table.time')} name="retentionDays" rules={[{ required: true }]}>
            <Select
              options={[
                { label: '7天', value: 7 },
                { label: '30天', value: 30 },
                { label: '60天', value: 60 },
                { label: '90天', value: 90 },
                { label: '180天', value: 180 },
                { label: '365天', value: 365 },
              ]}
              placeholder={t('common.pleaseSelect')}
            />
          </Form.Item>
          <Form.Item label="执行计划">
            <SchedulePicker value={scheduleValue ?? undefined} onChange={(val) => setScheduleValue(val)} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
