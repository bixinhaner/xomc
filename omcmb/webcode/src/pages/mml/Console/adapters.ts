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
import {
  parseMmlDeviceTaskResult,
  type ParseMmlResultOptions,
} from '@core/utils/mmlResultParser';
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
      isRequired: sf.isRequired,
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

function templateDescendsFromResolvedPath(template: string, resolvedPath: string): boolean {
  const templateSegments = template.split('.').filter(Boolean);
  const resolvedSegments = resolvedPath.split('.').filter(Boolean);
  if (templateSegments.length <= resolvedSegments.length) return false;
  return resolvedSegments.every((segment, index) => {
    const templateSegment = templateSegments[index];
    return templateSegment === '{i}' ? /^\d+$/.test(segment) : templateSegment === segment;
  });
}

function buildTaskResultColumns(details: MMLTaskCommandDetail[]): ResultColumn[] {
  const byPath = new Map<string, ResultColumn>();
  details.forEach((detail) => {
    const selectedTemplates = detail.selectedStandardPaths ?? [];
    (detail.paramPaths ?? []).forEach((rawPath) => {
      const path = rawPath.trim();
      if (!path) return;
      const existing = byPath.get(path);
      const matchingTemplates = path.endsWith('.')
        ? selectedTemplates.filter((template) => (
            !template.endsWith('.')
            && !template.endsWith('{i}')
            && templateDescendsFromResolvedPath(template, path)
          ))
        : [];
      if (!existing) {
        byPath.set(path, {
          key: `c${byPath.size}`,
          label: leafName(path),
          path,
          ...(matchingTemplates.length > 0
            ? { selectedPathTemplates: [...new Set(matchingTemplates)] }
            : {}),
        });
        return;
      }
      if (matchingTemplates.length > 0) {
        existing.selectedPathTemplates = [
          ...new Set([...(existing.selectedPathTemplates ?? []), ...matchingTemplates]),
        ];
      }
    });
  });
  return [...byPath.values()];
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
    const count = (s: string): number => (s.match(/\.\{i\}(?=\.|$)/g) ?? []).length;
    for (const p of command.paramPaths) {
      if (count(p.path) > count(source)) source = p.path;
    }
  }
  const segs = source.split('.');
  const slots: { key: string; label: string }[] = [];
  let n = 0;
  for (let k = 0; k < segs.length; k++) {
    // 同时统计中间 `.{i}.` 与末级对象 `.{i}`；后端查询替换支持两种形态。
    if (segs[k] === '{i}' && k > 0) {
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
  const paramsByPath = new Map(command.paramPaths.map((param) => [param.path, param]));
  const byResolvedPath = new Map<string, ResultColumn>();
  for (const column of buildColumns(command, checkedPaths)) {
    const originalPath = column.path;
    const resolvedPath = resolveQueryPath(originalPath, instanceSelectors);
    const sourceParam = paramsByPath.get(originalPath);
    const selectedPathTemplates = resolvedPath.endsWith('.')
      && resolvedPath !== originalPath
      && !sourceParam?.isObject
      ? [originalPath]
      : undefined;
    const existing = byResolvedPath.get(resolvedPath);
    if (!existing) {
      byResolvedPath.set(resolvedPath, {
        ...column,
        path: resolvedPath,
        ...(selectedPathTemplates ? { selectedPathTemplates } : {}),
      });
      continue;
    }

    // 同一对象前缀可能由多个叶子折叠而来，必须合并所有原始选择；如果其中一项本身
    // 是显式对象查询，则不设过滤模板，保留“展示全部后代”的对象查询语义。
    if (!selectedPathTemplates) {
      delete existing.selectedPathTemplates;
      continue;
    }
    if (existing.selectedPathTemplates) {
      existing.selectedPathTemplates = [
        ...new Set([...existing.selectedPathTemplates, ...selectedPathTemplates]),
      ];
    }
  }
  return [...byResolvedPath.values()];
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

function nonEmptyString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined;
}

function decodeXmlText(value: string): string {
  return value
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, "'")
    .replace(/&amp;/g, '&');
}

function readXmlTagText(xml: string, tag: string): string | undefined {
  const match = xml.match(new RegExp(`<(?:[a-zA-Z][\\w-]*:)?${tag}\\b[^>]*>([\\s\\S]*?)<\\/(?:[a-zA-Z][\\w-]*:)?${tag}>`));
  return match ? decodeXmlText(match[1].trim()) : undefined;
}

