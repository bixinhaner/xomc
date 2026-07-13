import dayjs from 'dayjs';
import type { PageResponse } from '@core/types/pagination';
import type { DeviceTaskResultItem, MMLTask, MMLTaskResultsStats } from '@core/types/mml';
import { parseMmlDeviceTaskResult, type ParsedMmlResult } from '@core/utils/mmlResultParser';
import { saveBlob } from '@core/utils/saveBlob';
import { formatSystemTime } from '@core/utils/systemTime';
import { getMMLTaskResultsPage } from '@core/hooks/api/useMML';

export const MML_TASK_RESULT_EXPORT_PAGE_SIZE = 100;

type TFunction = (key: string, values?: Record<string, string | number>) => string;

export type FetchMmlTaskResultsPage = (
  taskId: string,
  page: number,
  pageSize: number,
) => Promise<PageResponse<DeviceTaskResultItem, MMLTaskResultsStats>>;

const DEVICE_RESULT_STATUS_KEYS: Record<string, string> = {
  pending: 'mml.pendingStatus',
  running: 'mml.runningStatus',
  completed: 'mml.completedStatus',
  failed: 'mml.failedStatus',
};

function stringifyMessagePayload(payload: unknown): string {
  if (payload === null || payload === undefined || payload === '') return '';
  if (typeof payload === 'string') return payload;
  try {
    return JSON.stringify(payload, null, 2);
  } catch {
    return String(payload);
  }
}

export function taskResultCommandText(row: DeviceTaskResultItem): string {
  return row.mmlScript || row.planRawLine || row.commandCode || '';
}

export function taskResultRequestText(row: DeviceTaskResultItem): string {
  if (!row.request) return '';
  if (row.request.rawRequest) return row.request.rawRequest;
  return stringifyMessagePayload({
    method: row.request.method,
    ...(row.request.cwmpId ? { cwmp_id: row.request.cwmpId } : {}),
    ...(row.request.commandKey ? { command_key: row.request.commandKey } : {}),
    ...(row.request.payload !== undefined ? { params: row.request.payload } : {}),
  });
}

export function taskResultResponseText(row: DeviceTaskResultItem): string {
  if (row.result?.rawOutput) return row.result.rawOutput;
  if (row.result?.parsedData) return stringifyMessagePayload(row.result.parsedData);
  return '';
}

function formatTime(iso?: string | null): string {
  return formatSystemTime(iso, { placeholder: '' });
}

function parsedResultSummary(parsed: ParsedMmlResult, t: TFunction): string {
  switch (parsed.kind) {
    case 'gpv':
      return (parsed.params ?? [])
        .map((param) => `${param.name}=${param.value ?? ''}`)
        .join('\n');
    case 'spv':
      return parsed.status === 1
        ? t('mml.taskResult.parsed.spv.reboot')
        : t('mml.taskResult.parsed.spv.immediate');
    case 'add':
      return t('mml.taskResult.parsed.add.success', { n: parsed.instanceNumber ?? '-' });
    case 'delete':
      return t('mml.taskResult.parsed.delete.success');
    case 'reboot':
      return t('mml.taskResult.parsed.reboot.success');
    default:
      return t('mml.taskResult.parsed.notParsable');
  }
}

function parsedResultText(row: DeviceTaskResultItem, t: TFunction): string {
  if (!row.result?.parsedData) return '';
  const parsed = parseMmlDeviceTaskResult(row.result.parsedData);
  return parsed ? parsedResultSummary(parsed, t) : '';
}

function statusText(row: DeviceTaskResultItem, t: TFunction): string {
  if (!row.status) return '';
  const key = DEVICE_RESULT_STATUS_KEYS[row.status] ?? '';
  return key ? t(key) : row.status;
}

function resultText(row: DeviceTaskResultItem, t: TFunction): string {
  if (!row.result) return '';
  return row.result.success ? t('status.success') : t('status.failed');
}

export function escapeCsvCell(value: unknown): string {
  const raw = value === null || value === undefined ? '' : String(value);
  const formulaSafe = /^[\t\r]/.test(raw) || /^[=+\-@]/.test(raw.trimStart())
    ? `'${raw}`
    : raw;
  return /[",\r\n]/.test(formulaSafe)
    ? `"${formulaSafe.replace(/"/g, '""')}"`
    : formulaSafe;
}

export function buildMmlTaskResultsCsv(rows: DeviceTaskResultItem[], t: TFunction): string {
  const header = [
    t('table.index'),
    t('mml.resultDeviceCode'),
    t('mml.deviceName'),
    t('mml.scriptLineNoColumn'),
    t('mml.planOrder'),
    t('mml.resultCommand'),
    t('mml.status'),
    t('mml.result'),
    t('mml.failReason'),
    t('mml.taskResult.parsed.title'),
    t('mml.requestMessage'),
    t('mml.responseMessage'),
    t('mml.startTime'),
    t('mml.endTime'),
  ];

  const body = rows.map((row, index) => [
    index + 1,
    row.deviceSn,
    row.deviceName ?? '',
    row.planLineNo ? t('mml.scriptLineNo', { line: row.planLineNo }) : '',
    row.planOrder ?? '',
    taskResultCommandText(row),
    statusText(row, t),
    resultText(row, t),
    row.failReason ?? '',
    parsedResultText(row, t),
    taskResultRequestText(row),
    taskResultResponseText(row),
    formatTime(row.startedAt),
    formatTime(row.finishedAt),
  ]);

  return [header, ...body]
    .map((line) => line.map(escapeCsvCell).join(','))
    .join('\r\n');
}

export async function fetchAllMmlTaskResults(
  taskId: string,
  fetchPage: FetchMmlTaskResultsPage = getMMLTaskResultsPage,
): Promise<DeviceTaskResultItem[]> {
  const rows: DeviceTaskResultItem[] = [];
  let total = Number.POSITIVE_INFINITY;
  let page = 1;

  while (rows.length < total) {
    const data = await fetchPage(taskId, page, MML_TASK_RESULT_EXPORT_PAGE_SIZE);
    rows.push(...(data.items ?? []));
    total = data.total ?? rows.length;
    if ((data.items ?? []).length === 0) break;
    page += 1;
  }

  return rows;
}

function sanitizeFilenamePart(value: string): string {
  return value
    .trim()
    .replace(/[\\/:*?"<>|]+/g, '_')
    .replace(/\s+/g, '_')
    .slice(0, 80) || 'task';
}

export function buildMmlTaskResultsCsvFilename(task: MMLTask): string {
  const name = sanitizeFilenamePart(task.taskName || task.id);
  return `mml-task-results-${name}-${dayjs().format('YYYYMMDD-HHmmss')}.csv`;
}

export function downloadMmlTaskResultsCsv(task: MMLTask, rows: DeviceTaskResultItem[], t: TFunction): void {
  saveBlob(`\ufeff${buildMmlTaskResultsCsv(rows, t)}`, buildMmlTaskResultsCsvFilename(task), 'text/csv;charset=utf-8');
}
