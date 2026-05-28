import { useMemo, useState } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Tag,
  Popconfirm,
  message,
  Tooltip,
  Typography,
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
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

const { Text } = Typography;

interface PatternStats {
  minSortOrder: number;
  total: number;
  active: number;
}

const SORT_TAIL = Number.MAX_SAFE_INTEGER;

export default function ProductsPage() {
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
      title: '序号',
      width: 80,
      align: 'center' as const,
      render: (_: unknown, row: Product) => {
        const stats = patternStats.get(row.id);
        if (!stats) {
          return <Text type="secondary">—</Text>;
        }
        return <Tag color="blue">{stats.minSortOrder}</Tag>;
      },
    },
    {
      title: '产品名',
      dataIndex: 'name',
      width: 220,
      render: (v: string) => <Text strong>{v}</Text>,
    },
    {
      title: '厂商',
      dataIndex: 'vendor',
      width: 120,
    },
    {
      title: '制式',
      dataIndex: 'tech',
      width: 80,
      render: (v: string) => <Tag>{v.toUpperCase()}</Tag>,
    },
    {
      title: '指标设备类型',
      dataIndex: 'indicatorDeviceType',
      width: 130,
      render: (v: string) => <Tag color="cyan">{v?.toUpperCase()}</Tag>,
    },
    {
      title: '指标平台',
      dataIndex: 'indicatorPlatform',
      width: 140,
    },
    {
      title: '告警网元类型',
      dataIndex: 'alarmNeType',
      width: 130,
    },
    {
      title: '正则规则',
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
      title: '激活',
      width: 110,
      render: (_: unknown, row: Product) => {
        const stats = patternStats.get(row.id);
        if (!stats || stats.total === 0) {
          return <Tag>无规则</Tag>;
        }
        if (stats.active === stats.total) {
          return <Tag color="success">激活</Tag>;
        }
        if (stats.active === 0) {
          return <Tag>禁用</Tag>;
        }
        return (
          <Tag color="warning">
            部分 {stats.active}/{stats.total}
          </Tag>
        );
      },
    },
    {
      title: '上传',
      dataIndex: 'enableFiletype11',
      width: 70,
      render: (v: boolean) => (v ? <Tag color="success">开</Tag> : <Tag>关</Tag>),
    },
    {
      title: '未知告警',
      dataIndex: 'enableUnknownAlarm',
      width: 90,
      render: (v: boolean) => (v ? <Tag color="warning">接纳</Tag> : <Tag>丢弃</Tag>),
    },
    {
      title: '设备数',
      dataIndex: 'deviceCount',
      width: 80,
      render: (v: number) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: '操作',
      width: 220,
      render: (_: unknown, row: Product) => (
        <Space>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingProduct(row);
              setDrawerOpen(true);
            }}
          >
            编辑
          </Button>
          <Tooltip title="清空所有 swVersion 的 discovered 映射；下次 Bootstrap 重新建">
            <Popconfirm
              title={`确认重置产品「${row.name}」的发现映射？`}
              onConfirm={() =>
                resetDiscMut
                  .mutateAsync(row.id)
                  .then((r) => message.success(`已重置 ${r.deletedRows} 条`))
                  .catch((e) => message.error((e as Error).message))
              }
            >
              <Button size="small" icon={<ClearOutlined />}>
                重置发现
              </Button>
            </Popconfirm>
          </Tooltip>
          <Popconfirm
            title={`确认删除产品「${row.name}」？关联设备需先解绑`}
            onConfirm={() =>
              delMut
                .mutateAsync(row.id)
                .then(() => message.success('已删除'))
                .catch((e) => message.error((e as Error).message))
            }
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
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
            <Input.Search
              placeholder="搜索：产品名 / 厂商 / 描述"
              allowClear
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onSearch={(v) => setFilter((f) => ({ ...f, keyword: v || undefined }))}
              style={{ width: 240 }}
            />
            <MatchTester />
          </Space>
          <Space size={8}>
            <Popconfirm
              title="确认重载 XML？"
              description={
                <div style={{ maxWidth: 320 }}>
                  将从 <code>datamodels/param-mappings/products.xml</code> 重新装配所有产品。
                  <br />
                  UI 中对 <b>指标平台 / 告警网元类型 / 正则规则</b> 等字段的手工修改将被 XML 值覆盖。
                </div>
              }
              okText="确认重载"
              cancelText="取消"
              okButtonProps={{ danger: true }}
              placement="bottomRight"
              onConfirm={() => {
                // 不返回 Promise — 让 Popconfirm 立即关闭；loading 反馈交给触发按钮
                importMut
                  .mutateAsync()
                  .then((r) => message.success(`已重载：${r.reloaded}`))
                  .catch((e) => message.error((e as Error).message));
              }}
            >
              <Button icon={<CloudDownloadOutlined />} loading={importMut.isPending} danger>
                重载 XML
              </Button>
            </Popconfirm>
            <Button
              icon={<ReloadOutlined />}
              loading={cacheRefMut.isPending}
              onClick={() => {
                cacheRefMut
                  .mutateAsync()
                  .then(() => {
                    void refetch();
                    message.success('已刷新缓存');
                  })
                  .catch((e) => message.error((e as Error).message));
              }}
            >
              刷新缓存
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditingProduct(null);
                setDrawerOpen(true);
              }}
            >
              新增产品
            </Button>
          </Space>
        </div>
      </Card>

      <Card size="small">
        <Table<Product>
          rowKey="id"
          loading={isLoading}
          columns={columns}
          dataSource={sortedItems}
          pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
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
