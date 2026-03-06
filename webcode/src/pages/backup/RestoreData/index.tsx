import { useState, useMemo } from 'react';
import { Button, Form, Input, Modal, Select, Space, Steps, Tag, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';

interface RestoreRow extends Record<string, unknown> {
  id: string;
  restoreId: string;
  backupFile: string;
  restoreType: 'full' | 'config-only' | 'incremental';
  deviceRange: string;
  status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled';
  startTime: string;
  operator: string;
  progress: number;
}

const STATUS_MAP_KEYS: Record<string, { color: string; key: string }> = {
  pending: { color: 'default', key: 'status.pending' },
  running: { color: 'processing', key: 'status.running' },
  success: { color: 'success', key: 'status.success' },
  failed: { color: 'error', key: 'status.failed' },
  cancelled: { color: 'warning', key: 'status.cancelled' },
};

const RESTORE_TYPE_MAP: Record<string, string> = {
  full: '全量恢复',
  'config-only': '配置恢复',
  incremental: '增量恢复',
};

const mockData: RestoreRow[] = [
  { id: '1', restoreId: 'RST-20260301-001', backupFile: 'full_backup_20260228.tar.gz', restoreType: 'full', deviceRange: 'ENB00001', status: 'success', startTime: '2026-03-01 14:30:00', operator: 'admin', progress: 100 },
  { id: '2', restoreId: 'RST-20260228-002', backupFile: 'config_backup_20260227.tar.gz', restoreType: 'config-only', deviceRange: 'GNB00001', status: 'success', startTime: '2026-02-28 10:00:00', operator: 'operator1', progress: 100 },
  { id: '3', restoreId: 'RST-20260225-003', backupFile: 'full_backup_20260224.tar.gz', restoreType: 'full', deviceRange: 'ENB00002, ENB00003', status: 'failed', startTime: '2026-02-25 09:00:00', operator: 'admin', progress: 45 },
  { id: '4', restoreId: 'RST-20260220-004', backupFile: 'incremental_20260219.tar.gz', restoreType: 'incremental', deviceRange: 'ENB00001', status: 'success', startTime: '2026-02-20 02:00:00', operator: 'system', progress: 100 },
];

// Restore wizard state
interface RestoreWizardState {
  step: number;
  backupFile: string;
  restoreType: string;
  deviceRange: string[];
  confirmText: string;
}

export default function RestoreData() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [wizardState, setWizardState] = useState<RestoreWizardState>({
    step: 0,
    backupFile: '',
    restoreType: 'full',
    deviceRange: [],
    confirmText: '',
  });
  const [form] = Form.useForm();

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'restoreId', label: t('table.index'), type: 'input' },
    { name: 'operator', label: t('table.operator'), type: 'input' },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.pending'), value: 'pending' },
        { label: t('status.running'), value: 'running' },
        { label: t('status.success'), value: 'success' },
        { label: t('status.failed'), value: 'failed' },
      ],
    },
  ], [t]);

  const filteredSource = mockData.filter((row) => {
    if (filters.restoreId && !row.restoreId.includes(filters.restoreId as string)) return false;
    if (filters.operator && !row.operator.includes(filters.operator as string)) return false;
    if (filters.status && row.status !== filters.status) return false;
    return true;
  });

  const handleRestoreNext = () => {
    if (wizardState.step === 0) {
      form.validateFields(['backupFile', 'restoreType']).then((vals: Record<string, unknown>) => {
        setWizardState((prev) => ({
          ...prev,
          step: 1,
          backupFile: vals.backupFile as string,
          restoreType: vals.restoreType as string,
        }));
      }).catch(() => undefined);
    } else if (wizardState.step === 1) {
      form.validateFields(['deviceRange']).then((vals: Record<string, unknown>) => {
        setWizardState((prev) => ({
          ...prev,
          step: 2,
          deviceRange: vals.deviceRange as string[],
        }));
      }).catch(() => undefined);
    } else if (wizardState.step === 2) {
      if (wizardState.confirmText !== 'RESTORE') {
        void message.error(t('common.confirm'));
        return;
      }
      void message.success(t('common.submit'));
      setModalVisible(false);
      setWizardState({ step: 0, backupFile: '', restoreType: 'full', deviceRange: [], confirmText: '' });
      form.resetFields();
    }
  };

  const renderWizardContent = () => {
    switch (wizardState.step) {
      case 0:
        return (
          <Form form={form} layout="vertical">
            <Form.Item label={t('common.pleaseSelect')} name="backupFile" rules={[{ required: true }]}>
              <Select
                placeholder={t('common.pleaseSelect')}
                options={[
                  { label: 'full_backup_20260302.tar.gz (256 MB)', value: 'full_backup_20260302' },
                  { label: 'full_backup_20260301.tar.gz (248 MB)', value: 'full_backup_20260301' },
                  { label: 'config_backup_20260302.tar.gz (12 MB)', value: 'config_backup_20260302' },
                  { label: 'incremental_20260301.tar.gz (32 MB)', value: 'incremental_20260301' },
                ]}
              />
            </Form.Item>
            <Form.Item label={t('table.type')} name="restoreType" rules={[{ required: true }]} initialValue="full">
              <Select
                options={[
                  { label: RESTORE_TYPE_MAP.full, value: 'full' },
                  { label: RESTORE_TYPE_MAP['config-only'], value: 'config-only' },
                  { label: RESTORE_TYPE_MAP.incremental, value: 'incremental' },
                ]}
              />
            </Form.Item>
          </Form>
        );
      case 1:
        return (
          <Form form={form} layout="vertical">
            <Form.Item label={t('table.description')} name="deviceRange" rules={[{ required: true }]}>
              <Select
                mode="multiple"
                placeholder={t('common.pleaseSelect')}
                options={[
                  { label: 'ENB00001 - 北京朝阳基站01', value: 'ENB00001' },
                  { label: 'ENB00002 - 北京海淀基站01', value: 'ENB00002' },
                  { label: 'ENB00003 - 上海浦东基站01', value: 'ENB00003' },
                  { label: 'GNB00001 - 北京5G基站01', value: 'GNB00001' },
                ]}
              />
            </Form.Item>
          </Form>
        );
      case 2:
        return (
          <div>
            <div style={{ background: '#fff7e6', border: '1px solid #ffd591', borderRadius: 6, padding: 12, marginBottom: 16 }}>
              <div style={{ fontWeight: 600, color: '#d46b08', marginBottom: 8 }}>警告：此操作将覆盖现有配置</div>
              <div style={{ fontSize: 12, color: '#595959' }}>
                <div>备份文件: {wizardState.backupFile}</div>
                <div>恢复类型: {RESTORE_TYPE_MAP[wizardState.restoreType]}</div>
                <div>目标设备: {wizardState.deviceRange.join(', ')}</div>
              </div>
            </div>
            <Form layout="vertical">
              <Form.Item
                label={<span>请输入 <Tag color="red">RESTORE</Tag> 以确认恢复操作</span>}
                required
              >
                <Input
                  placeholder="输入 RESTORE"
                  value={wizardState.confirmText}
                  onChange={(e) => setWizardState((prev) => ({ ...prev, confirmText: e.target.value }))}
                />
              </Form.Item>
            </Form>
          </div>
        );
      default:
        return null;
    }
  };

  const columns: DataTableColumn<RestoreRow>[] = useMemo(() => [
    { key: 'restoreId', title: t('table.index'), dataIndex: 'restoreId', width: 180, mono: true, copyable: true },
    { key: 'backupFile', title: t('table.name'), dataIndex: 'backupFile', width: 240, ellipsis: true },
    {
      key: 'restoreType',
      title: t('table.type'),
      dataIndex: 'restoreType',
      width: 110,
      render: (val) => <Tag color="blue">{RESTORE_TYPE_MAP[val as string] ?? (val as string)}</Tag>,
    },
    { key: 'deviceRange', title: t('table.description'), dataIndex: 'deviceRange', width: 200, ellipsis: true },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 100,
      render: (val) => {
        const cfg = STATUS_MAP_KEYS[val as string] ?? STATUS_MAP_KEYS.pending;
        return <Tag color={cfg.color}>{t(cfg.key)}</Tag>;
      },
    },
    { key: 'startTime', title: t('table.createTime'), dataIndex: 'startTime', width: 160 },
    { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 100 },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.backup.restore')}
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setWizardState({ step: 0, backupFile: '', restoreType: 'full', deviceRange: [], confirmText: '' });
            form.resetFields();
            setModalVisible(true);
          }}
          danger
        >
          {t('common.add')}
        </Button>
      }
    >
      <FilterBar
        filterId="restore-data"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />
      <DataTable<RestoreRow>
        tableId="restore-data"
        columns={columns}
        dataSource={filteredSource}
        loading={false}
        rowKey="id"
        total={filteredSource.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        scroll={{ x: 1200 }}
      />

      <Modal
        title={t('common.add')}
        open={modalVisible}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        footer={
          <Space>
            {wizardState.step > 0 && <Button onClick={() => setWizardState((prev) => ({ ...prev, step: prev.step - 1 }))}>{t('common.prev')}</Button>}
            <Button onClick={() => { setModalVisible(false); form.resetFields(); }}>{t('common.cancel')}</Button>
            <Button
              type="primary"
              danger={wizardState.step === 2}
              onClick={handleRestoreNext}
            >
              {wizardState.step < 2 ? t('common.next') : t('common.confirm')}
            </Button>
          </Space>
        }
        width={560}
      >
        <Steps
          current={wizardState.step}
          size="small"
          style={{ marginBottom: 24 }}
          items={[
            { title: t('common.pleaseSelect') },
            { title: t('common.pleaseSelect') },
            { title: t('common.confirm') },
          ]}
        />
        {renderWizardContent()}
      </Modal>
    </ListPageLayout>
  );
}
