import { useEffect, useMemo, useRef, useState } from 'react';
import dayjs from 'dayjs';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  DatePicker,
  Descriptions,
  Drawer,
  Form,
  Input,
  InputNumber,
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
import { DeleteOutlined, DownloadOutlined, PlusOutlined, ExportOutlined, ReloadOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useLocation, useSearchParams } from 'react-router-dom';
import { useT } from '@/hooks/useT';
import { useTabStore } from '@core/store/tabStore';
import { useMenuStore } from '@core/store/menuStore';
import { resolveMenuLabel } from '@core/types/menu';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import SearchInput from '@/components/SearchInput';
import MRTasksPanel from '@/pages/mr/Tasks';
import KpiExportTasksPanel from '@/pages/transfer/KpiExport/KpiExportTasksPanel';
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
import { useSystemTimezoneValue } from '@core/hooks/api/useSystemTimezone';
import { useSystemLicense } from '@core/hooks/api/useSystemLicense';
import { useSoftwareVersions } from '@core/hooks/api/useSoftware';
import { useProductList } from '@core/hooks/api/useProducts';
import {
  filterDeviceScopedItemsByLicense,
  isDeviceStandardValueVisibleByLicense,
  isDeviceStandardVisibleByLicense,
} from '@core/utils/licenseFeatures';
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
import { firmwareLibraryFileTypeToFileManagerTab } from '@core/utils/firmwareFileType';
import {
  mergeCandidatesIntoMap,
  removeFromMap,
  pruneSelectionForProductClass,
} from './deviceSelection';
import {
  buildCategoryTabs,
  getExecutionModeOptions,
  buildDefaultUfteTaskName,
  getSoftwareLibraryFileTypeLabel,
  getStepLabels,
  localizeBuiltinCategoryLabel,
  localizeBuiltinTypeName,
  UPGRADE_LIKE_CATEGORIES,
  filterTaskTypesForCategory,
  filterTaskTypesByUPSLicense,
  filterFileTransferItemsByUPSLicense,
  resolveBackendCategoryParam,
} from '../shared';
import {
  renderDeviceStatus,
  renderEllipsisCell,
  renderTaskStatus,
} from '../shared.render';
import { resolveAutoSelectedCategory } from './categorySelection';
import { formatFailureReasonDisplay } from './failureReason';
import { resolveTargetFileDisplay } from './targetFileDisplay';
import {
  resolveTaskTypeFilterValue,
  shouldShowTaskTypeFilter,
} from './taskTypeFilterSelection';
import type { TransferStepId } from '@core/types/unifiedFileTransfer';
import { formatSystemTime, nowInSystemTimezone } from '@core/utils/systemTime';
import {
  isTransferSystemDateBefore,
  isTransferSystemTimeAfter,
  toTransferSystemTimeRFC3339,
} from '../transferTime';

const { Text, Title } = Typography;

// qa-614 c6 #367：任务详情 Descriptions 局部样式——标签/内容单行不换行（超长走
// ellipsis），并把详情区字号收紧至 12（不动全局 token，避免全 v1 页面回归）。
const DETAIL_LABEL_STYLE: React.CSSProperties = { whiteSpace: 'nowrap', fontSize: 12 };
const DETAIL_CONTENT_STYLE: React.CSSProperties = {
  whiteSpace: 'nowrap',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  fontSize: 12,
};

// buildDefaultTaskName 适配器：保持原签名（typeCode + username），内部委托给
// shared.buildDefaultUfteTaskName + appStore.locale，全局保持 i18n 一致。
function buildDefaultTaskName(typeCode: string | undefined, username: string | undefined, locale: string): string {
  return buildDefaultUfteTaskName(typeCode, username, locale, dayjs().format('YYYY-MM-DD HH:mm:ss'));
}

function isUpgradeTaskCategory(category?: string) {
  // qa-614 c6 #365 #373 / UPS：2G(gsm_upgrade) / UPS(ups_upgrade) 也是升级类，需固件选择；
  // #368：device_upgrade 是 4G/5G/2G/UPS 合并虚拟分类，同属升级类。
  return (
    category === 'gnb_upgrade' ||
    category === 'enb_upgrade' ||
    category === 'gsm_upgrade' ||
    category === 'ups_upgrade' ||
    category === 'device_upgrade'
  );
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
  if (haystack.includes('ups') || haystack.includes('ap')) {
    return 5;
  }
  if (taskType && isUpgradeTaskCategory(taskType.category)) {
    return 0;
  }
  return undefined;
}

function getUpgradeTypeLabel(category: string, fallback: string, t: (id: string) => string) {
  // qa-614 c6 #365 #373 #368 / UPS：2G(gsm_upgrade)、UPS(ups_upgrade) 与合并虚拟分类 device_upgrade 同归升级标签。
  if (
    category === 'gnb_upgrade' ||
    category === 'enb_upgrade' ||
    category === 'gsm_upgrade' ||
    category === 'ups_upgrade' ||
    category === 'device_upgrade'
  ) {
    return t('ufte.softLib.upgrade');
  }
  if (category === 'version_rollback') {
    return t('ufte.softLib.rollback');
  }
  return fallback;
}

/**
 * 新建任务抽屉的表单值。
 * 在 CreateUnifiedFileTransferTaskInput（提交载荷）之外，多一个 UI-only 的 `productClass`
 * 字段：用于按机型筛选固件候选（watch 后喂给候选查询的 productType），不直接提交。
 */
type TaskFormValues = CreateUnifiedFileTransferTaskInput & {
  productClass?: string;
};

