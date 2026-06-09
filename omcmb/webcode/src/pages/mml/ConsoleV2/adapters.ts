// MML 控制台 V2 —— 真实后端类型 → 页面视图类型 适配层（设计 §3.12.3）。
//
// 把 frontend-core 的真实 API 类型（Device / GroupTreeCommand / SubFieldDef /
// DeviceTaskResultItem）映射为 ConsoleV2 本地视图类型（DeviceItem / CommandItem /
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
  MMLOperationType,
  MMLTask,
} from '@core/types/mml';
import { parseMmlDeviceTaskResult } from '@core/utils/mmlResultParser';
import { isReadOp, opLabel } from './constants';
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
      path: sf.tr069Path,
      label: sf.label || sf.tr069Path.split('.').filter(Boolean).pop() || sf.tr069Path,
      writable: sf.accessType === 'READ_WRITE',
      isObject: sf.isObject,
      minValue: sf.minValue,
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

/**
 * MMLCustomCommand + 已过滤 paramPaths → ConsoleV2 CommandItem。
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

/** GroupTreeCommand + 已加载 sub-fields → ConsoleV2 CommandItem（id = 真实 mml_commands.id）。 */
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

/** 裸路径模式 ExecRequest → legacy POST /mml/execute 请求体（param_paths/param_values 下标对齐）。 */
export function buildRawExecutePayload(
  operationType: MMLOperationType,
  rows: { path: string; value: string }[],
  deviceSns: string[],
  taskName?: string,
  execMode: ExecMode = 'whole',
): Record<string, unknown> {
  const valid = rows.filter((r) => r.path.trim() !== '');
  const paths = valid.map((r) => r.path.trim());
  // task_name 用与命令记录一致的名称（req4 对应关系）；调用方未传时回退默认。
  const name =
    taskName ??
    `${operationType} ${paths[0] ?? ''}${
      deviceSns.length === 1 ? ` ${deviceSns[0]}` : ` 等${deviceSns.length}台`
    }`;
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
 * 「指定参数」(裸路径)执行的命令记录命名：用执行的 path 命名，优先取设备模型 path 字典里的
 * 友好名（nameMap，来自 standard_params.description），缺省回退路径叶子名；前缀操作中文标签。
 * 例：LST `Device.DeviceInfo.SoftwareVersion` → 「查询 软件版本」（无字典命中时「查询 SoftwareVersion」）。
 */
export function rawCommandName(
  op: MMLOperationType,
  paths: string[],
  nameMap?: Record<string, string>,
): string {
  const first = paths[0] ?? '';
  const friendly = (nameMap && nameMap[first]) || leafName(first) || first;
  const suffix = paths.length > 1 ? ` 等${paths.length}项` : '';
  return `${opLabel(op)} ${friendly}${suffix}`.trim();
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
      const parsed = parseMmlDeviceTaskResult(frame.result);
      if (parsed?.kind === 'gpv' && parsed.params) {
        const byPath = new Map(parsed.params.map((p) => [p.name, p.value]));
        const byLeaf = new Map(parsed.params.map((p) => [leafName(p.name), p.value]));
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
    const parsed = parseMmlDeviceTaskResult(item.result.parsedData);
    if (parsed?.kind === 'gpv' && parsed.params) {
      const byPath = new Map(parsed.params.map((p) => [p.name, p.value]));
      const byLeaf = new Map(parsed.params.map((p) => [leafName(p.name), p.value]));
      for (const c of columns) {
        const v = byPath.get(c.path) ?? byLeaf.get(leafName(c.path));
        if (v != null) cells[c.path] = v;
      }
    }
  }
  return {
    deviceSn: item.deviceSn,
    deviceTaskId: '',
    status,
    cells,
    faultCode: item.failReason,
    dispatchedAt: toClock(item.startedAt), // 下发时间 = device_tasks.sent_at（后端透传为 started_at）
    respondedAt: toClock(item.finishedAt),
    raw: item.result.rawOutput ?? '',
    elapsedMs: item.result.executionTime ?? 0,
  };
}

/**
 * 把后端逐条结果合并为「每设备一行」。
 * - 整体下发：每设备 1 条结果 → 直接 mapResultItemToRow。
 * - 逐 PATH：每设备 N 条结果（每 path 一条 device_task，command_index 定位 path）→
 *   合并为一行：成功 path 填读回值，失败 path 单元格标「✗ 失败」，行状态 = 全成功才 success，
 *   否则 failed；并填 pathTasks 供「查看」详情展示 path 级成败。
 * columns 按 command_index 顺序（逐 PATH 时 columns[i] 即第 i 条 command 的 path）。
 */
export function buildDeviceRows(
  items: DeviceTaskResultItem[],
  columns: ResultColumn[],
  read: boolean,
): ResultRow[] {
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
      if (base[i].status === 'failed' && path) cells[path] = '✗ 失败';
      pathTasks.push({
        pathIndex: idx,
        path,
        subTaskId: '',
        status: base[i].status,
        dispatchedAt: base[i].dispatchedAt ?? '',
        respondedAt: base[i].respondedAt ?? '',
        value: base[i].status === 'success' ? (base[i].cells[path] ?? '') : (it.failReason ?? '失败'),
      });
    });
    pathTasks.sort((a, b) => a.pathIndex - b.pathIndex);
    const allOk = base.every((r) => r.status === 'success');
    rows.push({
      ...base[0],
      status: allOk ? 'success' : 'failed',
      cells,
      faultCode: allOk ? undefined : '部分 path 失败',
      pathTasks,
    });
  }
  return rows;
}

/**
 * 真实任务（GET /mml/tasks/:id）→ ConsoleV2 命令记录（含结果行）。
 *
 * - 命令元信息取首条 `commandsDetail`（op_type + param_paths）；老任务缺 detail 时降级为
 *   仅摘要（columns/rows 尽力而为）。
 * - 结果行取任务内嵌 `results`（每台设备 success/rawOutput/parsedData）。
 */
export function mapTaskToRecord(task: MMLTask): ExecRecord {
  const detail = task.commandsDetail?.[0];
  const op = (detail?.operationType ?? 'LST') as MMLOperationType;
  const read = isReadOp(op);
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
