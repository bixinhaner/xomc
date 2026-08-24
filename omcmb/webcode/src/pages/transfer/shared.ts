import type {
  FirmwareLibraryFileType,
  TransferExecutionMode,
  TransferStepId,
  TransferRpcType,
  UnifiedFileTransferCategory,
  UnifiedFileTransferTaskType,
} from '@core/types/unifiedFileTransfer';

// 翻译函数签名 —— useT 返回的 (id, values?) => string，简化为窄类型供工具函数使用。
type Translate = (id: string, values?: Record<string, string | number>) => string;

/**
 * UFTE 共享纯工具 / 常量 / 类型。
 * 含 JSX 的 render 辅助（renderEllipsisCell / renderTaskStatus / renderDeviceStatus）
 * 与组件 TransferTemplateCard 已分别拆到 ./shared.render.tsx、./TransferTemplateCard.tsx
 * （react-refresh/only-export-components：JSX/组件与纯工具分文件）。
 */

/** 步骤链翻译。原 STEP_LABELS 常量已下线 —— 需要 t 才能本地化。 */
export function getStepLabels(t: Translate): Record<TransferStepId, string> {
  return {
    CHECK_PERMISSION: t('ufte.step.CHECK_PERMISSION'),
    CHECK_ONLINE: t('ufte.step.CHECK_ONLINE'),
    CHECK_CONFLICT: t('ufte.step.CHECK_CONFLICT'),
    PRE_VALIDATE: t('ufte.step.PRE_VALIDATE'),
    SEND_RPC: t('ufte.step.SEND_RPC'),
    WAIT_RPC_RESPONSE: t('ufte.step.WAIT_RPC_RESPONSE'),
    WAIT_FILE_TRANSFER: t('ufte.step.WAIT_FILE_TRANSFER'),
    WAIT_TRANSFER_COMPLETE: t('ufte.step.WAIT_TRANSFER_COMPLETE'),
    WAIT_INFORM_EVENT: t('ufte.step.WAIT_INFORM_EVENT'),
    WAIT_REBOOT_COMPLETE: t('ufte.step.WAIT_REBOOT_COMPLETE'),
  };
}

/** 执行方式选项 —— 同 STEP_LABELS 的转换，i18n 改造后需要 t。 */
export function getExecutionModeOptions(t: Translate): Array<{ label: string; value: TransferExecutionMode }> {
  return [
    { label: t('ufte.execMode.immediate'), value: 'immediate' },
    { label: t('ufte.execMode.scheduled'), value: 'scheduled' },
    { label: t('ufte.execMode.suspended'), value: 'suspended' },
  ];
}

// 内置 taskType / category 的翻译映射 —— seed 数据写死中文，按 typeCode/category 查表覆盖。
// 自定义模板（builtIn=false）的 displayName 是用户输入，**不翻译**，原样使用 fallback。
const BUILTIN_TYPE_CODES = new Set([
  'ENB_IMG_UPGRADE', 'ENB_PATCH_UPGRADE', 'ENB_FPGA_UPGRADE',
  'GNB_IMG_UPGRADE', 'GNB_FPGA_UPGRADE',
  'GSM_IMG_UPGRADE', // qa-614 c6 #365 #373：2G/GSM 升级
  'UPS_AP_UPGRADE',
  'VERSION_ROLLBACK',
  'RUNTIME_LOG_COLLECT', 'FAULT_LOG_COLLECT',
  'CONFIG_BACKUP', 'CONFIG_BACKUP_NV', 'CONFIG_BACKUP_XML',
  'CONFIG_RESTORE',
  'LICENSE_UPGRADE',
  'IMS_FILE_COLLECT', 'IMS_FILE_DISTRIBUTE', // 核心网文件采集/下发（docs/design/imscore-file-transfer.md）
]);

const BUILTIN_CATEGORY_CODES = new Set([
  'enb_upgrade', 'gnb_upgrade',
  'gsm_upgrade', // qa-614 c6 #365 #373：2G/GSM 升级分类
  'ups_upgrade',
  'device_upgrade', // qa-614 c6 #368 / UPS：4G/5G/2G/UPS 合并的虚拟『设备升级』分类
  'version_rollback',
  'station_log', 'config_backup', 'config_restore', 'license_upgrade',
  'ims_core', // 核心网（IMS Core）文件传输
  // F05：MR 测量虚拟分类（不走 UFTE 模板，作为入口聚合按钮跳到 /mr/tasks）
  'mr_measurement',
  // KPI-EXPORT：KPI 导出虚拟分类（不走 UFTE 模板，内联渲染 KpiExportTasksPanel）
  'kpi_export',
]);

