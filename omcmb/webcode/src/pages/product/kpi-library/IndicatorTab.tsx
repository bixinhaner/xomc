import { useMemo, useState } from 'react';
import { Card, Table, Input, Switch, Button, Space, message, Popconfirm } from 'antd';
import { EyeOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  useIndicatorList,
  useDeleteIndicator,
  useEnabledIndicators,
  useSetEnabledIndicators,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import IndicatorDrawer from './IndicatorDrawer';

interface Props {
  deviceType: DeviceType;
}

// 启用状态走 default 行（XML 真相源；运营商覆盖能力后端保留但 UI 不暴露选择器）
const OPERATOR_CODE = 'default';

export default function IndicatorTab({ deviceType }: Props) {
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const { data, isLoading } = useIndicatorList(deviceType, {
    keyword: keyword || undefined,
    page,
    pageSize,
  });
  const deleteMut = useDeleteIndicator();
  const { data: enabledData } = useEnabledIndicators(deviceType, OPERATOR_CODE);
  const setMut = useSetEnabledIndicators();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selected, setSelected] = useState<IndicatorInfo | null>(null);
  const [pendingId, setPendingId] = useState<string | null>(null);

  const items = useMemo(() => data?.items || [], [data]);
  const enabledSet = useMemo(() => new Set(enabledData?.items || []), [enabledData]);

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
      dataIndex: 'id',
      width: 80,
      render: (id: string) => (
        <Switch
          size="small"
          checked={enabledSet.has(id)}
          loading={setMut.isPending && pendingId === id}
          onChange={(checked) => {
            setPendingId(id);
            setMut.mutate(
              {
                deviceType,
                operatorCode: OPERATOR_CODE,
                indicatorIds: [id],
                enable: checked,
              },
              {
                onSettled: () => setPendingId(null),
                onError: (e) => message.error((e as Error).message),
              }
            );
          }}
        />
      ),
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
          onChange={(e) => {
            setKeyword(e.target.value);
            setPage(1);
          }}
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
        pagination={{
          current: page,
          pageSize,
          total: data?.total || 0,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
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
