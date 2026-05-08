import { useMemo, useState } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Tag,
  message,
  Tooltip,
  Popconfirm,
  Empty,
} from 'antd';
import {
  ReloadOutlined,
  ThunderboltOutlined,
  LinkOutlined,
} from '@ant-design/icons';
import { useOrphanDevices, useRematchOrphan } from '@core/hooks/api/useProducts';
import type { OrphanDevice } from '@core/types/product';
import BindProductModal from './BindProductModal';

export default function OrphanDevicesPage() {
  const [limit, setLimit] = useState(200);
  const { data, isLoading, refetch } = useOrphanDevices(limit);
  const rematchMut = useRematchOrphan();

  const [bindOpen, setBindOpen] = useState(false);
  const [bindTargets, setBindTargets] = useState<OrphanDevice[]>([]);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  const items = useMemo(() => data?.items || [], [data]);

  const columns = [
    {
      title: 'SN',
      dataIndex: 'serialNumber',
      width: 200,
    },
    { title: 'OUI', dataIndex: 'oui', width: 100 },
    {
      title: 'productClass',
      dataIndex: 'productClass',
      render: (v: string) => <Tag color="orange">{v || '—'}</Tag>,
    },
    { title: '运营商', dataIndex: 'carrier', width: 90 },
    { title: '厂商', dataIndex: 'manufacturer', width: 130 },
    {
      title: '最后 Inform',
      dataIndex: 'lastInformAt',
      width: 180,
      render: (v?: string) => v || '—',
    },
    {
      title: '操作',
      width: 120,
      render: (_: unknown, row: OrphanDevice) => (
        <Button
          size="small"
          icon={<LinkOutlined />}
          onClick={() => {
            setBindTargets([row]);
            setBindOpen(true);
          }}
        >
          绑定
        </Button>
      ),
    },
  ];

  const selectedRows = items.filter((d) => selectedKeys.includes(d.id));

  return (
    <div style={{ padding: 16 }}>
      <Card
        size="small"
        title={`孤儿设备 / Orphan Devices（${items.length}/${limit}）`}
        extra={
          <Space>
            <Tooltip title="后端 POST /products/orphan-devices/rematch；扫描所有 productClass 不空但 product_id 仍为 NULL 的设备，按现有 ProductRegistry 规则重新匹配">
              <Popconfirm
                title="确认触发全量重新匹配？"
                onConfirm={() =>
                  rematchMut
                    .mutateAsync()
                    .then((r) => message.success(`已重新绑定 ${r.rebound} 台${r.scanned ? ` / 扫描 ${r.scanned} 台` : ''}`))
                    .catch((e) => message.error((e as Error).message))
                }
              >
                <Button icon={<ThunderboltOutlined />} loading={rematchMut.isPending}>
                  全量重新匹配
                </Button>
              </Popconfirm>
            </Tooltip>
            <Button
              type="primary"
              icon={<LinkOutlined />}
              disabled={selectedRows.length === 0}
              onClick={() => {
                setBindTargets(selectedRows);
                setBindOpen(true);
              }}
            >
              批量绑定（{selectedRows.length}）
            </Button>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => {
                void refetch();
                setSelectedKeys([]);
              }}
            >
              刷新
            </Button>
          </Space>
        }
      >
        {items.length === 0 && !isLoading ? (
          <Empty description="无孤儿设备 — 所有设备 productClass 已被 ProductRegistry 命中" />
        ) : (
          <Table<OrphanDevice>
            rowKey="id"
            loading={isLoading}
            columns={columns}
            dataSource={items}
            size="small"
            pagination={{
              pageSize: 50,
              showSizeChanger: true,
              showTotal: (t) => `共 ${t} 台（受 limit=${limit} 限制）`,
              onShowSizeChange: (_p, size) => setLimit(size > 200 ? size : 200),
            }}
            rowSelection={{
              selectedRowKeys: selectedKeys,
              onChange: (keys) => setSelectedKeys(keys),
            }}
          />
        )}
      </Card>

      <BindProductModal
        open={bindOpen}
        devices={bindTargets}
        onClose={() => {
          setBindOpen(false);
          setBindTargets([]);
        }}
        onDone={() => {
          setSelectedKeys([]);
          void refetch();
        }}
      />
    </div>
  );
}
