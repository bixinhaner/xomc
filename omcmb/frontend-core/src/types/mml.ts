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
  /** 命令上下文参数码（mml_command_sub_fields.mml_code），MOD 脚本参数 key 使用它。 */
  mmlCode?: string;
  valueType?: string;
  defaultValue?: string;
  description?: string;
}

export type MMLOperationType = 'LST' | 'MOD' | 'ADD' | 'RMV' | 'DSP' | 'ACT' | 'DEA' | 'RST' | 'CLR' | 'UPG';

/**
 * MMLParamRef represents a parameter reference bound to a command.
 * Maps to backend MMLParamRef sourced from standard_params (the legacy
 * mml_params table has been retired).
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
  /** 所属 mml_command_groups.id，nullable */
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
  productClasses?: string[];
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
  /** device_tasks.id —— 子任务 ID（每条 RPC 一个；详情页「PATH 列表」复制用） */
  deviceTaskId?: string;
  /** device_tasks.command_index —— 逐 PATH 模式下定位该条结果属于哪个 path（命令序号） */
  commandIndex?: number;
  /** device_bound 计划行追溯字段。 */
  planLineNo?: number;
  planDeviceSn?: string;
  planOrder?: number;
  planRawLine?: string;
  commandCode?: string;
  operationType?: string;
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

export type MMLTaskExecuteMode = 'common' | 'device_bound';

/**
 * 任务 commands JSONB 数组中每条命令的"明细"形态（含 op_type 与勾选 path）。
 *
 * 后端 mml_tasks.commands 是 `Array<{command_code, operation_type, param_paths,
 * param_values, ...}>` 的 JSONB；旧字段 MMLTask.commands 把每条压成 command_code 单
 * 字符串够任务列表用，但任务记录"查看"页需要展示用户当时勾选了哪些 path —— 新增
 * 这个并行字段保留完整明细，老消费者继续读 commands: string[] 不受影响。
 */
export interface MMLTaskCommandDetail {
  commandCode: string;
  /** 友好命令名（后端 GetTask 按 command_code 注入；RAW 命令为 undefined）。 */
  commandName?: string;
  operationType?: MMLOperationType | string;
  paramPaths?: string[];
  /** MOD 操作的下发值；与 paramPaths 同序对应 */
  paramValues?: unknown[];
  /**
   * MOD 下发值的 path→value 映射（裸路径/自定义命令通道存于此，而非 paramValues 数组）；
   * 取下发值时优先 paramValues[i]，缺则回退 parameters[path]。
   */
  parameters?: Record<string, unknown>;
  planLineNo?: number;
  planDeviceSn?: string;
  planOrder?: number;
  planRawLine?: string;
}

export interface MMLTaskCommandInput {
  commandCode: string;
  operationType?: MMLOperationType | string;
  paramPaths?: string[];
  parameters?: Record<string, unknown>;
}

export interface MMLTaskPlanItem {
  lineNo: number;
  deviceSn: string;
  order: number;
  rawLine?: string;
  command: MMLTaskCommandInput;
}

export interface MMLTaskPlanStats {
  totalPlanItems: number;
  totalDevices: number;
}

export type MMLTaskCreateInput =
  Partial<Omit<MMLTask, 'id' | 'status' | 'results' | 'createdAt' | 'updatedAt' | 'commands'>> &
  Pick<MMLTask, 'taskName'> & {
    deviceSns?: string[];
    commands?: Array<string | MMLTaskCommandInput | MMLTaskCommandDetail>;
    executeMode?: MMLTaskExecuteMode;
    planItems?: MMLTaskPlanItem[];
  };

export interface MMLTask {
  id: string;
  taskName: string;
  scriptId?: string;
  deviceSns: string[];
  commands: string[];
  executeMode?: MMLTaskExecuteMode;
  planItems?: MMLTaskPlanItem[];
  planStats?: MMLTaskPlanStats;
  /** 命令明细（含 op_type / param_paths）；老接口可能不返回，UI 需做 fallback */
  commandsDetail?: MMLTaskCommandDetail[];
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

  /**
   * T-0168: 路径翻译审计元数据（持久化到 mml_tasks 4 列）。
   *
   * productResolved=false 表示设备 productClass 未匹配任何 product（激进路线下
   * 任务仍正常 fanout，所有 path 走 orphan_passthrough 原路径下发，触发
   * Prometheus 告警 mml_path_translation_orphan_total）。前端展开行顶部应显示
   * 橙色 Banner 提示运维补登记 product_class_patterns。
   */
  productResolved?: boolean;
  matchedProductId?: string;
  matchedProductClass?: string;
  /** 任务级翻译来源汇总：discovered/default/passthrough/orphan_passthrough/mixed */
  pathTranslationSource?: PathTranslationSource;
}

/** T-0168: 路径翻译来源枚举（per-path 与 task 级共用）。 */
export type PathTranslationSource =
  | 'discovered'
  | 'default'
  | 'passthrough'
  | 'orphan_passthrough'
  | 'mixed';

/**
 * T-0168: GET /mml/tasks/{id}/results 响应 stats 字段的单条路径翻译视图。
 *
 * 前端"任务记录列表行展开"读 ListResponse.stats.path_translations[] 渲染表格：
 *   standardPath | privatePath | 来源 Tag | 状态 Tag
 *
 * translated=false 时（passthrough / orphan_passthrough）privatePath === standardPath。
 */
export interface MMLPathTranslationView {
  standardPath: string;
  privatePath: string;
  translationSource: PathTranslationSource;
  translated: boolean;
}

/**
 * T-0168: GET /mml/tasks/{id}/results 响应 stats 字段结构。
 *
 * 复用 ListResponse<T>.stats interface{} 字段携带任务级翻译元数据 +
 * per-path 翻译详情，避免前端发两次请求。
 */
export interface MMLTaskResultsStats {
  pathTranslations?: MMLPathTranslationView[];
  productResolved: boolean;
  matchedProductClass?: string;
  pathTranslationSource?: PathTranslationSource;
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
