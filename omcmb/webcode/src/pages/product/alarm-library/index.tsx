import { useMemo, useState } from 'react';
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
} from 'antd';
import {
  PlusOutlined,
  EyeOutlined,
  DeleteOutlined,
  CloudDownloadOutlined,
  ReloadOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import {
  useAlarmDefinitionList,
  useDeleteAlarmDefinition,
  useAlarmSeverityLevels,
  useAlarmDefinitionImportDirectory,
  useAlarmDefinitionCacheRefresh,
  useUnknownAlarmStats,
} from '@core/hooks/api/useAlarmDefinitions';
import type { AlarmDefinition, AlarmDefinitionFilter } from '@core/types/alarmDefinition';
import AlarmDefinitionDrawer from './AlarmDefinitionDrawer';
import UnknownStatsModal from './UnknownStatsModal';

const UNKNOWN_OPTIONS = [
  { label: '已识别', value: 'false' },
  { label: '未识别 fallback', value: 'true' },
];

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

const OPTION_CATALOG_FILTER: AlarmDefinitionFilter = {
  page: 1,
  pageSize: 1000,
};

export default function AlarmLibraryPage() {
  const [filter, setFilter] = useState<AlarmDefinitionFilter>({ page: 1, pageSize: 20 });
  const [keyword, setKeyword] = useState('');
  const [unknownOpt, setUnknownOpt] = useState<'true' | 'false'>();
  const { data, isLoading } = useAlarmDefinitionList(filter);
  const { data: optionCatalogData } = useAlarmDefinitionList(OPTION_CATALOG_FILTER);
  const { data: unknownStatsData, isLoading: isUnknownStatsLoading } = useUnknownAlarmStats({
    days: 7,
  });
  const { data: sevData } = useAlarmSeverityLevels();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<AlarmDefinition | null>(null);
  const [statsOpen, setStatsOpen] = useState(false);

  const delMut = useDeleteAlarmDefinition();
  const importMut = useAlarmDefinitionImportDirectory();
  const cacheMut = useAlarmDefinitionCacheRefresh();
  const isUnknownOnly = unknownOpt === 'true';

  const items = useMemo(() => data?.items || [], [data]);
  const optionCatalogItems = useMemo(() => optionCatalogData?.items || [], [optionCatalogData]);
  const unknownFallbackItems = useMemo(() => {
    const keywordFilter = filter.keyword?.trim().toLowerCase();
    return (unknownStatsData?.items || [])
      .filter((item) => {
        if (!keywordFilter) {
          return true;
        }
        return item.identifier.toLowerCase().includes(keywordFilter);
      })
      .map((item) => ({
        id: `unknown-${item.productId || 'all'}-${item.identifier}`,
        identifier: item.identifier,
        neType: item.productName || '未知产品',
        cnName: '未识别 fallback 告警',
        enName: 'Unknown fallback alarm',
        severityCode: 31004,
        severityName: 'Warning',
        cnProbableCause: item.lastSeenAt ? `最近出现时间：${item.lastSeenAt}` : undefined,
        enProbableCause: item.lastSeenAt ? `Last seen at: ${item.lastSeenAt}` : undefined,
        cnSuggestion: '请通过“未识别频次”确认产品后补充告警字典。',
        enSuggestion: 'Check unknown stats and add a matching alarm definition.',
        isShow: true,
        isUnknown: true,
      })) satisfies AlarmDefinition[];
  }, [filter.keyword, unknownStatsData]);
  const tableItems = isUnknownOnly ? unknownFallbackItems : items;
  const tableLoading = isUnknownOnly ? isUnknownStatsLoading : isLoading;
  const neTypeOptions = useMemo(
    () =>
      Array.from(new Set(optionCatalogItems.map((item) => item.neType).filter(Boolean)))
        .sort((left, right) => left.localeCompare(right))
        .map((value) => ({ label: value, value })),
    [optionCatalogItems]
  );
  const severityOptions = useMemo(() => {
    const apiOptions = (sevData?.items || []).map((level) => ({
      label: `${level.code} - ${level.cnName} / ${level.enName}`,
      value: level.code,
    }));
    if (apiOptions.length > 0) {
      return apiOptions;
    }

    return Array.from(
      new Map(
        optionCatalogItems.map((item) => [
          item.severityCode,
          {
            label: `${item.severityCode} - ${item.severityName}`,
            value: item.severityCode,
          },
        ])
      ).values()
    ).sort((left, right) => left.value - right.value);
  }, [optionCatalogItems, sevData]);

  const columns = [
    {
      title: 'identifier',
      dataIndex: 'identifier',
      width: 140,
      render: (v: string, row: AlarmDefinition) => (
        <Space>
          <Tag color="blue">{v}</Tag>
          {row.isUnknown && <Tag color="warning">未识别</Tag>}
        </Space>
      ),
    },
    { title: '中文名', dataIndex: 'cnName', width: 200 },
    { title: '英文名', dataIndex: 'enName', width: 220, ellipsis: true },
    { title: '网元类型', dataIndex: 'neType', width: 110 },
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
            title={`确认删除告警定义「${row.identifier}」？`}
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
      <Card
        size="small"
        extra={
          <Space wrap size={12}>
            <Input.Search
              placeholder="搜索 identifier / 名称"
              allowClear
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onSearch={(v) => setFilter((f) => ({ ...f, keyword: v || undefined, page: 1 }))}
              style={{ width: 220 }}
            />
            <Space size={4}>
              <span style={FILTER_LABEL_STYLE}>网元类型</span>
              <Select
                placeholder="全部"
                allowClear
                disabled={isUnknownOnly}
                options={neTypeOptions}
                value={filter.neType}
                onChange={(value) =>
                  setFilter((current) => ({ ...current, neType: value, page: 1 }))
                }
                style={{ width: 140 }}
              />
            </Space>
            <Space size={4}>
              <span style={FILTER_LABEL_STYLE}>严重级别</span>
              <Select
                placeholder="全部"
                allowClear
                disabled={isUnknownOnly}
                options={severityOptions}
                value={filter.severityCode}
                onChange={(value) =>
                  setFilter((current) => ({ ...current, severityCode: value, page: 1 }))
                }
                style={{ width: 180 }}
              />
            </Space>
            <Space size={4}>
              <span style={FILTER_LABEL_STYLE}>识别状态</span>
              <Select
                placeholder="全部"
                allowClear
                value={unknownOpt}
                onChange={(value: 'true' | 'false' | undefined) => {
                  const nextIsUnknown = value === undefined ? undefined : value === 'true';
                  setUnknownOpt(value);
                  setFilter((current) => ({
                    ...current,
                    neType: value === 'true' ? undefined : current.neType,
                    severityCode: value === 'true' ? undefined : current.severityCode,
                    isUnknown: nextIsUnknown,
                    page: 1,
                  }));
                }}
                options={UNKNOWN_OPTIONS}
                style={{ width: 150 }}
              />
            </Space>
            <Space size={4}>
              <span style={FILTER_LABEL_STYLE}>统计</span>
              <Tooltip title="按 productId / days 聚合的未识别告警频次">
                <Button icon={<WarningOutlined />} onClick={() => setStatsOpen(true)}>
                  未识别频次
                </Button>
              </Tooltip>
            </Space>
            <Space size={4}>
              <span style={FILTER_LABEL_STYLE}>维护</span>
              <Popconfirm
                title="确认重载 XML？"
                description={
                  <div style={{ maxWidth: 320 }}>
                    将从 <code>datamodels/</code> 重新加载所有告警定义 XML。
                    <br />
                    UI 中对告警定义的编辑将被 XML 值覆盖；操作不可撤销。
                  </div>
                }
                okText="确认重载"
                cancelText="取消"
                okButtonProps={{ danger: true }}
                placement="bottomRight"
                onConfirm={() => {
                  importMut
                    .mutateAsync()
                    .then((result) => message.success(`已重载：${result.reloaded}`))
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
          </Space>
        }
      >
        <Table<AlarmDefinition>
          rowKey="id"
          loading={tableLoading}
          columns={columns}
          dataSource={tableItems}
          size="small"
          pagination={
            isUnknownOnly
              ? {
                  pageSize: 20,
                  showSizeChanger: false,
                }
              : {
                  current: data?.page || 1,
                  pageSize: data?.pageSize || 20,
                  total: data?.total || 0,
                  showSizeChanger: true,
                  onChange: (page, pageSize) => setFilter((f) => ({ ...f, page, pageSize })),
                }
          }
        />
      </Card>

      <AlarmDefinitionDrawer
        open={drawerOpen}
        definition={editing}
        onClose={() => {
          setDrawerOpen(false);
          setEditing(null);
        }}
      />
      <UnknownStatsModal open={statsOpen} onClose={() => setStatsOpen(false)} />
    </div>
  );
}
