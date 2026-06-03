import { useMemo, useState } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Tag,
  Popconfirm,
  message,
  Tooltip,
  Typography,
} from 'antd';
import { makeSeqColumn } from '@/components/Table/seqColumn';
import SearchInput from '@/components/SearchInput';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ClearOutlined,
  CloudDownloadOutlined,
} from '@ant-design/icons';
import {
  useProductList,
  useDeleteProduct,
  useResetDiscovered,
  useProductCacheRefresh,
  useProductImportDirectory,
  useMatchOrder,
} from '@core/hooks/api/useProducts';
import type { Product, ProductListFilter } from '@core/types/product';
import ProductDrawer from './ProductDrawer';
import MatchTester from './MatchTester';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

interface PatternStats {
  minSortOrder: number;
  total: number;
  active: number;
}

const SORT_TAIL = Number.MAX_SAFE_INTEGER;

export default function ProductsPage() {
  const t = useT();
  const [filter, setFilter] = useState<ProductListFilter>({});
  const [keyword, setKeyword] = useState('');
  const { data, isLoading, refetch } = useProductList(filter);
  const { data: matchOrderData } = useMatchOrder();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);

  const delMut = useDeleteProduct();
  const resetDiscMut = useResetDiscovered();
  const cacheRefMut = useProductCacheRefresh();
  const importMut = useProductImportDirectory();

  const items = useMemo(() => data?.items || [], [data]);

  const patternStats = useMemo(() => {
    const stats = new Map<string, PatternStats>();
    for (const row of matchOrderData?.items || []) {
      const cur = stats.get(row.productId);
      if (!cur) {
        stats.set(row.productId, {
          minSortOrder: row.sortOrder,
          total: 1,
          active: row.isActive ? 1 : 0,
        });
      } else {
        cur.minSortOrder = Math.min(cur.minSortOrder, row.sortOrder);
        cur.total += 1;
        if (row.isActive) cur.active += 1;
      }
    }
    return stats;
  }, [matchOrderData]);

  const sortedItems = useMemo(() => {
    return [...items].sort((a, b) => {
      const sa = patternStats.get(a.id)?.minSortOrder ?? SORT_TAIL;
      const sb = patternStats.get(b.id)?.minSortOrder ?? SORT_TAIL;
      return sa - sb;
    });
  }, [items, patternStats]);

  const columns = [
    {
      title: t('product.products.col.productName'),
      dataIndex: 'name',
      width: 220,
      render: (v: string) => <Text strong>{v}</Text>,
    },
    {
      title: t('common.vendor'),
      dataIndex: 'vendor',
      width: 120,
    },
    {
      title: t('product.products.tech'),
      dataIndex: 'tech',
      width: 90,
      render: (v: string) => <Tag>{v.toUpperCase()}</Tag>,
    },
    {
      title: t('product.products.radioModes'),
      dataIndex: 'radioModes',
      width: 110,
      render: (v?: string) => (v ? <Tag color="cyan">{v}</Tag> : <Text type="secondary">—</Text>),
    },
    {
      title: t('product.products.col.indicatorPlatform'),
      dataIndex: 'indicatorPlatform',
      width: 140,
    },
    {
      title: t('product.products.col.alarmNeType'),
      dataIndex: 'alarmNeType',
      width: 130,
    },
    {
      title: t('product.products.col.regex'),
      dataIndex: 'patterns',
      width: 280,
      render: (list?: string[]) => {
        const items = list || [];
        if (items.length === 0) {
          return <Text type="secondary">—</Text>;
        }
        const visible = items.slice(0, 3);
        const rest = items.slice(3);
        return (
          <Space size={[0, 4]} wrap>
            {visible.map((p) => (
              <Tag key={p} color="geekblue">
                {p}
              </Tag>
            ))}
            {rest.length > 0 && (
              <Tooltip title={rest.join('\n')} overlayStyle={{ whiteSpace: 'pre' }}>
                <Tag color="default">+{rest.length}</Tag>
              </Tooltip>
            )}
          </Space>
        );
      },
    },
    {
      title: t('product.products.col.unknownAlarm'),
      dataIndex: 'enableUnknownAlarm',
      width: 90,
      render: (v: boolean) => (v ? <Tag color="warning">{t('common.accept')}</Tag> : <Tag>{t('common.discard')}</Tag>),
    },
    {
      title: t('common.action'),
      width: 120,
      render: (_: unknown, row: Product) => (
        <Space>
          <Tooltip title={t('common.edit')}>
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => {
                setEditingProduct(row);
                setDrawerOpen(true);
              }}
            />
          </Tooltip>
          <Popconfirm
            title={t('product.products.confirmReset', { name: row.name })}
            onConfirm={() =>
              resetDiscMut
                .mutateAsync(row.id)
                .then((r) => message.success(t('product.products.resetSuccess', { count: r.deletedRows })))
                .catch((e) => message.error((e as Error).message))
            }
          >
            <Tooltip title={t('product.products.resetBtn')}>
              <Button size="small" icon={<ClearOutlined />} />
            </Tooltip>
          </Popconfirm>
          <Popconfirm
            title={t('product.products.confirmDelete', { name: row.name })}
            onConfirm={() =>
              delMut
                .mutateAsync(row.id)
                .then(() => message.success(t('common.deleted')))
                .catch((e) => message.error((e as Error).message))
            }
          >
            <Tooltip title={t('common.delete')}>
              <Button size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 16 }}>
      <Card
        size="small"
        style={{ marginBottom: 12 }}
        styles={{ body: { padding: '8px 12px' } }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 12,
            flexWrap: 'wrap',
          }}
        >
          <Space size={8} wrap>
            <SearchInput
              placeholder={t('product.products.searchPh')}
              allowClear
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onSearch={(v) => setFilter((f) => ({ ...f, keyword: v || undefined }))}
              style={{ width: 320 }}
              enterButton
            />
            <MatchTester />
          </Space>
          <Space size={8}>
            <Popconfirm
              title={t('product.products.reloadXmlTitle')}
              description={
                <div style={{ maxWidth: 320 }}>
                  {t('product.products.reloadHint1Pre')}<code>param-mappings/products.xml</code>{t('product.products.reloadHint1Post')}
                  <br />
                  {t('product.products.reloadDesc')}
                </div>
              }
              okText={t('product.products.reloadOk')}
              cancelText={t('common.cancel')}
              okButtonProps={{ danger: true }}
              placement="bottomRight"
              onConfirm={() => {
                // 不返回 Promise — 让 Popconfirm 立即关闭；loading 反馈交给触发按钮
                // 重载 XML 后主动刷新缓存 + 重拉列表(取消独立"刷新缓存"按钮,合并到此处)
                importMut
                  .mutateAsync()
                  .then(async (r) => {
                    try {
                      await cacheRefMut.mutateAsync();
                    } catch (e) {
                      message.warning((e as Error).message);
                    }
                    void refetch();
                    message.success(t('product.products.reloadSuccess', { count: r.reloaded }));
                  })
                  .catch((e) => message.error((e as Error).message));
              }}
            >
              <Button
                icon={<CloudDownloadOutlined />}
                loading={importMut.isPending || cacheRefMut.isPending}
                danger
              >
                {t('common.reloadXml')}
              </Button>
            </Popconfirm>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditingProduct(null);
                setDrawerOpen(true);
              }}
            >
              {t('product.products.createBtn')}
            </Button>
          </Space>
        </div>
      </Card>

      <Card size="small">
        <Table<Product>
          rowKey="id"
          loading={isLoading}
          columns={[makeSeqColumn<Product>({ title: t('table.rowNumber'), dataSource: sortedItems }), ...columns]}
          dataSource={sortedItems}
          pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (total) => t('common.totalCount', { count: total }) }}
          size="small"
        />
      </Card>

      <ProductDrawer
        open={drawerOpen}
        product={editingProduct}
        onClose={() => {
          setDrawerOpen(false);
          setEditingProduct(null);
        }}
      />
    </div>
  );
}