// qa-614 c6 #368：4G(enb_upgrade) + 5G(gnb_upgrade) + 2G(gsm_upgrade) + UPS(ups_upgrade) 在展示层合并为虚拟分类
// 'device_upgrade'（『设备升级』）。聚合/展开逻辑下沉到 frontend-core（页面共享，
// 见 @core/utils/ufteCategory），v1 这里只做 re-export 保持现有 import 路径不变。
export {
  DEVICE_UPGRADE_CATEGORY,
  DEVICE_UPGRADE_MEMBER_CATEGORIES,
  isDeviceUpgradeMember,
  resolveBackendCategoryParam,
  filterTaskTypesForCategory,
  filterTaskTypesByUPSLicense,
  filterFileTransferItemsByUPSLicense,
  isUPSFileTransferScope,
} from '@core/utils/ufteCategory';
import {
  DEVICE_UPGRADE_CATEGORY as DEVICE_UPGRADE_CATEGORY_LOCAL,
  isDeviceUpgradeMember as isDeviceUpgradeMemberLocal,
} from '@core/utils/ufteCategory';

/** 内置 taskType displayName 翻译。非内置 typeCode（用户自定义）原样返回 fallback。 */
export function localizeBuiltinTypeName(
  typeCode: string | undefined,
  fallback: string,
  t: Translate,
): string {
  if (!typeCode || !BUILTIN_TYPE_CODES.has(typeCode)) return fallback;
  const key = `ufte.builtin.type.${typeCode}`;
  const translated = t(key);
  return translated && translated !== key ? translated : fallback;
}

/** 内置 category label 翻译。自定义分类（运行时生成的 custom_xxx）原样返回 fallback。 */
export function localizeBuiltinCategoryLabel(
  category: string | undefined,
  fallback: string,
  t: Translate,
): string {
  if (!category || !BUILTIN_CATEGORY_CODES.has(category)) return fallback;
  const key = `ufte.builtin.category.${category}`;
  const translated = t(key);
  return translated && translated !== key ? translated : fallback;
}

/** 内置 taskType description 翻译。非内置原样返回 fallback。 */
export function localizeBuiltinDescription(
  typeCode: string | undefined,
  fallback: string,
  t: Translate,
): string {
  if (!typeCode || !BUILTIN_TYPE_CODES.has(typeCode)) return fallback;
  const key = `ufte.builtin.desc.${typeCode}`;
  const translated = t(key);
  return translated && translated !== key ? translated : fallback;
}

export const DEFAULT_CATEGORY_ORDER = [
  // #368/#483/UPS：4G/5G/2G/UPS 全部折叠进单个『设备升级』虚拟分类（buildCategoryTabs 已折叠），
  // 故这里不再单列 enb/gnb/gsm/ups_upgrade。
  'device_upgrade',
  'version_rollback',
  'station_log',
  'config_backup',
  'config_restore',
  'ims_core', // 核心网（与配置文件等并行的新 tab）
  'mr_measurement', // F05 末位
];

// typeCode 子 tab 顺序由后端 ufte_task_types.sort_order 字段控制
// （migrations/000146 加列 + builtInTaskTypes 写默认值）。前端不再二次排序，
// 直接采用 API 返回顺序；如需调整，在「模板配置」UI 修改 sort_order 即可。

export const TYPE_DRAWER_DEFAULT_STEPS: TransferStepId[] = [
  'CHECK_PERMISSION',
  'CHECK_ONLINE',
  'SEND_RPC',
  'WAIT_RPC_RESPONSE',
  'WAIT_TRANSFER_COMPLETE',
];

