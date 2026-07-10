export const MML_MAX_TASK_DEVICES = 200;
export const MML_MAX_PLAN_ITEMS = 2000;
export const MML_PREVIEW_PAGE_SIZE = 20;
export const MML_PREVIEW_PAGE_SIZE_OPTIONS = [20, 50, 100] as const;

export type MMLTaskScaleIssue =
  | { kind: 'devices'; current: number; max: typeof MML_MAX_TASK_DEVICES }
  | { kind: 'plan_items'; current: number; max: typeof MML_MAX_PLAN_ITEMS };

export function uniqueDeviceCount(deviceSns: readonly string[]): number {
  return new Set(deviceSns.map((sn) => sn.trim()).filter(Boolean)).size;
}

export function validateMmlTaskScale(
  deviceSns: readonly string[],
  planItemCount: number,
): MMLTaskScaleIssue | null {
  const deviceCount = uniqueDeviceCount(deviceSns);
  if (deviceCount > MML_MAX_TASK_DEVICES) {
    return { kind: 'devices', current: deviceCount, max: MML_MAX_TASK_DEVICES };
  }
  if (planItemCount > MML_MAX_PLAN_ITEMS) {
    return { kind: 'plan_items', current: planItemCount, max: MML_MAX_PLAN_ITEMS };
  }
  return null;
}

export interface MMLPreviewPage<T> {
  items: T[];
  page: number;
  pageCount: number;
  start: number;
  end: number;
  total: number;
}

export function paginateMmlPreview<T>(
  source: readonly T[],
  requestedPage: number,
  requestedPageSize: number,
): MMLPreviewPage<T> {
  const pageSize = Math.max(1, requestedPageSize);
  const total = source.length;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const page = Math.min(Math.max(1, requestedPage), pageCount);
  const startIndex = (page - 1) * pageSize;
  const items = source.slice(startIndex, startIndex + pageSize);
  return {
    items,
    page,
    pageCount,
    start: total === 0 ? 0 : startIndex + 1,
    end: Math.min(startIndex + items.length, total),
    total,
  };
}
