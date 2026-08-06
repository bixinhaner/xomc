// MML 控制台 V2 —— 真实后端类型 → 页面视图类型 适配层（设计 §3.12.3）。
//
// 把 frontend-core 的真实 API 类型（Device / GroupTreeCommand / SubFieldDef /
// DeviceTaskResultItem）映射为 Console 本地视图类型（DeviceItem / CommandItem /
// ResultRow），隔离对接面：组件只认本地 view 类型，换端点只改本文件。

import type { Device } from '@core/types/device';
import type {
  ConsoleSupportedOp,
  GroupTreeCommand,
  GroupTreeNode,
  StructuredStatement,
  SubFieldDef,
} from '@core/types/mmlConsole';
import type {
  DeviceTaskResultItem,
  MMLCustomCommand,
  MMLCustomCommandPathDef,
  MMLOperationType,
  MMLTask,
  MMLTaskCommandDetail,
} from '@core/types/mml';
import { parseMmlDeviceTaskResult } from '@core/utils/mmlResultParser';
import { isReadOp } from './constants';
import type {
  CommandItem,
  CommandParamPath,
  DeviceItem,
  DeviceStatus,
  ExecMode,
  ExecRecord,
  ExecStatus,
  PathTask,
  ResultColumn,
  ResultRow,
} from './types';

/**
 * 逐 PATH 合并时失败单元格/兜底故障的内部 sentinel。
 * 在 .ts 适配层定义（i18n guard 仅作用 .tsx），ResultTable 比对时复用此常量，
 * 避免在 .tsx 里硬编码中文字面量触发 guard。
 */
export const PATH_FAILED_CELL = '__MML_PATH_FAILED__';
export const PATH_TASK_FAILED_FALLBACK = '__MML_PATH_TASK_FAILED__';
export const PARTIAL_PATH_FAILED_FALLBACK = '__MML_PARTIAL_PATH_FAILED__';
export const DISPATCH_FAILED_FALLBACK = '__MML_DISPATCH_FAILED__';
export const READBACK_FAILED_FALLBACK = '__MML_READBACK_FAILED__';
export const MAX_OBJECT_PATH_COLUMNS = 80;
const CONSOLE_PARSE_OPTIONS = { maxParams: MAX_OBJECT_PATH_COLUMNS };

/** 真实设备 → 设备弹框行视图。 */
export function mapDeviceToItem(d: Device): DeviceItem {
  let status: DeviceStatus;
  if (!d.isOnline) status = 'offline';
  else if (d.alarmLevel && d.alarmLevel !== 'none') status = 'alarm';
  else status = 'online';
  return {
    sn: d.sn,
    productName: d.productName || d.deviceModel || '-',
    productClass: d.productClass || '',
    status,
    groupName: d.groupName || '-',
  };
}

/** SubFieldDef[] → 命令参数路径（结果表格列 / 勾选项 / 写入项来源）。 */
export function subFieldsToParamPaths(subFields: SubFieldDef[]): CommandParamPath[] {
  return subFields
    .filter((sf) => sf.tr069Path)
    .map((sf) => ({
      mmlCode: sf.mmlCode,
      path: sf.tr069Path,
      label: sf.label || sf.tr069Path.split('.').filter(Boolean).pop() || sf.tr069Path,
      writable: sf.accessType === 'READ_WRITE',
      isObject: sf.isObject,
      minValue: sf.minValue,
      maxValue: sf.maxValue,
      valueType: sf.valueType,
      defaultValue: sf.defaultValue,
      validationPattern: sf.validationPattern,
      enumOptions: sf.enumOptions,
      description: sf.description,
      defaultSelected: sf.defaultSelected,
    }));
}

/** 拍平后的命令条目（弹框只展示「分组 → 命令」二级）。 */
export interface FlatCommandEntry {
  groupName: string;
  command: GroupTreeCommand;
}

/** 把层级命令树（递归 children）拍平为「分组名 → 命令」二级列表。 */
export function flattenGroupTree(nodes: GroupTreeNode[]): FlatCommandEntry[] {
  const out: FlatCommandEntry[] = [];
  const walk = (node: GroupTreeNode): void => {
    for (const c of node.commands) out.push({ groupName: node.displayName, command: c });
    for (const child of node.children) walk(child);
  };
  nodes.forEach(walk);
  return out;
}

/**
 * 自定义命令的 paramPaths(string[]) → CommandParamPath[]。
 * 自定义命令无 sub_field 元属性：标签取路径叶子名；MOD/ADD 视为可写（用户在配置参数步骤填值），
 * LST/DSP/RMV 不可写。isObject 未知置 false，无 minValue。
 */
