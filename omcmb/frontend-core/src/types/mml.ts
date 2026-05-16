export type MMLParamType = 'string' | 'number' | 'boolean' | 'enum' | 'range' | 'ipAddress' | 'list' | 'unsignedInt';

export interface MMLParam {
  name: string;
  type: MMLParamType;
  required: boolean;
  defaultValue?: string | number | boolean;
  description: string;
  options?: Array<{ label: string; value: string | number }>;
  minValue?: number;
  maxValue?: number;
  pattern?: string;
  // Extended fields aligned with backend ParamMeta
  suggestedValue?: string | number | boolean;
  unit?: string;
  restartRequired?: boolean;
  helpText?: string;
  order?: number;
  enumValues?: string[];
}

// Represents a TR-069 parameter path bound to a command
export interface ParamPath {
  path: string;
  label: string;
  writable: boolean;
}

export type MMLOperationType = 'LST' | 'MOD' | 'ADD' | 'RMV' | 'DSP' | 'ACT' | 'DEA' | 'RST' | 'CLR' | 'UPG';

/**
 * MMLParamRef represents a parameter reference bound to a command.
 * Maps to backend MMLParamRef (mml_params table).
 */
export interface MMLParamRef {
  id: string;
  paramCode: string;
  paramNameZh: string;
  tr069Path: string;
  valueType: 'string' | 'number' | 'boolean' | 'enum';
  isWritable: boolean;
  /** 参数默认值，用于 MOD 操作输入框 placeholder 提示。空字符串表示无默认。 */
  defaultValue?: string;
  /** JavaScript 正则字符串（含 / / 包裹符），用作前端格式校验提示。 */
  jsRegex?: string;
  valueConstraint: Record<string, unknown>;
}

export interface MMLCommand {
  id: string;
  commandName: string;
  commandCode: string;
  category: string;
  description: string;
  params: MMLParam[];
  // Extended fields aligned with backend model (migration 000090 重建后的 schema)
  operationType?: MMLOperationType;
  /** 派生自 backend.target_paths（string[] → ParamPath[]）；writable 由 operationType 推断 */
  paramPaths?: ParamPath[];
  /** 派生自 operationType 单元素数组，保留是为兼容 ParamPathPanel 旧逻辑 */
  supportedOperations?: string[];
  helpDoc?: string;
  notes?: string;
  paramRefs?: MMLParamRef[];
  // 新增字段（standard-model 重建后由 mmlstandardloader 写入）
  /** ADD/RMV 操作的目标对象路径，nullable */
  targetObject?: string;
  /** 所属 mml_param_groups.id，nullable */
  groupId?: string;
  /** i18n 命令名 {"zh-CN":"...", "en-US":"..."} */
  commandNameI18n?: Record<string, string>;
  /** 危险命令二次确认标志 */
  requireConfirm?: boolean;
  /** i18n 确认提示文案 */
  confirmMsgI18n?: Record<string, string>;
  /**
   * @deprecated migration 000090 后 mml_commands.product_types 列已下线；
   * 字段保留为可选仅供页面兜底渲染（永远为 undefined / []）。
   */
  productTypes?: string[];
}

export interface MMLResult {
  success: boolean;
  rawOutput: string;
  parsedData?: Record<string, unknown>;
  executionTime: number;
  timestamp: string;
}

export type DeviceResultStatus = 'completed' | 'running' | 'pending';

export interface DeviceTaskResultItem {
  deviceSn: string;
  deviceName?: string;
  mmlScript?: string;
  status?: DeviceResultStatus;
  result: MMLResult;
  failReason?: string;
  startedAt?: string;
  finishedAt?: string;
}

// mml_scripts.status —— 数据库层面无 CHECK 约束，实际取值取决于脚本执行生命周期：
//   active / archived 为"定义态"；pending / running / paused / completed /
//   failed / cancelled 为"执行态"。UI 需按值渲染对应 Tag。
export type MMLScriptStatus =
  | 'active'
  | 'archived'
  | 'pending'
  | 'running'
  | 'paused'
  | 'completed'
  | 'failed'
  | 'cancelled';