function faultMessageFromRawResponse(rawResponse?: string): string | undefined {
  if (!rawResponse) return undefined;
  const soapCode = readXmlTagText(rawResponse, 'faultcode');
  const soapString = readXmlTagText(rawResponse, 'faultstring');
  const cwmpCode = readXmlTagText(rawResponse, 'FaultCode');
  const cwmpString = readXmlTagText(rawResponse, 'FaultString');
  const primaryString = cwmpString ?? soapString;
  const primaryCode = cwmpCode ?? soapCode;
  if (primaryCode && primaryString) return `[${primaryCode}] ${primaryString}`;
  return primaryString ?? primaryCode;
}

function formatParamFault(fault: unknown): string | undefined {
  if (!fault || typeof fault !== 'object') return undefined;
  const source = fault as Record<string, unknown>;
  const path = nonEmptyString(source.parameter_name) ?? nonEmptyString(source.parameterName);
  const code = typeof source.fault_code === 'number'
    ? String(source.fault_code)
    : nonEmptyString(source.fault_code) ?? nonEmptyString(source.faultCode);
  const text = nonEmptyString(source.fault_string) ?? nonEmptyString(source.faultString);
  const detail = [code, text].filter(Boolean).join(' ');
  if (path && detail) return `${path}: ${detail}`;
  return detail || path;
}

function deviceFaultMessageFromResult(result: unknown): string | undefined {
  if (!result || typeof result !== 'object') return undefined;
  const source = result as Record<string, unknown>;
  const paramFaults = Array.isArray(source.param_faults)
    ? source.param_faults.map(formatParamFault).filter((item): item is string => Boolean(item))
    : [];
  if (paramFaults.length > 0) return paramFaults.join('; ');

  const code = typeof source.fault_code === 'number'
    ? String(source.fault_code)
    : nonEmptyString(source.fault_code) ?? nonEmptyString(source.faultCode);
  const text = nonEmptyString(source.fault_string) ?? nonEmptyString(source.faultString);
  if (code && text) return `[${code}] ${text}`;
  if (text) return text;

  return faultMessageFromRawResponse(nonEmptyString(source.raw_response));
}

function resultItemFaultText(item: DeviceTaskResultItem, fallback?: string): string | undefined {
  return deviceFaultMessageFromResult(item.result?.parsedData) ?? item.failReason ?? fallback;
}

const pathTemplateMatcherCache = new Map<string, RegExp>();

function pathTemplateMatcher(template: string): RegExp {
  const cached = pathTemplateMatcherCache.get(template);
  if (cached) return cached;
  const source = template
    .split('.')
    .filter(Boolean)
    .map((segment) => (
      segment === '{i}'
        ? '\\d+'
        : segment.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    ))
    .join('\\.');
  const matcher = new RegExp(`^${source}$`);
  pathTemplateMatcherCache.set(template, matcher);
  return matcher;
}

function pathMatchesTemplate(path: string, template: string): boolean {
  return pathTemplateMatcher(template).test(path);
}

function resultPathMatchesColumn(path: string, column: ResultColumn): boolean {
  if (!column.path.endsWith('.')) {
    return path === column.path || leafName(path) === leafName(column.path);
  }
  if (path === column.path || !path.startsWith(column.path)) return false;
  const templates = column.selectedPathTemplates;
  return !templates?.length || templates.some((template) => pathMatchesTemplate(path, template));
}

function objectDescendantLabel(path: string, column: ResultColumn): string {
  const baseLabel = leafName(path);
  const pathSegments = path.split('.').filter(Boolean);
  const prefixSegments = column.path.split('.').filter(Boolean);
  const matchingTemplate = column.selectedPathTemplates?.find(
    (template) => pathMatchesTemplate(path, template),
  );
  const contexts: string[] = [];

  if (matchingTemplate) {
    const templateSegments = matchingTemplate.split('.').filter(Boolean);
    for (let index = prefixSegments.length; index < templateSegments.length; index += 1) {
      if (templateSegments[index] !== '{i}' || !/^\d+$/.test(pathSegments[index] ?? '')) continue;
      const objectName = templateSegments[index - 1];
      if (objectName && objectName !== '{i}') {
        contexts.push(`${objectName}.${pathSegments[index]}`);
      }
    }
  } else {
    // 显式对象查询没有原始叶子模板，按返回 PATH 中“对象名.数字实例”的结构兜底提取。
    let objectName = prefixSegments[prefixSegments.length - 1];
    for (let index = prefixSegments.length; index < pathSegments.length - 1; index += 1) {
      const segment = pathSegments[index];
      if (/^\d+$/.test(segment)) {
        if (objectName) contexts.push(`${objectName}.${segment}`);
      } else {
        objectName = segment;
      }
    }
  }

  return contexts.length > 0 ? `${baseLabel} [${contexts.join('/')}]` : baseLabel;
}