export function customCommandParamPaths(cc: MMLCustomCommand): CommandParamPath[] {
  const writable = cc.operationType === 'MOD' || cc.operationType === 'ADD';
  return (cc.paramPaths ?? [])
    .map((p) => p.trim())
    .filter(Boolean)
    .map((path) => ({
      path,
      label: path.split('.').filter(Boolean).pop() || path,
      writable,
      isObject: false,
    }));
}

export function customCommandPathDefsToParamPaths(
  paths: MMLCustomCommandPathDef[],
): CommandParamPath[] {
  return paths.map((p) => ({
    path: p.standardPath,
    label: p.description || p.standardPath.split('.').filter(Boolean).pop() || p.standardPath,
    writable: p.access.replace(/[_-]/g, '').toLowerCase() === 'readwrite',
    isObject: p.entryType === 'object',
    valueType: p.dataType,
    minValue: p.minValue,
    maxValue: p.maxValue,
    description: p.description,
    defaultSelected: p.defaultSelected,
  }));
}

/**
 * MMLCustomCommand + 已过滤 paramPaths → Console CommandItem。
 * 标记 isCustom=true：执行时强制走裸路径通道（结构化端点要 command_id，自定义命令没有）。
 */
export function mapCustomCommandItem(
  cc: MMLCustomCommand,
  groupName: string,
  paramPaths: CommandParamPath[],
): CommandItem {
  return {
    id: cc.id,
    groupName,
    commandCode: cc.commandCode,
    commandName: cc.commandName,
    operationType: cc.operationType,
    description: cc.description ?? '',
    paramPaths,
    isCustom: true,
  };
}

/** GroupTreeCommand + 已加载 sub-fields → Console CommandItem（id = 真实 mml_commands.id）。 */
export function mapCommandItem(
  groupName: string,
  c: GroupTreeCommand,
  subFields: SubFieldDef[],
): CommandItem {
  return {
    id: c.id,
    groupName,
    commandCode: c.commandCode,
    commandName: c.displayName,
    operationType: c.operationType,
    description: '',
    paramPaths: subFieldsToParamPaths(subFields),
    targetObject: c.targetObject,
  };
}

// ── 结果表格动态列派生 ─────────────────────────────────────────────────────────

/** 由命令的 paramPaths 派生结果表格动态列；可选 checkedPaths 仅保留勾选的列。 */
export function buildColumns(command: CommandItem, checkedPaths?: string[]): ResultColumn[] {
  const checked = checkedPaths ? new Set(checkedPaths) : null;
  return command.paramPaths
    .filter((p) => !checked || checked.has(p.path))
    .map((p, idx) => ({ key: `c${idx}`, label: p.label, path: p.path }));
}

/** 由裸路径列表派生结果表格动态列（参数路径指定模式）。列标签取路径叶子名。 */
export function buildColumnsFromRawPaths(paths: string[]): ResultColumn[] {
  return paths
    .map((p) => p.trim())
    .filter(Boolean)
    .map((path, idx) => ({ key: `c${idx}`, label: leafName(path), path }));
}

// ── 执行入参构造（设计 §3.12.2）────────────────────────────────────────────────

const SUPPORTED_OPS: ReadonlySet<string> = new Set(['LST', 'MOD', 'ADD', 'RMV']);

/** 结构化执行通道只支持 LST/MOD/ADD/RMV；其余操作类型回退裸路径通道。 */
export function isStructuredOp(op: MMLOperationType): op is ConsoleSupportedOp {
  return SUPPORTED_OPS.has(op);
}

/**
 * 提取命令需要用户填写的实例占位符（`.{i}.`）槽位 + 标签（占位符前一段对象名）。
 * - ADD/RMV：看 targetObject（父级对象路径）。
 * - LST/MOD：看 paramPaths 中 `.{i}.` 最多的一条（同一命令子字段通常共享对象祖先）。
 * 槽位 key 为零填充 `i01`/`i02`…（与后端 substituteInstanceSelectors 字典序左→右映射对齐）。
 */
export function computeInstanceSlots(command: CommandItem): { key: string; label: string }[] {
  const isAddRmv = command.operationType === 'ADD' || command.operationType === 'RMV';
  let source = '';
  if (isAddRmv) {
    source = command.targetObject ?? '';
  } else {
    const count = (s: string): number => (s.match(/\.\{i\}\./g) ?? []).length;
    for (const p of command.paramPaths) {
      if (count(p.path) > count(source)) source = p.path;
    }
  }
  const segs = source.split('.');
  const slots: { key: string; label: string }[] = [];
  let n = 0;
  for (let k = 0; k < segs.length; k++) {
    // 仅统计前后都有点的 `.{i}.`（与后端 strings.Count(path, ".{i}.") 一致）。
    if (segs[k] === '{i}' && k > 0 && k < segs.length - 1) {
      n += 1;
      slots.push({ key: `i${String(n).padStart(2, '0')}`, label: segs[k - 1] || `实例${n}` });
    }
  }
  return slots;
}

