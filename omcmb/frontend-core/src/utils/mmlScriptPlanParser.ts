import type {
  MMLTaskCommandInput,
  MMLTaskExecuteMode,
  MMLTaskPlanItem,
} from '../types/mml';

export interface MMLScriptPlanParseResult {
  executeMode: MMLTaskExecuteMode;
  commands: MMLTaskCommandInput[];
  planItems: MMLTaskPlanItem[];
  deviceSns: string[];
  warnings: string[];
  errors: string[];
}

export interface MMLScriptPlanParseOptions {
  format?: 'text' | 'csv' | 'auto';
}

interface ParsedLine {
  lineNo: number;
  rawLine: string;
  command: MMLTaskCommandInput;
  deviceSn?: string;
  order?: number;
}

interface ParsedPlanLine {
  commandLine: string;
  deviceSn?: string;
  order?: number;
}

export function parseMmlScriptPlan(
  content: string,
  options: MMLScriptPlanParseOptions = {},
): MMLScriptPlanParseResult {
  const format = options.format === 'auto' || !options.format ? detectFormat(content) : options.format;
  const parsed = format === 'csv' ? parseCsvPlan(content) : parseTextPlan(content);
  return buildResult(parsed);
}

export function parseMmlScriptCommands(content: string): MMLTaskCommandInput[] {
  return parseMmlScriptPlan(content, { format: 'text' }).commands;
}

function buildResult(parsed: ParsedLine[]): MMLScriptPlanParseResult {
  const commands = parsed.map((p) => p.command);
  const planItems: MMLTaskPlanItem[] = [];
  const deviceSns: string[] = [];
  const seenSn = new Set<string>();
  const nextOrderBySn = new Map<string, number>();
  const warnings: string[] = [];

  for (const p of parsed) {
    if (!p.deviceSn) continue;
    if (!seenSn.has(p.deviceSn)) {
      seenSn.add(p.deviceSn);
      deviceSns.push(p.deviceSn);
    }
    const nextOrder = (nextOrderBySn.get(p.deviceSn) ?? 0) + 1;
    const order = p.order && p.order > 0 ? p.order : nextOrder;
    nextOrderBySn.set(p.deviceSn, Math.max(nextOrder, order));
    planItems.push({
      lineNo: p.lineNo,
      deviceSn: p.deviceSn,
      order,
      rawLine: p.rawLine,
      command: p.command,
    });
  }

  if (planItems.length > 0 && planItems.length < parsed.length) {
    warnings.push('MIXED_DEVICE_BOUND_AND_COMMON_LINES');
  }

  return {
    executeMode: planItems.length > 0 ? 'device_bound' : 'common',
    commands,
    planItems,
    deviceSns,
    warnings,
    errors: [],
  };
}

function parseTextPlan(content: string): ParsedLine[] {
  const out: ParsedLine[] = [];
  const lines = content.split(/\r?\n/);
  for (let idx = 0; idx < lines.length; idx += 1) {
    const rawLine = lines[idx] ?? '';
    const cleaned = normalizeScriptLine(rawLine);
    if (!cleaned) continue;

    const plan = parseExplicitPipeLine(cleaned) ?? parseTrailingDeviceLine(cleaned);
    const command = parseCommand(plan.commandLine);
    if (!command) continue;
    out.push({
      lineNo: idx + 1,
      rawLine: rawLine.trim(),
      command,
      deviceSn: plan.deviceSn,
      order: plan.order,
    });
  }
  return out;
}

function parseExplicitPipeLine(line: string): ParsedPlanLine | null {
  const parts = splitTopLevel(line, '|').map((p) => p.trim()).filter(Boolean);
  if (parts.length < 2) return null;

  const sn = normalizeDeviceToken(parts[0] ?? '');
  if (!sn) return null;

  const maybeOrder = Number(parts[1]);
  if (Number.isFinite(maybeOrder) && maybeOrder > 0 && parts.length >= 3) {
    return { deviceSn: sn, order: maybeOrder, commandLine: parts.slice(2).join('|').trim() };
  }
  return { deviceSn: sn, commandLine: parts.slice(1).join('|').trim() };
}

function parseTrailingDeviceLine(line: string): ParsedPlanLine {
  const parts = splitTopLevel(line, ';');
  let tailIndex = parts.length - 1;
  while (tailIndex > 0 && !(parts[tailIndex] ?? '').trim()) {
    tailIndex -= 1;
  }
  const tail = tailIndex > 0 ? parts[tailIndex]?.trim() ?? '' : '';
  const deviceSn = normalizeDeviceToken(tail);
  if (deviceSn) {
    return {
      commandLine: parts.slice(0, tailIndex).join(';').trim(),
      deviceSn,
    };
  }
  return { commandLine: line.replace(/;+$/, '').trim() };
}

