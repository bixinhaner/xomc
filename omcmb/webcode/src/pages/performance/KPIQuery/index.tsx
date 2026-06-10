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
  Alert,
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
  Divider,
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
import { useIntl } from 'react-intl';
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
import { useCreateKpiExport } from '@core/hooks/api/useKpiExport';
import {
  kpiQueryToDashboardSelection,
  buildDashboardExportParams,
} from '@core/utils/kpiExportParams';
import { indicatorLibraryApi } from '@core/services/api/indicatorLibraryApi';
import type { DeviceType } from '@core/types/indicatorLibrary';
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
  payload: QueryTemplatePayload;
  customRange: [dayjs.Dayjs, dayjs.Dayjs] | null;
}

// KPI 编号格式（如 K900010043）。
// 指标编号（落库即编号化后，counter 与 KPI 的 metric_path 都是编号）：
//   K 编号 K\d+ / C 编号 C\d+ / 5G·2G 编号 KGNB\d+·KGSM\d+。标准名一律带点（OTHER.CellServiceTime），
//   不会被此正则误判为编号，故编号原样短路、点分名走下方解析。
const KPI_CODE_RE = /^(C\d+|KGNB\d+|KGSM\d+|K\d+)$/;

// 模板历史兼容：旧模板里 KPI 存的是显示名（落库改编号前），新链路按编号过滤会查空。
// 载入时把非编号项尝试映射回编号——按显示名在指标库找唯一 KPI 项则换成其编号；
// counter（点分名 = metric_path）与查不到的项原样保留；同名无法唯一映射的项记入 ambiguous 提示重选。
async function resolveTemplateMetricPaths(
  deviceType: DeviceType,
  paths: string[],
): Promise<{ paths: string[]; labels: Record<string, string>; ambiguous: string[] }> {
  const outPaths: string[] = [];
  const labels: Record<string, string> = {};
  const ambiguous: string[] = [];
  for (const p of paths) {
    if (KPI_CODE_RE.test(p)) {
      outPaths.push(p);
      continue;
    }
    let exact: Awaited<ReturnType<typeof indicatorLibraryApi.list>>['items'] = [];
    try {
      const { items } = await indicatorLibraryApi.list(deviceType, { keyword: p, pageSize: 50 });
      exact = items.filter((it) => it.enName === p || it.cnName === p);
    } catch {
      // 查询失败时原样保留，不阻断模板载入
      outPaths.push(p);
      continue;
    }
    const kpiMatches = exact.filter((it) => !it.isCounter);
    if (kpiMatches.length === 1) {
      const m = kpiMatches[0];
      outPaths.push(m.id);
      labels[m.id] = m.cnName || m.enName || m.id;
    } else if (kpiMatches.length > 1) {
      ambiguous.push(p);
      outPaths.push(p);
    } else {
      // 走到这里的 p 已非编号（编号在上方短路）：旧模板里残留的点分上报名，或查不到。
      // 原样保留——编号化后这类点分名 counter 查不到数据，仅兜底不阻断模板载入。
      outPaths.push(p);
      const counter = exact.find((it) => it.isCounter);
      if (counter) labels[p] = counter.cnName || p;
    }
  }
  return { paths: outPaths, labels, ambiguous };
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
  // 指标选中值（KPI=编号）→ 友好名，供「已选 N 个」摘要展示，避免露出 K 编号。
  const [metricLabels, setMetricLabels] = useState<Record<string, string>>({});
  const [devicePickerOpen, setDevicePickerOpen] = useState(false);
  const [metricPickerOpen, setMetricPickerOpen] = useState(false);
  // 设备/指标选择器的目标：'main' = 主查询表单；'modal' = 模板编辑 Modal
  const [pickerTarget, setPickerTarget] = useState<'main' | 'modal'>('main');

  const intl = useIntl();

  // ── 模板侧栏状态 ─────────────────────────────────────────────────
  const [templateTab, setTemplateTab] = useState<'public' | 'private'>('public');
  const [activeTemplateId, setActiveTemplateId] = useState<string | undefined>(undefined);

  const { data: templatesData, isLoading: templatesLoading, refetch: refetchTemplates } =
    useQueryTemplates({ pageSize: 200 });
  const createMut = useCreateQueryTemplate();
  const updateMut = useUpdateQueryTemplate();
  const deleteMut = useDeleteQueryTemplate();
  const createExport = useCreateKpiExport();

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
    payload: DEFAULT_PAYLOAD,
    customRange: null,
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
      // 让后端按 (时间桶 × 指标) 补齐占位行，避免该设备此时段全空时整张表"暂无数据"
      fillEmpty: true,
    };
  }, [submittedPayload, submittedRange]);

  const {
    data: aggregatedRows,
    total: aggTotal,
    truncated: aggTruncated,
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

  // 导出取「最近一次实际查询」的快照（submittedPayload/submittedRange），而非表单实时值，
  // 保证"导出=屏幕所见"。复用 dashboard 异步导出链路：建任务 → 文件管理下载，后端零改。
  const handleExport = () => {
    if (!submittedPayload || !submittedRange) return; // 按钮已禁用，双保险
    const sel = kpiQueryToDashboardSelection(submittedPayload, submittedRange);
    const ts = dayjs().format('YYYYMMDD_HHmmss');
    createExport.mutate(
      {
        sourceType: 'dashboard',
        params: buildDashboardExportParams(sel),
        taskName: `KPI导出_指标查询_${ts}`, // 与仪表盘导出区分，便于任务列表辨识
      },
      {
        onSuccess: () => message.success('导出任务已提交，请到「文件管理」下载'),
        onError: (e) => message.error(`导出提交失败：${(e as Error)?.message ?? '未知错误'}`),
      },
    );
  };

  const handleSelectTemplate = async (tpl: QueryTemplate) => {
    setActiveTemplateId(tpl.id);
    if (tpl.payload.timeRangePreset === 'custom' && tpl.payload.absoluteStart && tpl.payload.absoluteEnd) {
      setCustomRange([dayjs(tpl.payload.absoluteStart), dayjs(tpl.payload.absoluteEnd)]);
    } else {
      setCustomRange(null);
    }
    // 历史兼容：旧模板里 KPI 可能存的是显示名，映射回编号后再回填表单
    const dt = (tpl.payload.deviceType ?? 'ENB') as DeviceType;
    const { paths, labels, ambiguous } = await resolveTemplateMetricPaths(dt, tpl.payload.metricPaths);
    setMetricLabels((prev) => ({ ...prev, ...labels }));
    setPayload({ ...tpl.payload, metricPaths: paths });
    if (ambiguous.length > 0) {
      message.warning(`模板中以下指标存在同名、无法唯一确定，请在「指标」里重新选择：${ambiguous.join('、')}`);
    }
  };

  const handleOpenSaveModal = () => {
    // 从主表单复制当前条件作为初值（即"存为模板"工作流）；从侧栏 + 新建也走这里，复用主表单 default
    setSaveForm({
      open: true,
      mode: 'create',
      name: '',
      description: '',
      visibility: 'private',
      payload: { ...payload },
      customRange,
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
      payload: tpl.payload,
      customRange:
        tpl.payload.timeRangePreset === 'custom' && tpl.payload.absoluteStart && tpl.payload.absoluteEnd
          ? [dayjs(tpl.payload.absoluteStart), dayjs(tpl.payload.absoluteEnd)]
          : null,
    });
  };

  const handleSaveTemplate = async () => {
    if (!saveForm.name.trim()) {
      message.warning('请输入模板名称');
      return;
    }
    if (saveForm.payload.timeRangePreset === 'custom' && !saveForm.customRange) {
      message.warning('请选择自定义时间范围');
      return;
    }
    // 保存时回填 custom 模式的绝对时间（使用 Modal 内部的 payload + customRange，不是主表单）
    const payloadToSave: QueryTemplatePayload = {
      ...saveForm.payload,
      absoluteStart:
        saveForm.payload.timeRangePreset === 'custom' && saveForm.customRange
          ? saveForm.customRange[0].toISOString()
          : undefined,
      absoluteEnd:
        saveForm.payload.timeRangePreset === 'custom' && saveForm.customRange
          ? saveForm.customRange[1].toISOString()
          : undefined,
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

  // 「已选 N 个」摘要：KPI 用友好名（metricLabels）替代编号显示。
  const metricSummary = (paths: string[], head: number): string =>
    paths.length === 0
      ? ''
      : `已选 ${paths.length} 个：${paths
          .slice(0, head)
          .map((p) => metricLabels[p] ?? p)
          .join(', ')}${paths.length > head ? ' ...' : ''}`;

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
        onClick={() => void handleSelectTemplate(tpl)}
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
          <Space size={2}>
            <Tooltip title="新建模板（用当前查询条件，未填则后续可补）">
              <Button
                type="text"
                size="small"
                icon={<PlusOutlined />}
                onClick={handleOpenSaveModal}
              />
            </Tooltip>
            <Tooltip title="刷新列表">
              <Button
                type="text"
                size="small"
                icon={<ReloadOutlined />}
                onClick={() => void refetchTemplates()}
              />
            </Tooltip>
          </Space>
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
                  <Button onClick={() => { setPickerTarget('main'); setDevicePickerOpen(true); }}>列表选</Button>
                </Space.Compact>
              </Form.Item>

              <Form.Item label="指标" style={{ marginBottom: 0 }}>
                <Space.Compact style={{ width: 360 }}>
                  <Input
                    readOnly
                    value={metricSummary(payload.metricPaths, 2)}
                    placeholder="点击右侧按钮选择指标"
                  />
                  <Button onClick={() => { setPickerTarget('main'); setMetricPickerOpen(true); }}>列表选</Button>
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
                <Button
                  icon={<ExportOutlined />}
                  onClick={handleExport}
                  loading={createExport.isPending}
                  disabled={!submittedPayload || aggFetching}
                >
                  导出 CSV
                </Button>
                <Button
                  icon={<PlusOutlined />}
                  onClick={() => {
                    setPayload(DEFAULT_PAYLOAD);
                    setCustomRange(null);
                    setActiveTemplateId(undefined);
                    setSubmittedPayload(null);
                    setSubmittedRange(null);
                  }}
                >
                  重置
                </Button>
              </Space>
            </div>
          </Form>
        </Card>

        {aggTruncated ? (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 12 }}
            message={intl.formatMessage(
              { id: 'perf.dashboard.truncatedTip' },
              { shown: aggregatedRows.length, total: aggTotal },
            )}
          />
        ) : null}

        <Card size="small" title={<span><TableOutlined /> 查询结果</span>}>
          <PivotTable rows={aggregatedRows} loading={aggLoading || aggFetching} />
        </Card>

        <DevicePickerModal
          open={devicePickerOpen}
          onClose={() => setDevicePickerOpen(false)}
          onConfirm={(sns) => {
            if (pickerTarget === 'modal') {
              setSaveForm((s) => ({ ...s, payload: { ...s.payload, deviceSns: sns } }));
            } else {
              setPayload({ ...payload, deviceSns: sns });
            }
          }}
          initialSelected={pickerTarget === 'modal' ? saveForm.payload.deviceSns : payload.deviceSns}
        />

        <MetricPickerModal
          open={metricPickerOpen}
          onClose={() => setMetricPickerOpen(false)}
          onConfirm={(paths, labels) => {
            setMetricLabels((prev) => ({ ...prev, ...labels }));
            if (pickerTarget === 'modal') {
              setSaveForm((s) => ({ ...s, payload: { ...s.payload, metricPaths: paths } }));
            } else {
              setPayload({ ...payload, metricPaths: paths });
            }
          }}
          initialSelected={pickerTarget === 'modal' ? saveForm.payload.metricPaths : payload.metricPaths}
          initialDeviceType={
            (pickerTarget === 'modal' ? saveForm.payload.deviceType : payload.deviceType) ?? 'ENB'
          }
        />

        <Modal
          title={saveForm.mode === 'create' ? '新建查询模板' : '编辑查询模板'}
          open={saveForm.open}
          onCancel={() => setSaveForm((s) => ({ ...s, open: false }))}
          onOk={handleSaveTemplate}
          confirmLoading={createMut.isPending || updateMut.isPending}
          okText="保存"
          cancelText="取消"
          width={720}
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
                rows={2}
                value={saveForm.description}
                onChange={(e) => setSaveForm({ ...saveForm, description: e.target.value })}
                maxLength={500}
              />
            </Form.Item>

            <Divider titlePlacement="left" style={{ margin: '8px 0 16px' }}>
              查询配置
            </Divider>

            <Space wrap size="middle" align="start" style={{ width: '100%' }}>
              <Form.Item label="设备类型" style={{ marginBottom: 8 }}>
                <Select
                  style={{ width: 120 }}
                  value={saveForm.payload.deviceType}
                  onChange={(v) =>
                    setSaveForm((s) => ({ ...s, payload: { ...s.payload, deviceType: v } }))
                  }
                  options={DEVICE_TYPE_OPTIONS}
                />
              </Form.Item>

              <Form.Item label="粒度" style={{ marginBottom: 8 }}>
                <Select
                  style={{ width: 110 }}
                  value={saveForm.payload.granularity}
                  onChange={(v) =>
                    setSaveForm((s) => ({ ...s, payload: { ...s.payload, granularity: v } }))
                  }
                  options={GRANULARITY_OPTIONS}
                />
              </Form.Item>

              <Form.Item label="时间范围" style={{ marginBottom: 8 }}>
                <Space>
                  <Select
                    style={{ width: 140 }}
                    value={saveForm.payload.timeRangePreset}
                    onChange={(v) =>
                      setSaveForm((s) => ({ ...s, payload: { ...s.payload, timeRangePreset: v } }))
                    }
                    options={TIME_RANGE_OPTIONS}
                  />
                  {saveForm.payload.timeRangePreset === 'custom' && (
                    <RangePicker
                      showTime
                      value={saveForm.customRange}
                      onChange={(v) =>
                        setSaveForm((s) => ({
                          ...s,
                          customRange: v as [dayjs.Dayjs, dayjs.Dayjs] | null,
                        }))
                      }
                    />
                  )}
                </Space>
              </Form.Item>
            </Space>

            <Form.Item label="设备" style={{ marginBottom: 8 }}>
              <Space.Compact style={{ width: '100%' }}>
                <Input
                  readOnly
                  value={
                    saveForm.payload.deviceSns.length === 0
                      ? ''
                      : `已选 ${saveForm.payload.deviceSns.length} 个：${saveForm.payload.deviceSns.slice(0, 3).join(', ')}${saveForm.payload.deviceSns.length > 3 ? ' ...' : ''}`
                  }
                  placeholder="点击右侧按钮选择设备"
                />
                <Button
                  onClick={() => {
                    setPickerTarget('modal');
                    setDevicePickerOpen(true);
                  }}
                >
                  列表选
                </Button>
              </Space.Compact>
            </Form.Item>

            <Form.Item label="指标" style={{ marginBottom: 0 }}>
              <Space.Compact style={{ width: '100%' }}>
                <Input
                  readOnly
                  value={metricSummary(saveForm.payload.metricPaths, 3)}
                  placeholder="点击右侧按钮选择指标"
                />
                <Button
                  onClick={() => {
                    setPickerTarget('modal');
                    setMetricPickerOpen(true);
                  }}
                >
                  列表选
                </Button>
              </Space.Compact>
            </Form.Item>
          </Form>
        </Modal>
      </div>
    </TreeListPageLayout>
  );
}
