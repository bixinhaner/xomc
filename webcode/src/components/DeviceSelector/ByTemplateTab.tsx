import React from 'react';
import { Button, Table, Tag, Typography } from 'antd';
import type { TableProps } from 'antd';

interface DeviceTemplate {
  id: string;
  name: string;
  description: string;
  deviceType: string;
  count: number;
  sns: string[];
}

interface ByTemplateTabProps {
  selectedSns: string[];
  onSelectionChange: (sns: string[]) => void;
}

// Predefined templates (in production these would come from API)
const PRESET_TEMPLATES: DeviceTemplate[] = [
  {
    id: 'tpl_all_online',
    name: '所有在线设备',
    description: '当前连接状态为在线的所有设备',
    deviceType: '全部',
    count: 0,
    sns: [],
  },
  {
    id: 'tpl_all_enb',
    name: '所有 eNB 设备',
    description: '网络类型为 LTE 的基站设备',
    deviceType: 'eNB',
    count: 0,
    sns: [],
  },
  {
    id: 'tpl_all_gnb',
    name: '所有 gNB 设备',
    description: '网络类型为 NR 的基站设备',
    deviceType: 'gNB',
    count: 0,
    sns: [],
  },
  {
    id: 'tpl_alarm_devices',
    name: '有告警设备',
    description: '当前存在活跃告警的设备',
    deviceType: '全部',
    count: 0,
    sns: [],
  },
  {
    id: 'tpl_unmanaged',
    name: '未纳管设备',
    description: '管理状态为未纳管或预纳管的设备',
    deviceType: '全部',
    count: 0,
    sns: [],
  },
];

const ByTemplateTab: React.FC<ByTemplateTabProps> = ({
  selectedSns,
  onSelectionChange,
}) => {
  const [selectedTemplateId, setSelectedTemplateId] = React.useState<string | null>(null);

  const handleApply = (template: DeviceTemplate) => {
    setSelectedTemplateId(template.id);
    // In production, this would trigger an API call to fetch SNs for the template
    // For now merge with existing selection using template's sns array
    const merged = [...new Set([...selectedSns, ...template.sns])];
    onSelectionChange(merged);
  };

  const columns: TableProps<DeviceTemplate>['columns'] = [
    {
      title: '模板名称',
      dataIndex: 'name',
      key: 'name',
      render: (v: string, record) => (
        <span>
          <Typography.Text strong>{v}</Typography.Text>
          {selectedTemplateId === record.id && (
            <Tag color="blue" style={{ marginLeft: 8 }}>
              已选
            </Tag>
          )}
        </span>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '设备类型',
      dataIndex: 'deviceType',
      key: 'deviceType',
      width: 100,
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record) => (
        <Button
          type="link"
          size="small"
          onClick={() => handleApply(record)}
        >
          应用
        </Button>
      ),
    },
  ];

  return (
    <div>
      <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 12, fontSize: 13 }}>
        选择预设模板快速添加设备。应用后可在「已选设备」中查看和调整。
      </Typography.Text>
      <Table<DeviceTemplate>
        rowKey="id"
        columns={columns}
        dataSource={PRESET_TEMPLATES}
        size="small"
        pagination={false}
        onRow={(record) => ({
          style: {
            background: selectedTemplateId === record.id ? '#e6f7ff' : undefined,
            cursor: 'pointer',
          },
          onClick: () => handleApply(record),
        })}
      />
    </div>
  );
};

export default ByTemplateTab;
