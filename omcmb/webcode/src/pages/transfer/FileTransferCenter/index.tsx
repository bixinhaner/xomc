import { useEffect, useMemo, useRef, useState } from 'react';
import dayjs from 'dayjs';
import {
  Button,
  Card,
  Checkbox,
  DatePicker,
  Descriptions,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
  Progress,
  Radio,
  Select,
  Space,
  Table,
  Tag,
  Tabs,
  Tooltip,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, DownloadOutlined, EyeOutlined, PlusOutlined, ExportOutlined, ReloadOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { useT } from '@/hooks/useT';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import SearchInput from '@/components/SearchInput';
import MRTasksPanel from '@/pages/mr/Tasks';
import {
  useBatchDeleteUfteTasks,
  useCreateUnifiedFileTransferTask,
  useDeleteUfteTask,
  useStartUfteTask,
  useSuspendUfteTask,
  useTerminateUfteTask,
  useUnifiedFileTransferDeviceCandidates,
  useUnifiedFileTransferDevices,
  useUnifiedFileTransferTasks,
  useUnifiedFileTransferTaskTypes,
} from '@core/hooks/api/useUnifiedFileTransfer';
import { useProductClasses } from '@core/hooks/api/useDevices';
import { useSoftwareVersions } from '@core/hooks/api/useSoftware';
import { unifiedFileTransferApi } from '@core/services/api/unifiedFileTransferApi';
import { configSnapshotApi } from '@core/services/api/configSnapshotApi';
import type { BatchGetSnapshotsResult } from '@core/services/api/configSnapshotApi';
import { deviceLicenseApi } from '@core/services/api/deviceLicenseApi';
import type { BatchGetLicensesResult } from '@core/services/api/deviceLicenseApi';
import { useUserStore } from '@core/store/userStore';
import { useAppStore } from '@core/store/appStore';
import type { SoftwareVersion } from '@core/mock/data/software';
import type {
  CreateUnifiedFileTransferTaskInput,
  FirmwareLibraryFileType,
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferTask,
  UnifiedFileTransferTaskType,
} from '@core/types/unifiedFileTransfer';
import {
  buildCategoryTabs,
  getExecutionModeOptions,
  buildDefaultUfteTaskName,
  getSoftwareLibraryFileTypeLabel,
  getStepLabels,
  localizeBuiltinCategoryLabel,
  localizeBuiltinTypeName,
  UPGRADE_LIKE_CATEGORIES,
} from '../shared';
import {
  renderDeviceStatus,
  renderEllipsisCell,
  renderTaskStatus,
} from '../shared.render';
import type { TransferStepId } from '@core/types/unifiedFileTransfer';

const { Text, Title } = Typography;

// buildDefaultTaskName 适配器：保持原签名（typeCode + username），内部委托给
// shared.buildDefaultUfteTaskName + appStore.locale，全局保持 i18n 一致。
function buildDefaultTaskName(typeCode: string | undefined, username: string | undefined, locale: string): string {
  return buildDefaultUfteTaskName(typeCode, username, locale, dayjs().format('YYYY-MM-DD HH:mm:ss'));
}

function isUpgradeTaskCategory(category?: string) {
  return category === 'gnb_upgrade' || category === 'enb_upgrade';
}

function matchesScope(scope: string[], _category: string, productClass: string): boolean {
  const upper = productClass.toUpperCase();
  for (const s of scope) {
    const su = s.toUpperCase();
    if (upper === su || upper.includes(su) || su.includes(upper)) return true;
    if (su.includes(' ')) {
      const tokens = su.split(/\s+/).filter((t) => t.length >= 2);
      for (const token of tokens) {
        if (upper.includes(token)) return true;
      }
    }
  }
  return false;
}

function splitDeviceTypes(deviceType?: string) {
  return (deviceType ?? '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function needsFirmwareSelection(taskType?: UnifiedFileTransferTaskType) {
  return Boolean(taskType && taskType.rpcType === 'DOWNLOAD' && isUpgradeTaskCategory(taskType.category));
}

function buildFirmwareCandidateList(versions: SoftwareVersion[]) {
  const sorted = [...versions].sort((left, right) => {
    if (left.recommend !== right.recommend) {
      return left.recommend ? -1 : 1;
    }
    if (left.status !== right.status) {
      if (left.status === 'current') return -1;
      if (right.status === 'current') return 1;
    }
    return right.releaseDate.localeCompare(left.releaseDate);
  });

  return sorted;
}

function resolveFirmwareLibraryFileType(taskType?: UnifiedFileTransferTaskType): FirmwareLibraryFileType | undefined {
  if (taskType?.firmwareFileType !== undefined) {
    return taskType.firmwareFileType;
  }
  const haystack = `${taskType?.typeCode ?? ''} ${taskType?.displayName ?? ''} ${taskType?.fileType ?? ''} ${taskType?.fileTypeLabel ?? ''}`.toLowerCase();
  if (haystack.includes('fpga')) {
    return 6;
  }
  if (haystack.includes('patch')) {
    return 1;
  }
  if (taskType && isUpgradeTaskCategory(taskType.category)) {
    return 0;
  }
  return undefined;
}

function getUpgradeTypeLabel(category: string, fallback: string, t: (id: string) => string) {
  if (category === 'gnb_upgrade' || category === 'enb_upgrade') {
    return t('ufte.softLib.upgrade');
  }
  if (category === 'version_rollback') {
    return t('ufte.softLib.rollback');
  }
  return fallback;
}

export default function FileTransferCenter() {
  const t = useT();
  const queryClient = useQueryClient();
  // 任务名称自动填充用：取登录用户名拼前缀，displayName / username 哪个有用哪个。
  const currentUser = useUserStore((s) => s.currentUser);
  const taskNameUser = currentUser?.username || currentUser?.displayName || 'user';
  const appLocale = useAppStore((s) => s.locale);
  // lastAutoFilledTaskNameRef 记录最近一次自动填的名字。用户在表单里手动改过 → ref
  // 跟 form 值不再一致 → typeCode 切换时不覆盖；用户没改 → 切换业务时跟着刷新。
  const lastAutoFilledTaskNameRef = useRef<string>('');
  const { data: taskTypes = [], isLoading: taskTypesLoading } = useUnifiedFileTransferTaskTypes();
  const { data: productClasses = [] } = useProductClasses();
  // 原始 categories 来自后端 ufte_task_types，新增一个虚拟分类 "mr_measurement"
  // 作为入口聚合按钮（点击跳到独立的 MR 任务管理页 /mr/tasks）。MR 不走 UFTE
  // 任务模板（PRD F05 决策 A — 独立引擎），这里只做"入口聚合"。
  const categories = useMemo(() => {
    const base = buildCategoryTabs(taskTypes);
    // 末位插入虚拟项；templateCount=0 让现有 UFTE 模板渲染逻辑识别"无模板"
    return [
      ...base,
      {
        category: 'mr_measurement',
        categoryLabel: 'MR 测量', // 通过 localizeBuiltinCategoryLabel 翻译
        templateCount: 0,
      },
    ];
  }, [taskTypes]);
  // URL 参数初始化：?category=...&typeCode=... 用于外部 deep link（如
  // 设备列表批量"日志收集"自动跳到 station_log + RUNTIME_LOG_COLLECT tab）。
  const [urlSearchParams] = useSearchParams();
  const [selectedCategory, setSelectedCategory] = useState(() => urlSearchParams.get('category') ?? '');
  const [selectedTypeCode, setSelectedTypeCode] = useState(() => urlSearchParams.get('typeCode') ?? '');
  const [taskPage, setTaskPage] = useState(1);
  const [taskPageSize, setTaskPageSize] = useState(10);
  const [taskKeyword, setTaskKeyword] = useState('');
  const [taskKeywordInput, setTaskKeywordInput] = useState('');
  const [taskStatusFilter, setTaskStatusFilter] = useState<string>();
  const [devicePage, setDevicePage] = useState(1);
  const [devicePageSize, setDevicePageSize] = useState(10);
  const [deviceKeyword, setDeviceKeyword] = useState('');
  const [deviceKeywordInput, setDeviceKeywordInput] = useState('');
  const [deviceStatusFilter, setDeviceStatusFilter] = useState<string>();
  const [deviceProductClassFilter, setDeviceProductClassFilter] = useState<string>();
  const [viewMode, setViewMode] = useState<'tasks' | 'devices'>('tasks');
  const [taskDrawerOpen, setTaskDrawerOpen] = useState(false);
  // MR 测量 Tab 的新建抽屉 — 受控状态，让顶部 "New Task" 按钮接管打开（与 UFTE 风格一致）
  const [mrCreateOpen, setMrCreateOpen] = useState(false);
  const [taskForm] = Form.useForm<CreateUnifiedFileTransferTaskInput>();
  const drawerTypeCode = Form.useWatch('typeCode', taskForm);
  const drawerProductClass = Form.useWatch('productClass', taskForm);
  const [selectedDrawerDeviceIds, setSelectedDrawerDeviceIds] = useState<string[]>([]);
  // 批量输入 SN 弹窗：用户粘贴多个 SN（;/,/ 空格/换行/Tab 任意分隔），匹配
  // 当前可选设备 → 并入 selectedDrawerDeviceIds。不在候选名单的 SN 走 warning。
  const [batchSNInputOpen, setBatchSNInputOpen] = useState(false);
  const [batchSNInputText, setBatchSNInputText] = useState('');
  const [batchSNApplying, setBatchSNApplying] = useState(false);
  // T-0164: CONFIG_RESTORE 模式下"按设备 SN 检查 config_snapshots 表"的结果。
  // selectedDrawerDeviceIds + drawerTypeCode 变化时自动 refetch；缺失则禁用提交。
  const [snapshotProbe, setSnapshotProbe] = useState<BatchGetSnapshotsResult | null>(null);
  const [snapshotProbing, setSnapshotProbing] = useState(false);
  // T-0165: LICENSE_UPGRADE 模式下"按设备 SN 检查 device_licenses 表"的结果（同款）。
  const [licenseProbe, setLicenseProbe] = useState<BatchGetLicensesResult | null>(null);
  const [licenseProbing, setLicenseProbing] = useState(false);
  // probeTick：tab 重新聚焦时 +1，触发 probe useEffect 重新跑——用户在新 tab
  // 上传文件后切回，自动看到最新匹配状态，与 React Query refetchOnWindowFocus 同款体验。
  const [probeTick, setProbeTick] = useState(0);
  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === 'visible') {
        setProbeTick((t) => t + 1);
      }
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
  }, []);
  const [drawerDeviceKeyword, setDrawerDeviceKeyword] = useState('');
  const [drawerDeviceKeywordInput, setDrawerDeviceKeywordInput] = useState('');
  const [detailTask, setDetailTask] = useState<UnifiedFileTransferTask | null>(null);
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false);

  const { data: tasksData, isLoading: tasksLoading } = useUnifiedFileTransferTasks({
    page: taskPage,
    pageSize: taskPageSize,
    category: selectedCategory || undefined,
    keyword: taskKeyword || undefined,
    status: taskStatusFilter,
    typeCode: selectedTypeCode || undefined,
  });

  const { data: devicesData, isLoading: devicesLoading } = useUnifiedFileTransferDevices({
    page: devicePage,
    pageSize: devicePageSize,
    category: selectedCategory || undefined,
    keyword: deviceKeyword || undefined,
    status: deviceStatusFilter,
    typeCode: selectedTypeCode || undefined,
    productType: deviceProductClassFilter,
  });

  const createTaskMutation = useCreateUnifiedFileTransferTask();
  const startTaskMutation = useStartUfteTask();
  const suspendTaskMutation = useSuspendUfteTask();
  const terminateTaskMutation = useTerminateUfteTask();
  const deleteTaskMutation = useDeleteUfteTask();
  const batchDeleteTasksMutation = useBatchDeleteUfteTasks();
  const [selectedTaskIds, setSelectedTaskIds] = useState<React.Key[]>([]);
  // 切 tab/分类/类型 + 改过滤器 + 翻页 → 选中集自动清空，避免"显示 3 个已选但其实
  // 是上一个 tab 的任务 ID"的违和感。
  // 注：表格 dataSource 切换后，AntD 表格会自动收起"勾选行"显示，但 selectedTaskIds
  // 状态还在 → 顶部"批量删除（N）"按钮的计数错位；这条 useEffect 同步清掉。
  const recentTasks = tasksData?.items ?? [];
  const recentDevices = devicesData?.items ?? [];

  const filteredTaskTypes = useMemo(
    // 顺序由后端 ORDER BY sort_order ASC 控制（数据库字段 ufte_task_types.sort_order
    // 由内置模板初始化，未来可在「模板配置」页面拖拽调整）。前端不再二次排序，避免
    // 跟数据库源头不一致。
    () => taskTypes.filter((item) => item.category === selectedCategory),
    [selectedCategory, taskTypes],
  );

  const taskTypeOptions = useMemo(
    () => filteredTaskTypes.map((item) => ({
      label: localizeBuiltinTypeName(item.typeCode, item.displayName, t),
      value: item.typeCode,
    })),
    [filteredTaskTypes, t],
  );

  // EXECUTION_MODE_OPTIONS 常量已下线 —— 改用 getExecutionModeOptions(t) 适配 i18n
  const executionModeOptions = useMemo(() => getExecutionModeOptions(t), [t]);

  // STEP_LABELS 常量已下线 —— 改用 getStepLabels(t) 适配 i18n（用于详情步骤名展示）
  const stepLabels = useMemo(() => getStepLabels(t), [t]);

  const activeTaskType = useMemo(
    () => filteredTaskTypes.find((item) => item.typeCode === selectedTypeCode) ?? filteredTaskTypes[0],
    [filteredTaskTypes, selectedTypeCode],
  );

  const drawerTaskType = useMemo(
    () => taskTypes.find((item) => item.typeCode === drawerTypeCode) ?? activeTaskType,
    [activeTaskType, drawerTypeCode, taskTypes],
  );
  const firmwareLibraryFileType = resolveFirmwareLibraryFileType(drawerTaskType);
  const { data: firmwareData } = useSoftwareVersions({
    page: 1,
    pageSize: 200,
    fileType: firmwareLibraryFileType,
  });

  // 创建任务时全部三种执行方式可选（立即 / 挂起 / 定时）。
  // 定时模式下表单会条件渲染 DatePicker，handleCreateTask 把时间塞进 payload.scheduledAt。
  const createExecutionModeOptions = executionModeOptions;

  const { data: drawerDevicesData, isLoading: drawerDevicesLoading } = useUnifiedFileTransferDeviceCandidates({
    page: 1,
    pageSize: 200,
    category: selectedCategory || undefined,
    typeCode: drawerTaskType?.typeCode || selectedTypeCode || undefined,
    // 升级类任务（needsFirmwareSelection）用 drawerProductClass 表单字段
    // 缩窄候选设备 — UFTE 后端字段名是 productType。
    productType: needsFirmwareSelection(drawerTaskType) ? drawerProductClass : undefined,
    keyword: drawerDeviceKeyword || undefined,
  });

  const drawerDeviceCandidates = drawerDevicesData?.items ?? [];

  const firmwareCandidates = useMemo(
    () => buildFirmwareCandidateList(firmwareData?.items ?? []),
    [firmwareData?.items],
  );

  const drawerProductClassOptions = useMemo(() => {
    const scope = drawerTaskType?.platformScope ?? [];
    const category = drawerTaskType?.category ?? '';
    const realClasses = new Set<string>();
    productClasses.forEach((pc) => {
      if (scope.length === 0 || matchesScope(scope, category, pc)) {
        realClasses.add(pc);
      }
    });
    firmwareCandidates.forEach((item) => {
      splitDeviceTypes(item.deviceType).forEach((entry) => {
        if (scope.length === 0 || matchesScope(scope, category, entry)) {
          realClasses.add(entry);
        }
      });
    });
    return Array.from(realClasses).map((item) => ({ label: item, value: item }));
  }, [drawerTaskType, firmwareCandidates, productClasses]);

  const filteredFirmwareCandidates = useMemo(
    () => (drawerProductClass
      ? firmwareCandidates.filter((item) => {
          const deviceTypes = splitDeviceTypes(item.deviceType);
          return deviceTypes.length === 0 || deviceTypes.includes(drawerProductClass);
        })
      : []),
    [drawerProductClass, firmwareCandidates],
  );

  const firmwareOptions = useMemo(
    () => filteredFirmwareCandidates.map((item) => ({
      label: item.fileName || item.versionCode,
      value: item.id,
    })),
    [filteredFirmwareCandidates],
  );

  const deviceProductClassOptions = useMemo(() => {
    const values = new Set<string>(productClasses);
    recentDevices.forEach((item) => {
      if (item.productType) {
        values.add(item.productType);
      }
    });
    (activeTaskType?.platformScope ?? []).forEach((entry) => values.add(entry));
    return Array.from(values).map((item) => ({ label: item, value: item }));
  }, [activeTaskType?.platformScope, productClasses, recentDevices]);

  const templateTabItems = useMemo(
    () => filteredTaskTypes.map((item) => ({
      key: item.typeCode,
      label: (
        <Space size={6}>
          <span>{localizeBuiltinTypeName(item.typeCode, item.displayName, t)}</span>
          <Tag color={item.builtIn ? 'blue' : 'gold'}>{item.builtIn ? t('ufte.tag.builtIn') : t('ufte.tag.custom')}</Tag>
        </Space>
      ),
    })),
    [filteredTaskTypes, t],
  );

  const drawerSelectedDevices = useMemo(
    () => drawerDeviceCandidates.filter((item) => selectedDrawerDeviceIds.includes(item.id)),
    [drawerDeviceCandidates, selectedDrawerDeviceIds],
  );

  // T-0164: CONFIG_RESTORE 模式自动检测每台设备是否已有最新配置快照。
  // 用 dependency 的 join 字符串避免数组引用变化导致的无限刷新。
  const drawerSelectedSnsKey = useMemo(
    () => drawerSelectedDevices.map((d) => d.deviceSn).filter(Boolean).sort().join(','),
    [drawerSelectedDevices],
  );
  useEffect(() => {
    if (drawerTypeCode !== 'CONFIG_RESTORE') {
      setSnapshotProbe(null);
      return;
    }
    if (!drawerSelectedSnsKey) {
      setSnapshotProbe(null);
      return;
    }
    let cancelled = false;
    const sns = drawerSelectedSnsKey.split(',');
    setSnapshotProbing(true);
    configSnapshotApi
      .batchGet(sns)
      .then((res) => {
        if (!cancelled) setSnapshotProbe(res);
      })
      .catch(() => {
        if (!cancelled) setSnapshotProbe({ found: {}, missing: sns });
      })
      .finally(() => {
        if (!cancelled) setSnapshotProbing(false);
      });
    return () => {
      cancelled = true;
    };
  }, [drawerTypeCode, drawerSelectedSnsKey, probeTick]);

  const snapshotMissingCount = snapshotProbe?.missing.length ?? 0;
  const isConfigRestoreBlockedByMissing =
    drawerTypeCode === 'CONFIG_RESTORE' && snapshotMissingCount > 0;

  // T-0165: LICENSE_UPGRADE 同款 — 选完设备后批量查 device_licenses。
  useEffect(() => {
    if (drawerTypeCode !== 'LICENSE_UPGRADE') {
      setLicenseProbe(null);
      return;
    }
    if (!drawerSelectedSnsKey) {
      setLicenseProbe(null);
      return;
    }
    let cancelled = false;
    const sns = drawerSelectedSnsKey.split(',');
    setLicenseProbing(true);
    deviceLicenseApi
      .batchGet(sns)
      .then((res) => {
        if (!cancelled) setLicenseProbe(res);
      })
      .catch(() => {
        if (!cancelled) setLicenseProbe({ found: {}, missing: sns });
      })
      .finally(() => {
        if (!cancelled) setLicenseProbing(false);
      });
    return () => {
      cancelled = true;
    };
  }, [drawerTypeCode, drawerSelectedSnsKey, probeTick]);

  const licenseMissingCount = licenseProbe?.missing.length ?? 0;
  const isLicenseUpgradeBlockedByMissing =
    drawerTypeCode === 'LICENSE_UPGRADE' && licenseMissingCount > 0;

  const drawerDeviceColumns: ColumnsType<UnifiedFileTransferDeviceItem> = useMemo(
    () => [
      { title: t('ufte.col.deviceSn'), dataIndex: 'deviceSn', key: 'deviceSn', width: 160 },
      { title: t('ufte.col.stationName'), dataIndex: 'deviceName', key: 'deviceName', ellipsis: true },
      // 后端 UFTE DeviceItem 字段名是 productType（见 internal/ufte/model.go），
      // 不是 productClass —— 前端原 dataIndex 写错，真实数据下永远空。
      { title: t('ufte.col.productType'), dataIndex: 'productType', key: 'productType', width: 140 },
      { title: t('ufte.col.currentVersion'), dataIndex: 'currentVersion', key: 'currentVersion', width: 120 },
    ],
    [t],
  );

  const failureReasonColumn = {
    title: t('software.failureReason'),
    dataIndex: 'failureReason',
    key: 'failureReason',
    width: 260,
    render: (value: string, record: UnifiedFileTransferDeviceItem) => {
      if (!value) return '-';
      // 终止时 SoftwareService.TerminateUpgrade 给 sub_task 写的固定串
      // "task terminated by operator"——历史代码没用 code，这里做一层兜底翻译。
      const codeOrRaw = value === 'task terminated by operator' ? 'OPERATOR_TERMINATED' : value;
      const i18nLabel = t(`software.failureCode.${codeOrRaw}` as Parameters<typeof t>[0]);
      const display = i18nLabel && i18nLabel !== `software.failureCode.${codeOrRaw}` ? i18nLabel : value;
      // 设备厂商原始 fault（FaultCode + FaultString）放 Tooltip 里——i18n label 只看到统一
      // 错误码描述，hover 后能拿到设备端原文（如 "Upgrade failed, there is FaultString in
      // TransferComplete msg. FaultCode: 0, FaultString: httpUpload OM Http Put Upload stat
      // file error"），方便厂商侧排查。
      const detail = record.failureDetail;
      const text = <span style={{ color: '#ff4d4f' }}>{display}</span>;
      if (!detail || detail === display) return text;
      return (
        <Tooltip title={detail} placement="topLeft" overlayStyle={{ maxWidth: 480 }}>
          {text}
        </Tooltip>
      );
    },
  };

  const getTaskActionErrorMessage = (error: unknown, fallback: string) => {
    if (error instanceof Error && error.message.trim().length > 0) {
      return error.message;
    }
    return fallback;
  };

  const handleStartTask = (record: UnifiedFileTransferTask) => {
    void startTaskMutation.mutateAsync(record.id)
      .then(() => void message.success(t('ufte.msg.taskStarted')))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, t('ufte.msg.taskStartFailed'))));
  };

  const handleSuspendTask = (record: UnifiedFileTransferTask) => {
    void suspendTaskMutation.mutateAsync(record.id)
      .then(() => void message.success(t('ufte.msg.taskPaused')))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, t('ufte.msg.taskPauseFailed'))));
  };

  const handleTerminateTask = (record: UnifiedFileTransferTask) => {
    void terminateTaskMutation.mutateAsync(record.id)
      .then(() => void message.success(t('ufte.msg.taskTerminated')))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, t('ufte.msg.taskTerminateFailed'))));
  };

  const handleDeleteTask = (record: UnifiedFileTransferTask) => {
    void deleteTaskMutation.mutateAsync(record.id)
      .then(() => void message.success(t('ufte.msg.taskDeleted')))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, t('ufte.msg.taskDeleteFailed'))));
  };

  // 批量输入 SN → 解析 → 全量候选匹配 → 并入选中。
  // 分隔符：; , 空格 Tab 换行（任一）；自动 trim、去重、忽略空串。
  const handleApplyBatchSNs = async () => {
    const raw = batchSNInputText || '';
    const tokens = Array.from(new Set(
      raw.split(/[;,\s]+/).map((s) => s.trim()).filter(Boolean),
    ));
    if (tokens.length === 0) {
      void message.warning(t('ufte.msg.snAtLeastOne'));
      return;
    }
    setBatchSNApplying(true);
    try {
      // 当前可见 200 条候选不够，按同款过滤条件拉一次 pageSize=10000
      // 全量匹配。命中即累加，未命中给出明细。
      const resp = await unifiedFileTransferApi.getDeviceCandidates({
        page: 1,
        pageSize: 10000,
        category: selectedCategory || undefined,
        typeCode: drawerTaskType?.typeCode || selectedTypeCode || undefined,
        productType: needsFirmwareSelection(drawerTaskType) ? drawerProductClass : undefined,
      });
      const allCandidates = resp.items ?? [];
      const snToId = new Map(allCandidates.map((d) => [d.deviceSn, d.id]));
      const matchedIds: string[] = [];
      const unmatched: string[] = [];
      for (const sn of tokens) {
        const id = snToId.get(sn);
        if (id) matchedIds.push(id);
        else unmatched.push(sn);
      }
      // 与已选合并去重
      setSelectedDrawerDeviceIds((prev) => Array.from(new Set([...prev, ...matchedIds])));
      if (unmatched.length === 0) {
        void message.success(t('ufte.msg.batchSelectedCount', { count: matchedIds.length }));
      } else {
        void message.warning(t('ufte.msg.batchPartialMatch', {
          matched: matchedIds.length,
          unmatched: unmatched.length,
          sns: unmatched.slice(0, 5).join('、'),
          ellipsis: unmatched.length > 5 ? '…' : '',
        }));
      }
      setBatchSNInputOpen(false);
      setBatchSNInputText('');
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('ufte.msg.retryLater');
      void message.error(t('ufte.msg.batchInputFailed', { msg }));
    } finally {
      setBatchSNApplying(false);
    }
  };

  // 设备列表导出 CSV — 调后端 /ufte/devices/export 端点：复用 ListDevices
  // 过滤逻辑、不分页拿全量、服务端拼 CSV 流式返回。比客户端拼更可靠（避免
  // pageSize 上限、内存膨胀、SN 含逗号转义错位等问题）。
  const [devicesExporting, setDevicesExporting] = useState(false);
  const handleExportDevices = async () => {
    setDevicesExporting(true);
    try {
      const { blob, filename } = await unifiedFileTransferApi.exportDevices({
        category: selectedCategory || undefined,
        typeCode: selectedTypeCode || undefined,
        keyword: deviceKeyword || undefined,
        status: deviceStatusFilter,
        productType: deviceProductClassFilter,
        // 与当前 tab 的 deviceColumns 分支保持一致
        view: isUpgradeLikeCategory ? 'upgrade' : 'default',
      });
      if (blob.size === 0) {
        void message.warning(t('ufte.msg.exportEmpty'));
        return;
      }
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      void message.success(t('ufte.msg.exportStarted'));
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('ufte.msg.retryLater');
      void message.error(t('ufte.msg.exportFailed', { msg }));
    } finally {
      setDevicesExporting(false);
    }
  };

  const openDetailDrawer = (record: UnifiedFileTransferTask) => {
    setDetailTask(record);
    setDetailDrawerOpen(true);
  };

  const renderTaskActions = (record: UnifiedFileTransferTask) => {
    const status = record.status;
    const showStart = status === 'pending' || status === 'suspended';
    const showSuspend = status === 'in_progress';
    const showTerminate = status !== 'ended';
    const showDelete = status === 'ended' || status === 'pending' || status === 'suspended';
    return (
      <Space size={4} wrap={false}>
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => openDetailDrawer(record)}
          style={{ paddingInline: 0 }}
        >
          {t('common.detail')}
        </Button>
        {showStart ? (
          <Button
            type="link"
            size="small"
            loading={startTaskMutation.isPending}
            onClick={() => handleStartTask(record)}
            style={{ paddingInline: 0 }}
          >
            {t('common.start')}
          </Button>
        ) : null}
        {showSuspend ? (
          <Button
            type="link"
            size="small"
            loading={suspendTaskMutation.isPending}
            onClick={() => handleSuspendTask(record)}
            style={{ paddingInline: 0 }}
          >
            {t('ufte.action.pause')}
          </Button>
        ) : null}
        {showTerminate ? (
          <Popconfirm title={t('ufte.confirm.terminateTask')} onConfirm={() => handleTerminateTask(record)}>
            <Button type="link" size="small" danger loading={terminateTaskMutation.isPending} style={{ paddingInline: 0 }}>
              {t('common.terminate')}
            </Button>
          </Popconfirm>
        ) : null}
        {showDelete ? (
          <Popconfirm title={t('ufte.confirm.deleteTask')} onConfirm={() => handleDeleteTask(record)}>
            <Button type="link" size="small" danger loading={deleteTaskMutation.isPending} style={{ paddingInline: 0 }}>
              {t('common.delete')}
            </Button>
          </Popconfirm>
        ) : null}
      </Space>
    );
  };

  const taskActionColumn = {
    title: t('common.operation'),
    key: 'action',
    width: 190,
    align: 'center' as const,
    render: (_: unknown, record: UnifiedFileTransferTask) => renderTaskActions(record),
  };

  useEffect(() => {
    if (categories.length === 0) {
      setSelectedCategory('');
      return;
    }
    if (!categories.some((item) => item.category === selectedCategory)) {
      setSelectedCategory(categories[0].category);
    }
  }, [categories, selectedCategory]);

  useEffect(() => {
    const preferredType = filteredTaskTypes[0];
    if (!preferredType) {
      setSelectedTypeCode('');
      return;
    }
    if (!filteredTaskTypes.some((item) => item.typeCode === selectedTypeCode)) {
      setSelectedTypeCode(preferredType.typeCode);
    }
  }, [filteredTaskTypes, selectedTypeCode]);

  useEffect(() => {
    setTaskPage(1);
    setDevicePage(1);
    // 同步清空批量选中——切 tab / 改过滤 / 翻页时旧的选中 ID 已不在当前可见行
    setSelectedTaskIds([]);
  }, [selectedCategory, selectedTypeCode, taskKeyword, taskStatusFilter, deviceKeyword, deviceStatusFilter, deviceProductClassFilter]);

  // 翻页（taskPage / taskPageSize 变化）同样清空选中，避免跨页 ID 残留计数
  useEffect(() => {
    setSelectedTaskIds([]);
  }, [taskPage, taskPageSize]);

  useEffect(() => {
    setDeviceProductClassFilter(undefined);
  }, [selectedTypeCode]);

  const isUpgradeLikeCategory = UPGRADE_LIKE_CATEGORIES.has(selectedCategory);

  const getTypeDef = (typeCode: string) => taskTypes.find((item) => item.typeCode === typeCode);

  const getTaskTargetVersion = (record: UnifiedFileTransferTask) => {
    // 只有升级 / 回滚类才有"主任务目标版本"（固件版本号）；备份 / 日志采集 /
    // 配置恢复类后端返回空串，前端不再用 typeDef.fileNameTemplate 等模板字符串兜底
    // ——那只是占位符模板，对主任务来说没有意义。详见
    // docs/project/backup-display-fix-20260520.md F2。
    return record.targetVersion?.trim() ? record.targetVersion : '-';
  };

  const getTaskProductClass = (record: UnifiedFileTransferTask) => {
    if (record.productType) {
      return record.productType;
    }
    const typeDef = getTypeDef(record.typeCode);
    return typeDef?.platformScope?.[0] || '-';
  };

  const taskColumns: ColumnsType<UnifiedFileTransferTask> = useMemo(() => {
    if (isUpgradeLikeCategory) {
      return [
        taskActionColumn,
        {
          // 任务名称：默认按 业务_用户_时间 自动生成（约 30-40 字符），固定 280 + Tooltip 兜底。
          title: t('ufte.col.taskName'),
          dataIndex: 'taskName',
          key: 'taskName',
          width: 280,
          render: (_, record) => renderEllipsisCell(record.taskName, { strong: true }),
        },
        {
          title: t('ufte.col.operator'),
          dataIndex: 'createUser',
          key: 'createUser',
          width: 130,
          render: (value: string) => renderEllipsisCell(value),
        },
        {
          title: t('ufte.col.operationTime'),
          dataIndex: 'createdAt',
          key: 'createdAt',
          width: 180,
          render: (value: string) => new Date(value).toLocaleString('zh-CN'),
        },
        {
          title: t('common.status'),
          dataIndex: 'status',
          key: 'status',
          width: 110,
          render: (_, record) => renderTaskStatus(record.status, t),
        },
        {
          title: t('ufte.col.destVersion'),
          key: 'targetVersion',
          width: 180,
          render: (_, record) => renderEllipsisCell(getTaskTargetVersion(record)),
        },
        {
          title: t('ufte.col.upgradeType'),
          key: 'upgradeType',
          width: 120,
          render: (_, record) => <Tag color="blue">{getUpgradeTypeLabel(record.category, record.typeDisplayName, t)}</Tag>,
        },
        {
          title: t('ufte.col.productType'),
          key: 'productType',
          width: 130,
          render: (_, record) => renderEllipsisCell(getTaskProductClass(record)),
        },
        {
          title: t('ufte.col.upgradeProgress'),
          dataIndex: 'progress',
          key: 'progress',
          width: 150,
          render: (value: number, record) => (
            <Progress percent={value} size="small" status={record.status === 'ended' ? 'success' : record.status === 'in_progress' ? 'active' : 'normal'} />
          ),
        },
        {
          title: t('ufte.col.result'),
          dataIndex: 'result',
          key: 'result',
          width: 100,
          render: (value) => {
            if (!value) return '-';
            const color = value === 'success' ? 'success' : value === 'partial' ? 'warning' : 'error';
            const label = value === 'success' ? t('ufte.result.success') : value === 'partial' ? t('ufte.result.partial') : value === 'terminated' ? t('ufte.result.terminated') : t('ufte.result.failed');
            return <Tag color={color}>{label}</Tag>;
          },
        },
        {
          title: t('ufte.col.startTime'),
          key: 'startTime',
          width: 180,
          render: (_, record) => record.executionMode === 'scheduled' && record.scheduledAt ? new Date(record.scheduledAt).toLocaleString('zh-CN') : '-',
        },
        {
          title: t('ufte.col.endTime'),
          key: 'endTime',
          width: 180,
          render: (_, record) => record.status === 'ended' ? new Date(record.createdAt).toLocaleString('zh-CN') : '-',
        },
      ];
    }

    return [
      taskActionColumn,
      {
        // 任务名称（非升级类）：默认按 业务_用户_时间 自动生成；任务名前缀已带业务类型
        // （运行日志_admin_... / 配置文件备份_... 等），不再加 typeDisplayName 副行
        // 重复显示。固定 280 + Tooltip 兜底，避免长名挤压后续列。
        title: t('ufte.col.taskName'),
        dataIndex: 'taskName',
        key: 'taskName',
        width: 280,
        render: (_, record) => renderEllipsisCell(record.taskName, { strong: true }),
      },
      {
        title: t('ufte.col.executor'),
        dataIndex: 'createUser',
        key: 'createUser',
        width: 130,
        render: (value: string) => renderEllipsisCell(value),
      },
      {
        title: t('common.status'),
        dataIndex: 'status',
        key: 'status',
        width: 120,
        render: (_, record) => renderTaskStatus(record.status, t),
      },
      // 备份 / 日志采集 / 配置恢复 等 OUTPUT 文件类（非升级类）主任务跨多设备，没有
      // "当前步骤"概念——单设备的 RPC 步骤在设备列表展示。详见
      // docs/project/backup-display-fix-20260520.md F1。
      {
        title: t('ufte.col.progress'),
        dataIndex: 'progress',
        key: 'progress',
        width: 180,
        render: (value: number, record) => (
          <Space direction="vertical" size={4} style={{ width: '100%' }}>
            <Progress percent={value} size="small" status={record.status === 'ended' ? 'success' : 'active'} />
            <Text type="secondary">{t('ufte.col.statsBrief', { ok: record.successCount, fail: record.failCount, total: record.totalCount })}</Text>
          </Space>
        ),
      },
      {
        title: t('ufte.col.executionMode'),
        dataIndex: 'executionMode',
        key: 'executionMode',
        width: 120,
        render: (value) => {
          const label = executionModeOptions.find((item) => item.value === value)?.label ?? value;
          return <Tag>{label}</Tag>;
        },
      },
      {
        title: t('ufte.col.result'),
        dataIndex: 'result',
        key: 'result',
        width: 100,
        render: (value) => {
          if (!value) return '—';
          const color = value === 'success' ? 'success' : value === 'partial' ? 'warning' : 'error';
          const label = value === 'success' ? t('ufte.result.success') : value === 'partial' ? t('ufte.result.partial') : value === 'terminated' ? t('ufte.result.terminated') : t('ufte.result.failed');
          return <Tag color={color}>{label}</Tag>;
        },
      },
      {
        title: t('ufte.col.createdAt'),
        dataIndex: 'createdAt',
        key: 'createdAt',
        width: 180,
        render: (value: string) => new Date(value).toLocaleString('zh-CN'),
      },
    ];
  }, [isUpgradeLikeCategory, taskTypes, t]);

  const deviceColumns: ColumnsType<UnifiedFileTransferDeviceItem> = useMemo(() => {
    if (isUpgradeLikeCategory) {
      return [
        {
          title: t('ufte.col.stationCode'),
          dataIndex: 'deviceSn',
          key: 'deviceSn',
          width: 150,
          render: (value: string) => renderEllipsisCell(value),
        },
        {
          // 自动生成的 taskName 形如 "Upgrade_admin_2026-05-21 14:05:07"（约 35 字符），
          // 180 列宽无法容纳。280 + Tooltip 兜底，跟非升级类设备列表对齐。
          title: t('ufte.col.taskName'),
          dataIndex: 'taskName',
          key: 'taskName',
          width: 280,
          render: (value: string) => renderEllipsisCell(value, { strong: true }),
        },
        {
          title: t('ufte.col.sourceVersion'),
          dataIndex: 'currentVersion',
          key: 'currentVersion',
          width: 140,
          render: (value: string) => renderEllipsisCell(value),
        },
        {
          title: t('ufte.col.destVersion'),
          dataIndex: 'targetVersion',
          key: 'targetVersion',
          width: 160,
          render: (value: string) => renderEllipsisCell(value),
        },
        {
          title: t('ufte.col.upgradeType'),
          key: 'upgradeType',
          width: 120,
          render: (_, record) => <Tag color="blue">{getUpgradeTypeLabel(record.category, record.typeDisplayName, t)}</Tag>,
        },
        {
          title: t('ufte.col.productType'),
          dataIndex: 'productType',
          key: 'productType',
          width: 130,
          render: (value: string) => renderEllipsisCell(value),
        },
        {
          title: t('ufte.col.upgradeProgress'),
          dataIndex: 'progress',
          key: 'progress',
          width: 130,
          render: (value: number, record) => {
            let progressStatus: 'success' | 'exception' | 'active' | 'normal' = 'normal';
            if (record.status === 'ended') progressStatus = 'success';
            else if (record.status === 'failed') progressStatus = 'exception';
            else if (record.status === 'pending' || record.status === 'suspended') progressStatus = 'normal';
            else progressStatus = 'active';
            return <Progress percent={value} size="small" status={progressStatus} />;
          },
        },
        {
          title: t('ufte.col.result'),
          dataIndex: 'status',
          key: 'status',
          width: 110,
          render: (_, record) => renderDeviceStatus(record.status, t),
        },
        {
          title: t('ufte.col.operator'),
          dataIndex: 'operatorScope',
          key: 'operatorScope',
          width: 140,
          render: (value: string) => renderEllipsisCell(value),
        },
        failureReasonColumn,
        {
          title: t('ufte.col.operationTime'),
          dataIndex: 'lastReportAt',
          key: 'lastReportAt',
          width: 180,
          render: (value: string) => new Date(value).toLocaleString('zh-CN'),
        },
      ];
    }

    return [
      {
        // 任务列表场景：每行是「某任务在某设备上的执行情况」。主显示任务名（用户操作
        // 上下文，他刚创建的 testNV 想看这个任务的进展），副显示设备名（区分多设备）。
        // 列宽 280：UFTE 自动生成的任务名形如 "ConfigBackupNV_admin_2026-05-21 13:39:55"
        // 约 36 字符，hover Tooltip 看全名。
        title: t('ufte.col.taskOrDevice'),
        dataIndex: 'taskName',
        key: 'taskName',
        width: 280,
        render: (_, record) => renderEllipsisCell(record.taskName, {
          strong: true,
          subtitle: record.deviceName,
          maxWidth: 260,
        }),
      },
      {
        title: t('ufte.col.deviceSn'),
        dataIndex: 'deviceSn',
        key: 'deviceSn',
        width: 150,
        render: (value: string) => renderEllipsisCell(value),
      },
      {
        title: t('ufte.col.productType'),
        dataIndex: 'productType',
        key: 'productType',
        width: 130,
        render: (value: string) => renderEllipsisCell(value),
      },
      {
        title: t('ufte.col.currentVersion'),
        dataIndex: 'currentVersion',
        key: 'currentVersion',
        width: 140,
        render: (value: string) => renderEllipsisCell(value),
      },
      {
        // 非升级类（备份 / 日志采集 / 配置恢复）：使用后端按 {task_id8}/{sn} 渲染过的
        // targetFile（出现"backup-a1b2c3d4-SN001.nv"形式）。CPE 上传完成且 metadata
        // 落地后，后端附带 downloadUrl，UI 渲染为可点击链接。详见
        // docs/project/backup-display-fix-20260520.md F3。
        // 列宽 240 — 文件名常超 220，Tooltip 兜底完整看到。
        title: t('ufte.col.destVersionOrFile'),
        key: 'targetVersion',
        width: 240,
        render: (_, record) => {
          const file = record.targetFile?.trim();
          if (!file) {
            return '-';
          }
          const inner = record.downloadUrl ? (
            <a href={record.downloadUrl} target="_blank" rel="noopener noreferrer" style={{ display: 'inline-block', maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', verticalAlign: 'bottom' }}>
              {file}
            </a>
          ) : (
            <Text type="secondary" ellipsis style={{ maxWidth: 220 }}>{file}</Text>
          );
          return (
            <Tooltip title={file} placement="topLeft">
              {inner}
            </Tooltip>
          );
        },
      },
      {
        title: t('common.status'),
        dataIndex: 'status',
        key: 'status',
        width: 110,
        render: (_, record) => renderDeviceStatus(record.status, t),
      },
      { title: t('ufte.col.progress'), dataIndex: 'progress', key: 'progress', width: 180, render: (value: number) => <Progress percent={value} size="small" status={value === 100 ? 'success' : 'active'} /> },
      failureReasonColumn,
      {
        title: t('ufte.col.reportTime'),
        dataIndex: 'lastReportAt',
        key: 'lastReportAt',
        width: 180,
        render: (value: string) => new Date(value).toLocaleString('zh-CN'),
      },
    ];
  }, [isUpgradeLikeCategory, t]);

  const openTaskDrawer = (typeCode?: string) => {
    const nextTypeCode = typeCode || selectedTypeCode || activeTaskType?.typeCode;
    if (!nextTypeCode) {
      void message.warning(t('ufte.msg.noTemplate'));
      return;
    }
    const defaultTaskName = buildDefaultTaskName(nextTypeCode, taskNameUser, appLocale);
    lastAutoFilledTaskNameRef.current = defaultTaskName;
    taskForm.setFieldsValue({
      taskName: defaultTaskName,
      typeCode: nextTypeCode,
      productClass: undefined,
      firmwareId: undefined,
      isKeepConfig: true,
      executionMode: 'immediate',
    });
    setSelectedDrawerDeviceIds([]);
    setDrawerDeviceKeyword('');
    setDrawerDeviceKeywordInput('');
    setSelectedTypeCode(nextTypeCode);
    setTaskDrawerOpen(true);
  };

  // 切换"任务类型"时，如用户没动过 taskName（当前值 === 上次自动填的值）→ 重算填入；
  // 已被用户修改过 → 保留用户输入不打扰。
  useEffect(() => {
    if (!taskDrawerOpen || !drawerTypeCode) {
      return;
    }
    const currentTaskName = (taskForm.getFieldValue('taskName') as string | undefined) ?? '';
    if (currentTaskName && currentTaskName !== lastAutoFilledTaskNameRef.current) {
      return;
    }
    const nextName = buildDefaultTaskName(drawerTypeCode, taskNameUser, appLocale);
    lastAutoFilledTaskNameRef.current = nextName;
    taskForm.setFieldValue('taskName', nextName);
  }, [drawerTypeCode, taskDrawerOpen, taskForm, taskNameUser]);

  useEffect(() => {
    if (!taskDrawerOpen) {
      return;
    }
    if (!needsFirmwareSelection(drawerTaskType)) {
      taskForm.setFieldValue('productClass', undefined);
      taskForm.setFieldValue('firmwareId', undefined);
      taskForm.setFieldValue('isKeepConfig', undefined);
    } else {
      if (taskForm.getFieldValue('isKeepConfig') === undefined) {
        taskForm.setFieldValue('isKeepConfig', true);
      }
      if (!drawerProductClass && drawerProductClassOptions.length === 1) {
        taskForm.setFieldValue('productClass', drawerProductClassOptions[0].value);
        return;
      }
      const selectedFirmwareId = taskForm.getFieldValue('firmwareId') as string | undefined;
      if (selectedFirmwareId && !firmwareOptions.some((item) => item.value === selectedFirmwareId)) {
        taskForm.setFieldValue('firmwareId', undefined);
      }
    }

    const validIds = selectedDrawerDeviceIds.filter((id) => drawerDeviceCandidates.some((item) => item.id === id));
    if (validIds.length !== selectedDrawerDeviceIds.length) {
      setSelectedDrawerDeviceIds(validIds);
    }
  }, [drawerDeviceCandidates, drawerProductClass, drawerProductClassOptions, drawerTaskType, firmwareOptions, selectedDrawerDeviceIds, taskDrawerOpen, taskForm]);

  const handleCreateTask = async () => {
    // 防止用户连续点击「创建」按钮重复提交：mutation 进行中直接忽略后续点击。
    // 仅靠按钮 loading 不够 —— validateFields 是异步的，期间 isPending 仍为 false。
    if (createTaskMutation.isPending) {
      return;
    }
    const values = await taskForm.validateFields();
    if (selectedDrawerDeviceIds.length === 0) {
      void message.warning(t('ufte.msg.pickDevice'));
      return;
    }
    // T-0164: CONFIG_RESTORE 整批拒绝 — 任一设备缺快照即阻止提交。
    if (isConfigRestoreBlockedByMissing) {
      void message.error(t('ufte.msg.snapshotMissing', {
        sns: snapshotProbe?.missing.join(', ') ?? '',
      }));
      return;
    }
    // T-0165: LICENSE_UPGRADE 同款整批拒绝 — 任一设备缺 license 即阻止提交。
    if (isLicenseUpgradeBlockedByMissing) {
      void message.error(t('ufte.msg.licenseMissing', {
        sns: licenseProbe?.missing.join(', ') ?? '',
      }));
      return;
    }
    if (createTaskMutation.isPending) {
      return;
    }
    // scheduledAt 在表单里是 dayjs 实例，发请求前转 ISO 字符串（后端 RFC3339 解析）。
    // 非 scheduled 模式 form 不会渲染这个字段 → values.scheduledAt 为 undefined，直接传不影响。
    const rawScheduledAt = (values as { scheduledAt?: unknown }).scheduledAt;
    const scheduledAtIso = values.executionMode === 'scheduled' && rawScheduledAt
      ? (dayjs.isDayjs(rawScheduledAt) ? rawScheduledAt : dayjs(rawScheduledAt as string)).toISOString()
      : undefined;
    void createTaskMutation
      .mutateAsync({
        ...values,
        scheduledAt: scheduledAtIso,
        deviceIds: selectedDrawerDeviceIds,
        deviceCount: selectedDrawerDeviceIds.length,
      })
      .then(() => {
        void message.success(t('ufte.msg.taskCreated'));
        setTaskDrawerOpen(false);
        setSelectedDrawerDeviceIds([]);
        taskForm.resetFields();
      })
      .catch((error: unknown) => {
        void message.error(getTaskActionErrorMessage(error, t('ufte.msg.taskCreateFailed')));
      });
  };

  return (
    <ListPageLayout
      title={t('ufte.page.taskCreate')}
      extra={(
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            // MR 测量 Tab：打开 MR 创建抽屉；其它 Tab：走 UFTE 任务抽屉
            if (selectedCategory === 'mr_measurement') {
              setMrCreateOpen(true);
              return;
            }
            openTaskDrawer();
          }}
          disabled={
            selectedCategory === 'mr_measurement'
              ? false // MR 不依赖 activeTaskType
              : (taskTypesLoading || !activeTaskType)
          }
        >
          {t('ufte.action.newTask')}
        </Button>
      )}
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Card>
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Title level={4} style={{ margin: 0 }}>{t('ufte.page.taskCreate')}</Title>
            <Tabs
              activeKey={selectedCategory}
              items={categories.map((item) => ({
                key: item.category,
                label: localizeBuiltinCategoryLabel(item.category, item.categoryLabel, t),
              }))}
              onChange={(key) => setSelectedCategory(key)}
            />
            {/* MR 测量 Tab 选中时：① 不显示 UFTE 模板子 Tab，② 直接在 Tabs 下方
                内联渲染 MR 任务管理面板（视觉上就是 Tab 切换内容）。 */}
            {selectedCategory === 'mr_measurement' ? (
              <MRTasksPanel
                createOpen={mrCreateOpen}
                onCreateOpenChange={setMrCreateOpen}
              />
            ) : (
              <Tabs
                activeKey={selectedTypeCode}
                items={templateTabItems}
                onChange={(key) => setSelectedTypeCode(key)}
              />
            )}
          </Space>
        </Card>

        {/* 下方"任务列表 / 设备列表"区域：MR 测量 Tab 选中时隐藏（MR 已内嵌在上方 Card） */}
        {selectedCategory !== 'mr_measurement' && (
        <Card title={t('ufte.card.executionView')}>
          <Tabs
            activeKey={viewMode}
            onChange={(key) => setViewMode(key as 'tasks' | 'devices')}
            items={[
              {
                key: 'tasks',
                label: t('ufte.tab.taskList'),
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Space wrap>
                      <SearchInput
                        allowClear
                        placeholder={t('ufte.search.tasks')}
                        value={taskKeywordInput}
                        onChange={(event) => setTaskKeywordInput(event.target.value)}
                        onSearch={(value) => setTaskKeyword(value.trim())}
                        style={{ width: 280 }}
                      />
                      <Select
                        value={selectedTypeCode}
                        onChange={(value) => setSelectedTypeCode(value)}
                        options={taskTypeOptions}
                        style={{ width: 260 }}
                      />
                      <Select
                        allowClear
                        placeholder={t('ufte.filter.status')}
                        value={taskStatusFilter}
                        onChange={(value) => setTaskStatusFilter(value)}
                        options={[
                          { label: t('ufte.status.pending'), value: 'pending' },
                          { label: t('ufte.status.inProgress'), value: 'in_progress' },
                          { label: t('ufte.status.suspended'), value: 'suspended' },
                          { label: t('ufte.status.ended'), value: 'ended' },
                        ]}
                        style={{ width: 160 }}
                      />
                    </Space>
                    {selectedTaskIds.length > 0 && (
                      <Space>
                        <Button
                          danger
                          icon={<DeleteOutlined />}
                          loading={batchDeleteTasksMutation.isPending}
                          onClick={() => {
                            const ids = selectedTaskIds.map(String);
                            Modal.confirm({
                              title: t('ufte.msg.batchDeleteConfirmTitle'),
                              content: t('ufte.msg.batchDeleteConfirmContent', { count: ids.length }),
                              okType: 'danger',
                              onOk: async () => {
                                try {
                                  const res = await batchDeleteTasksMutation.mutateAsync(ids);
                                  if (res.failed.length === 0) {
                                    void message.success(t('ufte.msg.batchDeleteSuccess', { count: res.succeeded.length }));
                                  } else {
                                    void message.warning(t('ufte.msg.batchDeletePartial', {
                                      succeeded: res.succeeded.length,
                                      failed: res.failed.length,
                                      errors: res.failed.map((f) => f.error).join('；'),
                                    }));
                                  }
                                  setSelectedTaskIds((prev) =>
                                    prev.filter((k) => !res.succeeded.includes(String(k))),
                                  );
                                } catch {
                                  void message.error(t('ufte.msg.batchDeleteFailed'));
                                }
                              },
                            });
                          }}
                        >
                          {t('ufte.action.batchDeleteWithCount', { count: selectedTaskIds.length })}
                        </Button>
                        <Button type="link" onClick={() => setSelectedTaskIds([])}>
                          {t('ufte.action.cancelSelection')}
                        </Button>
                      </Space>
                    )}
                    <Table<UnifiedFileTransferTask>
                      rowKey="id"
                      columns={taskColumns}
                      dataSource={recentTasks}
                      loading={tasksLoading}
                      rowSelection={{
                        selectedRowKeys: selectedTaskIds,
                        onChange: (keys) => setSelectedTaskIds(keys),
                      }}
                      pagination={{
                        current: taskPage,
                        pageSize: taskPageSize,
                        total: tasksData?.total ?? 0,
                        showSizeChanger: true,
                        showTotal: (total) => t('ufte.common.totalCount', { count: total }),
                        onChange: (page, pageSize) => {
                          setTaskPage(page);
                          setTaskPageSize(pageSize);
                        },
                      }}
                      scroll={{ x: 2000 }}
                    />
                  </Space>
                ),
              },
              {
                key: 'devices',
                label: t('ufte.tab.deviceList'),
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Space wrap>
                      <SearchInput
                        allowClear
                        placeholder={t('ufte.search.devices')}
                        value={deviceKeywordInput}
                        onChange={(event) => setDeviceKeywordInput(event.target.value)}
                        onSearch={(value) => setDeviceKeyword(value.trim())}
                        style={{ width: 280 }}
                      />
                      <Select
                        value={selectedTypeCode}
                        onChange={(value) => setSelectedTypeCode(value)}
                        options={taskTypeOptions}
                        style={{ width: 260 }}
                      />
                      <Select
                        allowClear
                        showSearch
                        placeholder={t('ufte.filter.productType')}
                        value={deviceProductClassFilter}
                        onChange={(value) => setDeviceProductClassFilter(value)}
                        options={deviceProductClassOptions}
                        optionFilterProp="label"
                        style={{ width: 220 }}
                      />
                      <Select
                        allowClear
                        placeholder={t('ufte.filter.status')}
                        value={deviceStatusFilter}
                        onChange={(value) => setDeviceStatusFilter(value)}
                        options={[
                          { label: t('ufte.status.pending'), value: 'pending' },
                          // 升级 / 回滚类（Download RPC）
                          { label: t('ufte.status.downloading'), value: 'downloading' },
                          // 备份 / 日志采集类（Upload RPC）的两个子阶段
                          { label: t('ufte.status.uploading'), value: 'uploading' },
                          { label: t('ufte.status.awaitingTc'), value: 'awaiting_tc' },
                          { label: t('ufte.status.verifying'), value: 'verifying' },
                          { label: t('ufte.status.suspended'), value: 'suspended' },
                          { label: t('ufte.status.completed'), value: 'ended' },
                          { label: t('ufte.status.failed'), value: 'failed' },
                        ]}
                        style={{ width: 160 }}
                      />
                      <Button
                        icon={<DownloadOutlined />}
                        loading={devicesExporting}
                        onClick={() => { void handleExportDevices(); }}
                      >
                        {t('ufte.action.exportCsv')}
                      </Button>
                    </Space>
                    <Table<UnifiedFileTransferDeviceItem>
                      rowKey="id"
                      columns={deviceColumns}
                      dataSource={recentDevices}
                      loading={devicesLoading}
                      pagination={{
                        current: devicePage,
                        pageSize: devicePageSize,
                        total: devicesData?.total ?? 0,
                        showSizeChanger: true,
                        showTotal: (total) => t('ufte.common.totalCount', { count: total }),
                        onChange: (page, pageSize) => {
                          setDevicePage(page);
                          setDevicePageSize(pageSize);
                        },
                      }}
                      scroll={{ x: 2000 }}
                    />
                  </Space>
                ),
              },
            ]}
          />
        </Card>
        )}
      </Space>

      <Drawer
        title={t('ufte.drawer.newTask')}
        width={520}
        open={taskDrawerOpen}
        onClose={() => setTaskDrawerOpen(false)}
        destroyOnHidden
        extra={(
          <Space>
            <Button onClick={() => setTaskDrawerOpen(false)}>{t('common.cancel')}</Button>
            <Button
              type="primary"
              loading={createTaskMutation.isPending}
              disabled={isConfigRestoreBlockedByMissing || isLicenseUpgradeBlockedByMissing}
              onClick={() => void handleCreateTask()}
            >
              {t('ufte.action.create')}
            </Button>
          </Space>
        )}
      >
        <Form form={taskForm} layout="vertical">
          <Form.Item label={t('ufte.form.taskName')} name="taskName" rules={[{ required: true, message: t('ufte.form.taskName.required') }]}>
            <Input placeholder={t('ufte.form.taskName.placeholder')} />
          </Form.Item>
          <Form.Item label={t('ufte.form.taskType')} name="typeCode" rules={[{ required: true, message: t('ufte.form.taskType.required') }]}>
            <Select
              options={taskTypeOptions}
              placeholder={t('ufte.form.taskType.required')}
              disabled={taskTypeOptions.length <= 1}
            />
          </Form.Item>
          {needsFirmwareSelection(drawerTaskType) ? (
            <>
              <Form.Item label={t('ufte.form.productClass')} name="productClass" rules={[{ required: true, message: t('ufte.form.productClass.required') }]}>
                <Select
                  allowClear
                  showSearch
                  placeholder={t('ufte.form.productClass.required')}
                  options={drawerProductClassOptions}
                  optionFilterProp="label"
                />
              </Form.Item>
              <Form.Item
                label={t('ufte.form.firmware')}
                name="firmwareId"
                rules={[{ required: true, message: t('ufte.form.firmware.required') }]}
                extra={(
                  <Space direction="vertical" size={0}>
                    <Text type="secondary">{t('ufte.form.firmwareLibraryHint', { kind: getSoftwareLibraryFileTypeLabel(firmwareLibraryFileType, t) })}</Text>
                    <Space size={12}>
                      {/* 新窗口打开 — 用户在新 tab 上传完关闭即可回原弹窗，表单状态不丢；
                          回到原 tab 时 React Query 默认 refetchOnWindowFocus 会自动刷新固件列表。 */}
                      <Button
                        type="link"
                        style={{ paddingInline: 0 }}
                        icon={<ExportOutlined />}
                        // 不带 noopener — 让新 tab 内的"完成并关闭"按钮能调 window.close()
                        // 自闭。本应用同源，无被钓鱼风险。
                        // 跳到新的"文件管理 → 版本文件" tab —— 老 /software/firmware
                        // 路由在菜单下线后被 PrivateRoute 路径守卫拦成 403，新入口在
                        // "文件传输 / 文件管理"菜单内，所有角色可达。
                        onClick={() =>
                          window.open('/transfer/file-management?tab=version&return=ufte', '_blank')
                        }
                      >
                        {t('ufte.form.openFirmwareManager')}
                      </Button>
                      {/* 兜底：用户在同 tab 操作完手动回来 / 窗口未失焦时，点这里强刷固件列表。 */}
                      <Button
                        type="link"
                        style={{ paddingInline: 0 }}
                        icon={<ReloadOutlined />}
                        onClick={() => void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] })}
                      >
                        {t('ufte.form.refreshFirmwares')}
                      </Button>
                    </Space>
                  </Space>
                )}
              >
                <Select
                  showSearch
                  allowClear
                  disabled={!drawerProductClass}
                  placeholder={
                    !drawerProductClass
                      ? t('ufte.form.firmware.needProductFirst')
                      : firmwareOptions.length > 0
                        ? t('ufte.form.firmware.required')
                        : t('ufte.form.firmware.empty', { kind: getSoftwareLibraryFileTypeLabel(firmwareLibraryFileType, t) })
                  }
                  options={firmwareOptions}
                  optionFilterProp="label"
                />
              </Form.Item>
              <Form.Item name="isKeepConfig" valuePropName="checked">
                <Checkbox>{t('ufte.form.keepConfig')}</Checkbox>
              </Form.Item>
            </>
          ) : null}
          <Form.Item label={t('ufte.form.deviceSelection')} required>
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Space>
                <Input.Search
                  placeholder={t('ufte.form.deviceSearchPlaceholder')}
                  allowClear
                  style={{ width: 240 }}
                  value={drawerDeviceKeywordInput}
                  onChange={(e) => setDrawerDeviceKeywordInput(e.target.value)}
                  onSearch={(value) => setDrawerDeviceKeyword(value.trim())}
                />
                <Button
                  icon={<PlusOutlined />}
                  onClick={() => { setBatchSNInputOpen(true); setBatchSNInputText(''); }}
                >
                  {t('ufte.action.batchSnInput')}
                </Button>
                <Text type="secondary">{t('ufte.form.selectedCount', { count: drawerSelectedDevices.length })}</Text>
              </Space>
              <Table<UnifiedFileTransferDeviceItem>
                size="small"
                rowKey="id"
                loading={drawerDevicesLoading}
                columns={drawerDeviceColumns}
                dataSource={drawerDeviceCandidates}
                pagination={false}
                rowSelection={{
                  selectedRowKeys: selectedDrawerDeviceIds,
                  onChange: (selectedRowKeys) => {
                    setSelectedDrawerDeviceIds(selectedRowKeys.map((item) => String(item)));
                  },
                }}
                scroll={{ x: 600, y: 240 }}
              />
            </Space>
          </Form.Item>

          {drawerTypeCode === 'LICENSE_UPGRADE' ? (
            <Form.Item
              label={(
                <Space size={8}>
                  <span>{t('ufte.form.licenseSource')}</span>
                  <Button
                    type="primary"
                    size="small"
                    icon={<ExportOutlined />}
                    onClick={() =>
                      window.open('/transfer/file-management?tab=license&return=ufte', '_blank')
                    }
                  >
                    {t('ufte.form.openLicenseManager')}
                  </Button>
                </Space>
              )}
            >
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Text type="secondary">{t('ufte.form.licenseSourceHint')}</Text>

                {drawerSelectedDevices.length === 0 ? (
                  <Text type="secondary">{t('ufte.form.pickDevicesFirstLicense')}</Text>
                ) : licenseProbing ? (
                  <Tag color="processing">{t('ufte.tag.licenseChecking')}</Tag>
                ) : !licenseProbe ? null : (
                  <>
                    <Space size={12}>
                      {licenseMissingCount === 0 ? (
                        <Tag color="success">
                          {t('ufte.form.allDevicesReady', { count: Object.keys(licenseProbe.found).length })}
                        </Tag>
                      ) : (
                        <Tag color="error">
                          {t('ufte.form.devicesMissingLicense', { count: licenseMissingCount })}
                        </Tag>
                      )}
                    </Space>
                    <Table<UnifiedFileTransferDeviceItem>
                      size="small"
                      rowKey="id"
                      dataSource={drawerSelectedDevices}
                      pagination={false}
                      scroll={{ y: 200 }}
                      columns={[
                        { title: t('ufte.col.deviceSn'), dataIndex: 'deviceSn', key: 'sn', width: 180 },
                        {
                          title: t('ufte.col.fileName'),
                          key: 'file',
                          width: 240,
                          ellipsis: true,
                          render: (_, rec) => {
                            const lic = licenseProbe.found[rec.deviceSn];
                            return lic ? (
                              <Text code style={{ fontSize: 12 }}>{lic.fileName}</Text>
                            ) : (
                              <Tag color="error">{t('ufte.tag.missing')}</Tag>
                            );
                          },
                        },
                        {
                          title: t('common.updateTime'),
                          key: 'updateTime',
                          width: 160,
                          render: (_, rec) => {
                            const lic = licenseProbe.found[rec.deviceSn];
                            return lic ? dayjs(lic.updateTime).format('YYYY-MM-DD HH:mm') : '—';
                          },
                        },
                      ]}
                    />
                  </>
                )}
              </Space>
            </Form.Item>
          ) : null}

          {drawerTypeCode === 'CONFIG_RESTORE' ? (
            <Form.Item
              label={(
                <Space size={8}>
                  <span>{t('ufte.form.configSource')}</span>
                  <Button
                    type="primary"
                    size="small"
                    icon={<ExportOutlined />}
                    // 走新 tab，让原"任务创建"抽屉状态（已选设备、表单值）不丢。
                    // 用户在新 tab 维护完文件后点页面顶部"完成并关闭"自闭，回到本 tab。
                    onClick={() =>
                      window.open('/transfer/file-management?tab=config&return=ufte', '_blank')
                    }
                  >
                    {t('ufte.form.openConfigManager')}
                  </Button>
                </Space>
              )}
            >
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Text type="secondary">{t('ufte.form.configSourceHint')}</Text>

                {drawerSelectedDevices.length === 0 ? (
                  <Text type="secondary">{t('ufte.form.pickDevicesFirstSnapshot')}</Text>
                ) : snapshotProbing ? (
                  <Tag color="processing">{t('ufte.tag.snapshotChecking')}</Tag>
                ) : !snapshotProbe ? null : (
                  <>
                    <Space size={12}>
                      {snapshotMissingCount === 0 ? (
                        <Tag color="success">
                          {t('ufte.form.allDevicesReady', { count: Object.keys(snapshotProbe.found).length })}
                        </Tag>
                      ) : (
                        <Tag color="error">
                          {t('ufte.form.devicesMissingSnapshot', { count: snapshotMissingCount })}
                        </Tag>
                      )}
                    </Space>
                    <Table<UnifiedFileTransferDeviceItem>
                      size="small"
                      rowKey="id"
                      dataSource={drawerSelectedDevices}
                      pagination={false}
                      scroll={{ y: 200 }}
                      columns={[
                        { title: t('ufte.col.deviceSn'), dataIndex: 'deviceSn', key: 'sn', width: 180 },
                        {
                          title: t('ufte.col.fileName'),
                          key: 'file',
                          width: 240,
                          ellipsis: true,
                          render: (_, rec) => {
                            const snap = snapshotProbe.found[rec.deviceSn];
                            return snap ? (
                              <Text code style={{ fontSize: 12 }}>{snap.fileName}</Text>
                            ) : (
                              <Tag color="error">{t('ufte.tag.missing')}</Tag>
                            );
                          },
                        },
                        {
                          title: t('common.updateTime'),
                          key: 'updateTime',
                          width: 160,
                          render: (_, rec) => {
                            const snap = snapshotProbe.found[rec.deviceSn];
                            return snap ? dayjs(snap.updateTime).format('YYYY-MM-DD HH:mm') : '—';
                          },
                        },
                        {
                          title: t('ufte.col.source'),
                          key: 'source',
                          width: 100,
                          render: (_, rec) => {
                            const snap = snapshotProbe.found[rec.deviceSn];
                            if (!snap) return null;
                            return snap.source === 'backup' ? (
                              <Tag color="blue">{t('ufte.tag.backup')}</Tag>
                            ) : (
                              <Tag color="green">{t('ufte.tag.manualImport')}</Tag>
                            );
                          },
                        },
                      ]}
                    />
                  </>
                )}
              </Space>
            </Form.Item>
          ) : null}

          <Form.Item label={t('ufte.form.executionMode')} name="executionMode" rules={[{ required: true, message: t('ufte.form.executionMode.required') }]}>
            <Radio.Group options={createExecutionModeOptions} optionType="button" buttonStyle="solid" />
          </Form.Item>
          {/*
            定时执行：仅 executionMode='scheduled' 时显示日期选择器；其它模式 form value 留空。
            shouldUpdate 监听 executionMode 字段变化决定是否渲染。校验：必填 + 大于当前时间。
            提交时 handleCreateTask 走 form.getFieldValue('scheduledAt')（dayjs 对象）→ .toISOString()。
          */}
          <Form.Item noStyle shouldUpdate={(prev, curr) => prev.executionMode !== curr.executionMode}>
            {({ getFieldValue }) =>
              getFieldValue('executionMode') === 'scheduled' ? (
                <Form.Item
                  label={t('ufte.form.scheduledAt')}
                  name="scheduledAt"
                  rules={[
                    { required: true, message: t('ufte.form.scheduledAt.required') },
                    {
                      validator: (_, value) => {
                        if (!value) return Promise.resolve();
                        const target = dayjs.isDayjs(value) ? value : dayjs(value);
                        return target.isAfter(dayjs())
                          ? Promise.resolve()
                          : Promise.reject(new Error(t('ufte.form.scheduledAt.future')));
                      },
                    },
                  ]}
                >
                  <DatePicker
                    showTime
                    format="YYYY-MM-DD HH:mm:ss"
                    style={{ width: 240 }}
                    placeholder={t('ufte.form.scheduledAt.placeholder')}
                    disabledDate={(current) => current && current.isBefore(dayjs().startOf('day'))}
                  />
                </Form.Item>
              ) : null
            }
          </Form.Item>
          <Form.Item label={t('ufte.form.note')} name="note">
            <Input.TextArea rows={4} placeholder={t('ufte.form.note.placeholder')} />
          </Form.Item>
        </Form>

        {/* 批量输入 SN 弹窗 — 给当前任务的设备表加批量勾选入口 */}
        <Modal
          title={t('ufte.batchSnModal.title')}
          open={batchSNInputOpen}
          onCancel={() => setBatchSNInputOpen(false)}
          onOk={() => { void handleApplyBatchSNs(); }}
          okText={t('common.confirm')}
          cancelText={t('common.cancel')}
          confirmLoading={batchSNApplying}
          width={520}
          destroyOnHidden
        >
          <Form layout="vertical">
            <Form.Item label={t('ufte.batchSnModal.label')}>
              <Input.TextArea
                rows={6}
                placeholder={t('ufte.batchSnModal.placeholder')}
                value={batchSNInputText}
                onChange={(e) => setBatchSNInputText(e.target.value)}
                allowClear
              />
            </Form.Item>
            <Text type="secondary" style={{ fontSize: 12 }}>{t('ufte.batchSnModal.hint')}</Text>
          </Form>
        </Modal>
      </Drawer>

      <Drawer
        title={detailTask ? t('ufte.drawer.taskDetailWithName', { name: detailTask.taskName }) : t('ufte.drawer.taskDetail')}
        width={640}
        open={detailDrawerOpen}
        onClose={() => { setDetailDrawerOpen(false); setDetailTask(null); }}
        destroyOnHidden
      >
        {detailTask ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label={t('ufte.col.taskName')}>{detailTask.taskName}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.taskType')}>{localizeBuiltinTypeName(detailTask.typeCode, detailTask.typeDisplayName, t)}</Descriptions.Item>
              <Descriptions.Item label={t('common.status')}>{renderTaskStatus(detailTask.status, t)}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.result')}>
                {detailTask.result
                  ? <Tag color={detailTask.result === 'success' ? 'success' : detailTask.result === 'partial' ? 'warning' : 'error'}>{detailTask.result === 'success' ? t('ufte.result.success') : detailTask.result === 'partial' ? t('ufte.result.partial') : detailTask.result === 'terminated' ? t('ufte.result.terminated') : t('ufte.result.failed')}</Tag>
                  : '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.destVersion')}>{getTaskTargetVersion(detailTask)}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.productType')}>{getTaskProductClass(detailTask)}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.executionMode')}>
                <Tag>{executionModeOptions.find((o) => o.value === detailTask.executionMode)?.label ?? detailTask.executionMode}</Tag>
              </Descriptions.Item>
              {detailTask.executionMode === 'scheduled' && detailTask.scheduledAt ? (
                <Descriptions.Item label={t('ufte.form.scheduledAt')}>
                  {new Date(detailTask.scheduledAt).toLocaleString('zh-CN')}
                </Descriptions.Item>
              ) : null}
              <Descriptions.Item label={t('ufte.col.currentStep')}>{stepLabels[detailTask.currentStep as TransferStepId] ?? '-'}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.operator')}>{detailTask.createUser}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.createdAt')}>{new Date(detailTask.createdAt).toLocaleString('zh-CN')}</Descriptions.Item>
            </Descriptions>
            <Card title={t('ufte.card.executionProgress')} size="small">
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Progress
                  type="circle"
                  percent={detailTask.progress}
                  format={() => `${detailTask.progress}%`}
                />
                <Space wrap>
                  <Tag color="success">{t('ufte.tag.successCount', { count: detailTask.successCount })}</Tag>
                  <Tag color="error">{t('ufte.tag.failedCount', { count: detailTask.failCount })}</Tag>
                  <Tag>{t('ufte.tag.totalCount', { count: detailTask.totalCount })}</Tag>
                </Space>
              </Space>
            </Card>
          </Space>
        ) : null}
      </Drawer>
    </ListPageLayout>
  );
}