export default function FileTransferCenter() {
  const t = useT();
  const systemTimezone = useSystemTimezoneValue();
  const queryClient = useQueryClient();
  // 任务名称自动填充用：取登录用户名拼前缀，displayName / username 哪个有用哪个。
  const currentUser = useUserStore((s) => s.currentUser);
  const taskNameUser = currentUser?.username || currentUser?.displayName || 'user';
  const appLocale = useAppStore((s) => s.locale);

  // #375: 自注册「任务管理」页签。v1 多页签机制下激活页签标题取自 tabStore 的
  // 激活 tab.label；从其它页面通过 navigate('/transfer/center?...') 直跳进本页时
  // 没有任何 openTab，AppShell 的 syncActiveTabPath 又被同 pathname 守卫拦截
  // （旧激活 tab 仍是来源页面），导致激活页签标题未更新、内容却已是本页。仿
  // DeviceDetail 在自身 effect 里 openTab 自注册，覆盖任何入口（页面内跳转 /
  // 北向直链 / 侧栏点击均命中同一页签）。
  const location = useLocation();
  const openTab = useTabStore((s) => s.openTab);
  const flatMenus = useMenuStore((s) => s.flatMenus);
  useEffect(() => {
    // 按 routePath='/transfer/center' 反查菜单 key + nameI18n，兼容静态/动态两种菜单：
    //  - 动态菜单(DB)：key===routePath、labelRaw=true（用 nameI18n 切语言刷新）；
    //  - 查不到则回退静态：key='transfer-task-create'、label='nav.transfer.taskCreate'、labelRaw=false。
    // openTab 的 dedup(key 或 path 命中)确保与侧栏点击命中同一页签，不产生错配双页签。
    const menu = flatMenus.find((m) => m.routePath === '/transfer/center');
    const path = `/transfer/center${location.search}`;
    if (menu) {
      openTab({
        key: menu.routePath as string,
        label: resolveMenuLabel(menu, appLocale),
        labelRaw: true,
        path,
        closable: true,
      });
    } else {
      openTab({
        key: 'transfer-task-create',
        label: 'nav.transfer.taskCreate',
        labelRaw: false,
        path,
        closable: true,
      });
    }
  }, [flatMenus, location.search, appLocale, openTab]);
  // lastAutoFilledTaskNameRef 记录最近一次自动填的名字。用户在表单里手动改过 → ref
  // 跟 form 值不再一致 → typeCode 切换时不覆盖；用户没改 → 切换业务时跟着刷新。
  const lastAutoFilledTaskNameRef = useRef<string>('');
  const { data: taskTypes = [], isLoading: taskTypesLoading } = useUnifiedFileTransferTaskTypes();
  const { data: systemLicense, isLoading: systemLicenseLoading } = useSystemLicense();
  const showUPSOptions = isDeviceStandardVisibleByLicense(systemLicense, systemLicenseLoading, 'UPS');
  const licensedTaskTypes = useMemo(
    () => filterTaskTypesByUPSLicense(taskTypes, showUPSOptions),
    [taskTypes, showUPSOptions],
  );
  // 原始 categories 来自后端 ufte_task_types，新增一个虚拟分类 "mr_measurement"
  // 作为入口聚合按钮（点击跳到独立的 MR 任务管理页 /mr/tasks）。MR 不走 UFTE
  // 任务模板（PRD F05 决策 A — 独立引擎），这里只做"入口聚合"。
  const categories = useMemo(() => {
    const base = buildCategoryTabs(licensedTaskTypes);
    // 末位插入虚拟项；templateCount=0 让现有 UFTE 模板渲染逻辑识别"无模板"
    return [
      ...base,
      {
        category: 'mr_measurement',
        categoryLabel: t('ufte.builtin.category.mr_measurement'), // 通过 localizeBuiltinCategoryLabel 翻译
        templateCount: 0,
      },
      {
        // KPI-EXPORT：KPI 导出虚拟分类，内联渲染 KpiExportTasksPanel 看导出任务状态。
        category: 'kpi_export',
        categoryLabel: t('ufte.builtin.category.kpi_export'), // 通过 localizeBuiltinCategoryLabel 翻译
        templateCount: 0,
      },
    ];
  }, [licensedTaskTypes, t]);
  // URL 参数初始化：?category=...&typeCode=... 用于外部 deep link，直接定位到
  // 指定分类及任务类型。
  const [urlSearchParams] = useSearchParams();
  const [selectedCategory, setSelectedCategory] = useState(() => urlSearchParams.get('category') ?? '');
  // #127：标记 selectedCategory 是否来自用户主动选择（点击 Tab / URL deep link）。
  // 首屏 task-types 未返回时 categories 只含虚拟分类，自动选中会落在 'mr_measurement'；
  // 真实分类到达后仅当用户没主动选过时才回退，不覆盖用户选择。
  const categoryManuallyPickedRef = useRef(Boolean(urlSearchParams.get('category')));
  const initialTypeCodeParamRef = useRef<string | undefined>(urlSearchParams.get('typeCode') || undefined);
  const [selectedTypeCode, setSelectedTypeCode] = useState<string | undefined>(() => urlSearchParams.get('typeCode') || undefined);
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
  const [deviceProductNameFilter, setDeviceProductNameFilter] = useState<string>();
  const [viewMode, setViewMode] = useState<'tasks' | 'devices'>('tasks');
  const [taskDrawerOpen, setTaskDrawerOpen] = useState(false);
  // MR 测量 Tab 的新建抽屉 — 受控状态，让顶部 "New Task" 按钮接管打开（与 UFTE 风格一致）
  const [mrCreateOpen, setMrCreateOpen] = useState(false);
  const [taskForm] = Form.useForm<TaskFormValues>();
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
  // #215: 选设备候选列表真分页 —— 当前页码 / 每页条数。
  const [drawerDevicePage, setDrawerDevicePage] = useState(1);
  const [drawerDevicePageSize, setDrawerDevicePageSize] = useState(20);
  // #215: 已选设备清单（跨页累积）。候选列表分页后当前页只含本页设备，
  // 而已选设备可能分布在多页，故用 id→设备 的 Map 累积，保证"已选 N 台"
  // 文本、查看清单 Modal、CONFIG/LICENSE 探测表都能看到所有已选设备，不被分页截断。
  const [selectedDeviceMap, setSelectedDeviceMap] = useState<Record<string, UnifiedFileTransferDeviceItem>>({});
  // #215: 已选清单查看 Modal 开关。
  const [selectedDevicesModalOpen, setSelectedDevicesModalOpen] = useState(false);
  const [detailTask, setDetailTask] = useState<UnifiedFileTransferTask | null>(null);
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false);

  // qa-614 c6 #368 / UPS：device_upgrade 虚拟分类展开为后端 category 查询参数。
  // 选了具体 typeCode → 传 undefined（按 typeCode 精确过滤）；未选 → 传 device_upgrade，
  // 后端展开到 4G/5G/2G/UPS 升级成员，避免 UPS 被 enb_upgrade 二次过滤掉。
  const backendCategoryParam = resolveBackendCategoryParam(selectedCategory, selectedTypeCode);

  const { data: tasksData, isLoading: tasksLoading } = useUnifiedFileTransferTasks({
    page: taskPage,
    pageSize: taskPageSize,
    category: backendCategoryParam,
    keyword: taskKeyword || undefined,
    status: taskStatusFilter,
    typeCode: selectedTypeCode || undefined,
  });

  const { data: devicesData, isLoading: devicesLoading } = useUnifiedFileTransferDevices({
    page: devicePage,
    pageSize: devicePageSize,
    category: backendCategoryParam,
    keyword: deviceKeyword || undefined,
    status: deviceStatusFilter,
    typeCode: selectedTypeCode || undefined,
    productName: deviceProductNameFilter,
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
  const recentTasksRaw = tasksData?.items ?? [];
  const recentDevicesRaw = devicesData?.items ?? [];
  const recentTasks = useMemo(
    () => filterFileTransferItemsByUPSLicense(recentTasksRaw, showUPSOptions),
    [recentTasksRaw, showUPSOptions],
  );
  const recentDevices = useMemo(
    () => filterFileTransferItemsByUPSLicense(recentDevicesRaw, showUPSOptions),
    [recentDevicesRaw, showUPSOptions],
  );

  const filteredTaskTypes = useMemo(
    // 顺序由后端 ORDER BY sort_order ASC 控制（数据库字段 ufte_task_types.sort_order
    // 由内置模板初始化，未来可在「模板配置」页面拖拽调整）。前端不再二次排序，避免
    // 跟数据库源头不一致。
    // qa-614 c6 #368 / UPS：device_upgrade 虚拟分类下取 4G/5G/2G/UPS 全部升级模板。
    () => filterTaskTypesForCategory(licensedTaskTypes, selectedCategory),
    [licensedTaskTypes, selectedCategory],
  );

  const taskTypeOptions = useMemo(
    () => filteredTaskTypes.map((item) => ({
      label: localizeBuiltinTypeName(item.typeCode, item.displayName, t),
      value: item.typeCode,
    })),
    [filteredTaskTypes, t],
  );
  const showTaskTypeFilter = useMemo(
    () => shouldShowTaskTypeFilter(filteredTaskTypes),
    [filteredTaskTypes],
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
    () => licensedTaskTypes.find((item) => item.typeCode === drawerTypeCode) ?? activeTaskType,
    [activeTaskType, drawerTypeCode, licensedTaskTypes],
  );
  // #492：升级抽屉「产品类型」改为产品名。products 来自产品中心目录，用于把所选产品名映射成
  // product_id（固件按产品过滤）。
  const { data: productsData } = useProductList();
  const products = useMemo(() => productsData?.items ?? [], [productsData]);
  const visibleProducts = useMemo(
    () => filterDeviceScopedItemsByLicense(products, systemLicense, systemLicenseLoading),
    [products, systemLicense, systemLicenseLoading],
  );
  const drawerProductId = useMemo(
    () => products.find((p) => p.name === drawerProductClass)?.id,
    [products, drawerProductClass],
  );
  const firmwareLibraryFileType = resolveFirmwareLibraryFileType(drawerTaskType);
  const isUPSUpgradeDrawer = drawerTaskType?.typeCode === 'UPS_AP_UPGRADE' || drawerTaskType?.category === 'ups_upgrade';
  const firmwareManagerUrl = useMemo(() => {
    const params = new URLSearchParams({
      tab: 'version',
      return: 'ufte',
      fileType: firmwareLibraryFileTypeToFileManagerTab(firmwareLibraryFileType),
    });
    return `/transfer/file-management?${params.toString()}`;
  }, [firmwareLibraryFileType]);
  const { data: firmwareData } = useSoftwareVersions({
    page: 1,
    pageSize: 200,
    fileType: firmwareLibraryFileType,
    // #492：固件按所选产品过滤（产品名→product_id）。未选产品则不限。
    productId: drawerProductId,
  });

  // 创建任务时全部三种执行方式可选（立即 / 挂起 / 定时）。
  // 定时模式下表单会条件渲染 DatePicker，handleCreateTask 把时间塞进 payload.scheduledAt。
  const createExecutionModeOptions = executionModeOptions;

  const { data: drawerDevicesData, isLoading: drawerDevicesLoading } = useUnifiedFileTransferDeviceCandidates({
    page: drawerDevicePage,
    pageSize: drawerDevicePageSize,
    // qa-614 c6 #368：抽屉里 typeCode 必选（升级类），故传 undefined 让后端按
    // typeCode 精确匹配设备候选（productTechLookup 区分 4G/5G/2G）。
    category: resolveBackendCategoryParam(selectedCategory, drawerTaskType?.typeCode || selectedTypeCode || ''),
    typeCode: drawerTaskType?.typeCode || selectedTypeCode || undefined,
    // #492：升级类任务用所选「产品名」缩窄候选设备（后端按 productName 过滤：设备 productClass→产品名∈选中）。
    productName: needsFirmwareSelection(drawerTaskType) ? drawerProductClass || undefined : undefined,
    keyword: drawerDeviceKeyword || undefined,
  });

  const drawerDeviceCandidates = drawerDevicesData?.items ?? [];
  const drawerDeviceTotal = drawerDevicesData?.total ?? 0;

  // #215: 当前页拉到的候选设备并入 selectedDeviceMap，只补充设备对象信息，
  // 不改 selectedDrawerDeviceIds —— 翻页只是让"已选清单"能补齐其它页设备的展示数据。
  useEffect(() => {
    setSelectedDeviceMap((prev) => mergeCandidatesIntoMap(prev, drawerDeviceCandidates));
  }, [drawerDeviceCandidates]);

  // #215: 过滤条件（分类/任务类型/产品类型/关键字）变化时，候选集变了，重置回第 1 页。
  useEffect(() => {
    setDrawerDevicePage(1);
  }, [selectedCategory, drawerTaskType?.typeCode, selectedTypeCode, drawerProductClass, drawerDeviceKeyword]);

  // #215 回归守护：固件升级类任务是「按机型」的，换 productClass 筛选后，旧的不同
  // 机型已选设备必须剔除，否则升级会下发到不兼容机型。以 selectedDeviceMap 判定
  // （非当前页候选），翻页不会误删其它页合法已选。仅在 class / 任务类型变化时触发。
  useEffect(() => {
    if (!needsFirmwareSelection(drawerTaskType) || !drawerProductClass) {
      return;
    }
    const { kept, droppedCount } = pruneSelectionForProductClass(
      selectedDrawerDeviceIds,
      selectedDeviceMap,
      drawerProductClass,
    );
    if (droppedCount > 0) {
      setSelectedDrawerDeviceIds(kept);
      void message.warning(t('ufte.msg.selectionPrunedByClass', { count: droppedCount }));
    }
    // 只在 productClass / 任务类型变化时校正；selectedDrawerDeviceIds/Map 经闭包读取，
    // 换 class 当下旧设备对象仍在 map 中，判定有效。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [drawerProductClass, drawerTaskType]);

  const firmwareCandidates = useMemo(
    () => buildFirmwareCandidateList(firmwareData?.items ?? []),
    [firmwareData?.items],
  );

  // #492：升级抽屉「产品类型」下拉 = 模板适用产品名（products）。模板未配产品（=全部）时退为
  // 产品目录全集，便于按产品收窄。value/label 均为产品英文名。
  const drawerProductClassOptions = useMemo(() => {
    const tplProducts = drawerTaskType?.products ?? [];
    const names = tplProducts.length > 0 ? tplProducts : visibleProducts.map((p) => p.name);
    return Array.from(new Set(names))
      .filter((name) => isDeviceStandardValueVisibleByLicense(name, systemLicense, systemLicenseLoading))
      .filter(Boolean)
      .sort((left, right) => left.localeCompare(right, 'zh-CN'))
      .map((name) => ({ label: name, value: name }));
  }, [drawerTaskType?.products, systemLicense, systemLicenseLoading, visibleProducts]);

  const filteredFirmwareCandidates = useMemo(
    // #492：固件已按所选产品(product_id) 服务端过滤；这里仅在选了产品后展示其固件。
    () => (drawerProductClass ? firmwareCandidates : []),
    [drawerProductClass, firmwareCandidates],
  );

  const firmwareOptions = useMemo(
    () => filteredFirmwareCandidates.map((item) => ({
      label: item.fileName || item.versionCode,
      value: item.id,
    })),
    [filteredFirmwareCandidates],
  );

  // #524：设备列表「产品名称」筛选 —— 选项与传参均用产品名（与「产品名称」列展示口径一致）。
  // 与升级抽屉 drawerProductClassOptions 同口径：所选任务类型配了适用产品(products，产品英文名)
  // 就按其收窄，未配(=全部)则退回产品目录全集；设备已带回的 productName 再兜底并入（任务类型
  // 范围之外仍可见的产品名）。原先直接把 productClass 英文代码当选项、按 productType 过滤——文案
  // 改「按产品名称过滤」后名实不符，这里对齐为真·按产品名。
  const deviceProductNameOptions = useMemo(() => {
    const scoped = activeTaskType?.products ?? [];
    const names = new Set<string>(
      scoped.length > 0 ? scoped : visibleProducts.map((product) => product.name),
    );
    recentDevices.forEach((item) => {
      if (item.productName) {
        names.add(item.productName);
      }
    });
    return Array.from(names)
      .filter(Boolean)
      .filter((name) => isDeviceStandardValueVisibleByLicense(name, systemLicense, systemLicenseLoading))
      .sort((left, right) => left.localeCompare(right, 'zh-CN'))
      .map((name) => ({ label: name, value: name }));
  }, [activeTaskType?.products, recentDevices, systemLicense, systemLicenseLoading, visibleProducts]);

  // qa-614 c6 #368：原 templateTabItems（『模板』子页签数据）已删除——与执行视图
  // 任务列表上方的 typeCode 下拉框完全重复（ant-space-item 多余）。typeCode 选择
  // 统一收口到 Select，单一入口。

  // #215: 已选设备从跨页累积的 selectedDeviceMap 取，保证翻页后其它页的已选设备
  // 仍能在"已选 N 台"清单 / CONFIG/LICENSE 探测表里展示，不被当前页截断。
  const drawerSelectedDevices = useMemo(
    () => selectedDrawerDeviceIds
      .map((id) => selectedDeviceMap[id])
      .filter((item): item is UnifiedFileTransferDeviceItem => Boolean(item)),
    [selectedDeviceMap, selectedDrawerDeviceIds],
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
      // #492：候选列展示产品英文名（productName，由后端 productClass→ProductRegistry 解析）；
      // 解析不到（孤儿设备）回退裸 productClass。
      {
        title: t('ufte.col.productType'),
        key: 'productType',
        width: 140,
        render: (_: unknown, record: UnifiedFileTransferDeviceItem) => record.productName || record.productType || '-',
      },
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
      const { display } = formatFailureReasonDisplay(value, t);
      // 设备厂商原始 fault（FaultCode + FaultString）放 Tooltip 里——i18n label 只看到统一
      // 错误码描述，hover 后能拿到设备端原文（如 "FaultCode: 0, FaultString: httpUpload OM
      // Http Put Upload stat file error"），方便厂商侧排查。
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
        productName: needsFirmwareSelection(drawerTaskType) ? drawerProductClass || undefined : undefined,
      });
      const allCandidates = resp.items ?? [];
      const snToDevice = new Map(allCandidates.map((d) => [d.deviceSn, d]));
      const matchedIds: string[] = [];
      const matchedDevices: UnifiedFileTransferDeviceItem[] = [];
      const unmatched: string[] = [];
      for (const sn of tokens) {
        const device = snToDevice.get(sn);
        if (device) {
          matchedIds.push(device.id);
          matchedDevices.push(device);
        } else {
          unmatched.push(sn);
        }
      }
      // 与已选合并去重；同时把命中设备对象并入 selectedDeviceMap，供已选清单展示。
      setSelectedDrawerDeviceIds((prev) => Array.from(new Set([...prev, ...matchedIds])));
      setSelectedDeviceMap((prev) => {
        const next = { ...prev };
        for (const device of matchedDevices) {
          next[device.id] = device;
        }
        return next;
      });
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

  // #215: 从已选清单移除一台设备 —— 同时从 id 列表与设备 Map 清掉。
  const handleRemoveSelectedDevice = (deviceId: string) => {
    setSelectedDrawerDeviceIds((prev) => prev.filter((id) => id !== deviceId));
    setSelectedDeviceMap((prev) => removeFromMap(prev, deviceId));
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
        productName: deviceProductNameFilter,
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
    if (taskTypesLoading && categoryManuallyPickedRef.current) return;
    const next = resolveAutoSelectedCategory(categories, selectedCategory, categoryManuallyPickedRef.current);
    if (next !== null) {
      // 此 effect 产生的都是自动兜底选择（含 #127 虚拟分类→首个真实分类回退），
      // 清掉手选标记，保证后续真实分类到达时仍可继续回退。
      categoryManuallyPickedRef.current = false;
      setSelectedCategory(next);
    }
  }, [categories, selectedCategory, taskTypesLoading]);

  useEffect(() => {
    const initialTypeCode = initialTypeCodeParamRef.current;
    if (!initialTypeCode) return;
    if (filteredTaskTypes.some((item) => item.typeCode === initialTypeCode)) {
      initialTypeCodeParamRef.current = undefined;
      if (selectedTypeCode !== initialTypeCode) {
        setSelectedTypeCode(initialTypeCode);
      }
      return;
    }
    if (!taskTypesLoading && taskTypes.length > 0) {
      initialTypeCodeParamRef.current = undefined;
    }
  }, [filteredTaskTypes, selectedTypeCode, taskTypes.length, taskTypesLoading]);

  useEffect(() => {
    if (taskTypesLoading) return;
    const nextTypeCode = resolveTaskTypeFilterValue(filteredTaskTypes, selectedTypeCode);
    if (nextTypeCode !== selectedTypeCode) {
      setSelectedTypeCode(nextTypeCode);
    }
  }, [filteredTaskTypes, selectedTypeCode, taskTypesLoading]);

  useEffect(() => {
    setTaskPage(1);
    setDevicePage(1);
    // 同步清空批量选中——切 tab / 改过滤 / 翻页时旧的选中 ID 已不在当前可见行
    setSelectedTaskIds([]);
  }, [selectedCategory, selectedTypeCode, taskKeyword, taskStatusFilter, deviceKeyword, deviceStatusFilter, deviceProductNameFilter]);

  // 翻页（taskPage / taskPageSize 变化）同样清空选中，避免跨页 ID 残留计数
  useEffect(() => {
    setSelectedTaskIds([]);
  }, [taskPage, taskPageSize]);

  useEffect(() => {
    setDeviceProductNameFilter(undefined);
  }, [selectedTypeCode]);

  const isUpgradeLikeCategory = UPGRADE_LIKE_CATEGORIES.has(selectedCategory);

  const getTypeDef = (typeCode: string) => licensedTaskTypes.find((item) => item.typeCode === typeCode);

  const getTaskTargetVersion = (record: UnifiedFileTransferTask) => {
    // 只有升级 / 回滚类才有"主任务目标版本"（固件版本号）；备份 / 日志采集 /
    // 配置恢复类后端返回空串，前端不再用 typeDef.fileNameTemplate 等模板字符串兜底
    // ——那只是占位符模板，对主任务来说没有意义。详见
    // docs/project/backup-display-fix-20260520.md F2。
    return record.targetVersion?.trim() ? record.targetVersion : '-';
  };

  const getTaskProductClass = (record: UnifiedFileTransferTask) => {
    if (record.productName) {
      return record.productName;
    }
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
          render: (value: string) => {
            // 上报时间只在文件上报成功的终态才有值；未上报时为空，直接显示 '-'，
            // 不要把空值丢给 new Date()（会渲染成 "Invalid Date"）。
            if (!value) return '-';
            return formatSystemTime(value);
          },
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
          width: 180,
          // #628：升级类原本只画 Progress 条，看不出"一共多少台/完成多少/失败多少"。
          // 改为与非升级类同款两行布局，复用 ufte.col.statsBrief。
          render: (value: number, record) => (
            <Space direction="vertical" size={4} style={{ width: '100%' }}>
              <Progress percent={value} size="small" status={record.status === 'ended' ? 'success' : record.status === 'in_progress' ? 'active' : 'normal'} />
              <Text type="secondary">{t('ufte.col.statsBrief', { ok: record.successCount, fail: record.failCount, total: record.totalCount })}</Text>
            </Space>
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
          render: (_, record) => {
            if (record.startedAt) return formatSystemTime(record.startedAt);
            if (record.executionMode === 'scheduled' && record.scheduledAt) return formatSystemTime(record.scheduledAt);
            return '-';
          },
        },
        {
          title: t('ufte.col.endTime'),
          key: 'endTime',
          width: 180,
          render: (_, record) => (record.endedAt ? formatSystemTime(record.endedAt) : '-'),
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
        render: (value: string) => formatSystemTime(value),
      },
      {
        title: t('ufte.col.endTime'),
        key: 'endTime',
        width: 180,
        render: (_, record) => (record.endedAt ? formatSystemTime(record.endedAt) : '-'),
      },
    ];
  }, [isUpgradeLikeCategory, licensedTaskTypes, t]);

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
          key: 'productType',
          width: 130,
          // #492：展示产品英文名，解析不到回退裸 productClass。
          render: (_: unknown, record: UnifiedFileTransferDeviceItem) => renderEllipsisCell(record.productName || record.productType),
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
          // issue #655：原「操作时间」实际是 lastReportAt（文件上报成功终态才有值），命名歧义、
          // 中间态无任何时间信息。改为「开始时间 / 结束时间」两列直读 sub_task.started_at /
          // completed_at，执行中也能看到首次进入执行态的时刻。
          title: t('ufte.col.startTime'),
          dataIndex: 'startedAt',
          key: 'startedAt',
          width: 180,
          render: (value?: string) => (value ? formatSystemTime(value) : '-'),
        },
        {
          title: t('ufte.col.endTime'),
          dataIndex: 'endedAt',
          key: 'endedAt',
          width: 180,
          render: (value?: string) => (value ? formatSystemTime(value) : '-'),
        },
      ];
    }

    return [
      {
        // 任务列表场景：每行是「某任务在某设备上的执行情况」。主显示任务名（用户操作
        // 上下文，他刚创建的 testNV 想看这个任务的进展），副标题挂设备名（区分多设备）。
        // 列宽 280：UFTE 自动生成的任务名形如 "ConfigBackupNV_admin_2026-05-21 13:39:55"
        // 约 36 字符，hover Tooltip 看全名。
        // 表头统一为「任务名称」：本列主体就是任务（设备名仅作副标题、另有独立「设备 SN」列），
        // 原「任务/设备」名实不符；并与后端 CSV 导出 default 视图首列表头「任务名称」对齐。
        title: t('ufte.col.taskName'),
        dataIndex: 'taskName',
        key: 'taskName',
        width: 280,
        render: (_, record) => renderEllipsisCell(record.taskName, {
          strong: true,
          subtitle: record.deviceName,
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
        key: 'productType',
        width: 130,
        // #492：展示产品英文名，解析不到回退裸 productClass。
        render: (_: unknown, record: UnifiedFileTransferDeviceItem) => renderEllipsisCell(record.productName || record.productType),
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
          const fileState = resolveTargetFileDisplay(record, t('ufte.file.cleanedByQuota'));
          if (!fileState.file) {
            return '-';
          }
          const inner = fileState.deleted ? (
            <Text disabled ellipsis style={{ maxWidth: 220, cursor: 'not-allowed' }}>
              {fileState.file}
            </Text>
          ) : fileState.downloadUrl ? (
            <a href={fileState.downloadUrl} target="_blank" rel="noopener noreferrer" style={{ display: 'inline-block', maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', verticalAlign: 'bottom' }}>
              {fileState.file}
            </a>
          ) : (
            <Text type="secondary" ellipsis style={{ maxWidth: 220 }}>{fileState.file}</Text>
          );
          return (
            <Tooltip title={fileState.tooltip} placement="topLeft">
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
        // issue #655：与升级类同款，「上报时间」单列改为「开始时间 / 结束时间」两列。
        title: t('ufte.col.startTime'),
        dataIndex: 'startedAt',
        key: 'startedAt',
        width: 180,
        render: (value?: string) => (value ? formatSystemTime(value) : '-'),
      },
      {
        title: t('ufte.col.endTime'),
        dataIndex: 'endedAt',
        key: 'endedAt',
        width: 180,
        render: (value?: string) => (value ? formatSystemTime(value) : '-'),
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
      concurrency: 20,
      executionMode: 'immediate',
    });
    setSelectedDrawerDeviceIds([]);
    setSelectedDeviceMap({});
    setDrawerDevicePage(1);
    setDrawerDeviceKeyword('');
    setDrawerDeviceKeywordInput('');
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
      taskForm.setFieldValue('concurrency', undefined);
    } else {
      if (taskForm.getFieldValue('isKeepConfig') === undefined) {
        taskForm.setFieldValue('isKeepConfig', true);
      }
      if (taskForm.getFieldValue('concurrency') === undefined) {
        taskForm.setFieldValue('concurrency', 20);
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
    // #215: 候选列表启用真分页后，当前页只含本页设备，已选设备可能在其它页，
    // 不能再按"当前页候选"裁剪 selectedDrawerDeviceIds（否则翻页即丢已选）。
    // 选中跨页保留，用户可在"已选 N 台"清单里逐台移除。
  }, [drawerProductClass, drawerProductClassOptions, drawerTaskType, firmwareOptions, taskDrawerOpen, taskForm]);

  // 同步 in-flight 锁：防止快速连点「创建」时发出两条创建请求、落两条相同任务（issue #522）。
  // isPending 与按钮 loading 都是「渲染态」，在第一次 mutateAsync 触发的 re-render 落地前一直为 false，
  // 加上处理函数中间 await validateFields 的异步间隙，一个渲染周期内的连点都能越过 isPending 守卫。
  // 用 ref 在任何 await 之前同步置位、提交结束（成败）后释放，确保同一时刻只放行一次提交。
  const creatingTaskRef = useRef(false);

  const handleCreateTask = async () => {
    if (creatingTaskRef.current || createTaskMutation.isPending) {
      return;
    }
    creatingTaskRef.current = true;
    let values: TaskFormValues;
    try {
      values = await taskForm.validateFields();
    } catch {
      // 校验未通过：antd 已高亮对应字段，释放同步锁让用户改后重新提交。
      creatingTaskRef.current = false;
      return;
    }
    if (selectedDrawerDeviceIds.length === 0) {
      void message.warning(t('ufte.msg.pickDevice'));
      creatingTaskRef.current = false;
      return;
    }
    // T-0164: CONFIG_RESTORE 整批拒绝 — 任一设备缺快照即阻止提交。
    if (isConfigRestoreBlockedByMissing) {
      void message.error(t('ufte.msg.snapshotMissing', {
        sns: snapshotProbe?.missing.join(', ') ?? '',
      }));
      creatingTaskRef.current = false;
      return;
    }
    // T-0165: LICENSE_UPGRADE 同款整批拒绝 — 任一设备缺 license 即阻止提交。
    if (isLicenseUpgradeBlockedByMissing) {
      void message.error(t('ufte.msg.licenseMissing', {
        sns: licenseProbe?.missing.join(', ') ?? '',
      }));
      creatingTaskRef.current = false;
      return;
    }
    // scheduledAt 在表单里是 dayjs 实例，发请求前按系统时区附加偏移（后端 RFC3339 解析）。
    // 非 scheduled 模式 form 不会渲染这个字段 → values.scheduledAt 为 undefined，直接传不影响。
    const rawScheduledAt = (values as { scheduledAt?: unknown }).scheduledAt;
    const scheduledAtIso = values.executionMode === 'scheduled' && rawScheduledAt
      ? toTransferSystemTimeRFC3339(rawScheduledAt, systemTimezone)
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
        setSelectedDeviceMap({});
        setSelectedDevicesModalOpen(false);
        taskForm.resetFields();
      })
      .catch((error: unknown) => {
        void message.error(getTaskActionErrorMessage(error, t('ufte.msg.taskCreateFailed')));
      })
      .finally(() => {
        creatingTaskRef.current = false;
      });
  };

  return (
    <ListPageLayout
      title={t('ufte.page.taskCreate')}
      extra={(
        // KPI 导出 Tab 不在此建任务（导出入口在仪表盘/任务详情，T4），故隐藏新建按钮。
        selectedCategory === 'kpi_export' ? null : (
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
        )
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
              onChange={(key) => {
                // 用户手动点选分类 Tab：标记手选，#127 的自动回退不再覆盖
                categoryManuallyPickedRef.current = true;
                setSelectedCategory(key);
              }}
            />
            {/* MR 测量 / KPI 导出 Tab 选中时：直接在 Tabs 下方内联渲染各自的任务管理面板
                （视觉上就是 Tab 切换内容）。
                qa-614 c6 #368：删除原『模板』子页签（与下方执行视图任务列表上方的 typeCode
                下拉框 1:1 重复，DOM 里多余的 ant-space-item）。typeCode 选择统一收口到
                执行视图任务列表上方的 Select，单一入口。 */}
            {selectedCategory === 'mr_measurement' ? (
              <MRTasksPanel
                createOpen={mrCreateOpen}
                onCreateOpenChange={setMrCreateOpen}
              />
            ) : selectedCategory === 'kpi_export' ? (
              <KpiExportTasksPanel />
            ) : null}
          </Space>
        </Card>

        {/* 下方"任务列表 / 设备列表"区域：MR 测量 / KPI 导出 Tab 选中时隐藏（各自面板已内嵌在上方 Card） */}
        {selectedCategory !== 'mr_measurement' && selectedCategory !== 'kpi_export' && (
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
                      {showTaskTypeFilter ? (
                        <Select
                          allowClear
                          placeholder={t('ufte.filter.templateName')}
                          value={selectedTypeCode}
                          onChange={(value) => setSelectedTypeCode(value)}
                          options={taskTypeOptions}
                          style={{ width: 260 }}
                        />
                      ) : null}
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
                      {showTaskTypeFilter ? (
                        <Select
                          allowClear
                          placeholder={t('ufte.filter.templateName')}
                          value={selectedTypeCode}
                          onChange={(value) => setSelectedTypeCode(value)}
                          options={taskTypeOptions}
                          style={{ width: 260 }}
                        />
                      ) : null}
                      <Select
                        allowClear
                        showSearch
                        placeholder={t('ufte.filter.productType')}
                        value={deviceProductNameFilter}
                        onChange={(value) => setDeviceProductNameFilter(value)}
                        options={deviceProductNameOptions}
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
                          // 升级类（Download RPC）
                          { label: t('ufte.status.downloading'), value: 'downloading' },
                          // 版本回退类（GPV 检查 + SPV 触发）
                          { label: t('ufte.status.rollbackChecking'), value: 'rollback_checking' },
                          { label: t('ufte.status.rollingBack'), value: 'rolling_back' },
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
          {isUPSUpgradeDrawer ? (
            <Alert
              type="info"
              showIcon
              message={t('ufte.form.upsInformDispatchHint')}
              style={{ marginBottom: 16 }}
            />
          ) : null}
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
                        onClick={() => window.open(firmwareManagerUrl, '_blank')}
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
              <Form.Item
                label={t('ufte.form.concurrency')}
                name="concurrency"
                extra={t('ufte.form.concurrency.hint')}
              >
                <InputNumber min={1} max={100} precision={0} style={{ width: 120 }} />
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
                <Button
                  type="link"
                  size="small"
                  style={{ padding: 0 }}
                  disabled={drawerSelectedDevices.length === 0}
                  onClick={() => setSelectedDevicesModalOpen(true)}
                >
                  {t('ufte.form.selectedCount', { count: drawerSelectedDevices.length })}
                </Button>
              </Space>
              <Table<UnifiedFileTransferDeviceItem>
                size="small"
                rowKey="id"
                loading={drawerDevicesLoading}
                columns={drawerDeviceColumns}
                dataSource={drawerDeviceCandidates}
                pagination={{
                  current: drawerDevicePage,
                  pageSize: drawerDevicePageSize,
                  total: drawerDeviceTotal,
                  showSizeChanger: true,
                  showTotal: (total) => t('ufte.form.deviceTotal', { count: total }),
                  onChange: (page, pageSize) => {
                    setDrawerDevicePage(page);
                    setDrawerDevicePageSize(pageSize);
                  },
                }}
                rowSelection={{
                  selectedRowKeys: selectedDrawerDeviceIds,
                  // #215: 真分页后 preserveSelectedRowKeys 让其它页的已选项不被
                  // 当前页 onChange 覆盖丢失（受控 selectedRowKeys 跨页保留）。
                  preserveSelectedRowKeys: true,
                  onChange: (selectedRowKeys, selectedRows) => {
                    setSelectedDrawerDeviceIds(selectedRowKeys.map((item) => String(item)));
                    // 当前页勾选的设备对象并入 map，供已选清单展示。
                    setSelectedDeviceMap((prev) => {
                      const next = { ...prev };
                      for (const row of selectedRows) {
                        if (row) {
                          next[row.id] = row;
                        }
                      }
                      return next;
                    });
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
                            return lic ? formatSystemTime(lic.updateTime, { format: 'YYYY-MM-DD HH:mm', placeholder: '-' }) : '—';
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
                            return snap ? formatSystemTime(snap.updateTime, { format: 'YYYY-MM-DD HH:mm', placeholder: '-' }) : '—';
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
            提交时 handleCreateTask 走 form.getFieldValue('scheduledAt')（dayjs 对象）→ 按系统时区附加偏移。
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
                        return isTransferSystemTimeAfter(value, nowInSystemTimezone(systemTimezone), systemTimezone)
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
                    disabledDate={(current) => Boolean(current && isTransferSystemDateBefore(current, nowInSystemTimezone(systemTimezone), systemTimezone))}
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

        {/* #215 + 已选清单升级：Modal 内用 Table，前端模糊搜索 SN + 分页 10/20/50/100，子组件配合 destroyOnHidden 重置状态。 */}
        <Modal
          title={t('ufte.selectedModal.title', { count: drawerSelectedDevices.length })}
          open={selectedDevicesModalOpen}
          onCancel={() => setSelectedDevicesModalOpen(false)}
          footer={null}
          width={720}
          destroyOnHidden
        >
          <SelectedDevicesPanel
            devices={drawerSelectedDevices}
            onRemove={handleRemoveSelectedDevice}
          />
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
            {/* qa-614 c6 #367：标签/内容均单行不换行（fontSize 12 收紧详情区），
                超长值（任务名称/时间）整行占满（span=2）+ Tooltip 展示全文。 */}
            <Descriptions
              column={2}
              size="small"
              bordered
              labelStyle={DETAIL_LABEL_STYLE}
              contentStyle={DETAIL_CONTENT_STYLE}
            >
              <Descriptions.Item label={t('ufte.col.taskName')} span={2}>
                <Tooltip title={detailTask.taskName}>{detailTask.taskName}</Tooltip>
              </Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.taskType')}>{localizeBuiltinTypeName(detailTask.typeCode, detailTask.typeDisplayName, t)}</Descriptions.Item>
              <Descriptions.Item label={t('common.status')}>{renderTaskStatus(detailTask.status, t)}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.result')}>
                {detailTask.result
                  ? <Tag color={detailTask.result === 'success' ? 'success' : detailTask.result === 'partial' ? 'warning' : 'error'}>{detailTask.result === 'success' ? t('ufte.result.success') : detailTask.result === 'partial' ? t('ufte.result.partial') : detailTask.result === 'terminated' ? t('ufte.result.terminated') : t('ufte.result.failed')}</Tag>
                  : '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.destVersion')}>
                <Tooltip title={getTaskTargetVersion(detailTask)}>{getTaskTargetVersion(detailTask)}</Tooltip>
              </Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.productType')}>{getTaskProductClass(detailTask)}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.executionMode')}>
                <Tag>{executionModeOptions.find((o) => o.value === detailTask.executionMode)?.label ?? detailTask.executionMode}</Tag>
              </Descriptions.Item>
              {detailTask.executionMode === 'scheduled' && detailTask.scheduledAt ? (
                <Descriptions.Item label={t('ufte.form.scheduledAt')}>
                  {formatSystemTime(detailTask.scheduledAt)}
                </Descriptions.Item>
              ) : null}
              <Descriptions.Item label={t('ufte.col.currentStep')}>{stepLabels[detailTask.currentStep as TransferStepId] ?? '-'}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.operator')}>{detailTask.createUser}</Descriptions.Item>
              <Descriptions.Item label={t('ufte.col.createdAt')}>
                <Tooltip title={formatSystemTime(detailTask.createdAt)}>
                  {formatSystemTime(detailTask.createdAt)}
                </Tooltip>
              </Descriptions.Item>
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
            {/* #615：任务详情抽屉补「已选设备列表」——按 taskId 走 /ufte/devices 拉子任务，
                10s 自动刷新（与外层「执行明细」共用 hook）。lazy 渲染，不阻塞抽屉打开。 */}
            <TaskDetailDevicesPanel taskId={detailTask.id} />
          </Space>
        ) : null}
      </Drawer>
    </ListPageLayout>
  );
}

// TaskDetailDevicesPanel —— 详情抽屉内嵌「已选设备 / 执行明细」表。
// 与外层「执行明细」页签共用 useUnifiedFileTransferDevices，仅多传 taskId 收窄到当前任务。
// 后端 matchesDeviceFilter 按 DeviceItem.TaskID 精确匹配；空 taskId 不会发生（detailTask 必有 id）。
function TaskDetailDevicesPanel({ taskId }: { taskId: string }) {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const { data, isLoading } = useUnifiedFileTransferDevices({
    taskId,
    page,
    pageSize,
  });
  const rows = data?.items ?? [];
  const total = data?.total ?? 0;

  const columns: ColumnsType<UnifiedFileTransferDeviceItem> = [
    {
      title: t('ufte.col.deviceSn'),
      dataIndex: 'deviceSn',
      key: 'deviceSn',
      width: 150,
      ellipsis: true,
      render: (sn: string) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{sn || '-'}</Text>,
    },
    {
      title: t('ufte.col.productType'),
      dataIndex: 'productName',
      key: 'productName',
      width: 110,
      ellipsis: true,
      render: (_, record) => <Text style={{ fontSize: 12 }}>{record.productName || record.productType || '-'}</Text>,
    },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (_, record) => renderDeviceStatus(record.status, t),
    },
    {
      title: t('ufte.col.progress'),
      dataIndex: 'progress',
      key: 'progress',
      width: 90,
      render: (p: number) => <Progress percent={p} size="small" />,
    },
    {
      title: t('software.failureReason'),
      dataIndex: 'failureReason',
      key: 'failureReason',
      ellipsis: true,
      render: (reason: string | undefined, record) => {
        if (!reason) {
          return <Text type="secondary" style={{ fontSize: 12 }}>-</Text>;
        }
        const { display } = formatFailureReasonDisplay(reason, t);
        return (
          <Tooltip title={record.failureDetail || reason}>
            <Text type="danger" style={{ fontSize: 12 }}>{display}</Text>
          </Tooltip>
        );
      },
    },
  ];

  return (
    <Card
      title={t('ufte.tab.deviceList')}
      size="small"
      extra={<Text type="secondary" style={{ fontSize: 12 }}>{t('ufte.tag.totalCount', { count: total })}</Text>}
    >
      <Table<UnifiedFileTransferDeviceItem>
        rowKey="id"
        size="small"
        loading={isLoading}
        columns={columns}
        dataSource={rows}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          pageSizeOptions: ['10', '20', '50', '100'],
          size: 'small',
          onChange: (nextPage, nextSize) => {
            setPage(nextPage);
            if (nextSize && nextSize !== pageSize) {
              setPageSize(nextSize);
            }
          },
        }}
        scroll={{ x: 600 }}
      />
    </Card>
  );
}

