import { useState } from 'react';
import { Modal, Table, Input, Select, Tag, Space, Empty } from 'antd';
import { useUnknownAlarmStats } from '@core/hooks/api/useAlarmDefinitions';
import type { UnknownAlarmStat } from '@core/types/alarmDefinition';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  onClose: () => void;
}

export default function UnknownStatsModal({ open, onClose }: Props) {
  const t = useT();
  const DAYS_OPTIONS = [
    { label: t('product.alarm.unknown.day1'), value: 1 },
    { label: t('product.alarm.unknown.day7'), value: 7 },
    { label: t('product.alarm.unknown.day30'), value: 30 },
  ];
  const [productId, setProductId] = useState('');
  const [days, setDays] = useState(7);
  const { data, isLoading } = useUnknownAlarmStats({
    productId: productId.trim() || undefined,
    days,
  });

  const items = data?.items || [];

  const columns = [
    { title: t('product.alarm.unknown.colProductId'), dataIndex: 'productId', width: 220, ellipsis: true, render: (v?: string) => v || '—' },
    { title: t('product.alarm.unknown.colProductName'), dataIndex: 'productName', width: 200, render: (v?: string) => v || '—' },
    {
      title: t('product.alarm.unknown.colIdentifier'),
      dataIndex: 'identifier',
      width: 180,
      render: (v: string) => <Tag color="orange">{v}</Tag>,
    },
    { title: t('product.alarm.unknown.colCount'), dataIndex: 'count', width: 100 },
    { title: t('product.alarm.unknown.colLastSeen'), dataIndex: 'lastSeenAt', render: (v?: string) => v || '—' },
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
