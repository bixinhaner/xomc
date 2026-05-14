export type TransferRpcType = 'DOWNLOAD' | 'UPLOAD' | 'SET_PARAM_VALUES';

export type UnifiedFileTransferCategory = string;

export type TransferStepId =
  | 'CHECK_PERMISSION'
  | 'CHECK_ONLINE'
  | 'CHECK_CONFLICT'
  | 'PRE_VALIDATE'
  | 'SEND_RPC'
  | 'WAIT_RPC_RESPONSE'
  | 'WAIT_FILE_TRANSFER'
  | 'WAIT_TRANSFER_COMPLETE'
  | 'WAIT_INFORM_EVENT';

export type TransferTaskStatus = 'pending' | 'in_progress' | 'suspended' | 'ended';

export type TransferTaskResult = 'success' | 'partial' | 'failure' | 'terminated';

export type TransferExecutionMode = 'immediate' | 'scheduled' | 'suspended';

export interface UnifiedFileTransferOverview {
  enabledTypeCount: number;
  runningTaskCount: number;
  customTypeCount: number;
  successRate30d: number;
}

export interface UnifiedFileTransferTaskType {
  typeCode: string;
  category: UnifiedFileTransferCategory;
  categoryLabel: string;
  displayName: string;
  description: string;
  rpcType: TransferRpcType;
  builtIn: boolean;
  enabled: boolean;
  stepChain: TransferStepId[];
  postTcEventCode?: string;
  permissionCode: string;
  platformScope: string[];
  fileType: string;
  fileTypeLabel: string;
  fileTypeEditable: boolean;
  urlTemplate?: string;
  targetFileNameTemplate?: string;
  fileNameTemplate?: string;
  fileSizeField?: string;
  checksumField?: string;
  rawMode?: string;
  delaySeconds?: number;
  transportPath?: string;
  lastEditor: string;
  taskCount30d: number;
  successRate30d: number;
  updatedAt: string;
}

export interface UnifiedFileTransferTask {
  id: string;
  taskName: string;
  category: UnifiedFileTransferCategory;
  categoryLabel: string;
  typeCode: string;
  typeDisplayName: string;
  firmwareId?: string;
  targetVersion?: string;
  productType?: string;
  status: TransferTaskStatus;
  result?: TransferTaskResult;
  progress: number;
  totalCount: number;
  successCount: number;
  failCount: number;
  currentStep: TransferStepId;
  executionMode: TransferExecutionMode;
  createUser: string;
  createdAt: string;
  scheduledAt?: string;
  operatorScope: string;
}

export type UnifiedFileTransferDeviceStatus =
  | 'pending'
  | 'downloading'
  | 'verifying'
  | 'suspended'
  | 'ended'
  | 'failed';

export interface UnifiedFileTransferDeviceItem {
  id: string;
  taskId: string;
  taskName: string;
  category: UnifiedFileTransferCategory;
  categoryLabel: string;
  typeCode: string;
  typeDisplayName: string;
  deviceName: string;
  deviceSn: string;
  productType: string;
  currentVersion: string;
  targetVersion: string;
  status: UnifiedFileTransferDeviceStatus;
  result?: TransferTaskResult;
  progress: number;
  lastReportAt: string;
  operatorScope: string;
}

export interface CreateUnifiedFileTransferTaskInput {
  taskName: string;
  typeCode: string;
  firmwareId?: string;
  deviceCount: number;
  executionMode: TransferExecutionMode;
  note?: string;
}

export interface CreateUnifiedFileTransferTypeInput {
  category: UnifiedFileTransferCategory;
  categoryLabel: string;
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
  urlTemplate?: string;
  targetFileNameTemplate?: string;
  fileNameTemplate?: string;
  fileSizeField?: string;
  checksumField?: string;
  rawMode?: string;
  delaySeconds?: number;
  transportPath?: string;
}

export interface UpdateUnifiedFileTransferTaskTypeInput extends CreateUnifiedFileTransferTypeInput {
  typeCode: string;
}