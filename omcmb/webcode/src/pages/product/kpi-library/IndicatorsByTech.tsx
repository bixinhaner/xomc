/**
 * IndicatorsByTech — T-0180 二级 drill-down 详情页面(2026-05-29 用户调整)。
 *
 * 本组件本身不渲染 filter / toolbar — index.tsx 顶部 Card 统一承载
 * "返回 + 制式名 + 管理 XML 文件 + 分组筛选 + 平台筛选 + 关键字搜索"。
 * 本组件仅作受控 Table:接受 keyword/platform/groupId props,过滤变化时
 * 自动 reset pagination page=1。
 *
 * 列调整(用户决策):
 *   - 新增"分组"列(groupName,与分组筛选字段对齐)
 *   - 新增"平台"列(productClass,与平台筛选字段对齐)
 *   - "详情"列改名为"操作",EyeOutlined → EditOutlined,
 *     表明 IndicatorDrawer 内含公式 CRUD 支持修改
 */
import { useEffect, useMemo, useState } from 'react';
import { Card, Table, Switch, Button, Space, Tag, message } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import {
  useIndicatorList,
  useEnabledIndicators,
  useSetEnabledIndicators,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import IndicatorDrawer from './IndicatorDrawer';

interface Filter {
  keyword?: string;
  platform?: string;  // 2026-05-29:由 URL ?platform= 注入并锁定;不在 UI 提供切换
  groupId?: string;
}

interface Props {
  deviceType: DeviceType;
  filter: Filter;
}

// 启用状态走 default 行(XML 真相源)
const OPERATOR_CODE = 'default';

export default function IndicatorsByTech({ deviceType, filter }: Props) {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);

  // filter 任意变化即 reset page=1(避免分页 + 过滤错位返空)
  useEffect(() => {
    setPage(1);
  }, [filter.keyword, filter.platform, filter.groupId, deviceType]);

  const { data, isLoading } = useIndicatorList(deviceType, {
    keyword: filter.keyword || undefined,
    platformName: filter.platform || undefined,
    groupId: filter.groupId || undefined,
    page,
    pageSize,
  });

  const { data: enabledData } = useEnabledIndicators(deviceType, OPERATOR_CODE);
  const setMut = useSetEnabledIndicators();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selected, setSelected] = useState<IndicatorInfo | null>(null);
  const [pendingId, setPendingId] = useState<string | null>(null);

  const items = useMemo(() => data?.items || [], [data]);
  const enabledSet = useMemo(() => new Set(enabledData?.items || []), [enabledData]);

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 140 },
    { title: '中文名', dataIndex: 'cnName', width: 180 },
    { title: '英文名', dataIndex: 'enName', width: 220, ellipsis: true },
    {
      title: '分组',
      dataIndex: 'groupName',
      width: 140,
      render: (v: string, row: IndicatorInfo) =>
        v || row.groupId ? <Tag color="purple">{v || row.groupId}</Tag> : <span>—</span>,
    },
    {
      title: '平台',
      dataIndex: 'productClass',
      width: 130,
      render: (v: string) => (v ? <Tag color="cyan">{v}</Tag> : <span>—</span>),
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
      width: 80,
      render: (_: unknown, row: IndicatorInfo) => (
        <Button
          size="small"
          icon={<EditOutlined />}
          onClick={() => {
            setSelected(row);
            setDrawerOpen(true);
          }}
        />
      ),
    },
  ];

  return (
    <Card size="small">
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

// 删除原 Props.deviceType 的 platform/keyword 内嵌过滤,改为受控
export type { Filter as IndicatorsByTechFilter };
