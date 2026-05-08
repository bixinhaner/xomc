import { useMemo, useState } from 'react';
import { Card, Table, Input, Tag, Button, Space, message, Popconfirm } from 'antd';
import { EyeOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  useIndicatorList,
  useDeleteIndicator,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import IndicatorDrawer from './IndicatorDrawer';

interface Props {
  deviceType: DeviceType;
}

export default function IndicatorTab({ deviceType }: Props) {
  const [keyword, setKeyword] = useState('');
  const { data, isLoading } = useIndicatorList(deviceType, { keyword: keyword || undefined });
  const deleteMut = useDeleteIndicator();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selected, setSelected] = useState<IndicatorInfo | null>(null);

  const items = useMemo(() => data?.items || [], [data]);

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 140 },
    { title: '中文名', dataIndex: 'cnName', width: 200 },
    { title: '英文名', dataIndex: 'enName', width: 240, ellipsis: true },
    {
      title: '分组',
      dataIndex: 'groupName',
      width: 140,
      render: (v: string, row: IndicatorInfo) => v || row.groupId || '—',
    },
    { title: '计数器类型', dataIndex: 'counterType', width: 110 },
    ...(deviceType === 'GNB'
      ? []
      : [{ title: '级别', dataIndex: 'indicatorLevel', width: 90 }]),
    { title: '单位', dataIndex: 'unit', width: 80 },
    {
      title: '启用',
      dataIndex: 'isEnabled',
      width: 80,
      render: (v: boolean) => (v ? <Tag color="success">是</Tag> : <Tag>否</Tag>),
    },
    {
      title: '操作',
      width: 140,
      render: (_: unknown, row: IndicatorInfo) => (
        <Space>
          <Button
            size="small"
            icon={<EyeOutlined />}
            onClick={() => {
              setSelected(row);
              setDrawerOpen(true);
            }}
          >
            详情
          </Button>
          <Popconfirm
            title={`确认删除指标「${row.id}」？关联公式会一并删除`}
            onConfirm={() =>
              deleteMut
                .mutateAsync({ deviceType, id: row.id })
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
    <Card
      size="small"
      title={`${deviceType} 指标列表`}
      extra={
        <Input.Search
          placeholder="搜索 ID / 名称"
          allowClear
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          style={{ width: 260 }}
        />
      }
    >
      <Table<IndicatorInfo>
        rowKey="id"
        loading={isLoading}
        columns={columns}
        dataSource={items}
        size="small"
        pagination={{ pageSize: 50, showSizeChanger: true }}
      />
      <IndicatorDrawer
        open={drawerOpen}
        deviceType={deviceType}
        indicator={selected}
        onClose={() => {
          setDrawerOpen(false);
          setSelected(null);
        }}
      />
    </Card>
  );
}
