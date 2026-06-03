import { useMemo, useState } from 'react';
import { Card, Table, Tag, Input, Empty } from 'antd';
import { useOrphanDevices } from '@core/hooks/api/useProducts';
import type { OrphanDevice } from '@core/types/product';
import { makeSeqColumn } from '@/components/Table/seqColumn';
import { useT } from '@/hooks/useT';

export default function OrphanDevicesPage() {
  const t = useT();
  // 2026-06-02 用户决策:孤儿设备页改为纯只读列表 —— 去掉运营商/操作列、
  // 去掉"全量重新匹配"/"批量绑定"操作;搜索框模糊匹配 SN/OUI/产品类型/厂商。
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const { data, isLoading } = useOrphanDevices({ page, pageSize, search });

  const items = useMemo(() => data?.items || [], [data]);
  const total = data?.total ?? 0;

  const columns = [
    { title: 'SN', dataIndex: 'serialNumber', width: 200 },
    { title: 'OUI', dataIndex: 'oui', width: 100 },
    {
      title: t('device.productClass'),
      dataIndex: 'productClass',
      render: (v: string) => <Tag color="orange">{v || '—'}</Tag>,
    },
    { title: t('common.vendor'), dataIndex: 'manufacturer', width: 130 },
    {
      title: t('common.lastInform'),
      dataIndex: 'lastInformAt',
      width: 180,
      render: (v?: string) => v || '—',
    },
  ];

  return (
    <div style={{ padding: 16 }}>
      <Card size="small">
        <div style={{ marginBottom: 12 }}>
          <Input.Search
            placeholder={t('product.orphan.searchPh')}
            allowClear
            style={{ width: 320 }}
            onSearch={(v) => {
              setSearch(v.trim());
              setPage(1);
            }}
            enterButton
          />
        </div>
        {items.length === 0 && !isLoading ? (
          <Empty
            description={
              search
                ? t('product.orphan.notFoundSearch', { search })
                : t('product.orphan.emptyState')
            }
          />
        ) : (
          <Table<OrphanDevice>
            rowKey="id"
            loading={isLoading}
            columns={[makeSeqColumn<OrphanDevice>({ title: t('table.rowNumber'), current: page, pageSize }), ...columns]}
            dataSource={items}
            size="small"
            pagination={{
              current: page,
              pageSize,
              total,
              showSizeChanger: true,
              pageSizeOptions: ['10', '20', '50', '1000'],
              showTotal: (total) => t('product.orphan.totalDevices', { count: total }),
              onChange: (p, ps) => {
                setPage(p);
                if (ps !== pageSize) setPageSize(ps);
              },
            }}
          />
        )}
      </Card>
    </div>
  );
}
