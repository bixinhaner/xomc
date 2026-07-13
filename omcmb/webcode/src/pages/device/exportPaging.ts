/** 行数安全上限：防止超大筛选结果一次性拉爆浏览器内存。 */
export const EXPORT_ROW_CAP = 50000;
/** 分页拉全量时单页条数(后端 list 接口 PageSize 校验 max=1000,取满)。 */
export const EXPORT_PAGE_SIZE = 1000;
/** 并发拉取的最大并行请求数(全表 3W+ 行时,串行 60+ 次请求会慢到像卡死)。 */
export const EXPORT_CONCURRENCY = 6;

export interface PageResult<T> {
  items: T[];
  total: number;
}

/**
 * 按 fetchPage 分页拉取全部数据：先取第 1 页拿到 total,再以固定并发并行拉取
 * 其余页,最后按页序稳定拼接。相比串行逐页,全表(3W+ 行)导出从数十秒降到数秒。
 *  - fetchPage(page,pageSize) 由调用方提供(封装具体 API + 筛选条件);
 *  - onProgress 用于驱动"正在导出 x/total"进度反馈;
 *  - 受 EXPORT_ROW_CAP 上限保护,返回值已 slice 到上限。
 */
export async function fetchAllPaged<T>(
  fetchPage: (page: number, pageSize: number) => Promise<PageResult<T>>,
  opts: { pageSize?: number; cap?: number; onProgress?: (loaded: number, total: number) => void } = {},
): Promise<{ items: T[]; total: number; capped: boolean }> {
  const pageSize = opts.pageSize ?? EXPORT_PAGE_SIZE;
  const cap = opts.cap ?? EXPORT_ROW_CAP;

  const first = await fetchPage(1, pageSize);
  const grandTotal = first.total ?? first.items.length;
  const target = Math.min(grandTotal, cap);
  const acc: T[] = first.items.slice(0, cap);
  opts.onProgress?.(Math.min(acc.length, target), target);

  if (acc.length >= target || first.items.length < pageSize) {
    return { items: acc.slice(0, cap), total: grandTotal, capped: grandTotal > acc.length };
  }

  const totalPages = Math.ceil(target / pageSize);
  const pendingPages: number[] = [];
  for (let p = 2; p <= totalPages; p += 1) pendingPages.push(p);

  const byPage = new Map<number, T[]>();
  let cursor = 0;
  let loaded = acc.length;

  const worker = async (): Promise<void> => {
    for (;;) {
      const myIdx = cursor;
      cursor += 1;
      if (myIdx >= pendingPages.length) return;
      const pageNo = pendingPages[myIdx];
      const res = await fetchPage(pageNo, pageSize);
      byPage.set(pageNo, res.items);
      loaded += res.items.length;
      opts.onProgress?.(Math.min(loaded, target), target);
    }
  };

  await Promise.all(
    Array.from({ length: Math.min(EXPORT_CONCURRENCY, pendingPages.length) }, () => worker()),
  );

  for (let p = 2; p <= totalPages; p += 1) {
    const items = byPage.get(p);
    if (items) acc.push(...items);
  }
  const finalItems = acc.slice(0, cap);
  return { items: finalItems, total: grandTotal, capped: grandTotal > finalItems.length };
}
