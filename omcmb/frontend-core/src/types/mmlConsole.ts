/**
 * T-0123-P2-a Console 业务层类型 — frontend-core
 *
 * PRD: docs/design/mml-restore-old-interaction-plan-20260514.md §7.4
 * 后端契约：omcgo/internal/mml/mml_renderer.go Statement struct json tags
 *
 * 与 types/mml.ts 区别：mml.ts 包含历史 MML 命令/脚本/任务等模型；
 * 本文件专注 Console 三栏交互的运行期状态（statements、sub_field 元数据、
 * 命令树节点等）。后续 P2-b/c/d 组件层消费。
 */

import type { MMLOperationType } from './mml';

// ============================================================
// 后端 snake_case 类型（与 Go 后端 json tags 一一对应）
// ============================================================

/** 后端 `mml_renderer.go` Statement struct 的 wire 格式。 */
export interface BackendStatement {
  command_id?: string;                 // *uuid.UUID → string|undefined
  logical_code: string;
  operation_type: string;              // LST / MOD / ADD / RMV
  selected_sub_field_ids?: string[];   // LST 用：已选 sub_field id 列表
  selected_mml_codes?: string[];       // parser 内部 — UI round-trip 可见
  values?: Record<string, string>;     // MOD/ADD 用：mml_code → value 字符串
  rmv_instance_index?: number;
  unknown_codes?: string[];            // lookup 后未命中的 mml_code（FE toast）
}

/** GET /mml/group-tree 响应的子节点（递归结构）。后端 Go json tag 见 `omcgo/internal/mml/group_tree_repository.go GroupTreeNode`。 */
export interface BackendGroupTreeNode {
  id: string;
  /** 后端 json tag = "code"（GroupCode 在 Go struct） */
  code: string;
  /** 当前 lang 派生显示名 — "设备信息(LST DEVICE_INFO)" */
  name: string;
  /** 完整 i18n map (zh-CN/en-US) */
  name_i18n?: Record<string, string>;
  path: string;
  display_order: number;
  /**
   * CMCC TD-LTE v2.3 章节码（SA/SB/SC/.../SR）。
   * 老 catalog 行返回空串或缺失字段。前端不渲染章节为节点，但作为主排序键
   * 让对象级 group 跨章节按 SA→SB→SC 顺序排列。
   */
  chapter_code?: string;
  /** 'standard' | 'admin' — 来源标记 */
  source?: string;
  /** catalog_protected=true 的节点不可由 admin 修改/删除 */
  catalog_protected?: boolean;
  commands: BackendGroupTreeCommand[];
  children: BackendGroupTreeNode[];
}

export interface BackendGroupTreeCommand {
  id: string;
  command_code: string;
  logical_code: string;
  logical_name?: string;
  logical_name_i18n?: Record<string, string>;
  operation_type: string;
  display_name: string;
  rpc_method?: string;
  target_object?: string;
  target_paths?: string[];
  require_confirm: boolean;
  source?: string;
  catalog_protected?: boolean;
  /**
   * R-4.1.1: 每层 {i} 的取值范围 metadata（v2 catalog Loader 写入；v1 path 缺失）。
   * 后端 GroupTreeCommand 用 `json.RawMessage` 透传 JSONB 列原文，wire 形态：
   *   [{ layer, rangeExpr, rangeMin, rangeMax, dynamic, nSource?, description? }]
   * 内层 keys 已是 camelCase（与 Python 解析器对齐），axios 拦截器对其幂等。
   */
  instance_range_meta?: InstanceRange[];

  /**
   * T-0172 catalog 按 product_class 过滤的标注（仅在 group-tree 请求带
   * product_class 入参时填充；缺省请求不返回）。
   * - supported_path_count: 该 paramModel 下 is_supported=true 的 path 数
   *                         （LST/MOD = supported 交集大小；ADD/RMV = 0 或 1）
   * - unsupported_paths: 当前产品不支持的具体 path 列表（保留供调试/日志，前端不渲染）
   * - product_resolved: false = 孤儿设备（productClass 未匹配产品）
   */
  supported_path_count?: number;
  unsupported_paths?: string[];
  product_resolved?: boolean;
}

