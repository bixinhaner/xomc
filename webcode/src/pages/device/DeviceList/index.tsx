import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Modal, Space, Tag, Typography, message } from 'antd';
import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import StatisticsPanel from '@/components/StatisticsPanel';
import StatusIndicator from '@/components/StatusIndicator';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useDeviceList, useDeleteDevices } from '@/hooks/api/useDevices';
import { useT } from '@/hooks/useT';
import type { Device } from '@/types/device';

const { Link } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
  none: 'default',
};

const ENG_STATUS_COLOR: Record<string, string> = {
  commissioned: 'success',
  uncommissioned: 'default',
  decommissioned: 'error',
};

export default function DeviceList() {
  const t = useT();
  const navigate = useNavigate();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const queryParams = useMemo(
    () => ({ ...filterParams, page: currentPage, pageSize } as Parameters<typeof useDeviceList>[0]),
    [filterParams, currentPage, pageSize]
  );

  const { data, isLoading, refetch } = useDeviceList(queryParams);
  const deleteDevices = useDeleteDevices();

  const devices: Device[] = data?.items ?? [];
  const total = data?.total ?? 0;

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
    none: t('alarm.severity.none'),
  }), [t]);

  const ENG_STATUS_LABEL: Record<string, string> = useMemo(() => ({
    commissioned: t('device.engStatus.commissioned'),
    uncommissioned: t('device.engStatus.uncommissioned'),
    decommissioned: t('device.engStatus.decommissioned'),
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'name', label: t('device.name'), type: 'input' },
    { name: 'sn', label: 'SN', type: 'input' },
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
      name: 'productType',
      label: t('device.productType'),
      type: 'select',
      options: [
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'CPE', value: 'CPE' },
        { label: 'eGW', value: 'eGW' },
      ],
    },
    {
      name: 'networkType',
      label: t('device.networkType'),
      type: 'select',
      options: [
        { label: 'LTE-FDD', value: 'LTE-FDD' },
        { label: 'LTE-TDD', value: 'LTE-TDD' },
        { label: 'NR', value: 'NR' },
        { label: 'NB-IoT', value: 'NB-IoT' },
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
      name: 'alarmLevel',
      label: t('device.alarmLevel'),
      type: 'select',
      options: [
        { label: t('alarm.severity.critical'), value: 'critical' },
        { label: t('alarm.severity.major'), value: 'major' },
        { label: t('alarm.severity.minor'), value: 'minor' },
        { label: t('alarm.severity.warning'), value: 'warning' },
        { label: t('common.noAlarm'), value: 'none' },
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
        { label: '西北区', value: '西北区' },
        { label: '东北区', value: '东北区' },
      ],
    },
  ], [t]);

  // Statistics
  const statsItems = useMemo(() => {
    const online = devices.filter((d) => d.connStatus === 'online').length;
    const offline = devices.filter((d) => d.connStatus === 'offline').length;
    const alarmed = devices.filter((d) => d.alarmLevel !== 'none').length;
    return [
      { label: t('device.count.total'), value: total },
      { label: t('status.online'), value: online, color: '#52C41A' },
      { label: t('status.offline'), value: offline, color: '#8C8C8C' },
      { label: t('common.hasAlarm'), value: alarmed, color: '#FA8C16' },
    ];
  }, [devices, total, t]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const handleDelete = useCallback(
    (ids: string[]) => {
      Modal.confirm({
        title: t('common.confirmDelete'),
        content: t('common.deleteConfirmMsg', { count: ids.length }),
        okText: t('common.confirmDelete'),
        okType: 'danger',
        cancelText: t('common.cancel'),
        onOk: async () => {
          await deleteDevices.mutateAsync(ids);
          void message.success(t('common.deleteSuccess'));
          setSelectedRowKeys([]);
        },
      });
    },
    [deleteDevices, t]
  );

  const columns = useMemo(
    (): DataTableColumn<Device>[] => [
      {
        key: 'sn',
        title: 'SN',
        dataIndex: 'sn',
        width: 160,
        mono: true,
        copyable: true,
        render: (_val, record) => (
          <Link
            style={{ fontFamily: 'monospace', fontSize: 12 }}
            onClick={() => void navigate(`/device/detail/${record.sn}`)}
          >
            {record.sn}
          </Link>
        ),
      },
      { key: 'name', title: t('device.name'), dataIndex: 'name', width: 160, ellipsis: true },
      { key: 'vendor', title: t('device.vendor'), dataIndex: 'vendor', width: 100 },
      { key: 'productType', title: t('device.productType'), dataIndex: 'productType', width: 100 },
      { key: 'networkType', title: t('device.networkType'), dataIndex: 'networkType', width: 110 },
      { key: 'deviceModel', title: t('device.model'), dataIndex: 'deviceModel', width: 120, ellipsis: true },
      { key: 'region', title: t('device.region'), dataIndex: 'region', width: 100 },
      {
        key: 'connStatus',
        title: t('device.connStatus'),
        dataIndex: 'connStatus',
        width: 100,
        render: (_val, record) => (
          <StatusIndicator
            status={record.connStatus === 'online' ? 'online' : 'offline'}
            text={record.connStatus === 'online' ? t('status.online') : t('status.offline')}
          />
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
        key: 'engStatus',
        title: t('device.engStatus'),
        dataIndex: 'engStatus',
        width: 100,
        render: (_val, record) => (
          <Tag color={ENG_STATUS_COLOR[record.engStatus] ?? 'default'}>
            {ENG_STATUS_LABEL[record.engStatus] ?? record.engStatus}
          </Tag>
        ),
      },
      { key: 'ipAddress', title: t('device.ipAddress'), dataIndex: 'ipAddress', width: 140, mono: true },
      {
        key: 'lastOnlineTime',
        title: t('device.lastOnline'),
        dataIndex: 'lastOnlineTime',
        width: 160,
        render: (_val, record) =>
          record.lastOnlineTime
            ? new Date(record.lastOnlineTime).toLocaleString('zh-CN')
            : '-',
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
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => void navigate(`/device/edit/${record.id}`)}
            >
              {t('common.edit')}
            </Button>
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete([record.id])}
            >
              {t('common.delete')}
            </Button>
          </Space>
        ),
      },
    ],
    [navigate, handleDelete, t, SEVERITY_LABEL, ENG_STATUS_LABEL]
  );

  const batchActions = useMemo(
    (): BatchAction[] => [
      {
        key: 'batch-delete',
        label: t('common.batchDelete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: (keys) => handleDelete(keys as string[]),
      },
      {
        key: 'batch-config',
        label: t('common.batchConfig'),
        icon: <SettingOutlined />,
        onClick: () => void message.info(t('common.featureInDev')),
      },
      {
        key: 'export',
        label: t('common.export'),
        icon: <DownloadOutlined />,
        onClick: () => void message.info(t('common.exportInProgress')),
      },
    ],
    [handleDelete, t]
  );

  return (
    <ListPageLayout
      title={t('nav.device.list')}
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => void navigate('/device/registration')}
        >
          {t('common.addDevice')}
        </Button>
      }
    >
      <FilterBar
        filterId="device-list"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      <StatisticsPanel items={statsItems} style={{ marginBottom: 8 }} />

      <DataTable<Device>
        tableId="device-list-table"
        columns={columns}
        dataSource={devices}
        loading={isLoading}
        rowKey="id"
        selectable
        selectedRowKeys={selectedRowKeys}
        onSelectionChange={(keys) => setSelectedRowKeys(keys)}
        total={total}
        pageSize={pageSize}
        currentPage={currentPage}
        onPageChange={(page, size) => {
          setCurrentPage(page);
          setPageSize(size);
        }}
        batchActions={batchActions}
        onRefresh={() => void refetch()}
        defaultDensity="compact"
      />
    </ListPageLayout>
  );
}
