import { Drawer, Table, Tag, Typography, Spin, Empty } from 'antd';
import { useMatchOrder } from '@core/hooks/api/useProducts';
import type { MatchOrderRow } from '@core/types/product';

const { Text } = Typography;

interface Props {
  open: boolean;
  onClose: () => void;
}

export default function MatchOrderDrawer({ open, onClose }: Props) {
  const { data, isLoading } = useMatchOrder();
  const items = data?.items || [];

  const columns = [
    {
      title: '全局序号',
      dataIndex: 'sortOrder',
      width: 100,
      render: (v: number) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: '产品名',
      dataIndex: 'productName',
      width: 200,
    },
    {
      title: '正则规则',
      dataIndex: 'productClass',
      render: (v: string) => <Text code>{v}</Text>,
    },
    {
      title: '激活',
      dataIndex: 'isActive',
      width: 80,
      render: (v: boolean) => (v ? <Tag color="success">是</Tag> : <Tag>否</Tag>),
    },
  ];

  return (
    <Drawer
      title="全局匹配顺序 / Match Order"
      placement="right"
      width={720}
      open={open}
      onClose={onClose}
      destroyOnHidden
    >
      <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
        所有产品的所有正则规则按 sort_order 排列；命中由小到大优先级。修改顺序请到产品详情的"正则模式"段使用上下移按钮。
      </Text>
      {isLoading ? (
        <Spin />
      ) : items.length === 0 ? (
        <Empty description="无规则" />
      ) : (
        <Table<MatchOrderRow>
          rowKey="patternId"
          columns={columns}
          dataSource={items}
          size="small"
          pagination={{ pageSize: 20, showSizeChanger: false }}
        />
      )}
    </Drawer>
  );
}