export function buildDefaultInstanceSelectors(
  slots: { key: string; label: string }[],
  operationType: MMLOperationType,
): Record<string, string> {
  const finalIndex = slots.length - 1;
  return Object.fromEntries(
    slots.map((slot, index) => [
      slot.key,
      isReadOp(operationType) && index === finalIndex ? '' : '1',
    ]),
  );
}

export function resolveQueryPath(
  path: string,
  instanceSelectors?: Record<string, string>,
): string {
  let result = path;
  let layer = 1;
  while (true) {
    const marker = result.indexOf('.{i}');
    if (marker < 0) return result;

    const key = `i${String(layer).padStart(2, '0')}`;
    const value = instanceSelectors?.[key]?.trim() ?? '';
    if (value === '') return result.slice(0, marker + 1);

    const hasTrailingDot = result[marker + 4] === '.';
    const markerLength = hasTrailingDot ? 5 : 4;
    const trailingDot = hasTrailingDot ? '.' : '';
    result = `${result.slice(0, marker)}.${value}${trailingDot}${result.slice(marker + markerLength)}`;
    layer += 1;
  }
}

/**
 * 把 targetObject 里的 `.{i}.` 占位按 instanceSelectors 替换为具体实例号，用于
 * ADD/RMV 命令「目标对象路径」展示（与 computeInstanceSlots 同序：左→右 i01/i02…，缺省 1）。
 * 无占位 / 无 targetObject 时原样返回。
 */
export function resolveObjectPath(
  targetObject: string | undefined,
  instanceSelectors?: Record<string, string>,
): string {
  const src = targetObject ?? '';
  if (!src) return src;
  let n = 0;
  return src.replace(/\.\{i\}\./g, () => {
    n += 1;
    const v = instanceSelectors?.[`i${String(n).padStart(2, '0')}`] ?? '1';
    return `.${v}.`;
  });
}

/** CommandItem + 标准模式 ExecRequest → 结构化执行单条 statement（POST …/execute-statements-structured）。 */
export function buildStructuredStatement(
  command: CommandItem,
  checkedPaths: string[],
  values?: Record<string, string>,
  instance?: number,
  instanceSelectors?: Record<string, string>,
): StructuredStatement {
  const op = command.operationType as ConsoleSupportedOp;
  const stmt: StructuredStatement = {
    commandId: command.id,
    operationType: op,
    commandCode: command.commandCode,
    paths: checkedPaths,
  };
  // 父级 `.{i}.` 实例选择器（LST/MOD 作用于 path、ADD/RMV 作用于 targetObject）。
  if (instanceSelectors && Object.keys(instanceSelectors).length > 0) {
    stmt.instanceSelectors = instanceSelectors;
  }
  if (op === 'MOD' || op === 'ADD') {
    const checked = new Set(checkedPaths);
    const picked: Record<string, string> = {};
    command.paramPaths
      .filter((p) => p.writable && checked.has(p.path))
      .forEach((p) => {
        const v = values?.[p.path];
        if (v != null && v !== '') picked[p.path] = v;
      });
    if (Object.keys(picked).length > 0) stmt.values = picked;
  }
  if (op === 'RMV' && typeof instance === 'number') {
    stmt.instanceIndices = [instance];
  }
  return stmt;
}

export function buildStandardQueryColumns(
  command: CommandItem,
  checkedPaths: string[],
  instanceSelectors?: Record<string, string>,
): ResultColumn[] {
  const seen = new Set<string>();
  return buildColumns(command, checkedPaths)
    .map((column) => ({
      ...column,
      path: resolveQueryPath(column.path, instanceSelectors),
    }))
    .filter((column) => {
      if (seen.has(column.path)) return false;
      seen.add(column.path);
      return true;
    });
}

/**
 * 逐 PATH 下发顺序与结果列保持一致。
 * 查询 path 若因空实例截断为同一对象前缀，只保留首次出现的一条 statement；
 * 写命令不做截断去重，继续严格按用户选择逐条下发。
 */
export function buildPerPathStatementPaths(
  command: CommandItem,
  checkedPaths: string[],
  instanceSelectors?: Record<string, string>,
): string[] {
  const checked = new Set(checkedPaths);
  const orderedPaths = command.paramPaths
    .filter((item) => checked.has(item.path))
    .map((item) => item.path);
  if (!isReadOp(command.operationType)) return orderedPaths;

  const seen = new Set<string>();
  return orderedPaths.filter((path) => {
    const resolved = resolveQueryPath(path, instanceSelectors);
    if (seen.has(resolved)) return false;
    seen.add(resolved);
    return true;
  });
}