function objectInstanceSortKey(path: string, column: ResultColumn): number[] {
  const pathSegments = path.split('.').filter(Boolean);
  const prefixLength = column.path.split('.').filter(Boolean).length;
  const matchingTemplate = column.selectedPathTemplates?.find(
    (template) => pathMatchesTemplate(path, template),
  );

  if (matchingTemplate) {
    const templateSegments = matchingTemplate.split('.').filter(Boolean);
    return templateSegments.flatMap((segment, index) => (
      index >= prefixLength && segment === '{i}' && /^\d+$/.test(pathSegments[index] ?? '')
        ? [Number(pathSegments[index])]
        : []
    ));
  }

  // 显式对象查询没有原始叶子模板：对象前缀之后的数字段均视为动态实例层级。
  return pathSegments.slice(prefixLength, -1).flatMap((segment) => (
    /^\d+$/.test(segment) ? [Number(segment)] : []
  ));
}

function compareInstanceSortKeys(left: number[], right: number[]): number {
  const levels = Math.max(left.length, right.length);
  for (let index = 0; index < levels; index += 1) {
    const leftValue = left[index];
    const rightValue = right[index];
    if (leftValue === undefined) return -1;
    if (rightValue === undefined) return 1;
    if (leftValue !== rightValue) return leftValue - rightValue;
  }
  return 0;
}

function sortObjectDescendantPaths(paths: string[], column: ResultColumn): string[] {
  const templateOrder = (path: string): number => {
    const index = column.selectedPathTemplates?.findIndex(
      (template) => pathMatchesTemplate(path, template),
    ) ?? -1;
    return index < 0 ? Number.MAX_SAFE_INTEGER : index;
  };

  return paths
    .map((path, originalIndex) => ({
      path,
      originalIndex,
      instanceSortKey: objectInstanceSortKey(path, column),
    }))
    .sort((left, right) => {
      const instanceDifference = compareInstanceSortKeys(
        left.instanceSortKey,
        right.instanceSortKey,
      );
      if (instanceDifference) return instanceDifference;

      const templateDifference = templateOrder(left.path) - templateOrder(right.path);
      return templateDifference || left.originalIndex - right.originalIndex;
    })
    .map(({ path }) => path);
}

