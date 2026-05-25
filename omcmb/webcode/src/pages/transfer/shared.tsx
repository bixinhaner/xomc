import {
  Badge,
  Card,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { CSSProperties, ReactNode } from 'react';

import { useT } from '@/hooks/useT';
import type {
  FirmwareLibraryFileType,
  TransferExecutionMode,
  TransferRpcType,
  TransferStepId,
  UnifiedFileTransferCategory,
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferTask,
  UnifiedFileTransferTaskType,
} from '@core/types/unifiedFileTransfer';

const { Text } = Typography;

// 翻译函数签名 —— useT 返回的 (id, values?) => string，简化为窄类型供工具函数使用。
type Translate = (id: string, values?: Record<string, string | number>) => string;

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
  'VERSION_ROLLBACK',
  'RUNTIME_LOG_COLLECT', 'FAULT_LOG_COLLECT',
  'CONFIG_BACKUP', 'CONFIG_BACKUP_NV', 'CONFIG_BACKUP_XML',
  'CONFIG_RESTORE',
  'LICENSE_UPGRADE',
]);

const BUILTIN_CATEGORY_CODES = new Set([
  'enb_upgrade', 'gnb_upgrade', 'version_rollback',
  'station_log', 'config_backup', 'config_restore', 'license_upgrade',
]);

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
  'gnb_upgrade',
  'enb_upgrade',
  'version_rollback',
  'station_log',
  'config_backup',
  'config_restore',
];

// typeCode 子 tab 顺序由后端 ufte_task_types.sort_order 字段控制
// （migrations/000146 加列 + builtInTaskTypes 写默认值）。前端不再二次排序，
// 直接采用 API 返回顺序；如需调整，在「模板配置」UI 修改 sort_order 即可。

// renderEllipsisCell — 长文本列单行截断 + Tooltip 兜底。任务名称 / 设备名 /
// 固件版本 / 文件名 等业务字段都可能超过列宽，统一封装避免每列写重复 markup。
//
// 实现要点：用 <div> 块级容器 + CSS（overflow:hidden, text-overflow:ellipsis,
// white-space:nowrap）而不是 Antd Text ellipsis。原因：Text ellipsis 渲染为
// <span>（inline），被 Tooltip wrapper（也是 inline）包后，maxWidth/width 都
// 不生效；只能依赖 Text 内部 display:inline-block 但实测在 Table fixed layout
// 下仍会触发换行。改用 block div 自然撑满 td 宽度，css ellipsis 100% 可靠。
//
// 用法：
//   render: (_, record) => renderEllipsisCell(record.taskName)
//   render: (_, record) => renderEllipsisCell(record.taskName, { strong: true })
//   render: (_, record) => renderEllipsisCell(record.taskName, {
//     subtitle: record.deviceName,  // 副行也跟着 ellipsis + tooltip
//   })
export function renderEllipsisCell(
  value: string | undefined | null,
  opts?: {
    strong?: boolean;
    secondary?: boolean;
    subtitle?: string | undefined | null;
    /** 业务渲染失败时的 fallback，比如 '-'。默认 '-'。 */
    placeholder?: ReactNode;
  },
): ReactNode {
  const display = (value ?? '').toString().trim();
  const placeholder = opts?.placeholder ?? '-';
  const sub = (opts?.subtitle ?? '').toString().trim();
  if (!display && !sub) return placeholder;

  const baseCellStyle: CSSProperties = {
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    // block 让 maxWidth/width 100% 跟着 td 宽度走，避免 Tooltip wrapper inline 时
    // maxWidth 失效导致换行。
    display: 'block',
    width: '100%',
  };

  const mainNode = display ? (
    <Tooltip title={display} placement="topLeft" mouseEnterDelay={0.2}>
      <div
        style={{
          ...baseCellStyle,
          fontWeight: opts?.strong ? 600 : undefined,
          color: opts?.secondary ? 'rgba(0,0,0,0.45)' : undefined,
        }}
      >
        {display}
      </div>
    </Tooltip>
  ) : null;

  if (!sub) return mainNode ?? placeholder;

  return (
    <div style={{ minWidth: 0 }}>
      {mainNode}
      <Tooltip title={sub} placement="topLeft" mouseEnterDelay={0.2}>
        <div style={{ ...baseCellStyle, color: 'rgba(0,0,0,0.45)', fontSize: 12, marginTop: 2 }}>
          {sub}
        </div>
      </Tooltip>
    </div>
  );
}

export const TYPE_DRAWER_DEFAULT_STEPS: TransferStepId[] = [
  'CHECK_PERMISSION',
  'CHECK_ONLINE',
  'SEND_RPC',
  'WAIT_RPC_RESPONSE',
  'WAIT_TRANSFER_COMPLETE',
];

export const UPGRADE_LIKE_CATEGORIES = new Set(['gnb_upgrade', 'enb_upgrade', 'version_rollback']);