export function buildStandardRawRows(
  operationType: MMLOperationType,
  checkedPaths: string[],
  values?: Record<string, string>,
  instanceSelectors?: Record<string, string>,
): { path: string; value: string }[] {
  return checkedPaths.map((path) => ({
    path: isReadOp(operationType)
      ? resolveQueryPath(path, instanceSelectors)
      : path,
    value: values?.[path] ?? '',
  }));
}

/** 裸路径模式 ExecRequest → legacy POST /mml/execute 请求体（param_paths/param_values 下标对齐）。 */
export function buildRawExecutePayload(
  operationType: MMLOperationType,
  rows: { path: string; value: string }[],
  deviceSns: string[],
  taskName?: string,
  execMode: ExecMode = 'whole',
  commandName?: string,
): Record<string, unknown> {
  const valid = rows.filter((r) => r.path.trim() !== '');
  const paths = valid.map((r) => r.path.trim());
  // task_name 用与命令记录一致的名称（req4 对应关系）；调用方未传时回退默认。
  const name =
    taskName ??
    `${operationType} ${paths[0] ?? ''}${deviceSns.length === 1 ? ` ${deviceSns[0]}` : ''}`;
  return {
    device_sns: deviceSns,
    param_paths: paths,
    // ADD/RMV/LST 不消费 param_values，但保持下标对齐让后端按 i 配对。
    param_values: valid.map((r) => r.value ?? ''),
    operation_type: operationType,
    execute_type: 'immediate',
    // 逐 PATH：后端把每 path 拆成一条 command（每 path 一个 RPC），path 级成败独立。
    execute_mode: execMode === 'single-path' ? 'single_path' : 'whole',
    task_name: name,
    ...(commandName?.trim() ? { command_name: commandName.trim() } : {}),
  };
}

// ── SSE 结果帧 → 结果行（设计 §3.12.2，先降级 success/failed 两态）─────────────────

/** mml_device_frame 帧（后端 result_aggregator.go publishDeviceFrame）。 */
export interface DeviceFramePayload {
  task_id: string;
  device_task_id?: string;
  device_sn: string;
  method?: string;
  status: string;
  result?: unknown;
  error_message?: string;
  sent_at?: string;
  completed_at?: string;
}

/** mml_task_completed 帧。 */
export interface TaskCompletedPayload {
  task_id: string;
  status: string;
  result: string;
  success_count: number;
  failed_count: number;
}

/** 设备执行前的占位行（status=running，待 SSE 帧回填）。 */
export function initialPendingRows(deviceSns: string[]): ResultRow[] {
  return deviceSns.map((sn) => ({
    deviceSn: sn,
    deviceTaskId: '',
    status: 'running' as ExecStatus,
    cells: {},
    raw: '',
    elapsedMs: 0,
  }));
}

function leafName(path: string): string {
  return path.split('.').filter(Boolean).pop() ?? path;
}

/**
 * GPV partial path（以 `.` 结尾）会返回该对象下的多个叶子参数。控制台初始列只有
 * 用户输入的对象路径，因此要用设备实际返回并保存在 cells 中的叶子路径替换该占位列。
 * 普通叶子查询保持原列不变；多设备结果按首次出现顺序去重。
 */
export function expandObjectPathColumns(
  columns: ResultColumn[],
  rows: ResultRow[],
): ResultColumn[] {
  const expanded: ResultColumn[] = [];
  const emittedPaths = new Set<string>();
  let objectPathColumnCount = 0;
  for (const column of columns) {
    if (!column.path.endsWith('.')) {
      if (emittedPaths.has(column.path)) continue;
      emittedPaths.add(column.path);
      expanded.push(column);
      continue;
    }

    const descendantPaths: string[] = [];
    let hasDescendant = false;
    for (const row of rows) {
      for (const path of Object.keys(row.cells)) {
        if (path === column.path || !path.startsWith(column.path)) continue;
        if (objectPathColumnCount >= MAX_OBJECT_PATH_COLUMNS) continue;
        hasDescendant = true;
        if (emittedPaths.has(path)) continue;
        emittedPaths.add(path);
        objectPathColumnCount += 1;
        descendantPaths.push(path);
      }
    }

    if (!hasDescendant) {
      if (emittedPaths.has(column.path)) continue;
      emittedPaths.add(column.path);
      expanded.push(column);
      continue;
    }
    descendantPaths.forEach((path, index) => {
      expanded.push({ key: `${column.key}:child:${index}`, label: leafName(path), path });
    });
  }
  return expanded;
}

