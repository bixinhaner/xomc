import { useMemo, useState } from 'react';
import { Card, Table, Input, Switch, Button, Space, Select, message, Popconfirm } from 'antd';
import { EyeOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  useIndicatorList,
  useDeleteIndicator,
  useEnabledIndicators,
  useSetEnabledIndicators,
  usePlatformList,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import IndicatorDrawer from './IndicatorDrawer';
import { useT } from '@/hooks/useT';

interface Props {
  deviceType: DeviceType;
}

// 启用状态走 default 行（XML 真相源；运营商覆盖能力后端保留但 UI 不暴露选择器）
const OPERATOR_CODE = 'default';

export default function IndicatorTab({ deviceType }: Props) {
  const t = useT();
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
    { title: t('common.cnName'), dataIndex: 'cnName', width: 200 },
    { title: t('common.enName'), dataIndex: 'enName', width: 240, ellipsis: true },
    {
      title: t('common.group'),
      dataIndex: 'groupName',
      width: 140,
      render: (v: string, row: IndicatorInfo) => v || row.groupId || '—',
    },
    { title: t('product.kpi.col.counterType'), dataIndex: 'counterType', width: 110 },
    ...(deviceType === 'GNB'
      ? []
      : [{ title: t('common.level'), dataIndex: 'indicatorLevel', width: 90 }]),
    { title: t('common.unit'), dataIndex: 'unit', width: 80 },
    {
      title: t('common.enable'),
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
      title: t('common.action'),
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
            {t('common.detail')}
          </Button>
          <Popconfirm
            title={t('product.kpi.confirmDeleteIndicator', { id: row.id })}
            onConfirm={() =>
              deleteMut
                .mutateAsync({ deviceType, id: row.id })
                .then(() => message.success(t('common.deleted')))
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
      title={t('product.kpi.indicatorsByDevice', { deviceType })}
      extra={
        <Space>
          {showPlatformFilter && (
            <Select
              placeholder={t('product.kpi.platformFilterPh')}
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
            placeholder={t('product.kpi.idNameSearchPh')}
            allowClear
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value);
              setPage(1);
            }}
            style={{ width: 320 }}
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
          showTotal: (total) => t('common.totalCount', { count: total }),
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
        // 启用状态与列表开关同源(default 启用桶 enabledSet)
        enabled={selected ? enabledSet.has(selected.id) : false}
        onClose={() => {
          setDrawerOpen(false);
          setSelected(null);
        }}
      />
    </Card>
  );
}
