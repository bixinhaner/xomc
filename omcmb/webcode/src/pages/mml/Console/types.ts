// MML 控制台 V2 —— 页面本地类型定义
//
// V2 改版（设计见 docs/design/mml-console-redesign-20260603.md）以「结果表格」为主舞台，
// 设备/命令选择收纳进顶部选择条 + 弹框。当前阶段为前端 mock 实现，类型独立于真实
// MMLCommand，待后端 results-schema / export 端点就绪后再桥接。

import type { MMLOperationType } from '@core/types/mml';

/** 设备在线状态 */
export type DeviceStatus = 'online' | 'offline' | 'alarm';

/** 设备选择弹框中的一行设备 */
export interface DeviceItem {
  sn: string;
  productName: string;
  productClass: string;
  status: DeviceStatus;
  groupName: string;
}

/** 命令选择弹框中的一条命令 */
export interface CommandItem {
  id: string;
  groupName: string;
  commandCode: string;
  commandName: string;
  operationType: MMLOperationType;
  description: string;
  /** 该命令涉及的参数路径（LST 查询列 / MOD 写入项的来源） */
  paramPaths: CommandParamPath[];
  /**
   * ADD/RMV 的目标对象路径（含父级 `.{i}.` 占位符）；增/删对象时实例选择器按其
   * `.{i}.` 个数渲染。来自命令的 target_object（GroupTreeCommand）。
   */
  targetObject?: string;
  /**
   * 是否管理员自定义命令（mml_custom_command）。自定义命令无 mml_commands.id，
   * 结构化执行端点强制要 command_id，故标记后在执行时统一走 legacy 裸路径通道
   * （POST /mml/execute，按 paramPaths + 用户填值下发）。
   */
  isCustom?: boolean;
}

/** 命令绑定的参数路径 */
export interface CommandParamPath {
  /** 命令上下文参数码（如 USER_LABEL），脚本 MOD 参数和后端 formValues 均用它索引。 */
  mmlCode?: string;
  /** TR-069 标准路径（可能含 `.{i}.` 实例占位符） */
  path: string;
  /** 展示用短标签（结果表格列头） */
  label: string;
  /** 是否可写（MOD/ADD 时可填值） */
  writable: boolean;
  /** 该 path 末级是否为多实例对象（standard_params.entry_type='object'） */
  isObject: boolean;
	/** 当前参数模型 min/max；缺失时由后端回退 standard_params。 */
  minValue?: number;
  maxValue?: number;
  valueType?: string;
  defaultValue?: string;
  validationPattern?: string;
  enumOptions?: Array<{ value: string; label: string }>;
  description?: string;
  /** MML 配置中的默认勾选标记；兼容数据缺失时按 false 处理。 */
  defaultSelected?: boolean;
  /** MML 配置中的必填标记；false 时允许留空且不会下发。 */
  isRequired?: boolean;
}

/**
 * 单设备执行状态（设计 §3.11.2 读后核实四态 + 调度态）：
 * - success：写 OK 且读回核实通过 / 读类成功
 * - unverified：写 OK 但无法核实（只写/重启生效/查询失败），**不判失败**
 * - mismatch：写 OK 但读回值与预期不符（响应成功实际未生效）
 * - failed：RPC 本身失败
 */
export type ExecStatus = 'pending' | 'running' | 'success' | 'unverified' | 'mismatch' | 'failed';

/** unverified（已下发·未核实）的原因（设计 §3.11.2，列表须明确提示）。 */
export type UnverifiedReason = 'write-only' | 'reboot-required' | 'query-failed';

/** 写类命令读后核实的逐 path 对比项（预期下发值 vs 读回实际值）。 */
export interface VerifyItem {
  path: string;
  label: string;
  /** 预期下发值（MOD 为目标值；ADD/RMV 为语义描述） */
  expected: string;
  /** 读回实际值；不可读（only-write/reboot）时为空 */
  actual: string;
  /** 是否核实一致 */
  matched: boolean;
}

/** 逐 PATH 模式下单设备一个 path 的子任务（详情页展示，父任务 = deviceTaskId）。 */
export interface PathTask {
  pathIndex: number;
  path: string;
  /** 子任务 ID（逐 PATH 真实落地后 = device_tasks.id；mock 为伪 UUID） */
  subTaskId: string;
  status: ExecStatus;
  dispatchedAt: string;
  respondedAt: string;
  value: string;
  /** #196：操作类型 MOD（下发）/ LST（回读），用于「PATH 列表」前后对比；纯逐 PATH 场景不填 */
  opType?: 'MOD' | 'LST' | string;
}