/**
 * 「指定参数」(裸路径)执行的命令记录命名：用执行的 path 命名，优先取设备模型 path 字典里的
 * 友好名（nameMap，来自 standard_params.description），缺省回退路径叶子名；前缀操作中文标签。
 * 例：LST `Device.DeviceInfo.SoftwareVersion` → `Query SoftwareVersion`；无语言上下文时用 `LST` 兜底。
 */
export function rawCommandName(
  op: MMLOperationType,
  paths: string[],
  nameMap?: Record<string, string>,
  opText?: string,
  multiPathSuffix?: string,
): string {
  const first = paths[0] ?? '';
  const friendly = (nameMap && nameMap[first]) || leafName(first) || first;
  const suffix = paths.length > 1 ? (multiPathSuffix ?? '') : '';
  return `${opText || op} ${friendly}${suffix}`.trim();
}

function toClock(iso?: string): string | undefined {
  if (!iso) return undefined;
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? undefined : d.toLocaleTimeString('zh-CN', { hour12: false });
}

/**
 * 把一条 mml_device_frame 帧合并进对应设备行（§3.12.2）。
 *
 * - status：`completed` → success，其余（failed/expired…）→ failed（读后核实四态待后端，§3.11.6）。
 * - 读类（GPV）：按列回填 cells，优先精确路径匹配，回退叶子名匹配
 *   （CPE 返回的是 privatePath，列键是 standardPath，translation 后两者可能不同）。
 * - raw：优先取 result.raw_response（CWMP SOAP 原文）供详情页 XmlViewer 展示。
 */
export function applyFrameToRow(
  prev: ResultRow,
  frame: DeviceFramePayload,
  columns: ResultColumn[],
  read: boolean,
): ResultRow {
  const status: ExecStatus = frame.status === 'completed' ? 'success' : 'failed';
  const cells = { ...prev.cells };
  let raw = prev.raw;

  if (frame.result != null) {
    const env = frame.result as { raw_response?: unknown };
    if (typeof env.raw_response === 'string') {
      raw = env.raw_response;
    } else {
      raw = typeof frame.result === 'string' ? frame.result : JSON.stringify(frame.result, null, 2);
    }
    if (read && status === 'success') {
      const parsed = parseMmlDeviceTaskResult(frame.result, CONSOLE_PARSE_OPTIONS);
      if (parsed?.kind === 'gpv' && parsed.params) {
        const byPath = new Map(parsed.params.map((p) => [p.name, p.value]));
        const byLeaf = new Map(parsed.params.map((p) => [leafName(p.name), p.value]));
        // partial object path 查询会返回多个后代叶子；全部保留，供结果表动态展开。
        for (const p of parsed.params) cells[p.name] = p.value;
        for (const c of columns) {
          const v = byPath.get(c.path) ?? byLeaf.get(leafName(c.path));
          if (v != null) cells[c.path] = v;
        }
      }
    }
  }

  return {
    ...prev,
    deviceTaskId: frame.device_task_id ?? prev.deviceTaskId,
    status,
    cells,
    faultCode: frame.error_message || prev.faultCode,
    dispatchedAt: toClock(frame.sent_at) ?? prev.dispatchedAt,
    respondedAt: toClock(frame.completed_at) ?? prev.respondedAt,
    raw,
  };
}

// ── 命令记录跨刷新重建（P3，设计 §3.12.4）：凭 GET /mml/tasks/:id 重建 ExecRecord ──

/** 单台设备的落库结果（DeviceTaskResultItem）→ 结果行。读类按列回填 parsedData（exact→leaf）。 */
export function mapResultItemToRow(
  item: DeviceTaskResultItem,
  columns: ResultColumn[],
  read: boolean,
): ResultRow {
  const status: ExecStatus = item.result.success ? 'success' : 'failed';
  const cells: Record<string, string> = {};
  if (read && status === 'success' && item.result.parsedData) {
    // parsedData 是结果信封 {method, raw_response}，需解析 raw_response 的 GPV 取 name→value
    // （与 SSE 路径 applyFrameToRow 一致）；早前直接把信封当 name→value 映射导致读回值全空。
    const parsed = parseMmlDeviceTaskResult(item.result.parsedData, CONSOLE_PARSE_OPTIONS);
    if (parsed?.kind === 'gpv' && parsed.params) {
      const byPath = new Map(parsed.params.map((p) => [p.name, p.value]));
      const byLeaf = new Map(parsed.params.map((p) => [leafName(p.name), p.value]));
      // 与 SSE 路径一致：保留 partial object path 返回的全部后代叶子。
      for (const p of parsed.params) cells[p.name] = p.value;
      for (const c of columns) {
        const v = byPath.get(c.path) ?? byLeaf.get(leafName(c.path));
        if (v != null) cells[c.path] = v;
      }
    }
  }
  return {
    planLineNo: item.planLineNo,
    planOrder: item.planOrder,
    planRawLine: item.planRawLine,
    commandCode: item.commandCode,
    commandName: item.commandName,
    deviceSn: item.deviceSn,
    deviceTaskId: item.deviceTaskId ?? '',
    status,
    cells,
    faultCode: item.failReason,
    dispatchedAt: toClock(item.startedAt), // 下发时间 = device_tasks.sent_at（后端透传为 started_at）
    respondedAt: toClock(item.finishedAt),
    raw: item.result.rawOutput ?? '',
    elapsedMs: item.result.executionTime ?? 0,
  };
}