/** GET /mml/commands/:id/sub-fields 响应的单条 sub-field。 */
export interface BackendSubField {
  id: string;
  command_id: string;
  param_id: string;
  mml_code: string;
  label: string;                       // lang 派生
  label_i18n: Record<string, string>;
  tr069_path: string;
  value_type: string;
  access_type: string;                 // READ_ONLY / READ_WRITE
  is_object: boolean;
  supports_add: boolean;
  supports_delete: boolean;
  change_applies: string;              // OnReboot / Immediate / OnSession
  constraint_text: string;             // lang 派生
  constraint_text_i18n: Record<string, string>;
  default_value?: string;
  js_regex?: string;
  validation_pattern?: string;
  enum_options?: Array<{ value: string; label: string }>;
	/** 当前参数模型 min/max；缺失时由后端回退 standard_params。 */
  min_value?: number | null;
  max_value?: number | null;
  default_selected: boolean;
  is_required: boolean;
  sort_order: number;
  /**
   * T-0183: param_mappings.is_supported 派生 — 该 path 在当前 paramModel 是否支持。
   * false 表示 sweep-paths 探测 / iteration guide 标过 supported="false"。
   * console 默认不勾选 false 行(允许用户手动勾选作为兜底)。
   * 缺映射兜底为 true(默认显示并可选)。
   */
  is_supported?: boolean;
}

/** POST /mml/parse 响应中的 parse_error 单条。 */
export interface BackendParseError {
  statement_index: number;
  raw: string;
  reason: string;
}

/**
 * R-8.5 GET /mml/console/command-compatibility 响应。
 * 后端 Go struct: omcgo/internal/mml/command_compatibility.go CommandCompatibilityResult。
 * Axios 拦截器自动 snake → camel，前端消费见 CommandCompatibility（下方）。
 */
export interface BackendCommandCompatibility {
  product_class: string;
  product_id: string;
  param_model_id?: string;       // 空 UUID 时 omitempty 缺失
  unsupported_command_ids: string[];
}

// ============================================================
// 前端 camelCase 类型（消费侧）
// ============================================================

/**
 * 命令树节点（递归）。
 * `commands` 是该 group 直接挂的命令（不含子 group 内的命令）。
 * `children` 是子 group 节点。
 */
export interface GroupTreeNode {
  id: string;
  groupCode: string;
  path: string;                        // ltree path "BSC_CONFIGURATION.BASIC_INFO"
  displayName: string;                 // "设备信息(LST DEVICE_INFO)"
  displayNameI18n?: Record<string, string>;  // 完整 i18n
  displayOrder: number;
  /**
   * CMCC TD-LTE v2.3 章节码（SA/SB/...）。空串 / undefined 视为"未分章"，
   * CommandTree 排序时排末位。前端不渲染章节为节点，仅作主排序键。
   */
  chapterCode?: string;
  /** 'standard' | 'admin' — 来源标记，admin UI 用于显示来源 */
  source?: 'standard' | 'admin' | string;
  /** catalog_protected=true 时 admin 不可改/删（PRD §7.3 元数据差异化） */
  catalogProtected?: boolean;
  commands: GroupTreeCommand[];
  children: GroupTreeNode[];
}

export interface GroupTreeCommand {
  id: string;
  commandCode: string;
  logicalCode: string;
  logicalName?: string;
  logicalNameI18n?: Record<string, string>;
  operationType: MMLOperationType;
  displayName: string;
  rpcMethod?: string;
  targetObject?: string;
  targetPaths?: string[];
  requireConfirm: boolean;
  source?: 'standard' | 'admin' | string;
  /** catalog_protected=true 时 admin 不可改/删（PRD §7.3） */
  catalogProtected?: boolean;
  /**
   * R-4.1.1: 每层 {i} 占位符的取值范围 metadata（v2 catalog 写入；v1 catalog 缺失）。
   * Console Control Panel 的 InstanceArityInput 用此校验输入；
   * 与 cmd 共生但与 Statement 解耦 — 后续 statement 构造时透传到 Statement.instanceRangeMeta。
   */
  instanceRangeMeta?: InstanceRange[];

