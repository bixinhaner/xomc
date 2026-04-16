import { useState, useMemo } from 'react';
import { Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Tag, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useThresholds, useCreateThreshold, useUpdateThreshold, useDeleteThresholds } from '@/hooks/api/usePerformance';
import type { PerformanceThreshold } from '@/types/performance';
import { useT } from '@/hooks/useT';

interface ThresholdRow extends Record<string, unknown> {
  id: string;
  thresholdName: string;
  kpiName: string;
  kpiCode: string;
  warningValue: number;
  criticalValue: number;
  unit: string;
  alarmLevel: 'warning' | 'minor' | 'major' | 'critical';
  enabled: boolean;
  updateTime: string;
}

const ALARM_LEVEL_COLORS: Record<string, string> = {
  warning: 'gold',
  minor: 'orange',
  major: 'red',
  critical: 'purple',
};

const KPI_OPTION_KEYS: { labelKey: string; value: string; unit: string }[] = [
  { labelKey: 'kpi.rrcSetupSuccessRate', value: 'RRC_SR', unit: '%' },
  { labelKey: 'kpi.erabSetupSuccessRate', value: 'ERAB_SR', unit: '%' },
  { labelKey: 'kpi.handoverSuccessRate', value: 'HO_SR', unit: '%' },
  { labelKey: 'kpi.dlThroughput', value: 'DL_THROUGHPUT', unit: 'Mbps' },
  { labelKey: 'kpi.ulThroughput', value: 'UL_THROUGHPUT', unit: 'Mbps' },
  { labelKey: 'kpi.availability', value: 'AVAILABILITY', unit: '%' },
  { labelKey: 'kpi.pdcpLossRate', value: 'PDCP_LOSS', unit: '%' },
];

const MOCK_DATA_RAW = [
  { id: '1', thresholdName: 'RRC成功率低告警', kpiNameKey: 'kpi.rrcSetupSuccessRate', kpiCode: 'RRC_SR', warningValue: 97, criticalValue: 95, unit: '%', alarmLevel: 'major' as const, enabled: true, updateTime: '2026-01-15 10:00:00' },
  { id: '2', thresholdName: 'ERAB成功率低告警', kpiNameKey: 'kpi.erabSetupSuccessRate', kpiCode: 'ERAB_SR', warningValue: 97, criticalValue: 95, unit: '%', alarmLevel: 'major' as const, enabled: true, updateTime: '2026-01-15 10:00:00' },
  { id: '3', thresholdName: '切换成功率低告警', kpiNameKey: 'kpi.handoverSuccessRate', kpiCode: 'HO_SR', warningValue: 96, criticalValue: 93, unit: '%', alarmLevel: 'warning' as const, enabled: true, updateTime: '2026-02-01 09:00:00' },
  { id: '4', thresholdName: '可用率低告警', kpiNameKey: 'kpi.availability', kpiCode: 'AVAILABILITY', warningValue: 99.95, criticalValue: 99.9, unit: '%', alarmLevel: 'critical' as const, enabled: false, updateTime: '2026-02-10 14:00:00' },
  { id: '5', thresholdName: 'PDCP丢包率高告警', kpiNameKey: 'kpi.pdcpLossRate', kpiCode: 'PDCP_LOSS', warningValue: 0.1, criticalValue: 0.5, unit: '%', alarmLevel: 'major' as const, enabled: true, updateTime: '2026-02-15 11:00:00' },
];

