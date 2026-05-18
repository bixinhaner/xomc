import {
  Badge,
  Card,
  Space,
  Tag,
  Typography,
} from 'antd';

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
      return <Badge status="processing" text="下载中" />;
    case 'verifying':
      return <Badge status="processing" text="校验中" />;
    case 'suspended':
      return <Badge status="warning" text="已挂起" />;
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