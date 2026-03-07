import { useState, useMemo } from 'react';
import { Button, Input, Popconfirm, Space, Tag, Typography, message } from 'antd';
import { EditOutlined, RollbackOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useConfigParams, useUpdateConfigParam } from '@/hooks/api/useConfig';
import { useT } from '@/hooks/useT';

interface LiveParamRow extends Record<string, unknown> {
  id: string;
  deviceSn: string;
  paramPath: string;
  paramName: string;
  currentValue: string;
  defaultValue: string;
  status: 'normal' | 'modified' | 'error';
}

const mockData: LiveParamRow[] = [
  { id: '1', deviceSn: 'ENB00001', paramPath: '/cell/0/rf/txPower', paramName: '发射功率', currentValue: '46', defaultValue: '43', status: 'modified' },
  { id: '2', deviceSn: 'ENB00001', paramPath: '/cell/0/rf/rsPower', paramName: '参考信号功率', currentValue: '-3', defaultValue: '0', status: 'modified' },
  { id: '3', deviceSn: 'ENB00002', paramPath: '/cell/0/rrm/prachCfgIdx', paramName: 'PRACH配置索引', currentValue: '14', defaultValue: '14', status: 'normal' },
  { id: '4', deviceSn: 'GNB00001', paramPath: '/gnb/cell/0/nr/ssbSubCarrierSpacing', paramName: 'SSB子载波间隔', currentValue: '30', defaultValue: '30', status: 'normal' },
  { id: '5', deviceSn: 'GNB00001', paramPath: '/gnb/cell/0/nr/bandWidth', paramName: '带宽', currentValue: '100', defaultValue: '100', status: 'error' },
];

export default function LiveParamConfig() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editValue, setEditValue] = useState<string>('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data, isLoading, refetch } = useConfigParams({
    keyword: filters.paramName as string,
    deviceSn: filters.deviceSn as string,
    page,
    pageSize,
  });

  const updateParam = useUpdateConfigParam();

  const STATUS_MAP: Record<string, { color: string; text: string }> = useMemo(() => ({
    normal: { color: 'success', text: t('status.online') },
    modified: { color: 'warning', text: t('status.pending') },
    error: { color: 'error', text: t('status.failed') },
  }), [t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'deviceSn', label: t('device.sn'), type: 'input', placeholder: t('common.placeholder') },
    { name: 'paramPath', label: t('config.paramCode'), type: 'input', placeholder: t('common.placeholder') },
    { name: 'paramName', label: t('config.paramName'), type: 'input', placeholder: t('common.placeholder') },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.online'), value: 'normal' },
        { label: t('status.pending'), value: 'modified' },
        { label: t('status.failed'), value: 'error' },
      ],
    },
  ], [t]);

  const handleEdit = (record: LiveParamRow) => {
    setEditingId(record.id);
    setEditValue(record.currentValue);
  };

  const handleSave = (record: LiveParamRow) => {
    updateParam.mutate(
      { id: record.id, value: editValue },
      {
        onSuccess: () => {
          void message.success(t('common.save'));
          setEditingId(null);
        },
        onError: () => {
          void message.error(t('status.failed'));
        },
      },
    );
  };

  const handleRestore = (record: LiveParamRow) => {
    updateParam.mutate(
      { id: record.id, value: record.defaultValue },
      {
        onSuccess: () => {
          void message.success(t('common.reset'));
        },
      },
    );
  };

  const tableSource = (data?.items ?? mockData) as unknown as LiveParamRow[];

  const columns: DataTableColumn<LiveParamRow>[] = useMemo(() => [
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 140, mono: true, copyable: true },
    { key: 'paramPath', title: t('config.paramCode'), dataIndex: 'paramPath', width: 260, mono: true, ellipsis: true },
    { key: 'paramName', title: t('config.paramName'), dataIndex: 'paramName', width: 160 },
    {
      key: 'currentValue',
      title: t('config.paramValue'),
      dataIndex: 'currentValue',
      width: 160,
      render: (val, record) => {
        if (editingId === record.id) {
          return (
            <Space size={4}>
              <Input
                size="small"
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                style={{ width: 100 }}
              />
              <Button size="small" type="primary" onClick={() => handleSave(record)}>{t('common.save')}</Button>
              <Button size="small" onClick={() => setEditingId(null)}>{t('common.cancel')}</Button>
            </Space>
          );
        }
        return <Typography.Text>{val as string}</Typography.Text>;
      },
    },
    { key: 'defaultValue', title: t('config.defaultValue'), dataIndex: 'defaultValue', width: 100 },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const cfg = STATUS_MAP[val as string] ?? STATUS_MAP.normal;
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 140,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
            disabled={editingId !== null && editingId !== record.id}
          >
            {t('common.edit')}
          </Button>
          <Popconfirm
            title={t('common.confirmDelete')}
            onConfirm={() => handleRestore(record)}
          >
            <Button type="link" size="small" icon={<RollbackOutlined />} danger>
              {t('common.reset')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ], [t, editingId, editValue, STATUS_MAP]);

  return (
    <ListPageLayout title={t('nav.config.liveParam')}>
      <FilterBar
        filterId="live-param-config"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />
      <DataTable<LiveParamRow>
        tableId="live-param-config"
        columns={columns}
        dataSource={tableSource}
        loading={isLoading}
        rowKey="id"
        selectable
        total={data?.total ?? tableSource.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1100 }}
      />
    </ListPageLayout>
  );
}
