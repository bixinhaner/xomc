/**
 * AlarmLibraryPage — 告警库主页面(T-0179 drill-down 重设计)。
 *
 * 2026-05-29 用户决策:
 *   1. 与 product/param-model 页面 UI 对齐(drill-down 主从)
 *   2. 一级页面:NeTypesTable — 按 (ne_type, loaded_from) 聚合,每行点击下钻
 *   3. 二级页面:AlarmDefinitionTable — 按 ne_type 过滤,toolbar 含返回 + 新增(预填+锁定 ne_type)
 *   4. URL 同步 ?neType=ENB(刷新不回列表;返回按钮显式回一级)
 *   5. 8 个 icon 对齐 param-model 风格(返回 / 上传 / 重载 / 刷新 / 删除 / 详情 / 新增 / 警告)
 */
import { useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Input,
  Select,
  Popconfirm,
  message,
  Tooltip,
  Typography,
} from 'antd';
import {
  ArrowLeftOutlined,
  CloudDownloadOutlined,
  DeleteOutlined,
  EyeOutlined,
  PlusOutlined,
  ReloadOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import {
  useAlarmDefinitionList,
  useAlarmNeTypeStats,
  useDeleteAlarmDefinition,
  useAlarmSeverityLevels,
  useAlarmDefinitionImportDirectory,
  useAlarmDefinitionCacheRefresh,
} from '@core/hooks/api/useAlarmDefinitions';
import type {
  AlarmDefinition,
  AlarmDefinitionFilter,
  AlarmNeTypeStat,
} from '@core/types/alarmDefinition';
import AlarmDefinitionDrawer from './AlarmDefinitionDrawer';
import UnknownStatsModal from './UnknownStatsModal';

const { Text } = Typography;

const SEVERITY_TAG_COLORS: Record<number, string> = {
  1: 'red',
  31001: 'red',
  2: 'orange',
  31002: 'orange',
  3: 'gold',
  31003: 'gold',
  4: 'blue',
  31004: 'blue',
};

const FILTER_LABEL_STYLE = {
  color: 'rgba(0, 0, 0, 0.65)',
  fontSize: 12,
  whiteSpace: 'nowrap' as const,
};

export default function AlarmLibraryPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedNeType = searchParams.get('neType') || undefined;
  const inDetail = Boolean(selectedNeType);

  const setSelectedNeType = (next: string | undefined) => {
    const params = new URLSearchParams(searchParams);
    if (next) {
      params.set('neType', next);
    } else {
      params.delete('neType');
    }
    setSearchParams(params, { replace: false });
  };

  // ── 一级(NeTypes 聚合) ─────────────────────────────────────────
  const { data: neTypesData, isLoading: isNeTypesLoading } = useAlarmNeTypeStats();
  const [neTypesKeyword, setNeTypesKeyword] = useState('');
  const neTypesItems = useMemo<AlarmNeTypeStat[]>(() => {
    const items = neTypesData?.items || [];
    const k = neTypesKeyword.trim().toLowerCase();
    if (!k) return items;
    return items.filter(
      (it) => it.neType.toLowerCase().includes(k) || it.loadedFrom.toLowerCase().includes(k)
    );
  }, [neTypesData, neTypesKeyword]);

  // ── 二级(AlarmDefinition 详情) ─────────────────────────────────
  const [detailFilter, setDetailFilter] = useState<AlarmDefinitionFilter>({
    page: 1,
    pageSize: 20,
  });
  const detailQueryFilter = useMemo<AlarmDefinitionFilter>(
    () => (selectedNeType ? { ...detailFilter, neType: selectedNeType } : detailFilter),
    [detailFilter, selectedNeType]
  );
  const { data: detailData, isLoading: isDetailLoading } = useAlarmDefinitionList(
    inDetail ? detailQueryFilter : { page: 1, pageSize: 1 }
  );
  const { data: sevData } = useAlarmSeverityLevels();
  const detailItems = useMemo(() => detailData?.items || [], [detailData]);

  // ── 公共 ───────────────────────────────────────────────────────
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<AlarmDefinition | null>(null);
  const [statsOpen, setStatsOpen] = useState(false);

  const delMut = useDeleteAlarmDefinition();
  const importMut = useAlarmDefinitionImportDirectory();
  const cacheMut = useAlarmDefinitionCacheRefresh();

  const severityOptions = useMemo(
    () =>
      (sevData?.items || [])
        .map((level) => ({
          label: `${level.code} - ${level.cnName} / ${level.enName}`,
          value: level.code,
        }))
        .sort((left, right) => left.value - right.value),
    [sevData]
  );

  // ── 一级表列 ────────────────────────────────────────────────────
  const neTypesColumns = [
    {
      title: '网元类型',
      dataIndex: 'neType',
      width: 160,
      render: (v: string, row: AlarmNeTypeStat) => (
        <Button
          type="link"
          size="small"
          onClick={() => setSelectedNeType(row.neType)}
          style={{ padding: 0, fontWeight: 600 }}
        >
          {v}
        </Button>
      ),
    },
    {
      title: 'XML 来源',
      dataIndex: 'loadedFrom',
      width: 200,
      render: (v: string) =>
        v ? <Tag>{v}</Tag> : <Tag color="warning">未回填(请重载)</Tag>,
    },
    { title: '告警总数', dataIndex: 'total', width: 100 },
    {
      title: 'Critical',
      dataIndex: 'criticalCnt',
      width: 100,
      render: (v: number) => (v > 0 ? <Tag color="red">{v}</Tag> : <span>—</span>),
    },
    {
      title: 'Major',
      dataIndex: 'majorCnt',
      width: 100,
      render: (v: number) => (v > 0 ? <Tag color="orange">{v}</Tag> : <span>—</span>),
    },
    {
      title: 'Minor',
      dataIndex: 'minorCnt',
      width: 100,
      render: (v: number) => (v > 0 ? <Tag color="gold">{v}</Tag> : <span>—</span>),
    },
    {
      title: 'Warning',
      dataIndex: 'warningCnt',
      width: 100,
      render: (v: number) => (v > 0 ? <Tag color="blue">{v}</Tag> : <span>—</span>),
    },
  ];

  // ── 二级表列 ────────────────────────────────────────────────────
  const detailColumns = [
    {
      title: 'identifier',
      dataIndex: 'identifier',
      width: 140,
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
    { title: '中文名', dataIndex: 'cnName', width: 200 },
    { title: '英文名', dataIndex: 'enName', width: 220, ellipsis: true },
    {
      title: '严重级别',
      dataIndex: 'severityCode',
      width: 110,
      render: (v: number, row: AlarmDefinition) => (
        <Tag color={SEVERITY_TAG_COLORS[v] || 'default'}>
          {v} - {row.severityName}
        </Tag>
      ),
    },
    { title: '事件类型', dataIndex: 'eventType', width: 130 },
    {
      title: 'UI 可见',
      dataIndex: 'isShow',
      width: 80,
      render: (v: boolean) => (v ? <Tag color="success">是</Tag> : <Tag>否</Tag>),
    },
    {
      title: '操作',
      width: 130,
      render: (_: unknown, row: AlarmDefinition) => (
        <Space>
          <Button
            size="small"
            icon={<EyeOutlined />}
            onClick={() => {
              setEditing(row);
              setDrawerOpen(true);
            }}
          />
          <Popconfirm
            title={`确认删除告警定义「${row.identifier}」?`}
            onConfirm={() =>
              delMut
                .mutateAsync(row.identifier)
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
    <div style={{ padding: 16 }}>
      {/* 顶部 toolbar:列表态展示搜索;详情态展示返回 + 当前 ne_type + 新增 */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
          {inDetail ? (
            <Space wrap>
              <Button
                icon={<ArrowLeftOutlined />}
                onClick={() => setSelectedNeType(undefined)}
              >
                返回
              </Button>
              <Text strong>{selectedNeType} 的告警定义</Text>
              <Space size={4}>
                <span style={FILTER_LABEL_STYLE}>严重级别</span>
                <Select
                  placeholder="全部"
                  allowClear
                  options={severityOptions}
                  value={detailFilter.severityCode}
                  onChange={(value) =>
                    setDetailFilter((current) => ({ ...current, severityCode: value, page: 1 }))
                  }
                  style={{ width: 180 }}
                />
              </Space>
              <Input.Search
                placeholder="搜索 identifier / 名称"
                allowClear
                onSearch={(v) =>
                  setDetailFilter((f) => ({ ...f, keyword: v || undefined, page: 1 }))
                }
                style={{ width: 220 }}
              />
            </Space>
          ) : (
            <Space wrap>
              <Input.Search
                placeholder="搜索网元类型 / XML 来源"
                allowClear
                value={neTypesKeyword}
                onChange={(e) => setNeTypesKeyword(e.target.value)}
                style={{ width: 280 }}
              />
              <Tooltip title="按 productId / days 聚合的未识别告警频次">
                <Button icon={<WarningOutlined />} onClick={() => setStatsOpen(true)}>
                  未识别频次
                </Button>
              </Tooltip>
            </Space>
          )}
          <Space wrap>
            {inDetail && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  setEditing(null);
                  setDrawerOpen(true);
                }}
              >
                新增定义
              </Button>
            )}
            <Popconfirm
              title="确认重载 XML?"
              description={
                <div style={{ maxWidth: 320 }}>
                  将从 <code>datamodels/</code> 重新加载所有告警定义 XML。
                  <br />
                  UI 中对告警定义的编辑将被 XML 值覆盖;操作不可撤销。
                </div>
              }
              okText="确认重载"
              cancelText="取消"
              okButtonProps={{ danger: true }}
              placement="bottomRight"
              onConfirm={() => {
                importMut
                  .mutateAsync()
                  .then((result) => message.success(`已重载:${result.reloaded}`))
                  .catch((error) => message.error((error as Error).message));
              }}
            >
              <Button icon={<CloudDownloadOutlined />} loading={importMut.isPending} danger>
                重载 XML
              </Button>
            </Popconfirm>
            <Button
              icon={<ReloadOutlined />}
              loading={cacheMut.isPending}
              onClick={() =>
                cacheMut
                  .mutateAsync()
                  .then(() => message.success('已刷新缓存'))
                  .catch((e) => message.error((e as Error).message))
              }
            >
              刷新缓存
            </Button>
          </Space>
        </Space>
      </Card>

      {/* drill-down 主体:列表态 ↔ 详情态 二选一 */}
      <Card size="small">
        {inDetail ? (
          <Table<AlarmDefinition>
            rowKey="id"
            loading={isDetailLoading}
            columns={detailColumns}
            dataSource={detailItems}
            size="small"
            pagination={{
              current: detailData?.page || 1,
              pageSize: detailData?.pageSize || 20,
              total: detailData?.total || 0,
              showSizeChanger: true,
              onChange: (page, pageSize) =>
                setDetailFilter((f) => ({ ...f, page, pageSize })),
            }}
          />
        ) : (
          <Table<AlarmNeTypeStat>
            rowKey={(r) => `${r.neType}__${r.loadedFrom}`}
            loading={isNeTypesLoading}
            columns={neTypesColumns}
            dataSource={neTypesItems}
            size="small"
            pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 个网元类型` }}
          />
        )}
      </Card>

      <AlarmDefinitionDrawer
        open={drawerOpen}
        definition={editing}
        defaultNeType={!editing ? selectedNeType : undefined}
        lockNeType={!editing && Boolean(selectedNeType)}
        onClose={() => {
          setDrawerOpen(false);
          setEditing(null);
        }}
      />
      <UnknownStatsModal open={statsOpen} onClose={() => setStatsOpen(false)} />
    </div>
  );
}