  /**
   * T-0172 catalog 按 product_class 过滤的标注。仅在 useGroupTree 传入
   * productClass 时填充；缺省请求不返回。
   * - supportedPathCount: 该 paramModel 下 is_supported=true 的 path 数
   *                       （LST/MOD = supported 交集大小；ADD/RMV = 0 或 1）
   * - unsupportedPaths: 当前产品不支持的具体 path 列表（保留供调试/日志，前端不渲染）
   * - productResolved: false = 孤儿设备（productClass 未匹配任何产品）
   */
  supportedPathCount?: number;
  unsupportedPaths?: string[];
  productResolved?: boolean;
}

/**
 * R-4.1.1 单层 {i} 范围 metadata。
 *
 * 后端契约：mml_commands.instance_range_meta JSONB 的元素结构（json tag 已 camelCase；catalogloader 已下线）。
 * 由 Python 解析器从 cmcc-tdlte-southbound-data-model-v2.3.md 抽取，存 mml_commands.instance_range_meta JSONB。
 *
 * 字段语义：
 *   - layer: 1-based 层级（对齐 InstanceArityInput 的 iα/iβ/iγ 顺序）
 *   - rangeExpr: 规范原文（如 "1~3" / "0~N"），仅展示用
 *   - rangeMin / rangeMax: 解析后的整数边界；null 表示该方向无界 / 动态
 *   - dynamic: true 时上限由 nSource 命名的另一参数运行时决定，前端跳过上限校验
 *   - nSource: 仅 dynamic=true 时出现（驱动上限的参数名，例 PLMNListNumberOfEntries）
 *   - description: 规范原文描述（中文，spec 直接来源）
 */
export interface InstanceRange {
  layer: number;
  rangeExpr: string;
  rangeMin: number | null;
  rangeMax: number | null;
  dynamic: boolean;
  nSource?: string;
  description?: string;
}

/**
 * sub-field 元数据（命令上下文）。
 * 用于 LST Checklist / MOD InputList / ADD InputList 的渲染依据。
 * access_type / change_applies / is_object 等元数据驱动 UI 差异化（PRD §7.3）。
 */
export interface SubFieldDef {
  id: string;
  commandId: string;
  paramId: string;
  mmlCode: string;
  label: string;                       // 当前 lang 派生
  labelI18n: Record<string, string>;
  tr069Path: string;
  valueType: string;
  accessType: 'READ_ONLY' | 'READ_WRITE' | string;
  isObject: boolean;
  supportsAdd: boolean;
  supportsDelete: boolean;
  changeApplies: 'OnReboot' | 'Immediate' | 'OnSession' | string;
  constraintText: string;
  constraintTextI18n: Record<string, string>;
  defaultValue?: string;
  jsRegex?: string;
  validationPattern?: string;
  enumOptions?: Array<{ value: string; label: string }>;
	/** 当前参数模型 min/max；缺失时由后端回退 standard_params。 */
  minValue?: number;
  maxValue?: number;
  defaultSelected: boolean;
  isRequired: boolean;
  sortOrder: number;
  /**
   * TR-181 path 的中文含义说明（来自 standard_params.description，
   * 由 cmcc_tdlte_v23.json 等 spec seed 回填，详 migration 000155 + seed/000156）。
   * 可为空（非 cmcc-td-lte 来源的 standard_params 行 description 未维护）。
   * 前端 SubFieldChecklist / SubFieldInputList 行内 / Tooltip 渲染用。
   */
  description?: string;
  /**
   * T-0183: param_mappings.is_supported 派生(后端按 paramModel 维度聚合)。
   * false → console 默认不勾选(用户可手动勾选);PathPicker / SubFieldChecklist 加视觉标记。
   * undefined / true → 正常按 defaultSelected 决定。
   */
  isSupported?: boolean;
}

