import { useState } from 'react';
import { Modal, Table, Input, Select, Tag, Space, Empty } from 'antd';
import { useUnknownAlarmStats } from '@core/hooks/api/useAlarmDefinitions';
import type { UnknownAlarmStat } from '@core/types/alarmDefinition';

interface Props {
  open: boolean;
  onClose: () => void;
}

const DAYS_OPTIONS = [
  { label: '近 1 天', value: 1 },
  { label: '近 7 天', value: 7 },
  { label: '近 30 天', value: 30 },
];

export default function UnknownStatsModal({ open, onClose }: Props) {
  const [productId, setProductId] = useState('');
  const [days, setDays] = useState(7);
  const { data, isLoading } = useUnknownAlarmStats({
    productId: productId.trim() || undefined,
    days,
  });

  const items = data?.items || [];

  const columns = [
    { title: '产品 ID', dataIndex: 'productId', width: 220, ellipsis: true, render: (v?: string) => v || '—' },
    { title: '产品名', dataIndex: 'productName', width: 200, render: (v?: string) => v || '—' },
    {
      title: '告警标识 (identifier)',
      dataIndex: 'identifier',
      width: 180,
      render: (v: string) => <Tag color="orange">{v}</Tag>,
    },
    { title: '出现次数', dataIndex: 'count', width: 100 },
    { title: '最后一次', dataIndex: 'lastSeenAt', render: (v?: string) => v || '—' },
  ];

  return (
    <Modal
      title={t('product.alarm.unknown.title')}
      open={open}
      onCancel={onClose}
      footer={null}
      width={920}
      destroyOnHidden
    >
      <Space style={{ marginBottom: 12 }}>
        <Input
          placeholder={t('product.alarm.unknown.filterPh')}
          allowClear
          value={productId}
          onChange={(e) => setProductId(e.target.value)}
          style={{ width: 320 }}
        />
        <Select value={days} onChange={(v) => setDays(v)} options={DAYS_OPTIONS} style={{ width: 120 }} />
      </Space>
      {items.length === 0 && !isLoading ? (
        <Empty description={t('product.alarm.unknown.empty')} />
      ) : (
        <Table<UnknownAlarmStat>
          rowKey={(r) => `${r.productId || 'all'}-${r.identifier}`}
          loading={isLoading}
          columns={columns}
          dataSource={items}
          size="small"
          pagination={{ pageSize: 20 }}
        />
      )}
    </Modal>
  );
}
