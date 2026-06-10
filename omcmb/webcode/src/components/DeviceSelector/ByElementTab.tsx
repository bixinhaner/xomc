import React, { useMemo, useState } from 'react';
import { Input, Table, Tag } from 'antd';
import type { TableProps } from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import type { Device } from '@core/types/device';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';

interface ByElementTabProps {
  devices: Device[];
  selectedSns: string[];
  onSelectionChange: (sns: string[]) => void;
  max?: number;
}

const ByElementTab: React.FC<ByElementTabProps> = ({
  devices,
  selectedSns,
  onSelectionChange,
  max,
}) => {
  const t = useT();
  const [searchText, setSearchText] = useState('');
  const token = useThemeToken();

  const filtered = useMemo(() => {
    if (!searchText.trim()) return devices;
    const lower = searchText.toLowerCase();
    return devices.filter(
      (d) =>
        d.name.toLowerCase().includes(lower) ||
        d.sn.toLowerCase().includes(lower) ||
        d.deviceModel.toLowerCase().includes(lower)
    );
  }, [devices, searchText]);

  const columns: TableProps<Device>['columns'] = [
    {
      title: t('device.name'),
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
      width: 160,
    },
    {
      title: 'SN',
      dataIndex: 'sn',
      key: 'sn',
      width: 140,
      render: (v: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</span>
      ),
    },
    {
      title: t('device.model'),
      dataIndex: 'deviceModel',
      key: 'deviceModel',
      width: 120,
    },
    {
      title: t('device.productClass'),
      dataIndex: 'productClass',
      key: 'productClass',
      width: 100,
    },
    {
      title: t('common.status'),
      dataIndex: 'connStatus',
      key: 'connStatus',
      width: 80,
      render: (v: string) => (
        <Tag color={v === 'online' ? 'success' : 'default'}>
          {v === 'online' ? t('device.online') : t('device.offline')}
        </Tag>
      ),
    },
  ];

  const rowSelection: TableProps<Device>['rowSelection'] = {
    selectedRowKeys: selectedSns,
    onChange: (keys) => {
      const sns = keys as string[];
      if (max && sns.length > max) return;
      onSelectionChange(sns);
    },
    getCheckboxProps: (record) => ({
      disabled: max !== undefined && !selectedSns.includes(record.sn) && selectedSns.length >= max,
    }),
  };

  return (
    <div>
      <Input
        placeholder={t('deviceSelector.searchPlaceholder')}
        prefix={<SearchOutlined style={{ color: token.colorTextDisabled }} />}
        value={searchText}
        onChange={(e) => setSearchText(e.target.value)}
        allowClear
        style={{ marginBottom: 12 }}
      />
      <Table<Device>
        rowKey="sn"
        columns={columns}
        dataSource={filtered}
        rowSelection={rowSelection}
        size="small"
        pagination={{ pageSize: 10, showTotal: (total) => t('deviceSelector.totalUnit', { count: total }) }}
        scroll={{ y: 300 }}
      />
    </div>
  );
};

export default ByElementTab;