/** 普通 MML 子任务没有脚本计划行，不应显示永远为空的 Plan Row 列。 */
export function hasPlanRows(rows: Pick<ResultRow, 'planLineNo'>[]): boolean {
  return rows.some((row) => typeof row.planLineNo === 'number');
}

/**
 * 把后端逐条结果合并为「每设备一行」。
 * - 整体下发：每设备 1 条结果 → 直接 mapResultItemToRow。
 * - 逐 PATH：每设备 N 条结果（每 path 一条 device_task，command_index 定位 path）→
 *   合并为一行：成功 path 填读回值，失败 path 单元格标内部 sentinel，行状态 = 全成功才 success，
 *   否则 failed；并填 pathTasks 供「查看」详情展示 path 级成败。
 * columns 按 command_index 顺序（逐 PATH 时 columns[i] 即第 i 条 command 的 path）。
 */
export function buildDeviceRows(
  items: DeviceTaskResultItem[],
  columns: ResultColumn[],
  read: boolean,
): ResultRow[] {
  if (items.some((it) => typeof it.planLineNo === 'number')) {
    return [...items]
      .sort((a, b) => (a.planLineNo ?? 0) - (b.planLineNo ?? 0) || (a.planOrder ?? 0) - (b.planOrder ?? 0))
      .map((it) => mapResultItemToRow(it, columns, read));
  }

  const byDevice = new Map<string, DeviceTaskResultItem[]>();
  for (const it of items) {
    const arr = byDevice.get(it.deviceSn) ?? [];
    arr.push(it);
    byDevice.set(it.deviceSn, arr);
  }
  const rows: ResultRow[] = [];
  for (const [, devItems] of byDevice) {
    if (devItems.length <= 1) {
      rows.push(mapResultItemToRow(devItems[0], columns, read));
      continue;
    }
    // 逐 PATH 合并
    const base = devItems.map((it) => mapResultItemToRow(it, columns, read));
    const cells: Record<string, string> = Object.assign({}, ...base.map((r) => r.cells));
    const pathTasks: PathTask[] = [];
    devItems.forEach((it, i) => {
      const idx = typeof it.commandIndex === 'number' ? it.commandIndex : i;
      const path = columns[idx]?.path ?? '';
      if (base[i].status === 'failed' && path) cells[path] = PATH_FAILED_CELL;
      pathTasks.push({
        pathIndex: idx,
        path,
        subTaskId: it.deviceTaskId ?? '',
        status: base[i].status,
        dispatchedAt: base[i].dispatchedAt ?? '',
        respondedAt: base[i].respondedAt ?? '',
        value: base[i].status === 'success' ? (base[i].cells[path] ?? '') : (it.failReason ?? PATH_TASK_FAILED_FALLBACK),
      });
    });
    pathTasks.sort((a, b) => a.pathIndex - b.pathIndex);
    const allOk = base.every((r) => r.status === 'success');
    rows.push({
      ...base[0],
      status: allOk ? 'success' : 'failed',
      cells,
      faultCode: allOk ? undefined : PARTIAL_PATH_FAILED_FALLBACK,
      pathTasks,
    });
  }
  return rows;
}

const upperOp = (d?: MMLTaskCommandDetail): string => (d?.operationType ?? '').toString().toUpperCase();

/**
 * #196：MOD 自动回读复合（SetParameterValues 下发 + GetParameterValues 回读）→ 每设备一行。
 * 把「下发值（命令 paramValues）」与「回读值（回读 LST 结果）」按 path 关联成 verify 对比，
 * 并分别保留 MOD 下发响应报文（raw）与回读 LST 响应报文（readbackRaw）。
 */
