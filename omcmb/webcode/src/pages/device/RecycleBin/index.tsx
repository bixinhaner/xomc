import React, { useCallback, useMemo, useState } from 'react';
import { Button, Modal, Space, Tag, Typography, message } from 'antd';
import {
  DeleteOutlined,
  RedoOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

interface DeletedDevice {
  id: string;
  sn: string;
  deviceName: string;
  vendor: string;
  productType: string;
  region: string;
  deleteTime: string;
  operator: string;
  reason: string;
}

const MOCK_DELETED: DeletedDevice[] = [
  {
    id: '1', sn: 'SN-DEL-001', deviceName: '废弃基站-001', vendor: '华为',
    productType: 'eNB', region: '华北区', deleteTime: '2024-02-20 14:30:00',
    operator: '张工', reason: '设备老旧，已退网',
  },
  {
    id: '2', sn: 'SN-DEL-002', deviceName: '测试设备-002', vendor: '中兴',
    productType: 'gNB', region: '华东区', deleteTime: '2024-02-22 09:15:00',
    operator: '李工', reason: '测试完毕，清理',
  },
  {
    id: '3', sn: 'SN-DEL-003', deviceName: '故障设备-003', vendor: '爱立信',
    productType: 'CPE', region: '华南区', deleteTime: '2024-02-25 16:00:00',
    operator: '王工', reason: '设备故障不可修复',
  },
  {
    id: '4', sn: 'SN-DEL-004', deviceName: '误删设备-004', vendor: '大唐',
    productType: 'eGW', region: '西南区', deleteTime: '2024-02-28 11:00:00',
    operator: '赵工', reason: '误操作删除',
  },
  {
    id: '5', sn: 'SN-DEL-005', deviceName: '迁移设备-005', vendor: '华为',
    productType: 'eNB', region: '西北区', deleteTime: '2024-03-01 08:30:00',
    operator: '陈工', reason: '系统迁移，旧设备归档',
  },
];

export default function RecycleBin() {
  const t = useT();
  const [items, setItems] = useState<DeletedDevice[]>(MOCK_DELETED);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'sn', label: 'SN', type: 'input' },
    { name: 'deviceName', label: t('device.name'), type: 'input' },
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
    { name: 'timeRange', label: t('table.time'), type: 'date-range' },
  ], [t]);

  const filteredItems = useMemo(() => {
    return items.filter((item) => {
      if (filterParams.sn && !item.sn.toLowerCase().includes(String(filterParams.sn).toLowerCase())) return false;
      if (filterParams.deviceName && !item.deviceName.toLowerCase().includes(String(filterParams.deviceName).toLowerCase())) return false;
      if (filterParams.vendor && item.vendor !== filterParams.vendor) return false;
      return true;
    });
  }, [items, filterParams]);

  const handleRestore = useCallback((ids: string[]) => {
    Modal.confirm({
      title: t('common.confirm'),
      content: t('common.confirm'),
      okText: t('common.confirm'),
      icon: <RedoOutlined style={{ color: '#52C41A' }} />,
      onOk: () => {
        setItems((prev) => prev.filter((item) => !ids.includes(item.id)));
        setSelectedRowKeys([]);
        void message.success(t('status.success'));
      },
    });
  }, [t]);

  const handlePermanentDelete = useCallback((ids: string[]) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: t('common.deleteConfirmMsg', { count: ids.length }),
      okText: t('common.confirmDelete'),
      okType: 'danger',
      onOk: () => {
        setItems((prev) => prev.filter((item) => !ids.includes(item.id)));
        setSelectedRowKeys([]);
        void message.success(t('common.deleteSuccess'));
      },
    });
  }, [t]);

  const columns = useMemo(
    (): DataTableColumn<DeletedDevice>[] => [
      {
        key: 'sn',
        title: 'SN',
        dataIndex: 'sn',
        width: 150,
        mono: true,
        copyable: true,
        render: (v) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(v)}</Text>,
      },
      { key: 'deviceName', title: t('device.name'), dataIndex: 'deviceName', width: 180, ellipsis: true },
      { key: 'vendor', title: t('device.vendor'), dataIndex: 'vendor', width: 90 },
      {
        key: 'productType',
        title: t('device.productType'),
        dataIndex: 'productType',
        width: 90,
        render: (v) => <Tag>{String(v)}</Tag>,
      },
      { key: 'region', title: t('device.region'), dataIndex: 'region', width: 90 },
      { key: 'deleteTime', title: t('table.time'), dataIndex: 'deleteTime', width: 160 },
      { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 90 },
      {
        key: 'reason',
        title: t('table.description'),
        dataIndex: 'reason',
        width: 200,
        ellipsis: true,
        render: (v) => <Text type="secondary">{String(v)}</Text>,
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 160,
        fixed: 'right',
        render: (_val, record) => (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<RedoOutlined />}
              onClick={() => handleRestore([record.id])}
            >
              {t('common.back')}
            </Button>
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handlePermanentDelete([record.id])}
            >
              {t('common.delete')}
            </Button>
          </Space>
        ),
      },
    ],
    [handleRestore, handlePermanentDelete, t]
  );

  const batchActions = useMemo(
    (): BatchAction[] => [
      {
        key: 'batch-restore',
        label: t('common.back'),
        icon: <RedoOutlined />,
        onClick: (keys) => handleRestore(keys as string[]),
      },
      {
        key: 'batch-delete',
        label: t('common.batchDelete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: (keys) => handlePermanentDelete(keys as string[]),
      },
    ],
    [handleRestore, handlePermanentDelete, t]
  );

  return (
    <ListPageLayout
      title={t('nav.device.recycle')}
      subtitle={`${t('table.total')} ${items.length}`}
    >
      <FilterBar
        filterId="recycle-bin"
        fields={FILTER_FIELDS}
        onSearch={(v) => { setFilterParams(v); setCurrentPage(1); }}
        onReset={() => { setFilterParams({}); setCurrentPage(1); }}
        collapsedRows={1}
      />

      <DataTable<DeletedDevice>
        tableId="recycle-bin-table"
        columns={columns}
        dataSource={filteredItems}
        loading={false}
        rowKey="id"
        selectable
        selectedRowKeys={selectedRowKeys}
        onSelectionChange={(keys) => setSelectedRowKeys(keys)}
        total={filteredItems.length}
        pageSize={20}
        currentPage={currentPage}
        onPageChange={(p) => setCurrentPage(p)}
        batchActions={batchActions}
        defaultDensity="compact"
      />
    </ListPageLayout>
  );
}