function parseCsvPlan(content: string): ParsedLine[] {
  const rows = parseCsvRows(content).filter((r) => r.some((c) => c.trim() !== ''));
  if (rows.length === 0) return [];

  const first = rows[0].map((c) => c.trim().toLowerCase());
  const header = buildHeaderIndex(first);
  const hasHeader = header.command >= 0 || header.commandCode >= 0 || header.deviceSn >= 0;
  const dataRows = hasHeader ? rows.slice(1) : rows;
  const out: ParsedLine[] = [];

  for (let idx = 0; idx < dataRows.length; idx += 1) {
    const row = dataRows[idx];
    const fallbackLineNo = idx + 1 + (hasHeader ? 1 : 0);
    const lineNo = numericCell(row, header.lineNo, fallbackLineNo);
    const deviceSn = hasHeader ? normalizeDeviceToken(cell(row, header.deviceSn)) : normalizeDeviceToken(row[0] ?? '');
    const order = hasHeader ? numericCell(row, header.order) : numericCell(row, 1);
    const rawLine = cell(row, header.rawLine) || row.join(',');
    let command = parseCommand(cell(row, header.command));

    if (!command && header.commandCode >= 0) {
      const commandCode = cell(row, header.commandCode);
      if (commandCode) {
        command = {
          commandCode,
          operationType: cell(row, header.operationType) || deriveOperationType(commandCode),
        };
        const parameters = parseParameterPart(cell(row, header.parameters));
        if (Object.keys(parameters).length > 0) command.parameters = parameters;
      }
    }
    if (!command && !hasHeader) {
      const commandText = row.length >= 3 ? row.slice(2).join(',') : row[0] ?? '';
      command = parseCommand(commandText);
    }
    if (!command) continue;

    out.push({
      lineNo,
      rawLine,
      command,
      deviceSn: deviceSn || undefined,
      order,
    });
  }
  return out;
}

function buildHeaderIndex(headers: string[]) {
  const find = (...names: string[]) => headers.findIndex((h) => names.includes(h));
  return {
    lineNo: find('line_no', 'line', 'lineno', 'row'),
    deviceSn: find('device_sn', 'device sn', 'devicesn', 'sn', 'device'),
    order: find('order', 'seq', 'sequence'),
    rawLine: find('raw_line', 'raw line', 'raw', 'source'),
    command: find('command', 'mml', 'script', 'command_line'),
    commandCode: find('command_code', 'command code', 'commandcode'),
    operationType: find('operation_type', 'operation type', 'operationtype', 'op'),
    parameters: find('parameters', 'params'),
  };
}

function parseCommand(raw: string): MMLTaskCommandInput | null {
  const line = raw.replace(/;+$/, '').trim();
  if (!line) return null;

  const colonIndex = findTopLevelChar(line, ':');
  let commandCode = '';
  let paramPart = '';

  if (colonIndex >= 0) {
    commandCode = line.slice(0, colonIndex).trim();
    paramPart = line.slice(colonIndex + 1).trim();
  } else {
    const fields = line.split(/\s+/);
    const firstParamIndex = fields.findIndex((field) => field.includes('='));
    if (firstParamIndex >= 0) {
      commandCode = fields.slice(0, firstParamIndex).join(' ').trim();
      paramPart = fields.slice(firstParamIndex).join(' ');
    } else {
      commandCode = line;
    }
  }

  if (!commandCode) return null;
  const parameters = parseParameterPart(paramPart);
  const command: MMLTaskCommandInput = { commandCode };
  const operationType = deriveOperationType(commandCode);
  if (operationType) command.operationType = operationType;
  if (Object.keys(parameters).length > 0) command.parameters = parameters;
  return command;
}

function parseParameterPart(paramPart: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const token of splitParameterTokens(paramPart)) {
    const eq = token.indexOf('=');
    const colon = token.indexOf(':');
    const sep = eq > 0 ? eq : colon > 0 ? colon : -1;
    if (sep <= 0) continue;
    const key = token.slice(0, sep).trim();
    const value = token.slice(sep + 1).trim();
    if (key) out[key] = value;
  }
  return out;
}