export function buildMODReadbackRows(
  items: DeviceTaskResultItem[],
  setValues: Record<string, string>,
  commandMeta?: { commandName?: string; commandCode?: string },
): ResultRow[] {
  // 按结果报文判别下发(SPV)/回读(GPV)，不依赖 commands_detail 是否透出回读命令。
  const isLst = (it: DeviceTaskResultItem): boolean =>
    !!it.result?.parsedData && parseMmlDeviceTaskResult(it.result.parsedData, CONSOLE_PARSE_OPTIONS)?.kind === 'gpv';

  const byDevice = new Map<string, DeviceTaskResultItem[]>();
  for (const it of items) {
    const arr = byDevice.get(it.deviceSn) ?? [];
    arr.push(it);
    byDevice.set(it.deviceSn, arr);
  }

  const rows: ResultRow[] = [];
  for (const [deviceSn, devItems] of byDevice) {
    const modItems = devItems.filter((it) => !isLst(it));
    const lstItem = devItems.find((it) => isLst(it));
    const modTaskId = modItems[0]?.deviceTaskId ?? '';
    const lstTaskId = lstItem?.deviceTaskId ?? '';

    // 回读值 + 回读 path（GetParameterValues 响应的参数名即 PATH，修复回读行 PATH 为空）
    const readback = new Map<string, string>();
    const readbackPairs: { path: string; value: string }[] = [];
    if (lstItem?.result?.parsedData) {
      const parsed = parseMmlDeviceTaskResult(lstItem.result.parsedData, CONSOLE_PARSE_OPTIONS);
      if (parsed?.kind === 'gpv' && parsed.params) {
        for (const p of parsed.params) {
          readback.set(p.name, p.value);
          readback.set(leafName(p.name), p.value);
          readbackPairs.push({ path: p.name, value: p.value });
        }
      }
    }
    const readVal = (path: string): string => readback.get(path) ?? readback.get(leafName(path)) ?? '';
    const hasReadback = readbackPairs.length > 0;
    const modOk = modItems.length > 0 && modItems.every((it) => it.result?.success);
    const lstOk = lstItem?.result?.success === true;
    const firstMod = modItems[0];

    // 「PATH 列表」：MOD（下发）行 —— path 取自下发参数；LST（回读）行 —— path 取自回读响应。
    const pathTasks: PathTask[] = [];
    for (const path of Object.keys(setValues)) {
      pathTasks.push({
        pathIndex: 0,
        path,
        subTaskId: modTaskId,
        opType: 'MOD',
        status: modOk ? 'success' : 'failed',
        dispatchedAt: toClock(firstMod?.startedAt) ?? '',
        respondedAt: toClock(firstMod?.finishedAt) ?? '',
        value: modOk ? (setValues[path] ?? '') : (firstMod?.failReason ?? PATH_TASK_FAILED_FALLBACK),
      });
    }
    if (lstItem) {
      const lstRows = hasReadback
        ? readbackPairs
        : Object.keys(setValues).map((path) => ({ path, value: lstOk ? '' : (lstItem.failReason ?? READBACK_FAILED_FALLBACK) }));
      for (const { path, value } of lstRows) {
        pathTasks.push({
          pathIndex: 1,
          path,
          subTaskId: lstTaskId,
          opType: 'LST',
          status: lstOk ? 'success' : 'failed',
          dispatchedAt: toClock(lstItem.startedAt) ?? '',
          respondedAt: toClock(lstItem.finishedAt) ?? '',
          value,
        });
      }
    }

    const cells: Record<string, string> = {};
    for (const path of Object.keys(setValues)) {
      const rv = readVal(path);
      if (rv) cells[path] = rv;
    }

    let status: ExecStatus = 'success';
    if (!modOk) status = 'failed';
    else if (!hasReadback) status = 'unverified';
    else if (Object.keys(setValues).some((p) => readVal(p) !== setValues[p])) status = 'mismatch';

    rows.push({
      commandName: commandMeta?.commandName ?? firstMod?.commandName,
      commandCode: commandMeta?.commandCode ?? firstMod?.commandCode,
      deviceSn,
      deviceTaskId: modTaskId,
      status,
      cells,
      faultCode: modOk ? undefined : (firstMod?.failReason ?? DISPATCH_FAILED_FALLBACK),
      unverifiedReason: status === 'unverified' ? 'query-failed' : undefined,
      pathTasks,
      dispatchedAt: toClock(firstMod?.startedAt),
      respondedAt: toClock(lstItem?.finishedAt ?? firstMod?.finishedAt),
      raw: firstMod?.result?.rawOutput ?? '',
      readbackRaw: lstItem?.result?.rawOutput ?? '',
      elapsedMs: (firstMod?.result?.executionTime ?? 0) + (lstItem?.result?.executionTime ?? 0),
    });
  }
  return rows;
}

