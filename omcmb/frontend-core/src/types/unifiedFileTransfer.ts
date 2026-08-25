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
  /** #492 适用产品：产品英文名列表（引用产品管理 product_name）。非空时设备匹配走产品目录精确匹配，
   *  制式由所选产品派生；空则回退 platformScope。后端始终返回数组，老 mock 可缺省故标可选。 */
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
  fileType: string;
  firmwareId?: string;
  targetVersion?: string;
  productType?: string;
  /** 产品英文名（后端由 productClass 解析，优先展示；为空时前端回退 productType）。 */
  productName?: string;
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
  /** 任务真正开始下发的时间（首次进入 in_progress 时后端 PG repo 写入 upgrade_tasks.started_at）。未开始时后端 omitempty 不返回。 */
  startedAt?: string;
  /** 任务到达终态（ended：成功/失败/终止）时后端 PG repo 写入 upgrade_tasks.ended_at。未结束时后端 omitempty 不返回。 */
  endedAt?: string;
  scheduledAt?: string;
  operatorScope: string;
}

export type UnifiedFileTransferDeviceStatus =
  | 'pending'
  | 'downloading'   // 升级类 Download RPC，CPE 正在从 ACS 拉文件
  | 'rollback_checking' // 版本回退前置 GPV，检查设备是否允许回退
  | 'rolling_back'  // 版本回退 SPV 已下发，等待设备回退/重启完成
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
  /** #492 设备所属产品英文名（productClass 经 ProductRegistry 解析）。候选/设备列表展示产品名取代裸 productClass；孤儿设备为空。 */
  productName?: string;
  currentVersion: string;
  targetVersion: string;
  /** OUTPUT 文件类（备份 / 日志采集 / 配置恢复）的目标文件名，{task_id8}/{sn} 已渲染。
   *  升级 / 回滚类该字段不存在；详见 backup-display-fix-20260520.md B3。 */
  targetFile?: string;
  /** CPE 上传完成、ACS 落 MinIO、backup_restore_file 元数据就绪后由后端生成的
   *  1h 有效 presigned GET URL；未到位时为空，UI 仅灰显文件名不可点击。 */
  downloadUrl?: string;
  /** targetFile 对应的日志文件已被配额清理；文件名保留展示，但不可下载。 */
  fileDeleted?: boolean;
  status: UnifiedFileTransferDeviceStatus;
  result?: TransferTaskResult;
  progress: number;
  /** 子任务首次进入执行态（downloading/uploading/rebooting/verifying）时由后端 PG repo
   *  写入 upgrade_sub_tasks.started_at；未开始时后端 omitempty 不返回。issue #655 起前端
   *  设备列表「开始时间」列展示。 */
  startedAt?: string;
  /** 子任务到达终态（completed/failed/terminated）时由后端 PG repo 写入
   *  upgrade_sub_tasks.completed_at；未结束时后端 omitempty 不返回。issue #655 起前端
   *  设备列表「结束时间」列展示。 */
  endedAt?: string;
  /** 旧字段：仅在子任务到达 ended 终态时填 sub_task.updated_at。issue #655 起前端
   *  设备列表已改用 startedAt + endedAt 展示，本字段仍保留兼容 CSV / 北向 API。 */
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
  /** 任务内设备并发执行数，仅升级类（固件下载）任务生效；不传由后端兜底默认 20。 */
  concurrency?: number;
  deviceIds?: string[];
  deviceCount: number;
  executionMode: TransferExecutionMode;
  /**
   * ISO 8601 字符串。仅 executionMode='scheduled' 时由前端 DatePicker 提交；其它模式忽略。
   * 后端 ufte CreateTaskRequest.ScheduledAt 同名字段；time.Parse(RFC3339) 解析。
   */
  scheduledAt?: string;
  note?: string;
  /** 核心网（ims_core）任务：参数类型标签 P1~P12，区分 ImsCore_Parameters_Type 具体参数配置。 */
  paramType?: string;
  /** IMS_PARAM_DISTRIBUTE：参数文件库目标文件 ID（ims_param_files.id）。 */
  fileId?: string;
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
  /** #492 适用产品（产品英文名列表）。模板编辑改为产品名多选后提交此字段；留空则沿用 platformScope。 */
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

export interface UpdateUnifiedFileTransferTaskTypeInput extends CreateUnifiedFileTransferTypeInput {
  typeCode: string;
}
