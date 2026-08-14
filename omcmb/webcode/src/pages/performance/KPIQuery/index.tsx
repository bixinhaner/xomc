/**
 * T-0174 指标查询页（重做版）。
 *
 * 旧版 1500 行 mock 全部删除，替换为真后端对接：
 *   - 模板侧栏 → /api/v1/pm/query-templates
 *   - 设备/指标选择 → DevicePickerModal / MetricPickerModal
 *   - 查询 → /api/v1/pm/metrics/aggregated（多设备并发后合并）
 *   - 结果 → 透视表（行=时间 / 列=N 指标 / 单元格=值）
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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
  EyeOutlined,
  TeamOutlined,
  UserOutlined,
  ExportOutlined,
  TableOutlined,
  ClockCircleOutlined,
  FullscreenExitOutlined,
  FullscreenOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  BellOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useIntl } from 'react-intl';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { queryClient } from '@/providers/queryClient';
import {
  useQueryTemplates,
  useCreateQueryTemplate,
  useUpdateQueryTemplate,
  useDeleteQueryTemplate,
  useAggregatedMetricsByDevices,
  useMetricObjectsByDevices,
} from '@core/hooks/api/usePmQuery';
import { useUserStore } from '@core/store/userStore';
import { usePmPageStateStore } from '@core/store/pmPageStateStore';
import { useSystemTimezoneValue } from '@core/hooks/api/useSystemTimezone';
import { toSystemTimezoneRFC3339 } from '@core/utils/systemTime';
import { useCreateKpiExport } from '@core/hooks/api/useKpiExport';
import {
  kpiQueryToDashboardSelection,
  buildDashboardExportParams,
} from '@core/utils/kpiExportParams';
import type { DeviceType } from '@core/types/indicatorLibrary';
import { deviceTypeToTechnology, useTechnologyDictionary } from '@core/hooks/api/useTechnologyDictionary';
import type { Granularity } from '@core/types/pmDashboard';
import type {
  QueryTemplate,
  QueryTemplatePayload,
  TemplateVisibility,
  TimeRangePreset,
} from '@core/types/pmQuery';
import { buildAlignedPresetRange, getDefaultTimeRangeForGranularity } from '@core/utils/granularityTimeRange';
import DevicePickerModal from './components/DevicePickerModal';
import MetricPickerModal from '@/components/MetricPickerModal';
import PivotTable from './components/PivotTable';
import CellDrilldownSelector from '../PmDashboard/CellDrilldownSelector';
import { getEffectiveLdnsWithNrRecommendedDefault, type CellSelection } from '../PmDashboard/cellDrilldownUtils';
import { synchronizeUpdatedTemplateState } from './templateUpdateState';
import QueryTemplateDetailModal from './QueryTemplateDetailModal';
import ReportSubscriptionModal from './ReportSubscriptionModal';
import { resolveTemplateMetricPaths } from './templateMetricResolver';
import { PM_QUERY_SELECTION_LIMIT } from '@/constants/pmQueryLimits';
import { usePermission } from '@core/hooks/usePermission';
import {
  buildKpiQueryStateSnapshot,
  buildKpiQuerySubmittedSnapshot,
  kpiQuerySignature,
  KPI_QUERY_DEFAULT_PIVOT_PAGE_SIZE,
  PM_KPI_QUERY_PAGE_KEY,
  restoredKpiQueryDelayMs,
  restoreKpiQueryState,
  type KpiQuerySubmittedQuery,
} from './kpiQueryState';

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
  { labelKey: 'perf.kpiQuery.range.last3h', value: 'last_3h' },
  { labelKey: 'perf.kpiQuery.range.last24h', value: 'last_24h' },
  { labelKey: 'perf.kpiQuery.range.last7d', value: 'last_7d' },
  { labelKey: 'perf.kpiQuery.range.last30d', value: 'last_30d' },
  { labelKey: 'perf.kpiQuery.range.last6m', value: 'last_6m' },
  { labelKey: 'perf.kpiQuery.range.custom', value: 'custom' },
];

const DEFAULT_PAYLOAD: QueryTemplatePayload = {
  deviceSns: [],
  metricPaths: [],
  granularity: '15min',
  // 与 #595 联动表保持一致：15min → 近 3 小时
  timeRangePreset: getDefaultTimeRangeForGranularity('15min'),
  deviceType: 'ENB',
};

function formatMetricDisplay(id: string, label?: string): string {
  return label && label !== id ? `${id} ${label}` : id;
}

const createDefaultTemplatePayload = (): QueryTemplatePayload => ({
  ...DEFAULT_PAYLOAD,
  deviceSns: [...DEFAULT_PAYLOAD.deviceSns],
  metricPaths: [...DEFAULT_PAYLOAD.metricPaths],
});

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

const createBlankSaveTemplateForm = (open: boolean): SaveTemplateFormState => ({
  open,
  mode: 'create',
  name: '',
  description: '',
  visibility: 'private',
  payload: createDefaultTemplatePayload(),
  customRange: null,
});

export default function KPIQuery() {
  const token = useThemeToken();
  const { message } = App.useApp();
  const currentUser = useUserStore((s) => s.currentUser);
  const isSuperAdmin = currentUser?.isSuperAdmin ?? false;
  const canQuery = usePermission('performance:query:query');
  const canAddTemplate = usePermission('performance:query:add');
  const canEditTemplate = usePermission('performance:query:edit');
  const canDeleteTemplate = usePermission('performance:query:delete');
  // #459 子单 D：自定义时间范围按系统时区附加偏移后再发后端（后端按 RFC3339 解析为 UTC）。
  const systemTimezone = useSystemTimezoneValue();
  const {
    deviceTypeOptions,
    isLoading: deviceTypeOptionsLoading,
  } = useTechnologyDictionary();
  const firstAvailableDeviceType = deviceTypeOptions[0]?.value;
  const hasAvailableDeviceTypes = deviceTypeOptions.length > 0;
  const availableDeviceTypes = useMemo(
    () => new Set(deviceTypeOptions.map((option) => option.value)),
    [deviceTypeOptions],
  );
  const isAvailableDeviceType = useCallback(
    (deviceType?: DeviceType): boolean => Boolean(deviceType && availableDeviceTypes.has(deviceType)),
    [availableDeviceTypes],
  );
  const restoredState = useMemo(
    () => restoreKpiQueryState(usePmPageStateStore.getState().getPageState(PM_KPI_QUERY_PAGE_KEY)),
    [],
  );
  const skipNextSaveRef = useRef(false);

  // ── 查询表单状态 ─────────────────────────────────────────────────
  const [storedPayload, setPayload] = useState<QueryTemplatePayload>(restoredState.payload);
  const payload = useMemo<QueryTemplatePayload>(() => {
    if (
      deviceTypeOptionsLoading ||
      !firstAvailableDeviceType ||
      isAvailableDeviceType(storedPayload.deviceType)
    ) {
      return storedPayload;
    }
    return {
      ...storedPayload,
      deviceType: firstAvailableDeviceType,
      deviceSns: [],
      metricPaths: [],
    };
  }, [deviceTypeOptionsLoading, firstAvailableDeviceType, isAvailableDeviceType, storedPayload]);
  const [customRange, setCustomRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(restoredState.customRange);
  // #595: 用户手动修改过时间范围后标记 dirty，粒度切换不再覆盖
  const [timeRangeDirty, setTimeRangeDirty] = useState(restoredState.timeRangeDirty);
  // #619：测量对象（小区）下钻选择，按设备勾选要查的小区子集。
  const [storedCellSel, setCellSel] = useState<CellSelection>(restoredState.cellSel);
  const cellSel = useMemo<CellSelection>(
    () => (payload === storedPayload ? storedCellSel : {}),
    [payload, storedCellSel, storedPayload],
  );
  // 指标选中值（KPI=编号）→ 友好名，供「已选 N 个」摘要展示，避免露出 K 编号。
  const [metricLabels, setMetricLabels] = useState<Record<string, string>>({});
  const [devicePickerOpen, setDevicePickerOpen] = useState(false);
  const [metricPickerOpen, setMetricPickerOpen] = useState(false);
  // 设备/指标选择器的目标：'main' = 主查询表单；'modal' = 模板编辑 Modal
  const [pickerTarget, setPickerTarget] = useState<'main' | 'modal'>('main');

  const intl = useIntl();
  const t = useT();

  // ── 模板侧栏状态 ─────────────────────────────────────────────────
  const [templateTab, setTemplateTab] = useState<'public' | 'private'>(restoredState.templateTab);
  const [activeTemplateId, setActiveTemplateId] = useState<string | undefined>(restoredState.activeTemplateId);
  const [detailTemplateId, setDetailTemplateId] = useState<string | undefined>(undefined);
  const [reportTemplateId, setReportTemplateId] = useState<string | undefined>(undefined);
  // 左侧模板栏折叠态也属于本页轻量现场；不保存模板列表结果。
  const [sidebarCollapsed, setSidebarCollapsed] = useState(restoredState.sidebarCollapsed);

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
  const detailTemplate = useMemo(
    () => (templatesData?.items ?? []).find((tpl) => tpl.id === detailTemplateId) ?? null,
    [detailTemplateId, templatesData],
  );
  const reportTemplate = useMemo(
    () => (templatesData?.items ?? []).find((tpl) => tpl.id === reportTemplateId) ?? null,
    [reportTemplateId, templatesData],
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
  const [storedSaveForm, setSaveForm] = useState<SaveTemplateFormState>(() => createBlankSaveTemplateForm(false));
  const saveForm = useMemo<SaveTemplateFormState>(() => {
    if (
      deviceTypeOptionsLoading ||
      !firstAvailableDeviceType ||
      isAvailableDeviceType(storedSaveForm.payload.deviceType)
    ) {
      return storedSaveForm;
    }
    return {
      ...storedSaveForm,
      payload: {
        ...storedSaveForm.payload,
        deviceType: firstAvailableDeviceType,
        deviceSns: [],
        metricPaths: [],
      },
    };
  }, [deviceTypeOptionsLoading, firstAvailableDeviceType, isAvailableDeviceType, storedSaveForm]);

  // ── 查询执行状态 ─────────────────────────────────────────────────
  // submitted 是真正用于查询的快照；表单编辑时不立即查询，等用户点"查询"
  const [submitted, setSubmitted] = useState<KpiQuerySubmittedQuery | null>(restoredState.submitted);
  const [pivotPage, setPivotPage] = useState(restoredState.pivotPage);
  const [pivotPageSize, setPivotPageSize] = useState(restoredState.pivotPageSize);
  const [resultsMaximized, setResultsMaximized] = useState(restoredState.resultsMaximized);
  const [initialRestoredQueryDelayMs] = useState(() => (
    restoredState.shouldRestoreQuery
      ? restoredKpiQueryDelayMs(restoredState.savedAt, Date.now())
      : 0
  ));
  const [resultsQueryReady, setResultsQueryReady] = useState(!restoredState.shouldRestoreQuery);

  useEffect(() => {
    if (!restoredState.shouldRestoreQuery) return undefined;
    const timer = window.setTimeout(() => {
      queryClient.removeQueries({ queryKey: ['pm-aggregated'] });
      setResultsQueryReady(true);
    }, initialRestoredQueryDelayMs);
    return () => window.clearTimeout(timer);
  }, [initialRestoredQueryDelayMs, restoredState.shouldRestoreQuery]);

  const currentTechnology = payload.deviceType ? deviceTypeToTechnology(payload.deviceType) : undefined;
  const { isLoading: currentObjectsLoading } = useMetricObjectsByDevices(payload.deviceSns, currentTechnology);

  // #619：计算已提交查询快照里的有效 object_ldn 白名单（全选/未选 = 空数组 = 不过滤）。
  // 不依赖实时编辑态，避免查询后继续改设备/小区时串改上次结果和导出条件。
  const { byDevice: submittedObjectsByDevice } = useMetricObjectsByDevices(
    submitted?.payload.deviceSns ?? [],
    submitted?.payload.deviceType ? deviceTypeToTechnology(submitted.payload.deviceType) : undefined,
  );
  const effectiveLdns = useMemo(
    () => getEffectiveLdnsWithNrRecommendedDefault(submitted?.cellSel ?? {}, submittedObjectsByDevice),
    [submitted, submittedObjectsByDevice],
  );

  const baseAggParams = useMemo(() => {
    if (!submitted) return null;
    if (!isAvailableDeviceType(submitted.payload.deviceType)) return null;
    return {
      granularity: submitted.payload.granularity,
      metricPaths: submitted.payload.metricPaths,
      startTime: submitted.range.start,
      endTime: submitted.range.end,
      limit: pivotPageSize,
      offset: (pivotPage - 1) * pivotPageSize,
      pageBy: 'pivot_row' as const,
      // 让后端按 (时间桶 × 指标) 补齐占位行，避免该设备此时段全空时整张表"暂无数据"
      fillEmpty: true,
      // #619：测量对象后端过滤（空 = 不过滤）。
      objectLdns: effectiveLdns.length > 0 ? effectiveLdns : undefined,
    };
  }, [effectiveLdns, isAvailableDeviceType, pivotPage, pivotPageSize, submitted]);

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
    submitted?.payload.deviceSns ?? [],
    Boolean(baseAggParams) && resultsQueryReady,
  );
  const restoreQueryPending = Boolean(submitted) && !resultsQueryReady;
  const displayedRows = restoreQueryPending ? [] : aggregatedRows;
  const displayedTotal = restoreQueryPending ? 0 : aggTotal;
  const displayedLoading = restoreQueryPending || aggLoading || aggFetching;

  useEffect(() => {
    if (aggErrors.length > 0) {
      const first = aggErrors[0] as Error;
      message.error(t('perf.kpiQuery.queryFailed', { msg: first?.message ?? t('perf.kpiQuery.unknownError') }));
    }
  }, [aggErrors, message, t]);

  useEffect(() => {
    if (skipNextSaveRef.current) {
      skipNextSaveRef.current = false;
      return;
    }
    usePmPageStateStore.getState().savePageState(
      PM_KPI_QUERY_PAGE_KEY,
      buildKpiQueryStateSnapshot({
        payload,
        customRange,
        timeRangeDirty,
        cellSel,
        submitted,
        pivotPage,
        pivotPageSize,
        templateTab,
        activeTemplateId,
        sidebarCollapsed,
        resultsMaximized,
      }),
    );
  }, [
    activeTemplateId,
    cellSel,
    customRange,
    payload,
    pivotPage,
    pivotPageSize,
    resultsMaximized,
    sidebarCollapsed,
    submitted,
    templateTab,
    timeRangeDirty,
  ]);

  // ── 行为 ─────────────────────────────────────────────────────────
  const isSelectionExceedsLimit = (target: QueryTemplatePayload): boolean =>
    target.deviceSns.length > PM_QUERY_SELECTION_LIMIT || target.metricPaths.length > PM_QUERY_SELECTION_LIMIT;

  const warnIfSelectionExceedsLimit = (target: QueryTemplatePayload): boolean => {
    if (target.deviceSns.length > PM_QUERY_SELECTION_LIMIT) {
      message.warning(t('perf.kpiQuery.deviceLimitExceeded', {
        max: PM_QUERY_SELECTION_LIMIT,
        count: target.deviceSns.length,
      }));
      return true;
    }
    if (target.metricPaths.length > PM_QUERY_SELECTION_LIMIT) {
      message.warning(t('perf.kpiQuery.metricLimitExceeded', {
        max: PM_QUERY_SELECTION_LIMIT,
        count: target.metricPaths.length,
      }));
      return true;
    }
    return false;
  };

  const handleQuery = () => {
    if (!hasAvailableDeviceTypes || !isAvailableDeviceType(payload.deviceType)) {
      return;
    }
    if (payload.deviceSns.length === 0) {
      message.warning(t('perf.kpiQuery.selectDeviceRequired'));
      return;
    }
    if (payload.metricPaths.length === 0) {
      message.warning(t('perf.kpiQuery.selectMetricRequired'));
      return;
    }
    if (currentObjectsLoading) {
      message.warning(t('perf.drilldown.loadingObjects'));
      return;
    }
    if (warnIfSelectionExceedsLimit(payload)) {
      return;
    }
    let range: { start: string; end: string } | null;
    if (payload.timeRangePreset === 'custom') {
      if (!customRange) {
        message.warning(t('perf.kpiQuery.selectCustomRangeRequired'));
        return;
      }
      range = {
        // 用户在选择器里看到/选的是系统时区钟面，按系统时区附加偏移后再发后端。
        start: toSystemTimezoneRFC3339(customRange[0], systemTimezone) ?? customRange[0].toISOString(),
        end: toSystemTimezoneRFC3339(customRange[1], systemTimezone) ?? customRange[1].toISOString(),
      };
    } else {
      range = buildAlignedPresetRange({
        granularity: payload.granularity,
        preset: payload.timeRangePreset,
        systemTimezone,
      });
    }
    if (!range) {
      message.warning(t('perf.kpiQuery.selectRangeRequired'));
      return;
    }
    const nextSubmitted = buildKpiQuerySubmittedSnapshot(payload, range, cellSel);
    const sameSubmitted = kpiQuerySignature(nextSubmitted) === kpiQuerySignature(submitted);
    setResultsQueryReady(true);
    setSubmitted(nextSubmitted);
    setPivotPage(1);
    // 「查询」兼并旧「刷新」按钮的强刷语义：同条件再次点击也强制重拉一次最新数据
    // （react-query 默认 30s staleTime，同 key 不会重发——这里显式 refetch 覆盖）。
    if (sameSubmitted) {
      void refetchAgg();
    }
  };

  // 导出取「最近一次实际查询」的筛选快照（submitted），而非表单实时值。
  // 表格已是后端分页，导出不带当前页 limit/offset，口径是当前筛选条件下的全量数据。
  // 复用 dashboard 取数链路，但用 kpi_query 来源输出查询页表格列。
  const handleExport = () => {
    if (!submitted) {
      void warnIfSelectionExceedsLimit(payload);
      return;
    }
    if (!isAvailableDeviceType(submitted.payload.deviceType)) return;
    if (warnIfSelectionExceedsLimit(submitted.payload)) {
      return;
    }
    const sel = {
      ...kpiQueryToDashboardSelection(submitted.payload, submitted.range),
      objectLdns: effectiveLdns.length > 0 ? effectiveLdns : undefined,
    };
    const ts = dayjs().format('YYYYMMDD_HHmmss');
    createExport.mutate(
      {
        sourceType: 'kpi_query',
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
    setTimeRangeDirty(false);
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

  const handleOpenCreateModal = () => {
    setSaveForm(createBlankSaveTemplateForm(true));
  };

  const handleOpenSaveAsModal = () => {
    // 从主表单复制当前条件作为初值（即"存为模板"工作流）。
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

  const handleOpenUpdateModal = async (tpl: QueryTemplate) => {
    const dt = (tpl.payload.deviceType ?? 'ENB') as DeviceType;
    const { paths, labels } = await resolveTemplateMetricPaths(dt, tpl.payload.metricPaths);
    setMetricLabels((prev) => ({ ...prev, ...labels }));
    setSaveForm({
      open: true,
      mode: 'update',
      templateId: tpl.id,
      name: tpl.name,
      description: tpl.description ?? '',
      visibility: tpl.visibility,
      payload: { ...tpl.payload, metricPaths: paths },
      customRange:
        tpl.payload.timeRangePreset === 'custom' && tpl.payload.absoluteStart && tpl.payload.absoluteEnd
          ? [dayjs(tpl.payload.absoluteStart), dayjs(tpl.payload.absoluteEnd)]
          : null,
    });
  };

  const handleOpenDetail = async (tpl: QueryTemplate) => {
    const dt = (tpl.payload.deviceType ?? 'ENB') as DeviceType;
    const { labels } = await resolveTemplateMetricPaths(dt, tpl.payload.metricPaths);
    setMetricLabels((prev) => ({ ...prev, ...labels }));
    setDetailTemplateId(tpl.id);
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
    if (warnIfSelectionExceedsLimit(saveForm.payload)) {
      return;
    }
    // 保存时回填 custom 模式的绝对时间（使用 Modal 内部的 payload + customRange，不是主表单）
    const payloadToSave: QueryTemplatePayload = {
      ...saveForm.payload,
      absoluteStart:
        saveForm.payload.timeRangePreset === 'custom' && saveForm.customRange
          ? toSystemTimezoneRFC3339(saveForm.customRange[0], systemTimezone) ?? saveForm.customRange[0].toISOString()
          : undefined,
      absoluteEnd:
        saveForm.payload.timeRangePreset === 'custom' && saveForm.customRange
          ? toSystemTimezoneRFC3339(saveForm.customRange[1], systemTimezone) ?? saveForm.customRange[1].toISOString()
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
        const updated = await updateMut.mutateAsync({
          id: saveForm.templateId,
          input: {
            name: saveForm.name.trim(),
            description: saveForm.description,
            visibility: saveForm.visibility,
            payload: payloadToSave,
          },
        });
        if (activeTemplateId === updated.id) {
          const dt = (updated.payload.deviceType ?? 'ENB') as DeviceType;
          const { paths, labels } = await resolveTemplateMetricPaths(dt, updated.payload.metricPaths);
          const next = synchronizeUpdatedTemplateState(
            activeTemplateId,
            { formPayload: payload, submittedPayload: submitted?.payload ?? null },
            updated,
            paths,
          );
          setMetricLabels((prev) => ({ ...prev, ...labels }));
          setPayload(next.formPayload);
          setSubmitted((current) =>
            current && next.submittedPayload ? { ...current, payload: next.submittedPayload } : current,
          );
          setTimeRangeDirty(false);
          if (
            updated.payload.timeRangePreset === 'custom'
            && updated.payload.absoluteStart
            && updated.payload.absoluteEnd
          ) {
            setCustomRange([dayjs(updated.payload.absoluteStart), dayjs(updated.payload.absoluteEnd)]);
          } else {
            setCustomRange(null);
          }
        }
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

  // 「已选 N 个」摘要：统一显示指标 ID + 友好名，查询参数仍只使用 ID。
  const metricSummary = (paths: string[], head: number): string =>
    paths.length === 0
      ? ''
      : t('perf.kpiQuery.selectedSummary', {
          count: paths.length,
          items: paths
            .slice(0, head)
            .map((p) => formatMetricDisplay(p, metricLabels[p]))
            .join(', ') + (paths.length > head ? ' ...' : ''),
        });

  const metricSummaryTooltip = (paths: string[]) =>
    paths.length === 0 ? undefined : (
      <div style={{ maxHeight: 300, overflowY: 'auto' }}>
        {paths.map((path, index) => (
          <div key={path}>{index + 1}. {formatMetricDisplay(path, metricLabels[path])}</div>
        ))}
      </div>
    );

  const resultsFullscreenLabel = resultsMaximized
    ? t('perf.kpiQuery.restoreResults')
    : t('perf.kpiQuery.maximizeResults');

  // ── 渲染辅助 ─────────────────────────────────────────────────────
  const renderTemplateItem = (tpl: QueryTemplate) => {
    const canManageByOwnership = isSuperAdmin || tpl.creatorId === currentUser?.id;
    const canManageReport = isSuperAdmin || (tpl.visibility === 'private' && tpl.creatorId === currentUser?.id);
    const reportActionEnabled = canManageReport && canEditTemplate;
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
          [
            <Tooltip key="detail" title={t('perf.kpiQuery.detail.title')}>
              <Button
                type="text"
                size="small"
                icon={<EyeOutlined />}
                onClick={(e) => {
                  e.stopPropagation();
                  void handleOpenDetail(tpl);
                }}
              />
            </Tooltip>,
            ...(canManageByOwnership ? [
                <Tooltip key="report" title={reportActionEnabled ? t('perf.kpiQuery.report.action') : t('common.noPermission')}>
                  <Button
                    type="text"
                    size="small"
                    icon={<BellOutlined />}
                    disabled={!reportActionEnabled}
                    aria-label={t('perf.kpiQuery.report.action')}
                    onClick={(e) => {
                      e.stopPropagation();
                      setReportTemplateId(tpl.id);
                    }}
                  />
                </Tooltip>,
                <Tooltip key="edit" title={canEditTemplate ? t('common.edit') : t('common.noPermission')}>
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    disabled={!canEditTemplate}
                    aria-label={t('common.edit')}
                    onClick={(e) => {
                      e.stopPropagation();
                      void handleOpenUpdateModal(tpl);
                    }}
                  />
                </Tooltip>,
                <Tooltip key="del" title={canDeleteTemplate ? t('common.delete') : t('common.noPermission')}>
                  <Popconfirm
                    disabled={!canDeleteTemplate}
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
                      disabled={!canDeleteTemplate}
                      onClick={(e) => e.stopPropagation()}
                    />
                  </Popconfirm>
                </Tooltip>,
              ] : []),
          ]
        }
      >
        <List.Item.Meta
          title={
            <Space style={{ width: '100%', minWidth: 0 }}>
              <Text ellipsis={{ tooltip: tpl.name }} style={{ flex: 1, minWidth: 0 }}>
                {tpl.name}
              </Text>
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
            {t('perf.kpiQuery.queryTemplates')}
          </Title>
          <Space size={2}>
            <Tooltip title={canAddTemplate ? t('perf.kpiQuery.newTemplateTip') : t('common.noPermission')}>
              <Button
                type="text"
                size="small"
                icon={<PlusOutlined />}
                disabled={!canAddTemplate}
                aria-label={t('perf.kpiQuery.newTemplate')}
                onClick={handleOpenCreateModal}
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
    <div style={{ display: 'flex', width: '100%', height: '100%', overflow: 'hidden', gap: resultsMaximized ? 0 : 15 }}>
      {!resultsMaximized && (sidebarCollapsed ? (
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
            width: 360,
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
      ))}
      <div style={{ flex: 1, minWidth: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <div
        style={{
          padding: 16,
          height: '100%',
          minHeight: 0,
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          gap: resultsMaximized ? 0 : 12,
        }}
      >
        {!resultsMaximized && (
        <Card
          size="small"
          style={{
            flex: '0 1 auto',
            maxHeight: '45%',
            minHeight: 0,
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column',
          }}
          styles={{
            body: {
              flex: 1,
              minHeight: 0,
              overflow: 'hidden',
              display: 'flex',
              flexDirection: 'column',
            },
          }}
          title={
            <Space>
              <TableOutlined />
              <span>{t('perf.kpiQuery.queryConditions')}</span>
            </Space>
          }
        >
          <Form layout="vertical" size="middle" style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
            <div style={{ flex: 1, minHeight: 0, overflow: 'auto', paddingRight: 4 }}>
              <Space wrap size="middle" align="start">
                <Form.Item label={t('perf.kpiQuery.deviceType')} style={{ marginBottom: 0 }}>
                  <Select
                    style={{ width: 160 }}
                    value={payload.deviceType}
                    loading={deviceTypeOptionsLoading}
                    disabled={!hasAvailableDeviceTypes}
                    onChange={(v) => {
                      setPayload({ ...payload, deviceType: v, deviceSns: [], metricPaths: [] });
                      setCellSel({});
                    }}
                    options={deviceTypeOptions}
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
                    <Button
                      disabled={!hasAvailableDeviceTypes}
                      onClick={() => { setPickerTarget('main'); setDevicePickerOpen(true); }}
                    >
                      {t('perf.kpiQuery.pickFromList')}
                    </Button>
                  </Space.Compact>
                </Form.Item>

                <Form.Item label={t('perf.kpiQuery.metric')} style={{ marginBottom: 0 }}>
                  <Space.Compact style={{ width: 360 }}>
                    <Tooltip
                      title={metricSummaryTooltip(payload.metricPaths)}
                      placement="topLeft"
                      overlayStyle={{ maxWidth: 520 }}
                    >
                      <Input
                        readOnly
                        value={metricSummary(payload.metricPaths, 2)}
                        placeholder={t('perf.kpiQuery.selectMetricPlaceholder')}
                      />
                    </Tooltip>
                    <Button
                      disabled={!hasAvailableDeviceTypes}
                      onClick={() => { setPickerTarget('main'); setMetricPickerOpen(true); }}
                    >
                      {t('perf.kpiQuery.pickFromList')}
                    </Button>
                  </Space.Compact>
                </Form.Item>

                <Form.Item label={t('perf.granularity')} style={{ marginBottom: 0 }}>
                  <Radio.Group
                    value={payload.granularity}
                    onChange={(e) => {
                      const g = e.target.value as Granularity;
                      const next: QueryTemplatePayload = { ...payload, granularity: g };
                      // #595: 粒度切换时，若用户未手动修改过时间范围，自动联动
                      if (!timeRangeDirty) {
                        next.timeRangePreset = getDefaultTimeRangeForGranularity(g);
                      }
                      setPayload(next);
                    }}
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
                      onChange={(v) => {
                        setPayload({ ...payload, timeRangePreset: v });
                        setTimeRangeDirty(true);
                      }}
                      options={timeRangeOptions}
                      suffixIcon={<ClockCircleOutlined />}
                    />
                    {payload.timeRangePreset === 'custom' && (
                      <RangePicker
                        showTime
                        value={customRange}
                        onChange={(v) => {
                          setCustomRange(v as [dayjs.Dayjs, dayjs.Dayjs] | null);
                          setTimeRangeDirty(true);
                        }}
                      />
                    )}
                  </Space>
                </Form.Item>
              </Space>

              {/* #619：测量对象下钻选择器（选完设备后可选过滤小区） */}
              {payload.deviceSns.length > 0 && (
                <div style={{ marginTop: 12 }}>
                  <CellDrilldownSelector
                    deviceSns={payload.deviceSns}
                    technology={payload.deviceType ? deviceTypeToTechnology(payload.deviceType) : undefined}
                    useNrRecommendedDefault
                    value={cellSel}
                    onChange={setCellSel}
                  />
                </div>
              )}
            </div>

            <div style={{ flexShrink: 0, marginTop: 12, borderTop: `1px dashed ${token.colorBorderSecondary}`, paddingTop: 12 }}>
              <Space>
                <Tooltip title={canQuery ? undefined : t('common.noPermission')}>
                  <Button
                    type="primary"
                    icon={<TableOutlined />}
                    loading={aggFetching}
                    disabled={!canQuery || !hasAvailableDeviceTypes || currentObjectsLoading}
                    onClick={handleQuery}
                  >
                    {t('common.query')}
                  </Button>
                </Tooltip>
                <Tooltip title={canAddTemplate ? undefined : t('common.noPermission')}>
                  <Button icon={<SaveOutlined />} disabled={!canAddTemplate || !hasAvailableDeviceTypes} onClick={handleOpenSaveAsModal}>
                    {t('perf.kpiQuery.saveAsTemplate')}
                  </Button>
                </Tooltip>
                <Button
                  icon={<ExportOutlined />}
                  onClick={handleExport}
                  loading={createExport.isPending}
                  disabled={aggFetching || (!submitted && !isSelectionExceedsLimit(payload))}
                >
                  {t('perf.kpiQuery.exportCsv')}
                </Button>
                <Button
                  icon={<PlusOutlined />}
                  onClick={() => {
                    setPayload({
                      ...DEFAULT_PAYLOAD,
                      deviceSns: [],
                      metricPaths: [],
                      deviceType: firstAvailableDeviceType ?? DEFAULT_PAYLOAD.deviceType,
                    });
                    setCustomRange(null);
                    setTimeRangeDirty(false);
                    setCellSel({});
                    setActiveTemplateId(undefined);
                    setSubmitted(null);
                    setResultsQueryReady(true);
                    setPivotPage(1);
                    setPivotPageSize(KPI_QUERY_DEFAULT_PIVOT_PAGE_SIZE);
                    skipNextSaveRef.current = true;
                    usePmPageStateStore.getState().clearPageState(PM_KPI_QUERY_PAGE_KEY);
                  }}
                >
                  {t('common.reset')}
                </Button>
              </Space>
            </div>
          </Form>
        </Card>
        )}

        {!restoreQueryPending && aggTruncated ? (
          <Alert
            type="warning"
            showIcon
            style={{ flexShrink: 0 }}
            message={intl.formatMessage(
              { id: 'perf.dashboard.truncatedTip' },
              { shown: displayedRows.length, total: displayedTotal },
            )}
          />
        ) : null}

        <Card
          size="small"
          title={<span><TableOutlined /> {t('perf.kpiQuery.queryResults')}</span>}
          extra={
            <Tooltip title={resultsFullscreenLabel}>
              <Button
                type="text"
                size="small"
                icon={resultsMaximized ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
                aria-label={resultsFullscreenLabel}
                onClick={() => setResultsMaximized((current) => !current)}
              />
            </Tooltip>
          }
          style={{ flex: 1, minHeight: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}
          styles={{
            body: {
              flex: 1,
              minHeight: 0,
              overflow: 'hidden',
              display: 'flex',
              flexDirection: 'column',
            },
          }}
        >
          <PivotTable
            rows={displayedRows}
            loading={displayedLoading}
            pagination={{
              current: pivotPage,
              pageSize: pivotPageSize,
              total: displayedTotal,
              showSizeChanger: true,
              showTotal: (count) => t('perf.kpiQuery.pivot.totalRows', { count }),
              onChange: (page, pageSize) => {
                setPivotPage(page);
                setPivotPageSize(pageSize);
              },
            }}
            // gNB 查空时给更明确的引导（#201）：5G 真机样本厂商错配会让 KPI 算不出、
            // 后端返回 items=null，泛化「暂无数据」无法区分「指标库未注册」与「时段无采样」。
            // 仅在已发起查询（submitted 存在）且制式=gNB 时替换文案。
            emptyDescription={
              submitted?.payload.deviceType === 'GNB' ? (
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
              setCellSel({}); // #619：设备变更时清空小区选择
            }
          }}
          initialSelected={pickerTarget === 'modal' ? saveForm.payload.deviceSns : payload.deviceSns}
          // 外层「设备类型」是设备清单的唯一来源（#443）：按当前目标的 deviceType 映射制式，
          // 设备弹窗只列对应制式设备（main/modal 各按各自 deviceType）。
          technology={deviceTypeToTechnology(
            (pickerTarget === 'modal' ? saveForm.payload.deviceType : payload.deviceType) ?? 'ENB',
          )}
          maxSelected={PM_QUERY_SELECTION_LIMIT}
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
          initialLabels={metricLabels}
          initialDeviceType={
            (pickerTarget === 'modal' ? saveForm.payload.deviceType : payload.deviceType) ?? 'ENB'
          }
          // 外层「设备类型」是唯一来源（#443）：锁定弹窗内部类型，隐藏其重复下拉，跟随外层值。
          lockDeviceType
          maxSelected={PM_QUERY_SELECTION_LIMIT}
          enableBatchInput
          onlyEnabledIndicators
        />

        <QueryTemplateDetailModal
          open={detailTemplate != null}
          template={detailTemplate}
          metricLabels={metricLabels}
          onClose={() => setDetailTemplateId(undefined)}
        />

        <ReportSubscriptionModal
          open={reportTemplate != null}
          template={reportTemplate}
          onClose={() => setReportTemplateId(undefined)}
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
                  style={{ width: 160 }}
                  value={saveForm.payload.deviceType}
                  loading={deviceTypeOptionsLoading}
                  disabled={!hasAvailableDeviceTypes}
                  onChange={(v) =>
                    setSaveForm((s) => ({
                      ...s,
                      payload: { ...s.payload, deviceType: v, deviceSns: [], metricPaths: [] },
                    }))
                  }
                  options={deviceTypeOptions}
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
                <Tooltip
                  title={metricSummaryTooltip(saveForm.payload.metricPaths)}
                  placement="topLeft"
                  overlayStyle={{ maxWidth: 520 }}
                >
                  <Input
                    readOnly
                    value={metricSummary(saveForm.payload.metricPaths, 3)}
                    placeholder={t('perf.kpiQuery.selectMetricPlaceholder')}
                  />
                </Tooltip>
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