export default function ThresholdConfig() {
  const t = useT();
  const KPI_OPTIONS = useMemo(() =>
    KPI_OPTION_KEYS.map((k) => ({ label: t(k.labelKey), value: k.value, unit: k.unit })),
  [t]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRow, setEditingRow] = useState<ThresholdRow | null>(null);
  const [form] = Form.useForm();

  const { data, isLoading, refetch } = useThresholds({ page, pageSize });
  const createThreshold = useCreateThreshold();
  const updateThreshold = useUpdateThreshold();
  const deleteThresholds = useDeleteThresholds();

  const mockData: ThresholdRow[] = useMemo(() =>
    MOCK_DATA_RAW.map((r) => ({ ...r, kpiName: t(r.kpiNameKey) })),
  [t]);
  const tableSource = (data?.items ?? mockData) as unknown as ThresholdRow[];

  const ALARM_LEVEL_OPTIONS = useMemo(() => [
    { label: t('alarm.severity.warning'), value: 'warning' },
    { label: t('alarm.severity.minor'), value: 'minor' },
    { label: t('alarm.severity.major'), value: 'major' },
    { label: t('alarm.severity.critical'), value: 'critical' },
  ], [t]);

  const openEdit = (record: ThresholdRow) => {
    setEditingRow(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleSave = () => {
    form.validateFields().then((vals: Partial<PerformanceThreshold>) => {
      if (editingRow) {
        updateThreshold.mutate({ id: editingRow.id, data: vals }, {
          onSuccess: () => { void message.success(t('common.save')); setModalVisible(false); },
        });
      } else {
        createThreshold.mutate(vals as Omit<PerformanceThreshold, 'id' | 'createTime' | 'updateTime'>, {
          onSuccess: () => { void message.success(t('common.save')); setModalVisible(false); },
        });
      }
    }).catch(() => undefined);
  };

  const handleToggleEnabled = (record: ThresholdRow) => {
    updateThreshold.mutate(
      { id: record.id, data: { enabled: !record.enabled } },
      { onSuccess: () => void message.success(`${record.enabled ? t('common.disable') : t('common.enable')}`) },
    );
  };

  const columns: DataTableColumn<ThresholdRow>[] = useMemo(() => [
    { key: 'thresholdName', title: t('perf.kpiName'), dataIndex: 'thresholdName', width: 200 },
    { key: 'kpiCode', title: t('perf.kpiCode'), dataIndex: 'kpiCode', width: 140, mono: true },
    {
      key: 'warningValue',
      title: t('alarm.severity.warning'),
      dataIndex: 'warningValue',
      width: 120,
      render: (val, record) => `${val as number} ${record.unit as string}`,
    },
    {
      key: 'criticalValue',
      title: t('alarm.severity.critical'),
      dataIndex: 'criticalValue',
      width: 120,
      render: (val, record) => `${val as number} ${record.unit as string}`,
    },
    {
      key: 'alarmLevel',
      title: t('alarm.severity'),
      dataIndex: 'alarmLevel',
      width: 100,
      render: (val) => {
        const opt = ALARM_LEVEL_OPTIONS.find((o) => o.value === val);
        return <Tag color={ALARM_LEVEL_COLORS[val as string]}>{opt?.label ?? (val as string)}</Tag>;
      },
    },
    {
      key: 'enabled',
      title: t('table.status'),
      dataIndex: 'enabled',
      width: 90,
      render: (val, record) => (
        <Switch
          size="small"
          checked={val as boolean}
          onChange={() => handleToggleEnabled(record)}
          checkedChildren={t('common.enable')}
          unCheckedChildren={t('common.disable')}
        />
      ),
    },
    { key: 'updateTime', title: t('table.updateTime'), dataIndex: 'updateTime', width: 160 },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => openEdit(record)}>{t('common.edit')}</Button>
          <Popconfirm title={t('common.confirmDelete')} onConfirm={() => deleteThresholds.mutate([record.id])}>
            <Button type="link" size="small" danger>{t('common.delete')}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ], [t, ALARM_LEVEL_OPTIONS]);

  return (
    <ListPageLayout
      title={t('nav.performance.threshold')}
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
      <DataTable<ThresholdRow>
        tableId="threshold-config"
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

      <Modal
        title={editingRow ? t('common.edit') : t('common.add')}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        okText={t('common.save')}
        width={560}
        confirmLoading={createThreshold.isPending || updateThreshold.isPending}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('table.name')} name="thresholdName" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('perf.kpiName')} name="kpiCode" rules={[{ required: true }]}>
            <Select
              options={KPI_OPTIONS.map((k) => ({ label: `${k.label} (${k.unit})`, value: k.value }))}
              placeholder={t('common.pleaseSelect')}
            />
          </Form.Item>
          <Form.Item label={t('alarm.severity.warning')} name="warningValue" rules={[{ required: true }]}>
            <InputNumber style={{ width: '100%' }} placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('alarm.severity.critical')} name="criticalValue" rules={[{ required: true }]}>
            <InputNumber style={{ width: '100%' }} placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('alarm.severity')} name="alarmLevel" rules={[{ required: true }]}>
            <Select options={ALARM_LEVEL_OPTIONS} placeholder={t('common.pleaseSelect')} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