/**
 * R-8.5 命令兼容性响应（前端消费侧 camelCase）。
 * 由 axios 拦截器从 BackendCommandCompatibility 自动转换。
 */
export interface CommandCompatibility {
  productClass: string;
  productId: string;
  paramModelId?: string;
  unsupportedCommandIds: string[];
}

/**
 * Bundle C — 命令搜索后端响应单元。
 * 对应 GET /api/v1/mml/commands/search?q=...
 */
export interface BackendSearchCommand {
  command_id: string;
  command_code: string;
  logical_code: string;
  operation_type: string;
  display_name: string;
  logical_name: string;
  group_id: string;
  group_code: string;
  group_name: string;
  chapter_code: string;
  matched_paths: string[] | null;
  match_reasons: string[] | null;
}

/** 前端 camelCase 视图。axios 拦截器自动转换；本接口与 BackendSearchCommand 1:1。 */
export interface SearchCommand {
  commandId: string;
  commandCode: string;
  logicalCode: string;
  operationType: string;
  displayName: string;
  logicalName: string;
  groupId: string;
  groupCode: string;
  groupName: string;
  chapterCode: string;
  matchedPaths: string[];
  matchReasons: string[];
}

/**
 * Console 中单条 statement 的客户端状态。
 *
 * 注：uid 是客户端生成（crypto.randomUUID），用于 React key 与 store 内部
 * 寻址；commandId 是后端 mml_commands.id（uuid）。统计 / 提交 / round-trip 用
 * commandId，本地 UI 操作用 uid。
 */
export interface Statement {
  uid: string;                         // 客户端 nanoid 风格的 UUID
  commandId?: string;                  // 服务端解析后填；新增前可暂缺
  commandCode: string;                 // 例 "LST_DEVICE_INFO"
  logicalCode: string;                 // 例 "DEVICE_INFO"
  operationType: MMLOperationType;
  logicalNameI18n: Record<string, string>;
  /** 命令完整子字段（来自 GET /sub-fields），保留全集以便 toggle */
  subFields: SubFieldDef[];
  selectedSubFieldIds: string[];       // LST：勾选的 sub_field id
  values: Record<string, string>;      // MOD/ADD：mml_code → value
  /** RMV 单实例编号（兼容 MML 文本 parser；UI 编辑改走 rmvInstanceIndices） */
  rmvInstanceIndex?: number;
  /**
   * R-7 RMV 多实例选删（InstancePicker.checkbox/tags）。
   * 非空时优先于 rmvInstanceIndex；后端 executor 按"降序"展开为 N 条 DeleteObject，
   * 整个 statement 触发 sequential 模式（同设备串行下发），避免 CPE 索引重排错位。
   */
  rmvInstanceIndices?: number[];
  /**
   * R-4 多层 {i} 实例选择器（InstanceArityInput 写入）。
   * key 命名按 Greek 字母 iα/iβ/iγ 字典序对齐 sub_field.tr069Path 中 `.{i}.` 左到右位置；
   * 后端 executor 在编译 commands[] entry 时按 key 字典序替换 path 中的 `.{i}.` 占位符。
   * 路径 `.{i}.` 数 ≠ selectors 数 → 后端 400。
   */
  instanceSelectors?: Record<string, string>;
  unknownCodes: string[];              // parser/lookup 未命中
  /**
   * T-0130: RMV statement 用于 GPV 探测的目标对象路径（例 "Device.IP.Interface."）。
   * 在 CommandTree 构造 Statement 时从 command.targetObject 透传；InstancePicker 据此调
   * POST /ops/commands/rpc (action="get_param") 探测当前设备实例集合。
   */
  targetObject?: string;
  /**
   * R-4.1.1: 每层 {i} 占位符的取值范围 metadata。
   * CommandTree 构造 Statement 时从 cmd.instanceRangeMeta 透传，
   * RightPanel 渲染时下传 InstanceArityInput → 用户输入校验。
   * v1 catalog / 元数据缺失时为 undefined，组件兜底为"仅必填+整数格式"基本校验。
   */
  instanceRangeMeta?: InstanceRange[];
}

