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
} from '@core/hooks/api/useAlarmDefinitions';
import type { AlarmDefinition, AlarmDefinitionFilter } from '@core/types/alarmDefinition';
import AlarmDefinitionDrawer from './AlarmDefinitionDrawer';
import UnknownStatsModal from './UnknownStatsModal';

const UNKNOWN_OPTIONS = [
  { label: '全部', value: 'all' },
  { label: '已识别', value: 'false' },
  { label: '未识别 fallback', value: 'true' },
];

const SEVERITY_TAG_COLORS: Record<number, string> = {
  1: 'red',
  2: 'orange',
  3: 'gold',
  4: 'blue',
};

export default function AlarmLibraryPage() {
  const [filter, setFilter] = useState<AlarmDefinitionFilter>({ page: 1, pageSize: 20 });
  const [keyword, setKeyword] = useState('');
  const [unknownOpt, setUnknownOpt] = useState<'all' | 'true' | 'false'>('all');
  const { data, isLoading } = useAlarmDefinitionList(filter);
  const { data: sevData } = useAlarmSeverityLevels();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<AlarmDefinition | null>(null);
  const [statsOpen, setStatsOpen] = useState(false);

  const delMut = useDeleteAlarmDefinition();
  const importMut = useAlarmDefinitionImportDirectory();
  const cacheMut = useAlarmDefinitionCacheRefresh();

  const items = useMemo(() => data?.items || [], [data]);

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
        title="告警库 / Alarm Definitions"
        extra={
          <Space>
            <Input.Search
              placeholder="搜索 identifier / 名称"
              allowClear
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onSearch={(v) => setFilter((f) => ({ ...f, keyword: v || undefined, page: 1 }))}
              style={{ width: 220 }}
            />
            <Select
              placeholder="网元类型"
              allowClear
              options={[
                { label: '全部', value: '' },
                { label: 'eNodeB', value: 'eNodeB' },
                { label: 'gNodeB', value: 'gNodeB' },
                { label: 'BTS', value: 'BTS' },
              ]}
              value={filter.neType ?? ''}
              onChange={(v) => setFilter((f) => ({ ...f, neType: v || undefined, page: 1 }))}
              style={{ width: 110 }}
            />
            <Select
              placeholder="严重级别"
              allowClear
              options={[
                { label: '全部', value: -1 },
                ...((sevData?.items || []).map((s) => ({
                  label: `${s.code} ${s.enName}`,
                  value: s.code,
                }))),
              ]}
              value={filter.severityCode ?? -1}
              onChange={(v) => setFilter((f) => ({ ...f, severityCode: v >= 0 ? v : undefined, page: 1 }))}
              style={{ width: 130 }}
            />
            <Select
              value={unknownOpt}
              onChange={(v) => {
                setUnknownOpt(v);
                setFilter((f) => ({
                  ...f,
                  isUnknown: v === 'all' ? undefined : v === 'true',
                  page: 1,
                }));
              }}
              options={UNKNOWN_OPTIONS}
              style={{ width: 140 }}
            />
            <Tooltip title="按 productId / days 聚合的未识别告警频次">
              <Button icon={<WarningOutlined />} onClick={() => setStatsOpen(true)}>
                未识别频次
              </Button>
            </Tooltip>
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
                // 不返回 Promise — 让 Popconfirm 立即关闭；loading 反馈交给触发按钮
                importMut
                  .mutateAsync()
                  .then((r) => message.success(`已重载：${r.reloaded}`))
                  .catch((e) => message.error((e as Error).message));
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
          loading={isLoading}
          columns={columns}
          dataSource={items}
          size="small"
          pagination={{
            current: data?.page || 1,
            pageSize: data?.pageSize || 20,
            total: data?.total || 0,
            showSizeChanger: true,
            onChange: (page, pageSize) => setFilter((f) => ({ ...f, page, pageSize })),
          }}
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
