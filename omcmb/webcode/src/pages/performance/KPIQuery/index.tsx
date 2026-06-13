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
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useIntl } from 'react-intl';
import { useT } from '@/hooks/useT';
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

const GRANULARITY_OPTIONS: { labelKey: string; value: Granularity }[] = [
  { labelKey: 'perf.dashboard.granular15min', value: '15min' },
  { labelKey: 'perf.dashboard.granularHourly', value: 'hourly' },
  { labelKey: 'perf.dashboard.granularDaily', value: 'daily' },
  { labelKey: 'perf.dashboard.granularWeekly', value: 'weekly' },
  { labelKey: 'perf.dashboard.granularMonthly', value: 'monthly' },
];

const TIME_RANGE_OPTIONS: { labelKey: string; value: TimeRangePreset }[] = [
  { labelKey: 'perf.kpiQuery.range.last1h', value: 'last_1h' },
  { labelKey: 'perf.kpiQuery.range.last24h', value: 'last_24h' },
  { labelKey: 'perf.kpiQuery.range.last7d', value: 'last_7d' },
  { labelKey: 'perf.kpiQuery.range.last30d', value: 'last_30d' },
  { labelKey: 'perf.kpiQuery.range.custom', value: 'custom' },
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
  const t = useT();

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
    () => (templatesData?.items ?? []).filter((tpl) => tpl.visibility === 'public'),
    [templatesData],
  );
  const privateTemplates = useMemo(
    () => (templatesData?.items ?? []).filter((tpl) => tpl.visibility === 'private'),
    [templatesData],
  );

  const granularityOptions = useMemo(
    () => GRANULARITY_OPTIONS.map((o) => ({ label: t(o.labelKey), value: o.value })),
    [t],
  );
  const timeRangeOptions = useMemo(
    () => TIME_RANGE_OPTIONS.map((o) => ({ label: t(o.labelKey), value: o.value })),
    [t],
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
      message.error(t('perf.kpiQuery.queryFailed', { msg: first?.message ?? t('perf.kpiQuery.unknownError') }));
    }
  }, [aggErrors, message, t]);

  // ── 行为 ─────────────────────────────────────────────────────────
  const handleQuery = () => {
    if (payload.deviceSns.length === 0) {
      message.warning(t('perf.kpiQuery.selectDeviceRequired'));
      return;
    }
    if (payload.metricPaths.length === 0) {
      message.warning(t('perf.kpiQuery.selectMetricRequired'));
      return;
    }
    let range: { start: string; end: string } | null;
    if (payload.timeRangePreset === 'custom') {
      if (!customRange) {
        message.warning(t('perf.kpiQuery.selectCustomRangeRequired'));
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
      message.warning(t('perf.kpiQuery.selectRangeRequired'));
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
        taskName: t('perf.kpiQuery.exportTaskName', { ts }), // 与仪表盘导出区分，便于任务列表辨识
      },
      {
        onSuccess: () => message.success(t('perf.kpiQuery.exportSubmitted')),
        onError: (e) => message.error(t('perf.kpiQuery.exportFailed', { msg: (e as Error)?.message ?? t('perf.kpiQuery.unknownError') })),
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
      message.warning(t('perf.kpiQuery.ambiguousMetrics', { list: ambiguous.join(', ') }));
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
      message.warning(t('perf.kpiQuery.templateNameRequired'));
      return;
    }
    if (saveForm.payload.timeRangePreset === 'custom' && !saveForm.customRange) {
      message.warning(t('perf.kpiQuery.selectCustomRangeRequired'));
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
        message.success(t('perf.kpiQuery.templateCreated'));
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
        message.success(t('perf.kpiQuery.templateUpdated'));
      }
      setSaveForm((s) => ({ ...s, open: false }));
    } catch (err) {
      const e = err as Error & { response?: { status?: number } };
      if (e?.response?.status === 409) {
        message.error(t('perf.kpiQuery.nameConflict'));
      } else if (e?.response?.status === 403) {
        message.error(t('perf.kpiQuery.noPermissionPublic'));
      } else {
        message.error(t('perf.kpiQuery.saveFailed', { msg: e?.message ?? t('perf.kpiQuery.unknownError') }));
      }
    }
  };

  const handleDeleteTemplate = async (id: string) => {
    try {
      await deleteMut.mutateAsync(id);
      message.success(t('perf.kpiQuery.templateDeleted'));
      if (activeTemplateId === id) {
        setActiveTemplateId(undefined);
      }
    } catch (err) {
      const e = err as Error;
      message.error(t('perf.kpiQuery.deleteFailed', { msg: e?.message ?? t('perf.kpiQuery.unknownError') }));
    }
  };

  // 「已选 N 个」摘要：KPI 用友好名（metricLabels）替代编号显示。
  const metricSummary = (paths: string[], head: number): string =>
    paths.length === 0
      ? ''
      : t('perf.kpiQuery.selectedSummary', {
          count: paths.length,
          items: paths
            .slice(0, head)
            .map((p) => metricLabels[p] ?? p)
            .join(', ') + (paths.length > head ? ' ...' : ''),
        });

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
                <Tooltip key="edit" title={t('common.edit')}>
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
                  title={t('perf.kpiQuery.confirmDeleteTemplate')}
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
                <Tag color="blue">{t('perf.kpiQuery.public')}</Tag>
              ) : (
                <Tag color="default">{t('perf.kpiQuery.private')}</Tag>
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

  // 左侧模板栏折叠态（仅本页、仅当前会话内，不记忆；默认展开）。
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  const sidebar = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 16px', borderBottom: `1px solid ${token.colorBorderSecondary}` }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }}>
          <Title level={5} style={{ margin: 0 }}>
            {t('perf.kpiQuery.queryTemplates')}
          </Title>
          <Space size={2}>
            <Tooltip title={t('perf.kpiQuery.newTemplateTip')}>
              <Button
                type="text"
                size="small"
                icon={<PlusOutlined />}
                onClick={handleOpenSaveModal}
              />
            </Tooltip>
            <Tooltip title={t('perf.kpiQuery.refreshList')}>
              <Button
                type="text"
                size="small"
                icon={<ReloadOutlined />}
                onClick={() => void refetchTemplates()}
              />
            </Tooltip>
            <Tooltip title={t('perf.kpiQuery.collapseSidebar')}>
              <Button
                type="text"
                size="small"
                icon={<MenuFoldOutlined />}
                onClick={() => setSidebarCollapsed(true)}
                aria-label={t('perf.kpiQuery.collapseSidebar')}
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
                <TeamOutlined /> {t('perf.kpiQuery.public')} ({publicTemplates.length})
              </span>
            ),
            children: (
              <Spin spinning={templatesLoading}>
                <List
                  dataSource={publicTemplates}
                  renderItem={renderTemplateItem}
                  locale={{ emptyText: <Empty description={t('perf.kpiQuery.noPublicTemplates')} /> }}
                />
              </Spin>
            ),
          },
          {
            key: 'private',
            label: (
              <span>
                <UserOutlined /> {t('perf.kpiQuery.private')} ({privateTemplates.length})
              </span>
            ),
            children: (
              <Spin spinning={templatesLoading}>
                <List
                  dataSource={privateTemplates}
                  renderItem={renderTemplateItem}
                  locale={{ emptyText: <Empty description={t('perf.kpiQuery.noPrivateTemplates')} /> }}
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
    // 内联两栏布局（替代公共 TreeListPageLayout，仅本页）：左栏可折叠成瘦条，右栏被挤宽。
    // 视觉对齐原公共组件（bg/圆角6/边框/gap15），不动公共组件、其它页零影响。
    <div style={{ display: 'flex', width: '100%', height: '100%', overflow: 'hidden', gap: 15 }}>
      {sidebarCollapsed ? (
        // 折叠态：整条可点的瘦竖条 + 展开图标 + 竖排标题（与性能仪表盘任务栏收起样式统一）。
        <Tooltip title={t('perf.kpiQuery.expandSidebar')} placement="right">
          <div
            onClick={() => setSidebarCollapsed(false)}
            style={{
              width: 36,
              flexShrink: 0,
              cursor: 'pointer',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 8,
              paddingTop: 8,
              border: `1px solid ${token.colorBorderSecondary}`,
              borderRadius: token.borderRadiusLG,
              background: token.colorBgContainer,
            }}
          >
            <MenuUnfoldOutlined style={{ color: token.colorPrimary }} />
            <span
              style={{
                fontSize: 12,
                color: token.colorTextSecondary,
                writingMode: 'vertical-rl',
                letterSpacing: 2,
              }}
            >
              {t('perf.kpiQuery.queryTemplates')}
            </span>
          </div>
        </Tooltip>
      ) : (
        <div
          style={{
            width: 280,
            flexShrink: 0,
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
            background: token.colorBgContainer,
            borderRadius: 6,
            border: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          {sidebar}
        </div>
      )}
      <div style={{ flex: 1, minWidth: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: 16, height: '100%', overflow: 'auto' }}>
        <Card
          size="small"
          style={{ marginBottom: 12 }}
          title={
            <Space>
              <TableOutlined />
              <span>{t('perf.kpiQuery.queryConditions')}</span>
            </Space>
          }
        >
          <Form layout="vertical" size="middle">
            <Space wrap size="middle" align="start">
              <Form.Item label={t('perf.kpiQuery.deviceType')} style={{ marginBottom: 0 }}>
                <Select
                  style={{ width: 120 }}
                  value={payload.deviceType}
                  onChange={(v) => setPayload({ ...payload, deviceType: v })}
                  options={DEVICE_TYPE_OPTIONS}
                />
              </Form.Item>

              <Form.Item label={t('perf.kpiQuery.device')} style={{ marginBottom: 0 }}>
                <Space.Compact style={{ width: 360 }}>
                  <Input
                    readOnly
                    value={
                      payload.deviceSns.length === 0
                        ? ''
                        : t('perf.kpiQuery.selectedSummary', {
                            count: payload.deviceSns.length,
                            items: payload.deviceSns.slice(0, 2).join(', ') + (payload.deviceSns.length > 2 ? ' ...' : ''),
                          })
                    }
                    placeholder={t('perf.kpiQuery.selectDevicePlaceholder')}
                  />
                  <Button onClick={() => { setPickerTarget('main'); setDevicePickerOpen(true); }}>{t('perf.kpiQuery.pickFromList')}</Button>
                </Space.Compact>
              </Form.Item>

              <Form.Item label={t('perf.kpiQuery.metric')} style={{ marginBottom: 0 }}>
                <Space.Compact style={{ width: 360 }}>
                  <Input
                    readOnly
                    value={metricSummary(payload.metricPaths, 2)}
                    placeholder={t('perf.kpiQuery.selectMetricPlaceholder')}
                  />
                  <Button onClick={() => { setPickerTarget('main'); setMetricPickerOpen(true); }}>{t('perf.kpiQuery.pickFromList')}</Button>
                </Space.Compact>
              </Form.Item>

              <Form.Item label={t('perf.granularity')} style={{ marginBottom: 0 }}>
                <Radio.Group
                  value={payload.granularity}
                  onChange={(e) => setPayload({ ...payload, granularity: e.target.value })}
                  options={granularityOptions}
                  optionType="button"
                  buttonStyle="solid"
                />
              </Form.Item>

              <Form.Item label={t('perf.kpiQuery.timeRange')} style={{ marginBottom: 0 }}>
                <Space>
                  <Select
                    style={{ width: 140 }}
                    value={payload.timeRangePreset}
                    onChange={(v) => setPayload({ ...payload, timeRangePreset: v })}
                    options={timeRangeOptions}
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
                  {t('common.query')}
                </Button>
                <Button icon={<ReloadOutlined />} onClick={() => void refetchAgg()} disabled={!submittedPayload}>
                  {t('common.refresh')}
                </Button>
                <Button icon={<SaveOutlined />} onClick={handleOpenSaveModal}>
                  {t('perf.kpiQuery.saveAsTemplate')}
                </Button>
                <Button
                  icon={<ExportOutlined />}
                  onClick={handleExport}
                  loading={createExport.isPending}
                  disabled={!submittedPayload || aggFetching}
                >
                  {t('perf.kpiQuery.exportCsv')}
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
                  {t('common.reset')}
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

        <Card size="small" title={<span><TableOutlined /> {t('perf.kpiQuery.queryResults')}</span>}>
          <PivotTable
            rows={aggregatedRows}
            loading={aggLoading || aggFetching}
            // gNB 查空时给更明确的引导（#201）：5G 真机样本厂商错配会让 KPI 算不出、
            // 后端返回 items=null，泛化「暂无数据」无法区分「指标库未注册」与「时段无采样」。
            // 仅在已发起查询（submittedPayload 存在）且制式=gNB 时替换文案。
            emptyDescription={
              submittedPayload?.deviceType === 'GNB' ? (
                <Text type="secondary">
                  {t('perf.kpiQuery.gnbEmptyHint')}
                </Text>
              ) : undefined
            }
          />
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
          title={saveForm.mode === 'create' ? t('perf.kpiQuery.newTemplate') : t('perf.kpiQuery.editTemplate')}
          open={saveForm.open}
          onCancel={() => setSaveForm((s) => ({ ...s, open: false }))}
          onOk={handleSaveTemplate}
          confirmLoading={createMut.isPending || updateMut.isPending}
          okText={t('common.save')}
          cancelText={t('common.cancel')}
          width={720}
          destroyOnHidden
        >
          <Form layout="vertical">
            <Form.Item label={t('perf.kpiQuery.templateName')} required>
              <Input
                value={saveForm.name}
                onChange={(e) => setSaveForm({ ...saveForm, name: e.target.value })}
                maxLength={128}
                placeholder={t('perf.kpiQuery.templateNamePlaceholder')}
              />
            </Form.Item>
            <Form.Item label={t('perf.kpiQuery.visibility')}>
              <Radio.Group
                value={saveForm.visibility}
                onChange={(e) => setSaveForm({ ...saveForm, visibility: e.target.value })}
              >
                <Radio value="private">{t('perf.kpiQuery.privateOption')}</Radio>
                <Tooltip title={isSuperAdmin ? '' : t('perf.kpiQuery.onlySuperAdminPublic')}>
                  <Radio value="public" disabled={!isSuperAdmin}>
                    {t('perf.kpiQuery.publicOption')}
                  </Radio>
                </Tooltip>
              </Radio.Group>
            </Form.Item>
            <Form.Item label={t('perf.kpiQuery.description')}>
              <Input.TextArea
                rows={2}
                value={saveForm.description}
                onChange={(e) => setSaveForm({ ...saveForm, description: e.target.value })}
                maxLength={500}
              />
            </Form.Item>

            <Divider titlePlacement="left" style={{ margin: '8px 0 16px' }}>
              {t('perf.kpiQuery.queryConfig')}
            </Divider>

            <Space wrap size="middle" align="start" style={{ width: '100%' }}>
              <Form.Item label={t('perf.kpiQuery.deviceType')} style={{ marginBottom: 8 }}>
                <Select
                  style={{ width: 120 }}
                  value={saveForm.payload.deviceType}
                  onChange={(v) =>
                    setSaveForm((s) => ({ ...s, payload: { ...s.payload, deviceType: v } }))
                  }
                  options={DEVICE_TYPE_OPTIONS}
                />
              </Form.Item>

              <Form.Item label={t('perf.granularity')} style={{ marginBottom: 8 }}>
                <Select
                  style={{ width: 110 }}
                  value={saveForm.payload.granularity}
                  onChange={(v) =>
                    setSaveForm((s) => ({ ...s, payload: { ...s.payload, granularity: v } }))
                  }
                  options={granularityOptions}
                />
              </Form.Item>

              <Form.Item label={t('perf.kpiQuery.timeRange')} style={{ marginBottom: 8 }}>
                <Space>
                  <Select
                    style={{ width: 140 }}
                    value={saveForm.payload.timeRangePreset}
                    onChange={(v) =>
                      setSaveForm((s) => ({ ...s, payload: { ...s.payload, timeRangePreset: v } }))
                    }
                    options={timeRangeOptions}
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

            <Form.Item label={t('perf.kpiQuery.device')} style={{ marginBottom: 8 }}>
              <Space.Compact style={{ width: '100%' }}>
                <Input
                  readOnly
                  value={
                    saveForm.payload.deviceSns.length === 0
                      ? ''
                      : t('perf.kpiQuery.selectedSummary', {
                          count: saveForm.payload.deviceSns.length,
                          items: saveForm.payload.deviceSns.slice(0, 3).join(', ') + (saveForm.payload.deviceSns.length > 3 ? ' ...' : ''),
                        })
                  }
                  placeholder={t('perf.kpiQuery.selectDevicePlaceholder')}
                />
                <Button
                  onClick={() => {
                    setPickerTarget('modal');
                    setDevicePickerOpen(true);
                  }}
                >
                  {t('perf.kpiQuery.pickFromList')}
                </Button>
              </Space.Compact>
            </Form.Item>

            <Form.Item label={t('perf.kpiQuery.metric')} style={{ marginBottom: 0 }}>
              <Space.Compact style={{ width: '100%' }}>
                <Input
                  readOnly
                  value={metricSummary(saveForm.payload.metricPaths, 3)}
                  placeholder={t('perf.kpiQuery.selectMetricPlaceholder')}
                />
                <Button
                  onClick={() => {
                    setPickerTarget('modal');
                    setMetricPickerOpen(true);
                  }}
                >
                  {t('perf.kpiQuery.pickFromList')}
                </Button>
              </Space.Compact>
            </Form.Item>
          </Form>
        </Modal>
      </div>
      </div>
    </div>
  );
}
