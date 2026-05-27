/**
 * 设备组选择 Modal — 仿 DevicePickerModal 但简化（设备组数量小，前端一次性拿全量树后摊平，
 * 客户端按名称过滤，多选 + 已选面板）。
 */

import { useMemo, useState } from 'react';
import { Modal, Input, Table, Space, Button, Tag, Typography, Tooltip, Divider } from 'antd';
import { SearchOutlined, ClearOutlined } from '@ant-design/icons';
import { useDeviceGroups } from '@core/hooks/api/useDevices';
import type { DeviceGroup } from '@core/types/device';

const { Text } = Typography;

interface DeviceGroupPickerModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (selectedIds: string[]) => void;
  initialSelected?: string[];
}

export default function DeviceGroupPickerModal({
  open,
  onClose,
  onConfirm,
  initialSelected = [],
}: DeviceGroupPickerModalProps) {
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState<string[]>(initialSelected);

  const { data, isLoading } = useDeviceGroups();
  const allGroups = data?.groups ?? [];

  const filtered = useMemo(() => {
    const kw = search.trim().toLowerCase();
    if (!kw) return allGroups;
    return allGroups.filter((g) => g.name.toLowerCase().includes(kw));
  }, [allGroups, search]);

  // 名字 → 已选项展示用
  const nameById = useMemo(() => {
    const m = new Map<string, string>();
    allGroups.forEach((g) => m.set(g.id, g.name));
    return m;
  }, [allGroups]);

  const columns = useMemo(
    () => [
      { title: '设备组名', dataIndex: 'name', key: 'name', ellipsis: true },
      { title: '设备数', dataIndex: 'deviceCount', key: 'deviceCount', width: 90 },
      {
        title: '类型',
        dataIndex: 'builtIn',
        key: 'builtIn',
        width: 80,
        render: (v: number) => (v === 1 ? <Tag color="blue">内置</Tag> : <Tag>自定义</Tag>),
      },
      { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    ],
    [],
  );

  const handleConfirm = () => {
    onConfirm(selected);
    onClose();
  };

  const rowSelection = {
    selectedRowKeys: selected,
    preserveSelectedRowKeys: true,
    onChange: (keys: React.Key[]) => setSelected(keys as string[]),
  };

  return (
    <Modal
      title={`选择设备组（已选 ${selected.length} 个）`}
      open={open}
      onCancel={onClose}
      onOk={handleConfirm}
      okText="确定"
      cancelText="取消"
      width={760}
      destroyOnHidden
    >
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <Input
          placeholder="按设备组名 搜索"
          prefix={<SearchOutlined />}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{ width: 320 }}
          allowClear
        />

        <Table<DeviceGroup>
          rowKey="id"
          size="small"
          loading={isLoading}
          columns={columns}
          dataSource={filtered}
          rowSelection={rowSelection}
          pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
          scroll={{ y: 320 }}
        />

        <Divider style={{ margin: '4px 0' }} />

        <div>
          <Space style={{ marginBottom: 6 }}>
            <Text strong>已选设备组：</Text>
            <Button
              size="small"
              icon={<ClearOutlined />}
              onClick={() => setSelected([])}
              disabled={selected.length === 0}
            >
              全部清空
            </Button>
          </Space>
          <div
            style={{
              maxHeight: 100,
              overflowY: 'auto',
              background: 'var(--ant-color-fill-quaternary, #fafafa)',
              padding: 8,
              borderRadius: 4,
            }}
          >
            {selected.length === 0 ? (
              <Text type="secondary">未选择任何设备组</Text>
            ) : (
              selected.map((id) => {
                const name = nameById.get(id) ?? id;
                return (
                  <Tooltip key={id} title={`${name} (${id})`}>
                    <Tag
                      closable
                      onClose={() => setSelected(selected.filter((s) => s !== id))}
                      style={{ marginBottom: 4 }}
                    >
                      {name}
                    </Tag>
                  </Tooltip>
                );
              })
            )}
          </div>
        </div>
      </Space>
    </Modal>
  );
}