/** parse 错误（不阻塞 statement 显示，前端 toast 提示） */
export interface ParseError {
  statementIndex: number;
  raw: string;
  reason: string;
}

// ============================================================
// 请求 / 响应（与 P1 backend 5 endpoint 一一对应）
// ============================================================

/** POST /mml/render 请求（UI → mml_string）。 */
export interface RenderRequest {
  commandId: string;
  operationType: MMLOperationType;
  selectedSubFieldIds?: string[];
  values?: Record<string, string>;
  rmvInstanceIndex?: number;
}

/** POST /mml/parse 请求（mml_string → statements）。 */
export interface ParseRequest {
  mmlString: string;
  lang?: string;
}

/** POST /mml/parse 响应。 */
export interface ParseResponse {
  statements: Statement[];
  parseErrors: ParseError[];
}

/** POST /mml/execute-statements 请求。 */
export interface ExecuteStatementsRequest {
  statements: Statement[];
  deviceSns: string[];
  taskName?: string;
  creator?: string;
  executor?: string;
  executeType?: 'immediate' | 'scheduled' | 'periodic' | 'suspended';
}

/** P2-a 简化版本地渲染器支持的操作类型。 */
export type ConsoleSupportedOp = 'LST' | 'MOD' | 'ADD' | 'RMV';

// ============================================================
// R-9.2 结构化执行入参（CMCC TD-LTE v2.3 catalog 改造）
//
// 方案：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §6.6.1
//
// 目的：替代"前端渲染 MML 文本 → 后端解析 MML"的 round-trip 模式，
// 直接把结构化 path / value / 实例号上行到 service。后端 fanout 阶段
// 按 device.product_class 调 ParamModel Translator 把 standardPath → privatePath。
//
// 兼容期：Statement / ExecuteStatementsRequest 旧类型保留 1 个 release，
// 之后从 Console 主链路移除（ScriptTask 继续使用 MML 文本通道）。
// ============================================================

/** R-9.2: 单条结构化 statement —— 一份命令的完整执行上下文。 */
export interface StructuredStatement {
  /** mml_commands.id — 命令树叶子的唯一标识 */
  commandId: string;
  /** 命令的 op_type（在树叶上固化；不可在执行时切换） */
  operationType: ConsoleSupportedOp;
  /** 命令路径模板（如 `Device.DeviceInfo.*`），仅用于 UI 显示与回溯 */
  commandCode?: string;
  /** 用户在 SubFieldChecklist / SubFieldInputList 勾选的 standardPath 列表 */
  paths: string[];
  /** MOD / ADD 模式的 key=standardPath / value=新值；key 必须 ⊆ paths */
  values?: Record<string, string>;
  /**
   * R-4 多层 {i} 实例号填值：`{iα: '1', iβ: '2'}`；LST 允许部分留空 = partial path。
   * 与 instance_indices 互斥（一个 statement 用其一）。
   */
  instanceSelectors?: Record<string, string>;
  /** RMV 多选实例号；与 instance_selectors 互斥。 */
  instanceIndices?: number[];
}

/** R-9.2: POST /mml/console/execute-statements 结构化入参（替代旧 ExecuteStatementsRequest）。 */
export interface StructuredExecuteRequest {
  deviceSns: string[];
  statements: StructuredStatement[];
  taskName?: string;
  creator?: string;
  executor?: string;
  executeType?: 'immediate' | 'scheduled' | 'periodic' | 'suspended';
}

/** R-9.3: 执行响应中的单条翻译结果（写入 device_task.params 同时回显到 TerminalPanel）。 */
export interface TranslatedPathResult {
  /** 用户视角 standardPath */
  standard: string;
  /** 实际下发 privatePath（passthrough 时与 standard 相同） */
  private: string;
  /** discovered / default / passthrough */
  source: 'discovered' | 'default' | 'passthrough';
}

/** R-9.3: 执行响应中 per-device 的翻译摘要，供 TerminalPanel 按 source 上色回显。 */
export interface ExecutedPathsPerDevice {
  [deviceSn: string]: TranslatedPathResult[];
}