/**
 * 真实任务（GET /mml/tasks/:id）→ Console 命令记录（含结果行）。
 *
 * - 命令元信息取首条 `commandsDetail`（op_type + param_paths）；老任务缺 detail 时降级为
 *   仅摘要（columns/rows 尽力而为）。
 * - 结果行取任务内嵌 `results`（每台设备 success/rawOutput/parsedData）。
 */
export function mapTaskToRecord(task: MMLTask): ExecRecord {
  const detail = task.commandsDetail?.[0];
  const op = (detail?.operationType ?? 'LST') as MMLOperationType;
  const read = isReadOp(op);

  // #196：MOD 自动回读复合（首命令 MOD + 追加回读 LST）→ 专用「下发 vs 回读」关联视图。
  const details = task.commandsDetail ?? [];
  const isMODReadback =
    details.length >= 2 &&
    upperOp(details[0]) === 'MOD' &&
    details.some((d, i) => i > 0 && upperOp(d) === 'LST');
  if (isMODReadback) {
    // 下发值：path -> value（取所有 MOD 命令的 paramPaths/paramValues，同序对应）
    const setValues: Record<string, string> = {};
    for (const d of details) {
      if (upperOp(d) !== 'MOD') continue;
      (d.paramPaths ?? []).forEach((p, i) => {
        // 裸路径/自定义 MOD 的下发值在 parameters[path]；结构化命令的
        // parameters 使用 param_refs[].param_code 作为 key，而 paramPaths 是 TR-069 path。
        // 优先 paramValues，兼容新旧两种快照形态。
        const paramCode = d.paramRefs?.find((ref) => ref.tr069Path === p)?.paramCode;
        const v = d.paramValues?.[i] ?? d.parameters?.[p] ?? (paramCode ? d.parameters?.[paramCode] : undefined);
        setValues[p] = v != null ? String(v as unknown) : '';
      });
    }
    const cols = buildColumnsFromRawPaths(Object.keys(setValues)); // Object.keys 去重 → 不再 PATH(2)
    const rawCode0 = (detail?.commandCode ?? '').startsWith('RAW');
    const name = rawCode0
      ? rawCommandName('MOD' as MMLOperationType, Object.keys(setValues))
      : (detail?.commandName ?? detail?.commandCode ?? task.taskName ?? task.id);
    return {
      id: task.id,
      status: 'done',
      commandId: task.id,
      time: toClock(task.finishedAt ?? task.createdAt) ?? '',
      commandName: name,
      operationType: 'MOD' as MMLOperationType,
      deviceCount: task.totalDevices || task.deviceSns.length,
      execMeta: { operationType: 'MOD' as MMLOperationType, read: false, label: name, commandName: name },
      columns: cols,
      rows: buildMODReadbackRows((task.results ?? []) as unknown as DeviceTaskResultItem[], setValues, {
        commandName: detail?.commandName,
        commandCode: detail?.commandCode,
      }),
      // 跨刷新惰性补结果行（useConsoleHistory）据此走 buildMODReadbackRows，而非逐 PATH。
      setValues,
    };
  }
  // 逐 PATH 任务有多条 command（每 path 一条）→ 列取所有 command 的 path 展平（按 command_index 序，
  // 与 buildDeviceRows 的 columns[command_index] 定位一致）；整体下发时即首条 command 的全部 path。
  const allPaths = (task.commandsDetail ?? []).flatMap((c) => c.paramPaths ?? []);
  const columns = allPaths.length ? buildColumnsFromRawPaths(allPaths) : [];
  // 裸路径任务（后端 command_code = "RAW LST/MOD/ADD/RMV"）用执行 path 命名（跨刷新重建时
  // 无字典异步查询，回退路径叶子名）；结构化命令优先用后端注入的友好命令名（command_name，
  // 如「列出 设备基本信息」），缺失再回退 command_code → task_name。
  const rawCode = (detail?.commandCode ?? '').startsWith('RAW') && (detail?.paramPaths?.length ?? 0) > 0;
  const commandName = rawCode
    ? rawCommandName(op, detail!.paramPaths!)
    : (detail?.commandName ?? detail?.commandCode ?? task.taskName ?? task.id);
  const items = (task.results ?? []) as unknown as DeviceTaskResultItem[];
  // 逐 PATH 任务每设备多条结果 → buildDeviceRows 合并为每设备一行（整体下发时退化为一行/设备）。
  const rows = buildDeviceRows(items, columns, read);
  return {
    id: task.id,
    // 跨刷新从后端重建的都是已落库的历史任务，记录态视为已完成。
    status: 'done',
    commandId: task.id,
    time: toClock(task.finishedAt ?? task.createdAt) ?? '',
    commandName,
    operationType: op,
    deviceCount: task.totalDevices || task.deviceSns.length,
    execMeta: { operationType: op, read, label: commandName, commandName },
    columns,
    rows,
  };
}