function parseOptionsForColumns(columns: ResultColumn[]): ParseMmlResultOptions {
  if (!columns.some((column) => column.selectedPathTemplates?.length)) {
    return {};
  }

  const exactPaths = new Set<string>();
  const leafPaths = new Set<string>();
  const objectRules = columns
    .filter((column) => column.path.endsWith('.'))
    .map((column) => {
      const templates = column.selectedPathTemplates ?? [];
      const matchersByLeaf = templates.length > 0 ? new Map<string, RegExp[]>() : null;
      templates.forEach((template) => {
        const leaf = leafName(template);
        const matchers = matchersByLeaf?.get(leaf) ?? [];
        matchers.push(pathTemplateMatcher(template));
        matchersByLeaf?.set(leaf, matchers);
      });
      return { prefix: column.path, matchersByLeaf };
    });
  columns.forEach((column) => {
    if (column.path.endsWith('.')) return;
    exactPaths.add(column.path);
    leafPaths.add(leafName(column.path));
  });

  return {
    includeParam: (param) => {
      const path = param.name;
      const leaf = leafName(path);
      if (exactPaths.has(path) || leafPaths.has(leaf)) return true;
      return objectRules.some(({ prefix, matchersByLeaf }) => {
        if (path === prefix || !path.startsWith(prefix)) return false;
        if (!matchersByLeaf) return true;
        return matchersByLeaf.get(leaf)?.some((matcher) => matcher.test(path)) ?? false;
      });
    },
  };
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
  const expandedDescendantPaths = new Set<string>();
  for (const column of columns) {
    if (!column.path.endsWith('.')) {
      if (emittedPaths.has(column.path)) continue;
      emittedPaths.add(column.path);
      expanded.push(column);
      continue;
    }

    const descendantPaths: string[] = [];
    const seenDescendantPaths = new Set<string>();
    let hasDescendant = false;
    for (const row of rows) {
      for (const path of Object.keys(row.cells)) {
        if (!resultPathMatchesColumn(path, column)) continue;
        hasDescendant = true;
        if (emittedPaths.has(path) || seenDescendantPaths.has(path)) continue;
        seenDescendantPaths.add(path);
        descendantPaths.push(path);
      }
    }

    if (!hasDescendant) {
      if (emittedPaths.has(column.path)) continue;
      emittedPaths.add(column.path);
      expanded.push(column);
      continue;
    }
    // 设备返回顺序不稳定，先按实例号数值排序再展开全部后代参数。
    // 不能在这里设置固定列数上限：例如 QOS 10 个实例 × 35 个参数，
    // 截断会导致后续实例数据已返回却无法在列表和详情中查看。
    const sortedDescendantPaths = sortObjectDescendantPaths(descendantPaths, column);
    sortedDescendantPaths.forEach((path) => {
      emittedPaths.add(path);
      expandedDescendantPaths.add(path);
    });
    const leafCounts = new Map<string, number>();
    sortedDescendantPaths.forEach((path) => {
      const leaf = leafName(path);
      leafCounts.set(leaf, (leafCounts.get(leaf) ?? 0) + 1);
    });
    sortedDescendantPaths.forEach((path, index) => {
      const leaf = leafName(path);
      expanded.push({
        key: `${column.key}:child:${index}`,
        label: (leafCounts.get(leaf) ?? 0) > 1 ? objectDescendantLabel(path, column) : leaf,
        path,
      });
    });
  }

  // 某些设备模型会把旧标准叶子映射到新对象下的私有叶子，同时命令还会查询该新对象。
  // 对象展开后，旧叶子列没有精确值却与真实后代形成同名重复列。仅在「旧列全无值」且
  // 对象名 + 实例 + 叶子名的结构尾部一致时移除旧列，避免误伤其他对象的同名参数。
  const structuralTail = (path: string): string => path
    .split('.')
    .filter(Boolean)
    .map((segment) => (segment === '{i}' || /^\d+$/.test(segment) ? '{i}' : segment))
    .slice(-3)
    .join('.');
  const descendantTails = new Set(
    [...expandedDescendantPaths]
      .filter((path) => rows.some((row) => Object.prototype.hasOwnProperty.call(row.cells, path)))
      .map(structuralTail),
  );

  return expanded.filter((column) => {
    if (expandedDescendantPaths.has(column.path) || column.path.endsWith('.')) return true;
    const hasOwnValue = rows.some((row) => Object.prototype.hasOwnProperty.call(row.cells, column.path));
    return hasOwnValue || !descendantTails.has(structuralTail(column.path));
  });
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
      const parsed = parseMmlDeviceTaskResult(frame.result, parseOptionsForColumns(columns));
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
    faultCode: status === 'failed'
      ? (deviceFaultMessageFromResult(frame.result) ?? frame.error_message ?? prev.faultCode)
      : prev.faultCode,
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
    const parsed = parseMmlDeviceTaskResult(item.result.parsedData, parseOptionsForColumns(columns));
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
    faultCode: resultItemFaultText(item),
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
  submittedValues?: Record<string, string>,
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
      const row = mapResultItemToRow(devItems[0], columns, read);
      if (!read && row.status === 'success' && submittedValues) {
        row.cells = { ...row.cells, ...submittedValues };
      }
      rows.push(row);
      continue;
    }
    // 逐 PATH 合并
    const base = devItems.map((it) => mapResultItemToRow(it, columns, read));
    const cells: Record<string, string> = Object.assign(
      {},
      ...base.map((r) => r.cells),
      ...(submittedValues && base.every((r) => r.status === 'success') ? [submittedValues] : []),
    );
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
        value: base[i].status === 'success'
          ? (base[i].cells[path] ?? '')
          : (resultItemFaultText(it, PATH_TASK_FAILED_FALLBACK) ?? PATH_TASK_FAILED_FALLBACK),
      });
    });
    pathTasks.sort((a, b) => a.pathIndex - b.pathIndex);
    const allOk = base.every((r) => r.status === 'success');
    const firstFault = base
      .filter((r) => r.status === 'failed')
      .map((r) => r.faultCode)
      .find((fault): fault is string => Boolean(fault));
    rows.push({
      ...base[0],
      status: allOk ? 'success' : 'failed',
      cells,
      faultCode: allOk ? undefined : (firstFault ?? PARTIAL_PATH_FAILED_FALLBACK),
      pathTasks,
    });
  }
  return rows;
}

const upperOp = (d?: MMLTaskCommandDetail): string => (d?.operationType ?? '').toString().toUpperCase();

