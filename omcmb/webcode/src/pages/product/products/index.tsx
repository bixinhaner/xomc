import { useMemo, useState } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
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
  AppstoreOutlined,
  ClearOutlined,
  CloudDownloadOutlined,
} from '@ant-design/icons';
import {
  useProductList,
  useDeleteProduct,
  useResetDiscovered,
  useProductCacheRefresh,
  useProductImportDirectory,
} from '@core/hooks/api/useProducts';
import type { Product, ProductListFilter } from '@core/types/product';
import ProductDrawer from './ProductDrawer';
import MatchOrderDrawer from './MatchOrderDrawer';
import MatchTester from './MatchTester';

const { Text } = Typography;

const TECH_OPTIONS = [
  { label: '全部', value: '' },
  { label: 'LTE', value: 'lte' },
  { label: 'NR', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
];

export default function ProductsPage() {
  const [filter, setFilter] = useState<ProductListFilter>({});
  const [keyword, setKeyword] = useState('');
  const { data, isLoading, refetch } = useProductList(filter);

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);
  const [matchOrderOpen, setMatchOrderOpen] = useState(false);

  const delMut = useDeleteProduct();
  const resetDiscMut = useResetDiscovered();
  const cacheRefMut = useProductCacheRefresh();
  const importMut = useProductImportDirectory();

  const items = useMemo(() => data?.items || [], [data]);

  const columns = [
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
      render: (v: string) => <Tag color="cyan">{v}</Tag>,
    },
    {
      title: '指标平台',
      dataIndex: 'indicatorPlatform',
      width: 140,
    },
    {
      title: '告警网元类型',
      dataIndex: 'alarmNeType',
      width: 120,
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
        title="产品管理 / Products"
        extra={
          <Space>
            <Input.Search
              placeholder="搜索：产品名 / 厂商 / 描述"
              allowClear
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onSearch={(v) => setFilter((f) => ({ ...f, keyword: v || undefined }))}
              style={{ width: 240 }}
            />
            <Select
              placeholder="制式"
              allowClear
              options={TECH_OPTIONS}
              value={filter.tech || ''}
              onChange={(v) => setFilter((f) => ({ ...f, tech: v || undefined }))}
              style={{ width: 100 }}
            />
            <Button icon={<AppstoreOutlined />} onClick={() => setMatchOrderOpen(true)}>
              全局匹配顺序
            </Button>
            <Tooltip title="POST /products/import-directory；从后端 datamodels/ 重新加载产品 XML">
              <Button
                icon={<CloudDownloadOutlined />}
                loading={importMut.isPending}
                onClick={() =>
                  importMut
                    .mutateAsync()
                    .then((r) => message.success(`已重载：${r.reloaded}`))
                    .catch((e) => message.error((e as Error).message))
                }
              >
                重载 XML
              </Button>
            </Tooltip>
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
        }
      />

      <MatchTester />

      <Card size="small">
        <Table<Product>
          rowKey="id"
          loading={isLoading}
          columns={columns}
          dataSource={items}
          pagination={{ pageSize: 20, showSizeChanger: true }}
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
      <MatchOrderDrawer open={matchOrderOpen} onClose={() => setMatchOrderOpen(false)} />
    </div>
  );
}
