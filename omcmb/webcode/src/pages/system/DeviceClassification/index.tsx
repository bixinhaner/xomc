import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Tree, Tag, Space, Input } from 'antd';
import { EyeOutlined, SearchOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import { useProductClasses, useDeviceList } from '@core/hooks/api/useDevices';
import type { Device } from '@core/types/device';

const PAGE_SIZE = 20;
const ALL_KEY = '__all__';

export default function DeviceClassification() {
  const t = useT();
  const navigate = useNavigate();
  const [selectedClass, setSelectedClass] = useState<string>(ALL_KEY);
  const [searchText, setSearchText] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PAGE_SIZE);

  const { data: classes, isLoading: classesLoading } = useProductClasses();
  const productClasses = useMemo(() => classes ?? [], [classes]);

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(selectedClass !== ALL_KEY ? { productModel: selectedClass } : {}),
      ...(searchText.trim() ? { searchText: searchText.trim() } : {}),
    }),
    [page, pageSize, selectedClass, searchText]
  );
  const { data: list, isLoading: listLoading } = useDeviceList(params);

  const rows = list?.items ?? [];
  const total = list?.total ?? 0;

  // 产品类型树：全部设备 + 每个 product class 一个叶子节点（来自真实 /products 注册表）。
  const classificationTree: DataNode[] = useMemo(
    () => [
      {
        key: ALL_KEY,
        title: t('device.allDevices'),
        children: productClasses.map((pc) => ({ key: pc, title: pc })),
      },
    ],
    [productClasses, t]
  );

  const columns: DataTableColumn<Device & Record<string, unknown>>[] = useMemo(
    () => [
      { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 160, mono: true, copyable: true },
      {
        key: 'name',
        title: t('device.name'),
        dataIndex: 'name',
        ellipsis: true,
        render: (val, record) => (val as string) || record.sn || '-',
      },
      { key: 'vendor', title: t('table.vendor'), dataIndex: 'vendor', width: 110, render: (val) => (val as string) || '-' },
      {
        key: 'productClass',
        title: t('device.productClass'),
        dataIndex: 'productClass',
        width: 130,
        render: (val) => (val as string) || '-',
      },
      {
        key: 'networkType',
        title: t('table.type'),
        dataIndex: 'networkType',
        width: 90,
        render: (val) => ((val as string) || '-').toUpperCase(),
      },
      {
        key: 'isOnline',
        title: t('device.connStatus'),
        dataIndex: 'isOnline',
        width: 100,
        render: (val) =>
          (val as boolean) ? (
            <Tag color="green">{t('status.online')}</Tag>
          ) : (
            <Tag color="red">{t('status.offline')}</Tag>
          ),
      },
      {
        key: 'softwareVersion',
        title: t('device.softwareVersion'),
        dataIndex: 'softwareVersion',
        width: 200,
        ellipsis: true,
        render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{(val as string) || '-'}</span>,
      },
      {
        key: 'lastOnlineTime',
        title: t('device.lastOnlineTime'),
        dataIndex: 'lastOnlineTime',
        width: 180,
        render: (val) => (val as string) || '-',
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 100,
        fixed: 'right',
        render: (_, record) => (
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => navigate(`/device/list?sn=${encodeURIComponent(record.sn)}`)}
          >
            {t('common.view')}
          </Button>
        ),
      },
    ],
    [t, navigate]
  );

  const treePanel = (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '12px 8px', borderBottom: '1px solid #f0f0f0', display: 'flex', gap: 6, alignItems: 'center' }}>
        <span style={{ fontWeight: 500, fontSize: 14 }}>{t('nav.system.deviceClass')}</span>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: 8 }}>
        <Tree
          treeData={classificationTree}
          defaultExpandAll
          selectedKeys={[selectedClass]}
          onSelect={(keys) => {
            if (keys.length > 0) {
              setSelectedClass(String(keys[0]));
              setPage(1);
            }
          }}
        />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: '#fafafa' }}>
          <span style={{ fontWeight: 500 }}>
            {selectedClass === ALL_KEY ? t('device.allDevices') : selectedClass} ({total})
          </span>
          <Space>
            <Input
              placeholder={t('common.search')}
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => {
                setSearchText(e.target.value);
                setPage(1);
              }}
              style={{ width: 220 }}
              size="small"
              allowClear
            />
          </Space>
        </div>
        <div style={{ flex: 1, overflow: 'hidden' }}>
          <DataTable<Device & Record<string, unknown>>
            tableId="device-classification-list"
            columns={columns}
            dataSource={rows as (Device & Record<string, unknown>)[]}
            loading={classesLoading || listLoading}
            rowKey="id"
            total={total}
            currentPage={page}
            pageSize={pageSize}
            onPageChange={(p, s) => {
              setPage(p);
              setPageSize(s);
            }}
            scroll={{ x: 'max-content', y: 'calc(100vh - 320px)' }}
          />
        </div>
      </div>
    </TreeListPageLayout>
  );
}
