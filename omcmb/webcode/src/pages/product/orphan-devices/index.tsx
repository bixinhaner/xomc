import { useMemo, useState } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Tag,
  Input,
  message,
  Tooltip,
  Popconfirm,
  Empty,
  Row,
  Col,
} from 'antd';
import { ThunderboltOutlined, LinkOutlined } from '@ant-design/icons';
import { useOrphanDevices, useRematchOrphan } from '@core/hooks/api/useProducts';
import type { OrphanDevice } from '@core/types/product';
import BindProductModal from './BindProductModal';

export default function OrphanDevicesPage() {
  // 2026-05-28 改造:
  //   1. server-side 分页 (10/20/50/1000)
  //   2. SN 模糊搜索
  //   3. 取消"刷新"按钮 — hook 已设 staleTime:0 + refetchOnMount,每次进页都
  //      重新拉数据;批量绑定 / rematch / SN 搜索后 invalidate 同样自动刷新
  const [page, setPage] = useState(1);
  // 2026-05-28 用户决策:默认 10 条/页(原 50)
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const { data, isLoading } = useOrphanDevices({ page, pageSize, search });
  const rematchMut = useRematchOrphan();

  const [bindOpen, setBindOpen] = useState(false);
  const [bindTargets, setBindTargets] = useState<OrphanDevice[]>([]);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  const items = useMemo(() => data?.items || [], [data]);
  const total = data?.total ?? 0;

  const columns = [
    { title: 'SN', dataIndex: 'serialNumber', width: 200 },
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
  const selectedCount = selectedRows.length;

  return (
    <div style={{ padding: 16 }}>
      <Card size="small">
        {/* 2026-05-28 用户决策:去掉"孤儿设备 共 N 台"标题,搜索框靠左;
            分页 footer 自带"共 N 台",信息不丢。 */}
        <Row align="middle" justify="space-between" style={{ marginBottom: 12 }} gutter={8}>
          <Col>
            <Input.Search
              placeholder="按 SN 模糊搜索"
              allowClear
              style={{ width: 240 }}
              onSearch={(v) => {
                setSearch(v.trim());
                setPage(1);
                setSelectedKeys([]);
              }}
              enterButton
            />
          </Col>
          <Col>
            <Space>
              {/* 2026-05-28 用户决策:rematch 改异步执行 + per-admin 排重。
                  - API 立即返 202,前端 toast"正在匹配中,请稍等"
                  - 同一管理员重复触发 → 后端返 409,前端 toast 提醒等待
                  - 真实绑定结果靠下一次刷新页面体现(staleTime:0 进页自动拉) */}
              <Tooltip title="扫描所有 product_class 不空,但 product_id 仍为 NULL 的设备">
                <Popconfirm
                  title="确认触发全量重新匹配?"
                  onConfirm={() =>
                    rematchMut
                      .mutateAsync()
                      .then((r) => {
                        // 后端业务码区分:
                        //   accepted → 锁获取成功,goroutine 已派发
                        //   running  → 已有 rematch 在执行中,提示用户等待
                        if (r.status === 'running') {
                          message.warning('正在执行中,请等待刷新完成');
                        } else {
                          message.info('正在后台执行,请稍等');
                        }
                      })
                      .catch((e) =>
                        message.error((e as Error).message || '触发失败'),
                      )
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
                disabled={selectedCount === 0}
                onClick={() => {
                  setBindTargets(selectedRows);
                  setBindOpen(true);
                }}
              >
                {selectedCount > 0 ? `批量绑定 ${selectedCount} 台` : '批量绑定'}
              </Button>
            </Space>
          </Col>
        </Row>
        {items.length === 0 && !isLoading ? (
          <Empty
            description={
              search
                ? `未找到 SN 包含 "${search}" 的孤儿设备`
                : '无孤儿设备 — 所有设备 productClass 已被 ProductRegistry 命中'
            }
          />
        ) : (
          <Table<OrphanDevice>
            rowKey="id"
            loading={isLoading}
            columns={columns}
            dataSource={items}
            size="small"
            pagination={{
              current: page,
              pageSize,
              total,
              showSizeChanger: true,
              pageSizeOptions: ['10', '20', '50', '1000'],
              showTotal: (t) => `共 ${t} 台`,
              onChange: (p, ps) => {
                setPage(p);
                if (ps !== pageSize) setPageSize(ps);
                setSelectedKeys([]);
              },
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
        }}
      />
    </div>
  );
}
