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
import { Card, Table, Switch, Button, Tag, Tooltip, message } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import {
  useIndicatorList,
  useEnabledIndicators,
  useSetEnabledIndicators,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import IndicatorDrawer from './IndicatorDrawer';
import { enabledIndicatorErrorMessage } from './enabledIndicatorErrors';
import { makeSeqColumn } from '@/components/Table/seqColumn';
import { useT } from '@/hooks/useT';
import {
  PRODUCT_TABLE_DEFAULT_PAGE_SIZE,
  PRODUCT_TABLE_PAGE_SIZE_OPTIONS,
} from '../pagination';

interface Filter {
  keyword?: string;
  platform?: string;  // 2026-05-29:由 URL ?platform= 注入并锁定;不在 UI 提供切换
  groupId?: string;
  isEnabled?: boolean;
}

interface Props {
  deviceType: DeviceType;
  filter: Filter;
}

// 启用状态走 default 行(XML 真相源)
const OPERATOR_CODE = 'default';
const NAME_COLUMN_WIDTH = 240;

export default function IndicatorsByTech({ deviceType, filter }: Props) {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PRODUCT_TABLE_DEFAULT_PAGE_SIZE);

  // filter 任意变化即 reset page=1(避免分页 + 过滤错位返空)
  useEffect(() => {
    setPage(1);
  }, [filter.keyword, filter.platform, filter.groupId, filter.isEnabled, deviceType]);

  const { data, isLoading } = useIndicatorList(deviceType, {
    keyword: filter.keyword || undefined,
    platformName: filter.platform || undefined,
    groupId: filter.groupId || undefined,
    operatorCode: OPERATOR_CODE,
    isEnabled: filter.isEnabled,
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
    { title: 'ID', dataIndex: 'id', width: 150 },
    { title: t('common.cnName'), dataIndex: 'cnName', width: NAME_COLUMN_WIDTH, ellipsis: true },
    { title: t('common.enName'), dataIndex: 'enName', width: NAME_COLUMN_WIDTH, ellipsis: true },
    {
      title: t('common.group'),
      dataIndex: 'groupName',
      width: 150,
      render: (v: string, row: IndicatorInfo) =>
        v || row.groupId ? <Tag color="purple">{v || row.groupId}</Tag> : <span>—</span>,
    },
    // 2026-05-29 用户决策:删除"平台"列 — 详情态由 URL ?platform= 锁定,
    // 同一视图下所有行都属同一 platform,列冗余。
    // 2026-06-03 用户决策:删除"计数器类型 / 单位"列 — 数据常空,无展示价值。
    ...(deviceType === 'GNB'
      ? []
      : [{ title: t('common.level'), dataIndex: 'indicatorLevel', width: 100 }]),
    {
      title: t('common.enable'),
      dataIndex: 'id',
      width: 90,
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
                onError: (e) => message.error(enabledIndicatorErrorMessage(e, t)),
              }
            );
          }}
        />
      ),
    },
    {
      title: t('common.action'),
      width: 130,
      render: (_: unknown, row: IndicatorInfo) => (
        // 单一入口:详情抽屉(含基础信息编辑 + 全平台公式 CRUD)。Issue #535 收口,去掉第二个编辑按钮。
        <Tooltip title={t('common.detail')}>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setSelected(row);
              setDrawerOpen(true);
            }}
          />
        </Tooltip>
      ),
    },
  ];

  return (
    <Card size="small">
      <Table<IndicatorInfo>
        rowKey="id"
        loading={isLoading}
        columns={[makeSeqColumn<IndicatorInfo>({ title: t('table.rowNumber'), current: page, pageSize }), ...columns]}
        dataSource={items}
        size="small"
        pagination={{
          current: page,
          pageSize,
          total: data?.total || 0,
          showSizeChanger: true,
          pageSizeOptions: PRODUCT_TABLE_PAGE_SIZE_OPTIONS,
          showTotal: (n) => t('common.totalCount', { count: n }),
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
        // 启用状态唯一真值源:与列表开关同走 default 启用桶(enabledSet),
        // 避免用 indicator.isEnabled(列表行字段未反映 default 桶)导致抽屉恒显未启用。
        enabled={selected ? enabledSet.has(selected.id) : false}
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