function splitParameterTokens(input: string): string[] {
  const tokens: string[] = [];
  let current = '';
  let braceDepth = 0;
  let quote: string | null = null;
  for (const ch of input) {
    if ((ch === '"' || ch === "'") && !quote) quote = ch;
    else if (quote === ch) quote = null;
    if (!quote) {
      if (ch === '{') braceDepth += 1;
      if (ch === '}' && braceDepth > 0) braceDepth -= 1;
      if (braceDepth === 0 && (ch === ',' || /\s/.test(ch))) {
        if (current.trim()) tokens.push(current.trim());
        current = '';
        continue;
      }
    }
    current += ch;
  }
  if (current.trim()) tokens.push(current.trim());
  return tokens;
}

function normalizeScriptLine(raw: string): string {
  const line = raw.trim();
  if (!line || line.startsWith('#') || line.startsWith('//')) return '';
  const commentIndex = findInlineCommentIndex(line);
  return (commentIndex >= 0 ? line.slice(0, commentIndex) : line).trim();
}

function normalizeDeviceToken(raw: string): string {
  let value = raw.trim();
  if (!value) return '';
  value = value.replace(/^["']|["']$/g, '').trim();
  if (value.startsWith('{') && value.endsWith('}')) {
    value = value.slice(1, -1).trim();
  }
  if (!value || /^(serial\s*number|device\s*sn|device_sn|sn)$/i.test(value)) {
    return '';
  }
  if (/[=:]/.test(value)) return '';
  return value;
}

function deriveOperationType(commandCode: string): MMLTaskCommandInput['operationType'] | undefined {
  const op = commandCode.trim().split(/\s+|_/)[0]?.toUpperCase();
  return op || undefined;
}

function detectFormat(content: string): 'text' | 'csv' {
  const first = content
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find((line) => line && !line.startsWith('#') && !line.startsWith('//'));
  if (!first) return 'text';
  const lower = first.toLowerCase();
  if (first.includes(',') && (lower.includes('device') || lower.includes('sn') || lower.includes('command'))) {
    return 'csv';
  }
  return 'text';
}

function splitTopLevel(input: string, sep: string): string[] {
  const out: string[] = [];
  let current = '';
  let braceDepth = 0;
  let quote: string | null = null;
  for (const ch of input) {
    if ((ch === '"' || ch === "'") && !quote) quote = ch;
    else if (quote === ch) quote = null;
    if (!quote) {
      if (ch === '{') braceDepth += 1;
      if (ch === '}' && braceDepth > 0) braceDepth -= 1;
      if (braceDepth === 0 && ch === sep) {
        out.push(current);
        current = '';
        continue;
      }
    }
    current += ch;
  }
  out.push(current);
  return out;
}

function findTopLevelChar(input: string, target: string): number {
  let braceDepth = 0;
  let quote: string | null = null;
  for (let i = 0; i < input.length; i += 1) {
    const ch = input[i];
    if ((ch === '"' || ch === "'") && !quote) quote = ch;
    else if (quote === ch) quote = null;
    if (quote) continue;
    if (ch === '{') braceDepth += 1;
    if (ch === '}' && braceDepth > 0) braceDepth -= 1;
    if (braceDepth === 0 && ch === target) return i;
  }
  return -1;
}

function findInlineCommentIndex(line: string): number {
  const hash = line.indexOf('#');
  const slash = line.indexOf('//');
  if (hash < 0) return slash;
  if (slash < 0) return hash;
  return Math.min(hash, slash);
}

function parseCsvRows(input: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let cellValue = '';
  let inQuote = false;
  for (let i = 0; i < input.length; i += 1) {
    const ch = input[i];
    const next = input[i + 1];
    if (ch === '"') {
      if (inQuote && next === '"') {
        cellValue += '"';
        i += 1;
      } else {
        inQuote = !inQuote;
      }
      continue;
    }
    if (!inQuote && ch === ',') {
      row.push(cellValue);
      cellValue = '';
      continue;
    }
    if (!inQuote && (ch === '\n' || ch === '\r')) {
      if (ch === '\r' && next === '\n') i += 1;
      row.push(cellValue);
      rows.push(row);
      row = [];
      cellValue = '';
      continue;
    }
    cellValue += ch;
  }
  row.push(cellValue);
  rows.push(row);
  return rows;
}

function cell(row: string[], index: number): string {
  return index >= 0 ? row[index]?.trim() ?? '' : '';
}

function numericCell(row: string[], index: number, fallback?: number): number {
  const value = Number(cell(row, index));
  return Number.isFinite(value) && value > 0 ? value : fallback ?? 0;
}