/** 结果表格的一行（= 一台设备） */
export interface ResultRow {
  planLineNo?: number;
  planOrder?: number;
  planRawLine?: string;
  commandCode?: string;
  commandName?: string;
  deviceSn: string;
  /** 设备任务 ID（= device_tasks.id；逐 PATH 时为该设备父任务 ID） */
  deviceTaskId: string;
  status: ExecStatus;
  /** path -> 值（LST 查询结果 / 写类读回值），失败/未核实时为空 */
  cells: Record<string, string>;
  /** 失败故障码（RPC 失败时填充） */
  faultCode?: string;
  /** unverified 原因（status==='unverified' 时） */
  unverifiedReason?: UnverifiedReason;
  /** 写类命令读后核实对比（success/mismatch 时填充） */
  verify?: VerifyItem[];
  /** 逐 PATH 子任务（详情页展示） */
  pathTasks?: PathTask[];
  /** RPC 任务下发时间 HH:mm:ss（= device_tasks.sent_at） */
  dispatchedAt?: string;
  /** RPC 任务执行响应时间 HH:mm:ss（= device_tasks.completed_at） */
  respondedAt?: string;
  /** 原始报文（SSE 文本流备查；MOD 回读复合时 = MOD 下发响应报文） */
  raw: string;
  /** #196：MOD 回读复合时的回读 LST 响应报文（GetParameterValuesResponse），用于详情页第二个报文页签 */
  readbackRaw?: string;
  /** 耗时（ms） */
  elapsedMs: number;
}

/** 结果表格动态列定义（来自命令的 paramPaths） */
export interface ResultColumn {
  key: string;
  label: string;
  path: string;
  /**
   * 叶子查询因空实例选择器折叠为对象 GPV 时，保留用户原始勾选的标准 Path 模板。
   * 对象响应动态展开只接纳匹配这些模板的实例化叶子；缺省表示显式对象查询，展示全部后代。
   */
  selectedPathTemplates?: string[];
}

/** 执行模式 */
export type ExecMode = 'whole' | 'single-path';

/** 左操作区模式：标准参数（结构化）/ 参数路径指定（裸路径专家）。 */
export type OperationMode = 'standard' | 'raw';

/** 裸路径模式的一行（path + value，value 仅 MOD 使用）。 */
export interface RawPathRow {
  id: number;
  path: string;
  value: string;
}

/** 裸路径模式编辑态。 */
export interface RawPathPayload {
  operationType: MMLOperationType;
  rows: RawPathRow[];
}

/** 执行请求（标准 / 裸路径两模式的判别联合）。 */
export type ExecRequest =
  | {
      mode: 'standard';
      /** 勾选的参数路径（= tr069 标准路径，结构化执行的 paths） */
      checkedPaths: string[];
      /** MOD/ADD 写入值（key = 路径）；LST/RMV 不消费 */
      values?: Record<string, string>;
      /** RMV 删除的实例号 */
      instance?: number;
      /** 父级 `.{i}.` 实例选择器（key=i01/i02…；查询可空，写类默认 1） */
      instanceSelectors?: Record<string, string>;
      /** 下发方式：whole=整体一条 RPC；single-path=逐 PATH 每 path 一条 RPC（成败独立） */
      execMode?: ExecMode;
    }
  | {
      mode: 'raw';
      operationType: MMLOperationType;
      rows: RawPathRow[];
      execMode?: ExecMode;
    };

/** 一次执行的元信息（驱动结果表格列语义 + 「查看」详情的任务信息区）。 */
export interface ExecMeta {
  operationType: MMLOperationType;
  /** 是否「读」类操作（true→参数值矩阵，false→状态+故障矩阵）。 */
  read: boolean;
  /** 原始报文头部标识 / 命令码。 */
  label: string;
  /** 标准模式的命令名（裸路径模式为空）。 */
  commandName?: string;
}

/** 命令记录的执行态：running=已下发执行中(点击执行即插入)，done=已收口(结果就位)。 */
export type ExecRecordStatus = 'running' | 'done';

/** 一次执行命令记录(含结果快照,供「命令记录」列表点击回看)。 */
export interface ExecRecord {
  id: string;
  /** 记录级执行态：点击执行先以 running 插入，完成后原地更新为 done。 */
  status: ExecRecordStatus;
  /** 命令 ID（= mml_tasks.id，每次批量执行全局唯一；mock 用 crypto.randomUUID 模拟） */
  commandId: string;
  /** 执行时间 HH:mm:ss */
  time: string;
  /** 执行的命令名称(标准模式命令名 / 裸路径模式 "裸路径 LST" 等) */
  commandName: string;
  operationType: MMLOperationType;
  /** 本次执行的设备数 */
  deviceCount: number;
  execMeta: ExecMeta;
  columns: ResultColumn[];
  rows: ResultRow[];
  /** 该记录中命令名/结果行对应的界面语言；语言切换后需要重新拉结果。 */
  locale?: string;
  /**
   * #196：MOD 下发值 path→value。MOD 命令后端自动追加回读 LST（compound）；跨刷新惰性补结果行时
   * 据此走 buildMODReadbackRows 关联「下发 vs 回读」（而非逐 PATH 合并）。非 MOD 复合为空。
   */
  setValues?: Record<string, string>;
}

/** 导出格式 */
export type ExportFormat = 'csv' | 'xlsx' | 'json';