// ============================================================
// R-9.2 Statement → StructuredStatement 转换（前端切结构化通道入口）
// ============================================================

/**
 * statementToStructured —— 把 UI 编辑态 Statement 翻译为后端结构化入参。
 *
 * 映射规则（与后端 console_structured.go StructuredToStatement 对偶）：
 *   - SelectedSubFieldIds (sf.id) → paths (sf.tr069Path)
 *   - values (key=mml_code) → values (key=tr069Path)
 *   - rmvInstanceIndices / rmvInstanceIndex → instanceIndices
 *
 * 边界：subFields 必须已加载（来自 GET /commands/:id/sub-fields），
 * 解析 MML 文本得到的 Statement.subFields 可能为空 — 调用方应避免对此类
 * statement 调用该函数（MmlEditor 走旧通道）。
 */
export function statementToStructured(stmt: Statement): StructuredStatement {
  const sfByID = new Map<string, SubFieldDef>();
  const sfByMmlCode = new Map<string, SubFieldDef>();
  for (const sf of stmt.subFields) {
    sfByID.set(sf.id, sf);
    sfByMmlCode.set(sf.mmlCode, sf);
  }

  // paths：仅纳入命中 sub_field 且 tr069Path 非空的项。命中 sub_field 但
  // 缺 tr069Path 是数据异常（catalog loader 阶段应已 reject），UI 层静默 skip。
  const paths: string[] = [];
  for (const id of stmt.selectedSubFieldIds) {
    const sf = sfByID.get(id);
    if (sf?.tr069Path) paths.push(sf.tr069Path);
  }

  // values：key 从 mml_code 反查 sub_field → 取 tr069Path 作为新 key。
  // 同样静默 skip 缺 tr069Path 的字段（与后端 422 unknown_paths 行为对齐）。
  //
  // 用户决策 2026-05-23：MOD path 改为选填（参考 LST），未勾选行的 value 不应
  // 被下发（用户已显式 "取消修改"）。这里按 selectedSubFieldIds 过滤即可
  // —— store 仍保留所有 values（再次勾选可恢复），但 wire 上只送选中部分。
  // 对 LST / ADD / RMV 无影响：LST 不消费 values；ADD 当前不渲染 Checkbox，
  // selectedSubFieldIds = 所有字段（CommandTree.replaceStatement default 选中
  // defaultSelected），过滤为 no-op。
  const selectedIDSet = new Set(stmt.selectedSubFieldIds);
  const filterValuesBySelection = stmt.operationType === 'MOD';
  const values: Record<string, string> = {};
  for (const [mmlCode, v] of Object.entries(stmt.values)) {
    const sf = sfByMmlCode.get(mmlCode);
    if (!sf?.tr069Path) continue;
    if (filterValuesBySelection && !selectedIDSet.has(sf.id)) continue;
    values[sf.tr069Path] = v;
  }

  // instance indices：rmvInstanceIndices 优先；旧单 Index 兼容回退为 [Index]
  let instanceIndices: number[] | undefined;
  if (stmt.rmvInstanceIndices && stmt.rmvInstanceIndices.length > 0) {
    instanceIndices = stmt.rmvInstanceIndices;
  } else if (typeof stmt.rmvInstanceIndex === 'number') {
    instanceIndices = [stmt.rmvInstanceIndex];
  }

  const out: StructuredStatement = {
    commandId: stmt.commandId ?? '',
    operationType: stmt.operationType as ConsoleSupportedOp,
    paths,
  };
  if (stmt.commandCode) out.commandCode = stmt.commandCode;
  if (Object.keys(values).length > 0) out.values = values;
  if (instanceIndices) out.instanceIndices = instanceIndices;
  // R-4：instanceSelectors 直通；后端 executor 按 key 字典序左到右映射 `.{i}.`
  if (stmt.instanceSelectors && Object.keys(stmt.instanceSelectors).length > 0) {
    out.instanceSelectors = stmt.instanceSelectors;
  }
  return out;
}