export type MMLScriptType = 'manual' | 'batch';

// mml_scripts.result 的聚合结果标识（前端显示用）。
export type MMLScriptResult = 'success' | 'partial' | 'failed';

export interface MMLScript {
  id: string;
  scriptName: string;
  description: string;
  content: string;
  creator: string;
  createTime: string;
  updateTime: string;
  tags: string[];
  // New fields from mml_scripts restructure
  status: MMLScriptStatus;
  startTime?: string;
  endTime?: string;
  type: MMLScriptType;
  progress: number;
  result?: Record<string, unknown>;
  // P1 扩展：最近一次执行态。每次执行详情查 mml_tasks。
  lastRunStatus?: string;
  lastRunAt?: string;
}

export type MMLTaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'paused' | 'cancelled';

export type MMLExecuteType = 'immediate' | 'scheduled' | 'periodic' | 'suspended';

export type MMLTaskResult = 'success' | 'partial' | 'failed';

export interface MMLTask {
  id: string;
  taskName: string;
  scriptId?: string;
  deviceSns: string[];
  commands: string[];
  status: MMLTaskStatus;
  results: Array<{ deviceSn: string; result: MMLResult }>;
  createdAt: string;
  updatedAt: string;
  creator: string;

  // Scheduling
  executeType: MMLExecuteType;
  scheduledAt?: string;
  periodStart?: string;
  periodEnd?: string;
  periodTime?: string;

  // Retry strategy
  offlineRetry: boolean;
  offlineRetryWait: number;
  failedRetry: boolean;
  failedRetryCount: number;
  failedRetryInterval: number;

  // Execution timestamps
  startedAt?: string;
  finishedAt?: string;

  // Statistics
  totalDevices: number;
  successCount: number;
  failedCount: number;
  result?: MMLTaskResult;

  // Scheduler fields (P2/P3, docs/design/mml-task-flow-design-20260424.md)
  // nextTriggerAt: 下次触发时刻（scheduled 一次性；periodic 滚动更新）
  // parentTaskId: periodic 子实例指向模板；模板行为 null
  nextTriggerAt?: string;
  parentTaskId?: string;

  /**
   * Sprint B-6: standard-model 重建后老 mml_custom_command 引用的 command_code
   * 在 mml_commands 表中已不存在；后端把这些孤儿码降级为 `orphan=true` 透传，
   * task 仍创建成功但 Fanouter 会跳过下发。前端按此列表给用户 toast 提示
   * "命令已下线"，避免任务列表显示"已提交但 0 设备成功"让人困惑。
   */
  orphanCommandCodes?: string[];

  /**
   * 整改方案 Stage 3 — 路径翻译警告（standardPath ↔ privatePath）。
   * 由后端 GetTask 聚合 device_tasks.has_path_translation_miss 计数后填充；
   * undefined 表示无 miss 或后端未启用聚合（兼容旧版）。
   *
   * AnyMiss=true 时前端任务详情页显示警告 Tag："{deviceCount} 个设备共 {pathCount}
   * 条 path 未翻译，用 standardPath 兜底下发"。
   */
  pathTranslationWarning?: {
    anyMiss: boolean;
    deviceCount: number;
    pathCount: number;
  };
}

/**
 * MMLCustomCommand represents a user-defined custom command.
 * Renamed from MMLTemplate; maps to mml_custom_command table.
 * Backend route paths remain /mml/templates for backward compat.
 */
export interface MMLCustomCommand {
  id: string;
  commandName: string;
  commandCode: string;
  operationType: MMLOperationType;
  commandScope: 'private' | 'public';
  categoryGroup: string;
  parameters: Record<string, string | number | boolean>;
  paramPaths: string[];
  description: string;
  creator: string;
  createdAt: string;
  updatedAt: string;
}
