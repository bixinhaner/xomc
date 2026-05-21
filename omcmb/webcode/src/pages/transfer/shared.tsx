import {
  Badge,
  Card,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { CSSProperties, ReactNode } from 'react';

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

export const STEP_LABELS: Record<TransferStepId, string> = {
  CHECK_PERMISSION: '权限检查',
  CHECK_ONLINE: '在线检查',
  CHECK_CONFLICT: '并发冲突检查',
  PRE_VALIDATE: '前置校验',
  SEND_RPC: '发送 RPC',
  WAIT_RPC_RESPONSE: '等待 RPC 响应',
  WAIT_FILE_TRANSFER: '等待文件传输',
  WAIT_TRANSFER_COMPLETE: '等待 TransferComplete',
  WAIT_INFORM_EVENT: '等待 Inform 事件',
  WAIT_REBOOT_COMPLETE: '等待重启完成',
};

export const EXECUTION_MODE_OPTIONS: Array<{ label: string; value: TransferExecutionMode }> = [
  { label: '立即执行', value: 'immediate' },
  { label: '计划执行', value: 'scheduled' },
  { label: '挂起创建', value: 'suspended' },
];

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

export const SOFTWARE_LIBRARY_FILE_TYPE_OPTIONS: Array<{ label: string; value: FirmwareLibraryFileType }> = [
  { label: '软件主镜像', value: 0 },
  { label: 'PATCH 补丁包', value: 1 },
  { label: 'AP 固件', value: 5 },
  { label: 'FPGA 文件', value: 6 },
];

export function getSoftwareLibraryFileTypeLabel(value?: FirmwareLibraryFileType) {
  if (value === undefined) {
    return '-';
  }
  return SOFTWARE_LIBRARY_FILE_TYPE_OPTIONS.find((item) => item.value === value)?.label ?? `类型 ${value}`;
}

export function renderTaskStatus(status: UnifiedFileTransferTask['status']) {
  switch (status) {
    case 'in_progress':
      return <Badge status="processing" text="执行中" />;
    case 'suspended':
      return <Badge status="warning" text="已挂起" />;
    case 'ended':
      return <Badge status="success" text="已结束" />;
    default:
      return <Badge status="default" text="待执行" />;
  }
}

export function renderDeviceStatus(status: UnifiedFileTransferDeviceItem['status']) {
  switch (status) {
    case 'downloading':
      // 升级 / 回滚类 Download RPC：CPE 正在从 ACS 拉镜像 / 补丁文件
      return <Badge status="processing" text="下载中" />;
    case 'uploading':
      // 备份 / 日志采集 Upload RPC：Upload 命令已派发，等 UploadResponse + CPE 通过 HTTP PUT
      // 把文件上传到 ACS（这两步在 TR-069 上紧贴，ACS 侧合并到同一段展示）
      return <Badge status="processing" text="上传中" />;
    case 'awaiting_tc':
      // 备份 / 日志采集 Upload RPC：文件已落到 ACS MinIO（backup_restore_file 已 upsert），
      // 等 CPE 主动发 TransferComplete SOAP 来结束传输事务
      return <Badge status="processing" text="等待 TransferComplete" />;
    case 'verifying':
      return <Badge status="processing" text="校验中" />;
    // 设备子任务的 'suspended' 既可能是"用户挂起创建"也可能是"设备离线等待"，
    // 后者占比更高（设备 inform 间隔 5 min，挂起→开始时常碰到设备短暂掉线）。
    // 合并文案为"已挂起 / 待上线"，避免用户以为操作未生效。详见
    // docs/project/backup-display-fix-20260520.md F11。
    case 'suspended':
      return <Badge status="warning" text="已挂起 / 待上线" />;
    case 'ended':
      return <Badge status="success" text="已完成" />;
    case 'failed':
      return <Badge status="error" text="失败" />;
    default:
      return <Badge status="default" text="待执行" />;
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
    throw new Error('请选择现有业务 Tab，或者输入新的 Tab 名称');
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
          <Text strong>{taskType.displayName}</Text>
          <Tag color={taskType.builtIn ? 'blue' : 'gold'}>{taskType.builtIn ? '内置' : '自定义'}</Tag>
          <Tag>{taskType.rpcType}</Tag>
          <Tag color="cyan">FileType {taskType.fileType}</Tag>
          {taskType.firmwareFileType !== undefined ? (
            <Tag color="geekblue">文件库 {getSoftwareLibraryFileTypeLabel(taskType.firmwareFileType)}</Tag>
          ) : null}
          <Tag color={taskType.enabled ? 'green' : 'default'}>{taskType.enabled ? '已启用' : '已停用'}</Tag>
        </Space>
        <Text type="secondary">{taskType.description}</Text>
      </Space>
    </Card>
  );
}