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
} from '@ant-design/icons';
import {
  useProductList,
  useDeleteProduct,
  useMatchOrder,
} from '@core/hooks/api/useProducts';
import { useSystemLicense } from '@core/hooks/api/useSystemLicense';
import type { Product, ProductListFilter } from '@core/types/product';
import { filterDeviceScopedItemsByLicense } from '@core/utils/licenseFeatures';
import ProductDrawer from './ProductDrawer';
import MatchTester from './MatchTester';
import { useT } from '@/hooks/useT';
import {
  PRODUCT_TABLE_DEFAULT_PAGE_SIZE,
  PRODUCT_TABLE_PAGE_SIZE_OPTIONS,
} from '../pagination';

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
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PRODUCT_TABLE_DEFAULT_PAGE_SIZE);
  const { data, isLoading } = useProductList(filter);
  const { data: matchOrderData } = useMatchOrder();
  const { data: systemLicense, isLoading: systemLicenseLoading } = useSystemLicense();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);

  const delMut = useDeleteProduct();

  const items = useMemo(() => {
    const source = data?.items || [];
    return filterDeviceScopedItemsByLicense(source, systemLicense, systemLicenseLoading);
  }, [data, systemLicense, systemLicenseLoading]);

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
      render: (v: string) => {
        const value = (v || '').trim();
        return value ? <Tag>{value.toUpperCase()}</Tag> : <Text type="secondary">—</Text>;
      },
    },
    {
      title: t('product.products.paramModel'),
      dataIndex: 'paramModelName',
      width: 160,
      ellipsis: true,
      render: (v?: string) => v || '—',
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
          {row.isBuiltin ? (
            <Tooltip title={t('product.products.builtinNoDelete')}>
              <Button size="small" danger icon={<DeleteOutlined />} disabled />
            </Tooltip>
          ) : (
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
          )}
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
              onSearch={(v) => {
                setFilter((f) => ({ ...f, keyword: v.trim() || undefined }));
                setPage(1);
              }}
              style={{ width: 320 }}
              enterButton
            />
            <MatchTester />
          </Space>
          <Space size={8}>
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
          pagination={{
            current: page,
            pageSize,
            total: sortedItems.length,
            showSizeChanger: true,
            pageSizeOptions: PRODUCT_TABLE_PAGE_SIZE_OPTIONS,
            showTotal: (n) => t('common.totalCount', { count: n }),
            onChange: (p, ps) => {
              setPage(p);
              if (ps !== pageSize) setPageSize(ps);
            },
          }}
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
