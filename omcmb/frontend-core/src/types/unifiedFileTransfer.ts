export type TransferRpcType = 'DOWNLOAD' | 'UPLOAD' | 'SET_PARAM_VALUES';

export type FirmwareLibraryFileType = 0 | 1 | 5 | 6;

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
  | 'WAIT_INFORM_EVENT'
  | 'WAIT_REBOOT_COMPLETE';

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
  firmwareFileType?: FirmwareLibraryFileType;
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
  isKeepConfig?: boolean;
  status: TransferTaskStatus;
  result?: TransferTaskResult;
  progress: number;
  totalCount: number;
  successCount: number;
  failCount: number;
  // currentStep 仅升级 / 回滚类有意义；备份 / 日志采集 / 配置恢复（LogCollect 类）
  // 后端返回空串，前端 transfer 列表不再展示此列（详见 backup-display-fix-20260520.md F1）。
  currentStep?: TransferStepId | '';
  executionMode: TransferExecutionMode;
  createUser: string;
  createdAt: string;
  scheduledAt?: string;
  operatorScope: string;
}

export type UnifiedFileTransferDeviceStatus =
  | 'pending'
  | 'downloading'   // 升级 / 回滚类 Download RPC，CPE 正在从 ACS 拉文件
  | 'uploading'     // 备份 / 日志采集类 Upload RPC，已派发 Upload + 等 UploadResponse + CPE PUT 文件中
  | 'awaiting_tc'   // 备份 / 日志采集类 Upload RPC，文件已落 ACS，等 CPE 主动发 TransferComplete
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
  /** OUTPUT 文件类（备份 / 日志采集 / 配置恢复）的目标文件名，{task_id8}/{sn} 已渲染。
   *  升级 / 回滚类该字段不存在；详见 backup-display-fix-20260520.md B3。 */
  targetFile?: string;
  /** CPE 上传完成、ACS 落 MinIO、backup_restore_file 元数据就绪后由后端生成的
   *  1h 有效 presigned GET URL；未到位时为空，UI 仅灰显文件名不可点击。 */
  downloadUrl?: string;
  status: UnifiedFileTransferDeviceStatus;
  result?: TransferTaskResult;
  progress: number;
  lastReportAt: string;
  operatorScope: string;
  failureReason?: string;
  /** 设备失败的原始详情（FaultCode + FaultString 拼接），用于失败信息列 Tooltip 展示设备厂商原文。 */
  failureDetail?: string;
}

export interface CreateUnifiedFileTransferTaskInput {
  taskName: string;
  typeCode: string;
  productType?: string;
  firmwareId?: string;
  isKeepConfig?: boolean;
  deviceIds?: string[];
  deviceCount: number;
  executionMode: TransferExecutionMode;
  /**
   * ISO 8601 字符串。仅 executionMode='scheduled' 时由前端 DatePicker 提交；其它模式忽略。
   * 后端 ufte CreateTaskRequest.ScheduledAt 同名字段；time.Parse(RFC3339) 解析。
   */
  scheduledAt?: string;
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

export interface UpdateUnifiedFileTransferTaskTypeInput extends CreateUnifiedFileTransferTypeInput {
  typeCode: string;
}