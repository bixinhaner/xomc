/**
 * IndicatorsByTech — T-0180 P4 二级 drill-down 详情页面。
 *
 * 行为基础来自原 IndicatorTab.tsx(T-0098 P3-03);本组件保留全部 CRUD/启用/平台过滤逻辑,
 * 删除"行级"操作改在 XMLFilesModal 走"按文件删"粒度(对齐 PRD §5 非目标 #3)。
 *
 * 顶部 Card 标题去除(由 index.tsx 顶部 toolbar 承担"返回 + 制式名"),
 * extra 保留平台筛选 + 搜索框。
 */
import { useMemo, useState } from 'react';
import { Card, Table, Input, Switch, Button, Space, Select, message } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import {
  useIndicatorList,
  useEnabledIndicators,
  useSetEnabledIndicators,
  usePlatformList,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import IndicatorDrawer from './IndicatorDrawer';

interface Props {
  deviceType: DeviceType;
}

// 启用状态走 default 行(XML 真相源)
const OPERATOR_CODE = 'default';

export default function IndicatorsByTech({ deviceType }: Props) {
  const [keyword, setKeyword] = useState('');
  const [platform, setPlatform] = useState<string | undefined>(undefined);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);

  const { data: platformData } = usePlatformList(deviceType);
  const platformOptions = useMemo(
    () => (platformData?.items || []).map((p) => ({ label: p, value: p })),
    [platformData]
  );
  const showPlatformFilter = platformOptions.length > 1;

  const { data, isLoading } = useIndicatorList(deviceType, {
    keyword: keyword || undefined,
    platformName: platform,
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
      title: '详情',
      width: 80,
      render: (_: unknown, row: IndicatorInfo) => (
        <Button
          size="small"
          icon={<EyeOutlined />}
          onClick={() => {
            setSelected(row);
            setDrawerOpen(true);
          }}
        />
      ),
    },
  ];

  return (
    <Card
      size="small"
      extra={
        <Space>
          {showPlatformFilter && (
            <Select
              placeholder="按平台筛选"
              allowClear
              value={platform}
              onChange={(v) => {
                setPlatform(v);
                setPage(1);
              }}
              options={platformOptions}
              style={{ width: 200 }}
            />
          )}
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
        </Space>
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
