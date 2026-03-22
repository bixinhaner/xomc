import React, { useCallback, useMemo } from 'react';
import { Button, Tag, Typography } from 'antd';
import { DownloadOutlined, EditOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import StatusIndicator from '@/components/StatusIndicator';
import type { Device, EngStatus } from '@/types/device';

const { Title, Text } = Typography;

export interface DeviceListPanelProps {
  devices: Device[];
  total: number;
  loading: boolean;
  selectedDeviceIds: React.Key[];
  currentPage: number;
  pageSize: number;
  selectedGroupName: string | undefined;
  batchActions: BatchAction[];
  onSelectionChange: (keys: React.Key[]) => void;
  onPageChange: (page: number, size: number) => void;
  onRefresh: () => void;
  onExport: (format: 'xlsx' | 'csv') => void;
  onEditDevice: (device: Device) => void;
  t: (id: string, values?: Record<string, unknown>) => string;
}

export default function DeviceListPanel({
  devices,
  total,
  loading,
  selectedDeviceIds,
  currentPage,
  pageSize,
  selectedGroupName,
  batchActions,
  onSelectionChange,
  onPageChange,
  onRefresh,
  onExport,
  onEditDevice,
  t,
}: DeviceListPanelProps) {
  const calculateOfflineDays = useCallback((lastOnlineTime: string): number => {
    if (!lastOnlineTime) return 0;
    const lastOnline = new Date(lastOnlineTime);
    const now = new Date();
    const diffMs = now.getTime() - lastOnline.getTime();
    return Math.max(0, Math.floor(diffMs / (1000 * 60 * 60 * 24)));
  }, []);

  const columns = useMemo(
    (): DataTableColumn<Device>[] => [
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 80,
        fixed: 'left',
        render: (_val, record) => (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => onEditDevice(record)}
          >
            {t('common.edit')}
          </Button>
        ),
      },
      {
        key: 'connStatus',
        title: t('device.connStatus'),
        dataIndex: 'connStatus',
        width: 90,
        render: (val: string) => (
          <StatusIndicator
            status={val === 'online' ? 'online' : 'offline'}
            text={val === 'online' ? t('status.online') : t('status.offline')}
          />
        ),
      },
      {
        key: 'engStatus',
        title: t('device.installStatus'),
        dataIndex: 'engStatus',
        width: 100,
        render: (val: EngStatus) => {
          const statusMap: Record<EngStatus, { label: string; color: string }> = {
            commissioned: { label: t('device.engStatus.commissioned'), color: 'green' },
            uncommissioned: { label: t('device.engStatus.uncommissioned'), color: 'orange' },
            decommissioned: { label: t('device.engStatus.decommissioned'), color: 'red' },
          };
          const { label, color } = statusMap[val] || { label: val, color: 'default' };
          return <Tag color={color}>{label}</Tag>;
        },
      },
      {
        key: 'sn',
        title: t('device.serialNumber'),
        dataIndex: 'sn',
        width: 150,
        mono: true,
        copyable: true,
      },
      { key: 'name', title: t('device.stationName'), dataIndex: 'name', width: 160, ellipsis: true },
      { key: 'macAddress', title: t('device.macAddress'), dataIndex: 'macAddress', width: 150, mono: true },
      { key: 'groupName', title: t('device.groupName'), dataIndex: 'groupName', width: 140, ellipsis: true },
      { key: 'longitude', title: t('device.longitude'), dataIndex: 'longitude', width: 100 },
      { key: 'latitude', title: t('device.latitude'), dataIndex: 'latitude', width: 100 },
      { key: 'gpsHeight', title: t('device.height'), dataIndex: 'gpsHeight', width: 80 },
      {
        key: 'offlineDays',
        title: t('device.offlineDays'),
        width: 100,
        render: (_val, record) => {
          if (record.connStatus === 'online') return '-';
          return calculateOfflineDays(record.lastOnlineTime);
        },
      },
    ],
    [t, calculateOfflineDays, onEditDevice]
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', padding: 16, gap: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Title level={5} style={{ margin: 0 }}>
          {selectedGroupName ?? t('common.all')}
          <Text type="secondary" style={{ fontSize: 13, marginLeft: 8, fontWeight: 400 }}>
            {t('table.total')} {total}
          </Text>
        </Title>
        <Button
          type="primary"
          icon={<DownloadOutlined />}
          onClick={() => onExport('xlsx')}
        >
          {t('common.export')}
        </Button>
      </div>

      <DataTable<Device>
        tableId="device-grouping-table"
        columns={columns}
        dataSource={devices}
        loading={loading}
        rowKey="id"
        selectable
        selectedRowKeys={selectedDeviceIds}
        onSelectionChange={onSelectionChange}
        batchActions={batchActions}
        total={total}
        pageSize={pageSize}
        currentPage={currentPage}
        onPageChange={onPageChange}
        onRefresh={onRefresh}
        defaultDensity="compact"
      />
    </div>
  );
}