// TASK_NAME_PREFIX_BY_TYPE_I18N：UFTE 新建任务默认名前缀，按 typeCode 区分业务，
// 同时提供 zh-CN / en-US 两套字面值。命名规则与升级模块沿袭：
//   <prefix>_<username>_<YYYY-MM-DD HH:mm:ss>
// 例：中文环境 → "运行日志_admin_2026-05-25 13:40:36"
//     英文环境 → "RuntimeLog_admin_2026-05-25 13:40:36"
export const TASK_NAME_PREFIX_BY_TYPE_I18N: Record<string, { zh: string; en: string }> = {
  ENB_IMG_UPGRADE:     { zh: '4G升级',         en: 'Upgrade' },
  GNB_IMG_UPGRADE:     { zh: '5G升级',         en: 'Upgrade' },
  ENB_PATCH_UPGRADE:   { zh: '基站补丁升级',   en: 'Upgrade' },
  ENB_FPGA_UPGRADE:    { zh: 'FPGA升级',       en: 'Upgrade' },
  VERSION_ROLLBACK:    { zh: '版本回退',       en: 'Rollback' },
  CONFIG_BACKUP_NV:    { zh: '配置备份NV',     en: 'ConfigBackupNV' },
  CONFIG_BACKUP_XML:   { zh: '配置备份XML',    en: 'ConfigBackupXML' },
  CONFIG_RESTORE:      { zh: '配置文件恢复',   en: 'ConfigRestore' },
  RUNTIME_LOG_COLLECT: { zh: '运行日志',       en: 'RuntimeLog' },
  FAULT_LOG_COLLECT:   { zh: '故障日志',       en: 'FaultLog' },
  LICENSE_UPGRADE:     { zh: '设备License升级', en: 'LicenseUpgrade' },
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

export function renderTaskStatus(status: UnifiedFileTransferTask['status'], t: Translate) {
  switch (status) {
    case 'in_progress':
      return <Badge status="processing" text={t('ufte.status.inProgress')} />;
    case 'suspended':
      return <Badge status="warning" text={t('ufte.status.suspended')} />;
    case 'ended':
      return <Badge status="success" text={t('ufte.status.ended')} />;
    default:
      return <Badge status="default" text={t('ufte.status.pending')} />;
  }
}

export function renderDeviceStatus(status: UnifiedFileTransferDeviceItem['status'], t: Translate) {
  switch (status) {
    case 'downloading':
      // 升级 / 回滚类 Download RPC：CPE 正在从 ACS 拉镜像 / 补丁文件
      return <Badge status="processing" text={t('ufte.status.downloading')} />;
    case 'uploading':
      // 备份 / 日志采集 Upload RPC：Upload 命令已派发，等 UploadResponse + CPE 通过 HTTP PUT
      // 把文件上传到 ACS（这两步在 TR-069 上紧贴，ACS 侧合并到同一段展示）
      return <Badge status="processing" text={t('ufte.status.uploading')} />;
    case 'awaiting_tc':
      // 备份 / 日志采集 Upload RPC：文件已落到 ACS MinIO（backup_restore_file 已 upsert），
      // 等 CPE 主动发 TransferComplete SOAP 来结束传输事务
      return <Badge status="processing" text={t('ufte.status.awaitingTc')} />;
    case 'verifying':
      return <Badge status="processing" text={t('ufte.status.verifying')} />;
    // 设备子任务的 'suspended' 既可能是"用户挂起创建"也可能是"设备离线等待"，
    // 后者占比更高（设备 inform 间隔 5 min，挂起→开始时常碰到设备短暂掉线）。
    // 合并文案为"已挂起 / 待上线"，避免用户以为操作未生效。详见
    // docs/project/backup-display-fix-20260520.md F11。
    case 'suspended':
      return <Badge status="warning" text={t('ufte.status.suspendedOrOffline')} />;
    case 'ended':
      return <Badge status="success" text={t('ufte.status.completed')} />;
    case 'failed':
      return <Badge status="error" text={t('ufte.status.failed')} />;
    default:
      return <Badge status="default" text={t('ufte.status.pending')} />;
  }
}

export function buildCategoryTabs(taskTypes: UnifiedFileTransferTaskType[]): CategoryTabItem[] {
  const categoryMap = new Map<string, CategoryTabItem>();
  taskTypes.forEach((item) => {
    const existing = categoryMap.get(item.category);
    if (existing) {
      existing.templateCount += 1;
      return;
    }
    categoryMap.set(item.category, {
      category: item.category,
      categoryLabel: item.categoryLabel,
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

export function TransferTemplateCard({
  taskType,
  active = false,
}: {
  taskType: UnifiedFileTransferTaskType;
  active?: boolean;
}) {
  const t = useT();
  const displayName = localizeBuiltinTypeName(taskType.typeCode, taskType.displayName, t);
  const description = localizeBuiltinDescription(taskType.typeCode, taskType.description, t);
  return (
    <Card
      size="small"
      style={{
        borderColor: active ? '#1677ff' : undefined,
        boxShadow: active ? '0 0 0 1px rgba(22,119,255,0.18)' : undefined,
        height: '100%',
      }}
    >
      <Space direction="vertical" size={8} style={{ width: '100%' }}>
        <Space wrap>
          <Text strong>{displayName}</Text>
          <Tag color={taskType.builtIn ? 'blue' : 'gold'}>{taskType.builtIn ? t('ufte.tag.builtIn') : t('ufte.tag.custom')}</Tag>
          <Tag>{taskType.rpcType}</Tag>
          <Tag color="cyan">FileType {taskType.fileType}</Tag>
          {taskType.firmwareFileType !== undefined ? (
            <Tag color="geekblue">{t('ufte.template.softLibTag', { label: getSoftwareLibraryFileTypeLabel(taskType.firmwareFileType, t) })}</Tag>
          ) : null}
          <Tag color={taskType.enabled ? 'green' : 'default'}>{taskType.enabled ? t('ufte.template.enabled') : t('ufte.template.disabled')}</Tag>
        </Space>
        <Text type="secondary">{description}</Text>
      </Space>
    </Card>
  );
}