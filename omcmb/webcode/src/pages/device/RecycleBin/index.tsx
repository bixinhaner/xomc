import React, { useCallback, useMemo, useState } from 'react';
import { App, Button, Tag } from 'antd';
import {
  DeleteOutlined,
  ExportOutlined,
  ImportOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import ImportModal from './ImportModal';

// 设备类型
type DeviceType = 'eNB' | 'gNB' | 'CPE';

// 设备类型颜色映射
const DEVICE_TYPE_COLOR: Record<DeviceType, string> = {
  eNB: 'blue',
  gNB: 'green',
  CPE: 'orange',
};

// 回收站设备通用接口
interface RecycleDeviceItem {
  id: string;
  serial_number: string;
  deviceType: DeviceType;
  offlineDays: number;
  group_id: string;
  group_name: string;
  moveType: '0' | '1';
  moveTime: string;
  move_author: string;
  product?: string;
  host_name?: string;
  mac_address?: string;
  longitude?: number | string;
  latitude?: number | string;
  height?: number | string;
  macaddress?: string;
  distance?: number | string;
}

// 设备组
interface DeviceGroup {
  id: string;
  group_name: string;
}

// Mock 设备组
const MOCK_DEVICE_GROUPS: DeviceGroup[] = [
  { id: '1', group_name: 'Default Group' },
  { id: '2', group_name: 'Beijing Region' },
  { id: '3', group_name: 'Shanghai Region' },
  { id: '4', group_name: 'Guangzhou Region' },
  { id: '5', group_name: 'Test Group' },
];

// Mock 数据
const MOCK_ALL_DATA: RecycleDeviceItem[] = [
  {
    id: '1',
    serial_number: 'SN-ENB-001',
    deviceType: 'eNB',
    host_name: 'eNB-Beijing-001',
    mac_address: '00:11:22:33:44:01',
    longitude: 116.48,
    latitude: 39.99,
    height: 30,
    offlineDays: 5,
    group_id: '2',
    group_name: 'Beijing Region',
    moveType: '1',
    moveTime: '2024-03-10 10:00:00',
    move_author: 'admin',
  },
  {
    id: '2',
    serial_number: 'SN-ENB-002',
    deviceType: 'eNB',
    host_name: 'eNB-Shanghai-001',
    mac_address: '00:11:22:33:44:02',
    longitude: 121.47,
    latitude: 31.23,
    height: 25,
    offlineDays: 10,
    group_id: '3',
    group_name: 'Shanghai Region',
    moveType: '0',
    moveTime: '2024-03-08 14:30:00',
    move_author: 'system',
  },
  {
    id: '3',
    serial_number: 'SN-GNB-001',
    deviceType: 'gNB',
    mac_address: '00:11:22:33:44:03',
    longitude: 113.26,
    latitude: 23.13,
    height: 35,
    offlineDays: 3,
    group_id: '4',
    group_name: 'Guangzhou Region',
    moveType: '1',
    moveTime: '2024-03-12 09:00:00',
    move_author: 'admin',
  },
  {
    id: '4',
    serial_number: 'SN-CPE-001',
    deviceType: 'CPE',
    macaddress: 'AA:BB:CC:DD:EE:01',
    longitude: 116.31,
    latitude: 40.05,
    height: 5,
    distance: 1200,
    offlineDays: 7,
    group_id: '1',
    group_name: 'Default Group',
    moveType: '1',
    moveTime: '2024-03-09 16:00:00',
    move_author: 'admin',
  },
];

// 回收方式映射
const getMoveTypeLabel = (t: (key: string) => string, value: string) => {
  const map: Record<string, string> = {
    '0': t('recycle.auto'),
    '1': t('recycle.manual'),
  };
  return map[value] || value;
};

export default function RecycleBin() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [importModalOpen, setImportModalOpen] = useState(false);

  // 过滤数据
  const filteredData = useMemo(() => {
    return MOCK_ALL_DATA.filter((item) => {
      if (filterParams.deviceType && item.deviceType !== filterParams.deviceType) {
        return false;
      }
      if (filterParams.searchText) {
        const search = String(filterParams.searchText).toLowerCase();
        const matchSn = item.serial_number?.toLowerCase().includes(search);
        const matchMac = (item.mac_address || item.macaddress)?.toLowerCase().includes(search);
        if (!matchSn && !matchMac) return false;
      }
      if (filterParams.group_id && item.group_id !== filterParams.group_id) {
        return false;
      }
      if (filterParams.moveType && item.moveType !== filterParams.moveType) {
        return false;
      }
      return true;
    });
  }, [filterParams]);

  // 移出回收站（带确认）
  const handleRestore = useCallback(
    (ids: React.Key[]) => {
      modal.confirm({
        title: t('common.confirm'),
        content: t('recycle.restoreConfirm', { count: ids.length }),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        icon: <ExportOutlined style={{ color: '#52C41A' }} />,
        onOk: () => {
          message.success(t('status.success'));
          setSelectedRowKeys([]);
        },
      });
    },
    [t, modal, message]
  );

  // 批量删除（带确认）
  const handlePermanentDelete = useCallback(
    (ids: React.Key[]) => {
      modal.confirm({
        title: t('common.confirmDelete'),
        content: t('recycle.deleteConfirm', { count: ids.length }),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        okType: 'danger',
        onOk: () => {
          message.success(t('common.deleteSuccess'));
          setSelectedRowKeys([]);
        },
      });
    },
    [t, modal, message]
  );

  // 打开导入弹窗
  const handleOpenImportModal = useCallback(() => {
    setImportModalOpen(true);
  }, []);

  // 导入完成
  const handleImportComplete = useCallback(() => {
    // TODO: 刷新数据
    message.success(t('common.success'));
  }, [message, t]);

  // 筛选字段配置
  const FILTER_FIELDS: FilterField[] = useMemo(
    () => [
      {
        name: 'searchText',
        label: t('common.search'),
        type: 'input',
        placeholder: t('recycle.searchPlaceholder'),
      },
      {
        name: 'deviceType',
        label: t('device.radioMode'),
        type: 'select',
        options: [
          { label: 'All', value: '' },
          { label: 'eNB', value: 'eNB' },
          { label: 'gNB', value: 'gNB' },
          { label: 'CPE', value: 'CPE' },
        ],
      },
      {
        name: 'group_id',
        label: t('device.groupName'),
        type: 'select',
        options: [
          { label: 'All', value: '' },
          ...MOCK_DEVICE_GROUPS.map((g) => ({ label: g.group_name, value: g.id })),
        ],
      },
      {
        name: 'moveType',
        label: t('recycle.moveType'),
        type: 'select',
        options: [
          { label: 'All', value: '' },
          { label: t('recycle.auto'), value: '0' },
          { label: t('recycle.manual'), value: '1' },
        ],
      },
    ],
    [t]
  );

  // 列定义
  const columns: DataTableColumn<RecycleDeviceItem>[] = useMemo(
    () => [
      {
        key: 'serial_number',
        title: t('device.serialNumber'),
        dataIndex: 'serial_number',
        width: 140,
        mono: true,
        copyable: true,
      },
      {
        key: 'deviceType',
        title: t('device.radioMode'),
        dataIndex: 'deviceType',
        width: 100,
        render: (v) => (
          <Tag color={DEVICE_TYPE_COLOR[v as DeviceType] || 'default'}>{String(v)}</Tag>
        ),
      },
      { key: 'host_name', title: 'HostName', dataIndex: 'host_name', width: 140, ellipsis: true },
      {
        key: 'mac',
        title: t('device.macAddress'),
        width: 130,
        mono: true,
        render: (_v, record) => record.mac_address || record.macaddress || '-',
      },
      { key: 'longitude', title: t('device.longitude'), dataIndex: 'longitude', width: 90 },
      { key: 'latitude', title: t('device.latitude'), dataIndex: 'latitude', width: 90 },
      { key: 'height', title: t('recycle.height'), dataIndex: 'height', width: 70 },
      { key: 'offlineDays', title: t('recycle.offlineDays'), dataIndex: 'offlineDays', width: 90 },
      { key: 'group_name', title: t('recycle.groupName'), dataIndex: 'group_name', width: 120 },
      {
        key: 'moveType',
        title: t('recycle.moveType'),
        dataIndex: 'moveType',
        width: 90,
        render: (v) => <Tag color={v === '1' ? 'blue' : 'green'}>{getMoveTypeLabel(t, String(v))}</Tag>,
      },
      { key: 'moveTime', title: t('recycle.moveTime'), dataIndex: 'moveTime', width: 160 },
      { key: 'move_author', title: t('recycle.account'), dataIndex: 'move_author', width: 90 },
    ],
    [t]
  );

  // 批量操作
  const batchActions: BatchAction[] = useMemo(
    () => [
      {
        key: 'batch-restore',
        label: t('recycle.restore'),
        icon: <ExportOutlined />,
        onClick: handleRestore,
      },
      {
        key: 'batch-delete',
        label: t('common.batchDelete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: handlePermanentDelete,
      },
    ],
    [handleRestore, handlePermanentDelete, t]
  );

  return (
    <ListPageLayout
      title={t('nav.device.recycle')}
      subtitle={`${t('table.total')} ${filteredData.length}`}
      extra={
        <Button type="primary" icon={<ImportOutlined />} onClick={handleOpenImportModal}>
          {t('common.import')}
        </Button>
      }
    >
      <FilterBar
        filterId="recycle-bin"
        fields={FILTER_FIELDS}
        onSearch={(v) => {
          setFilterParams(v);
          setCurrentPage(1);
        }}
        onReset={() => {
          setFilterParams({});
          setCurrentPage(1);
        }}
        collapsedRows={1}
      />

      <DataTable<RecycleDeviceItem>
        tableId="recycle-bin-table"
        columns={columns}
        dataSource={filteredData}
        loading={false}
        rowKey="id"
        selectable
        selectedRowKeys={selectedRowKeys}
        onSelectionChange={(keys) => setSelectedRowKeys(keys)}
        total={filteredData.length}
        pageSize={20}
        currentPage={currentPage}
        onPageChange={(p) => setCurrentPage(p)}
        batchActions={batchActions}
        defaultDensity="compact"
      />

      <ImportModal
        open={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        onConfirm={handleImportComplete}
      />
    </ListPageLayout>
  );
}
