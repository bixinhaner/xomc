import React, { useCallback, useMemo, useState } from 'react';
import { Modal, Tag, message } from 'antd';
import {
  DeleteOutlined,
  ExportOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

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
const MOVE_TYPE_MAP: Record<string, string> = {
  '0': '自动',
  '1': '手动',
};

export default function RecycleBin() {
  const t = useT();
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [currentPage, setCurrentPage] = useState(1);

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
      Modal.confirm({
        title: '确认',
        content: `确认将选中的 ${ids.length} 个设备移出回收站？`,
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        icon: <ExportOutlined style={{ color: '#52C41A' }} />,
        onOk: () => {
          void message.success(t('status.success'));
          setSelectedRowKeys([]);
        },
      });
    },
    [t]
  );

  // 批量删除（带确认）
  const handlePermanentDelete = useCallback(
    (ids: React.Key[]) => {
      Modal.confirm({
        title: t('common.confirmDelete'),
        content: `确认删除选中的 ${ids.length} 个设备？此操作不可恢复。`,
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        okType: 'danger',
        onOk: () => {
          void message.success(t('common.deleteSuccess'));
          setSelectedRowKeys([]);
        },
      });
    },
    [t]
  );

  // 筛选字段配置
  const FILTER_FIELDS: FilterField[] = useMemo(
    () => [
      {
        name: 'searchText',
        label: '搜索',
        type: 'input',
        placeholder: '小站编码 / MAC地址',
      },
      {
        name: 'deviceType',
        label: '基站制式',
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
        label: '设备组',
        type: 'select',
        options: [
          { label: 'All', value: '' },
          ...MOCK_DEVICE_GROUPS.map((g) => ({ label: g.group_name, value: g.id })),
        ],
      },
      {
        name: 'moveType',
        label: '回收方式',
        type: 'select',
        options: [
          { label: 'All', value: '' },
          { label: '自动', value: '0' },
          { label: '手动', value: '1' },
        ],
      },
    ],
    []
  );

  // 列定义
  const columns: DataTableColumn<RecycleDeviceItem>[] = useMemo(
    () => [
      {
        key: 'serial_number',
        title: '小站编码',
        dataIndex: 'serial_number',
        width: 140,
        mono: true,
        copyable: true,
      },
      {
        key: 'deviceType',
        title: '基站制式',
        dataIndex: 'deviceType',
        width: 100,
        render: (v) => (
          <Tag color={DEVICE_TYPE_COLOR[v as DeviceType] || 'default'}>{String(v)}</Tag>
        ),
      },
      { key: 'host_name', title: 'HostName', dataIndex: 'host_name', width: 140, ellipsis: true },
      {
        key: 'mac',
        title: 'MAC地址',
        width: 130,
        mono: true,
        render: (_v, record) => record.mac_address || record.macaddress || '-',
      },
      { key: 'longitude', title: '经度', dataIndex: 'longitude', width: 90 },
      { key: 'latitude', title: '纬度', dataIndex: 'latitude', width: 90 },
      { key: 'height', title: '高度', dataIndex: 'height', width: 70 },
      { key: 'offlineDays', title: '离线天数', dataIndex: 'offlineDays', width: 90 },
      { key: 'group_name', title: '设备组名称', dataIndex: 'group_name', width: 120 },
      {
        key: 'moveType',
        title: '回收方式',
        dataIndex: 'moveType',
        width: 90,
        render: (v) => <Tag color={v === '1' ? 'blue' : 'green'}>{MOVE_TYPE_MAP[String(v)]}</Tag>,
      },
      { key: 'moveTime', title: '回收时间', dataIndex: 'moveTime', width: 160 },
      { key: 'move_author', title: '账户', dataIndex: 'move_author', width: 90 },
    ],
    []
  );

  // 批量操作
  const batchActions: BatchAction[] = useMemo(
    () => [
      {
        key: 'batch-restore',
        label: '移出回收站',
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
    <ListPageLayout title="回收站" subtitle={`${t('table.total')} ${filteredData.length}`}>
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
    </ListPageLayout>
  );
}
