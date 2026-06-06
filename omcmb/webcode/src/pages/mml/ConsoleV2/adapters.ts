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
import type { DeviceTaskResultItem, MMLOperationType, MMLTask } from '@core/types/mml';
import { parseMmlDeviceTaskResult } from '@core/utils/mmlResultParser';
import { isReadOp } from './constants';
import type {
  CommandItem,
  CommandParamPath,
  DeviceItem,
  DeviceStatus,
  ExecRecord,
  ExecStatus,
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

/** CommandItem + 标准模式 ExecRequest → 结构化执行单条 statement（POST …/execute-statements-structured）。 */
export function buildStructuredStatement(
  command: CommandItem,
  checkedPaths: string[],
  values?: Record<string, string>,
  instance?: number,
): StructuredStatement {
  const op = command.operationType as ConsoleSupportedOp;
  const stmt: StructuredStatement = {
    commandId: command.id,
    operationType: op,
    commandCode: command.commandCode,
    paths: checkedPaths,
  };
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
): Record<string, unknown> {
  const valid = rows.filter((r) => r.path.trim() !== '');
  const paths = valid.map((r) => r.path.trim());
  return {
    device_sns: deviceSns,
    param_paths: paths,
    // ADD/RMV/LST 不消费 param_values，但保持下标对齐让后端按 i 配对。
    param_values: valid.map((r) => r.value ?? ''),
    operation_type: operationType,
    execute_type: 'immediate',
    task_name: `${operationType} ${paths[0] ?? ''}${
      deviceSns.length === 1 ? ` ${deviceSns[0]}` : ` 等${deviceSns.length}台`
    }`,
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
    const entries = Object.entries(item.result.parsedData).map(
      ([k, v]) => [k, v == null ? '' : String(v)] as const,
    );
    const byPath = new Map(entries);
    const byLeaf = new Map(entries.map(([k, v]) => [leafName(k), v]));
    for (const c of columns) {
      const v = byPath.get(c.path) ?? byLeaf.get(leafName(c.path));
      if (v != null) cells[c.path] = v;
    }
  }
  return {
    deviceSn: item.deviceSn,
    deviceTaskId: '',
    status,
    cells,
    faultCode: item.failReason,
    respondedAt: toClock(item.finishedAt),
    raw: item.result.rawOutput ?? '',
    elapsedMs: item.result.executionTime ?? 0,
  };
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
  const columns = detail?.paramPaths ? buildColumnsFromRawPaths(detail.paramPaths) : [];
  const commandName = detail?.commandCode ?? task.taskName ?? task.id;
  const items = (task.results ?? []) as unknown as DeviceTaskResultItem[];
  const rows = items.map((r) => mapResultItemToRow(r, columns, read));
  return {
    id: task.id,
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