// SelectedDevicesPanel —— 已选清单 Modal 内嵌面板：SN 模糊搜索 + 分页 10/20/50/100 + 单台移除。
// 数据源是父组件 selectedDeviceMap → drawerSelectedDevices 数组（in-memory），无后端请求。
// Modal destroyOnHidden 时本组件卸载，搜索词/页码自动重置。
function SelectedDevicesPanel({
  devices,
  onRemove,
}: {
  devices: UnifiedFileTransferDeviceItem[];
  onRemove: (id: string) => void;
}) {
  const t = useT();
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return devices;
    return devices.filter((d) => (d.deviceSn || '').toLowerCase().includes(kw));
  }, [devices, keyword]);

  const columns: ColumnsType<UnifiedFileTransferDeviceItem> = [
    {
      title: t('ufte.col.deviceSn'),
      dataIndex: 'deviceSn',
      key: 'deviceSn',
      ellipsis: true,
      render: (sn: string) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{sn || '-'}</Text>,
    },
    {
      title: t('ufte.col.productType'),
      dataIndex: 'productName',
      key: 'productName',
      width: 180,
      ellipsis: true,
      render: (_, record) => <Text style={{ fontSize: 12 }}>{record.productName || record.productType || '-'}</Text>,
    },
    {
      title: t('common.action'),
      key: 'action',
      width: 90,
      align: 'right',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          danger
          icon={<DeleteOutlined />}
          onClick={() => onRemove(record.id)}
        >
          {t('common.delete')}
        </Button>
      ),
    },
  ];

  return (
    <Space direction="vertical" size={8} style={{ width: '100%' }}>
      <Input.Search
        allowClear
        size="small"
        placeholder={t('ufte.selectedModal.searchPlaceholder')}
        value={keyword}
        onChange={(e) => {
          setKeyword(e.target.value);
          setPage(1);
        }}
      />
      <Table<UnifiedFileTransferDeviceItem>
        rowKey="id"
        size="small"
        columns={columns}
        dataSource={filtered}
        locale={{ emptyText: t('ufte.selectedModal.empty') }}
        pagination={{
          current: page,
          pageSize,
          total: filtered.length,
          showSizeChanger: true,
          pageSizeOptions: ['10', '20', '50', '100'],
          size: 'small',
          onChange: (nextPage, nextSize) => {
            setPage(nextPage);
            if (nextSize && nextSize !== pageSize) {
              setPageSize(nextSize);
            }
          },
        }}
      />
    </Space>
  );
}
