import React, { useMemo, useState } from 'react';
import { Input, Modal, Table, Tag } from 'antd';
import type { TableProps } from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import type { NE } from '@/types/device';
import { useThemeToken } from '@/hooks/useThemeToken';

export interface NESelectorProps {
  visible: boolean;
  onOk: (selectedSns: string[]) => void;
  onCancel: () => void;
  neList?: NE[];
  defaultSelected?: string[];
  max?: number;
  title?: string;
}

const MOCK_NE_LIST: NE[] = [];

const NESelector: React.FC<NESelectorProps> = ({
  visible,
  onOk,
  onCancel,
  neList = MOCK_NE_LIST,
  defaultSelected = [],
  max,
  title = '选择网元',
}) => {
  const [searchText, setSearchText] = useState('');
  const [selectedSns, setSelectedSns] = useState<string[]>(defaultSelected);
  const token = useThemeToken();

  const filtered = useMemo(() => {
    if (!searchText.trim()) return neList;
    const lower = searchText.toLowerCase();
    return neList.filter(
      (ne) =>
        ne.neName.toLowerCase().includes(lower) ||
        ne.sn.toLowerCase().includes(lower) ||
        ne.neType.toLowerCase().includes(lower)
    );
  }, [neList, searchText]);

  const columns: TableProps<NE>['columns'] = [
    {
      title: '网元名称',
      dataIndex: 'neName',
      key: 'neName',
      ellipsis: true,
      width: 180,
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
      title: '类型',
      dataIndex: 'neType',
      key: 'neType',
      width: 100,
    },
    {
      title: '状态',
      dataIndex: 'connStatus',
      key: 'connStatus',
      width: 80,
      render: (v: string) => (
        <Tag color={v === 'online' ? 'success' : 'default'}>
          {v === 'online' ? '在线' : '离线'}
        </Tag>
      ),
    },
  ];

  const rowSelection: TableProps<NE>['rowSelection'] = {
    selectedRowKeys: selectedSns,
    onChange: (keys) => {
      const sns = keys as string[];
      if (max && sns.length > max) return;
      setSelectedSns(sns);
    },
    getCheckboxProps: (record) => ({
      disabled:
        max !== undefined &&
        !selectedSns.includes(record.sn) &&
        selectedSns.length >= max,
    }),
  };

  const handleOk = () => {
    onOk(selectedSns);
  };

  const handleCancel = () => {
    setSelectedSns(defaultSelected);
    onCancel();
  };

  return (
    <Modal
      title={title}
      open={visible}
      onOk={handleOk}
      onCancel={handleCancel}
      okText="确认"
      cancelText="取消"
      width={640}
      destroyOnClose
    >
      <div style={{ marginBottom: 12 }}>
        <Input
          placeholder="搜索网元名称、SN、类型..."
          prefix={<SearchOutlined style={{ color: token.colorTextDisabled }} />}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          allowClear
        />
      </div>

      <div style={{ marginBottom: 8, fontSize: 12, color: token.colorTextSecondary }}>
        已选 {selectedSns.length} 个
        {max ? ` / 最多 ${max} 个` : ''}
      </div>

      <Table<NE>
        rowKey="sn"
        columns={columns}
        dataSource={filtered}
        rowSelection={rowSelection}
        size="small"
        pagination={{ pageSize: 8, showTotal: (t) => `共 ${t} 个` }}
        scroll={{ y: 280 }}
      />
    </Modal>
  );
};

export default NESelector;