export const UPGRADE_LIKE_CATEGORIES = new Set([
  'gnb_upgrade', 'enb_upgrade',
  'gsm_upgrade', // qa-614 c6 #365 #373
  'ups_upgrade',
  'device_upgrade', // qa-614 c6 #368：合并虚拟分类
  // 版本回退（version_rollback）不再归入「升级类」列布局：它是独立业务，套升级列会多出
  // 无意义的「升级类型」列、且任务/设备列与「设备升级」逐列雷同（很不统一）。摘出后落入
  // 通用（非升级）列分支，与「日志收集/配置备份」同款，设备导出亦随 isUpgradeLikeCategory
  // 走 default 视图（与后端 CSV default 表头对齐）。
]);

// TASK_NAME_PREFIX_BY_TYPE_I18N：UFTE 新建任务默认名前缀，按 typeCode 区分业务，
// 同时提供 zh-CN / en-US 两套字面值。命名规则与升级模块沿袭：
//   <prefix>_<username>_<YYYY-MM-DD HH:mm:ss>
// 例：中文环境 → "运行日志_admin_2026-05-25 13:40:36"
//     英文环境 → "RuntimeLog_admin_2026-05-25 13:40:36"
export const TASK_NAME_PREFIX_BY_TYPE_I18N: Record<string, { zh: string; en: string }> = {
  ENB_IMG_UPGRADE:     { zh: '4G升级',         en: 'Upgrade' },
  GNB_IMG_UPGRADE:     { zh: '5G升级',         en: 'Upgrade' },
  GSM_IMG_UPGRADE:     { zh: '2G升级',         en: 'Upgrade' }, // qa-614 c6 #365 #373
  UPS_AP_UPGRADE:      { zh: 'UPS升级',        en: 'UPSUpgrade' },
  ENB_PATCH_UPGRADE:   { zh: '基站补丁升级',   en: 'Upgrade' },
  ENB_FPGA_UPGRADE:    { zh: 'FPGA升级',       en: 'Upgrade' },
  VERSION_ROLLBACK:    { zh: '版本回退',       en: 'Rollback' },
  CONFIG_BACKUP_NV:    { zh: '配置备份NV',     en: 'ConfigBackupNV' },
  CONFIG_BACKUP_XML:   { zh: '配置备份XML',    en: 'ConfigBackupXML' },
  CONFIG_RESTORE:      { zh: '配置文件恢复',   en: 'ConfigRestore' },
  RUNTIME_LOG_COLLECT: { zh: '运行日志',       en: 'RuntimeLog' },
  FAULT_LOG_COLLECT:   { zh: '故障日志',       en: 'FaultLog' },
  LICENSE_UPGRADE:     { zh: '设备License升级', en: 'LicenseUpgrade' },
  IMS_FILE_COLLECT:    { zh: '核心网文件采集', en: 'ImsFileCollect' },
  IMS_FILE_DISTRIBUTE: { zh: '核心网文件下发', en: 'ImsFileDistribute' },
};

/**
 * 跨页面统一的 UFTE 默认任务名生成。locale 取自 appStore.locale —
 * 'zh-' 前缀走 zh 表，其它走 en。FileTransferCenter 抽屉、设备列表批量操作等
 * 所有触发 UFTE 创建任务的地方都应走这一份，避免风格不一致。
 */
export function buildDefaultUfteTaskName(
  typeCode: string | undefined,
  username: string | undefined,
  locale: string,
  nowText: string,
): string {
  const useZh = locale.startsWith('zh');
  const entry = typeCode ? TASK_NAME_PREFIX_BY_TYPE_I18N[typeCode] : undefined;
  const prefix = entry ? (useZh ? entry.zh : entry.en) : (useZh ? '任务' : 'Task');
  const user = (username && username.trim()) || (useZh ? '用户' : 'user');
  return `${prefix}_${user}_${nowText}`;
}

export interface CategoryTabItem {
  category: UnifiedFileTransferCategory;
  categoryLabel: string;
  templateCount: number;
}

export interface TaskTypeFormValues {
  categorySelection?: string;
  categoryCustomLabel?: string;
  displayName: string;
  description: string;
  rpcType: TransferRpcType;
  stepChain: TransferStepId[];
  postTcEventCode?: string;
  enabled: boolean;
  platformScope: string[];
  /** #492 适用产品（产品英文名多选）。取代 platformScope 作为模板编辑的可选范围控件。 */
  products?: string[];
  fileType: string;
  fileTypeLabel: string;
  fileTypeEditable: boolean;
  firmwareFileType?: FirmwareLibraryFileType;
  urlTemplate?: string;
  targetFileNameTemplate?: string;
  fileNameTemplate?: string;
  fileSizeField?: string;
  checksumField?: string;
  rawMode?: string;
  delaySeconds?: number;
  transportPath?: string;
}

