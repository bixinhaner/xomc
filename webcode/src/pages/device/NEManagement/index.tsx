import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Space, Tag } from 'antd';
import { EyeOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import StatusIndicator from '@/components/StatusIndicator';
import { useNEList } from '@/hooks/api/useDevices';
import { useT } from '@/hooks/useT';
import type { NE } from '@/types/device';

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'red', major: 'orange', minor: 'gold', warning: 'blue', none: 'default',
};

export default function NEManagement() {
  const t = useT();
  const navigate = useNavigate();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
    none: t('alarm.severity.none'),
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('table.name'), type: 'input', placeholder: t('common.placeholder') },
    {
      name: 'neType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'CPE', value: 'CPE' },
        { label: 'eGW', value: 'eGW' },
      ],
    },
    {
      name: 'vendor',
      label: t('device.vendor'),
      type: 'select',
      options: [
        { label: '华为', value: '华为' },
        { label: '中兴', value: '中兴' },
        { label: '爱立信', value: '爱立信' },
        { label: '大唐', value: '大唐' },
        { label: '京信', value: '京信' },
      ],
    },
    {
      name: 'connStatus',
      label: t('device.connStatus'),
      type: 'select',
      options: [
        { label: t('status.online'), value: 'online' },
        { label: t('status.offline'), value: 'offline' },
      ],
    },
    {
      name: 'region',
      label: t('device.region'),
      type: 'select',
      options: [
        { label: '华北区', value: '华北区' },
        { label: '华东区', value: '华东区' },
        { label: '华南区', value: '华南区' },
        { label: '西南区', value: '西南区' },
      ],
    },
  ], [t]);

  const queryParams = useMemo(
    () => ({
      keyword: (filterParams.keyword as string | undefined),
      page: currentPage,
      pageSize,
    }),
    [filterParams, currentPage, pageSize]
  );

  const { data, isLoading, refetch } = useNEList(queryParams);
  const neList: NE[] = data?.items ?? [];
  const total = data?.total ?? 0;

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const columns = useMemo(
    (): DataTableColumn<NE>[] => [
      { key: 'neName', title: t('table.name'), dataIndex: 'neName', width: 180, ellipsis: true },
      {
        key: 'sn',
        title: 'SN',
        dataIndex: 'sn',
        width: 160,
        mono: true,
        copyable: true,
        render: (_val, record) => (
          <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{record.sn}</span>
        ),
      },
      { key: 'neType', title: t('table.type'), dataIndex: 'neType', width: 90, render: (v) => <Tag>{String(v)}</Tag> },
      { key: 'vendor', title: t('device.vendor'), dataIndex: 'vendor', width: 90 },
      { key: 'region', title: t('device.region'), dataIndex: 'region', width: 100 },
      { key: 'subnet', title: t('table.description'), dataIndex: 'subnet', width: 150, mono: true },
      {
        key: 'connStatus',
        title: t('device.connStatus'),
        dataIndex: 'connStatus',
        width: 100,
        render: (_val, record) => (
          <StatusIndicator status={record.connStatus === 'online' ? 'online' : 'offline'} />
        ),
      },
      {
        key: 'alarmLevel',
        title: t('device.alarmLevel'),
        dataIndex: 'alarmLevel',
        width: 100,
        render: (_val, record) => (
          <Tag color={SEVERITY_COLOR[record.alarmLevel] ?? 'default'}>
            {SEVERITY_LABEL[record.alarmLevel] ?? record.alarmLevel}
          </Tag>
        ),
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 120,
        fixed: 'right',
        render: (_val, record) => (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => void navigate(`/device/detail/${record.sn}`)}
            >
              {t('common.detail')}
            </Button>
          </Space>
        ),
      },
    ],
    [navigate, t, SEVERITY_LABEL]
  );

  return (
    <ListPageLayout
      title={t('nav.device.ne')}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
            {t('common.refresh')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />}>
            {t('common.add')}
          </Button>
        </Space>
      }
    >
      <FilterBar
        filterId="ne-management"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      <DataTable<NE>
        tableId="ne-management-table"
        columns={columns}
        dataSource={neList}
        loading={isLoading}
        rowKey="id"
        selectable
        total={total}
        pageSize={pageSize}
        currentPage={currentPage}
        onPageChange={(page, size) => {
          setCurrentPage(page);
          setPageSize(size);
        }}
        onRefresh={() => void refetch()}
        defaultDensity="compact"
      />
    </ListPageLayout>
  );
}
