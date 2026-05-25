import { useEffect, useMemo, useRef, useState } from 'react';
import dayjs from 'dayjs';
import {
  Alert,
  Button,
  Card,
  Checkbox,
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
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useT } from '@/hooks/useT';

import ListPageLayout from '@/components/Layout/ListPageLayout';
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
  EXECUTION_MODE_OPTIONS,
  buildDefaultUfteTaskName,
  getSoftwareLibraryFileTypeLabel,
  renderDeviceStatus,
  renderEllipsisCell,
  renderTaskStatus,
  STEP_LABELS,
  UPGRADE_LIKE_CATEGORIES,
} from '../shared';
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

function getUpgradeTypeLabel(category: string, fallback: string) {
  if (category === 'gnb_upgrade' || category === 'enb_upgrade') {
    return '软件升级';
  }
  if (category === 'version_rollback') {
    return '版本回退';
  }
  return fallback;
}

export default function FileTransferCenter() {
  const navigate = useNavigate();
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
  const categories = useMemo(() => buildCategoryTabs(taskTypes), [taskTypes]);
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
    () => filteredTaskTypes.map((item) => ({ label: item.displayName, value: item.typeCode })),
    [filteredTaskTypes],
  );

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

  const createExecutionModeOptions = useMemo(
    () => EXECUTION_MODE_OPTIONS.filter((item) => item.value !== 'scheduled'),
    [],
  );

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
          <span>{item.displayName}</span>
          <Tag color={item.builtIn ? 'blue' : 'gold'}>{item.builtIn ? '内置' : '自定义'}</Tag>
        </Space>
      ),
    })),
    [filteredTaskTypes],
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
      { title: '设备 SN', dataIndex: 'deviceSn', key: 'deviceSn', width: 160 },
      { title: '站点名称', dataIndex: 'deviceName', key: 'deviceName', ellipsis: true },
      // 后端 UFTE DeviceItem 字段名是 productType（见 internal/ufte/model.go），
      // 不是 productClass —— 前端原 dataIndex 写错，真实数据下永远空。
      { title: '产品类型', dataIndex: 'productType', key: 'productType', width: 140 },
      { title: '当前版本', dataIndex: 'currentVersion', key: 'currentVersion', width: 120 },
    ],
    [],
  );

  // Failure reason i18n — mirrors UpgradePlan renderFailureReason
  const DEVICE_CODES = new Set(['DOWNLOAD_FAULT', 'TC_FAULT', 'UPGRADE_5G_FAILED']);
  const TIMEOUT_CODES = new Set(['DOWNLOAD_TIMEOUT', 'TASK_TIMEOUT']);

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
      .then(() => void message.success('任务已启动'))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, '任务启动失败')));
  };

  const handleSuspendTask = (record: UnifiedFileTransferTask) => {
    void suspendTaskMutation.mutateAsync(record.id)
      .then(() => void message.success('任务已暂停'))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, '任务暂停失败')));
  };

  const handleTerminateTask = (record: UnifiedFileTransferTask) => {
    void terminateTaskMutation.mutateAsync(record.id)
      .then(() => void message.success('任务已终止'))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, '任务终止失败')));
  };

  const handleDeleteTask = (record: UnifiedFileTransferTask) => {
    void deleteTaskMutation.mutateAsync(record.id)
      .then(() => void message.success('任务已删除'))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, '任务删除失败')));
  };

  // 批量输入 SN → 解析 → 全量候选匹配 → 并入选中。
  // 分隔符：; , 空格 Tab 换行（任一）；自动 trim、去重、忽略空串。
  const handleApplyBatchSNs = async () => {
    const raw = batchSNInputText || '';
    const tokens = Array.from(new Set(
      raw.split(/[;,\s]+/).map((s) => s.trim()).filter(Boolean),
    ));
    if (tokens.length === 0) {
      void message.warning('请输入至少一个 SN');
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
        void message.success(`已批量选中 ${matchedIds.length} 台`);
      } else {
        void message.warning(
          `成功 ${matchedIds.length} 台；未匹配 ${unmatched.length} 个 SN：${unmatched.slice(0, 5).join('、')}${unmatched.length > 5 ? '…' : ''}`,
        );
      }
      setBatchSNInputOpen(false);
      setBatchSNInputText('');
    } catch (e) {
      const msg = e instanceof Error ? e.message : '请稍后重试';
      void message.error(`批量输入失败：${msg}`);
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
        void message.warning('当前过滤条件下没有设备数据可导出');
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
      void message.success('已开始下载');
    } catch (e) {
      const msg = e instanceof Error ? e.message : '请稍后重试';
      void message.error(`导出失败：${msg}`);
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
          详情
        </Button>
        {showStart ? (
          <Button
            type="link"
            size="small"
            loading={startTaskMutation.isPending}
            onClick={() => handleStartTask(record)}
            style={{ paddingInline: 0 }}
          >
            开始
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
            暂停
          </Button>
        ) : null}
        {showTerminate ? (
          <Popconfirm title="确认终止该任务？" onConfirm={() => handleTerminateTask(record)}>
            <Button type="link" size="small" danger loading={terminateTaskMutation.isPending} style={{ paddingInline: 0 }}>
              终止
            </Button>
          </Popconfirm>
        ) : null}
        {showDelete ? (
          <Popconfirm title="确认删除该任务？" onConfirm={() => handleDeleteTask(record)}>
            <Button type="link" size="small" danger loading={deleteTaskMutation.isPending} style={{ paddingInline: 0 }}>
              删除
            </Button>
          </Popconfirm>
        ) : null}
      </Space>
    );
  };

  const taskActionColumn = {
    title: '操作',
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
          title: '任务名称',
          dataIndex: 'taskName',
          key: 'taskName',
          width: 280,
          render: (_, record) => renderEllipsisCell(record.taskName, { strong: true }),
        },
        {
          title: '操作人',
          dataIndex: 'createUser',
          key: 'createUser',
          width: 130,
          render: (value: string) => renderEllipsisCell(value),
        },
        {
          title: '操作时间',
          dataIndex: 'createdAt',
          key: 'createdAt',
          width: 180,
          render: (value: string) => new Date(value).toLocaleString('zh-CN'),
        },
        {
          title: '状态',
          dataIndex: 'status',
          key: 'status',
          width: 110,
          render: (_, record) => renderTaskStatus(record.status),
        },
        {
          title: '目标版本',
          key: 'targetVersion',
          width: 180,
          render: (_, record) => renderEllipsisCell(getTaskTargetVersion(record)),
        },
        {
          title: '升级类型',
          key: 'upgradeType',
          width: 120,
          render: (_, record) => <Tag color="blue">{getUpgradeTypeLabel(record.category, record.typeDisplayName)}</Tag>,
        },
        {
          title: '产品类型',
          key: 'productType',
          width: 130,
          render: (_, record) => renderEllipsisCell(getTaskProductClass(record)),
        },
        {
          title: '升级进度',
          dataIndex: 'progress',
          key: 'progress',
          width: 150,
          render: (value: number, record) => (
            <Progress percent={value} size="small" status={record.status === 'ended' ? 'success' : record.status === 'in_progress' ? 'active' : 'normal'} />
          ),
        },
        {
          title: '结果',
          dataIndex: 'result',
          key: 'result',
          width: 100,
          render: (value) => {
            if (!value) return '-';
            const color = value === 'success' ? 'success' : value === 'partial' ? 'warning' : 'error';
            const label = value === 'success' ? '成功' : value === 'partial' ? '部分成功' : value === 'terminated' ? '已终止' : '失败';
            return <Tag color={color}>{label}</Tag>;
          },
        },
        {
          title: '开始时间',
          key: 'startTime',
          width: 180,
          render: (_, record) => record.executionMode === 'scheduled' && record.scheduledAt ? new Date(record.scheduledAt).toLocaleString('zh-CN') : '-',
        },
        {
          title: '结束时间',
          key: 'endTime',
          width: 180,
          render: (_, record) => record.status === 'ended' ? new Date(record.createdAt).toLocaleString('zh-CN') : '-',
        },
      ];
    }

    return [
      taskActionColumn,
      {
        // 任务名称（非升级类）：默认按 业务_用户_时间 自动生成；副行展示业务类型。
        // 固定 280 + Tooltip 兜底，避免长名挤压后续列。
        title: '任务名称',
        dataIndex: 'taskName',
        key: 'taskName',
        width: 280,
        render: (_, record) => renderEllipsisCell(record.taskName, {
          strong: true,
          subtitle: record.typeDisplayName,
          maxWidth: 260,
        }),
      },
      {
        title: '执行人',
        dataIndex: 'createUser',
        key: 'createUser',
        width: 130,
        render: (value: string) => renderEllipsisCell(value),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 120,
        render: (_, record) => renderTaskStatus(record.status),
      },
      // 备份 / 日志采集 / 配置恢复 等 OUTPUT 文件类（非升级类）主任务跨多设备，没有
      // "当前步骤"概念——单设备的 RPC 步骤在设备列表展示。详见
      // docs/project/backup-display-fix-20260520.md F1。
      {
        title: '进度',
        dataIndex: 'progress',
        key: 'progress',
        width: 180,
        render: (value: number, record) => (
          <Space direction="vertical" size={4} style={{ width: '100%' }}>
            <Progress percent={value} size="small" status={record.status === 'ended' ? 'success' : 'active'} />
            <Text type="secondary">成功 {record.successCount} / 失败 {record.failCount} / 总数 {record.totalCount}</Text>
          </Space>
        ),
      },
      {
        title: '执行方式',
        dataIndex: 'executionMode',
        key: 'executionMode',
        width: 120,
        render: (value) => {
          const label = EXECUTION_MODE_OPTIONS.find((item) => item.value === value)?.label ?? value;
          return <Tag>{label}</Tag>;
        },
      },
      {
        title: '结果',
        dataIndex: 'result',
        key: 'result',
        width: 100,
        render: (value) => {
          if (!value) return '—';
          const color = value === 'success' ? 'success' : value === 'partial' ? 'warning' : 'error';
          const label = value === 'success' ? '成功' : value === 'partial' ? '部分成功' : value === 'terminated' ? '已终止' : '失败';
          return <Tag color={color}>{label}</Tag>;
        },
      },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        key: 'createdAt',
        width: 180,
        render: (value: string) => new Date(value).toLocaleString('zh-CN'),
      },
    ];
  }, [isUpgradeLikeCategory, taskTypes]);

  const deviceColumns: ColumnsType<UnifiedFileTransferDeviceItem> = useMemo(() => {
    if (isUpgradeLikeCategory) {
      return [
        {
          title: '基站编码',
          dataIndex: 'deviceSn',
          key: 'deviceSn',
          width: 150,
          render: (value: string) => renderEllipsisCell(value),
        },
        {
          // 自动生成的 taskName 形如 "Upgrade_admin_2026-05-21 14:05:07"（约 35 字符），
          // 180 列宽无法容纳。280 + Tooltip 兜底，跟非升级类设备列表对齐。
          title: '任务名称',
          dataIndex: 'taskName',
          key: 'taskName',
          width: 280,
          render: (value: string) => renderEllipsisCell(value, { strong: true }),
        },
        {
          title: '源版本',
          dataIndex: 'currentVersion',
          key: 'currentVersion',
          width: 140,
          render: (value: string) => renderEllipsisCell(value),
        },
        {
          title: '目标版本',
          dataIndex: 'targetVersion',
          key: 'targetVersion',
          width: 160,
          render: (value: string) => renderEllipsisCell(value),
        },
        {
          title: '升级类型',
          key: 'upgradeType',
          width: 120,
          render: (_, record) => <Tag color="blue">{getUpgradeTypeLabel(record.category, record.typeDisplayName)}</Tag>,
        },
        {
          title: '产品类型',
          dataIndex: 'productType',
          key: 'productType',
          width: 130,
          render: (value: string) => renderEllipsisCell(value),
        },
        {
          title: '升级进度',
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
          title: '结果',
          dataIndex: 'status',
          key: 'status',
          width: 110,
          render: (_, record) => renderDeviceStatus(record.status),
        },
        {
          title: '操作人',
          dataIndex: 'operatorScope',
          key: 'operatorScope',
          width: 140,
          render: (value: string) => renderEllipsisCell(value),
        },
        failureReasonColumn,
        {
          title: '操作时间',
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
        title: '任务/设备',
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
        title: '设备 SN',
        dataIndex: 'deviceSn',
        key: 'deviceSn',
        width: 150,
        render: (value: string) => renderEllipsisCell(value),
      },
      {
        title: '产品类型',
        dataIndex: 'productType',
        key: 'productType',
        width: 130,
        render: (value: string) => renderEllipsisCell(value),
      },
      {
        title: '当前版本',
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
        title: '目标版本/目标文件',
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
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 110,
        render: (_, record) => renderDeviceStatus(record.status),
      },
      { title: '进度', dataIndex: 'progress', key: 'progress', width: 180, render: (value: number) => <Progress percent={value} size="small" status={value === 100 ? 'success' : 'active'} /> },
      failureReasonColumn,
      {
        title: '上报时间',
        dataIndex: 'lastReportAt',
        key: 'lastReportAt',
        width: 180,
        render: (value: string) => new Date(value).toLocaleString('zh-CN'),
      },
    ];
  }, [isUpgradeLikeCategory]);

  const openTaskDrawer = (typeCode?: string) => {
    const nextTypeCode = typeCode || selectedTypeCode || activeTaskType?.typeCode;
    if (!nextTypeCode) {
      void message.warning('当前业务视图下还没有模板，请联系管理员先维护模板。');
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
      void message.warning('请选择设备。');
      return;
    }
    // T-0164: CONFIG_RESTORE 整批拒绝 — 任一设备缺快照即阻止提交。
    if (isConfigRestoreBlockedByMissing) {
      void message.error(
        `以下设备无可用配置快照，请先备份或在"配置快照库"导入：${snapshotProbe?.missing.join(', ') ?? ''}`,
      );
      return;
    }
    // T-0165: LICENSE_UPGRADE 同款整批拒绝 — 任一设备缺 license 即阻止提交。
    if (isLicenseUpgradeBlockedByMissing) {
      void message.error(
        `以下设备未上传 license，请先在"文件管理 → License 文件"导入：${licenseProbe?.missing.join(', ') ?? ''}`,
      );
      return;
    }
    if (createTaskMutation.isPending) {
      return;
    }
    void createTaskMutation
      .mutateAsync({
        ...values,
        deviceIds: selectedDrawerDeviceIds,
        deviceCount: selectedDrawerDeviceIds.length,
      })
      .then(() => {
        void message.success('任务已创建。');
        setTaskDrawerOpen(false);
        setSelectedDrawerDeviceIds([]);
        taskForm.resetFields();
      })
      .catch((error: unknown) => {
        void message.error(getTaskActionErrorMessage(error, '创建任务失败，请稍后重试'));
      });
  };

  return (
    <ListPageLayout
      title="任务创建"
      extra={(
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => openTaskDrawer()}
          disabled={taskTypesLoading || !activeTaskType}
        >
          新建任务
        </Button>
      )}
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Card>
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Title level={4} style={{ margin: 0 }}>任务创建</Title>
            <Tabs
              activeKey={selectedCategory}
              items={categories.map((item) => ({ key: item.category, label: item.categoryLabel }))}
              onChange={(key) => setSelectedCategory(key)}
            />
            <Tabs
              activeKey={selectedTypeCode}
              items={templateTabItems}
              onChange={(key) => setSelectedTypeCode(key)}
            />
          </Space>
        </Card>

        <Card title="执行视图">
          <Tabs
            activeKey={viewMode}
            onChange={(key) => setViewMode(key as 'tasks' | 'devices')}
            items={[
              {
                key: 'tasks',
                label: '任务列表',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Space wrap>
                      <Input.Search
                        allowClear
                        placeholder="按任务名称、类型搜索"
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
                        placeholder="按状态过滤"
                        value={taskStatusFilter}
                        onChange={(value) => setTaskStatusFilter(value)}
                        options={[
                          { label: '待执行', value: 'pending' },
                          { label: '执行中', value: 'in_progress' },
                          { label: '已挂起', value: 'suspended' },
                          { label: '已结束', value: 'ended' },
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
                              title: '确认批量删除？',
                              content: `将删除 ${ids.length} 个任务（含其设备子任务 + MinIO 备份/日志文件），不可恢复。运行中的任务请先终止。`,
                              okType: 'danger',
                              onOk: async () => {
                                try {
                                  const res = await batchDeleteTasksMutation.mutateAsync(ids);
                                  if (res.failed.length === 0) {
                                    void message.success(`已删除 ${res.succeeded.length} 个任务`);
                                  } else {
                                    void message.warning(
                                      `成功 ${res.succeeded.length}，失败 ${res.failed.length}：${res.failed.map((f) => f.error).join('；')}`,
                                    );
                                  }
                                  setSelectedTaskIds((prev) =>
                                    prev.filter((k) => !res.succeeded.includes(String(k))),
                                  );
                                } catch {
                                  void message.error('批量删除请求失败，请稍后重试');
                                }
                              },
                            });
                          }}
                        >
                          批量删除（{selectedTaskIds.length}）
                        </Button>
                        <Button type="link" onClick={() => setSelectedTaskIds([])}>
                          取消选择
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
                        showTotal: (total) => `共 ${total} 条`,
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
                label: '设备列表',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Space wrap>
                      <Input.Search
                        allowClear
                        placeholder="按设备名称、SN、任务名称搜索"
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
                        placeholder="按产品类型过滤"
                        value={deviceProductClassFilter}
                        onChange={(value) => setDeviceProductClassFilter(value)}
                        options={deviceProductClassOptions}
                        optionFilterProp="label"
                        style={{ width: 220 }}
                      />
                      <Select
                        allowClear
                        placeholder="按状态过滤"
                        value={deviceStatusFilter}
                        onChange={(value) => setDeviceStatusFilter(value)}
                        options={[
                          { label: '待执行', value: 'pending' },
                          // 升级 / 回滚类（Download RPC）
                          { label: '下载中', value: 'downloading' },
                          // 备份 / 日志采集类（Upload RPC）的两个子阶段
                          { label: '上传中', value: 'uploading' },
                          { label: '等待 TransferComplete', value: 'awaiting_tc' },
                          { label: '校验中', value: 'verifying' },
                          { label: '已挂起', value: 'suspended' },
                          { label: '已完成', value: 'ended' },
                          { label: '失败', value: 'failed' },
                        ]}
                        style={{ width: 160 }}
                      />
                      <Button
                        icon={<DownloadOutlined />}
                        loading={devicesExporting}
                        onClick={() => { void handleExportDevices(); }}
                      >
                        导出 CSV
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
                        showTotal: (total) => `共 ${total} 条`,
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
      </Space>

      <Drawer
        title="新建任务"
        width={520}
        open={taskDrawerOpen}
        onClose={() => setTaskDrawerOpen(false)}
        destroyOnClose
        extra={(
          <Space>
            <Button onClick={() => setTaskDrawerOpen(false)}>取消</Button>
            <Button
              type="primary"
              loading={createTaskMutation.isPending}
              disabled={isConfigRestoreBlockedByMissing || isLicenseUpgradeBlockedByMissing}
              onClick={() => void handleCreateTask()}
            >
              创建
            </Button>
          </Space>
        )}
      >
        <Form form={taskForm} layout="vertical">
          <Form.Item label="任务名称" name="taskName" rules={[{ required: true, message: '请输入任务名称' }]}>
            <Input placeholder="例如：Upgrade_admin_2026-05-21 05:33:55（默认按业务_用户_时间生成，可改）" />
          </Form.Item>
          <Form.Item label="任务类型" name="typeCode" rules={[{ required: true, message: '请选择任务类型' }]}> 
            <Select
              options={taskTypeOptions}
              placeholder="请选择任务类型"
              disabled={taskTypeOptions.length <= 1}
            />
          </Form.Item>
          {needsFirmwareSelection(drawerTaskType) ? (
            <>
              <Form.Item label="产品类型" name="productClass" rules={[{ required: true, message: '请选择产品类型' }]}> 
                <Select
                  allowClear
                  showSearch
                  placeholder="请选择产品类型"
                  options={drawerProductClassOptions}
                  optionFilterProp="label"
                />
              </Form.Item>
              <Form.Item
                label="升级文件"
                name="firmwareId"
                rules={[{ required: true, message: '请选择升级文件' }]}
                extra={(
                  <Space direction="vertical" size={0}>
                    <Text type="secondary">当前模板查询的软件库分类：{getSoftwareLibraryFileTypeLabel(firmwareLibraryFileType)}</Text>
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
                        维护升级文件（新窗口）
                      </Button>
                      {/* 兜底：用户在同 tab 操作完手动回来 / 窗口未失焦时，点这里强刷固件列表。 */}
                      <Button
                        type="link"
                        style={{ paddingInline: 0 }}
                        icon={<ReloadOutlined />}
                        onClick={() => void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] })}
                      >
                        刷新固件列表
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
                      ? '请先选择产品类型'
                      : firmwareOptions.length > 0
                        ? '请选择升级文件'
                        : `暂无可用${getSoftwareLibraryFileTypeLabel(firmwareLibraryFileType)}，请先到软件管理对应分类上传`
                  }
                  options={firmwareOptions}
                  optionFilterProp="label"
                />
              </Form.Item>
              <Form.Item name="isKeepConfig" valuePropName="checked">
                <Checkbox>保留配置</Checkbox>
              </Form.Item>
            </>
          ) : null}
          <Form.Item label="选择设备" required>
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Space>
                <Input.Search
                  placeholder="按 SN 搜索设备"
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
                  批量输入 SN
                </Button>
                <Text type="secondary">已选 {drawerSelectedDevices.length} 台</Text>
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
                  <span>License 文件来源（按设备最新 license）</span>
                  <Button
                    type="primary"
                    size="small"
                    icon={<ExportOutlined />}
                    onClick={() =>
                      window.open('/transfer/file-management?tab=license&return=ufte', '_blank')
                    }
                  >
                    打开 License 文件管理
                  </Button>
                </Space>
              )}
            >
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Text type="secondary">
                  每台设备升级时使用各自最新一份 license；缺失则整批拒绝。
                  点击右上方「打开 License 文件管理」可上传新文件、删除旧文件。
                </Text>

                {drawerSelectedDevices.length === 0 ? (
                  <Text type="secondary">请先在上方选择设备，下表自动展示 license 详情。</Text>
                ) : licenseProbing ? (
                  <Tag color="processing">检查 license 中…</Tag>
                ) : !licenseProbe ? null : (
                  <>
                    <Space size={12}>
                      {licenseMissingCount === 0 ? (
                        <Tag color="success">
                          全部设备已就绪（{Object.keys(licenseProbe.found).length} 台）
                        </Tag>
                      ) : (
                        <Tag color="error">
                          {licenseMissingCount} 台设备无可用 license，整批不能提交
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
                        { title: '设备 SN', dataIndex: 'deviceSn', key: 'sn', width: 180 },
                        {
                          title: '文件名称',
                          key: 'file',
                          width: 240,
                          ellipsis: true,
                          render: (_, rec) => {
                            const lic = licenseProbe.found[rec.deviceSn];
                            return lic ? (
                              <Text code style={{ fontSize: 12 }}>{lic.fileName}</Text>
                            ) : (
                              <Tag color="error">缺失</Tag>
                            );
                          },
                        },
                        {
                          title: '更新时间',
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
                  <span>配置文件来源（按设备最新快照）</span>
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
                    打开配置文件管理
                  </Button>
                </Space>
              )}
            >
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Text type="secondary">
                  每台设备恢复时使用各自最新一份配置快照；缺失则整批拒绝。
                  点击右上方「打开配置文件管理」可查看所有快照、导入新文件、删除旧文件。
                </Text>

                {drawerSelectedDevices.length === 0 ? (
                  <Text type="secondary">请先在上方选择设备，下表自动展示快照详情。</Text>
                ) : snapshotProbing ? (
                  <Tag color="processing">检查快照中…</Tag>
                ) : !snapshotProbe ? null : (
                  <>
                    <Space size={12}>
                      {snapshotMissingCount === 0 ? (
                        <Tag color="success">
                          全部设备已就绪（{Object.keys(snapshotProbe.found).length} 台）
                        </Tag>
                      ) : (
                        <Tag color="error">
                          {snapshotMissingCount} 台设备无可用快照，整批不能提交
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
                        { title: '设备 SN', dataIndex: 'deviceSn', key: 'sn', width: 180 },
                        {
                          title: '文件名称',
                          key: 'file',
                          width: 240,
                          ellipsis: true,
                          render: (_, rec) => {
                            const snap = snapshotProbe.found[rec.deviceSn];
                            return snap ? (
                              <Text code style={{ fontSize: 12 }}>{snap.fileName}</Text>
                            ) : (
                              <Tag color="error">缺失</Tag>
                            );
                          },
                        },
                        {
                          title: '更新时间',
                          key: 'updateTime',
                          width: 160,
                          render: (_, rec) => {
                            const snap = snapshotProbe.found[rec.deviceSn];
                            return snap ? dayjs(snap.updateTime).format('YYYY-MM-DD HH:mm') : '—';
                          },
                        },
                        {
                          title: '来源',
                          key: 'source',
                          width: 100,
                          render: (_, rec) => {
                            const snap = snapshotProbe.found[rec.deviceSn];
                            if (!snap) return null;
                            return snap.source === 'backup' ? (
                              <Tag color="blue">备份</Tag>
                            ) : (
                              <Tag color="green">手动导入</Tag>
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

          <Form.Item label="执行方式" name="executionMode" rules={[{ required: true, message: '请选择执行方式' }]}>
            <Radio.Group options={createExecutionModeOptions} optionType="button" buttonStyle="solid" />
          </Form.Item>
          <Form.Item label="备注" name="note">
            <Input.TextArea rows={4} placeholder="可填写灰度范围、验证目标或领导评审备注" />
          </Form.Item>
        </Form>

        {/* 批量输入 SN 弹窗 — 给当前任务的设备表加批量勾选入口 */}
        <Modal
          title="批量输入 SN"
          open={batchSNInputOpen}
          onCancel={() => setBatchSNInputOpen(false)}
          onOk={() => { void handleApplyBatchSNs(); }}
          okText="确定"
          cancelText="取消"
          confirmLoading={batchSNApplying}
          width={520}
          destroyOnClose
        >
          <Form layout="vertical">
            <Form.Item label="Serial Number">
              <Input.TextArea
                rows={6}
                placeholder="粘贴或输入多个 SN，按分号 ; 逗号 , 空格、Tab 或换行分隔"
                value={batchSNInputText}
                onChange={(e) => setBatchSNInputText(e.target.value)}
                allowClear
              />
            </Form.Item>
            <Text type="secondary" style={{ fontSize: 12 }}>
              多个设备 SN 可用分号 (;)、逗号 (,)、空格 或换行分隔；自动去重、忽略空白。
              未在当前任务候选列表中的 SN 会在提交后给出提示。
            </Text>
          </Form>
        </Modal>
      </Drawer>

      <Drawer
        title={detailTask ? `任务详情 · ${detailTask.taskName}` : '任务详情'}
        width={640}
        open={detailDrawerOpen}
        onClose={() => { setDetailDrawerOpen(false); setDetailTask(null); }}
        destroyOnClose
      >
        {detailTask ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label="任务名称">{detailTask.taskName}</Descriptions.Item>
              <Descriptions.Item label="任务类型">{detailTask.typeDisplayName}</Descriptions.Item>
              <Descriptions.Item label="状态">{renderTaskStatus(detailTask.status)}</Descriptions.Item>
              <Descriptions.Item label="结果">
                {detailTask.result
                  ? <Tag color={detailTask.result === 'success' ? 'success' : detailTask.result === 'partial' ? 'warning' : 'error'}>{detailTask.result === 'success' ? '成功' : detailTask.result === 'partial' ? '部分成功' : detailTask.result === 'terminated' ? '已终止' : '失败'}</Tag>
                  : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="目标版本">{getTaskTargetVersion(detailTask)}</Descriptions.Item>
              <Descriptions.Item label="产品类型">{getTaskProductClass(detailTask)}</Descriptions.Item>
              <Descriptions.Item label="执行方式">
                <Tag>{EXECUTION_MODE_OPTIONS.find((o) => o.value === detailTask.executionMode)?.label ?? detailTask.executionMode}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="当前步骤">{STEP_LABELS[detailTask.currentStep as TransferStepId] ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="操作人">{detailTask.createUser}</Descriptions.Item>
              <Descriptions.Item label="创建时间">{new Date(detailTask.createdAt).toLocaleString('zh-CN')}</Descriptions.Item>
            </Descriptions>
            <Card title="执行进度" size="small">
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Progress
                  type="circle"
                  percent={detailTask.progress}
                  format={() => `${detailTask.progress}%`}
                />
                <Space wrap>
                  <Tag color="success">成功 {detailTask.successCount}</Tag>
                  <Tag color="error">失败 {detailTask.failCount}</Tag>
                  <Tag>总数 {detailTask.totalCount}</Tag>
                </Space>
              </Space>
            </Card>
          </Space>
        ) : null}
      </Drawer>
    </ListPageLayout>
  );
}