export function getSoftwareLibraryFileTypeOptions(
  t: Translate,
): Array<{ label: string; value: FirmwareLibraryFileType }> {
  return [
    { label: t('ufte.softLib.image'),      value: 0 },
    { label: t('ufte.softLib.patch'),      value: 1 },
    { label: t('ufte.softLib.apFirmware'), value: 5 },
    { label: t('ufte.softLib.fpga'),       value: 6 },
  ];
}

export function getSoftwareLibraryFileTypeLabel(value: FirmwareLibraryFileType | undefined, t: Translate): string {
  if (value === undefined) {
    return '-';
  }
  const opt = getSoftwareLibraryFileTypeOptions(t).find((item) => item.value === value);
  return opt ? opt.label : t('ufte.softLib.unknown', { value });
}

export function buildCategoryTabs(taskTypes: UnifiedFileTransferTaskType[]): CategoryTabItem[] {
  const categoryMap = new Map<string, CategoryTabItem>();
  taskTypes.forEach((item) => {
    // qa-614 c6 #368：4G(enb_upgrade) + 5G(gnb_upgrade) + 2G(gsm_upgrade) + UPS(ups_upgrade) 在展示层折叠为单个
    // 虚拟分类 device_upgrade（『设备升级』）。categoryLabel 用 key，渲染时由
    // localizeBuiltinCategoryLabel('device_upgrade') 翻译。
    const displayCategory = isDeviceUpgradeMemberLocal(item.category)
      ? DEVICE_UPGRADE_CATEGORY_LOCAL
      : item.category;
    const displayLabel = isDeviceUpgradeMemberLocal(item.category)
      ? DEVICE_UPGRADE_CATEGORY_LOCAL
      : item.categoryLabel;
    const existing = categoryMap.get(displayCategory);
    if (existing) {
      existing.templateCount += 1;
      return;
    }
    categoryMap.set(displayCategory, {
      category: displayCategory,
      categoryLabel: displayLabel,
      templateCount: 1,
    });
  });

  return Array.from(categoryMap.values()).sort((left, right) => {
    const leftOrder = DEFAULT_CATEGORY_ORDER.indexOf(left.category);
    const rightOrder = DEFAULT_CATEGORY_ORDER.indexOf(right.category);
    if (leftOrder !== -1 || rightOrder !== -1) {
      if (leftOrder === -1) {
        return 1;
      }
      if (rightOrder === -1) {
        return -1;
      }
      return leftOrder - rightOrder;
    }
    return left.categoryLabel.localeCompare(right.categoryLabel, 'zh-CN');
  });
}

export function buildCategoryPayload(
  values: TaskTypeFormValues,
  categories: CategoryTabItem[],
  editingType?: UnifiedFileTransferTaskType | null,
): { category: string; categoryLabel: string } {
  const customLabel = values.categoryCustomLabel?.trim();
  if (customLabel) {
    const existingByLabel = categories.find((item) => item.categoryLabel === customLabel);
    if (existingByLabel) {
      return { category: existingByLabel.category, categoryLabel: existingByLabel.categoryLabel };
    }
    if (editingType && editingType.categoryLabel === customLabel) {
      return { category: editingType.category, categoryLabel: editingType.categoryLabel };
    }
    return {
      category: `custom_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 6)}`,
      categoryLabel: customLabel,
    };
  }

  if (!values.categorySelection) {
    // 不能本地化抛错 —— 该函数纯逻辑无 t 参数；调用方在 catch 里翻 i18n key
    // ufte.template.pickCategory 显示。这里抛 key 而非中文，保持与 i18n 体系一致。
    throw new Error('ufte.template.pickCategory');
  }

  const existingByKey = categories.find((item) => item.category === values.categorySelection);
  if (existingByKey) {
    return { category: existingByKey.category, categoryLabel: existingByKey.categoryLabel };
  }

  return {
    category: values.categorySelection,
    categoryLabel: values.categorySelection,
  };
}
