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
  valueConstraint: Record<string, unknown>;
}

export interface MMLCommand {
  id: string;
  commandName: string;
  commandCode: string;
  category: string;
  description: string;
  params: MMLParam[];
  productTypes: string[];
  // Extended fields aligned with backend model
  operationType?: MMLOperationType;
  paramPaths?: ParamPath[];
  supportedOperations?: string[];
  helpDoc?: string;
  notes?: string;
  paramRefs?: MMLParamRef[];
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
  productTypes: string[];
  creator: string;
  createdAt: string;
  updatedAt: string;
}
