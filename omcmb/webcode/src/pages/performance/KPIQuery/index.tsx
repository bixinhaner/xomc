/**
 * T-0174 指标查询页（重做版）。
 *
 * 旧版 1500 行 mock 全部删除，替换为真后端对接：
 *   - 模板侧栏 → /api/v1/pm/query-templates
 *   - 设备/指标选择 → DevicePickerModal / MetricPickerModal
 *   - 查询 → /api/v1/pm/metrics/aggregated（多设备并发后合并）
 *   - 结果 → 透视表（行=时间 / 列=N 指标 / 单元格=值）
 */

import { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Radio,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
  Empty,
  App,
  Tabs,
  List,
  Popconfirm,
  Spin,
  DatePicker,
} from 'antd';
import {
  ReloadOutlined,
  SaveOutlined,
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  TeamOutlined,
  UserOutlined,
  ExportOutlined,
  TableOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import { useThemeToken } from '@/hooks/useThemeToken';
import {
  useQueryTemplates,
  useCreateQueryTemplate,
  useUpdateQueryTemplate,
  useDeleteQueryTemplate,
  useAggregatedMetricsByDevices,
} from '@core/hooks/api/usePmQuery';
import { useUserStore } from '@core/store/userStore';
import type { Granularity } from '@core/types/pmDashboard';
import type {
  QueryTemplate,
  QueryTemplatePayload,
  TemplateVisibility,
  TimeRangePreset,
} from '@core/types/pmQuery';
import DevicePickerModal from './components/DevicePickerModal';
import MetricPickerModal from './components/MetricPickerModal';
import PivotTable from './components/PivotTable';

const { Text, Title } = Typography;
const { RangePicker } = DatePicker;

const GRANULARITY_OPTIONS: { label: string; value: Granularity }[] = [
  { label: '15 分钟', value: '15min' },
  { label: '小时', value: 'hourly' },
  { label: '日', value: 'daily' },
  { label: '周', value: 'weekly' },
  { label: '月', value: 'monthly' },
];

const TIME_RANGE_OPTIONS: { label: string; value: TimeRangePreset }[] = [
  { label: '近 1 小时', value: 'last_1h' },
  { label: '近 24 小时', value: 'last_24h' },
  { label: '近 7 天', value: 'last_7d' },
  { label: '近 30 天', value: 'last_30d' },
  { label: '自定义', value: 'custom' },
];

const DEVICE_TYPE_OPTIONS = [
  { label: 'eNB', value: 'ENB' as const },
  { label: 'gNB', value: 'GNB' as const },
  { label: 'GSM', value: 'GSM' as const },
];

const DEFAULT_PAYLOAD: QueryTemplatePayload = {
  deviceSns: [],
  metricPaths: [],
  granularity: '15min',
  timeRangePreset: 'last_1h',
  deviceType: 'ENB',
};

interface SaveTemplateFormState {
  open: boolean;
  mode: 'create' | 'update';
  templateId?: string;
  name: string;
  description: string;
  visibility: TemplateVisibility;
}

function presetToRange(preset: TimeRangePreset): { start: string; end: string } | null {
  const now = dayjs();
  switch (preset) {
    case 'last_1h':
      return { start: now.subtract(1, 'hour').toISOString(), end: now.toISOString() };
    case 'last_24h':
      return { start: now.subtract(24, 'hour').toISOString(), end: now.toISOString() };
    case 'last_7d':
      return { start: now.subtract(7, 'day').toISOString(), end: now.toISOString() };
    case 'last_30d':
      return { start: now.subtract(30, 'day').toISOString(), end: now.toISOString() };
    case 'custom':
      return null;
  }
}

export default function KPIQuery() {
  const token = useThemeToken();
  const { message } = App.useApp();
  const currentUser = useUserStore((s) => s.currentUser);
  const isSuperAdmin = currentUser?.isSuperAdmin ?? false;

  // ── 查询表单状态 ─────────────────────────────────────────────────
  const [payload, setPayload] = useState<QueryTemplatePayload>(DEFAULT_PAYLOAD);
  const [customRange, setCustomRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [devicePickerOpen, setDevicePickerOpen] = useState(false);
  const [metricPickerOpen, setMetricPickerOpen] = useState(false);

  // ── 模板侧栏状态 ─────────────────────────────────────────────────
  const [templateTab, setTemplateTab] = useState<'public' | 'private'>('public');
  const [activeTemplateId, setActiveTemplateId] = useState<string | undefined>(undefined);

  const { data: templatesData, isLoading: templatesLoading, refetch: refetchTemplates } =
    useQueryTemplates({ pageSize: 200 });
  const createMut = useCreateQueryTemplate();
  const updateMut = useUpdateQueryTemplate();
  const deleteMut = useDeleteQueryTemplate();

  const publicTemplates = useMemo(
    () => (templatesData?.items ?? []).filter((t) => t.visibility === 'public'),
    [templatesData],
  );
  const privateTemplates = useMemo(
    () => (templatesData?.items ?? []).filter((t) => t.visibility === 'private'),
    [templatesData],
  );

  // ── 存为模板 Modal ───────────────────────────────────────────────
  const [saveForm, setSaveForm] = useState<SaveTemplateFormState>({
    open: false,
    mode: 'create',
    name: '',
    description: '',
    visibility: 'private',
  });

  // ── 查询执行状态 ─────────────────────────────────────────────────
  // submittedPayload 是真正用于查询的快照；表单编辑时不立即查询，等用户点"查询"
  const [submittedPayload, setSubmittedPayload] = useState<QueryTemplatePayload | null>(null);
  const [submittedRange, setSubmittedRange] = useState<{ start: string; end: string } | null>(null);

  const baseAggParams = useMemo(() => {
    if (!submittedPayload || !submittedRange) return null;
    return {
      granularity: submittedPayload.granularity,
      metricPaths: submittedPayload.metricPaths,
      startTime: submittedRange.start,
      endTime: submittedRange.end,
      limit: 5000,
    };
  }, [submittedPayload, submittedRange]);

  const {
    data: aggregatedRows,
    isLoading: aggLoading,
    isFetching: aggFetching,
    errors: aggErrors,
    refetch: refetchAgg,
  } = useAggregatedMetricsByDevices(
    baseAggParams ?? {
      granularity: '15min',
      metricPaths: [],
      startTime: undefined,
      endTime: undefined,
    },
    submittedPayload?.deviceSns ?? [],
    Boolean(baseAggParams),
  );

  useEffect(() => {
    if (aggErrors.length > 0) {
      const first = aggErrors[0] as Error;
      message.error(`查询失败：${first?.message ?? '未知错误'}`);
    }
  }, [aggErrors, message]);

  // ── 行为 ─────────────────────────────────────────────────────────
  const handleQuery = () => {
    if (payload.deviceSns.length === 0) {
      message.warning('请选择至少一个设备');
      return;
    }
    if (payload.metricPaths.length === 0) {
      message.warning('请选择至少一个指标');
      return;
    }
    let range: { start: string; end: string } | null;
    if (payload.timeRangePreset === 'custom') {
      if (!customRange) {
        message.warning('请选择自定义时间范围');
        return;
      }
      range = {
        start: customRange[0].toISOString(),
        end: customRange[1].toISOString(),
      };
    } else {
      range = presetToRange(payload.timeRangePreset);
    }
    if (!range) {
      message.warning('请选择时间范围');
      return;
    }
    setSubmittedPayload(payload);
    setSubmittedRange(range);
  };

  const handleSelectTemplate = (tpl: QueryTemplate) => {
    setActiveTemplateId(tpl.id);
    setPayload(tpl.payload);
    if (tpl.payload.timeRangePreset === 'custom' && tpl.payload.absoluteStart && tpl.payload.absoluteEnd) {
      setCustomRange([dayjs(tpl.payload.absoluteStart), dayjs(tpl.payload.absoluteEnd)]);
    } else {
      setCustomRange(null);
    }
  };

  const handleOpenSaveModal = () => {
    if (payload.deviceSns.length === 0 || payload.metricPaths.length === 0) {
      message.warning('请先填好查询条件再存为模板');
      return;
    }
    setSaveForm({
      open: true,
      mode: 'create',
      name: '',
      description: '',
      visibility: 'private',
    });
  };

  const handleOpenUpdateModal = (tpl: QueryTemplate) => {
    setSaveForm({
      open: true,
      mode: 'update',
      templateId: tpl.id,
      name: tpl.name,
      description: tpl.description ?? '',
      visibility: tpl.visibility,
    });
  };

  const handleSaveTemplate = async () => {
    if (!saveForm.name.trim()) {
      message.warning('请输入模板名称');
      return;
    }
    // 保存时回填 custom 模式的绝对时间
    const payloadToSave: QueryTemplatePayload = {
      ...payload,
      absoluteStart:
        payload.timeRangePreset === 'custom' && customRange ? customRange[0].toISOString() : undefined,
      absoluteEnd:
        payload.timeRangePreset === 'custom' && customRange ? customRange[1].toISOString() : undefined,
    };
    try {
      if (saveForm.mode === 'create') {
        await createMut.mutateAsync({
          name: saveForm.name.trim(),
          visibility: saveForm.visibility,
          description: saveForm.description,
          payload: payloadToSave,
        });
        message.success('模板已创建');
      } else if (saveForm.templateId) {
        await updateMut.mutateAsync({
          id: saveForm.templateId,
          input: {
            name: saveForm.name.trim(),
            description: saveForm.description,
            visibility: saveForm.visibility,
            payload: payloadToSave,
          },
        });
        message.success('模板已更新');
      }
      setSaveForm((s) => ({ ...s, open: false }));
    } catch (err) {
      const e = err as Error & { response?: { status?: number } };
      if (e?.response?.status === 409) {
        message.error('名称已被你创建的另一个模板占用');
      } else if (e?.response?.status === 403) {
        message.error('权限不足：仅 super_admin 可创建/修改公共模板');
      } else {
        message.error(`保存失败：${e?.message ?? '未知错误'}`);
      }
    }
  };

  const handleDeleteTemplate = async (id: string) => {
    try {
      await deleteMut.mutateAsync(id);
      message.success('模板已删除');
      if (activeTemplateId === id) {
        setActiveTemplateId(undefined);
      }
    } catch (err) {
      const e = err as Error;
      message.error(`删除失败：${e?.message ?? '未知错误'}`);
    }
  };

  // ── 渲染辅助 ─────────────────────────────────────────────────────
  const renderTemplateItem = (tpl: QueryTemplate) => {
    const canEdit = isSuperAdmin || tpl.creatorId === currentUser?.id;
    return (
      <List.Item
        key={tpl.id}
        style={{
          padding: '8px 12px',
          cursor: 'pointer',
          background: activeTemplateId === tpl.id ? token.colorBgTextHover : 'transparent',
          borderRadius: 4,
        }}
        onClick={() => handleSelectTemplate(tpl)}
        actions={
          canEdit
            ? [
                <Tooltip key="edit" title="编辑">
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={(e) => {
                      e.stopPropagation();
                      handleOpenUpdateModal(tpl);
                    }}
                  />
                </Tooltip>,
                <Popconfirm
                  key="del"
                  title="确定删除该模板？"
                  onConfirm={(e) => {
                    e?.stopPropagation();
                    void handleDeleteTemplate(tpl.id);
                  }}
                  onCancel={(e) => e?.stopPropagation()}
                >
                  <Button
                    type="text"
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={(e) => e.stopPropagation()}
                  />
                </Popconfirm>,
              ]
            : undefined
        }
      >
        <List.Item.Meta
          title={
            <Space>
              <Text>{tpl.name}</Text>
              {tpl.visibility === 'public' ? (
                <Tag color="blue">公共</Tag>
              ) : (
                <Tag color="default">私有</Tag>
              )}
            </Space>
          }
          description={
            tpl.description ? (
              <Text type="secondary" ellipsis style={{ fontSize: 12 }}>
                {tpl.description}
              </Text>
            ) : null
          }
        />
      </List.Item>
    );
  };

  const sidebar = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 16px', borderBottom: `1px solid ${token.colorBorderSecondary}` }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }}>
          <Title level={5} style={{ margin: 0 }}>
            查询模板
          </Title>
          <Button
            type="text"
            size="small"
            icon={<ReloadOutlined />}
            onClick={() => void refetchTemplates()}
          />
        </Space>
      </div>
      <Tabs
        activeKey={templateTab}
        onChange={(k) => setTemplateTab(k as 'public' | 'private')}
        items={[
          {
            key: 'public',
            label: (
              <span>
                <TeamOutlined /> 公共 ({publicTemplates.length})
              </span>
            ),
            children: (
              <Spin spinning={templatesLoading}>
                <List
                  dataSource={publicTemplates}
                  renderItem={renderTemplateItem}
                  locale={{ emptyText: <Empty description="暂无公共模板" /> }}
                />
              </Spin>
            ),
          },
          {
            key: 'private',
            label: (
              <span>
                <UserOutlined /> 私有 ({privateTemplates.length})
              </span>
            ),
            children: (
              <Spin spinning={templatesLoading}>
                <List
                  dataSource={privateTemplates}
                  renderItem={renderTemplateItem}
                  locale={{ emptyText: <Empty description="暂无私有模板" /> }}
                />
              </Spin>
            ),
          },
        ]}
        style={{ flex: 1, overflow: 'auto', padding: '0 8px' }}
      />
    </div>
  );

  return (
    <TreeListPageLayout tree={sidebar} defaultTreeWidth={280} minTreeWidth={220}>
      <div style={{ padding: 16, height: '100%', overflow: 'auto' }}>
        <Card
          size="small"
          style={{ marginBottom: 12 }}
          title={
            <Space>
              <TableOutlined />
              <span>查询条件</span>
            </Space>
          }
        >
          <Form layout="vertical" size="middle">
            <Space wrap size="middle" align="start">
              <Form.Item label="设备类型" style={{ marginBottom: 0 }}>
                <Select
                  style={{ width: 120 }}
                  value={payload.deviceType}
                  onChange={(v) => setPayload({ ...payload, deviceType: v })}
                  options={DEVICE_TYPE_OPTIONS}
                />
              </Form.Item>

              <Form.Item label="设备" style={{ marginBottom: 0 }}>
                <Space.Compact style={{ width: 360 }}>
                  <Input
                    readOnly
                    value={
                      payload.deviceSns.length === 0
                        ? ''
                        : `已选 ${payload.deviceSns.length} 个：${payload.deviceSns.slice(0, 2).join(', ')}${payload.deviceSns.length > 2 ? ' ...' : ''}`
                    }
                    placeholder="点击右侧按钮选择设备"
                  />
                  <Button onClick={() => setDevicePickerOpen(true)}>列表选</Button>
                </Space.Compact>
              </Form.Item>

              <Form.Item label="指标" style={{ marginBottom: 0 }}>
                <Space.Compact style={{ width: 360 }}>
                  <Input
                    readOnly
                    value={
                      payload.metricPaths.length === 0
                        ? ''
                        : `已选 ${payload.metricPaths.length} 个：${payload.metricPaths.slice(0, 2).join(', ')}${payload.metricPaths.length > 2 ? ' ...' : ''}`
                    }
                    placeholder="点击右侧按钮选择指标"
                  />
                  <Button onClick={() => setMetricPickerOpen(true)}>列表选</Button>
                </Space.Compact>
              </Form.Item>

              <Form.Item label="粒度" style={{ marginBottom: 0 }}>
                <Radio.Group
                  value={payload.granularity}
                  onChange={(e) => setPayload({ ...payload, granularity: e.target.value })}
                  options={GRANULARITY_OPTIONS}
                  optionType="button"
                  buttonStyle="solid"
                />
              </Form.Item>

              <Form.Item label="时间范围" style={{ marginBottom: 0 }}>
                <Space>
                  <Select
                    style={{ width: 140 }}
                    value={payload.timeRangePreset}
                    onChange={(v) => setPayload({ ...payload, timeRangePreset: v })}
                    options={TIME_RANGE_OPTIONS}
                    suffixIcon={<ClockCircleOutlined />}
                  />
                  {payload.timeRangePreset === 'custom' && (
                    <RangePicker
                      showTime
                      value={customRange}
                      onChange={(v) => setCustomRange(v as [dayjs.Dayjs, dayjs.Dayjs] | null)}
                    />
                  )}
                </Space>
              </Form.Item>
            </Space>

            <div style={{ marginTop: 16, borderTop: `1px dashed ${token.colorBorderSecondary}`, paddingTop: 12 }}>
              <Space>
                <Button type="primary" icon={<TableOutlined />} loading={aggFetching} onClick={handleQuery}>
                  查询
                </Button>
                <Button icon={<ReloadOutlined />} onClick={() => void refetchAgg()} disabled={!submittedPayload}>
                  刷新
                </Button>
                <Button icon={<SaveOutlined />} onClick={handleOpenSaveModal}>
                  存为模板
                </Button>
                <Tooltip title="导出 Excel/CSV — 留作后续阶段实现">
                  <Button icon={<ExportOutlined />} disabled>
                    导出
                  </Button>
                </Tooltip>
                <Button
                  icon={<PlusOutlined />}
                  onClick={() => {
                    setPayload(DEFAULT_PAYLOAD);
                    setCustomRange(null);
                    setActiveTemplateId(undefined);
                    setSubmittedPayload(null);
                  }}
                >
                  重置
                </Button>
              </Space>
            </div>
          </Form>
        </Card>

        <Card size="small" title={<span><TableOutlined /> 查询结果</span>}>
          <PivotTable rows={aggregatedRows} loading={aggLoading || aggFetching} />
        </Card>

        <DevicePickerModal
          open={devicePickerOpen}
          onClose={() => setDevicePickerOpen(false)}
          onConfirm={(sns) => setPayload({ ...payload, deviceSns: sns })}
          initialSelected={payload.deviceSns}
        />

        <MetricPickerModal
          open={metricPickerOpen}
          onClose={() => setMetricPickerOpen(false)}
          onConfirm={(paths) => setPayload({ ...payload, metricPaths: paths })}
          initialSelected={payload.metricPaths}
          initialDeviceType={payload.deviceType ?? 'ENB'}
        />

        <Modal
          title={saveForm.mode === 'create' ? '存为查询模板' : '更新查询模板'}
          open={saveForm.open}
          onCancel={() => setSaveForm((s) => ({ ...s, open: false }))}
          onOk={handleSaveTemplate}
          confirmLoading={createMut.isPending || updateMut.isPending}
          okText="保存"
          cancelText="取消"
          destroyOnHidden
        >
          <Form layout="vertical">
            <Form.Item label="模板名称" required>
              <Input
                value={saveForm.name}
                onChange={(e) => setSaveForm({ ...saveForm, name: e.target.value })}
                maxLength={128}
                placeholder="例如：eNB 基础 KPI"
              />
            </Form.Item>
            <Form.Item label="可见性">
              <Radio.Group
                value={saveForm.visibility}
                onChange={(e) => setSaveForm({ ...saveForm, visibility: e.target.value })}
              >
                <Radio value="private">私有（仅本人可见）</Radio>
                <Tooltip title={isSuperAdmin ? '' : '仅 super_admin 可创建公共模板'}>
                  <Radio value="public" disabled={!isSuperAdmin}>
                    公共（所有人可见）
                  </Radio>
                </Tooltip>
              </Radio.Group>
            </Form.Item>
            <Form.Item label="描述">
              <Input.TextArea
                rows={3}
                value={saveForm.description}
                onChange={(e) => setSaveForm({ ...saveForm, description: e.target.value })}
                maxLength={500}
              />
            </Form.Item>
          </Form>
        </Modal>
      </div>
    </TreeListPageLayout>
  );
}