function comparableMODValue(value: string): string {
  const normalized = value.trim().toLowerCase();
  if (normalized === 'true' || normalized === '1') return '1';
  if (normalized === 'false' || normalized === '0') return '0';
  return value;
}

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
    !!it.result?.parsedData && parseMmlDeviceTaskResult(it.result.parsedData)?.kind === 'gpv';

  const byDevice = new Map<string, DeviceTaskResultItem[]>();
  for (const it of items) {
    const arr = byDevice.get(it.deviceSn) ?? [];
    arr.push(it);
    byDevice.set(it.deviceSn, arr);
  }

  const rows: ResultRow[] = [];
  for (const [deviceSn, devItems] of byDevice) {
    const modItems = devItems.filter((it) => !isLst(it));
    const lstItems = devItems.filter((it) => isLst(it));
    const lstItem = lstItems[0];
    const modTaskId = modItems[0]?.deviceTaskId ?? '';
    const lstTaskId = lstItem?.deviceTaskId ?? '';

    // 回读值 + 回读 path（GetParameterValues 响应的参数名即 PATH，修复回读行 PATH 为空）
    const readback = new Map<string, string>();
    const readbackPairs: { path: string; value: string; item: DeviceTaskResultItem }[] = [];
    for (const item of lstItems) {
      if (!item.result?.parsedData) continue;
      const parsed = parseMmlDeviceTaskResult(item.result.parsedData);
      if (parsed?.kind === 'gpv' && parsed.params) {
        for (const p of parsed.params) {
          readback.set(p.name, p.value);
          readback.set(leafName(p.name), p.value);
          readbackPairs.push({ path: p.name, value: p.value, item });
        }
      }
    }
    const readVal = (path: string): string => readback.get(path) ?? readback.get(leafName(path)) ?? '';
    const hasReadback = readbackPairs.length > 0;
    const modOk = modItems.length > 0 && modItems.every((it) => it.result?.success);
    const lstOk = lstItems.length > 0 && lstItems.every((item) => item.result?.success === true);
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
        value: modOk
          ? (setValues[path] ?? '')
          : (firstMod ? (resultItemFaultText(firstMod, PATH_TASK_FAILED_FALLBACK) ?? PATH_TASK_FAILED_FALLBACK) : PATH_TASK_FAILED_FALLBACK),
      });
    }
    if (lstItem) {
      const lstRows: { path: string; value: string; item?: DeviceTaskResultItem }[] = hasReadback
        ? readbackPairs
        : Object.keys(setValues).map((path) => ({
            path,
            value: lstOk
              ? ''
              : (resultItemFaultText(lstItem, READBACK_FAILED_FALLBACK) ?? READBACK_FAILED_FALLBACK),
          }));
      for (const { path, value, item = lstItem } of lstRows) {
        pathTasks.push({
          pathIndex: 1,
          path,
          subTaskId: item?.deviceTaskId ?? lstTaskId,
          opType: 'LST',
          status: item?.result?.success ? 'success' : 'failed',
          dispatchedAt: toClock(item?.startedAt) ?? '',
          respondedAt: toClock(item?.finishedAt) ?? '',
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
    else if (Object.keys(setValues).some((p) => (
      comparableMODValue(readVal(p)) !== comparableMODValue(setValues[p])
    ))) status = 'mismatch';

    rows.push({
      commandName: commandMeta?.commandName ?? firstMod?.commandName,
      commandCode: commandMeta?.commandCode ?? firstMod?.commandCode,
      deviceSn,
      deviceTaskId: modTaskId,
      status,
      cells,
      faultCode: modOk
        ? undefined
        : (firstMod ? (resultItemFaultText(firstMod, DISPATCH_FAILED_FALLBACK) ?? DISPATCH_FAILED_FALLBACK) : DISPATCH_FAILED_FALLBACK),
      unverifiedReason: status === 'unverified' ? 'query-failed' : undefined,
      pathTasks,
      dispatchedAt: toClock(firstMod?.startedAt),
      respondedAt: toClock(lstItems.at(-1)?.finishedAt ?? firstMod?.finishedAt),
      raw: firstMod?.result?.rawOutput ?? '',
      readbackRaw: lstItem?.result?.rawOutput ?? '',
      elapsedMs: (firstMod?.result?.executionTime ?? 0) + lstItems.reduce((sum, item) => sum + (item.result?.executionTime ?? 0), 0),
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
  const columns = buildTaskResultColumns(task.commandsDetail ?? []);
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
