import type { DeviceTaskResultItem } from '@core/types/mml';
import { parseMmlCommandDisplay } from '@core/utils/mmlCommandDisplay';

interface RequestValue {
  name: string;
  value: string;
}

function baseCommandText(row: DeviceTaskResultItem): string {
  return row.commandName || row.mmlScript || row.planRawLine || row.commandCode || '';
}

function normalizeOperation(value?: string): string {
  return (value || '').trim().toUpperCase();
}

function requestValues(row: DeviceTaskResultItem): RequestValue[] {
  const payload = row.request?.payload;
  if (!payload || typeof payload !== 'object') return [];
  const payloadObject = payload as { values?: unknown; parameter_values?: unknown; parameters?: unknown };
  const rawValues = payloadObject.values ?? payloadObject.parameter_values ?? payloadObject.parameters;
  if (rawValues && typeof rawValues === 'object' && !Array.isArray(rawValues)) {
    return Object.entries(rawValues as Record<string, unknown>).map(([name, value]) => ({
      name,
      value: value === null || value === undefined ? '' : String(value),
    }));
  }
  if (!Array.isArray(rawValues)) return [];
  return rawValues
    .map((item) => {
      if (!item || typeof item !== 'object') return null;
      const rec = item as { name?: unknown; value?: unknown };
      if (typeof rec.name !== 'string' || !rec.name.trim()) return null;
      return {
        name: rec.name.trim(),
        value: rec.value === null || rec.value === undefined ? '' : String(rec.value),
      };
    })
    .filter((item): item is RequestValue => item !== null);
}

function parameterParentPath(path: string): string {
  const dot = path.lastIndexOf('.');
  return dot >= 0 ? path.slice(0, dot + 1) : '';
}

function parameterLeaf(path: string): string {
  const trimmed = path.endsWith('.') ? path.slice(0, -1) : path;
  const dot = trimmed.lastIndexOf('.');
  return dot >= 0 ? trimmed.slice(dot + 1) : trimmed;
}

function formatModFromRequest(row: DeviceTaskResultItem): string {
  if (normalizeOperation(row.operationType) !== 'MOD') return '';
  if (row.request?.method && row.request.method !== 'SetParameterValues') return '';
  const values = requestValues(row);
  if (values.length === 0) return '';
  const parent = parameterParentPath(values[0].name);
  const params = values.map((item) => `${parameterLeaf(item.name)}=${item.value}`).join(',');
  return `MOD ${parent || values[0].name}:${params}`;
}

function formatWithActualOperation(row: DeviceTaskResultItem, raw: string): string {
  const actualOp = normalizeOperation(row.operationType);
  if (!actualOp) return raw;
  const parsed = parseMmlCommandDisplay(raw);
  const rawOp = normalizeOperation(parsed.operation);
  if (!rawOp || rawOp === actualOp) return raw;
  const params = parsed.params.map((item) => `${item.key}=${item.value}`).join(',');
  const target = parsed.target || parsed.commandHead;
  return `${actualOp}${target ? ` ${target}` : ''}${params ? `:${params}` : ''}`;
}

export function taskResultCommandText(row: DeviceTaskResultItem): string {
  const fromRequest = formatModFromRequest(row);
  if (fromRequest) return fromRequest;
  const raw = baseCommandText(row);
  return raw ? formatWithActualOperation(row, raw) : '';
}
